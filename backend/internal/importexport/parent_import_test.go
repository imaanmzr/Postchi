package importexport

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"

	appdb "github.com/imaanmzr/postchi/backend/internal/db"
	"github.com/imaanmzr/postchi/backend/internal/testutil"
)

func TestResolveImportParentCreate(t *testing.T) {
	ctx := context.Background()
	pool := requireIntegrationDB(t)
	userID, wsID := testutil.SeedWorkspace(t, ctx, pool)
	store := appdb.NewStore(pool)
	h := NewHandler(store, nil)

	parentID, err := h.resolveImportParent(ctx, wsID, userID, importParentRequest{
		CreateParent: &struct {
			Name string `json:"name"`
		}{Name: "Git Imports"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if parentID == nil {
		t.Fatal("expected parent collection id")
	}

	cols, err := store.ListCollectionsByWorkspace(ctx, appdb.PGUUID(wsID))
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, col := range cols {
		if appdb.FromPGUUID(col.ID) == *parentID && col.Name == "Git Imports" && !col.ParentID.Valid {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("created parent collection not found: %+v", cols)
	}
}

func TestResolveImportParentExisting(t *testing.T) {
	ctx := context.Background()
	pool := requireIntegrationDB(t)
	userID, wsID := testutil.SeedWorkspace(t, ctx, pool)
	store := appdb.NewStore(pool)
	h := NewHandler(store, nil)

	existing, err := h.createImportParentCollection(ctx, wsID, userID, "Existing Parent", nil)
	if err != nil {
		t.Fatal(err)
	}
	id := existing.String()
	parentID, err := h.resolveImportParent(ctx, wsID, userID, importParentRequest{
		ParentID: &id,
	})
	if err != nil {
		t.Fatal(err)
	}
	if parentID == nil || *parentID != *existing {
		t.Fatalf("parent = %v, want %v", parentID, existing)
	}
}

func TestResolveImportParentConflict(t *testing.T) {
	ctx := context.Background()
	pool := requireIntegrationDB(t)
	userID, wsID := testutil.SeedWorkspace(t, ctx, pool)
	h := NewHandler(appdb.NewStore(pool), nil)

	id := uuid.New().String()
	_, err := h.resolveImportParent(ctx, wsID, userID, importParentRequest{
		ParentID: &id,
		CreateParent: &struct {
			Name string `json:"name"`
		}{Name: "Nope"},
	})
	if err == nil {
		t.Fatal("expected conflict error")
	}
}

func TestParentIDFromBrunoConfig(t *testing.T) {
	id := uuid.New()
	config := map[string]any{"parent_collection_id": id.String()}
	got := parentIDFromBrunoConfig(config)
	if got == nil || *got != id {
		t.Fatalf("got %v want %v", got, id)
	}
	setBrunoConfigParentID(config, nil)
	if parentIDFromBrunoConfig(config) != nil {
		t.Fatal("expected nil after clear")
	}
	raw, _ := json.Marshal(config)
	if string(raw) != "{}" {
		t.Fatalf("config = %s", raw)
	}
}
