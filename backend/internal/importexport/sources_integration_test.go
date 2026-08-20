package importexport

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/imaanmzr/postchi/backend/internal/auth"
	appdb "github.com/imaanmzr/postchi/backend/internal/db"
	"github.com/imaanmzr/postchi/backend/internal/shared/crypto"
	"github.com/imaanmzr/postchi/backend/internal/testutil"
)

func TestBrunoSourceCRUDAndExport(t *testing.T) {
	ctx := context.Background()
	pool := requireIntegrationDB(t)
	userID, wsID := testutil.SeedWorkspace(t, ctx, pool)
	store := appdb.NewStore(pool)
	cryptoSvc, _ := crypto.NewService("0123456789abcdef0123456789abcdef")
	h := NewHandler(store, cryptoSvc)

	postman := []byte(`{"info":{"name":"Export API","schema":"https://schema.getpostman.com/json/collection/v2.1.0/collection.json"},"item":[{"name":"Ping","request":{"method":"GET","url":"https://example.com/ping"}}]}`)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/import/postman?workspace_id="+wsID.String(), bytes.NewReader(postman))
	req = req.WithContext(context.WithValue(req.Context(), auth.UserIDKey, userID.String()))
	h.ImportPostman(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("import status %d: %s", rr.Code, rr.Body.String())
	}
	var imported ImportResult
	_ = json.Unmarshal(rr.Body.Bytes(), &imported)
	colID := imported.CollectionID

	t.Run("export postman", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/export/postman?collection_id="+colID, nil)
		h.ExportPostman(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("export bruno zip", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/export/bruno?collection_id="+colID, nil)
		h.ExportBruno(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status %d: %s", rr.Code, rr.Body.String())
		}
		if ct := rr.Header().Get("Content-Type"); !strings.Contains(ct, "zip") {
			t.Fatalf("content type=%q", ct)
		}
	})

	gitServer := newGitLabFileServer(t, map[string]string{
		"collection.bru": "meta {\n  name: CRUD API\n  type: collection\n}\n",
		"ping.bru":       "meta {\n  name: Ping\n  type: http\n}\nget {\n  url: https://example.com/ping\n}\n",
	})
	defer gitServer.Close()
	sourceID := createGitSource(t, ctx, h, wsID, userID, gitServer.URL+"/group/repository", "CRUD API")

	t.Run("update source", func(t *testing.T) {
		body := `{"name":"Renamed API","config":{"repo_url":"` + gitServer.URL + `/group/repository","branch":"main"}}`
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPatch, "/api/workspaces/"+wsID.String()+"/bruno-sources/"+sourceID.String(), strings.NewReader(body))
		routeContext := chi.NewRouteContext()
		routeContext.URLParams.Add("id", wsID.String())
		routeContext.URLParams.Add("sourceId", sourceID.String())
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeContext))
		h.UpdateBrunoSource(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("sync collection source alias", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/collection-sources/"+sourceID.String()+"/sync", nil)
		routeContext := chi.NewRouteContext()
		routeContext.URLParams.Add("id", sourceID.String())
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeContext))
		req = req.WithContext(context.WithValue(req.Context(), auth.UserIDKey, userID.String()))
		h.SyncCollectionSource(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("delete source", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodDelete, "/api/workspaces/"+wsID.String()+"/bruno-sources/"+sourceID.String(), nil)
		routeContext := chi.NewRouteContext()
		routeContext.URLParams.Add("id", wsID.String())
		routeContext.URLParams.Add("sourceId", sourceID.String())
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeContext))
		h.DeleteBrunoSource(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status %d: %s", rr.Code, rr.Body.String())
		}
	})
}

func TestImportCollectionGitAlias(t *testing.T) {
	ctx := context.Background()
	pool := requireIntegrationDB(t)
	userID, wsID := testutil.SeedWorkspace(t, ctx, pool)
	store := appdb.NewStore(pool)
	h := NewHandler(store, nil)

	gitServer := newGitLabFileServer(t, map[string]string{
		"collection.bru": "meta {\n  name: Alias API\n  type: collection\n}\n",
		"health.bru":     "meta {\n  name: Health\n  type: http\n}\nget {\n  url: https://example.com/health\n}\n",
	})
	defer gitServer.Close()

	body := fmt.Sprintf(`{"name":"Alias Import","repo_url":%q,"access_token":"gitlab-token"}`, gitServer.URL+"/group/repository")
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/workspaces/"+wsID.String()+"/imports/git", strings.NewReader(body))
	routeContext := chi.NewRouteContext()
	routeContext.URLParams.Add("id", wsID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeContext))
	req = req.WithContext(context.WithValue(req.Context(), auth.UserIDKey, userID.String()))
	h.ImportCollectionGit(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("status %d: %s", rr.Code, rr.Body.String())
	}
}

func TestImportCollectionGitWithParent(t *testing.T) {
	ctx := context.Background()
	pool := requireIntegrationDB(t)
	userID, wsID := testutil.SeedWorkspace(t, ctx, pool)
	store := appdb.NewStore(pool)
	h := NewHandler(store, nil)

	parentID, err := h.createImportParentCollection(ctx, wsID, userID, "Import Target", nil)
	if err != nil {
		t.Fatal(err)
	}

	gitServer := newGitLabFileServer(t, map[string]string{
		"collection.bru": "meta {\n  name: Nested API\n  type: collection\n}\n",
		"ping.bru":       "meta {\n  name: Ping\n  type: http\n}\nget {\n  url: https://example.com/ping\n}\n",
	})
	defer gitServer.Close()

	body := fmt.Sprintf(
		`{"name":"Nested Import","repo_url":%q,"access_token":"gitlab-token","parent_id":%q}`,
		gitServer.URL+"/group/repository",
		parentID.String(),
	)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/workspaces/"+wsID.String()+"/imports/git", strings.NewReader(body))
	routeContext := chi.NewRouteContext()
	routeContext.URLParams.Add("id", wsID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeContext))
	req = req.WithContext(context.WithValue(req.Context(), auth.UserIDKey, userID.String()))
	h.ImportCollectionGit(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("status %d: %s", rr.Code, rr.Body.String())
	}
	var imported ImportResult
	_ = json.Unmarshal(rr.Body.Bytes(), &imported)
	if imported.CollectionID == "" {
		t.Fatal("missing collection id")
	}
	colUUID, _ := uuid.Parse(imported.CollectionID)
	row, err := store.GetCollection(ctx, appdb.PGUUID(colUUID))
	if err != nil {
		t.Fatal(err)
	}
	if !row.ParentID.Valid || appdb.FromPGUUID(row.ParentID) != *parentID {
		t.Fatalf("imported collection parent = %v, want %v", row.ParentID, parentID)
	}
}

func TestListBrunoSourcesInvalidWorkspace(t *testing.T) {
	h := NewHandler(nil, nil)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/workspaces/not-a-uuid/bruno-sources", nil)
	routeContext := chi.NewRouteContext()
	routeContext.URLParams.Add("id", "not-a-uuid")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeContext))
	h.ListBrunoSources(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status %d", rr.Code)
	}
}

func TestUpdateBrunoSourceValidation(t *testing.T) {
	h := NewHandler(nil, nil)
	wsID := uuid.New()
	sourceID := uuid.New()
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/workspaces/"+wsID.String()+"/bruno-sources/"+sourceID.String(), strings.NewReader(`{}`))
	routeContext := chi.NewRouteContext()
	routeContext.URLParams.Add("id", wsID.String())
	routeContext.URLParams.Add("sourceId", sourceID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeContext))
	h.UpdateBrunoSource(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status %d: %s", rr.Code, rr.Body.String())
	}
}
