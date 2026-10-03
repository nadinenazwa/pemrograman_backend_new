package tests

import (
	"context"
	"testing"
	"time"

	"siakad/config"
	"siakad/internal/repositories"
)

// Example business rule test that requires a DB connection.
// It skips if DB is not reachable.
func TestBatasSKS(t *testing.T) {
	// IPK >= 3.0 -> 24 SKS
	batas1 := repositories.GetBatasSKS(3.50)
	if batas1 != 24 {
		t.Errorf("Expected 24, got %d", batas1)
	}

	// IPK 2.50 - 2.99 -> 21 SKS
	batas2 := repositories.GetBatasSKS(2.75)
	if batas2 != 21 {
		t.Errorf("Expected 21, got %d", batas2)
	}

	// IPK < 2.50 -> 18 SKS
	batas3 := repositories.GetBatasSKS(2.00)
	if batas3 != 18 {
		t.Errorf("Expected 18, got %d", batas3)
	}
}

func TestEnrollmentTransaction(t *testing.T) {
	config.LoadEnv()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	pool, err := config.ConnectDB(ctx)
	if err != nil {
		t.Skip("Database not available for integration test")
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		t.Skip("Database not available for integration test")
	}

	// Just checking the repo initialization, the rest is handled in handlers E2E which
	// needs full DB seeded.
	repo := repositories.NewEnrollmentRepository(pool)
	if repo == nil {
		t.Error("Repo should not be nil")
	}
}
