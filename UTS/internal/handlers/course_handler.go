package handlers

import (
	"github.com/gofiber/fiber/v2"

	"siakad/internal/repositories"
	"siakad/internal/utils"
)

type CourseHandler struct {
	repo *repositories.CourseRepository
}

func NewCourseHandler(repo *repositories.CourseRepository) *CourseHandler {
	return &CourseHandler{repo: repo}
}

func (h *CourseHandler) List(c *fiber.Ctx) error {
	semester := c.Query("semester", "")
	search := c.Query("search", "")
	available := c.Query("available", "") == "true"

	courses, err := h.repo.List(c.Context(), semester, search, available)
	if err != nil {
		return utils.ErrorHandler(c, err)
	}

	return utils.SendSuccess(c, fiber.StatusOK, "Data mata kuliah berhasil diambil", courses)
}
