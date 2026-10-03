package config

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

// LoadEnv loads .env file
func LoadEnv() {
	_ = godotenv.Load()
}

// GetEnv gets an environment variable or default
func GetEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

// ConnectDB connects to PostgreSQL
func ConnectDB(ctx context.Context) (*pgxpool.Pool, error) {
	dbURL := GetEnv("DATABASE_URL", "")
	if dbURL == "" {
		// fallback to construct from parts
		host := GetEnv("DB_HOST", "localhost")
		port := GetEnv("DB_PORT", "5432")
		user := GetEnv("DB_USER", "postgres")
		password := GetEnv("DB_PASSWORD", "postgres")
		dbname := GetEnv("DB_NAME", "siakad_mini")
		dbURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", user, password, host, port, dbname)
	}

	config, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		return nil, fmt.Errorf("unable to parse database url: %v", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to database: %v", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("database unreachable: %v", err)
	}

	return pool, nil
}
