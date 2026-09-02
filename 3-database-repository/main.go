package main

import (
	"context"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"

	"api-students-db/app/repository"
	"api-students-db/config"
	"api-students-db/database"
)

func main() {
	config.LoadEnv()

	pool, err := database.NewPool(context.Background())
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer pool.Close()

	studentRepo = repository.NewStudentRepository(pool)

	app := fiber.New()

	v1 := app.Group("/api/v1")

	v1.Get("/health", func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(c.UserContext(), 2*time.Second)
		defer cancel()
		if err := pool.Ping(ctx); err != nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"success": false, "message": "database tidak dapat dihubungi",
			})
		}
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"success": true, "message": "server dan database berjalan",
		})
	})

	v1.Get("/students", GetStudents)
	v1.Get("/students/:id", GetStudentByID)
	v1.Post("/students", CreateStudent)
	v1.Put("/students/:id", ReplaceStudent)
	v1.Patch("/students/:id", UpdateStudentPartial)
	v1.Delete("/students/:id", DeleteStudent)

	log.Println("Server berjalan di http://localhost:3000")
	log.Fatal(app.Listen(":3000"))
}