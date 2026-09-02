package main

import (
	"strings"
	"github.com/gofiber/fiber/v2"
)

// Cek NIM Duplikat (untuk HTTP 409)
func isNIMExists(nim string, excludeID int) bool {
	for _, s := range studentList {
		if s.NIM == nim && s.ID != excludeID {
			return true
		}
	}
	return false
}

// Cek Header Content-Type (untuk HTTP 415)
func checkContentTypeJSON(c *fiber.Ctx) error {
	contentType := c.Get("Content-Type")
	if !strings.HasPrefix(contentType, "application/json") {
		return c.Status(fiber.StatusUnsupportedMediaType).JSON(APIResponse{
			Success: false,
			Message: "Content-Type harus application/json",
		})
	}
	return nil
}