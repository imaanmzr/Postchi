package docsync

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/imaanmzr/postchi/backend/internal/db"
	"github.com/imaanmzr/postchi/backend/internal/db/sqlc"
	"github.com/imaanmzr/postchi/backend/internal/shared/respond"
)

const docAlreadyLinkedMsg = "This document is already linked to this request."

func normalizeDocPath(path string) string {
	path = strings.ToLower(strings.TrimSpace(path))
	path = strings.ReplaceAll(path, " ", "")
	path = strings.ReplaceAll(path, "\\", "/")
	return path
}

func docPathsForCompare(slug, sourcePath string) []string {
	out := make([]string, 0, 2)
	if p := normalizeDocPath(sourcePath); p != "" {
		out = append(out, p)
	}
	if p := normalizeDocPath(strings.ReplaceAll(slug, "-", "/")); p != "" {
		out = append(out, p)
	}
	return out
}

func docsOverlap(slugA, pathA, slugB, pathB string) bool {
	if slugA == slugB {
		return true
	}
	pathsA := docPathsForCompare(slugA, pathA)
	pathsB := docPathsForCompare(slugB, pathB)
	for _, a := range pathsA {
		for _, b := range pathsB {
			if a == b {
				return true
			}
		}
	}
	return false
}

func (h *Handler) requestAlreadyLinkedToDoc(
	ctx context.Context,
	requestID uuid.UUID,
	target sqlc.GetWorkspaceDocByIDRow,
) (bool, error) {
	existing, err := h.store.ListManualDocLinksForRequest(ctx, db.PGUUID(requestID))
	if err != nil {
		return false, err
	}
	targetID := db.FromPGUUID(target.ID).String()
	for _, row := range existing {
		if db.FromPGUUID(row.ID).String() == targetID {
			return true, nil
		}
		if docsOverlap(row.Slug, row.SourcePath, target.Slug, target.SourcePath) {
			return true, nil
		}
	}
	return false, nil
}

type DocLinkItem struct {
	ID                string `json:"id"`
	RequestID         string `json:"request_id"`
	RequestName       string `json:"request_name"`
	Method            string `json:"method"`
	URL               string `json:"url"`
	SourceOperationID string `json:"source_operation_id"`
	CollectionName    string `json:"collection_name"`
}

func (h *Handler) ListDocLinks(w http.ResponseWriter, r *http.Request) {
	wsID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid workspace id")
		return
	}
	docID, err := uuid.Parse(chi.URLParam(r, "docId"))
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid doc id")
		return
	}
	if _, err := h.store.GetWorkspaceDocByID(r.Context(), sqlc.GetWorkspaceDocByIDParams{
		ID:          db.PGUUID(docID),
		WorkspaceID: db.PGUUID(wsID),
	}); err != nil {
		respond.Error(w, http.StatusNotFound, "doc not found")
		return
	}
	rows, err := h.store.ListManualDocLinksByDoc(r.Context(), sqlc.ListManualDocLinksByDocParams{
		WorkspaceDocID: db.PGUUID(docID),
		WorkspaceID:    db.PGUUID(wsID),
	})
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "query failed")
		return
	}
	list := make([]DocLinkItem, 0, len(rows))
	for _, row := range rows {
		list = append(list, DocLinkItem{
			ID:                db.FromPGUUID(row.ID).String(),
			RequestID:         db.FromPGUUID(row.RequestID).String(),
			RequestName:       row.Name,
			Method:            row.Method,
			URL:               row.Url,
			SourceOperationID: row.SourceOperationID,
			CollectionName:    row.CollectionName,
		})
	}
	respond.JSON(w, http.StatusOK, list)
}

func (h *Handler) CreateDocLink(w http.ResponseWriter, r *http.Request) {
	wsID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid workspace id")
		return
	}
	docID, err := uuid.Parse(chi.URLParam(r, "docId"))
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid doc id")
		return
	}
	if _, err := h.store.GetWorkspaceDocByID(r.Context(), sqlc.GetWorkspaceDocByIDParams{
		ID:          db.PGUUID(docID),
		WorkspaceID: db.PGUUID(wsID),
	}); err != nil {
		respond.Error(w, http.StatusNotFound, "doc not found")
		return
	}
	var body struct {
		RequestID   string `json:"request_id"`
		OperationID string `json:"operation_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid body")
		return
	}
	requestID := strings.TrimSpace(body.RequestID)
	operationID := strings.TrimSpace(body.OperationID)
	if requestID == "" && operationID == "" {
		respond.Error(w, http.StatusBadRequest, "request_id or operation_id required")
		return
	}
	if requestID != "" && operationID != "" {
		respond.Error(w, http.StatusBadRequest, "provide request_id or operation_id, not both")
		return
	}

	ctx := r.Context()
	created := make([]DocLinkItem, 0)

	if requestID != "" {
		rid, err := uuid.Parse(requestID)
		if err != nil {
			respond.Error(w, http.StatusBadRequest, "invalid request_id")
			return
		}
		if _, err := h.store.VerifyRequestInWorkspace(ctx, sqlc.VerifyRequestInWorkspaceParams{
			RequestID:   db.PGUUID(rid),
			WorkspaceID: db.PGUUID(wsID),
		}); err != nil {
			respond.Error(w, http.StatusBadRequest, "request not in workspace")
			return
		}
		targetDoc, err := h.store.GetWorkspaceDocByID(ctx, sqlc.GetWorkspaceDocByIDParams{
			ID:          db.PGUUID(docID),
			WorkspaceID: db.PGUUID(wsID),
		})
		if err != nil {
			respond.Error(w, http.StatusNotFound, "doc not found")
			return
		}
		alreadyLinked, err := h.requestAlreadyLinkedToDoc(ctx, rid, targetDoc)
		if err != nil {
			respond.Error(w, http.StatusInternalServerError, "query failed")
			return
		}
		if alreadyLinked {
			respond.Error(w, http.StatusConflict, docAlreadyLinkedMsg)
			return
		}
		item, err := h.createSingleDocLink(ctx, wsID, docID, rid)
		if err != nil {
			respond.Error(w, http.StatusInternalServerError, "create failed")
			return
		}
		created = append(created, item)
	} else {
		ids, err := h.store.ListRequestIDsByOperationInWorkspace(ctx, sqlc.ListRequestIDsByOperationInWorkspaceParams{
			WorkspaceID:   db.PGUUID(wsID),
			OperationID:   operationID,
		})
		if err != nil {
			respond.Error(w, http.StatusInternalServerError, "query failed")
			return
		}
		if len(ids) == 0 {
			respond.Error(w, http.StatusBadRequest, "no requests match operation_id")
			return
		}
		for _, idRow := range ids {
			rid := db.FromPGUUID(idRow)
			item, err := h.createSingleDocLink(ctx, wsID, docID, rid)
			if err != nil {
				respond.Error(w, http.StatusInternalServerError, "create failed")
				return
			}
			created = append(created, item)
		}
	}

	if len(created) == 1 {
		respond.JSON(w, http.StatusCreated, created[0])
		return
	}
	respond.JSON(w, http.StatusCreated, created)
}

func (h *Handler) createSingleDocLink(ctx context.Context, wsID, docID, requestID uuid.UUID) (DocLinkItem, error) {
	link, err := h.store.CreateManualDocLink(ctx, sqlc.CreateManualDocLinkParams{
		WorkspaceDocID: db.PGUUID(docID),
		RequestID:      db.PGUUID(requestID),
	})
	if err != nil {
		return DocLinkItem{}, err
	}
	rows, err := h.store.ListManualDocLinksByDoc(ctx, sqlc.ListManualDocLinksByDocParams{
		WorkspaceDocID: db.PGUUID(docID),
		WorkspaceID:    db.PGUUID(wsID),
	})
	if err != nil {
		return DocLinkItem{}, err
	}
	linkID := db.FromPGUUID(link.ID)
	for _, row := range rows {
		if db.FromPGUUID(row.ID) == linkID && db.FromPGUUID(row.RequestID) == requestID {
			return DocLinkItem{
				ID:                linkID.String(),
				RequestID:         requestID.String(),
				RequestName:       row.Name,
				Method:            row.Method,
				URL:               row.Url,
				SourceOperationID: row.SourceOperationID,
				CollectionName:    row.CollectionName,
			}, nil
		}
	}
	return DocLinkItem{
		ID:        linkID.String(),
		RequestID: requestID.String(),
	}, nil
}

func (h *Handler) DeleteDocLink(w http.ResponseWriter, r *http.Request) {
	wsID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid workspace id")
		return
	}
	linkID, err := uuid.Parse(chi.URLParam(r, "linkId"))
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid link id")
		return
	}
	if _, err := h.store.GetManualDocLink(r.Context(), sqlc.GetManualDocLinkParams{
		ID:          db.PGUUID(linkID),
		WorkspaceID: db.PGUUID(wsID),
	}); err != nil {
		respond.Error(w, http.StatusNotFound, "not found")
		return
	}
	if err := h.store.DeleteManualDocLink(r.Context(), sqlc.DeleteManualDocLinkParams{
		ID:          db.PGUUID(linkID),
		WorkspaceID: db.PGUUID(wsID),
	}); err != nil {
		respond.Error(w, http.StatusInternalServerError, "delete failed")
		return
	}
	respond.JSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
