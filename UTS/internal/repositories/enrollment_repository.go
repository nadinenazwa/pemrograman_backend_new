package repositories

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"siakad/internal/models"
)

type EnrollmentRepository struct {
	pool *pgxpool.Pool
}

func NewEnrollmentRepository(pool *pgxpool.Pool) *EnrollmentRepository {
	return &EnrollmentRepository{pool: pool}
}

// Create uses transaction and row locking to ensure consistency
func (r *EnrollmentRepository) Create(ctx context.Context, studentID int, req *models.CreateEnrollmentRequest, batasSKS int) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// 1. Cek course and lock course row for update? No, row locking is better on course itself if we had terisi column,
	// but terisi is calculated dynamically from enrollments.
	// To prevent race conditions safely, we should lock the course row anyway.
	var course models.Course
	err = tx.QueryRow(ctx, `SELECT id, sks, kuota FROM courses WHERE id = $1 FOR UPDATE`, req.CourseID).Scan(&course.ID, &course.SKS, &course.Kuota)
	if err != nil {
		if err == pgx.ErrNoRows {
			return errors.New("course_not_found")
		}
		return err
	}

	// 2. Check duplicate enrollment
	var exists bool
	err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM enrollments WHERE student_id = $1 AND course_id = $2 AND tahun_akademik = $3)`,
		studentID, course.ID, req.TahunAkademik).Scan(&exists)
	if err != nil {
		return err
	}
	if exists {
		return errors.New("duplicate_enrollment")
	}

	// 3. Cek kuota
	var terisi int
	err = tx.QueryRow(ctx, `SELECT COUNT(*) FROM enrollments WHERE course_id = $1`, course.ID).Scan(&terisi)
	if err != nil {
		return err
	}
	if terisi >= course.Kuota {
		return errors.New("kuota_penuh")
	}

	// 4. Hitung total SKS mahasiswa
	var totalSKS int
	err = tx.QueryRow(ctx, `
		SELECT COALESCE(SUM(c.sks), 0) 
		FROM enrollments e 
		JOIN courses c ON e.course_id = c.id 
		WHERE e.student_id = $1
	`, studentID).Scan(&totalSKS)
	if err != nil {
		return err
	}

	if totalSKS+course.SKS > batasSKS {
		return errors.New(fmt.Sprintf("sks_limit:%d", batasSKS-totalSKS))
	}

	// 5. Insert enrollment
	_, err = tx.Exec(ctx, `
		INSERT INTO enrollments (student_id, course_id, tahun_akademik, created_at)
		VALUES ($1, $2, $3, NOW())
	`, studentID, course.ID, req.TahunAkademik)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *EnrollmentRepository) Delete(ctx context.Context, enrollmentID int, studentID int) error {
	cmdTag, err := r.pool.Exec(ctx, `DELETE FROM enrollments WHERE id = $1 AND student_id = $2`, enrollmentID, studentID)
	if err != nil {
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}
