package database

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

// NewMongoClient membuat koneksi ke MongoDB menggunakan URI dari environment variable.
// Mengembalikan error jika URI tidak dikonfigurasi atau koneksi gagal.
// Caller wajib memanggil client.Disconnect() saat aplikasi berhenti.
func NewMongoClient(ctx context.Context) (*mongo.Client, error) {
	uri := os.Getenv("MONGO_URI")
	if uri == "" {
		return nil, fmt.Errorf("MONGO_URI tidak dikonfigurasi di environment")
	}

	opts := options.Client().ApplyURI(uri).
		SetConnectTimeout(10 * time.Second).
		SetServerSelectionTimeout(10 * time.Second)

	client, err := mongo.Connect(opts)
	if err != nil {
		return nil, fmt.Errorf("gagal membuat koneksi MongoDB: %w", err)
	}

	// Ping untuk memastikan server dapat dihubungi sebelum aplikasi start
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := client.Ping(pingCtx, readpref.Primary()); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, fmt.Errorf("gagal terhubung ke MongoDB: %w", err)
	}

	return client, nil
}

// GetMongoDB mengembalikan database instance berdasarkan MONGO_DATABASE env var.
// Fallback ke "student_portfolio_db" jika tidak dikonfigurasi.
func GetMongoDB(client *mongo.Client) *mongo.Database {
	dbName := os.Getenv("MONGO_DATABASE")
	if dbName == "" {
		dbName = "student_portfolio_db"
	}
	return client.Database(dbName)
}
