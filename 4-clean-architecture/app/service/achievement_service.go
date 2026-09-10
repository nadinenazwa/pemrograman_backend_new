package service

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"api-students-db/app/repository"
	"api-students-db/helper"
)

type AchievementService struct {
	studentRepo     repository.StudentRepository
	achievementRepo repository.AchievementRepository
}

func NewAchievementService(
	studentRepo repository.StudentRepository,
	achievementRepo repository.AchievementRepository,
) *AchievementService {
	return &AchievementService{studentRepo: studentRepo, achievementRepo: achievementRepo}
}

// GET /api/v1/students/:nim/achievements
func (s *AchievementService) ListByStudentNIM(c *fiber.Ctx) error {
	ctx, cancel := helper.ReqCtx(c)
	defer cancel()

	nim := c.Params("nim")

	student, err := s.studentRepo.FindByNIM(ctx, nim)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return helper.Fail(c, fiber.StatusNotFound, "Mahasiswa dengan NIM tersebut tidak ditemukan")
		}
		return helper.Fail(c, fiber.StatusInternalServerError, "Gagal mengambil data mahasiswa")
	}

	achievements, err := s.achievementRepo.FindByStudentNIM(ctx, nim)
	if err != nil {
		return helper.Fail(c, fiber.StatusInternalServerError, "Gagal mengambil daftar prestasi")
	}

	return helper.Success(c, fiber.StatusOK, "Daftar prestasi mahasiswa berhasil diambil", fiber.Map{
		"student":      student,
		"achievements": achievements,
	})
}