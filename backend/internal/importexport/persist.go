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
	headers, _ := json.Marshal(req.Headers)
	params, _ := json.Marshal(req.Params)
	pathVars, _ := json.Marshal(req.PathVars)
	body, _ := json.Marshal(req.Body)
	authB, _ := json.Marshal(req.Auth)
	settings, _ := json.Marshal(req.Settings)
	err := q.InsertImportedRequest(ctx, sqlc.InsertImportedRequestParams{
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
