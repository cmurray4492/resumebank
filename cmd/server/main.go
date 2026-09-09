// Command server runs the resumebank.biz HTTP server.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"resumebank/internal/app"
	"resumebank/internal/background"
	"resumebank/internal/config"
	"resumebank/internal/db"
	"resumebank/internal/web"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	pool, err := db.Open(ctx, cfg.DatabaseURL)
	cancel()
	if err != nil {
		log.Fatalf("database error: %v", err)
	}
	defer pool.Close()

	if cfg.AutoMigrate {
		migrateCtx, migrateCancel := context.WithTimeout(context.Background(), 60*time.Second)
		err := db.Migrate(migrateCtx, pool)
		migrateCancel()
		if err != nil {
			log.Fatalf("auto-migration failed: %v", err)
		}
	}

	a, err := app.New(cfg, pool)
	if err != nil {
		log.Fatalf("app init error: %v", err)
	}

	initCtx, initCancel := context.WithTimeout(context.Background(), 30*time.Second)
	if err := a.RefreshSitemap(initCtx); err != nil {
		log.Printf("initial sitemap generation failed: %v", err)
	}
	if err := a.RefreshSearchIndex(initCtx); err != nil {
		log.Printf("initial search index refresh failed: %v", err)
	}
	initCancel()

	// Embeddings backfill gets its own longer, independent timeout: a slow
	// or unreachable Ollama server here must never delay server startup or
	// be conflated with the sitemap/search-index refreshes above.
	embedCtx, embedCancel := context.WithTimeout(context.Background(), 60*time.Second)
	if err := a.RefreshMissingEmbeddings(embedCtx); err != nil {
		log.Printf("initial embeddings backfill failed: %v", err)
	}
	embedCancel()

	bgCtx, stopBackground := context.WithCancel(context.Background())
	defer stopBackground()
	go background.RunEvery(bgCtx, 24*time.Hour, "sitemap refresh", a.RefreshSitemap)
	go background.RunEvery(bgCtx, time.Hour, "search index refresh", a.RefreshSearchIndex)
	go background.RunEvery(bgCtx, 5*time.Minute, "embeddings backfill", a.RefreshMissingEmbeddings)

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      web.NewRouter(a),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("listening on :%s (env=%s)", cfg.Port, cfg.Env)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Println("shutting down...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown error: %v", err)
	}
}
