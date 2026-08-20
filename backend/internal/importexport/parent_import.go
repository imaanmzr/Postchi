package importexport

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/imaanmzr/postchi/backend/internal/db"
	"github.com/imaanmzr/postchi/backend/internal/db/sqlc"
)

type importParentRequest struct {
	ParentID      *string `json:"parent_id"`
	CreateParent  *struct {
		Name string `json:"name"`
	} `json:"create_parent"`
}

func (h *Handler) resolveImportParent(ctx context.Context, wsID, userID uuid.UUID, req importParentRequest) (*uuid.UUID, error) {
	createName := ""
	if req.CreateParent != nil {
		createName = strings.TrimSpace(req.CreateParent.Name)
	}
	parentIDStr := ""
	if req.ParentID != nil {
		parentIDStr = strings.TrimSpace(*req.ParentID)
	}

	if createName != "" && parentIDStr != "" {
		return nil, fmt.Errorf("provide either parent_id or create_parent, not both")
	}
	if createName != "" {
		return h.createImportParentCollection(ctx, wsID, userID, createName, nil)
	}
	if parentIDStr == "" {
		return nil, nil
	}
	parentID, err := uuid.Parse(parentIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid parent_id")
	}
	if err := h.validateCollectionInWorkspace(ctx, parentID, userID); err != nil {
		return nil, err
	}
	return &parentID, nil
}

func (h *Handler) createImportParentCollection(ctx context.Context, wsID, userID uuid.UUID, name string, parentID *uuid.UUID) (*uuid.UUID, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("parent collection name is required")
	}
	empty := json.RawMessage(`{}`)
	id, err := h.store.CreateCollection(ctx, sqlc.CreateCollectionParams{
		WorkspaceID:        db.PGUUID(wsID),
		ParentID:           db.PGUUIDPtr(parentID),
		Name:               name,
		Description:        "",
		SortOrder:          0,
		Variables:          empty,
		Headers:            empty,
		Auth:               empty,
		Presets:            empty,
		Proxy:              empty,
		ClientCertificates: empty,
		Secrets:            empty,
		CreatedBy:          db.PGUUID(userID),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create parent collection")
	}
	colID := db.FromPGUUID(id)
	return &colID, nil
}

func parentIDFromBrunoConfig(config map[string]any) *uuid.UUID {
	raw, _ := config["parent_collection_id"].(string)
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return nil
	}
	return &id
}

func setBrunoConfigParentID(config map[string]any, parentID *uuid.UUID) {
	if config == nil {
		return
	}
	if parentID == nil {
		delete(config, "parent_collection_id")
		return
	}
	config["parent_collection_id"] = parentID.String()
}
