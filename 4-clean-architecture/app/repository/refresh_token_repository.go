package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RefreshToken merepresentasikan record refresh token di database.
type RefreshToken struct {
	ID        int
	UserID    int
	TokenHash string
	ExpiresAt time.Time
	RevokedAt *time.Time
	CreatedAt time.Time
}

// RefreshTokenRepository mendefinisikan kontrak akses data untuk refresh token.
type RefreshTokenRepository interface {
	Create(ctx context.Context, userID int, tokenHash string, expiresAt time.Time) error
	FindByTokenHash(ctx context.Context, tokenHash string) (RefreshToken, error)
	RevokeByTokenHash(ctx context.Context, tokenHash string) error
	RevokeAllByUserID(ctx context.Context, userID int) error
}

type refreshTokenPostgresRepository struct {
	pool *pgxpool.Pool
}

// NewRefreshTokenRepository membuat instance RefreshTokenRepository baru.
func NewRefreshTokenRepository(pool *pgxpool.Pool) RefreshTokenRepository {
	return &refreshTokenPostgresRepository{pool: pool}
}

func (r *refreshTokenPostgresRepository) Create(
	ctx context.Context, userID int, tokenHash string, expiresAt time.Time,
) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
		 VALUES ($1, $2, $3)`,
		userID, tokenHash, expiresAt,
	)
	if err != nil {
		return fmt.Errorf("menyimpan refresh token: %w", err)
	}
	return nil
}

func (r *refreshTokenPostgresRepository) FindByTokenHash(
	ctx context.Context, tokenHash string,
) (RefreshToken, error) {
	var rt RefreshToken
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, token_hash, expires_at, revoked_at, created_at
		 FROM refresh_tokens WHERE token_hash = $1`, tokenHash,
	).Scan(&rt.ID, &rt.UserID, &rt.TokenHash, &rt.ExpiresAt, &rt.RevokedAt, &rt.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return RefreshToken{}, ErrNotFound
		}
		return RefreshToken{}, fmt.Errorf("mengambil refresh token: %w", err)
	}
	return rt, nil
}

func (r *refreshTokenPostgresRepository) RevokeByTokenHash(
	ctx context.Context, tokenHash string,
) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE refresh_tokens SET revoked_at = NOW() WHERE token_hash = $1 AND revoked_at IS NULL`,
		tokenHash,
	)
	if err != nil {
		return fmt.Errorf("mencabut refresh token: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *refreshTokenPostgresRepository) RevokeAllByUserID(
	ctx context.Context, userID int,
) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE refresh_tokens SET revoked_at = NOW() WHERE user_id = $1 AND revoked_at IS NULL`,
		userID,
	)
	if err != nil {
		return fmt.Errorf("mencabut semua refresh token user: %w", err)
	}
	return nil
}
