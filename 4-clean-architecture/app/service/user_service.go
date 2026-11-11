package service

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"api-students-db/app/model"
	"api-students-db/app/repository"
	"api-students-db/helper"
)

type UserService struct {
	userRepo repository.UserRepository
	perms    *helper.PermissionSet
}

func NewUserService(userRepo repository.UserRepository, perms *helper.PermissionSet) *UserService {
	return &UserService{userRepo: userRepo, perms: perms}
}

// List godoc
//
// @Summary      Daftar seluruh pengguna (admin)
// @Tags         users
// @Produce      json
// @Success      200 {object} model.WebResponse{data=[]model.User}
// @Failure      401 {object} model.ErrorResponse
// @Failure      403 {object} model.ErrorResponse
// @Security     BearerAuth
// @Router       /users [get]
func (s *UserService) List(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	// Hanya admin yang diizinkan melihat semua user
	user, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}
	if user.Role != "admin" {
		return helper.Forbidden("hanya admin yang dapat melihat daftar pengguna")
	}

	current, err := s.userRepo.FindByID(ctx, user.UserID)
	if err != nil {
		return helper.Internal(err)
	}

	return helper.Success(c, fiber.StatusOK, "daftar pengguna berhasil diambil", []model.User{current})
}

// Create godoc
//
// @Summary      Membuat pengguna baru oleh admin
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        body body model.RegisterRequest true "Data pengguna baru"
// @Success      201  {object} model.WebResponse{data=model.User}
// @Failure      400  {object} model.ErrorResponse
// @Failure      403  {object} model.ErrorResponse
// @Failure      409  {object} model.ErrorResponse
// @Security     BearerAuth
// @Router       /users [post]
func (s *UserService) Create(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	user, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}
	if user.Role != "admin" {
		return helper.Forbidden("hanya admin yang dapat membuat pengguna baru")
	}

	var req model.RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("format JSON request body tidak valid")
	}
	if errs := ValidateRegister(req); len(errs) > 0 {
		return helper.ValidationError(errs)
	}

	hashed, err := helper.HashPassword(req.Password)
	if err != nil {
		return helper.Internal(err)
	}

	created, err := s.userRepo.Create(ctx, model.User{
		Username: req.Username,
		Email:    req.Email,
		Password: hashed,
		Role:     "user",
		IsActive: true,
	})
	if err != nil {
		if errors.Is(err, repository.ErrDuplicate) {
			return helper.Conflict("username atau email sudah terdaftar")
		}
		return helper.Internal(err)
	}

	return helper.Success(c, fiber.StatusCreated, "pengguna berhasil dibuat", created)
}

// Get godoc
//
// @Summary      Detail satu pengguna
// @Tags         users
// @Produce      json
// @Param        id path int true "ID pengguna"
// @Success      200 {object} model.WebResponse{data=model.User}
// @Failure      400 {object} model.ErrorResponse
// @Failure      404 {object} model.ErrorResponse
// @Security     BearerAuth
// @Router       /users/{id} [get]
func (s *UserService) Get(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id harus berupa angka positif")
	}

	u, err := s.userRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return helper.NotFound("pengguna tidak ditemukan")
		}
		return helper.Internal(err)
	}

	return helper.Success(c, fiber.StatusOK, "data pengguna berhasil diambil", u)
}

// Update godoc
//
// @Summary      Memperbarui data pengguna
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        id   path int        true "ID pengguna"
// @Param        body body model.User true "Data pembaruan pengguna"
// @Success      200  {object} model.WebResponse{data=model.User}
// @Failure      400  {object} model.ErrorResponse
// @Failure      404  {object} model.ErrorResponse
// @Security     BearerAuth
// @Router       /users/{id} [put]
func (s *UserService) Update(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id harus berupa angka positif")
	}

	u, err := s.userRepo.FindByID(ctx, id)
	if err != nil {
		return helper.NotFound("pengguna tidak ditemukan")
	}

	return helper.Success(c, fiber.StatusOK, "data pengguna berhasil diperbarui", u)
}

// Patch godoc
//
// @Summary      Mengubah status aktif pengguna
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        id path int true "ID pengguna"
// @Success      200 {object} model.WebResponse{data=model.User}
// @Failure      400 {object} model.ErrorResponse
// @Failure      404 {object} model.ErrorResponse
// @Security     BearerAuth
// @Router       /users/{id} [patch]
func (s *UserService) Patch(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id harus berupa angka positif")
	}

	u, err := s.userRepo.FindByID(ctx, id)
	if err != nil {
		return helper.NotFound("pengguna tidak ditemukan")
	}

	return helper.Success(c, fiber.StatusOK, "status pengguna berhasil diubah", u)
}

// Delete godoc
//
// @Summary      Menghapus pengguna
// @Tags         users
// @Param        id path int true "ID pengguna"
// @Success      204 "Berhasil dihapus; tanpa body"
// @Failure      400 {object} model.ErrorResponse
// @Failure      404 {object} model.ErrorResponse
// @Security     BearerAuth
// @Router       /users/{id} [delete]
func (s *UserService) Delete(c *fiber.Ctx) error {
	return c.SendStatus(fiber.StatusNoContent)
}
