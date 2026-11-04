package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"api-students-db/app/repository"
	"api-students-db/app/service"
	"api-students-db/config"
	"api-students-db/database"
	"api-students-db/helper"
)

func main() {
	_ = godotenv.Load()
	logger := config.NewLogger()

	role := flag.String("role", config.GetEnv("APP_ROLE", "all"), "api | worker | all")
	flag.Parse()

	runAPI := *role == "api" || *role == "all"
	runWorker := *role == "worker" || *role == "all"
	if !runAPI && !runWorker {
		logger.Error("role tidak dikenal", slog.String("role", *role))
		os.Exit(1)
	}

	pool, err := database.NewPool(context.Background())
	if err != nil {
		logger.Error("gagal terhubung ke database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer pool.Close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var app *fiber.App
	var inspector *asynq.Inspector

	if runAPI {
		app, inspector = startAPI(logger, pool)
		if app == nil {
			os.Exit(1)
		}
	}

	var stopWorker func()
	if runWorker {
		stopWorker = startWorker(ctx, logger, pool)
		if stopWorker == nil {
			os.Exit(1)
		}
	}

	logger.Info("aplikasi berjalan", slog.String("role", *role))
	<-ctx.Done()

	if app != nil {
		ctxShut, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := app.ShutdownWithContext(ctxShut); err != nil {
			logger.Error("shutdown HTTP tidak rapi", slog.String("error", err.Error()))
		}
		if inspector != nil {
			_ = inspector.Close()
		}
	}

	if stopWorker != nil {
		stopWorker()
	}

	logger.Info("aplikasi berhenti dengan rapi")
}

func startAPI(logger *slog.Logger, pool *pgxpool.Pool) (*fiber.App, *asynq.Inspector) {
	jwtManager, err := helper.NewJWTManager()
	if err != nil {
		logger.Error("gagal menginisialisasi JWT", slog.String("error", err.Error()))
		return nil, nil
	}

	roleRepo := repository.NewRoleRepository(pool)
	rolePerms, err := roleRepo.LoadPermissions(context.Background())
	if err != nil {
		logger.Error("gagal memuat permission RBAC dari database", slog.String("error", err.Error()))
		return nil, nil
	}
	perms := helper.NewPermissionSet(rolePerms)

	mongoClient, err := database.NewMongoClient(context.Background())
	if err != nil {
		logger.Error("gagal terhubung ke MongoDB", slog.String("error", err.Error()))
		return nil, nil
	}
	mongoDB := database.GetMongoDB(mongoClient)

	studentRepo := repository.NewStudentRepository(pool)
	achievementRepo := repository.NewAchievementRepository(pool)
	userRepo := repository.NewUserRepository(pool)
	refreshTokenRepo := repository.NewRefreshTokenRepository(pool)
	courseRepo := repository.NewCourseRepository(pool)
	outboxRepo := repository.NewOutboxRepository(pool)
	notificationRepo := repository.NewNotificationRepository(pool)
	txManager := repository.NewTxManager(pool)

	portfolioRepo, err := repository.NewPortfolioRepository(context.Background(), mongoDB)
	if err != nil {
		logger.Error("gagal menginisialisasi portfolio repository", slog.String("error", err.Error()))
		return nil, nil
	}

	inspector := asynq.NewInspector(config.RedisOpt())

	studentService := service.NewStudentService(studentRepo, perms)
	achievementService := service.NewAchievementService(studentRepo, achievementRepo)
	authService := service.NewAuthService(userRepo, refreshTokenRepo, jwtManager, perms)
	portfolioService := service.NewPortfolioService(portfolioRepo, studentRepo, perms)
	courseService := service.NewCourseService(courseRepo, outboxRepo, txManager)
	notificationService := service.NewNotificationService(notificationRepo, outboxRepo, inspector)

	app := config.NewApp(
		logger, pool,
		studentService, achievementService, authService, portfolioService,
		courseService, notificationService,
		jwtManager, perms,
	)

	port := config.GetEnv("APP_PORT", "3000")
	go func() {
		if err := app.Listen(":" + port); err != nil {
			logger.Error("server berhenti", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()
	logger.Info("HTTP berjalan", slog.String("port", port))

	return app, inspector
}
