package repositories

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"siakad/internal/models"
)

type CourseRepository struct {
	pool *pgxpool.Pool
}

func NewCourseRepository(pool *pgxpool.Pool) *CourseRepository {
	return &CourseRepository{pool: pool}
}

func (r *CourseRepository) List(ctx context.Context, semester, search string, availableOnly bool) ([]models.CourseWithQuota, error) {
	query := `
		SELECT c.id, c.kode_mk, c.nama_mk, c.sks, c.semester, c.kuota,
		       COALESCE(e.terisi, 0) as terisi
		FROM courses c
		LEFT JOIN (
			SELECT course_id, COUNT(*) as terisi FROM enrollments GROUP BY course_id
		) e ON c.id = e.course_id
		WHERE 1=1
	`
	args := []interface{}{}
	argId := 1
	conditions := []string{}

	if semester != "" {
		conditions = append(conditions, fmt.Sprintf("c.semester = $%d", argId))
		args = append(args, semester)
		argId++
	}
	if search != "" {
		conditions = append(conditions, fmt.Sprintf("(c.kode_mk ILIKE $%d OR c.nama_mk ILIKE $%d)", argId, argId))
		args = append(args, "%"+search+"%")
		argId++
	}

	if len(conditions) > 0 {
		query += " AND " + strings.Join(conditions, " AND ")
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var courses []models.CourseWithQuota
	for rows.Next() {
		var c models.CourseWithQuota
		err := rows.Scan(&c.ID, &c.KodeMK, &c.NamaMK, &c.SKS, &c.Semester, &c.Kuota, &c.Terisi)
		if err != nil {
			return nil, err
		}
		c.SisaKuota = c.Kuota - c.Terisi

		if availableOnly && c.SisaKuota <= 0 {
			continue
		}
		courses = append(courses, c)
	}

	if courses == nil {
		courses = []models.CourseWithQuota{}
	}

	return courses, nil
}
