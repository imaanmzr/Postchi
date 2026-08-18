package testutil

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/imaanmzr/postchi/backend/internal/db"
	"github.com/imaanmzr/postchi/backend/internal/db/sqlc"
)

func SeedWorkspace(t *testing.T, ctx context.Context, pool *pgxpool.Pool) (uuid.UUID, uuid.UUID) {
	t.Helper()
	store := db.NewStore(pool)
	email := "test-" + uuid.New().String()[:8] + "@example.com"
	userID, err := store.CreateUser(ctx, sqlc.CreateUserParams{
		Email:        email,
		PasswordHash: "hash",
		DisplayName:  email,
	})
	if err != nil {
		t.Fatalf("user: %v", err)
	}
	wsID, err := store.CreateWorkspace(ctx, sqlc.CreateWorkspaceParams{
		Name:        "Test Workspace",
		Description: "",
		CreatedBy:   userID,
	})
	if err != nil {
		t.Fatalf("workspace: %v", err)
	}
	if err := store.AddWorkspaceOwner(ctx, sqlc.AddWorkspaceOwnerParams{
		WorkspaceID: wsID,
		UserID:      userID,
	}); err != nil {
		t.Fatalf("member: %v", err)
	}
	return db.FromPGUUID(userID), db.FromPGUUID(wsID)
}

func SeedCollection(t *testing.T, ctx context.Context, pool *pgxpool.Pool, wsID, userID uuid.UUID, name string, parentID *uuid.UUID) uuid.UUID {
	t.Helper()
	store := db.NewStore(pool)
	colID, err := store.CreateCollection(ctx, sqlc.CreateCollectionParams{
		WorkspaceID: db.PGUUID(wsID),
		ParentID:    db.PGUUIDPtr(parentID),
		Name:        name,
		CreatedBy:   db.PGUUID(userID),
	})
	if err != nil {
		t.Fatalf("collection: %v", err)
	}
	return db.FromPGUUID(colID)
}

func SeedRequest(t *testing.T, ctx context.Context, pool *pgxpool.Pool, colID, userID uuid.UUID, name string) uuid.UUID {
	t.Helper()
	store := db.NewStore(pool)
	reqID, err := store.CreateRequest(ctx, sqlc.CreateRequestParams{
		CollectionID: db.PGUUID(colID),
		Name:         name,
		Method:       "GET",
		Url:          "https://example.com",
		CreatedBy:    db.PGUUID(userID),
	})
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	return db.FromPGUUID(reqID)
}
