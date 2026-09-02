package main

import (
	"log"

	"github.com/gofiber/fiber/v2"
)

func main() {
	app := fiber.New()

	v1 := app.Group("/api/v1")
	v1.Get("/students", GetStudents)
	v1.Get("/students/:id", GetStudentByID)
	v1.Post("/students", CreateStudent)
	v1.Put("/students/:id", ReplaceStudent)
	v1.Patch("/students/:id", UpdateStudentPartial)
	v1.Delete("/students/:id", DeleteStudent)

	log.Println("Server berjalan di http://localhost:3000")
	log.Fatal(app.Listen(":3000"))
}
