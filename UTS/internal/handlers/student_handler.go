package handlers

import (
	"math"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"

	"siakad/internal/models"
	"siakad/internal/repositories"
	"siakad/internal/utils"
)

type StudentHandler struct {
	repo *repositories.StudentRepository
}

func NewStudentHandler(repo *repositories.StudentRepository) *StudentHandler {
	return &StudentHandler{repo: repo}
}

func (h *StudentHandler) List(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	perPage, _ := strconv.Atoi(c.Query("per_page", "10"))

	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 10
	}
	if perPage > 50 {
		perPage = 50
	}

	prodi := c.Query("prodi", "")
	angkatan := c.Query("angkatan", "")
	search := c.Query("search", "")
	sort := c.Query("sort", "")

	students, total, err := h.repo.List(c.Context(), page, perPage, prodi, angkatan, search, sort)
	if err != nil {
		return utils.ErrorHandler(c, err)
	}

	meta := models.Meta{
		CurrentPage: page,
		PerPage:     perPage,
		Total:       total,
		LastPage:    int(math.Ceil(float64(total) / float64(perPage))),
	}
	if meta.LastPage == 0 {
		meta.LastPage = 1
	}

	return utils.SendSuccessWithMeta(c, fiber.StatusOK, "Data mahasiswa berhasil diambil", students, meta)
}

func (h *StudentHandler) Create(c *fiber.Ctx) error {
	var req models.CreateStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.SendError(c, fiber.StatusUnprocessableEntity, "Validasi gagal", nil)
	}

	if errs := utils.ValidateStruct(req); errs != nil {
		return utils.SendError(c, fiber.StatusUnprocessableEntity, "Validasi gagal", errs)
	}

	yearNow := time.Now().Year()
	angkatanInt, err := strconv.Atoi(req.Angkatan)
	if err != nil || angkatanInt > yearNow {
		errs := map[string][]string{"angkatan": {"Angkatan tidak valid atau melebihi tahun berjalan"}}
		return utils.SendError(c, fiber.StatusUnprocessableEntity, "Validasi gagal", errs)
	}

	hashPass, err := utils.HashPassword(req.NIM)
	if err != nil {
		return utils.ErrorHandler(c, err)
	}

	student := models.Student{
		NIM:      req.NIM,
		Nama:     req.Nama,
		Prodi:    req.Prodi,
		Angkatan: req.Angkatan,
	}
	if req.IPKTerakhir != nil {
		student.IPKTerakhir = *req.IPKTerakhir
	}

	err = h.repo.CreateWithUser(c.Context(), &student, req.Email, hashPass)
	if err != nil {
		// simplify duplicate check
		return utils.SendError(c, fiber.StatusUnprocessableEntity, "Validasi gagal: duplicate email atau NIM", nil)
	}

	return utils.SendSuccess(c, fiber.StatusCreated, "Mahasiswa berhasil dibuat", student)
}

func (h *StudentHandler) Get(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return utils.SendError(c, fiber.StatusNotFound, "Mahasiswa tidak ditemukan", nil)
	}

	role := c.Locals("role").(string)
	userID := c.Locals("user_id").(int)

	student, err := h.repo.FindByID(c.Context(), id)
	if err != nil {
		return utils.ErrorHandler(c, err)
	}
	if student == nil {
		return utils.SendError(c, fiber.StatusNotFound, "Mahasiswa tidak ditemukan", nil)
	}

	if role == "mahasiswa" && student.UserID != userID {
		return utils.SendError(c, fiber.StatusForbidden, "Akses ditolak", nil)
	}

	courses, totalSKS, err := h.repo.GetStudentCourses(c.Context(), student.ID)
	if err != nil {
		return utils.ErrorHandler(c, err)
	}

	batasSKS := repositories.GetBatasSKS(student.IPKTerakhir)

	res := models.StudentDetailResponse{
		Student:  *student,
		Courses:  courses,
		TotalSKS: totalSKS,
		BatasSKS: batasSKS,
	}

	return utils.SendSuccess(c, fiber.StatusOK, "Detail mahasiswa berhasil diambil", res)
}

func (h *StudentHandler) Update(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return utils.SendError(c, fiber.StatusNotFound, "Mahasiswa tidak ditemukan", nil)
	}

	var req models.UpdateStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.SendError(c, fiber.StatusUnprocessableEntity, "Validasi gagal", nil)
	}

	if errs := utils.ValidateStruct(req); errs != nil {
		return utils.SendError(c, fiber.StatusUnprocessableEntity, "Validasi gagal", errs)
	}

	yearNow := time.Now().Year()
	angkatanInt, err := strconv.Atoi(req.Angkatan)
	if err != nil || angkatanInt > yearNow {
		errs := map[string][]string{"angkatan": {"Angkatan tidak valid atau melebihi tahun berjalan"}}
		return utils.SendError(c, fiber.StatusUnprocessableEntity, "Validasi gagal", errs)
	}

	existing, err := h.repo.FindByID(c.Context(), id)
	if err != nil {
		return utils.ErrorHandler(c, err)
	}
	if existing == nil {
		return utils.SendError(c, fiber.StatusNotFound, "Mahasiswa tidak ditemukan", nil)
	}

	existing.Nama = req.Nama
	existing.Prodi = req.Prodi
	existing.Angkatan = req.Angkatan
	if req.IPKTerakhir != nil {
		existing.IPKTerakhir = *req.IPKTerakhir
	}

	if err := h.repo.Update(c.Context(), id, existing); err != nil {
		return utils.ErrorHandler(c, err)
	}

	return utils.SendSuccess(c, fiber.StatusOK, "Mahasiswa berhasil diubah", existing)
}

func (h *StudentHandler) Delete(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return utils.SendError(c, fiber.StatusNotFound, "Mahasiswa tidak ditemukan", nil)
	}

	existing, err := h.repo.FindByID(c.Context(), id)
	if err != nil {
		return utils.ErrorHandler(c, err)
	}
	if existing == nil {
		return utils.SendError(c, fiber.StatusNotFound, "Mahasiswa tidak ditemukan", nil)
	}

	if err := h.repo.SoftDelete(c.Context(), id); err != nil {
		return utils.ErrorHandler(c, err)
	}

	return c.Status(fiber.StatusNoContent).Send(nil)
}
