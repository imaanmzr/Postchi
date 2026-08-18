package environment

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/imaanmzr/postchi/backend/internal/auth"
	appdb "github.com/imaanmzr/postchi/backend/internal/db"
	"github.com/imaanmzr/postchi/backend/internal/shared/crypto"
	"github.com/imaanmzr/postchi/backend/internal/shared/db"
	"github.com/imaanmzr/postchi/backend/internal/testutil"
)

func TestEnvironmentHandlerIntegration(t *testing.T) {
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

	userID, wsID := testutil.SeedWorkspace(t, ctx, pool)
	cryptoSvc, _ := crypto.NewService("postchi-dev-encryption-key-32b!!")
	h := NewHandler(appdb.NewStore(pool), cryptoSvc)

	t.Run("create with stage and resolve variables", func(t *testing.T) {
		createRR := httptest.NewRecorder()
		createBody := `{"workspace_id":"` + wsID.String() + `","name":"Dev","stage":"dev","variables":[{"key":"apiKey","value":"secret","phase":"pre_request","enabled":true,"type":"string","is_secret":true}]}`
		createR := httptest.NewRequest(http.MethodPost, "/api/environments", strings.NewReader(createBody))
		createR = createR.WithContext(context.WithValue(createR.Context(), auth.UserIDKey, userID.String()))
		h.Create(createRR, createR)
		if createRR.Code != http.StatusCreated {
			t.Fatalf("create status %d: %s", createRR.Code, createRR.Body.String())
		}
		var env Environment
		_ = json.Unmarshal(createRR.Body.Bytes(), &env)
		if env.Stage != "dev" {
			t.Fatalf("expected stage dev, got %s", env.Stage)
		}

		resolveRR := httptest.NewRecorder()
		resolveBody := `{"names":["apiKey","missingVar"]}`
		resolveR := httptest.NewRequest(http.MethodPost, "/api/environments/"+env.ID+"/resolve-variables", strings.NewReader(resolveBody))
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", env.ID)
		resolveR = resolveR.WithContext(context.WithValue(resolveR.Context(), chi.RouteCtxKey, rctx))
		h.ResolveVariables(resolveRR, resolveR)
		if resolveRR.Code != http.StatusOK {
			t.Fatalf("resolve status %d: %s", resolveRR.Code, resolveRR.Body.String())
		}
		var res map[string][]string
		_ = json.Unmarshal(resolveRR.Body.Bytes(), &res)
		if len(res["existing"]) != 1 || res["existing"][0] != "apiKey" {
			t.Fatalf("unexpected existing: %v", res["existing"])
		}
		if len(res["missing"]) != 1 || res["missing"][0] != "missingVar" {
			t.Fatalf("unexpected missing: %v", res["missing"])
		}
	})
}
