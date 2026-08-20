package docsync

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/imaanmzr/postchi/backend/internal/auth"
	"github.com/imaanmzr/postchi/backend/internal/db"
	"github.com/imaanmzr/postchi/backend/internal/db/sqlc"
	"github.com/imaanmzr/postchi/backend/internal/shared/respond"
)

func (h *Handler) ListDocLinkSuggestions(w http.ResponseWriter, r *http.Request) {
	wsID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid workspace id")
		return
	}
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	if status == "" {
		status = "pending"
	}
	rows, err := h.store.ListDocLinkSuggestions(r.Context(), sqlc.ListDocLinkSuggestionsParams{
		WorkspaceID: db.PGUUID(wsID),
		Status:      status,
	})
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "query failed")
		return
	}
	type item struct {
		ID             string          `json:"id"`
		DocID          string          `json:"doc_id"`
		DocTitle       string          `json:"doc_title"`
		DocSlug        string          `json:"doc_slug"`
		RequestID      string          `json:"request_id"`
		RequestName    string          `json:"request_name"`
		Method         string          `json:"method"`
		URL            string          `json:"url"`
		CollectionName string          `json:"collection_name"`
		Reason         string          `json:"reason"`
		Confidence     string          `json:"confidence"`
		Evidence       json.RawMessage `json:"evidence"`
		Status         string          `json:"status"`
	}
	out := make([]item, 0, len(rows))
	for _, row := range rows {
		out = append(out, item{
			ID:             db.FromPGUUID(row.ID).String(),
			DocID:          db.FromPGUUID(row.WorkspaceDocID).String(),
			DocTitle:       row.DocTitle,
			DocSlug:        row.DocSlug,
			RequestID:      db.FromPGUUID(row.RequestID).String(),
			RequestName:    row.RequestName,
			Method:         row.Method,
			URL:            row.Url,
			CollectionName: row.CollectionName,
			Reason:         row.Reason,
			Confidence:     row.Confidence,
			Evidence:       row.Evidence,
			Status:         row.Status,
		})
	}
	if out == nil {
		out = []item{}
	}
	respond.JSON(w, http.StatusOK, out)
}

func (h *Handler) AcceptDocLinkSuggestion(w http.ResponseWriter, r *http.Request) {
	h.reviewDocLinkSuggestion(w, r, "accepted", true)
}

func (h *Handler) RejectDocLinkSuggestion(w http.ResponseWriter, r *http.Request) {
	h.reviewDocLinkSuggestion(w, r, "rejected", false)
}

func (h *Handler) AcceptAllDocLinkSuggestions(w http.ResponseWriter, r *http.Request) {
	wsID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid workspace id")
		return
	}
	confidence := strings.TrimSpace(r.URL.Query().Get("confidence"))
	if confidence == "" {
		confidence = "high"
	}
	userID, _ := auth.UserIDFromContext(r.Context())
	ctx := r.Context()
	rows, err := h.store.ListDocLinkSuggestions(ctx, sqlc.ListDocLinkSuggestionsParams{
		WorkspaceID: db.PGUUID(wsID),
		Status:      "pending",
	})
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "query failed")
		return
	}
	var accepted int
	for _, row := range rows {
		if row.Confidence != confidence && !(confidence == "high" && row.Confidence == "exact") {
			continue
		}
		if err := h.acceptSuggestion(ctx, wsID, db.FromPGUUID(row.ID), userID); err != nil {
			respond.Error(w, http.StatusInternalServerError, "accept failed")
			return
		}
		accepted++
	}
	respond.JSON(w, http.StatusOK, map[string]int{"accepted": accepted})
}

func (h *Handler) reviewDocLinkSuggestion(w http.ResponseWriter, r *http.Request, status string, createLink bool) {
	wsID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid workspace id")
		return
	}
	sugID, err := uuid.Parse(chi.URLParam(r, "suggestionId"))
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid suggestion id")
		return
	}
	userID, _ := auth.UserIDFromContext(r.Context())
	ctx := r.Context()
	if createLink {
		if err := h.acceptSuggestion(ctx, wsID, sugID, userID); err != nil {
			if errors.Is(err, errSuggestionNotFound) {
				respond.Error(w, http.StatusNotFound, "not found")
				return
			}
			respond.Error(w, http.StatusInternalServerError, "accept failed")
			return
		}
		respond.JSON(w, http.StatusOK, map[string]string{"status": "accepted"})
		return
	}
	row, err := h.store.GetDocLinkSuggestion(ctx, sqlc.GetDocLinkSuggestionParams{
		ID:          db.PGUUID(sugID),
		WorkspaceID: db.PGUUID(wsID),
	})
	if err != nil {
		respond.Error(w, http.StatusNotFound, "not found")
		return
	}
	if err := h.store.UpdateDocLinkSuggestionStatus(ctx, sqlc.UpdateDocLinkSuggestionStatusParams{
		Status:     status,
		ReviewedBy: db.PGUUID(userID),
		ID:         row.ID,
		WorkspaceID: row.WorkspaceID,
	}); err != nil {
		respond.Error(w, http.StatusInternalServerError, "update failed")
		return
	}
	respond.JSON(w, http.StatusOK, map[string]string{"status": status})
}

var errSuggestionNotFound = errors.New("suggestion not found")

func (h *Handler) acceptSuggestion(ctx context.Context, wsID, sugID, userID uuid.UUID) error {
	row, err := h.store.GetDocLinkSuggestion(ctx, sqlc.GetDocLinkSuggestionParams{
		ID:          db.PGUUID(sugID),
		WorkspaceID: db.PGUUID(wsID),
	})
	if err != nil {
		return errSuggestionNotFound
	}
	docID := db.FromPGUUID(row.WorkspaceDocID)
	reqID := db.FromPGUUID(row.RequestID)
	if _, err := h.store.CreateManualDocLink(ctx, sqlc.CreateManualDocLinkParams{
		WorkspaceDocID: db.PGUUID(docID),
		RequestID:      db.PGUUID(reqID),
	}); err != nil {
		return err
	}
	return h.store.UpdateDocLinkSuggestionStatus(ctx, sqlc.UpdateDocLinkSuggestionStatusParams{
		Status:      "accepted",
		ReviewedBy:  db.PGUUID(userID),
		ID:          row.ID,
		WorkspaceID: row.WorkspaceID,
	})
}

func (h *Handler) mergePendingSuggestions(ctx context.Context, wsID, docID uuid.UUID, byRequest map[string]*docRequestLinkEntry) {
	rows, err := h.store.ListPendingDocLinkSuggestionsByDoc(ctx, sqlc.ListPendingDocLinkSuggestionsByDocParams{
		WorkspaceDocID: db.PGUUID(docID),
		WorkspaceID:    db.PGUUID(wsID),
	})
	if err != nil {
		return
	}
	for _, row := range rows {
		rid := db.FromPGUUID(row.RequestID).String()
		sugID := db.FromPGUUID(row.ID).String()
		if e, ok := byRequest[rid]; ok {
			e.item.LinkSources = appendUniqueSource(e.item.LinkSources, "suggested")
			e.item.SuggestionID = &sugID
			e.item.Confidence = row.Confidence
			e.item.Reason = row.Reason
			continue
		}
		byRequest[rid] = &docRequestLinkEntry{
			item: DocRequestLinkItem{
				RequestID:         rid,
				RequestName:       row.RequestName,
				Method:            row.Method,
				URL:               row.Url,
				SourceOperationID: row.SourceOperationID,
				CollectionName:    row.CollectionName,
				LinkSources:       []string{"suggested"},
				SuggestionID:      &sugID,
				Confidence:        row.Confidence,
				Reason:            row.Reason,
			},
		}
	}
}

type docRequestLinkEntry struct {
	item DocRequestLinkItem
}
