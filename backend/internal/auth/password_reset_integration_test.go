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

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	appdb "github.com/imaanmzr/postchi/backend/internal/db"
	"github.com/imaanmzr/postchi/backend/internal/db/sqlc"
	"github.com/imaanmzr/postchi/backend/internal/shared/config"
	"github.com/imaanmzr/postchi/backend/internal/shared/db"
)

func TestPasswordResetIntegration(t *testing.T) {
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
	cfg := &config.Config{
		AppPublicURL:     "http://localhost:3000",
		PasswordResetTTL: time.Hour,
	}
	h := NewHandler(store, tokens, cfg)

	email := "pwreset-" + uuid.New().String()[:8] + "@example.com"
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

	t.Run("forgot password returns generic success", func(t *testing.T) {
		rr := httptest.NewRecorder()
		body := `{"email":"` + email + `"}`
		r := httptest.NewRequest(http.MethodPost, "/api/auth/forgot-password", strings.NewReader(body))
		h.ForgotPassword(rr, r)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
		var resp map[string]string
		if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
			t.Fatal(err)
		}
		if resp["message"] != forgotPasswordMessage {
			t.Fatalf("unexpected message: %q", resp["message"])
		}
	})

	t.Run("forgot password for unknown email returns generic success", func(t *testing.T) {
		rr := httptest.NewRecorder()
		body := `{"email":"missing-` + uuid.New().String()[:8] + `@example.com"}`
		r := httptest.NewRequest(http.MethodPost, "/api/auth/forgot-password", strings.NewReader(body))
		h.ForgotPassword(rr, r)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	rawToken, err := generateRandomToken(32)
	if err != nil {
		t.Fatal(err)
	}
	tokenHash := HashToken(rawToken)
	expiresAt := time.Now().Add(time.Hour)
	if err := store.CreatePasswordResetToken(ctx, sqlc.CreatePasswordResetTokenParams{
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: appdb.PGTimestamptz(expiresAt),
	}); err != nil {
		t.Fatal(err)
	}

	t.Run("preview valid reset token", func(t *testing.T) {
		rr := httptest.NewRecorder()
		r := resetPasswordRequest(http.MethodGet, rawToken, "")
		h.PreviewResetPassword(rr, r)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
		var preview resetPasswordPreview
		if err := json.Unmarshal(rr.Body.Bytes(), &preview); err != nil {
			t.Fatal(err)
		}
		if preview.Email != email || preview.Expired {
			t.Fatalf("unexpected preview: %#v", preview)
		}
	})

	t.Run("reset password succeeds", func(t *testing.T) {
		rr := httptest.NewRecorder()
		body := `{"password":"` + newPassword + `"}`
		r := resetPasswordRequest(http.MethodPost, rawToken, body)
		h.ResetPassword(rr, r)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("reset token is single use", func(t *testing.T) {
		rr := httptest.NewRecorder()
		body := `{"password":"another-password"}`
		r := resetPasswordRequest(http.MethodPost, rawToken, body)
		h.ResetPassword(rr, r)
		if rr.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d: %s", rr.Code, rr.Body.String())
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

func resetPasswordRequest(method, token, body string) *http.Request {
	r := httptest.NewRequest(method, "/api/auth/reset-password/"+token, strings.NewReader(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("token", token)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}
