package service

import (
	"context"
	"errors"
	"time"

	"github.com/gofiber/fiber/v2"

	"api-students-db/app/model"
	"api-students-db/app/repository"
	"api-students-db/helper"
)

// AuthService menangani logika bisnis autentikasi.
type AuthService struct {
	userRepo    repository.UserRepository
	refreshRepo repository.RefreshTokenRepository
	jwt         *helper.JWTManager
	perms       *helper.PermissionSet
}

// NewAuthService membuat instance AuthService baru.
func NewAuthService(
	userRepo repository.UserRepository,
	refreshRepo repository.RefreshTokenRepository,
	jwtManager *helper.JWTManager,
	perms *helper.PermissionSet,
) *AuthService {
	return &AuthService{
		userRepo:    userRepo,
		refreshRepo: refreshRepo,
		jwt:         jwtManager,
		perms:       perms,
	}
}

// Register godoc
//
// @Summary      Mendaftarkan akun baru
// @Description  Role akun baru selalu "user". Password minimal 8 karakter dan memuat huruf serta angka.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body body     model.RegisterRequest true "Data akun baru"
// @Success      201  {object} model.WebResponse{data=model.User}
// @Failure      400  {object} model.ErrorResponse "Body bukan JSON yang valid"
// @Failure      409  {object} model.ErrorResponse "Username sudah dipakai"
// @Failure      422  {object} model.ErrorResponse "Validasi gagal; detail per field ada di fields"
// @Router       /auth/register [post]
func (s *AuthService) Register(c *fiber.Ctx) error {
	ctx, cancel := helper.ReqCtx(c)
	defer cancel()

	var req model.RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.Fail(c, fiber.StatusBadRequest, "Format JSON request body tidak valid")
	}

	if errs := ValidateRegister(req); len(errs) > 0 {
		return helper.FailValidation(c, "Validasi registrasi gagal", errs)
	}

	// Hash password
	hashedPassword, err := helper.HashPassword(req.Password)
	if err != nil {
		return helper.Fail(c, fiber.StatusInternalServerError, "Gagal memproses registrasi")
	}

	// Buat user dengan role default "user" (mencegah mass assignment)
	user, err := s.userRepo.Create(ctx, model.User{
		Username: req.Username,
		Email:    req.Email,
		Password: hashedPassword,
		Role:     "user",
		IsActive: true,
	})
	if err != nil {
		if errors.Is(err, repository.ErrDuplicate) {
			return helper.Fail(c, fiber.StatusConflict, "Username atau email sudah terdaftar")
		}
		return helper.Fail(c, fiber.StatusInternalServerError, "Gagal mendaftarkan user")
	}

	// Password tidak akan muncul di response karena json:"-"
	return helper.Created(c, "Registrasi berhasil", user, "/api/v1/auth/me")
}

// Login godoc
//
// @Summary      Login dan mendapatkan pasangan token
// @Description  Dibatasi 5 kali gagal per menit per IP.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body body     model.LoginRequest true "Username dan password"
// @Success      200  {object} model.WebResponse{data=model.TokenPair}
// @Failure      401  {object} model.ErrorResponse "Username atau password salah"
// @Failure      429  {object} model.ErrorResponse "Terlalu banyak percobaan login"
// @Router       /auth/login [post]
func (s *AuthService) Login(c *fiber.Ctx) error {
	ctx, cancel := helper.ReqCtx(c)
	defer cancel()

	var req model.LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.Fail(c, fiber.StatusBadRequest, "Format JSON request body tidak valid")
	}

	if errs := ValidateLogin(req); len(errs) > 0 {
		return helper.FailValidation(c, "Validasi login gagal", errs)
	}

	// Cari user berdasarkan username
	user, err := s.userRepo.FindByUsername(ctx, req.Username)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			// Dummy bcrypt verification untuk mencegah user enumeration
			_ = helper.VerifyPassword("$2a$12$dummyhashvaluefortimingattak000000000000000000000", req.Password)
			return helper.Fail(c, fiber.StatusUnauthorized, "Username atau password salah")
		}
		return helper.Fail(c, fiber.StatusInternalServerError, "Gagal memproses login")
	}

	// Verifikasi password
	if err := helper.VerifyPassword(user.Password, req.Password); err != nil {
		return helper.Fail(c, fiber.StatusUnauthorized, "Username atau password salah")
	}

	// Periksa apakah user aktif
	if !user.IsActive {
		return helper.Fail(c, fiber.StatusForbidden, "Akun tidak aktif")
	}

	// Generate token pair
	tokenPair, err := s.generateTokenPair(ctx, user)
	if err != nil {
		return helper.Fail(c, fiber.StatusInternalServerError, "Gagal membuat token")
	}

	return helper.Success(c, fiber.StatusOK, "Login berhasil", tokenPair)
}

// Refresh godoc
//
// @Summary      Menukar refresh token dengan pasangan token baru
// @Description  Refresh token lama langsung dicabut (rotation).
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body body     model.RefreshRequest true "Refresh token"
// @Success      200  {object} model.WebResponse{data=model.TokenPair}
// @Failure      401  {object} model.ErrorResponse "Refresh token tidak valid, kedaluwarsa, atau sudah dicabut"
// @Router       /auth/refresh [post]
func (s *AuthService) Refresh(c *fiber.Ctx) error {
	ctx, cancel := helper.ReqCtx(c)
	defer cancel()

	var req model.RefreshRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.Fail(c, fiber.StatusBadRequest, "Format JSON request body tidak valid")
	}

	if req.RefreshToken == "" {
		return helper.Fail(c, fiber.StatusBadRequest, "Refresh token wajib diisi")
	}

	// Hash token untuk mencari di database
	tokenHash := helper.HashToken(req.RefreshToken)

	// Cari token aktif di database
	rt, err := s.refreshRepo.FindByTokenHash(ctx, tokenHash)
	if err != nil {
		return helper.Fail(c, fiber.StatusUnauthorized, "Refresh token tidak valid")
	}

	// Pastikan belum di-revoke
	if rt.RevokedAt != nil {
		return helper.Fail(c, fiber.StatusUnauthorized, "Refresh token sudah dicabut")
	}

	// Pastikan belum expired
	if time.Now().After(rt.ExpiresAt) {
		return helper.Fail(c, fiber.StatusUnauthorized, "Refresh token sudah kedaluwarsa")
	}

	// Pastikan user masih aktif
	user, err := s.userRepo.FindByID(ctx, rt.UserID)
	if err != nil {
		return helper.Fail(c, fiber.StatusUnauthorized, "User tidak ditemukan")
	}
	if !user.IsActive {
		return helper.Fail(c, fiber.StatusForbidden, "Akun tidak aktif")
	}

	// Revoke refresh token lama (rotation)
	_ = s.refreshRepo.RevokeByTokenHash(ctx, tokenHash)

	// Generate token pair baru
	tokenPair, err := s.generateTokenPair(ctx, user)
	if err != nil {
		return helper.Fail(c, fiber.StatusInternalServerError, "Gagal membuat token baru")
	}

	return helper.Success(c, fiber.StatusOK, "Token berhasil diperbarui", tokenPair)
}

// Logout godoc
//
// @Summary      Logout pengguna
// @Description  Mencabut refresh token pengguna sehingga tidak dapat digunakan kembali.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body body     model.LogoutRequest true "Refresh token yang akan dicabut"
// @Success      200  {object} model.WebResponse{data=string}
// @Failure      400  {object} model.ErrorResponse "Body bukan JSON yang valid atau token kosong"
// @Router       /auth/logout [post]
func (s *AuthService) Logout(c *fiber.Ctx) error {
	ctx, cancel := helper.ReqCtx(c)
	defer cancel()

	var req model.LogoutRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.Fail(c, fiber.StatusBadRequest, "Format JSON request body tidak valid")
	}

	if req.RefreshToken == "" {
		return helper.Fail(c, fiber.StatusBadRequest, "Refresh token wajib diisi")
	}

	// Hash token dan revoke
	tokenHash := helper.HashToken(req.RefreshToken)
	_ = s.refreshRepo.RevokeByTokenHash(ctx, tokenHash)

	return helper.Success(c, fiber.StatusOK, "Logout berhasil", "ok")
}

// Me godoc
//
// @Summary      Mendapatkan profil pengguna yang sedang login
// @Description  Mengembalikan informasi akun dan daftar permission pengguna.
// @Tags         auth
// @Produce      json
// @Success      200  {object} model.WebResponse{data=model.UserProfile}
// @Failure      401  {object} model.ErrorResponse "Belum terautentikasi"
// @Failure      404  {object} model.ErrorResponse "User tidak ditemukan"
// @Security     BearerAuth
// @Router       /auth/me [get]
func (s *AuthService) Me(c *fiber.Ctx) error {
	ctx, cancel := helper.ReqCtx(c)
	defer cancel()

	authUser, ok := c.Locals("auth_user").(model.AuthUser)
	if !ok {
		return helper.Fail(c, fiber.StatusUnauthorized, "Tidak terautentikasi")
	}

	userID := authUser.ID
	if userID == 0 {
		userID = authUser.UserID
	}

	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return helper.Fail(c, fiber.StatusNotFound, "User tidak ditemukan")
	}

	userPerms := s.perms.PermissionsFor(user.Role)
	if userPerms == nil {
		userPerms = []string{}
	}

	return helper.Success(c, fiber.StatusOK, "Data user berhasil diambil", model.UserProfile{
		User:        user,
		Permissions: userPerms,
	})
}

// generateTokenPair membuat access token dan refresh token, menyimpan hash refresh token ke DB.
func (s *AuthService) generateTokenPair(ctx context.Context, user model.User) (model.TokenPair, error) {
	// Generate access token
	accessToken, err := s.jwt.GenerateAccessToken(user.ID, user.Username, user.Role)
	if err != nil {
		return model.TokenPair{}, err
	}

	// Generate refresh token (cryptographically secure random)
	refreshToken, err := helper.GenerateRefreshToken()
	if err != nil {
		return model.TokenPair{}, err
	}

	// Simpan HASH refresh token ke database
	tokenHash := helper.HashToken(refreshToken)
	expiresAt := time.Now().Add(helper.RefreshTTL())

	if err := s.refreshRepo.Create(ctx, user.ID, tokenHash, expiresAt); err != nil {
		return model.TokenPair{}, err
	}

	return model.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}
