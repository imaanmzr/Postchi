package apispec

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/imaanmzr/postchi/backend/internal/auth"
	"github.com/imaanmzr/postchi/backend/internal/db"
	"github.com/imaanmzr/postchi/backend/internal/db/sqlc"
	openapiimport "github.com/imaanmzr/postchi/backend/internal/importexport/openapi"
	"github.com/imaanmzr/postchi/backend/internal/shared/config"
	"github.com/imaanmzr/postchi/backend/internal/testutil"
)

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL not set")
	}
	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	return pool
}

func TestInsertSyncedRequestIdentitySpec(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	userID, wsID := testutil.SeedWorkspace(t, ctx, pool)

	data, err := os.ReadFile("/tmp/swagger.json")
	if err != nil {
		t.Skip("download swagger to /tmp/swagger.json first")
	}

	parsed, err := openapiimport.ParseWithHash(data, "Identity")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(parsed.Operations) == 0 {
		t.Fatal("expected operations")
	}

	colID := testutil.SeedCollection(t, ctx, pool, wsID, userID, "identity-insert-test", nil)
	store := db.NewStore(pool)
	specID, err := store.CreateApiSpec(ctx, sqlc.CreateApiSpecParams{
		WorkspaceID:  db.PGUUID(wsID),
		CollectionID: db.PGUUID(colID),
		Name:         "identity-insert-test",
		SpecUrl:      "https://example.com/swagger.json",
		SpecHash:     "hash",
		BaseUrlVar:   "baseUrl",
		CreatedBy:    db.PGUUID(userID),
	})
	if err != nil {
		t.Fatalf("spec: %v", err)
	}

	tx, q, err := store.Begin(ctx)
	if err != nil {
		t.Fatalf("tx: %v", err)
	}
	defer tx.Rollback(ctx)

	op := parsed.Operations[0]
	if err := insertSyncedRequest(ctx, q, colID, db.FromPGUUID(specID), op, userID); err != nil {
		t.Fatalf("insertSyncedRequest: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit: %v", err)
	}
}

func TestSyncApplyInsertsRequests(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	userID, wsID := testutil.SeedWorkspace(t, ctx, pool)

	h := NewHandler(db.NewStore(pool), &config.Config{})

	if _, err := os.ReadFile("/tmp/swagger.json"); err != nil {
		t.Skip("download swagger to /tmp/swagger.json first")
	}

	colID := testutil.SeedCollection(t, ctx, pool, wsID, userID, "identity-test", nil)
	store := db.NewStore(pool)
	specID, err := store.CreateApiSpec(ctx, sqlc.CreateApiSpecParams{
		WorkspaceID:  db.PGUUID(wsID),
		CollectionID: db.PGUUID(colID),
		Name:         "identity-test",
		SpecUrl:      "https://api.example.com/swagger/v1/swagger.json",
		SpecHash:     "test-hash",
		BaseUrlVar:   "baseUrl",
		CreatedBy:    db.PGUUID(userID),
	})
	if err != nil {
		t.Fatalf("spec: %v", err)
	}

	syncBody, _ := json.Marshal(map[string]bool{"apply": true})
	req := httptest.NewRequest(http.MethodPost, "/api/api-specs/"+db.FromPGUUID(specID).String()+"/sync", bytes.NewReader(syncBody))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", db.FromPGUUID(specID).String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(context.WithValue(req.Context(), auth.UserIDKey, userID.String()))

	rr := httptest.NewRecorder()
	h.Sync(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("sync status %d body %s", rr.Code, rr.Body.String())
	}
	var diff SyncDiff
	if err := json.NewDecoder(rr.Body).Decode(&diff); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(diff.Added) == 0 {
		t.Fatalf("expected added operations, got diff %+v", diff)
	}
	synced, err := store.ListSyncedRequestsBySpec(ctx, specID)
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	count := len(synced)
	if count != len(diff.Added) {
		t.Fatalf("expected %d requests, got %d", len(diff.Added), count)
	}
}

func TestCreateImportsEndpoints(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	userID, wsID := testutil.SeedWorkspace(t, ctx, pool)

	if _, err := os.ReadFile("/tmp/swagger.json"); err != nil {
		t.Skip("download swagger to /tmp/swagger.json first")
	}

	h := NewHandler(db.NewStore(pool), &config.Config{})
	store := db.NewStore(pool)

	createBody, _ := json.Marshal(map[string]string{
		"name":     "identity-create",
		"spec_url": "https://api.example.com/swagger/v1/swagger.json",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/workspaces/"+wsID.String()+"/api-specs", bytes.NewReader(createBody))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", wsID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(context.WithValue(req.Context(), auth.UserIDKey, userID.String()))

	rr := httptest.NewRecorder()
	h.Create(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create status %d body %s", rr.Code, rr.Body.String())
	}
	var spec ApiSpec
	if err := json.NewDecoder(rr.Body).Decode(&spec); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if spec.LastSyncedAt == nil || *spec.LastSyncedAt == "" {
		t.Fatal("expected last_synced_at after create")
	}
	specID, err := uuid.Parse(spec.ID)
	if err != nil {
		t.Fatalf("spec id: %v", err)
	}
	synced, err := store.ListSyncedRequestsBySpec(ctx, db.PGUUID(specID))
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	count := len(synced)
	if count == 0 {
		t.Fatal("expected imported requests after create")
	}
}
