package request

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
	"github.com/imaanmzr/postchi/backend/internal/shared/config"
	"github.com/imaanmzr/postchi/backend/internal/shared/crypto"
	"github.com/imaanmzr/postchi/backend/internal/shared/db"
	"github.com/imaanmzr/postchi/backend/internal/testutil"
)

func TestHandlerIntegration(t *testing.T) {
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
	cfg := &config.Config{EncryptionKey: "postchi-dev-encryption-key-32b!!"}
	cryptoSvc, err := crypto.NewService(cfg.EncryptionKey)
	if err != nil {
		t.Fatalf("crypto: %v", err)
	}
	h := NewHandler(appdb.NewStore(pool), cfg, cryptoSvc)
	store := appdb.NewStore(pool)

	parentCol := testutil.SeedCollection(t, ctx, pool, wsID, userID, "Parent", nil)
	childCol := testutil.SeedCollection(t, ctx, pool, wsID, userID, "Child Folder", &parentCol)
	reqParent := testutil.SeedRequest(t, ctx, pool, parentCol, userID, "In Parent")
	reqChild := testutil.SeedRequest(t, ctx, pool, childCol, userID, "In Child")

	t.Run("ListByWorkspace returns requests in nested folders", func(t *testing.T) {
		rr := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/api/workspaces/"+wsID.String()+"/requests", nil)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", wsID.String())
		r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
		h.ListByWorkspace(rr, r)
		if rr.Code != http.StatusOK {
			t.Fatalf("status %d: %s", rr.Code, rr.Body.String())
		}
		var list []Model
		if err := json.Unmarshal(rr.Body.Bytes(), &list); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(list) < 2 {
			t.Fatalf("expected at least 2 requests, got %d: %s", len(list), rr.Body.String())
		}
		ids := map[string]bool{}
		for _, m := range list {
			ids[m.ID] = true
		}
		if !ids[reqParent.String()] || !ids[reqChild.String()] {
			t.Fatalf("missing seeded requests in list: %+v", ids)
		}
	})

	t.Run("template child inherits url from template", func(t *testing.T) {
		templateID := testutil.SeedRequest(t, ctx, pool, parentCol, userID, "Template")
		_, _ = store.Pool.Exec(ctx, `UPDATE requests SET is_template=true, url='https://template.example' WHERE id=$1`, templateID)

		rr := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/api/requests/"+templateID.String()+"/children", strings.NewReader(`{"name":"Variant"}`))
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", templateID.String())
		r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
		r = r.WithContext(context.WithValue(r.Context(), auth.UserIDKey, userID.String()))
		h.CreateChild(rr, r)
		if rr.Code != http.StatusCreated {
			t.Fatalf("create child status %d: %s", rr.Code, rr.Body.String())
		}
		var child Model
		_ = json.Unmarshal(rr.Body.Bytes(), &child)

		getRR := httptest.NewRecorder()
		getR := httptest.NewRequest(http.MethodGet, "/api/requests/"+child.ID, nil)
		getCtx := chi.NewRouteContext()
		getCtx.URLParams.Add("id", child.ID)
		getR = getR.WithContext(context.WithValue(getR.Context(), chi.RouteCtxKey, getCtx))
		h.Get(getRR, getR)
		if getRR.Code != http.StatusOK {
			t.Fatalf("get child status %d: %s", getRR.Code, getRR.Body.String())
		}
		var merged Model
		_ = json.Unmarshal(getRR.Body.Bytes(), &merged)
		if merged.URL != "https://template.example" {
			t.Fatalf("expected inherited url, got %s", merged.URL)
		}
	})
}
