package service

import (
	"encoding/csv"
	"errors"
	"strconv"
	"strings"

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

// List mengembalikan daftar mahasiswa menggunakan cursor-based pagination.
// Mendukung content negotiation: JSON (default), CSV, 406 untuk format lain.
func (s *StudentService) List(c *fiber.Ctx) error {
	ctx, cancel := helper.ReqCtx(c)
	defer cancel()

	// Parse cursor query
	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	if limit < 1 {
		limit = 10
	} else if limit > 50 {
		limit = 50
	}

	q := model.CursorQuery{
		Limit:  limit,
		Cursor: c.Query("cursor", ""),
		Search: c.Query("search", ""),
	}

	students, hasMore, nextCursor, err := s.repo.FindAllCursor(ctx, q)
	if err != nil {
		return helper.ErrInternal("Gagal mengambil daftar mahasiswa", err)
	}

	// Content negotiation berdasarkan Accept header
	accept := c.Get("Accept", "application/json")
	switch {
	case strings.Contains(accept, "text/csv"):
		return respondCSV(c, students)
	case strings.Contains(accept, "application/json"),
		strings.Contains(accept, "*/*"),
		accept == "":
		return respondStudentListJSON(c, students, q.Limit, hasMore, nextCursor)
	default:
		return helper.ErrNotAcceptable("Format yang diminta tidak didukung. Gunakan application/json atau text/csv")
	}
}

// respondStudentListJSON mengirim response JSON dengan cursor metadata.
func respondStudentListJSON(c *fiber.Ctx, students []model.Student, limit int, hasMore bool, nextCursor string) error {
	meta := model.CursorMeta{
		Limit:   limit,
		HasMore: hasMore,
	}
	if hasMore {
		meta.NextCursor = nextCursor
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": "Berhasil mengambil daftar mahasiswa",
		"data":    students,
		"meta":    meta,
	})
}

// respondCSV mengirim response CSV menggunakan encoding/csv.
func respondCSV(c *fiber.Ctx, students []model.Student) error {
	c.Set("Content-Type", "text/csv; charset=utf-8")
	c.Set("Content-Disposition", `attachment; filename="students.csv"`)

	c.Status(fiber.StatusOK)

	writer := csv.NewWriter(c)

	// Header
	if err := writer.Write([]string{"id", "nim", "name", "grade", "is_active", "owner_id", "created_at"}); err != nil {
		return helper.ErrInternal("Gagal menulis CSV header", err)
	}

	// Rows
	for _, s := range students {
		record := []string{
			strconv.Itoa(s.ID),
			s.NIM,
			s.Name,
			strconv.FormatFloat(s.Grade, 'f', 2, 64),
			strconv.FormatBool(s.IsActive),
			strconv.Itoa(s.OwnerID),
			s.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
		if err := writer.Write(record); err != nil {
			return helper.ErrInternal("Gagal menulis CSV row", err)
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return helper.ErrInternal("Gagal flush CSV", err)
	}

	return nil
}

func (s *StudentService) Get(c *fiber.Ctx) error {
	ctx, cancel := helper.ReqCtx(c)
	defer cancel()

	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return helper.ErrBadRequest("ID harus berupa angka integer")
	}

	student, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return translateStudentError(err, "Gagal mengambil data mahasiswa")
	}

	// Ownership check: owner boleh akses, non-owner perlu student:read:any
	authUser, ok := currentUser(c)
	if !ok {
		return helper.ErrUnauthorized("Token autentikasi diperlukan")
	}
	if !CanAccessStudent(authUser, student.OwnerID, s.perms, "student:read:any") {
		return helper.ErrForbidden("Anda tidak memiliki izin untuk mengakses data ini")
	}

	return helper.Success(c, fiber.StatusOK, "Data mahasiswa ditemukan", student)
}

func (s *StudentService) Create(c *fiber.Ctx) error {
	ctx, cancel := helper.ReqCtx(c)
	defer cancel()

	var req model.CreateStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.ErrBadRequest("Format JSON request body tidak valid")
	}

	// Declarative validation menggunakan tags
	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.NewValidationError(errs)
	}

	// owner_id otomatis dari user yang sedang login, BUKAN dari request body
	authUser, ok := currentUser(c)
	if !ok {
		return helper.ErrUnauthorized("Token autentikasi diperlukan")
	}

	baru, err := s.repo.Create(ctx, model.Student{
		NIM: req.NIM, Name: req.Name, Grade: *req.Grade, IsActive: *req.IsActive,
		OwnerID: authUser.ID,
	})
	if err != nil {
		return translateStudentError(err, "Gagal menyimpan data mahasiswa")
	}

	return helper.Created(c, "Mahasiswa berhasil ditambahkan", baru,
		"/api/v1/students/"+strconv.Itoa(baru.ID))
}

func (s *StudentService) Replace(c *fiber.Ctx) error {
	ctx, cancel := helper.ReqCtx(c)
	defer cancel()

	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return helper.ErrBadRequest("ID harus berupa angka integer")
	}

	// Fetch data existing untuk cek ownership
	existing, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return translateStudentError(err, "Gagal mengambil data mahasiswa")
	}

	// Ownership check: owner boleh update, non-owner perlu student:update:any
	authUser, ok := currentUser(c)
	if !ok {
		return helper.ErrUnauthorized("Token autentikasi diperlukan")
	}
	if !CanAccessStudent(authUser, existing.OwnerID, s.perms, "student:update:any") {
		return helper.ErrForbidden("Anda tidak memiliki izin untuk mengubah data ini")
	}

	var req model.CreateStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.ErrBadRequest("Format JSON tidak valid")
	}

	// Declarative validation
	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.NewValidationError(errs)
	}

	// owner_id TIDAK dapat diubah melalui PUT — gunakan existing.OwnerID
	hasil, err := s.repo.Update(ctx, model.Student{
		ID: id, NIM: req.NIM, Name: req.Name, Grade: *req.Grade, IsActive: *req.IsActive,
	})
	if err != nil {
		return translateStudentError(err, "Gagal memperbarui data mahasiswa")
	}

	return helper.Success(c, fiber.StatusOK,
		"Data mahasiswa berhasil diganti secara keseluruhan (PUT)", hasil)
}

func (s *StudentService) Patch(c *fiber.Ctx) error {
	ctx, cancel := helper.ReqCtx(c)
	defer cancel()

	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return helper.ErrBadRequest("ID harus berupa angka integer")
	}

	var req model.UpdateStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.ErrBadRequest("Format JSON tidak valid")
	}

	// Cek PATCH kosong — aturan antar-field, tidak bisa dari validation tag
	if req.IsEmptyPatch() {
		return helper.ErrBadRequest("Minimal satu field harus dikirim untuk PATCH")
	}

	// Cek apakah field yang dikirim berisi string kosong (misalnya name:"")
	if req.Name != nil && *req.Name == "" {
		return helper.NewValidationError(map[string]string{
			"name": "name tidak boleh kosong jika dikirim",
		})
	}
	if req.NIM != nil && *req.NIM == "" {
		return helper.NewValidationError(map[string]string{
			"nim": "nim tidak boleh kosong jika dikirim",
		})
	}

	// Declarative validation untuk field yang dikirim
	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.NewValidationError(errs)
	}

	current, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return translateStudentError(err, "Gagal mengambil data mahasiswa")
	}

	// Ownership check: owner boleh update, non-owner perlu student:update:any
	authUser, ok := currentUser(c)
	if !ok {
		return helper.ErrUnauthorized("Token autentikasi diperlukan")
	}
	if !CanAccessStudent(authUser, current.OwnerID, s.perms, "student:update:any") {
		return helper.ErrForbidden("Anda tidak memiliki izin untuk mengubah data ini")
	}

	// owner_id TIDAK dapat diubah melalui PATCH
	updated := ApplyPatch(current, req)

	hasil, err := s.repo.Update(ctx, updated)
	if err != nil {
		return translateStudentError(err, "Gagal memperbarui data mahasiswa")
	}

	return helper.Success(c, fiber.StatusOK,
		"Sebagian data mahasiswa berhasil diperbarui (PATCH)", hasil)
}

func (s *StudentService) Delete(c *fiber.Ctx) error {
	ctx, cancel := helper.ReqCtx(c)
	defer cancel()

	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return helper.ErrBadRequest("ID harus berupa angka integer")
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return translateStudentError(err, "Gagal menghapus data mahasiswa")
	}

	return helper.NoContent(c)
}

// translateStudentError mengonversi repository error menjadi AppError.
// Raw database error TIDAK pernah dikirim ke client — hanya masuk ke log.
func translateStudentError(err error, pesanUmum string) error {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		return helper.ErrNotFoundMsg("Data mahasiswa tidak ditemukan")
	case errors.Is(err, repository.ErrDuplicate):
		return helper.ErrConflict("NIM sudah terdaftar dalam sistem")
	default:
		return helper.ErrInternal(pesanUmum, err)
	}
}