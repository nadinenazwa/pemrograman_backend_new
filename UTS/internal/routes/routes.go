package routes

import (
	"github.com/gofiber/fiber/v2"

	"siakad/internal/handlers"
	"siakad/internal/middleware"
)

func SetupRoutes(app *fiber.App,
	authHandler *handlers.AuthHandler,
	studentHandler *handlers.StudentHandler,
	courseHandler *handlers.CourseHandler,
	enrollmentHandler *handlers.EnrollmentHandler) {

	v1 := app.Group("/api/v1")

	// Auth
	auth := v1.Group("/auth")
	auth.Post("/login", middleware.LoginRateLimiter(), authHandler.Login)
	auth.Get("/me", middleware.Protected(), authHandler.Me)

	// Students
	students := v1.Group("/students", middleware.Protected())
	students.Get("/", middleware.AdminOnly(), studentHandler.List)
	students.Post("/", middleware.AdminOnly(), studentHandler.Create)
	students.Get("/:id", studentHandler.Get) // Role checked inside handler
	students.Put("/:id", middleware.AdminOnly(), studentHandler.Update)
	students.Delete("/:id", middleware.AdminOnly(), studentHandler.Delete)

	// Courses
	courses := v1.Group("/courses", middleware.Protected())
	courses.Get("/", courseHandler.List)

	// Enrollments
	enrollments := v1.Group("/enrollments", middleware.Protected(), middleware.MahasiswaOnly())
	enrollments.Post("/", enrollmentHandler.Create)
	enrollments.Delete("/:id", enrollmentHandler.Delete)
}
