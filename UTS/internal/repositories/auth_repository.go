package repositories

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"siakad/internal/models"
)

type AuthRepository struct {
	pool *pgxpool.Pool
}

func NewAuthRepository(pool *pgxpool.Pool) *AuthRepository {
	return &AuthRepository{pool: pool}
}

func (r *AuthRepository) FindUserByEmail(ctx context.Context, email string) (*models.User, error) {
	query := `SELECT id, email, password, role FROM users WHERE email = $1`
	row := r.pool.QueryRow(ctx, query, email)

	var user models.User
	err := row.Scan(&user.ID, &user.Email, &user.Password, &user.Role)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	// For student, we must check if deleted_at is not null
	if user.Role == "mahasiswa" {
		var deletedAt *string
		err := r.pool.QueryRow(ctx, `SELECT deleted_at::text FROM students WHERE user_id = $1`, user.ID).Scan(&deletedAt)
		if err == nil && deletedAt != nil {
			return nil, nil // treated as not found / cannot login
		}
	}

	return &user, nil
}

func (r *AuthRepository) GetMeStudentData(ctx context.Context, userID int) (*models.MeStudentData, error) {
	query := `SELECT nim, nama, prodi, angkatan FROM students WHERE user_id = $1 AND deleted_at IS NULL`
	row := r.pool.QueryRow(ctx, query, userID)

	var student models.MeStudentData
	err := row.Scan(&student.NIM, &student.Nama, &student.Prodi, &student.Angkatan)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &student, nil
}

func (r *AuthRepository) GetUserByID(ctx context.Context, id int) (*models.User, error) {
	query := `SELECT id, email, password, role FROM users WHERE id = $1`
	row := r.pool.QueryRow(ctx, query, id)

	var user models.User
	err := row.Scan(&user.ID, &user.Email, &user.Password, &user.Role)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}
