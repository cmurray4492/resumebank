// Command migrate applies pending database migrations.
package main

import (
	"context"
	"log"
	"time"

	"resumebank/internal/config"
	"resumebank/internal/db"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pool, err := db.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database error: %v", err)
	}
	defer pool.Close()

	if err := db.Migrate(ctx, pool); err != nil {
		log.Fatalf("migration error: %v", err)
	}
	log.Println("migrations applied successfully")
}
