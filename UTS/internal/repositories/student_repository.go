package repositories

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"siakad/internal/models"
)

type StudentRepository struct {
	pool *pgxpool.Pool
}

func NewStudentRepository(pool *pgxpool.Pool) *StudentRepository {
	return &StudentRepository{pool: pool}
}

func (r *StudentRepository) List(ctx context.Context, page, perPage int, prodi, angkatan, search, sort string) ([]models.Student, int, error) {
	queryCount := `SELECT count(*) FROM students WHERE deleted_at IS NULL`
	queryData := `SELECT id, user_id, nim, nama, prodi, angkatan, ipk_terakhir FROM students WHERE deleted_at IS NULL`

	conditions := []string{}
	args := []interface{}{}
	argId := 1

	if prodi != "" {
		conditions = append(conditions, fmt.Sprintf("prodi = $%d", argId))
		args = append(args, prodi)
		argId++
	}
	if angkatan != "" {
		conditions = append(conditions, fmt.Sprintf("angkatan = $%d", argId))
		args = append(args, angkatan)
		argId++
	}
	if search != "" {
		conditions = append(conditions, fmt.Sprintf("(nim ILIKE $%d OR nama ILIKE $%d)", argId, argId))
		args = append(args, "%"+search+"%")
		argId++
	}

	if len(conditions) > 0 {
		whereClause := " AND " + strings.Join(conditions, " AND ")
		queryCount += whereClause
		queryData += whereClause
	}

	// Sort
	if sort == "-ipk_terakhir" {
		queryData += " ORDER BY ipk_terakhir DESC"
	} else if sort == "nama" {
		queryData += " ORDER BY nama ASC"
	} else {
		queryData += " ORDER BY id ASC" // default
	}

	var total int
	err := r.pool.QueryRow(ctx, queryCount, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * perPage
	queryData += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argId, argId+1)
	args = append(args, perPage, offset)

	rows, err := r.pool.Query(ctx, queryData, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var students []models.Student
	for rows.Next() {
		var s models.Student
		err := rows.Scan(&s.ID, &s.UserID, &s.NIM, &s.Nama, &s.Prodi, &s.Angkatan, &s.IPKTerakhir)
		if err != nil {
			return nil, 0, err
		}
		students = append(students, s)
	}
	if students == nil {
		students = []models.Student{}
	}

	return students, total, nil
}

func (r *StudentRepository) CreateWithUser(ctx context.Context, s *models.Student, userEmail, userPasswordHash string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Create user
	var userID int
	err = tx.QueryRow(ctx, `INSERT INTO users (email, password, role) VALUES ($1, $2, 'mahasiswa') RETURNING id`, userEmail, userPasswordHash).Scan(&userID)
	if err != nil {
		return err
	}

	// Create student
	err = tx.QueryRow(ctx, `
		INSERT INTO students (user_id, nim, nama, prodi, angkatan, ipk_terakhir) 
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING id
	`, userID, s.NIM, s.Nama, s.Prodi, s.Angkatan, s.IPKTerakhir).Scan(&s.ID)
	if err != nil {
		return err
	}
	s.UserID = userID

	return tx.Commit(ctx)
}

func (r *StudentRepository) FindByID(ctx context.Context, id int) (*models.Student, error) {
	query := `SELECT id, user_id, nim, nama, prodi, angkatan, ipk_terakhir FROM students WHERE id = $1 AND deleted_at IS NULL`
	row := r.pool.QueryRow(ctx, query, id)

	var s models.Student
	err := row.Scan(&s.ID, &s.UserID, &s.NIM, &s.Nama, &s.Prodi, &s.Angkatan, &s.IPKTerakhir)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}

func (r *StudentRepository) FindByUserID(ctx context.Context, userID int) (*models.Student, error) {
	query := `SELECT id, user_id, nim, nama, prodi, angkatan, ipk_terakhir FROM students WHERE user_id = $1 AND deleted_at IS NULL`
	row := r.pool.QueryRow(ctx, query, userID)

	var s models.Student
	err := row.Scan(&s.ID, &s.UserID, &s.NIM, &s.Nama, &s.Prodi, &s.Angkatan, &s.IPKTerakhir)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}

func (r *StudentRepository) Update(ctx context.Context, id int, s *models.Student) error {
	query := `UPDATE students SET nama = $1, prodi = $2, angkatan = $3, ipk_terakhir = $4 WHERE id = $5 AND deleted_at IS NULL`
	cmdTag, err := r.pool.Exec(ctx, query, s.Nama, s.Prodi, s.Angkatan, s.IPKTerakhir, id)
	if err != nil {
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *StudentRepository) SoftDelete(ctx context.Context, id int) error {
	query := `UPDATE students SET deleted_at = $1 WHERE id = $2 AND deleted_at IS NULL`
	cmdTag, err := r.pool.Exec(ctx, query, time.Now(), id)
	if err != nil {
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *StudentRepository) GetStudentCourses(ctx context.Context, studentID int) ([]models.Course, int, error) {
	query := `
		SELECT c.id, c.kode_mk, c.nama_mk, c.sks, c.semester, c.kuota
		FROM courses c
		JOIN enrollments e ON c.id = e.course_id
		WHERE e.student_id = $1
	`
	rows, err := r.pool.Query(ctx, query, studentID)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var courses []models.Course
	totalSKS := 0
	for rows.Next() {
		var c models.Course
		if err := rows.Scan(&c.ID, &c.KodeMK, &c.NamaMK, &c.SKS, &c.Semester, &c.Kuota); err != nil {
			return nil, 0, err
		}
		courses = append(courses, c)
		totalSKS += c.SKS
	}
	if courses == nil {
		courses = []models.Course{}
	}

	return courses, totalSKS, nil
}

func GetBatasSKS(ipk float64) int {
	if ipk >= 3.00 {
		return 24
	}
	if ipk >= 2.50 {
		return 21
	}
	return 18
}
