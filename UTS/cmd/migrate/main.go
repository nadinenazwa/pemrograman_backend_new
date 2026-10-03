package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"siakad/config"
)

func main() {
	config.LoadEnv()
	ctx := context.Background()
	pool, err := config.ConnectDB(ctx)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	schema, err := os.ReadFile("database/migrations/schema.sql")
	if err != nil {
		log.Fatalf("Failed to read schema.sql: %v", err)
	}

	_, err = pool.Exec(ctx, string(schema))
	if err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	fmt.Println("Migration completed successfully.")
}
