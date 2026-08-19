package catalog

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/imaanmzr/postchi/backend/internal/db"
	"github.com/imaanmzr/postchi/backend/internal/db/sqlc"
	"github.com/imaanmzr/postchi/backend/internal/shared/operationid"
	"github.com/imaanmzr/postchi/backend/internal/shared/respond"
)

type Handler struct {
	store *db.Store
}

func NewHandler(store *db.Store) *Handler {
	return &Handler{store: store}
}

type CollectionSummary struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Description     string `json:"description"`
	RequestCount    int    `json:"request_count"`
	DocumentedCount int    `json:"documented_count"`
}

type EndpointItem struct {
	ID                 string          `json:"id"`
	CollectionID       string          `json:"collection_id"`
	CollectionName     string          `json:"collection_name"`
	Name               string          `json:"name"`
	Method             string          `json:"method"`
	URL                string          `json:"url"`
	Description        string          `json:"description"`
	Tags               []string        `json:"tags"`
	ResponseCodes      []string        `json:"response_codes"`
	SourceSpecID       *string         `json:"source_spec_id,omitempty"`
	SourceOperationID  string          `json:"source_operation_id,omitempty"`
	ApiDoc             json.RawMessage `json:"api_doc"`
	DocsComplete       bool            `json:"docs_complete"`
}

type CatalogResponse struct {
	Collections []CollectionSummary `json:"collections"`
	Endpoints   []EndpointItem      `json:"endpoints"`
}

func (h *Handler) WorkspaceCatalog(w http.ResponseWriter, r *http.Request) {
	wsID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid workspace id")
		return
	}
	resp, err := h.buildCatalog(r, wsID, "", parseFilters(r))
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "catalog failed")
		return
	}
	respond.JSON(w, http.StatusOK, resp)
}

func (h *Handler) CollectionCatalog(w http.ResponseWriter, r *http.Request) {
	colID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid collection id")
		return
	}
	wsID, err := h.store.GetCollectionWorkspaceID(r.Context(), db.PGUUID(colID))
	if err != nil {
		respond.Error(w, http.StatusNotFound, "collection not found")
		return
	}
	resp, err := h.buildCatalog(r, db.FromPGUUID(wsID), colID.String(), parseFilters(r))
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "catalog failed")
		return
	}
	respond.JSON(w, http.StatusOK, resp)
}

type catalogFilters struct {
	Query        string
	Tag          string
	Method       string
	Undocumented bool
	SpecID       string
}

func parseFilters(r *http.Request) catalogFilters {
	return catalogFilters{
		Query:        strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q"))),
		Tag:          strings.TrimSpace(r.URL.Query().Get("tag")),
		Method:       strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("method"))),
		Undocumented: r.URL.Query().Get("undocumented") == "true",
		SpecID:       strings.TrimSpace(r.URL.Query().Get("spec_id")),
	}
}

func (h *Handler) buildCatalog(r *http.Request, wsID uuid.UUID, collectionID string, f catalogFilters) (CatalogResponse, error) {
	resp := CatalogResponse{Collections: []CollectionSummary{}, Endpoints: []EndpointItem{}}
	pgWS := db.PGUUID(wsID)

	linkedRequestIDs, err := h.manualLinkedRequestIDs(r.Context(), pgWS)
	if err != nil {
		return resp, err
	}
	frontmatterLinked, err := h.frontmatterLinkedRequestIDs(r.Context(), pgWS)
	if err != nil {
		return resp, err
	}

	colMap := map[string]CollectionSummary{}
	if collectionID != "" {
		colUUID, _ := uuid.Parse(collectionID)
		rows, err := h.store.ListCatalogCollectionsByID(r.Context(), sqlc.ListCatalogCollectionsByIDParams{
			WorkspaceID:  pgWS,
			CollectionID: db.PGUUID(colUUID),
		})
		if err != nil {
			return resp, err
		}
		for _, row := range rows {
			sid := db.FromPGUUID(row.ID).String()
			colMap[sid] = CollectionSummary{ID: sid, Name: row.Name, Description: row.Description}
		}
	} else {
		rows, err := h.store.ListCatalogCollections(r.Context(), pgWS)
		if err != nil {
			return resp, err
		}
		for _, row := range rows {
			sid := db.FromPGUUID(row.ID).String()
			colMap[sid] = CollectionSummary{ID: sid, Name: row.Name, Description: row.Description}
		}
	}

	if collectionID != "" {
		colUUID, _ := uuid.Parse(collectionID)
		rows, err := h.store.ListCatalogEndpointsByWorkspaceAndCollection(r.Context(), sqlc.ListCatalogEndpointsByWorkspaceAndCollectionParams{
			WorkspaceID:  pgWS,
			CollectionID: db.PGUUID(colUUID),
		})
		if err != nil {
			return resp, err
		}
		h.scanCatalogEndpointRows(rows, colMap, f, linkedRequestIDs, frontmatterLinked, &resp)
	} else {
		rows, err := h.store.ListCatalogEndpointsByWorkspace(r.Context(), pgWS)
		if err != nil {
			return resp, err
		}
		h.scanCatalogEndpointRows(rows, colMap, f, linkedRequestIDs, frontmatterLinked, &resp)
	}

	for _, col := range colMap {
		resp.Collections = append(resp.Collections, col)
	}
	return resp, nil
}

type catalogEndpointRow struct {
	ID                 uuid.UUID
	CollectionID       uuid.UUID
	ColName            string
	Name               string
	Method             string
	URL                string
	Description        string
	ApiDoc             []byte
	SourceSpecID       uuid.UUID
	SourceValid        bool
	SourceOperationID  string
}

func (h *Handler) manualLinkedRequestIDs(ctx context.Context, wsID pgtype.UUID) (map[string]bool, error) {
	rows, err := h.store.ListManualDocLinksByWorkspace(ctx, wsID)
	if err != nil {
		return nil, err
	}
	out := make(map[string]bool, len(rows))
	for _, row := range rows {
		out[db.FromPGUUID(row.RequestID).String()] = true
	}
	return out, nil
}

func (h *Handler) frontmatterLinkedRequestIDs(ctx context.Context, wsID pgtype.UUID) (map[string]bool, error) {
	docRows, err := h.store.ListWorkspaceDocs(ctx, wsID)
	if err != nil {
		return nil, err
	}
	reqRows, err := h.store.ListRequestsByWorkspace(ctx, wsID)
	if err != nil {
		return nil, err
	}
	type docOps struct {
		linked []string
	}
	docs := make([]docOps, 0, len(docRows))
	for _, d := range docRows {
		if len(d.LinkedOperationIds) == 0 {
			continue
		}
		docs = append(docs, docOps{linked: d.LinkedOperationIds})
	}
	out := make(map[string]bool)
	for _, req := range reqRows {
		rid := db.FromPGUUID(req.ID).String()
		aliases := operationid.AliasesForRequest(req.Method, req.Url, req.SourceOperationID)
		for _, doc := range docs {
			if operationid.Matches(doc.linked, aliases) {
				out[rid] = true
				break
			}
		}
	}
	return out, nil
}

func (h *Handler) scanCatalogEndpointRows(rows any, colMap map[string]CollectionSummary, f catalogFilters, linkedRequestIDs, frontmatterLinked map[string]bool, resp *CatalogResponse) {
	switch typed := rows.(type) {
	case []sqlc.ListCatalogEndpointsByWorkspaceRow:
		for _, row := range typed {
			h.applyEndpoint(catalogEndpointRow{
				ID: db.FromPGUUID(row.ID), CollectionID: db.FromPGUUID(row.CollectionID),
				ColName: row.Name, Name: row.Name_2, Method: row.Method, URL: row.Url,
				Description: row.Description, ApiDoc: row.ApiDoc,
				SourceSpecID: db.FromPGUUID(row.SourceSpecID), SourceValid: row.SourceSpecID.Valid,
				SourceOperationID: row.SourceOperationID,
			}, colMap, f, linkedRequestIDs, frontmatterLinked, resp)
		}
	case []sqlc.ListCatalogEndpointsByWorkspaceAndCollectionRow:
		for _, row := range typed {
			h.applyEndpoint(catalogEndpointRow{
				ID: db.FromPGUUID(row.ID), CollectionID: db.FromPGUUID(row.CollectionID),
				ColName: row.Name, Name: row.Name_2, Method: row.Method, URL: row.Url,
				Description: row.Description, ApiDoc: row.ApiDoc,
				SourceSpecID: db.FromPGUUID(row.SourceSpecID), SourceValid: row.SourceSpecID.Valid,
				SourceOperationID: row.SourceOperationID,
			}, colMap, f, linkedRequestIDs, frontmatterLinked, resp)
		}
	}
}

func (h *Handler) applyEndpoint(row catalogEndpointRow, colMap map[string]CollectionSummary, f catalogFilters, linkedRequestIDs, frontmatterLinked map[string]bool, resp *CatalogResponse) {
	ep := EndpointItem{
		ID:             row.ID.String(),
		CollectionID:   row.CollectionID.String(),
		CollectionName: row.ColName,
		Name:           row.Name,
		Method:         row.Method,
		URL:            row.URL,
		Description:    row.Description,
		ApiDoc:         row.ApiDoc,
	}
	if row.SourceOperationID != "" {
		ep.SourceOperationID = row.SourceOperationID
	}
	if row.SourceValid {
		s := row.SourceSpecID.String()
		ep.SourceSpecID = &s
	}
	ep.Tags, ep.ResponseCodes = parseApiDocMeta(row.ApiDoc)
	ep.DocsComplete = isEndpointDocumented(row.Description, row.ApiDoc) ||
		linkedRequestIDs[ep.ID] || frontmatterLinked[ep.ID]

	if f.Method != "" && ep.Method != f.Method {
		return
	}
	if f.Tag != "" && !containsTag(ep.Tags, f.Tag) {
		return
	}
	if f.Undocumented && ep.DocsComplete {
		return
	}
	if f.SpecID != "" && (ep.SourceSpecID == nil || *ep.SourceSpecID != f.SpecID) {
		return
	}
	if f.Query != "" && !matchesQuery(ep, f.Query) {
		return
	}

	col := colMap[ep.CollectionID]
	col.RequestCount++
	if ep.DocsComplete {
		col.DocumentedCount++
	}
	colMap[ep.CollectionID] = col
	resp.Endpoints = append(resp.Endpoints, ep)
}

func parseApiDocMeta(apiDoc []byte) (tags []string, codes []string) {
	if len(apiDoc) == 0 {
		return nil, nil
	}
	var doc map[string]any
	if json.Unmarshal(apiDoc, &doc) != nil {
		return nil, nil
	}
	if rawTags, ok := doc["tags"].([]any); ok {
		for _, t := range rawTags {
			if s, ok := t.(string); ok {
				tags = append(tags, s)
			}
		}
	}
	if responses, ok := doc["responses"].(map[string]any); ok {
		for code := range responses {
			codes = append(codes, code)
		}
	}
	return tags, codes
}

func isEndpointDocumented(desc string, apiDoc []byte) bool {
	if desc != "" {
		return true
	}
	var doc map[string]any
	if json.Unmarshal(apiDoc, &doc) != nil {
		return false
	}
	if responses, ok := doc["responses"].(map[string]any); ok && len(responses) > 0 {
		return true
	}
	return false
}

func containsTag(tags []string, tag string) bool {
	for _, t := range tags {
		if strings.EqualFold(t, tag) {
			return true
		}
	}
	return false
}

func matchesQuery(ep EndpointItem, q string) bool {
	hay := strings.ToLower(ep.Name + " " + ep.Method + " " + ep.URL + " " + ep.Description)
	return strings.Contains(hay, q)
}
