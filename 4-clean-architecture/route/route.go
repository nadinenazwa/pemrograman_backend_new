package route

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"api-students-db/app/model"
	"api-students-db/app/service"
	"api-students-db/middleware"
)

func Register(app *fiber.App, pool *pgxpool.Pool, studentService *service.StudentService) {
	v1 := app.Group("/api/v1")

	v1.Get("/health", healthCheck(pool))

	students := v1.Group("/students", middleware.RequireJSON)
	students.Get("/", studentService.List)
	students.Get("/:id", studentService.Get)
	students.Post("/", studentService.Create)
	students.Put("/:id", studentService.Replace)
	students.Patch("/:id", studentService.Patch)
	students.Delete("/:id", studentService.Delete)
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