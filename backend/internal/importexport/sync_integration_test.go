package importexport

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/imaanmzr/postchi/backend/internal/auth"
	appdb "github.com/imaanmzr/postchi/backend/internal/db"
	"github.com/imaanmzr/postchi/backend/internal/db/sqlc"
	"github.com/imaanmzr/postchi/backend/internal/shared/crypto"
	"github.com/imaanmzr/postchi/backend/internal/testutil"
)

func TestSyncIntegration(t *testing.T) {
	ctx := context.Background()
	pool := requireIntegrationDB(t)

	ownerID, wsID := testutil.SeedWorkspace(t, ctx, pool)
	viewerID := seedViewer(t, ctx, pool, wsID)
	store := appdb.NewStore(pool)
	cryptoSvc, err := crypto.NewService("0123456789abcdef0123456789abcdef")
	if err != nil {
		t.Fatalf("crypto: %v", err)
	}
	h := NewHandler(store, cryptoSvc)

	gitServer := newGitLabFileServer(t, map[string]string{
		"collection.bru": "meta {\n  name: Sync API\n  type: collection\n}\n",
		"health.bru":     "meta {\n  name: Health\n  type: http\n  seq: 1\n}\n\nget {\n  url: https://example.com/health\n}\n",
	})
	defer gitServer.Close()

	sourceID := createGitSource(t, ctx, h, wsID, ownerID, gitServer.URL+"/group/repository", "Sync API")

	t.Run("create source performs initial sync", func(t *testing.T) {
		reqs, err := store.ListBrunoSyncedRequests(ctx, appdb.PGUUID(sourceID))
		if err != nil {
			t.Fatal(err)
		}
		if len(reqs) < 1 {
			t.Fatalf("expected synced requests after create, got %d", len(reqs))
		}
	})

	t.Run("repeated sync is idempotent", func(t *testing.T) {
		first := syncSource(t, h, sourceID, ownerID)
		second := syncSource(t, h, sourceID, ownerID)
		if second.AddedRequests != 0 || second.AddedCollections != 0 {
			t.Fatalf("expected no new rows, got %+v after %+v", second, first)
		}
	})

	t.Run("viewer cannot sync", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/bruno-sources/"+sourceID.String()+"/sync", nil)
		routeContext := chi.NewRouteContext()
		routeContext.URLParams.Add("id", sourceID.String())
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeContext))
		req = req.WithContext(context.WithValue(req.Context(), auth.UserIDKey, viewerID.String()))
		h.SyncBrunoSource(rr, req)
		if rr.Code != http.StatusForbidden {
			t.Fatalf("status %d: %s", rr.Code, rr.Body.String())
		}
	})
}

func createGitSource(t *testing.T, ctx context.Context, h *Handler, wsID, userID uuid.UUID, repoURL, name string) uuid.UUID {
	t.Helper()
	body := `{"name":"` + name + `","config":{"repo_url":"` + repoURL + `","branch":"main"},"access_token":"gitlab-token"}`
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/workspaces/"+wsID.String()+"/bruno-sources", strings.NewReader(body))
	routeContext := chi.NewRouteContext()
	routeContext.URLParams.Add("id", wsID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeContext))
	req = req.WithContext(context.WithValue(req.Context(), auth.UserIDKey, userID.String()))
	h.CreateBrunoSource(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create source status %d: %s", rr.Code, rr.Body.String())
	}
	var resp struct {
		Source struct {
			ID string `json:"id"`
		} `json:"source"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	id, err := uuid.Parse(resp.Source.ID)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func syncSource(t *testing.T, h *Handler, sourceID, userID uuid.UUID) BrunoSyncResult {
	t.Helper()
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/bruno-sources/"+sourceID.String()+"/sync", nil)
	routeContext := chi.NewRouteContext()
	routeContext.URLParams.Add("id", sourceID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeContext))
	req = req.WithContext(context.WithValue(req.Context(), auth.UserIDKey, userID.String()))
	h.SyncBrunoSource(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("sync status %d: %s", rr.Code, rr.Body.String())
	}
	var result BrunoSyncResult
	if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	return result
}

func seedViewer(t *testing.T, ctx context.Context, pool *pgxpool.Pool, wsID uuid.UUID) uuid.UUID {
	t.Helper()
	store := appdb.NewStore(pool)
	userID, err := store.CreateUser(ctx, sqlc.CreateUserParams{
		Email:        "viewer-" + uuid.NewString()[:8] + "@example.com",
		PasswordHash: "hash",
		DisplayName:  "Viewer",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.UpsertWorkspaceMember(ctx, sqlc.UpsertWorkspaceMemberParams{
		WorkspaceID: appdb.PGUUID(wsID),
		UserID:      userID,
		Role:        "viewer",
	}); err != nil {
		t.Fatal(err)
	}
	return appdb.FromPGUUID(userID)
}

func TestImportForbiddenForNonMember(t *testing.T) {
	ctx := context.Background()
	pool := requireIntegrationDB(t)
	_, wsID := testutil.SeedWorkspace(t, ctx, pool)
	outsiderID := seedViewer(t, ctx, pool, wsID)
	store := appdb.NewStore(pool)
	_ = store.DeleteWorkspaceMember(ctx, sqlc.DeleteWorkspaceMemberParams{
		WorkspaceID: appdb.PGUUID(wsID),
		UserID:      appdb.PGUUID(outsiderID),
	})
	h := NewHandler(store, nil)
	data := []byte(`{"info":{"name":"x","schema":"https://schema.getpostman.com/json/collection/v2.1.0/collection.json"},"item":[{"name":"r","request":{"method":"GET","url":"https://example.com"}}]}`)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/import/postman?workspace_id="+wsID.String(), bytes.NewReader(data))
	req = req.WithContext(context.WithValue(req.Context(), auth.UserIDKey, outsiderID.String()))
	h.ImportPostman(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status %d: %s", rr.Code, rr.Body.String())
	}
}
