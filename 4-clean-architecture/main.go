package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"api-students-db/app/repository" 
	"api-students-db/app/service"
	"api-students-db/config"
	"api-students-db/database"
	"github.com/joho/godotenv"
)

func main() {
	// 1. Konfigurasi dan logger
	_ = godotenv.Load() // Memuat file .env
	logger := config.NewLogger()

	// 2. Database
	pool, err := database.NewPool(context.Background())
	if err != nil {
		logger.Error("gagal terhubung ke database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer pool.Close()

	// 3. Perakitan dari dalam ke luar: repository -> service
	studentRepo := repository.NewStudentRepository(pool)
	achievementRepo := repository.NewAchievementRepository(pool) 
	studentService := service.NewStudentService(studentRepo)
	achievementService := service.NewAchievementService(studentRepo, achievementRepo) 

	// 4. Aplikasi Fiber
	app := config.NewApp(logger, pool, studentService, achievementService)

	// Membaca port dari .env, jika tidak ada gunakan default "3000"
	port := config.GetEnv("APP_PORT", "3000")

	// 5. Jalankan server di dalam goroutine
	go func() {
		if err := app.Listen(":" + port); err != nil {
			logger.Error("server berhenti", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()
	logger.Info("server berjalan", slog.String("port", port))

	// 6. Graceful shutdown: tunggu sinyal interupsi (Ctrl+C)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("sinyal berhenti diterima, menutup server...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := app.ShutdownWithContext(ctx); err != nil {
		logger.Error("gagal menutup server dengan rapi", slog.String("error", err.Error()))
	}
	logger.Info("server berhenti dengan rapi")
}