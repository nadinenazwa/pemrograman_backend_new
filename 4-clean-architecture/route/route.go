package route

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/swagger"
	"github.com/jackc/pgx/v5/pgxpool"

	"api-students-db/app/service"
	"api-students-db/helper"
	"api-students-db/middleware"
)

type Dependencies struct {
	Pool                *pgxpool.Pool
	StudentService      *service.StudentService
	AchievementService  *service.AchievementService
	AuthService         *service.AuthService
	PortfolioService    *service.PortfolioService
	CourseService       *service.CourseService
	NotificationService *service.NotificationService
	ReviewService       *service.ReviewService
	UserService         *service.UserService
	JWT                 *helper.JWTManager
	Permissions         *helper.PermissionSet
	EnableSwagger       bool
}

func Register(app *fiber.App, deps Dependencies) {
	if deps.EnableSwagger {
		// Swagger UI: http://localhost:3000/swagger/index.html
		app.Get("/swagger/*", swagger.HandlerDefault)
	}

	v1 := app.Group("/api/v1")

	if deps.Pool != nil {
		v1.Get("/health", healthCheck(deps.Pool))
	} else {
		v1.Get("/health", func(c *fiber.Ctx) error {
			return helper.Success(c, fiber.StatusOK, "server berjalan", fiber.Map{"status": "up"})
		})
	}

	// === Endpoint publik: Auth ===
	if deps.AuthService != nil {
		auth := v1.Group("/auth", middleware.RequireJSON)
		auth.Post("/register", deps.AuthService.Register)
		auth.Post("/login", middleware.LoginRateLimiter(), deps.AuthService.Login)
		auth.Post("/refresh", deps.AuthService.Refresh)
		auth.Post("/logout", deps.AuthService.Logout)
	}

	// === Endpoint yang membutuhkan autentikasi ===
	var authRequired fiber.Handler
	if deps.JWT != nil {
		authRequired = middleware.RequireAuth(deps.JWT)
	} else {
		authRequired = func(c *fiber.Ctx) error { return c.Next() }
	}

	if deps.AuthService != nil {
		v1.Get("/auth/me", authRequired, deps.AuthService.Me)
	}

	// === Students ===
	if deps.StudentService != nil {
		students := v1.Group("/students", authRequired)
		if deps.Permissions != nil {
			students.Get("/", middleware.RequirePermission(deps.Permissions, "student:list"), deps.StudentService.List)
			students.Post("/", middleware.RequireJSON, middleware.RequirePermission(deps.Permissions, "student:create"), deps.StudentService.Create)
			students.Delete("/:id", middleware.RequirePermission(deps.Permissions, "student:delete"), deps.StudentService.Delete)
		} else {
			students.Get("/", deps.StudentService.List)
			students.Post("/", middleware.RequireJSON, deps.StudentService.Create)
			students.Delete("/:id", deps.StudentService.Delete)
		}
		students.Get("/:id", deps.StudentService.Get)
		students.Put("/:id", middleware.RequireJSON, deps.StudentService.Replace)
		students.Patch("/:id", middleware.RequireJSON, deps.StudentService.Patch)
	}

	// === Achievements ===
	if deps.AchievementService != nil {
		v1.Get("/students/:nim/achievements", deps.AchievementService.ListByStudentNIM)
	}

	// === Portfolio endpoints (MongoDB) ===
	if deps.PortfolioService != nil {
		students := v1.Group("/students", authRequired)
		students.Get("/:id/portfolio", deps.PortfolioService.GetPortfolio)
		students.Put("/:id/portfolio", middleware.RequireJSON, deps.PortfolioService.UpsertPortfolio)
		students.Post("/:id/portfolio/projects", middleware.RequireJSON, deps.PortfolioService.AddProject)
		students.Delete("/:id/portfolio/projects/:projectId", deps.PortfolioService.DeleteProject)

		v1.Get("/portfolios/stats/technologies", authRequired, deps.PortfolioService.TechStats)
	}

	// === Courses & KRS (Modul 9, 11, 12) ===
	if deps.CourseService != nil {
		courses := v1.Group("/courses", middleware.RequireJSON, authRequired)
		courses.Get("/", deps.CourseService.List)
		courses.Get("/mine", deps.CourseService.Mine)
		courses.Get("/:id", deps.CourseService.Get)
		courses.Post("/:id/enroll", deps.CourseService.Enroll)
		courses.Delete("/:id/enroll", deps.CourseService.Drop)
		if deps.Permissions != nil {
			courses.Patch("/:id", middleware.RequirePermission(deps.Permissions, "course:manage"), deps.CourseService.UpdateQuota)
		} else {
			courses.Patch("/:id", deps.CourseService.UpdateQuota)
		}

		enrollments := v1.Group("/enrollments", middleware.RequireJSON, authRequired)
		enrollments.Post("/swap", deps.CourseService.Swap)
	}

	// === Reviews (Modul 12) ===
	if deps.ReviewService != nil {
		courseReviews := v1.Group("/courses/:id/reviews", authRequired)
		courseReviews.Get("/", deps.ReviewService.ListByCourse)
		courseReviews.Post("/", middleware.RequireJSON, deps.ReviewService.Create)
		v1.Get("/courses/:id/reviews/summary", authRequired, deps.ReviewService.Summary)

		reviews := v1.Group("/reviews", authRequired)
		reviews.Get("/:id", deps.ReviewService.Get)
		reviews.Patch("/:id", middleware.RequireJSON, deps.ReviewService.Update)
		reviews.Delete("/:id", deps.ReviewService.Delete)
		reviews.Post("/:id/replies", middleware.RequireJSON, deps.ReviewService.AddReply)
		reviews.Post("/:id/helpful", deps.ReviewService.VoteHelpful)
	}

	// === Notifications & Monitoring (Modul 11) ===
	if deps.NotificationService != nil {
		notifications := v1.Group("/notifications", middleware.RequireJSON, authRequired)
		notifications.Get("/mine", deps.NotificationService.Mine)

		admin := v1.Group("/admin", middleware.RequireJSON, authRequired)
		if deps.Permissions != nil {
			admin.Get("/queue", middleware.RequirePermission(deps.Permissions, "queue:monitor"), deps.NotificationService.Monitor)
		} else {
			admin.Get("/queue", deps.NotificationService.Monitor)
		}
	}

	// === Users (Modul 12 D.3) ===
	if deps.UserService != nil {
		users := v1.Group("/users", authRequired)
		users.Get("/", deps.UserService.List)
		users.Post("/", middleware.RequireJSON, deps.UserService.Create)
		users.Get("/:id", deps.UserService.Get)
		users.Put("/:id", middleware.RequireJSON, deps.UserService.Update)
		users.Patch("/:id", middleware.RequireJSON, deps.UserService.Patch)
		users.Delete("/:id", deps.UserService.Delete)
	}
}

func healthCheck(pool *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		if pool == nil {
			return helper.Success(c, fiber.StatusOK, "server berjalan (tanpa database pool)", fiber.Map{
				"postgres": "disabled",
			})
		}

		if err := pool.Ping(ctx); err != nil {
			return helper.Internal(err)
		}

		return helper.Success(c, fiber.StatusOK, "server, PostgreSQL, dan MongoDB berjalan", fiber.Map{
			"postgres": "up",
			"mongo":    "up",
		})
	}
}
