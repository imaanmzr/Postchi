package importexport

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"github.com/imaanmzr/postchi/backend/internal/db"
	"github.com/imaanmzr/postchi/backend/internal/db/sqlc"
	"github.com/imaanmzr/postchi/backend/internal/importexport/model"
	"github.com/imaanmzr/postchi/backend/internal/shared/operationid"
)

func (h *Handler) persistCollectionTx(ctx context.Context, q *sqlc.Queries, wsID, userID uuid.UUID, col model.Collection, parentID *uuid.UUID) (uuid.UUID, ImportResult, error) {
	vars, err := json.Marshal(col.Variables)
	if err != nil {
		return uuid.Nil, ImportResult{}, err
	}
	headers, err := json.Marshal(col.Headers)
	if err != nil {
		return uuid.Nil, ImportResult{}, err
	}
	authB, err := json.Marshal(col.Auth)
	if err != nil {
		return uuid.Nil, ImportResult{}, err
	}

	id, err := q.InsertImportedCollection(ctx, sqlc.InsertImportedCollectionParams{
		WorkspaceID:      db.PGUUID(wsID),
		ParentID:         db.PGUUIDPtr(parentID),
		Name:             col.Name,
		Description:      col.Description,
		SortOrder:        int32(col.SortOrder),
		Variables:        vars,
		Headers:          headers,
		Auth:             authB,
		PreRequestScript: col.PreRequestScript,
		TestScript:       col.TestScript,
		CreatedBy:        db.PGUUID(userID),
	})
	if err != nil {
		return uuid.Nil, ImportResult{}, fmt.Errorf("insert collection: %w", err)
	}
	colID := db.FromPGUUID(id)

	result := ImportResult{Collections: 1}
	for _, req := range col.Requests {
		if err := h.insertRequestTx(ctx, q, colID, userID, req); err != nil {
			return uuid.Nil, ImportResult{}, err
		}
		result.Requests++
	}
	for _, child := range col.Children {
		cid := colID
		_, childResult, err := h.persistCollectionTx(ctx, q, wsID, userID, child, &cid)
		if err != nil {
			return uuid.Nil, ImportResult{}, err
		}
		result.Collections += childResult.Collections
		result.Requests += childResult.Requests
	}
	return colID, result, nil
}

func (h *Handler) insertRequestTx(ctx context.Context, q *sqlc.Queries, colID, userID uuid.UUID, req model.Request) error {
	headers, err := json.Marshal(req.Headers)
	if err != nil {
		return fmt.Errorf("marshal headers for %q: %w", req.Name, err)
	}
	params, err := json.Marshal(req.Params)
	if err != nil {
		return fmt.Errorf("marshal params for %q: %w", req.Name, err)
	}
	pathVars, err := json.Marshal(req.PathVars)
	if err != nil {
		return fmt.Errorf("marshal path vars for %q: %w", req.Name, err)
	}
	body, err := json.Marshal(req.Body)
	if err != nil {
		return fmt.Errorf("marshal body for %q: %w", req.Name, err)
	}
	authB, err := json.Marshal(req.Auth)
	if err != nil {
		return fmt.Errorf("marshal auth for %q: %w", req.Name, err)
	}
	settings, err := json.Marshal(req.Settings)
	if err != nil {
		return fmt.Errorf("marshal settings for %q: %w", req.Name, err)
	}
	err = q.InsertImportedRequest(ctx, sqlc.InsertImportedRequestParams{
		CollectionID:        db.PGUUID(colID),
		Name:                req.Name,
		Method:              req.Method,
		Url:                 req.URL,
		Headers:             headers,
		Params:              params,
		PathVars:            pathVars,
		Body:                body,
		Auth:                authB,
		Settings:            settings,
		PreRequestScript:    req.PreRequestScript,
		TestScript:          req.TestScript,
		SortOrder:           int32(req.SortOrder),
		Description:         req.Description,
		SourceOperationID:   operationid.CanonicalFromMethodURL(req.Method, req.URL),
		CreatedBy:           db.PGUUID(userID),
	})
	if err != nil {
		return fmt.Errorf("insert request %q: %w", req.Name, err)
	}
	return nil
}

func (h *Handler) persistCollection(ctx context.Context, wsID, userID uuid.UUID, col model.Collection, parentID *uuid.UUID) (uuid.UUID, ImportResult, error) {
	var id uuid.UUID
	var result ImportResult
	err := h.store.WithTx(ctx, func(q *sqlc.Queries) error {
		var err error
		id, result, err = h.persistCollectionTx(ctx, q, wsID, userID, col, parentID)
		return err
	})
	if err != nil {
		return uuid.Nil, ImportResult{}, err
	}
	result.CollectionID = id.String()
	return id, result, nil
}

func (h *Handler) validateWorkspace(ctx context.Context, wsID uuid.UUID) error {
	exists, err := h.store.WorkspaceExists(ctx, db.PGUUID(wsID))
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("workspace not found")
	}
	return nil
}

func (h *Handler) validateWorkspaceEditor(ctx context.Context, wsID, userID uuid.UUID) error {
	if err := h.validateWorkspace(ctx, wsID); err != nil {
		return err
	}
	role, err := h.store.GetWorkspaceMemberRole(ctx, sqlc.GetWorkspaceMemberRoleParams{
		WorkspaceID: db.PGUUID(wsID),
		UserID:      db.PGUUID(userID),
	})
	if err != nil {
		return fmt.Errorf("not a workspace member")
	}
	if role != "editor" && role != "owner" {
		return fmt.Errorf("insufficient permissions")
	}
	return nil
}

func (h *Handler) validateCollectionInWorkspace(ctx context.Context, colID, userID uuid.UUID) error {
	_, err := h.store.GetCollectionWorkspaceForMember(ctx, sqlc.GetCollectionWorkspaceForMemberParams{
		UserID:       db.PGUUID(userID),
		CollectionID: db.PGUUID(colID),
	})
	if err != nil {
		return fmt.Errorf("collection not found or access denied")
	}
	return nil
}
