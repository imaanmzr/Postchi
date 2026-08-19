package docsync

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/imaanmzr/postchi/backend/internal/auth"
	"github.com/imaanmzr/postchi/backend/internal/db"
	"github.com/imaanmzr/postchi/backend/internal/db/sqlc"
	"github.com/imaanmzr/postchi/backend/internal/docsync/linkmatcher"
	"github.com/imaanmzr/postchi/backend/internal/shared/operationid"
	"github.com/imaanmzr/postchi/backend/internal/shared/respond"
)

func (h *Handler) AnalyzeDocLinks(w http.ResponseWriter, r *http.Request) {
	wsID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid workspace id")
		return
	}
	ctx := r.Context()
	refreshRejected := r.URL.Query().Get("refresh") == "rejected"

	docRows, err := h.store.ListWorkspaceDocs(ctx, db.PGUUID(wsID))
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "query failed")
		return
	}
	reqRows, err := h.store.ListRequestsByWorkspace(ctx, db.PGUUID(wsID))
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "query failed")
		return
	}

	docs := make([]linkmatcher.Doc, 0, len(docRows))
	docByID := make(map[string]linkmatcher.Doc, len(docRows))
	for _, row := range docRows {
		d := linkmatcher.Doc{
			ID:                 db.FromPGUUID(row.ID).String(),
			Slug:               row.Slug,
			Title:              row.Title,
			SourcePath:         row.SourcePath,
			ContentMD:          row.ContentMd,
			LinkedOperationIDs: row.LinkedOperationIds,
		}
		docs = append(docs, d)
		docByID[d.ID] = d
	}

	collectionNames := map[string]string{}
	for _, row := range reqRows {
		collectionNames[db.FromPGUUID(row.CollectionID).String()] = ""
	}
	colRows, _ := h.store.ListCatalogCollections(ctx, db.PGUUID(wsID))
	for _, c := range colRows {
		collectionNames[db.FromPGUUID(c.ID).String()] = c.Name
	}

	requests := make([]linkmatcher.Request, 0, len(reqRows))
	reqByID := make(map[string]linkmatcher.Request, len(reqRows))
	for _, row := range reqRows {
		cid := db.FromPGUUID(row.CollectionID).String()
		req := linkmatcher.Request{
			ID:                db.FromPGUUID(row.ID).String(),
			Name:              row.Name,
			Method:            row.Method,
			URL:               row.Url,
			SourceOperationID: row.SourceOperationID,
			CollectionName:    collectionNames[cid],
		}
		requests = append(requests, req)
		reqByID[req.ID] = req
	}

	manualRows, _ := h.store.ListManualDocLinksByWorkspace(ctx, db.PGUUID(wsID))
	manualPairs := make(map[string]bool, len(manualRows))
	for _, row := range manualRows {
		key := db.FromPGUUID(row.WorkspaceDocID).String() + ":" + db.FromPGUUID(row.RequestID).String()
		manualPairs[key] = true
	}

	existing, _ := h.store.ListDocLinkSuggestionsForAnalyze(ctx, db.PGUUID(wsID))
	rejected := make(map[string]bool)
	for _, row := range existing {
		if row.Status == "rejected" && !refreshRejected {
			key := db.FromPGUUID(row.WorkspaceDocID).String() + ":" + db.FromPGUUID(row.RequestID).String()
			rejected[key] = true
		}
	}

	skip := func(docID, requestID string) bool {
		if manualPairs[docID+":"+requestID] {
			return true
		}
		if rejected[docID+":"+requestID] {
			return true
		}
		doc := docByID[docID]
		req := reqByID[requestID]
		if operationid.Matches(doc.LinkedOperationIDs, operationid.AliasesForRequest(req.Method, req.URL, req.SourceOperationID)) {
			return true
		}
		return false
	}

	candidates := linkmatcher.Analyze(docs, requests, skip)
	var upserted int
	keepIDs := make([]pgtype.UUID, 0, len(candidates))
	for _, c := range candidates {
		docUUID, _ := uuid.Parse(c.DocID)
		reqUUID, _ := uuid.Parse(c.RequestID)
		row, err := h.store.UpsertDocLinkSuggestion(ctx, sqlc.UpsertDocLinkSuggestionParams{
			WorkspaceID:    db.PGUUID(wsID),
			WorkspaceDocID: db.PGUUID(docUUID),
			RequestID:      db.PGUUID(reqUUID),
			Reason:         c.Reason,
			Confidence:     c.Confidence,
			Evidence:       linkmatcher.EvidenceJSON(c.Evidence),
		})
		if err != nil {
			respond.Error(w, http.StatusInternalServerError, "save failed")
			return
		}
		keepIDs = append(keepIDs, row.ID)
		upserted++
	}
	if len(keepIDs) > 0 {
		_ = h.store.DeleteStalePendingDocLinkSuggestions(ctx, sqlc.DeleteStalePendingDocLinkSuggestionsParams{
			WorkspaceID: db.PGUUID(wsID),
			KeepIds:     keepIDs,
		})
	} else {
		_ = h.store.DeleteStalePendingDocLinkSuggestions(ctx, sqlc.DeleteStalePendingDocLinkSuggestionsParams{
			WorkspaceID: db.PGUUID(wsID),
			KeepIds:     []pgtype.UUID{},
		})
	}

	pendingCount, _ := h.store.CountPendingDocLinkSuggestions(ctx, db.PGUUID(wsID))
	respond.JSON(w, http.StatusOK, map[string]int{
		"created":       upserted,
		"updated":       upserted,
		"pending_total": int(pendingCount),
	})
}

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
		if row.Confidence != confidence {
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
