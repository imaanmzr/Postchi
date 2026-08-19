package share

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
	"github.com/imaanmzr/postchi/backend/internal/shared/db"
	"github.com/imaanmzr/postchi/backend/internal/testutil"
)

func TestShareHandlerIntegration(t *testing.T) {
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
	colID := testutil.SeedCollection(t, ctx, pool, wsID, userID, "Share Col", nil)
	reqID := testutil.SeedRequest(t, ctx, pool, colID, userID, "Shareable")

	cfg := &config.Config{AppPublicURL: "http://localhost:3000"}
	h := NewHandler(appdb.NewStore(pool), cfg)
	store := appdb.NewStore(pool)

	t.Run("create and get link share", func(t *testing.T) {
		rr := httptest.NewRecorder()
		body := `{"kind":"request","source_id":"` + reqID.String() + `","workspace_id":"` + wsID.String() + `","visibility":"link"}`
		r := httptest.NewRequest(http.MethodPost, "/api/shares", strings.NewReader(body))
		r = r.WithContext(context.WithValue(r.Context(), auth.UserIDKey, userID.String()))
		h.Create(rr, r)
		if rr.Code != http.StatusCreated {
			t.Fatalf("create status %d: %s", rr.Code, rr.Body.String())
		}
		var created Share
		_ = json.Unmarshal(rr.Body.Bytes(), &created)

		getRR := httptest.NewRecorder()
		getR := httptest.NewRequest(http.MethodGet, "/api/shares/"+created.Token, nil)
		getCtx := chi.NewRouteContext()
		getCtx.URLParams.Add("token", created.Token)
		getR = getR.WithContext(context.WithValue(getR.Context(), chi.RouteCtxKey, getCtx))
		h.GetByToken(getRR, getR)
		if getRR.Code != http.StatusOK {
			t.Fatalf("get status %d: %s", getRR.Code, getRR.Body.String())
		}
	})

	t.Run("create workspace catalog share", func(t *testing.T) {
		rr := httptest.NewRecorder()
		body := `{"kind":"catalog","source_id":"` + wsID.String() + `","workspace_id":"` + wsID.String() + `","landing_request_id":"` + reqID.String() + `","visibility":"link"}`
		r := httptest.NewRequest(http.MethodPost, "/api/shares", strings.NewReader(body))
		r = r.WithContext(context.WithValue(r.Context(), auth.UserIDKey, userID.String()))
		h.Create(rr, r)
		if rr.Code != http.StatusCreated {
			t.Fatalf("create status %d: %s", rr.Code, rr.Body.String())
		}

		var created Share
		if err := json.Unmarshal(rr.Body.Bytes(), &created); err != nil {
			t.Fatalf("decode share: %v", err)
		}
		endpoints, ok := created.Snapshot["endpoints"].([]any)
		if !ok || len(endpoints) != 1 {
			t.Fatalf("expected one catalog endpoint, got %#v", created.Snapshot["endpoints"])
		}
		collections, ok := created.Snapshot["collections"].([]any)
		if !ok || len(collections) != 1 {
			t.Fatalf("expected one catalog collection, got %#v", created.Snapshot["collections"])
		}
		if created.Snapshot["landing_request_id"] != reqID.String() {
			t.Fatalf("expected landing request %s, got %#v", reqID, created.Snapshot["landing_request_id"])
		}
	})

	t.Run("reject catalog landing request outside snapshot", func(t *testing.T) {
		rr := httptest.NewRecorder()
		body := `{"kind":"catalog","source_id":"` + wsID.String() + `","workspace_id":"` + wsID.String() + `","landing_request_id":"` + wsID.String() + `","visibility":"link"}`
		r := httptest.NewRequest(http.MethodPost, "/api/shares", strings.NewReader(body))
		r = r.WithContext(context.WithValue(r.Context(), auth.UserIDKey, userID.String()))
		h.Create(rr, r)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("revoked share returns gone", func(t *testing.T) {
		rr := httptest.NewRecorder()
		body := `{"kind":"request","source_id":"` + reqID.String() + `","workspace_id":"` + wsID.String() + `","visibility":"link"}`
		r := httptest.NewRequest(http.MethodPost, "/api/shares", strings.NewReader(body))
		r = r.WithContext(context.WithValue(r.Context(), auth.UserIDKey, userID.String()))
		h.Create(rr, r)
		var created Share
		_ = json.Unmarshal(rr.Body.Bytes(), &created)

		revRR := httptest.NewRecorder()
		revR := httptest.NewRequest(http.MethodDelete, "/api/shares/"+created.ID, nil)
		revCtx := chi.NewRouteContext()
		revCtx.URLParams.Add("id", created.ID)
		revR = revR.WithContext(context.WithValue(revR.Context(), chi.RouteCtxKey, revCtx))
		revR = revR.WithContext(context.WithValue(revR.Context(), auth.UserIDKey, userID.String()))
		h.Revoke(revRR, revR)

		getRR := httptest.NewRecorder()
		getR := httptest.NewRequest(http.MethodGet, "/api/shares/"+created.Token, nil)
		getCtx := chi.NewRouteContext()
		getCtx.URLParams.Add("token", created.Token)
		getR = getR.WithContext(context.WithValue(getR.Context(), chi.RouteCtxKey, getCtx))
		h.GetByToken(getRR, getR)
		if getRR.Code != http.StatusGone {
			t.Fatalf("expected 410, got %d", getRR.Code)
		}
	})

	t.Run("import share creates independent copy", func(t *testing.T) {
		rr := httptest.NewRecorder()
		body := `{"kind":"request","source_id":"` + reqID.String() + `","workspace_id":"` + wsID.String() + `","visibility":"link"}`
		r := httptest.NewRequest(http.MethodPost, "/api/shares", strings.NewReader(body))
		r = r.WithContext(context.WithValue(r.Context(), auth.UserIDKey, userID.String()))
		h.Create(rr, r)
		var created Share
		_ = json.Unmarshal(rr.Body.Bytes(), &created)

		targetCol := testutil.SeedCollection(t, ctx, pool, wsID, userID, "Import Target", nil)
		importRR := httptest.NewRecorder()
		importBody := `{"workspace_id":"` + wsID.String() + `","collection_id":"` + targetCol.String() + `"}`
		importR := httptest.NewRequest(http.MethodPost, "/api/shares/"+created.Token+"/import", strings.NewReader(importBody))
		importCtx := chi.NewRouteContext()
		importCtx.URLParams.Add("token", created.Token)
		importR = importR.WithContext(context.WithValue(importR.Context(), chi.RouteCtxKey, importCtx))
		importR = importR.WithContext(context.WithValue(importR.Context(), auth.UserIDKey, userID.String()))
		h.Import(importRR, importR)
		if importRR.Code != http.StatusCreated {
			t.Fatalf("import status %d: %s", importRR.Code, importRR.Body.String())
		}
		reqs, _ := store.ListRequestIDsByCollection(ctx, appdb.PGUUID(targetCol))
		if len(reqs) != 1 {
			t.Fatalf("expected 1 imported request, got %d", len(reqs))
		}
	})
}
