package repository

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"api-students-db/app/model"
)

var (
	ErrNotFound  = errors.New("data tidak ditemukan")
	ErrDuplicate = errors.New("data sudah ada")
)

type StudentRepository interface {
	FindAll(ctx context.Context, q model.ListQuery) ([]model.Student, int, error)
	FindAllCursor(ctx context.Context, q model.CursorQuery) ([]model.Student, bool, string, error)
	FindByID(ctx context.Context, id int) (model.Student, error)
	Create(ctx context.Context, s model.Student) (model.Student, error)
	Update(ctx context.Context, s model.Student) (model.Student, error)
	Delete(ctx context.Context, id int) error
	FindByNIM(ctx context.Context, nim string) (model.Student, error)
}

// Daftar putih kolom yang boleh dipakai di ORDER BY.
var kolomUrutStudent = map[string]string{
	"id":    "id",
	"nim":   "nim",
	"name":  "name",
	"grade": "grade",
}

type studentPostgresRepository struct {
	pool *pgxpool.Pool
}

func NewStudentRepository(pool *pgxpool.Pool) StudentRepository {
	return &studentPostgresRepository{pool: pool}
}

func buildStudentFilter(q model.ListQuery) (string, []any) {
	where := " WHERE 1 = 1"
	args := []any{}

	if q.Search != "" {
		where += fmt.Sprintf(" AND name ILIKE $%d", len(args)+1)
		args = append(args, "%"+q.Search+"%")
	}
	if q.IsActive != nil {
		where += fmt.Sprintf(" AND is_active = $%d", len(args)+1)
		args = append(args, *q.IsActive)
	}
	return where, args
}

// FindAll menggunakan offset-based pagination (dipertahankan untuk kompatibilitas).
func (r *studentPostgresRepository) FindAll(
	ctx context.Context, q model.ListQuery,
) ([]model.Student, int, error) {
	where, args := buildStudentFilter(q)

	var total int
	err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM students"+where, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("menghitung student: %w", err)
	}

	arah := "ASC"
	if q.Order == "desc" {
		arah = "DESC"
	}
	kolom, ok := kolomUrutStudent[q.Sort]
	if !ok {
		kolom = "id"
	}

	sqlText := fmt.Sprintf(
		`SELECT id, nim, name, grade, is_active, owner_id, created_at
		 FROM students%s
		 ORDER BY %s %s
		 LIMIT $%d OFFSET $%d`,
		where, kolom, arah, len(args)+1, len(args)+2,
	)
	args = append(args, q.Limit, q.Offset())

	rows, err := r.pool.Query(ctx, sqlText, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("mengambil daftar student: %w", err)
	}
	defer rows.Close()

	hasil := []model.Student{}
	for rows.Next() {
		var s model.Student
		if err := rows.Scan(&s.ID, &s.NIM, &s.Name, &s.Grade, &s.IsActive, &s.OwnerID, &s.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("membaca baris student: %w", err)
		}
		hasil = append(hasil, s)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("membaca hasil query: %w", err)
	}
	return hasil, total, nil
}

// FindAllCursor menggunakan cursor-based (keyset) pagination.
// ORDER BY created_at DESC, id DESC — cursor menggunakan pasangan (created_at, id).
// Menggunakan LIMIT+1 pattern untuk menentukan has_more.
func (r *studentPostgresRepository) FindAllCursor(
	ctx context.Context, q model.CursorQuery,
) ([]model.Student, bool, string, error) {
	args := []any{}
	where := " WHERE 1=1"

	if q.Search != "" {
		where += fmt.Sprintf(" AND name ILIKE $%d", len(args)+1)
		args = append(args, "%"+q.Search+"%")
	}

	// Decode cursor jika ada
	if q.Cursor != "" {
		cursorCreatedAt, cursorID, err := decodeCursor(q.Cursor)
		if err != nil {
			return nil, false, "", fmt.Errorf("cursor tidak valid: %w", err)
		}
		where += fmt.Sprintf(" AND (created_at, id) < ($%d, $%d)", len(args)+1, len(args)+2)
		args = append(args, cursorCreatedAt, cursorID)
	}

	// Query LIMIT+1 untuk mendeteksi has_more
	fetchLimit := q.Limit + 1
	sqlText := fmt.Sprintf(
		`SELECT id, nim, name, grade, is_active, owner_id, created_at
		 FROM students%s
		 ORDER BY created_at DESC, id DESC
		 LIMIT $%d`,
		where, len(args)+1,
	)
	args = append(args, fetchLimit)

	rows, err := r.pool.Query(ctx, sqlText, args...)
	if err != nil {
		return nil, false, "", fmt.Errorf("mengambil daftar student (cursor): %w", err)
	}
	defer rows.Close()

	hasil := []model.Student{}
	for rows.Next() {
		var s model.Student
		if err := rows.Scan(&s.ID, &s.NIM, &s.Name, &s.Grade, &s.IsActive, &s.OwnerID, &s.CreatedAt); err != nil {
			return nil, false, "", fmt.Errorf("membaca baris student: %w", err)
		}
		hasil = append(hasil, s)
	}
	if err := rows.Err(); err != nil {
		return nil, false, "", fmt.Errorf("membaca hasil query: %w", err)
	}

	// Tentukan has_more dan potong row tambahan
	hasMore := len(hasil) > q.Limit
	if hasMore {
		hasil = hasil[:q.Limit] // potong row tambahan
	}

	// Buat next cursor dari row terakhir yang benar-benar dikirim
	nextCursor := ""
	if hasMore && len(hasil) > 0 {
		last := hasil[len(hasil)-1]
		nextCursor = encodeCursor(last.CreatedAt, last.ID)
	}

	return hasil, hasMore, nextCursor, nil
}

// encodeCursor mengenkode pasangan (created_at, id) menjadi base64 string.
func encodeCursor(createdAt time.Time, id int) string {
	raw := fmt.Sprintf("%s|%d", createdAt.Format(time.RFC3339Nano), id)
	return base64.StdEncoding.EncodeToString([]byte(raw))
}

// decodeCursor mendekode base64 cursor menjadi pasangan (created_at, id).
func decodeCursor(cursor string) (time.Time, int, error) {
	decoded, err := base64.StdEncoding.DecodeString(cursor)
	if err != nil {
		return time.Time{}, 0, fmt.Errorf("gagal decode base64: %w", err)
	}

	parts := strings.SplitN(string(decoded), "|", 2)
	if len(parts) != 2 {
		return time.Time{}, 0, fmt.Errorf("format cursor tidak valid")
	}

	t, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return time.Time{}, 0, fmt.Errorf("gagal parse timestamp: %w", err)
	}

	var id int
	if _, err := fmt.Sscanf(parts[1], "%d", &id); err != nil {
		return time.Time{}, 0, fmt.Errorf("gagal parse id: %w", err)
	}

	return t, id, nil
}

func (r *studentPostgresRepository) FindByID(
	ctx context.Context, id int,
) (model.Student, error) {
	var s model.Student
	err := r.pool.QueryRow(ctx,
		`SELECT id, nim, name, grade, is_active, owner_id, created_at FROM students WHERE id = $1`, id,
	).Scan(&s.ID, &s.NIM, &s.Name, &s.Grade, &s.IsActive, &s.OwnerID, &s.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Student{}, ErrNotFound
		}
		return model.Student{}, fmt.Errorf("mengambil student: %w", err)
	}
	return s, nil
}

func (r *studentPostgresRepository) Create(
	ctx context.Context, s model.Student,
) (model.Student, error) {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO students (nim, name, grade, is_active, owner_id)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, created_at`,
		s.NIM, s.Name, s.Grade, s.IsActive, s.OwnerID,
	).Scan(&s.ID, &s.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return model.Student{}, ErrDuplicate
		}
		return model.Student{}, fmt.Errorf("menyimpan student: %w", err)
	}
	return s, nil
}

func (r *studentPostgresRepository) Update(
	ctx context.Context, s model.Student,
) (model.Student, error) {
	err := r.pool.QueryRow(ctx,
		`UPDATE students SET nim = $1, name = $2, grade = $3, is_active = $4
		 WHERE id = $5
		 RETURNING id, nim, name, grade, is_active, owner_id, created_at`,
		s.NIM, s.Name, s.Grade, s.IsActive, s.ID,
	).Scan(&s.ID, &s.NIM, &s.Name, &s.Grade, &s.IsActive, &s.OwnerID, &s.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Student{}, ErrNotFound
		}
		if isUniqueViolation(err) {
			return model.Student{}, ErrDuplicate
		}
		return model.Student{}, fmt.Errorf("memperbarui student: %w", err)
	}
	return s, nil
}

func (r *studentPostgresRepository) Delete(ctx context.Context, id int) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM students WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("menghapus student: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}

func (r *studentPostgresRepository) FindByNIM(
	ctx context.Context, nim string,
) (model.Student, error) {
	var s model.Student
	err := r.pool.QueryRow(ctx,
		`SELECT id, nim, name, grade, is_active, owner_id, created_at FROM students WHERE nim = $1`, nim,
	).Scan(&s.ID, &s.NIM, &s.Name, &s.Grade, &s.IsActive, &s.OwnerID, &s.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Student{}, ErrNotFound
		}
		return model.Student{}, fmt.Errorf("mengambil student by nim: %w", err)
	}
	return s, nil
}