package docsync

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/imaanmzr/postchi/backend/internal/db"
	"github.com/imaanmzr/postchi/backend/internal/db/sqlc"
	"github.com/imaanmzr/postchi/backend/internal/docsync/linkmatcher"
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

	autoResult, err := h.runWorkspaceAnalyzeAutoLink(ctx, wsID)
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "auto-link failed")
		return
	}

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

	docs := docRowsToLinkmatcher(docRows)

	collectionNames := map[string]string{}
	for _, row := range reqRows {
		collectionNames[db.FromPGUUID(row.CollectionID).String()] = ""
	}
	colRows, _ := h.store.ListCatalogCollections(ctx, db.PGUUID(wsID))
	for _, c := range colRows {
		collectionNames[db.FromPGUUID(c.ID).String()] = c.Name
	}
	requests := requestRowsToLinkmatcher(reqRows, collectionNames)

	reqByID := make(map[string]linkmatcher.Request, len(requests))
	for _, req := range requests {
		reqByID[req.ID] = req
	}

	skip, err := h.buildAutoLinkSkip(ctx, wsID, docs, reqByID, refreshRejected)
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "query failed")
		return
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
		"auto_linked":   autoResult.AutoLinked,
		"created":       upserted,
		"updated":       upserted,
		"pending_total": int(pendingCount),
	})
}
