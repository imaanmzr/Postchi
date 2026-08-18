// Command migrate applies pending SQL migrations and exits.
// Intended for CI/CD init containers / one-shot Jobs:
//
//	docker run --rm -e DATABASE_URL=... postchi-api /app/postchi-migrate
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"

	"github.com/imaanmzr/postchi/backend/internal/shared/config"
	"github.com/imaanmzr/postchi/backend/internal/shared/db"
)

func main() {
	log, _ := zap.NewProduction()
	defer log.Sync()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal("config", zap.Error(err))
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if err := db.WaitForDatabase(ctx, cfg.DatabaseURL, cfg.DBReadyTimeout); err != nil {
		log.Fatal("database", zap.Error(err))
	}

	migrationsPath := db.ResolveMigrationsPath(cfg.MigrationsPath)
	log.Info("running database migrations", zap.String("path", migrationsPath))
	if err := db.RunMigrations(cfg.DatabaseURL, migrationsPath); err != nil {
		log.Fatal("migrations", zap.Error(err))
	}
	log.Info("database migrations complete")
	os.Exit(0)
}
