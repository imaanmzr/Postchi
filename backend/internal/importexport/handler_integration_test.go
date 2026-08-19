package importexport

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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
	"github.com/imaanmzr/postchi/backend/internal/db/sqlc"
	"github.com/imaanmzr/postchi/backend/internal/shared/db"
	"github.com/imaanmzr/postchi/backend/internal/testutil"
)

func TestHandlerIntegration(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL not set")
	}

	ctx := context.Background()
	if err := db.RunMigrations(databaseURL, "file://"+filepath.Join("..", "..", "..", "migrations")); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	defer pool.Close()

	userID, wsID := seedWorkspace(t, ctx, pool)
	store := appdb.NewStore(pool)
	h := NewHandler(store)
	tokens := auth.NewService("test-secret-key-32-chars-minimum!", "postchi", 0, 0)

	t.Run("postman nested import counts", func(t *testing.T) {
		data, _ := os.ReadFile(filepath.Join("testdata", "postman", "nested.json"))
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/import/postman?workspace_id="+wsID.String(), bytes.NewReader(data))
		req = req.WithContext(context.WithValue(req.Context(), auth.UserIDKey, userID.String()))
		h.ImportPostman(rr, req)
		if rr.Code != http.StatusCreated {
			t.Fatalf("status %d: %s", rr.Code, rr.Body.String())
		}
		var result ImportResult
		_ = json.Unmarshal(rr.Body.Bytes(), &result)
		if result.Requests < 1 || result.Collections < 1 {
			t.Fatalf("expected collections and requests, got %+v", result)
		}
	})

	t.Run("curl import creates request", func(t *testing.T) {
		colID, err := store.CreateCollection(ctx, sqlc.CreateCollectionParams{
			WorkspaceID:        appdb.PGUUID(wsID),
			Name:               "curl-col",
			Variables:          []byte(`{"pre_request":[],"post_response":[]}`),
			Headers:            []byte(`[]`),
			Auth:               []byte(`{}`),
			Presets:            []byte(`[]`),
			Proxy:              []byte(`{}`),
			ClientCertificates: []byte(`[]`),
			Secrets:            []byte(`[]`),
			CreatedBy:          appdb.PGUUID(userID),
		})
		if err != nil {
			t.Fatalf("collection: %v", err)
		}
		body := `{"command":"curl -X GET https://example.com","collection_id":"` + appdb.FromPGUUID(colID).String() + `","name":"curl req"}`
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/import/curl", strings.NewReader(body))
		req = req.WithContext(context.WithValue(req.Context(), auth.UserIDKey, userID.String()))
		h.ImportCurl(rr, req)
		if rr.Code != http.StatusCreated {
			t.Fatalf("status %d: %s", rr.Code, rr.Body.String())
		}
		reqs, _ := store.ListRequestIDsByCollection(ctx, colID)
		if len(reqs) != 1 {
			t.Fatalf("expected 1 request, got %d", len(reqs))
		}
	})

	t.Run("postman showcase import", func(t *testing.T) {
		data, err := os.ReadFile(filepath.Join("..", "..", "..", "docs", "postchi-showcase.postman.json"))
		if err != nil {
			t.Fatal(err)
		}
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/import/postman?workspace_id="+wsID.String(), bytes.NewReader(data))
		req = req.WithContext(context.WithValue(req.Context(), auth.UserIDKey, userID.String()))
		h.ImportPostman(rr, req)
		if rr.Code != http.StatusCreated {
			t.Fatalf("status %d: %s", rr.Code, rr.Body.String())
		}
		var result ImportResult
		_ = json.Unmarshal(rr.Body.Bytes(), &result)
		if result.Requests != 24 {
			t.Fatalf("expected 24 requests, got %+v", result)
		}
		if result.Collections != 10 {
			t.Fatalf("expected 10 collections, got %+v", result)
		}
	})

	t.Run("empty postman collection fails", func(t *testing.T) {
		empty := []byte(`{"info":{"name":"Empty","schema":"https://schema.getpostman.com/json/collection/v2.1.0/collection.json"},"item":[]}`)
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/import/postman?workspace_id="+wsID.String(), bytes.NewReader(empty))
		req = req.WithContext(context.WithValue(req.Context(), auth.UserIDKey, userID.String()))
		h.ImportPostman(rr, req)
		if rr.Code == http.StatusCreated {
			t.Fatalf("expected failure for empty import, got 201")
		}
		_ = tokens
	})

	t.Run("nested Git Bruno repository imports transactionally", func(t *testing.T) {
		gitServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Private-Token") != "gitlab-token" {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			escapedPath := r.URL.EscapedPath()
			switch {
			case strings.Contains(r.URL.Path, "/repository/tree"):
				_, _ = w.Write([]byte(`[
					{"path":"collection.bru","type":"blob"},
					{"path":"Orders/folder.bru","type":"blob"},
					{"path":"Orders/list.bru","type":"blob"},
					{"path":"health.bru","type":"blob"},
					{"path":"environments/local.bru","type":"blob"},
					{"path":"README.md","type":"blob"}
				]`))
			case strings.Contains(escapedPath, "collection%252Ebru/raw") || strings.Contains(escapedPath, "collection%2Ebru/raw"):
				_, _ = w.Write([]byte("meta {\n  name: Repository API\n  type: collection\n}\n"))
			case strings.Contains(escapedPath, "Orders%252Ffolder%252Ebru/raw") || strings.Contains(escapedPath, "Orders%2Ffolder%2Ebru/raw"):
				_, _ = w.Write([]byte("meta {\n  name: Orders\n  seq: 1\n}\n"))
			case strings.Contains(escapedPath, "Orders%252Flist%252Ebru/raw") || strings.Contains(escapedPath, "Orders%2Flist%2Ebru/raw"):
				_, _ = w.Write([]byte("meta {\n  name: List orders\n  type: http\n  seq: 1\n}\n\nget {\n  url: https://example.com/orders\n}\n"))
			case strings.Contains(escapedPath, "health%252Ebru/raw") || strings.Contains(escapedPath, "health%2Ebru/raw"):
				_, _ = w.Write([]byte("meta {\n  name: Health\n  type: http\n  seq: 1\n}\n\nget {\n  url: https://example.com/health\n}\n"))
			default:
				http.NotFound(w, r)
			}
		}))
		defer gitServer.Close()

		body := fmt.Sprintf(`{
			"name":"Imported Git API",
			"repo_url":%q,
			"branch":"main",
			"path_prefix":"",
			"access_token":"gitlab-token"
		}`, gitServer.URL+"/group/repository")
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/workspaces/"+wsID.String()+"/imports/bruno/git", strings.NewReader(body))
		routeContext := chi.NewRouteContext()
		routeContext.URLParams.Add("id", wsID.String())
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeContext))
		req = req.WithContext(context.WithValue(req.Context(), auth.UserIDKey, userID.String()))
		h.ImportBrunoGit(rr, req)
		if rr.Code != http.StatusCreated {
			t.Fatalf("status %d: %s", rr.Code, rr.Body.String())
		}
		var result ImportResult
		if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		if result.Collections != 2 || result.Requests != 2 || result.CollectionID == "" {
			t.Fatalf("unexpected result: %+v", result)
		}

		collections, err := store.ListCollectionsByWorkspace(ctx, appdb.PGUUID(wsID))
		if err != nil {
			t.Fatal(err)
		}
		var rootFound, childFound bool
		for _, collection := range collections {
			if collection.Name == "Imported Git API" {
				rootFound = true
			}
			if collection.Name == "Orders" && collection.ParentID.Valid {
				childFound = true
			}
		}
		if !rootFound || !childFound {
			t.Fatalf("expected root and nested collection, got %+v", collections)
		}
	})

	t.Run("malformed Git repository leaves no partial collection", func(t *testing.T) {
		before, err := store.ListCollectionsByWorkspace(ctx, appdb.PGUUID(wsID))
		if err != nil {
			t.Fatal(err)
		}
		gitServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case strings.Contains(r.URL.Path, "/repository/tree"):
				_, _ = w.Write([]byte(`[
					{"path":"collection.bru","type":"blob"},
					{"path":"broken.bru","type":"blob"}
				]`))
			case strings.Contains(r.URL.EscapedPath(), "collection%2Ebru/raw") || strings.Contains(r.URL.EscapedPath(), "collection%252Ebru/raw"):
				_, _ = w.Write([]byte("meta {\n  name: API\n  type: collection\n}\n"))
			case strings.Contains(r.URL.EscapedPath(), "broken%2Ebru/raw") || strings.Contains(r.URL.EscapedPath(), "broken%252Ebru/raw"):
				_, _ = w.Write([]byte("meta {\n  name: Broken\n  type: http\n}\n"))
			default:
				http.NotFound(w, r)
			}
		}))
		defer gitServer.Close()

		body := fmt.Sprintf(`{"name":"Must not persist","repo_url":%q,"access_token":"token"}`,
			gitServer.URL+"/group/repository")
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/workspaces/"+wsID.String()+"/imports/bruno/git", strings.NewReader(body))
		routeContext := chi.NewRouteContext()
		routeContext.URLParams.Add("id", wsID.String())
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeContext))
		req = req.WithContext(context.WithValue(req.Context(), auth.UserIDKey, userID.String()))
		h.ImportBrunoGit(rr, req)
		if rr.Code != http.StatusUnprocessableEntity {
			t.Fatalf("status %d: %s", rr.Code, rr.Body.String())
		}
		after, err := store.ListCollectionsByWorkspace(ctx, appdb.PGUUID(wsID))
		if err != nil {
			t.Fatal(err)
		}
		if len(after) != len(before) {
			t.Fatalf("malformed import persisted collections: before=%d after=%d", len(before), len(after))
		}
	})
}

func seedWorkspace(t *testing.T, ctx context.Context, pool *pgxpool.Pool) (uuid.UUID, uuid.UUID) {
	t.Helper()
	return testutil.SeedWorkspace(t, ctx, pool)
}
