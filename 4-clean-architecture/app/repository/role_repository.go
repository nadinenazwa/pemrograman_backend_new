package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// RoleRepository mendefinisikan kontrak untuk mengambil data RBAC dari database.
type RoleRepository interface {
	LoadPermissions(ctx context.Context) (map[string][]string, error)
}

type rolePostgresRepository struct {
	pool *pgxpool.Pool
}

// NewRoleRepository membuat instance RoleRepository baru.
func NewRoleRepository(pool *pgxpool.Pool) RoleRepository {
	return &rolePostgresRepository{pool: pool}
}

// LoadPermissions mengambil seluruh mapping role → permissions dari database.
// Menggunakan LEFT JOIN agar role tanpa permission (seperti 'user') tetap muncul.
// Dipanggil SEKALI saat startup, tidak per-request.
func (r *rolePostgresRepository) LoadPermissions(ctx context.Context) (map[string][]string, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT r.name, COALESCE(rp.permission_name, '')
		 FROM roles r
		 LEFT JOIN role_permissions rp
		     ON rp.role_name = r.name
		 ORDER BY r.name, rp.permission_name`,
	)
	if err != nil {
		return nil, fmt.Errorf("mengambil role permissions: %w", err)
	}
	defer rows.Close()

	result := make(map[string][]string)
	for rows.Next() {
		var roleName, permName string
		if err := rows.Scan(&roleName, &permName); err != nil {
			return nil, fmt.Errorf("membaca baris role permission: %w", err)
		}

		// Inisialisasi slice jika role belum ada
		if _, ok := result[roleName]; !ok {
			result[roleName] = []string{}
		}

		// Jika permission bukan string kosong (dari COALESCE), tambahkan
		if permName != "" {
			result[roleName] = append(result[roleName], permName)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("membaca hasil query role permissions: %w", err)
	}

	return result, nil
}
