package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"api-students-db/app/model"
)

type AchievementRepository interface {
	FindByStudentNIM(ctx context.Context, nim string) ([]model.Achievement, error)
}

type achievementPostgresRepository struct {
	pool *pgxpool.Pool
}

func NewAchievementRepository(pool *pgxpool.Pool) AchievementRepository {
	return &achievementPostgresRepository{pool: pool}
}

func (r *achievementPostgresRepository) FindByStudentNIM(
	ctx context.Context, nim string,
) ([]model.Achievement, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id_achievement, nim_students, nama_prestasi, juara
		 FROM achievements
		 WHERE nim_students = $1
		 ORDER BY id_achievement ASC`,
		nim,
	)
	if err != nil {
		return nil, fmt.Errorf("mengambil achievements: %w", err)
	}
	defer rows.Close()

	hasil := []model.Achievement{}
	for rows.Next() {
		var a model.Achievement
		if err := rows.Scan(&a.IDAchievement, &a.NIMStudents, &a.NamaPrestasi, &a.Juara); err != nil {
			return nil, fmt.Errorf("membaca baris achievement: %w", err)
		}
		hasil = append(hasil, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("membaca hasil query achievements: %w", err)
	}
	return hasil, nil
}