package main

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"api-students-db/app/model"
	"api-students-db/app/repository"
)

// studentRepo disuntikkan dari main.go saat aplikasi start.
var studentRepo repository.StudentRepository

func terjemahkanError(c *fiber.Ctx, err error, pesanUmum string) error {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		return c.Status(fiber.StatusNotFound).JSON(model.APIResponse{
			Success: false, Message: "Data mahasiswa tidak ditemukan",
		})
	case errors.Is(err, repository.ErrDuplicate):
		return c.Status(fiber.StatusConflict).JSON(model.APIResponse{
			Success: false, Message: "NIM sudah terdaftar dalam sistem",
		})
	default:
		return c.Status(fiber.StatusInternalServerError).JSON(model.APIResponse{
			Success: false, Message: pesanUmum,
		})
	}
}

// GET /api/v1/students
func GetStudents(c *fiber.Ctx) error {
	ctx, cancel := reqCtx(c)
	defer cancel()

	q := parseListQuery(c)

	students, total, err := studentRepo.FindAll(ctx, q)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(model.APIResponse{
			Success: false, Message: "Gagal mengambil daftar mahasiswa",
		})
	}

	totalPages := 0
	if q.Limit > 0 {
		totalPages = (total + q.Limit - 1) / q.Limit
	}

	return c.Status(fiber.StatusOK).JSON(model.APIResponse{
		Success: true,
		Message: "Berhasil mengambil daftar mahasiswa",
		Data:    students,
		Meta: &model.Meta{
			Page: q.Page, Limit: q.Limit, Total: total, TotalPages: totalPages,
		},
	})
}

// GET /api/v1/students/:id
func GetStudentByID(c *fiber.Ctx) error {
	ctx, cancel := reqCtx(c)
	defer cancel()

	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(model.APIResponse{
			Success: false, Message: "ID harus berupa angka integer",
		})
	}

	student, err := studentRepo.FindByID(ctx, id)
	if err != nil {
		return terjemahkanError(c, err, "Gagal mengambil data mahasiswa")
	}

	return c.Status(fiber.StatusOK).JSON(model.APIResponse{
		Success: true, Message: "Data mahasiswa ditemukan", Data: student,
	})
}

// POST /api/v1/students
func CreateStudent(c *fiber.Ctx) error {
	ctx, cancel := reqCtx(c)
	defer cancel()

	if err := checkContentTypeJSON(c); err != nil {
		return err
	}

	var req model.CreateStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(model.APIResponse{
			Success: false, Message: "Format JSON request body tidak valid",
		})
	}

	errorsMap := make(map[string]string)
	if len(req.NIM) == 0 {
		errorsMap["nim"] = "NIM wajib diisi"
	}
	if len(req.Name) == 0 {
		errorsMap["name"] = "Nama wajib diisi"
	}
	if req.Grade == nil {
		errorsMap["grade"] = "Grade wajib diisi"
	} else if *req.Grade < 0 || *req.Grade > 100 {
		errorsMap["grade"] = "Grade harus bernilai antara 0 sampai 100"
	}
	if req.IsActive == nil {
		errorsMap["is_active"] = "IsActive wajib diisi"
	}

	if len(errorsMap) > 0 {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(model.APIResponse{
			Success: false, Message: "Validasi isi permintaan gagal", Errors: errorsMap,
		})
	}

	// Keunikan NIM TIDAK diperiksa manual lewat SELECT lebih dulu —
	// UNIQUE INDEX di database yang menjaminnya, tanpa celah race condition.
	baru, err := studentRepo.Create(ctx, model.Student{
		NIM: req.NIM, Name: req.Name, Grade: *req.Grade, IsActive: *req.IsActive,
	})
	if err != nil {
		return terjemahkanError(c, err, "Gagal menyimpan data mahasiswa")
	}

	c.Set("Location", "/api/v1/students/"+strconv.Itoa(baru.ID))
	return c.Status(fiber.StatusCreated).JSON(model.APIResponse{
		Success: true, Message: "Mahasiswa berhasil ditambahkan", Data: baru,
	})
}

// PUT /api/v1/students/:id
func ReplaceStudent(c *fiber.Ctx) error {
	ctx, cancel := reqCtx(c)
	defer cancel()

	if err := checkContentTypeJSON(c); err != nil {
		return err
	}

	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(model.APIResponse{
			Success: false, Message: "ID harus berupa angka integer",
		})
	}

	var req model.CreateStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(model.APIResponse{
			Success: false, Message: "Format JSON tidak valid",
		})
	}

	errorsMap := make(map[string]string)
	if len(req.NIM) == 0 {
		errorsMap["nim"] = "NIM wajib diisi"
	}
	if len(req.Name) == 0 {
		errorsMap["name"] = "Nama wajib diisi"
	}
	if req.Grade == nil {
		errorsMap["grade"] = "Grade wajib diisi"
	}
	if req.IsActive == nil {
		errorsMap["is_active"] = "IsActive wajib diisi"
	}

	if len(errorsMap) > 0 {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(model.APIResponse{
			Success: false, Message: "PUT membutuhkan seluruh field dikirim ulang lengkap", Errors: errorsMap,
		})
	}

	hasil, err := studentRepo.Update(ctx, model.Student{
		ID: id, NIM: req.NIM, Name: req.Name, Grade: *req.Grade, IsActive: *req.IsActive,
	})
	if err != nil {
		return terjemahkanError(c, err, "Gagal memperbarui data mahasiswa")
	}

	return c.Status(fiber.StatusOK).JSON(model.APIResponse{
		Success: true, Message: "Data mahasiswa berhasil diganti secara keseluruhan (PUT)", Data: hasil,
	})
}

// PATCH /api/v1/students/:id
func UpdateStudentPartial(c *fiber.Ctx) error {
	ctx, cancel := reqCtx(c)
	defer cancel()

	if err := checkContentTypeJSON(c); err != nil {
		return err
	}

	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(model.APIResponse{
			Success: false, Message: "ID harus berupa angka integer",
		})
	}

	var req model.UpdateStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(model.APIResponse{
			Success: false, Message: "Format JSON tidak valid",
		})
	}

	if req.Grade != nil && (*req.Grade < 0 || *req.Grade > 100) {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(model.APIResponse{
			Success: false, Message: "Validasi gagal", Errors: map[string]string{"grade": "Grade harus 0-100"},
		})
	}

	// PATCH = baca dulu dari database, ubah field yang dikirim saja, simpan kembali.
	saatIni, err := studentRepo.FindByID(ctx, id)
	if err != nil {
		return terjemahkanError(c, err, "Gagal mengambil data mahasiswa")
	}

	if req.NIM != nil {
		saatIni.NIM = *req.NIM
	}
	if req.Name != nil {
		saatIni.Name = *req.Name
	}
	if req.Grade != nil {
		saatIni.Grade = *req.Grade
	}
	if req.IsActive != nil {
		saatIni.IsActive = *req.IsActive
	}

	hasil, err := studentRepo.Update(ctx, saatIni)
	if err != nil {
		return terjemahkanError(c, err, "Gagal memperbarui data mahasiswa")
	}

	return c.Status(fiber.StatusOK).JSON(model.APIResponse{
		Success: true, Message: "Sebagian data mahasiswa berhasil diperbarui (PATCH)", Data: hasil,
	})
}

// DELETE /api/v1/students/:id
func DeleteStudent(c *fiber.Ctx) error {
	ctx, cancel := reqCtx(c)
	defer cancel()

	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(model.APIResponse{
			Success: false, Message: "ID harus berupa angka integer",
		})
	}

	if err := studentRepo.Delete(ctx, id); err != nil {
		return terjemahkanError(c, err, "Gagal menghapus data mahasiswa")
	}

	return c.SendStatus(fiber.StatusNoContent)
}