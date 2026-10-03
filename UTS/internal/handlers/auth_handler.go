package handlers

import (
	"github.com/gofiber/fiber/v2"

	"siakad/internal/models"
	"siakad/internal/repositories"
	"siakad/internal/utils"
)

type AuthHandler struct {
	repo *repositories.AuthRepository
}

func NewAuthHandler(repo *repositories.AuthRepository) *AuthHandler {
	return &AuthHandler{repo: repo}
}

func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req models.LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.SendError(c, fiber.StatusUnprocessableEntity, "Validasi gagal", nil)
	}

	errs := utils.ValidateStruct(req)
	if errs != nil {
		return utils.SendError(c, fiber.StatusUnprocessableEntity, "Validasi gagal", errs)
	}

	ctx := c.Context()
	user, err := h.repo.FindUserByEmail(ctx, req.Email)
	if err != nil {
		return utils.ErrorHandler(c, err)
	}

	if user == nil || !utils.CheckPasswordHash(req.Password, user.Password) {
		return utils.SendError(c, fiber.StatusUnauthorized, "Credential salah", nil)
	}

	token, expiresIn, err := utils.GenerateToken(user.ID, user.Email, user.Role)
	if err != nil {
		return utils.ErrorHandler(c, err)
	}

	res := models.LoginResponse{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresIn:   expiresIn,
	}
	res.User.ID = user.ID
	res.User.Email = user.Email
	res.User.Role = user.Role

	return utils.SendSuccess(c, fiber.StatusOK, "Login berhasil", res)
}

func (h *AuthHandler) Me(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int)
	ctx := c.Context()

	user, err := h.repo.GetUserByID(ctx, userID)
	if err != nil {
		return utils.ErrorHandler(c, err)
	}
	if user == nil {
		return utils.SendError(c, fiber.StatusUnauthorized, "User tidak ditemukan", nil)
	}

	res := models.MeResponse{
		User: *user,
	}
	res.User.Password = "" // ensure password is not included (already ignored in json)

	if user.Role == "mahasiswa" {
		student, err := h.repo.GetMeStudentData(ctx, user.ID)
		if err != nil {
			return utils.ErrorHandler(c, err)
		}
		if student == nil {
			return utils.SendError(c, fiber.StatusUnauthorized, "Mahasiswa tidak ditemukan atau dihapus", nil)
		}
		res.Student = student
	}

	return utils.SendSuccess(c, fiber.StatusOK, "Data user berhasil diambil", res)
}
