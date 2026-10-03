package handlers

import (
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"

	"siakad/internal/models"
	"siakad/internal/repositories"
	"siakad/internal/utils"
)

type EnrollmentHandler struct {
	repo        *repositories.EnrollmentRepository
	studentRepo *repositories.StudentRepository
}

func NewEnrollmentHandler(repo *repositories.EnrollmentRepository, studentRepo *repositories.StudentRepository) *EnrollmentHandler {
	return &EnrollmentHandler{repo: repo, studentRepo: studentRepo}
}

func (h *EnrollmentHandler) Create(c *fiber.Ctx) error {
	var req models.CreateEnrollmentRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.SendError(c, fiber.StatusUnprocessableEntity, "Validasi gagal", nil)
	}
	if errs := utils.ValidateStruct(req); errs != nil {
		return utils.SendError(c, fiber.StatusUnprocessableEntity, "Validasi gagal", errs)
	}

	userID := c.Locals("user_id").(int)
	ctx := c.Context()

	student, err := h.studentRepo.FindByUserID(ctx, userID)
	if err != nil {
		return utils.ErrorHandler(c, err)
	}
	if student == nil {
		return utils.SendError(c, fiber.StatusForbidden, "Hanya mahasiswa yang dapat melakukan krs", nil)
	}

	batasSKS := repositories.GetBatasSKS(student.IPKTerakhir)

	err = h.repo.Create(ctx, student.ID, &req, batasSKS)
	if err != nil {
		msg := err.Error()
		if msg == "course_not_found" {
			return utils.SendError(c, fiber.StatusUnprocessableEntity, "Mata kuliah tidak ditemukan", nil)
		}
		if msg == "duplicate_enrollment" {
			return utils.SendError(c, fiber.StatusConflict, "Mata kuliah sudah pernah diambil pada tahun akademik ini", nil)
		}
		if msg == "kuota_penuh" {
			return utils.SendError(c, fiber.StatusUnprocessableEntity, "Kuota penuh", nil)
		}
		if strings.HasPrefix(msg, "sks_limit:") {
			sisa := strings.Split(msg, ":")[1]
			return utils.SendError(c, fiber.StatusUnprocessableEntity, "Total SKS melebihi batas. Sisa SKS: "+sisa, nil)
		}
		return utils.ErrorHandler(c, err)
	}

	return utils.SendSuccess(c, fiber.StatusCreated, "Enrollment berhasil ditambahkan", nil)
}

func (h *EnrollmentHandler) Delete(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return utils.SendError(c, fiber.StatusNotFound, "Enrollment tidak ditemukan", nil)
	}

	userID := c.Locals("user_id").(int)
	ctx := c.Context()

	student, err := h.studentRepo.FindByUserID(ctx, userID)
	if err != nil {
		return utils.ErrorHandler(c, err)
	}
	if student == nil {
		return utils.SendError(c, fiber.StatusForbidden, "Hanya mahasiswa yang dapat menghapus krs", nil)
	}

	err = h.repo.Delete(ctx, id, student.ID)
	if err != nil {
		if err.Error() == "enrollment_not_found" {
			return utils.SendError(c, fiber.StatusNotFound, "Enrollment tidak ditemukan", nil)
		}
		if err.Error() == "forbidden_enrollment" {
			return utils.SendError(c, fiber.StatusForbidden, "Akses ditolak: bukan milik Anda", nil)
		}
		return utils.ErrorHandler(c, err)
	}

	return c.Status(fiber.StatusNoContent).Send(nil)
}
