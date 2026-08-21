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
	"github.com/imaanmzr/postchi/backend/internal/shared/config"
	"github.com/imaanmzr/postchi/backend/internal/shared/db"
)

func TestRegisterDomainAllowlistIntegration(t *testing.T) {
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
	tokens := NewService("test-secret", "postchi", 15*time.Minute, 7*24*time.Hour)

	t.Run("allowlist rejects outside domain", func(t *testing.T) {
		cfg := &config.Config{RegistrationAllowedDomains: []string{"company.com"}}
		h := NewHandler(store, tokens, cfg)
		email := "user-" + uuid.New().String()[:8] + "@other.com"
		rr := httptest.NewRecorder()
		body := `{"email":"` + email + `","password":"secret123","display_name":"User"}`
		r := httptest.NewRequest(http.MethodPost, "/api/auth/register", strings.NewReader(body))
		h.Register(rr, r)
		if rr.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("allowlist accepts inside domain", func(t *testing.T) {
		cfg := &config.Config{RegistrationAllowedDomains: []string{"company.com"}}
		h := NewHandler(store, tokens, cfg)
		email := "user-" + uuid.New().String()[:8] + "@company.com"
		rr := httptest.NewRecorder()
		body := `{"email":"` + email + `","password":"secret123","display_name":"User"}`
		r := httptest.NewRequest(http.MethodPost, "/api/auth/register", strings.NewReader(body))
		h.Register(rr, r)
		if rr.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
		}
		var resp authResponse
		if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if resp.User.Email != email {
			t.Fatalf("expected %s, got %s", email, resp.User.Email)
		}
	})

	t.Run("empty allowlist permits any domain", func(t *testing.T) {
		cfg := &config.Config{}
		h := NewHandler(store, tokens, cfg)
		email := "user-" + uuid.New().String()[:8] + "@random.org"
		rr := httptest.NewRecorder()
		body := `{"email":"` + email + `","password":"secret123","display_name":"User"}`
		r := httptest.NewRequest(http.MethodPost, "/api/auth/register", strings.NewReader(body))
		h.Register(rr, r)
		if rr.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
		}
	})
}
