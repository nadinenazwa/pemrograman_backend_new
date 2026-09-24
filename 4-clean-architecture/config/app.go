package config

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"api-students-db/app/service"
	"api-students-db/helper"
	"api-students-db/middleware"
	"api-students-db/route"
)

func NewApp(
	logger *slog.Logger, pool *pgxpool.Pool,
	studentService *service.StudentService,
	achievementService *service.AchievementService,
	authService *service.AuthService,
	jwtManager *helper.JWTManager,
	perms *helper.PermissionSet,
) *fiber.App {
	app := fiber.New(fiber.Config{
		BodyLimit: 1 * 1024 * 1024, // 1 MB
	})

	allowedOrigins := GetEnv("ALLOWED_ORIGINS", "*")
	middleware.Register(app, logger, allowedOrigins)
	route.Register(app, pool, studentService, achievementService, authService, jwtManager, perms)

	return app
}