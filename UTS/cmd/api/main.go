package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"

	"siakad/config"
	"siakad/internal/handlers"
	"siakad/internal/repositories"
	"siakad/internal/routes"
	"siakad/internal/utils"
)

func main() {
	config.LoadEnv()

	ctx := context.Background()
	pool, err := config.ConnectDB(ctx)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	// Init repositories
	authRepo := repositories.NewAuthRepository(pool)
	studentRepo := repositories.NewStudentRepository(pool)
	courseRepo := repositories.NewCourseRepository(pool)
	enrollmentRepo := repositories.NewEnrollmentRepository(pool)

	// Init handlers
	authHandler := handlers.NewAuthHandler(authRepo)
	studentHandler := handlers.NewStudentHandler(studentRepo)
	courseHandler := handlers.NewCourseHandler(courseRepo)
	enrollmentHandler := handlers.NewEnrollmentHandler(enrollmentRepo, studentRepo)

	app := fiber.New(fiber.Config{
		ErrorHandler: utils.ErrorHandler,
	})

	routes.SetupRoutes(app, authHandler, studentHandler, courseHandler, enrollmentHandler)

	port := config.GetEnv("APP_PORT", "3000")

	// Graceful shutdown
	go func() {
		if err := app.Listen(":" + port); err != nil {
			log.Fatalf("Error starting server: %v", err)
		}
	}()

	fmt.Println("Server is running on port", port)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("Gracefully shutting down...")
	if err := app.ShutdownWithTimeout(5 * time.Second); err != nil {
		log.Fatalf("Error shutting down server: %v", err)
	}
}
