package config

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"

	"api-students-db/helper"
	"api-students-db/middleware"
	"api-students-db/route"
)

func NewApp(logger *slog.Logger, deps route.Dependencies) *fiber.App {
	app := fiber.New(fiber.Config{
		BodyLimit:    1 * 1024 * 1024, // 1 MB
		ErrorHandler: helper.ErrorHandler,
	})

	allowedOrigins := GetEnv("ALLOWED_ORIGINS", "*")
	middleware.Register(app, logger, allowedOrigins)
	route.Register(app, deps)

	return app
}
