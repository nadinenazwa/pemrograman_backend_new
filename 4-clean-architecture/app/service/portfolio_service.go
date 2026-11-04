package service

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"api-students-db/app/model"
	"api-students-db/app/repository"
	"api-students-db/helper"
)

// PortfolioService menangani HTTP request dan business logic portofolio mahasiswa.
// Mengikuti pola yang sama dengan StudentService:
//   - Validasi input
//   - Verifikasi mahasiswa ada di PostgreSQL
//   - Delegasi operasi MongoDB ke PortfolioRepository
//   - Translasi error ke AppError
type PortfolioService struct {
	portfolioRepo repository.PortfolioRepository
	studentRepo   repository.StudentRepository
	perms         *helper.PermissionSet
}

// NewPortfolioService membuat PortfolioService baru.
func NewPortfolioService(
	portfolioRepo repository.PortfolioRepository,
	studentRepo repository.StudentRepository,
	perms *helper.PermissionSet,
) *PortfolioService {
	return &PortfolioService{
		portfolioRepo: portfolioRepo,
		studentRepo:   studentRepo,
		perms:         perms,
	}
}

// ===== Handler Methods =====

// GetPortfolio menangani GET /students/:id/portfolio
// Memvalidasi ID, memastikan mahasiswa ada di PostgreSQL,
// lalu mengambil portofolio dari MongoDB.
func (s *PortfolioService) GetPortfolio(c *fiber.Ctx) error {
	ctx, cancel := helper.ReqCtx(c)
	defer cancel()

	studentID, err := parseStudentID(c)
	if err != nil {
		return err
	}

	// Verifikasi mahasiswa ada di PostgreSQL
	if _, err := s.studentRepo.FindByID(ctx, studentID); err != nil {
		return translatePortfolioStudentError(err)
	}

	portfolio, err := s.portfolioRepo.FindByStudentID(ctx, studentID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return helper.ErrNotFoundMsg("Portofolio mahasiswa belum tersedia")
		}
		return helper.ErrInternal("Gagal mengambil portofolio", err)
	}

	return helper.Success(c, fiber.StatusOK, "Portofolio mahasiswa ditemukan", portfolio)
}

// UpsertPortfolio menangani PUT /students/:id/portfolio
// Membuat portofolio baru atau mengganti seluruh isinya (semantik PUT).
// Satu mahasiswa tetap memiliki tepat satu portofolio berkat unique index MongoDB.
func (s *PortfolioService) UpsertPortfolio(c *fiber.Ctx) error {
	ctx, cancel := helper.ReqCtx(c)
	defer cancel()

	studentID, err := parseStudentID(c)
	if err != nil {
		return err
	}

	// Verifikasi mahasiswa ada di PostgreSQL
	student, err := s.studentRepo.FindByID(ctx, studentID)
	if err != nil {
		return translatePortfolioStudentError(err)
	}

	// Otorisasi: owner atau user dengan permission student:update:any
	authUser, ok := currentUser(c)
	if !ok {
		return helper.ErrUnauthorized("Token autentikasi diperlukan")
	}
	if !CanAccessStudent(authUser, student.OwnerID, s.perms, "student:update:any") {
		return helper.ErrForbidden("Anda tidak memiliki izin untuk mengubah portofolio ini")
	}

	var req model.UpsertPortfolioRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.ErrBadRequest("Format JSON request body tidak valid")
	}

	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.NewValidationError(errs)
	}

	portfolio, err := s.portfolioRepo.Upsert(ctx, studentID, req)
	if err != nil {
		return helper.ErrInternal("Gagal menyimpan portofolio", err)
	}

	return helper.Success(c, fiber.StatusOK, "Portofolio berhasil disimpan", portfolio)
}

// AddProject menangani POST /students/:id/portfolio/projects
// Menambahkan satu proyek ke array projects secara atomik.
// Batas maksimal 20 proyek dijaga di level MongoDB (atomic).
func (s *PortfolioService) AddProject(c *fiber.Ctx) error {
	ctx, cancel := helper.ReqCtx(c)
	defer cancel()

	studentID, err := parseStudentID(c)
	if err != nil {
		return err
	}

	// Verifikasi mahasiswa ada di PostgreSQL
	student, err := s.studentRepo.FindByID(ctx, studentID)
	if err != nil {
		return translatePortfolioStudentError(err)
	}

	// Otorisasi: owner atau user dengan permission student:update:any
	authUser, ok := currentUser(c)
	if !ok {
		return helper.ErrUnauthorized("Token autentikasi diperlukan")
	}
	if !CanAccessStudent(authUser, student.OwnerID, s.perms, "student:update:any") {
		return helper.ErrForbidden("Anda tidak memiliki izin untuk mengubah portofolio ini")
	}

	var req model.AddProjectRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.ErrBadRequest("Format JSON request body tidak valid")
	}

	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.NewValidationError(errs)
	}

	portfolio, limitReached, err := s.portfolioRepo.AddProject(ctx, studentID, req)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return helper.ErrNotFoundMsg("Portofolio mahasiswa belum tersedia. Buat portofolio terlebih dahulu.")
		}
		return helper.ErrInternal("Gagal menambahkan proyek", err)
	}

	if limitReached {
		return helper.ErrConflict("Batas maksimal 20 proyek sudah tercapai")
	}

	return helper.Success(c, fiber.StatusCreated, "Proyek berhasil ditambahkan", portfolio)
}

// DeleteProject menangani DELETE /students/:id/portfolio/projects/:projectId
// Menghapus satu proyek dari array projects menggunakan $pull.
func (s *PortfolioService) DeleteProject(c *fiber.Ctx) error {
	ctx, cancel := helper.ReqCtx(c)
	defer cancel()

	studentID, err := parseStudentID(c)
	if err != nil {
		return err
	}

	projectID := c.Params("projectId")
	if projectID == "" {
		return helper.ErrBadRequest("Project ID diperlukan")
	}

	// Verifikasi mahasiswa ada di PostgreSQL
	student, err := s.studentRepo.FindByID(ctx, studentID)
	if err != nil {
		return translatePortfolioStudentError(err)
	}

	// Otorisasi: owner atau user dengan permission student:update:any
	authUser, ok := currentUser(c)
	if !ok {
		return helper.ErrUnauthorized("Token autentikasi diperlukan")
	}
	if !CanAccessStudent(authUser, student.OwnerID, s.perms, "student:update:any") {
		return helper.ErrForbidden("Anda tidak memiliki izin untuk mengubah portofolio ini")
	}

	found, err := s.portfolioRepo.DeleteProject(ctx, studentID, projectID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return helper.ErrNotFoundMsg("Portofolio mahasiswa tidak ditemukan")
		}
		if errors.Is(err, repository.ErrInvalidProjectID) {
			return helper.ErrBadRequest("Format project ID tidak valid")
		}
		return helper.ErrInternal("Gagal menghapus proyek", err)
	}

	if !found {
		return helper.ErrNotFoundMsg("Proyek tidak ditemukan dalam portofolio")
	}

	return helper.NoContent(c)
}

// TechStats menangani GET /portfolios/stats/technologies
// Mengembalikan 10 teknologi teratas menggunakan aggregation pipeline MongoDB.
// Tidak memerlukan otorisasi khusus — data agregat, bukan data personal.
func (s *PortfolioService) TechStats(c *fiber.Ctx) error {
	ctx, cancel := helper.ReqCtx(c)
	defer cancel()

	stats, err := s.portfolioRepo.TechStats(ctx)
	if err != nil {
		return helper.ErrInternal("Gagal mengambil statistik teknologi", err)
	}

	return helper.Success(c, fiber.StatusOK, "Statistik teknologi berhasil diambil", stats)
}

// ===== Helper functions =====

// parseStudentID mem-parse parameter :id dari URL dan memvalidasinya.
func parseStudentID(c *fiber.Ctx) (int, error) {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id <= 0 {
		return 0, helper.ErrBadRequest("ID mahasiswa harus berupa angka integer positif")
	}
	return id, nil
}

// translatePortfolioStudentError mengonversi error student repository ke AppError.
func translatePortfolioStudentError(err error) error {
	if errors.Is(err, repository.ErrNotFound) {
		return helper.ErrNotFoundMsg("Mahasiswa tidak ditemukan")
	}
	return helper.ErrInternal("Gagal memverifikasi data mahasiswa", err)
}
