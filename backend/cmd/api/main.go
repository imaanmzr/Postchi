package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"

	"github.com/imaanmzr/postchi/backend/internal/server"
	"github.com/imaanmzr/postchi/backend/internal/shared/config"
	"github.com/imaanmzr/postchi/backend/internal/shared/db"
)

// Pinned: Go 1.26.3, chi v5.2.1, pgx v5.7.4, zap v1.27.0
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
	if cfg.AutoMigrate {
		log.Info("running database migrations", zap.String("path", migrationsPath))
		if err := db.RunMigrations(cfg.DatabaseURL, migrationsPath); err != nil {
			log.Fatal("migrations", zap.Error(err))
		}
		log.Info("database migrations complete")
	} else {
		log.Info("AUTO_MIGRATE=false; skipping migrations")
	}

	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal("db", zap.Error(err))
	}
	defer pool.Close()

	srv := server.New(cfg, log, pool)

	go func() {
		log.Info("postchi api listening", zap.String("port", cfg.HTTPPort))
		if err := srv.Start(); err != nil && err != context.Canceled {
			log.Fatal("server", zap.Error(err))
		}
	}()

	<-ctx.Done()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("shutdown", zap.Error(err))
		os.Exit(1)
	}
}
