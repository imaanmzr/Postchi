package importexport

import (
	"bytes"
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
}

func seedWorkspace(t *testing.T, ctx context.Context, pool *pgxpool.Pool) (uuid.UUID, uuid.UUID) {
	t.Helper()
	return testutil.SeedWorkspace(t, ctx, pool)
}
