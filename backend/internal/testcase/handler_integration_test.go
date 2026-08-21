package testcase

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
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/imaanmzr/postchi/backend/internal/auth"
	appdb "github.com/imaanmzr/postchi/backend/internal/db"
	"github.com/imaanmzr/postchi/backend/internal/shared/db"
	"github.com/imaanmzr/postchi/backend/internal/testutil"
)

func TestTestCaseCRUDIntegration(t *testing.T) {
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
	colID := testutil.SeedCollection(t, ctx, pool, wsID, userID, "API", nil)
	reqID := testutil.SeedRequest(t, ctx, pool, colID, userID, "Get health")
	h := NewHandler(appdb.NewStore(pool))

	createRR := httptest.NewRecorder()
	createReq := httptest.NewRequest(http.MethodPost, "/api/workspaces/"+wsID.String()+"/test-cases", strings.NewReader(`{"title":"Health check","description":"Verify /health returns 200"}`))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", wsID.String())
	createReq = createReq.WithContext(context.WithValue(createReq.Context(), chi.RouteCtxKey, rctx))
	createReq = createReq.WithContext(context.WithValue(createReq.Context(), auth.UserIDKey, userID.String()))
	h.Create(createRR, createReq)
	if createRR.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", createRR.Code, createRR.Body.String())
	}

	var tc TestCase
	if err := json.Unmarshal(createRR.Body.Bytes(), &tc); err != nil {
		t.Fatalf("decode: %v", err)
	}

	linkRR := httptest.NewRecorder()
	linkReq := httptest.NewRequest(http.MethodPost, "/api/workspaces/"+wsID.String()+"/test-cases/"+tc.ID+"/requests/"+reqID.String(), nil)
	rctx2 := chi.NewRouteContext()
	rctx2.URLParams.Add("id", wsID.String())
	rctx2.URLParams.Add("testCaseId", tc.ID)
	rctx2.URLParams.Add("requestId", reqID.String())
	linkReq = linkReq.WithContext(context.WithValue(linkReq.Context(), chi.RouteCtxKey, rctx2))
	h.AddRequestLink(linkRR, linkReq)
	if linkRR.Code != http.StatusCreated {
		t.Fatalf("link: %d %s", linkRR.Code, linkRR.Body.String())
	}

	getRR := httptest.NewRecorder()
	getReq := httptest.NewRequest(http.MethodGet, "/api/workspaces/"+wsID.String()+"/test-cases/"+tc.ID, nil)
	rctx3 := chi.NewRouteContext()
	rctx3.URLParams.Add("id", wsID.String())
	rctx3.URLParams.Add("testCaseId", tc.ID)
	getReq = getReq.WithContext(context.WithValue(getReq.Context(), chi.RouteCtxKey, rctx3))
	h.Get(getRR, getReq)
	if getRR.Code != http.StatusOK {
		t.Fatalf("get: %d %s", getRR.Code, getRR.Body.String())
	}

	var fetched TestCase
	_ = json.Unmarshal(getRR.Body.Bytes(), &fetched)
	if len(fetched.Requests) != 1 || fetched.Requests[0].ID != reqID.String() {
		t.Fatalf("expected linked request, got %+v", fetched.Requests)
	}
	_ = uuid.Nil
}
