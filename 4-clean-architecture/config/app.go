package config

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"api-students-db/app/service"
	"api-students-db/middleware"
	"api-students-db/route"
)

func NewApp(logger *slog.Logger, pool *pgxpool.Pool, studentService *service.StudentService) *fiber.App {
	app := fiber.New()

	middleware.Register(app, logger)
	route.Register(app, pool, studentService)

	return app
}