package workspace

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/imaanmzr/postchi/backend/internal/auth"
	appdb "github.com/imaanmzr/postchi/backend/internal/db"
	"github.com/imaanmzr/postchi/backend/internal/shared/db"
	"github.com/imaanmzr/postchi/backend/internal/testutil"
)

func TestCreateWorkspaceTypeIntegration(t *testing.T) {
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

	userID, _ := testutil.SeedWorkspace(t, ctx, pool)
	store := appdb.NewStore(pool)
	h := NewHandler(store)
	suffix := uuid.New().String()[:8]

	t.Run("pm workspace skips default collection", func(t *testing.T) {
		body := `{"name":"PM Workspace ` + suffix + `","type":"pm"}`
		rr := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/api/workspaces", strings.NewReader(body))
		r = r.WithContext(context.WithValue(r.Context(), auth.UserIDKey, userID.String()))
		h.Create(rr, r)
		if rr.Code != http.StatusCreated {
			t.Fatalf("create pm: %d %s", rr.Code, rr.Body.String())
		}
		var ws Workspace
		if err := json.Unmarshal(rr.Body.Bytes(), &ws); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if ws.Type != "pm" {
			t.Fatalf("expected pm type, got %s", ws.Type)
		}
		wsID, _ := uuid.Parse(ws.ID)
		cols, err := store.ListCollectionsByWorkspace(ctx, appdb.PGUUID(wsID))
		if err != nil {
			t.Fatalf("collections: %v", err)
		}
		if len(cols) != 0 {
			t.Fatalf("pm workspace should not have default collection, got %d", len(cols))
		}
	})

	t.Run("tester workspace gets default collection", func(t *testing.T) {
		body := `{"name":"Tester Workspace ` + suffix + `","type":"tester"}`
		rr := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/api/workspaces", strings.NewReader(body))
		r = r.WithContext(context.WithValue(r.Context(), auth.UserIDKey, userID.String()))
		h.Create(rr, r)
		if rr.Code != http.StatusCreated {
			t.Fatalf("create tester: %d %s", rr.Code, rr.Body.String())
		}
		var ws Workspace
		_ = json.Unmarshal(rr.Body.Bytes(), &ws)
		wsID, _ := uuid.Parse(ws.ID)
		cols, err := store.ListCollectionsByWorkspace(ctx, appdb.PGUUID(wsID))
		if err != nil {
			t.Fatalf("collections: %v", err)
		}
		if len(cols) != 1 {
			t.Fatalf("tester workspace should have default collection, got %d", len(cols))
		}
	})
}
