package collection

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

	appdb "github.com/imaanmzr/postchi/backend/internal/db"
	"github.com/imaanmzr/postchi/backend/internal/shared/db"
	"github.com/imaanmzr/postchi/backend/internal/testutil"
)

func TestUpdateVariablesPreservesNameAndParent(t *testing.T) {
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
	store := appdb.NewStore(pool)
	h := NewHandler(store)

	parentID := testutil.SeedCollection(t, ctx, pool, wsID, userID, "Identity", nil)
	colID := testutil.SeedCollection(t, ctx, pool, wsID, userID, "AldyPay API Collection", &parentID)

	body := `{"variables":{"pre_request":[{"enabled":true,"name":"uatBaseUrl","value":"https://gateway.uat.example.com","type":"string"}],"post_response":[]}}`
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPatch, "/api/collections/"+colID.String(), strings.NewReader(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", colID.String())
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	h.Update(rr, r)
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rr.Code, rr.Body.String())
	}

	var got Collection
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Name != "AldyPay API Collection" {
		t.Fatalf("response name = %q", got.Name)
	}
	if got.ParentID == nil || *got.ParentID != parentID.String() {
		t.Fatalf("response parent_id = %v, want %s", got.ParentID, parentID)
	}
	if len(got.Variables.PreRequest) != 1 || got.Variables.PreRequest[0].Name != "uatBaseUrl" {
		t.Fatalf("response variables = %+v", got.Variables)
	}

	row, err := store.GetCollection(ctx, appdb.PGUUID(colID))
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if row.Name != "AldyPay API Collection" {
		t.Fatalf("db name = %q", row.Name)
	}
	if !row.ParentID.Valid || appdb.FromPGUUID(row.ParentID) != parentID {
		t.Fatalf("db parent_id was cleared")
	}
}
