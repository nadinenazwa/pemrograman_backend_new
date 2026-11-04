package route

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"api-students-db/app/model"
	"api-students-db/app/service"
	"api-students-db/helper"
	"api-students-db/middleware"
)

func Register(
	app *fiber.App,
	pool *pgxpool.Pool,
	studentService *service.StudentService,
	achievementService *service.AchievementService,
	authService *service.AuthService,
	portfolioService *service.PortfolioService,
	courseService *service.CourseService,
	notificationService *service.NotificationService,
	jwtManager *helper.JWTManager,
	perms *helper.PermissionSet,
) {
	v1 := app.Group("/api/v1")

	v1.Get("/health", healthCheck(pool))

	// === Endpoint publik: Auth ===
	auth := v1.Group("/auth", middleware.RequireJSON)
	auth.Post("/register", authService.Register)
	auth.Post("/login", middleware.LoginRateLimiter(), authService.Login)
	auth.Post("/refresh", authService.Refresh)
	auth.Post("/logout", authService.Logout)

	// === Endpoint yang membutuhkan autentikasi ===
	authRequired := middleware.RequireAuth(jwtManager)

	// /auth/me
	auth.Get("/me", authRequired, authService.Me)

	// === Seluruh endpoint students membutuhkan login ===
	students := v1.Group("/students", authRequired)

	// GET /students — tanpa RequireJSON karena mendukung content negotiation (JSON/CSV)
	students.Get("/", middleware.RequirePermission(perms, "student:list"), studentService.List)

	// POST/PUT/PATCH/DELETE membutuhkan RequireJSON
	students.Post("/", middleware.RequireJSON, middleware.RequirePermission(perms, "student:create"), studentService.Create)
	students.Delete("/:id", middleware.RequirePermission(perms, "student:delete"), studentService.Delete)

	// Ownership-based: service menangani otorisasi via CanAccessStudent
	students.Get("/:id", studentService.Get)
	students.Put("/:id", middleware.RequireJSON, studentService.Replace)
	students.Patch("/:id", middleware.RequireJSON, studentService.Patch)

	// Achievements (tidak ada perubahan RBAC)
	students.Get("/:nim/achievements", achievementService.ListByStudentNIM)

	// === Portfolio endpoints (MongoDB) ===
	// GET & PUT /students/:id/portfolio
	students.Get("/:id/portfolio", portfolioService.GetPortfolio)
	students.Put("/:id/portfolio", middleware.RequireJSON, portfolioService.UpsertPortfolio)

	// POST /students/:id/portfolio/projects
	students.Post("/:id/portfolio/projects", middleware.RequireJSON, portfolioService.AddProject)

	// DELETE /students/:id/portfolio/projects/:projectId
	students.Delete("/:id/portfolio/projects/:projectId", portfolioService.DeleteProject)

	// === Portfolio statistics (tidak perlu :id) ===
	v1.Get("/portfolios/stats/technologies", authRequired, portfolioService.TechStats)

	// === Courses & KRS (Modul 9 & 11) ===
	if courseService != nil {
		courses := v1.Group("/courses", middleware.RequireJSON, authRequired)
		courses.Get("/", courseService.List)
		courses.Get("/mine", courseService.Mine)
		courses.Get("/:id", courseService.Get)
		courses.Post("/:id/enroll", courseService.Enroll)
		courses.Delete("/:id/enroll", courseService.Drop)
		courses.Patch("/:id", middleware.RequirePermission(perms, "course:manage"), courseService.UpdateQuota)

		enrollments := v1.Group("/enrollments", middleware.RequireJSON, authRequired)
		enrollments.Post("/swap", courseService.Swap)
	}

	// === Notifications & Monitoring (Modul 11) ===
	if notificationService != nil {
		notifications := v1.Group("/notifications", middleware.RequireJSON, authRequired)
		notifications.Get("/mine", notificationService.Mine)

		admin := v1.Group("/admin", middleware.RequireJSON, authRequired)
		admin.Get("/queue", middleware.RequirePermission(perms, "queue:monitor"), notificationService.Monitor)
	}
}

func healthCheck(pool *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(c.UserContext(), 2*time.Second)
		defer cancel()
		if err := pool.Ping(ctx); err != nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(model.APIResponse{
				Success: false, Message: "database tidak dapat dihubungi",
			})
		}
		return c.Status(fiber.StatusOK).JSON(model.APIResponse{
			Success: true, Message: "server dan database berjalan",
		})
	}
}
