package docsync

import (
	"net/http"
	"sort"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/imaanmzr/postchi/backend/internal/db"
	"github.com/imaanmzr/postchi/backend/internal/db/sqlc"
	"github.com/imaanmzr/postchi/backend/internal/docsync/linkmatcher"
	"github.com/imaanmzr/postchi/backend/internal/shared/operationid"
	"github.com/imaanmzr/postchi/backend/internal/shared/respond"
)

type DocRequestLinkItem struct {
	RequestID         string   `json:"request_id"`
	RequestName       string   `json:"request_name"`
	Method            string   `json:"method"`
	URL               string   `json:"url"`
	SourceOperationID string   `json:"source_operation_id"`
	CollectionName    string   `json:"collection_name"`
	LinkSources       []string `json:"link_sources"`
	LinkID            *string  `json:"link_id,omitempty"`
	SuggestionID      *string  `json:"suggestion_id,omitempty"`
	Confidence        string   `json:"confidence,omitempty"`
	Reason            string   `json:"reason,omitempty"`
}

func (h *Handler) ListDocRequestLinks(w http.ResponseWriter, r *http.Request) {
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
	ctx := r.Context()
	doc, err := h.store.GetWorkspaceDocByID(ctx, sqlc.GetWorkspaceDocByIDParams{
		ID:          db.PGUUID(docID),
		WorkspaceID: db.PGUUID(wsID),
	})
	if err != nil {
		respond.Error(w, http.StatusNotFound, "doc not found")
		return
	}

	type entry = docRequestLinkEntry
	byRequest := make(map[string]*entry)

	if len(doc.LinkedOperationIds) > 0 {
		reqRows, err := h.store.ListRequestsByWorkspace(ctx, db.PGUUID(wsID))
		if err != nil {
			respond.Error(w, http.StatusInternalServerError, "query failed")
			return
		}
		for _, req := range reqRows {
			aliases := operationid.AliasesForRequest(req.Method, req.Url, req.SourceOperationID)
			if !operationid.Matches(doc.LinkedOperationIds, aliases) {
				continue
			}
			rid := db.FromPGUUID(req.ID).String()
			byRequest[rid] = &entry{
				item: DocRequestLinkItem{
					RequestID:         rid,
					RequestName:       req.Name,
					Method:            req.Method,
					URL:               req.Url,
					SourceOperationID: req.SourceOperationID,
					LinkSources:       []string{"frontmatter"},
				},
			}
		}
	}

	if len(doc.LinkedRequestNames) > 0 {
		reqRows, err := h.store.ListRequestsByWorkspace(ctx, db.PGUUID(wsID))
		if err != nil {
			respond.Error(w, http.StatusInternalServerError, "query failed")
			return
		}
		nameSet := make(map[string]struct{}, len(doc.LinkedRequestNames))
		for _, n := range doc.LinkedRequestNames {
			nameSet[n] = struct{}{}
		}
		for _, req := range reqRows {
			slug := linkmatcher.RequestSlug(linkmatcher.Request{Name: req.Name})
			if _, ok := nameSet[slug]; !ok {
				continue
			}
			rid := db.FromPGUUID(req.ID).String()
			if e, ok := byRequest[rid]; ok {
				e.item.LinkSources = appendUniqueSource(e.item.LinkSources, "frontmatter")
				continue
			}
			byRequest[rid] = &entry{
				item: DocRequestLinkItem{
					RequestID:         rid,
					RequestName:       req.Name,
					Method:            req.Method,
					URL:               req.Url,
					SourceOperationID: req.SourceOperationID,
					LinkSources:       []string{"frontmatter"},
				},
			}
		}
	}

	manualRows, err := h.store.ListManualDocLinksByDoc(ctx, sqlc.ListManualDocLinksByDocParams{
		WorkspaceDocID: db.PGUUID(docID),
		WorkspaceID:    db.PGUUID(wsID),
	})
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "query failed")
		return
	}
	for _, row := range manualRows {
		rid := db.FromPGUUID(row.RequestID).String()
		linkID := db.FromPGUUID(row.ID).String()
		if e, ok := byRequest[rid]; ok {
			e.item.LinkSources = appendUniqueSource(e.item.LinkSources, "manual")
			e.item.LinkID = &linkID
			e.item.CollectionName = row.CollectionName
			if e.item.RequestName == "" {
				e.item.RequestName = row.Name
			}
			continue
		}
		byRequest[rid] = &entry{
			item: DocRequestLinkItem{
				RequestID:         rid,
				RequestName:       row.Name,
				Method:            row.Method,
				URL:               row.Url,
				SourceOperationID: row.SourceOperationID,
				CollectionName:    row.CollectionName,
				LinkSources:       []string{"manual"},
				LinkID:            &linkID,
			},
		}
	}

	h.mergePendingSuggestions(ctx, wsID, docID, byRequest)

	out := make([]DocRequestLinkItem, 0, len(byRequest))
	for _, e := range byRequest {
		out = append(out, e.item)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].CollectionName != out[j].CollectionName {
			return out[i].CollectionName < out[j].CollectionName
		}
		return out[i].RequestName < out[j].RequestName
	})
	if out == nil {
		out = []DocRequestLinkItem{}
	}
	respond.JSON(w, http.StatusOK, out)
}

func appendUniqueSource(sources []string, source string) []string {
	for _, s := range sources {
		if s == source {
			return sources
		}
	}
	return append(sources, source)
}
