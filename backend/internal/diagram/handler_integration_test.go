package diagram

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
	"github.com/imaanmzr/postchi/backend/internal/shared/db"
	"github.com/imaanmzr/postchi/backend/internal/testutil"
)

func TestDiagramCRUDIntegration(t *testing.T) {
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
	h := NewHandler(appdb.NewStore(pool))

	createRR := httptest.NewRecorder()
	createReq := httptest.NewRequest(http.MethodPost, "/api/workspaces/"+wsID.String()+"/diagrams", strings.NewReader(`{"title":"Checkout flow"}`))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", wsID.String())
	createReq = createReq.WithContext(context.WithValue(createReq.Context(), chi.RouteCtxKey, rctx))
	createReq = createReq.WithContext(context.WithValue(createReq.Context(), auth.UserIDKey, userID.String()))
	h.Create(createRR, createReq)
	if createRR.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d: %s", createRR.Code, createRR.Body.String())
	}

	var created Diagram
	if err := json.Unmarshal(createRR.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create: %v", err)
	}
	if created.Slug == "" || created.Title != "Checkout flow" {
		t.Fatalf("unexpected create response: %+v", created)
	}

	patchRR := httptest.NewRecorder()
	patchBody := `{"content":{"type":"excalidraw","version":2,"elements":[{"id":"rect1","type":"rectangle"}],"appState":{},"files":{}}}`
	patchReq := httptest.NewRequest(http.MethodPatch, "/api/workspaces/"+wsID.String()+"/diagrams/"+created.Slug, strings.NewReader(patchBody))
	rctx2 := chi.NewRouteContext()
	rctx2.URLParams.Add("id", wsID.String())
	rctx2.URLParams.Add("slug", created.Slug)
	patchReq = patchReq.WithContext(context.WithValue(patchReq.Context(), chi.RouteCtxKey, rctx2))
	h.Update(patchRR, patchReq)
	if patchRR.Code != http.StatusOK {
		t.Fatalf("patch: expected 200, got %d: %s", patchRR.Code, patchRR.Body.String())
	}

	getRR := httptest.NewRecorder()
	getReq := httptest.NewRequest(http.MethodGet, "/api/workspaces/"+wsID.String()+"/diagrams/"+created.Slug, nil)
	rctx3 := chi.NewRouteContext()
	rctx3.URLParams.Add("id", wsID.String())
	rctx3.URLParams.Add("slug", created.Slug)
	getReq = getReq.WithContext(context.WithValue(getReq.Context(), chi.RouteCtxKey, rctx3))
	h.Get(getRR, getReq)
	if getRR.Code != http.StatusOK {
		t.Fatalf("get: expected 200, got %d: %s", getRR.Code, getRR.Body.String())
	}

	var fetched Diagram
	if err := json.Unmarshal(getRR.Body.Bytes(), &fetched); err != nil {
		t.Fatalf("decode get: %v", err)
	}
	elements, ok := fetched.Content["elements"].([]any)
	if !ok || len(elements) != 1 {
		t.Fatalf("expected saved element, got %+v", fetched.Content)
	}

	listRR := httptest.NewRecorder()
	listReq := httptest.NewRequest(http.MethodGet, "/api/workspaces/"+wsID.String()+"/diagrams", nil)
	rctx4 := chi.NewRouteContext()
	rctx4.URLParams.Add("id", wsID.String())
	listReq = listReq.WithContext(context.WithValue(listReq.Context(), chi.RouteCtxKey, rctx4))
	h.List(listRR, listReq)
	if listRR.Code != http.StatusOK {
		t.Fatalf("list: expected 200, got %d: %s", listRR.Code, listRR.Body.String())
	}
}
