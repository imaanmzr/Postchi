package request

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/imaanmzr/postchi/backend/internal/db"
	"github.com/imaanmzr/postchi/backend/internal/db/sqlc"
	"github.com/imaanmzr/postchi/backend/internal/shared/operationid"
	"github.com/imaanmzr/postchi/backend/internal/shared/respond"
)

type LinkedWorkspaceDoc struct {
	ID          string   `json:"id"`
	Slug        string   `json:"slug"`
	Title       string   `json:"title"`
	ContentMD   string   `json:"content_md"`
	LinkSources []string `json:"link_sources"`
	LinkID      *string  `json:"link_id,omitempty"`
}

type DocsBundleResponse struct {
	ApiDoc              json.RawMessage    `json:"api_doc"`
	Description         string             `json:"description"`
	LinkedWorkspaceDocs []LinkedWorkspaceDoc `json:"linked_workspace_docs"`
}

func (h *Handler) GetDocsBundle(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	ctx := r.Context()
	row, err := h.store.GetRequestDocsBundle(ctx, db.PGUUID(id))
	if err != nil {
		respond.Error(w, http.StatusNotFound, "not found")
		return
	}
	wsID, err := h.store.GetCollectionWorkspaceID(ctx, row.CollectionID)
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "query failed")
		return
	}
	linked, err := buildLinkedWorkspaceDocs(ctx, h.store, wsID, id, row.Method, row.Url, row.SourceOperationID)
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "query failed")
		return
	}
	respond.JSON(w, http.StatusOK, DocsBundleResponse{
		ApiDoc:              row.ApiDoc,
		Description:         row.Description,
		LinkedWorkspaceDocs: linked,
	})
}

func buildLinkedWorkspaceDocs(
	ctx context.Context,
	store *db.Store,
	wsID pgtype.UUID,
	requestID uuid.UUID,
	method, url, operationID string,
) ([]LinkedWorkspaceDoc, error) {
	type entry struct {
		doc         LinkedWorkspaceDoc
		manualID    *string
		frontmatter bool
		manual      bool
	}
	byDocID := make(map[string]*entry)

	aliases := operationid.AliasesForRequest(method, url, operationID)
	if len(aliases) > 0 {
		rows, err := store.ListWorkspaceDocsByOperationIDs(ctx, sqlc.ListWorkspaceDocsByOperationIDsParams{
			WorkspaceID:   wsID,
			OperationIds: aliases,
		})
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			if !operationid.Matches(row.LinkedOperationIds, aliases) {
				continue
			}
			docID := db.FromPGUUID(row.ID).String()
			byDocID[docID] = &entry{
				doc: LinkedWorkspaceDoc{
					ID:        docID,
					Slug:      row.Slug,
					Title:     row.Title,
					ContentMD: row.ContentMd,
				},
				frontmatter: true,
			}
		}
	}

	manualRows, err := store.ListManualDocLinksForRequest(ctx, db.PGUUID(requestID))
	if err != nil {
		return nil, err
	}
	for _, row := range manualRows {
		docID := db.FromPGUUID(row.ID).String()
		linkID := db.FromPGUUID(row.LinkID).String()
		e, ok := byDocID[docID]
		if !ok {
			byDocID[docID] = &entry{
				doc: LinkedWorkspaceDoc{
					ID:        docID,
					Slug:      row.Slug,
					Title:     row.Title,
					ContentMD: row.ContentMd,
				},
				manualID: &linkID,
				manual:   true,
			}
			continue
		}
		e.manual = true
		e.manualID = &linkID
		if e.doc.ContentMD == "" {
			e.doc.ContentMD = row.ContentMd
		}
	}

	out := make([]LinkedWorkspaceDoc, 0, len(byDocID))
	for _, e := range byDocID {
		sources := make([]string, 0, 2)
		if e.frontmatter {
			sources = append(sources, "frontmatter")
		}
		if e.manual {
			sources = append(sources, "manual")
		}
		e.doc.LinkSources = sources
		if e.manual {
			e.doc.LinkID = e.manualID
		}
		out = append(out, e.doc)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Title < out[j].Title
	})
	if out == nil {
		out = []LinkedWorkspaceDoc{}
	}
	return out, nil
}
