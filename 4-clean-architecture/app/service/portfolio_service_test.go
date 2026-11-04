package service_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"api-students-db/app/model"
	"api-students-db/app/repository"
	"api-students-db/app/service"
	"api-students-db/helper"
	"api-students-db/middleware"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// mockStudentRepo mengimplementasikan repository.StudentRepository
type mockStudentRepo struct {
	students map[int]model.Student
}

func newMockStudentRepo() *mockStudentRepo {
	return &mockStudentRepo{
		students: make(map[int]model.Student),
	}
}

func (m *mockStudentRepo) FindByID(ctx context.Context, id int) (model.Student, error) {
	s, ok := m.students[id]
	if !ok {
		return model.Student{}, repository.ErrNotFound
	}
	return s, nil
}

func (m *mockStudentRepo) FindAll(ctx context.Context, q model.ListQuery) ([]model.Student, int, error) {
	return nil, 0, nil
}
func (m *mockStudentRepo) FindAllCursor(ctx context.Context, q model.CursorQuery) ([]model.Student, bool, string, error) {
	return nil, false, "", nil
}
func (m *mockStudentRepo) FindByNIM(ctx context.Context, nim string) (model.Student, error) {
	return model.Student{}, repository.ErrNotFound
}
func (m *mockStudentRepo) Create(ctx context.Context, s model.Student) (model.Student, error) {
	m.students[s.ID] = s
	return s, nil
}
func (m *mockStudentRepo) Update(ctx context.Context, s model.Student) (model.Student, error) {
	m.students[s.ID] = s
	return s, nil
}
func (m *mockStudentRepo) Delete(ctx context.Context, id int) error {
	delete(m.students, id)
	return nil
}

func setupPortfolioServiceTest(t *testing.T) (*fiber.App, *mockStudentRepo, repository.PortfolioRepository, func()) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client, err := mongo.Connect(options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		t.Skipf("MongoDB lokal tidak tersedia: %v", err)
	}
	if err := client.Ping(ctx, nil); err != nil {
		t.Skipf("MongoDB ping gagal: %v", err)
	}

	dbName := fmt.Sprintf("test_portfolio_service_%d", time.Now().UnixNano())
	db := client.Database(dbName)

	portfolioRepo, err := repository.NewPortfolioRepository(ctx, db)
	if err != nil {
		t.Fatalf("gagal inisialisasi portfolio repo: %v", err)
	}

	studentRepo := newMockStudentRepo()

	// Setup RBAC permissions
	rolePerms := map[string][]string{
		"admin": {"student:list", "student:create", "student:read", "student:update", "student:update:any", "student:delete"},
		"user":  {},
	}
	perms := helper.NewPermissionSet(rolePerms)

	svc := service.NewPortfolioService(portfolioRepo, studentRepo, perms)

	app := fiber.New(fiber.Config{
		ErrorHandler: helper.ErrorHandler,
	})

	// Helper middleware untuk simulasi login user
	app.Use(func(c *fiber.Ctx) error {
		userIDHeader := c.Get("X-User-ID")
		userRoleHeader := c.Get("X-User-Role")
		if userIDHeader != "" {
			var uid int
			_, _ = fmt.Sscanf(userIDHeader, "%d", &uid)
			c.Locals("auth_user", model.AuthUser{
				ID:       uid,
				Username: "testuser",
				Role:     userRoleHeader,
			})
		}
		return c.Next()
	})

	// Routes
	app.Get("/api/v1/students/:id/portfolio", svc.GetPortfolio)
	app.Put("/api/v1/students/:id/portfolio", middleware.RequireJSON, svc.UpsertPortfolio)
	app.Post("/api/v1/students/:id/portfolio/projects", middleware.RequireJSON, svc.AddProject)
	app.Delete("/api/v1/students/:id/portfolio/projects/:projectId", svc.DeleteProject)
	app.Get("/api/v1/portfolios/stats/technologies", svc.TechStats)

	cleanup := func() {
		cleanCtx, cleanCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanCancel()
		_ = db.Drop(cleanCtx)
		_ = client.Disconnect(cleanCtx)
	}

	return app, studentRepo, portfolioRepo, cleanup
}

func TestPortfolioService_GetPortfolio_NotFoundStudent(t *testing.T) {
	app, _, _, cleanup := setupPortfolioServiceTest(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/students/999/portfolio", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test gagal: %v", err)
	}

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 for non-existent student, got %d", resp.StatusCode)
	}
}

func TestPortfolioService_GetPortfolio_PortfolioNotAvailable(t *testing.T) {
	app, studentRepo, _, cleanup := setupPortfolioServiceTest(t)
	defer cleanup()

	// Mahasiswa ada di Postgres
	studentRepo.students[1] = model.Student{
		ID:      1,
		OwnerID: 10,
		Name:    "Budi",
		NIM:     "12345678",
	}

	// Tapi portofolio belum dibuat
	req := httptest.NewRequest(http.MethodGet, "/api/v1/students/1/portfolio", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test gagal: %v", err)
	}

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 for uncreated portfolio, got %d", resp.StatusCode)
	}
}

func TestPortfolioService_UpsertPortfolio_ForbiddenNonOwner(t *testing.T) {
	app, studentRepo, _, cleanup := setupPortfolioServiceTest(t)
	defer cleanup()

	// Student milik user ID 10
	studentRepo.students[1] = model.Student{
		ID:      1,
		OwnerID: 10,
		Name:    "Budi",
		NIM:     "12345678",
	}

	// Request dari user ID 99 (role "user", bukan owner)
	body := `{"headline":"Junior Go Dev","bio":"Bio lengkap mahasiswa teknik informatika"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/students/1/portfolio", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", "99")
	req.Header.Set("X-User-Role", "user")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test gagal: %v", err)
	}

	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden untuk non-owner, got %d", resp.StatusCode)
	}
}

func TestPortfolioService_UpsertAndGet_SuccessOwner(t *testing.T) {
	app, studentRepo, _, cleanup := setupPortfolioServiceTest(t)
	defer cleanup()

	studentRepo.students[1] = model.Student{
		ID:      1,
		OwnerID: 10,
		Name:    "Budi",
		NIM:     "12345678",
	}

	// 1. PUT Portfolio oleh owner (User ID 10)
	body := `{"headline":"Senior Backend Engineer","bio":"Spesialis Clean Architecture dan MongoDB Go Driver"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/students/1/portfolio", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", "10")
	req.Header.Set("X-User-Role", "user")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test gagal: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	// 2. GET Portfolio
	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/students/1/portfolio", nil)
	getResp, err := app.Test(getReq)
	if err != nil {
		t.Fatalf("app.Test GET gagal: %v", err)
	}
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK pada GET, got %d", getResp.StatusCode)
	}
}

func TestPortfolioService_AddProject_And_DeleteProject(t *testing.T) {
	app, studentRepo, _, cleanup := setupPortfolioServiceTest(t)
	defer cleanup()

	studentRepo.students[1] = model.Student{
		ID:      1,
		OwnerID: 10,
		Name:    "Budi",
		NIM:     "12345678",
	}

	// 1. Inisialisasi portofolio
	initBody := `{"headline":"Dev","bio":"Deskripsi mahasiswa teknik informatika"}`
	putReq := httptest.NewRequest(http.MethodPut, "/api/v1/students/1/portfolio", bytes.NewBufferString(initBody))
	putReq.Header.Set("Content-Type", "application/json")
	putReq.Header.Set("X-User-ID", "10")
	putReq.Header.Set("X-User-Role", "user")
	_, _ = app.Test(putReq)

	// 2. POST project baru
	projBody := `{"title":"REST API Go","technologies":["Go","PostgreSQL"],"year":2024}`
	postReq := httptest.NewRequest(http.MethodPost, "/api/v1/students/1/portfolio/projects", bytes.NewBufferString(projBody))
	postReq.Header.Set("Content-Type", "application/json")
	postReq.Header.Set("X-User-ID", "10")
	postReq.Header.Set("X-User-Role", "user")

	postResp, err := app.Test(postReq)
	if err != nil {
		t.Fatalf("app.Test POST project gagal: %v", err)
	}
	if postResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d", postResp.StatusCode)
	}

	var postResult struct {
		Data model.StudentPortfolio `json:"data"`
	}
	_ = json.NewDecoder(postResp.Body).Decode(&postResult)
	if len(postResult.Data.Projects) != 1 {
		t.Fatalf("expected 1 project, got %d", len(postResult.Data.Projects))
	}
	projectID := postResult.Data.Projects[0].ID

	// 3. DELETE project yang baru ditambahkan
	delReq := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/students/1/portfolio/projects/%s", projectID), nil)
	delReq.Header.Set("X-User-ID", "10")
	delReq.Header.Set("X-User-Role", "user")

	delResp, err := app.Test(delReq)
	if err != nil {
		t.Fatalf("app.Test DELETE gagal: %v", err)
	}
	if delResp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 No Content, got %d", delResp.StatusCode)
	}

	// 4. DELETE ulang proyek yang sama -> harus 404
	delResp2, err := app.Test(delReq)
	if err != nil {
		t.Fatalf("app.Test DELETE kedua gagal: %v", err)
	}
	if delResp2.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 Not Found untuk proyek yang sudah terhapus, got %d", delResp2.StatusCode)
	}
}

func TestPortfolioService_TechStats_Endpoint(t *testing.T) {
	app, _, _, cleanup := setupPortfolioServiceTest(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/portfolios/stats/technologies", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test stats gagal: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK untuk stats, got %d", resp.StatusCode)
	}
}
