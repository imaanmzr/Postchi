package db

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping db: %w", err)
	}
	return pool, nil
}

// ResolveMigrationsPath normalizes MIGRATIONS_PATH for local `go run` and Docker.
func ResolveMigrationsPath(migrationsPath string) string {
	path := migrationsPath
	if strings.HasPrefix(path, "file://") {
		fsPath := strings.TrimPrefix(path, "file://")
		if !filepath.IsAbs(fsPath) {
			if _, err := os.Stat(fsPath); os.IsNotExist(err) {
				return "file://" + filepath.Join("..", "migrations")
			}
		}
		return path
	}
	if !filepath.IsAbs(path) {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return "file://" + filepath.Join("..", "migrations")
		}
		return "file://" + path
	}
	return "file://" + path
}

// WaitForDatabase retries until Postgres accepts connections or timeout/ctx ends.
func WaitForDatabase(ctx context.Context, databaseURL string, timeout time.Duration) error {
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	deadline := time.Now().Add(timeout)
	var lastErr error
	for {
		pool, err := pgxpool.New(ctx, databaseURL)
		if err == nil {
			pingErr := pool.Ping(ctx)
			pool.Close()
			if pingErr == nil {
				return nil
			}
			lastErr = pingErr
		} else {
			lastErr = err
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("database not ready after %s: %w", timeout, lastErr)
		}
		select {
		case <-ctx.Done():
			if lastErr != nil {
				return fmt.Errorf("database wait canceled: %w (last: %v)", ctx.Err(), lastErr)
			}
			return ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}
	}
}

func RunMigrations(databaseURL, migrationsPath string) error {
	m, err := migrate.New(migrationsPath, databaseURL)
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migrate up: %w", err)
	}
	return nil
}
