package service

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"api-students-db/app/model"
	"api-students-db/app/repository"
	"api-students-db/helper"
)

type StudentService struct {
	repo  repository.StudentRepository
	perms *helper.PermissionSet
}

func NewStudentService(repo repository.StudentRepository, perms *helper.PermissionSet) *StudentService {
	return &StudentService{repo: repo, perms: perms}
}

// currentUser mengekstrak AuthUser dari Fiber Locals.
// Digunakan oleh handler methods (bukan pure function).
func currentUser(c *fiber.Ctx) (model.AuthUser, bool) {
	u, ok := c.Locals("auth_user").(model.AuthUser)
	return u, ok
}

func (s *StudentService) List(c *fiber.Ctx) error {
	ctx, cancel := helper.ReqCtx(c)
	defer cancel()

	q := helper.ParseListQuery(c)

	students, total, err := s.repo.FindAll(ctx, q)
	if err != nil {
		return helper.Fail(c, fiber.StatusInternalServerError, "Gagal mengambil daftar mahasiswa")
	}

	return helper.SuccessList(c, "Berhasil mengambil daftar mahasiswa", students, &model.Meta{
		Page: q.Page, Limit: q.Limit, Total: total,
		TotalPages: CountTotalPages(total, q.Limit),
	})
}

func (s *StudentService) Get(c *fiber.Ctx) error {
	ctx, cancel := helper.ReqCtx(c)
	defer cancel()

	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return helper.Fail(c, fiber.StatusBadRequest, "ID harus berupa angka integer")
	}

	student, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return translateError(c, err, "Gagal mengambil data mahasiswa")
	}

	// Ownership check: owner boleh akses, non-owner perlu student:read:any
	authUser, ok := currentUser(c)
	if !ok {
		return helper.Fail(c, fiber.StatusUnauthorized, "Token autentikasi diperlukan")
	}
	if !CanAccessStudent(authUser, student.OwnerID, s.perms, "student:read:any") {
		return helper.Fail(c, fiber.StatusForbidden, "Anda tidak memiliki izin untuk mengakses data ini")
	}

	return helper.Success(c, fiber.StatusOK, "Data mahasiswa ditemukan", student)
}

func (s *StudentService) Create(c *fiber.Ctx) error {
	ctx, cancel := helper.ReqCtx(c)
	defer cancel()

	var req model.CreateStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.Fail(c, fiber.StatusBadRequest, "Format JSON request body tidak valid")
	}

	if errs := ValidateCreate(req); len(errs) > 0 {
		return helper.FailValidation(c, "Validasi isi permintaan gagal", errs)
	}

	// owner_id otomatis dari user yang sedang login, BUKAN dari request body
	authUser, ok := currentUser(c)
	if !ok {
		return helper.Fail(c, fiber.StatusUnauthorized, "Token autentikasi diperlukan")
	}

	baru, err := s.repo.Create(ctx, model.Student{
		NIM: req.NIM, Name: req.Name, Grade: *req.Grade, IsActive: *req.IsActive,
		OwnerID: authUser.ID,
	})
	if err != nil {
		return translateError(c, err, "Gagal menyimpan data mahasiswa")
	}

	return helper.Created(c, "Mahasiswa berhasil ditambahkan", baru,
		"/api/v1/students/"+strconv.Itoa(baru.ID))
}

func (s *StudentService) Replace(c *fiber.Ctx) error {
	ctx, cancel := helper.ReqCtx(c)
	defer cancel()

	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return helper.Fail(c, fiber.StatusBadRequest, "ID harus berupa angka integer")
	}

	// Fetch data existing untuk cek ownership
	existing, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return translateError(c, err, "Gagal mengambil data mahasiswa")
	}

	// Ownership check: owner boleh update, non-owner perlu student:update:any
	authUser, ok := currentUser(c)
	if !ok {
		return helper.Fail(c, fiber.StatusUnauthorized, "Token autentikasi diperlukan")
	}
	if !CanAccessStudent(authUser, existing.OwnerID, s.perms, "student:update:any") {
		return helper.Fail(c, fiber.StatusForbidden, "Anda tidak memiliki izin untuk mengubah data ini")
	}

	var req model.CreateStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.Fail(c, fiber.StatusBadRequest, "Format JSON tidak valid")
	}

	if errs := ValidateReplace(req); len(errs) > 0 {
		return helper.FailValidation(c, "PUT membutuhkan seluruh field dikirim ulang lengkap", errs)
	}

	// owner_id TIDAK dapat diubah melalui PUT — gunakan existing.OwnerID
	hasil, err := s.repo.Update(ctx, model.Student{
		ID: id, NIM: req.NIM, Name: req.Name, Grade: *req.Grade, IsActive: *req.IsActive,
	})
	if err != nil {
		return translateError(c, err, "Gagal memperbarui data mahasiswa")
	}

	return helper.Success(c, fiber.StatusOK,
		"Data mahasiswa berhasil diganti secara keseluruhan (PUT)", hasil)
}

func (s *StudentService) Patch(c *fiber.Ctx) error {
	ctx, cancel := helper.ReqCtx(c)
	defer cancel()

	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return helper.Fail(c, fiber.StatusBadRequest, "ID harus berupa angka integer")
	}

	var req model.UpdateStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.Fail(c, fiber.StatusBadRequest, "Format JSON tidak valid")
	}

	if errs := ValidatePatch(req); len(errs) > 0 {
		return helper.FailValidation(c, "Validasi gagal", errs)
	}

	current, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return translateError(c, err, "Gagal mengambil data mahasiswa")
	}

	// Ownership check: owner boleh update, non-owner perlu student:update:any
	authUser, ok := currentUser(c)
	if !ok {
		return helper.Fail(c, fiber.StatusUnauthorized, "Token autentikasi diperlukan")
	}
	if !CanAccessStudent(authUser, current.OwnerID, s.perms, "student:update:any") {
		return helper.Fail(c, fiber.StatusForbidden, "Anda tidak memiliki izin untuk mengubah data ini")
	}

	// owner_id TIDAK dapat diubah melalui PATCH
	updated := ApplyPatch(current, req)

	hasil, err := s.repo.Update(ctx, updated)
	if err != nil {
		return translateError(c, err, "Gagal memperbarui data mahasiswa")
	}

	return helper.Success(c, fiber.StatusOK,
		"Sebagian data mahasiswa berhasil diperbarui (PATCH)", hasil)
}

func (s *StudentService) Delete(c *fiber.Ctx) error {
	ctx, cancel := helper.ReqCtx(c)
	defer cancel()

	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return helper.Fail(c, fiber.StatusBadRequest, "ID harus berupa angka integer")
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return translateError(c, err, "Gagal menghapus data mahasiswa")
	}

	return helper.NoContent(c)
}

func translateError(c *fiber.Ctx, err error, pesanUmum string) error {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		return helper.Fail(c, fiber.StatusNotFound, "Data mahasiswa tidak ditemukan")
	case errors.Is(err, repository.ErrDuplicate):
		return helper.Fail(c, fiber.StatusConflict, "NIM sudah terdaftar dalam sistem")
	default:
		return helper.Fail(c, fiber.StatusInternalServerError, pesanUmum)
	}
}