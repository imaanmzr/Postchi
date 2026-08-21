package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	appdb "github.com/imaanmzr/postchi/backend/internal/db"
	"github.com/imaanmzr/postchi/backend/internal/db/sqlc"
	"github.com/imaanmzr/postchi/backend/internal/shared/config"
	"github.com/imaanmzr/postchi/backend/internal/shared/db"
)

func TestChangePasswordIntegration(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL not set")
	}

	ctx := context.Background()
	root := filepath.Join("..", "..", "..", "migrations")
	if err := db.RunMigrations(databaseURL, "file://"+root); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	defer pool.Close()

	store := appdb.NewStore(pool)
	tokens := NewService("test-secret-key-32-bytes-long!", "postchi", 15*time.Minute, 7*24*time.Hour)
	cfg := &config.Config{}
	h := NewHandler(store, tokens, cfg)

	email := "pwchange-" + uuid.New().String()[:8] + "@example.com"
	oldPassword := "old-secret-1"
	newPassword := "new-secret-2"
	hash, err := HashPassword(oldPassword)
	if err != nil {
		t.Fatal(err)
	}
	userID, err := store.CreateUser(ctx, sqlc.CreateUserParams{
		Email:        email,
		PasswordHash: hash,
		DisplayName:  "User",
	})
	if err != nil {
		t.Fatal(err)
	}
	uid := appdb.FromPGUUID(userID)

	t.Run("wrong current password rejected", func(t *testing.T) {
		rr := httptest.NewRecorder()
		body := `{"current_password":"wrong","new_password":"` + newPassword + `"}`
		r := httptest.NewRequest(http.MethodPost, "/api/auth/change-password", strings.NewReader(body))
		r = r.WithContext(context.WithValue(r.Context(), UserIDKey, uid.String()))
		h.ChangePassword(rr, r)
		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("change password succeeds", func(t *testing.T) {
		rr := httptest.NewRecorder()
		body := `{"current_password":"` + oldPassword + `","new_password":"` + newPassword + `"}`
		r := httptest.NewRequest(http.MethodPost, "/api/auth/change-password", strings.NewReader(body))
		r = r.WithContext(context.WithValue(r.Context(), UserIDKey, uid.String()))
		h.ChangePassword(rr, r)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
		var resp map[string]any
		if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
			t.Fatal(err)
		}
		tokens, ok := resp["tokens"].(map[string]any)
		if !ok || tokens["access_token"] == "" || tokens["refresh_token"] == "" {
			t.Fatalf("expected new tokens, got %#v", resp["tokens"])
		}
	})

	t.Run("login with new password", func(t *testing.T) {
		rr := httptest.NewRecorder()
		body := `{"email":"` + email + `","password":"` + newPassword + `"}`
		r := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(body))
		h.Login(rr, r)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("old password no longer works", func(t *testing.T) {
		rr := httptest.NewRecorder()
		body := `{"email":"` + email + `","password":"` + oldPassword + `"}`
		r := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(body))
		h.Login(rr, r)
		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d: %s", rr.Code, rr.Body.String())
		}
	})
}
