// Command createadmin creates or updates an admin account. There is no
// public admin signup route, so this is the only way to provision one:
//
//	go run ./cmd/createadmin -email=admin@example.com -password=some-long-password
//
// Running it again with the same email updates that account's password
// (and promotes it to admin if it wasn't already), so it's also how you
// reset a lost admin password.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"resumebank/internal/auth"
	"resumebank/internal/config"
	"resumebank/internal/db"
	"resumebank/internal/models"
	"resumebank/internal/repo"
)

func main() {
	email := flag.String("email", "", "admin account email (required)")
	password := flag.String("password", "", "admin account password (required, min 8 characters)")
	flag.Parse()

	if *email == "" || *password == "" {
		fmt.Fprintln(os.Stderr, "usage: createadmin -email=admin@example.com -password=...")
		os.Exit(1)
	}
	if len(*password) < 8 {
		log.Fatal("password must be at least 8 characters")
	}

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

	passwordHash, err := auth.HashPassword(*password)
	if err != nil {
		log.Fatalf("hashing password: %v", err)
	}

	users := repo.NewUserRepo(pool)
	existing, err := users.GetByEmail(ctx, *email)
	if err != nil && err != repo.ErrNotFound {
		log.Fatalf("looking up existing user: %v", err)
	}

	if existing != nil {
		if existing.Role != models.RoleAdmin {
			log.Fatalf("a %s account already exists with email %s - refusing to convert it to admin; use a different email for the admin account", existing.Role, *email)
		}
		if err := users.SetPassword(ctx, existing.ID, passwordHash); err != nil {
			log.Fatalf("resetting admin password: %v", err)
		}
		log.Printf("reset password for existing admin account %s", *email)
		return
	}

	if _, err := users.Create(ctx, *email, passwordHash, models.RoleAdmin); err != nil {
		log.Fatalf("creating admin account: %v", err)
	}
	log.Printf("created admin account %s", *email)
}
