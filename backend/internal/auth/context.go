package auth

import (
	"context"

	"github.com/google/uuid"
)

type wsContextKey struct{}

type WorkspaceContext struct {
	ID   uuid.UUID
	Role string
}

func WithWorkspaceContext(ctx context.Context, id uuid.UUID, role string) context.Context {
	return context.WithValue(ctx, wsContextKey{}, WorkspaceContext{ID: id, Role: role})
}

func WorkspaceFromContext(ctx context.Context) (WorkspaceContext, bool) {
	ws, ok := ctx.Value(wsContextKey{}).(WorkspaceContext)
	return ws, ok
}
