package collection

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/imaanmzr/postchi/backend/internal/auth"
	"github.com/imaanmzr/postchi/backend/internal/db"
	"github.com/imaanmzr/postchi/backend/internal/db/sqlc"
	"github.com/imaanmzr/postchi/backend/internal/request"
	"github.com/imaanmzr/postchi/backend/internal/shared/domain"
	"github.com/imaanmzr/postchi/backend/internal/shared/respond"
)

type Handler struct {
	store *db.Store
}

func NewHandler(store *db.Store) *Handler {
	return &Handler{store: store}
}

type Collection struct {
	ID                 string               `json:"id"`
	WorkspaceID        string               `json:"workspace_id"`
	ParentID           *string              `json:"parent_id,omitempty"`
	Name               string               `json:"name"`
	Description        string               `json:"description"`
	SortOrder          int                  `json:"sort_order"`
	Variables          domain.VariablesSpec `json:"variables"`
	Headers            []request.KVPair     `json:"headers"`
	Auth               request.AuthSpec     `json:"auth"`
	Presets            json.RawMessage      `json:"presets,omitempty"`
	Proxy              json.RawMessage      `json:"proxy,omitempty"`
	ClientCertificates json.RawMessage      `json:"client_certificates,omitempty"`
	Secrets            json.RawMessage      `json:"secrets,omitempty"`
	PreRequestScript   string               `json:"pre_request_script"`
	TestScript         string               `json:"test_script"`
}

func (h *Handler) ListByWorkspace(w http.ResponseWriter, r *http.Request) {
	wsID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid workspace id")
		return
	}
	rows, err := h.store.ListCollectionsByWorkspace(r.Context(), db.PGUUID(wsID))
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to list collections")
		return
	}
	list := make([]Collection, 0, len(rows))
	for _, row := range rows {
		list = append(list, collectionFromListCollectionsByWorkspaceRow(row))
	}
	respond.JSON(w, http.StatusOK, list)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())
	var req Collection
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" || req.WorkspaceID == "" {
		respond.Error(w, http.StatusBadRequest, "name and workspace_id required")
		return
	}
	c, err := h.insert(r, req, userID)
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to create collection")
		return
	}
	respond.JSON(w, http.StatusCreated, c)
}

func (h *Handler) insert(r *http.Request, req Collection, userID uuid.UUID) (Collection, error) {
	wsID, _ := uuid.Parse(req.WorkspaceID)
	var parentID *uuid.UUID
	if req.ParentID != nil && *req.ParentID != "" {
		p, _ := uuid.Parse(*req.ParentID)
		parentID = &p
	}
	vars, headers, authB, presets, proxy, certs, secrets := marshalCollectionFields(req)
	id, err := h.store.CreateCollection(r.Context(), sqlc.CreateCollectionParams{
		WorkspaceID:        db.PGUUID(wsID),
		ParentID:           db.PGUUIDPtr(parentID),
		Name:               req.Name,
		Description:        req.Description,
		SortOrder:          int32(req.SortOrder),
		Variables:          vars,
		Headers:            headers,
		Auth:               authB,
		Presets:            presets,
		Proxy:              proxy,
		ClientCertificates: certs,
		Secrets:            secrets,
		PreRequestScript:   req.PreRequestScript,
		TestScript:         req.TestScript,
		CreatedBy:          db.PGUUID(userID),
	})
	if err != nil {
		return Collection{}, err
	}
	req.ID = db.FromPGUUID(id).String()
	return req, nil
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	existingRow, err := h.store.GetCollection(r.Context(), db.PGUUID(id))
	if err != nil {
		respond.Error(w, http.StatusNotFound, "not found")
		return
	}
	existing := collectionFromGetCollectionRow(existingRow)
	req, err := applyCollectionPatch(existing, r.Body)
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid body")
		return
	}
	vars, headers, authB, presets, proxy, certs, secrets := marshalCollectionFields(req)
	parentChanged := !parentIDsEqual(existing.ParentID, req.ParentID)
	var parentID *uuid.UUID
	if req.ParentID != nil && *req.ParentID != "" {
		p, err := uuid.Parse(*req.ParentID)
		if err != nil {
			respond.Error(w, http.StatusBadRequest, "invalid parent id")
			return
		}
		if p == id {
			respond.Error(w, http.StatusBadRequest, "collection cannot be its own parent")
			return
		}
		if parentChanged {
			var wouldCycle bool
			if err := h.store.Pool.QueryRow(r.Context(), `
				WITH RECURSIVE ancestors AS (
					SELECT id, parent_id, ARRAY[id] AS path FROM collections WHERE id = $1
					UNION ALL
					SELECT c.id, c.parent_id, a.path || c.id
					FROM collections c JOIN ancestors a ON c.id = a.parent_id
					WHERE NOT c.id = ANY(a.path)
				)
				SELECT EXISTS (SELECT 1 FROM ancestors WHERE id = $2)
			`, p, id).Scan(&wouldCycle); err != nil {
				respond.Error(w, http.StatusInternalServerError, "failed to validate hierarchy")
				return
			}
			if wouldCycle {
				respond.Error(w, http.StatusBadRequest, "move would create a folder cycle")
				return
			}
		}
		parentID = &p
	}
	err = h.store.UpdateCollection(r.Context(), sqlc.UpdateCollectionParams{
		Name:               req.Name,
		Description:        req.Description,
		ParentID:           db.PGUUIDPtr(parentID),
		SortOrder:          int32(req.SortOrder),
		Variables:          vars,
		Headers:            headers,
		Auth:               authB,
		Presets:            presets,
		Proxy:              proxy,
		ClientCertificates: certs,
		Secrets:            secrets,
		PreRequestScript:   req.PreRequestScript,
		TestScript:         req.TestScript,
		ID:                 db.PGUUID(id),
	})
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to update")
		return
	}
	updated, err := h.store.GetCollection(r.Context(), db.PGUUID(id))
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to load collection")
		return
	}
	respond.JSON(w, http.StatusOK, collectionFromGetCollectionRow(updated))
}

func applyCollectionPatch(existing Collection, body io.Reader) (Collection, error) {
	patched := existing
	if err := json.NewDecoder(body).Decode(&patched); err != nil {
		return Collection{}, err
	}
	patched.ID = existing.ID
	patched.WorkspaceID = existing.WorkspaceID
	if strings.TrimSpace(patched.Name) == "" {
		patched.Name = existing.Name
	}
	return patched, nil
}

func parentIDsEqual(a, b *string) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func marshalCollectionFields(req Collection) ([]byte, []byte, []byte, []byte, []byte, []byte, []byte) {
	vars, _ := json.Marshal(req.Variables)
	headers, _ := json.Marshal(req.Headers)
	authB, _ := json.Marshal(req.Auth)
	presets := defaultJSON(req.Presets, "[]")
	proxy := defaultJSON(req.Proxy, "{}")
	certs := defaultJSON(req.ClientCertificates, "[]")
	secrets := defaultJSON(req.Secrets, "[]")
	return vars, headers, authB, presets, proxy, certs, secrets
}

func defaultJSON(raw json.RawMessage, fallback string) []byte {
	if len(raw) > 0 {
		return raw
	}
	return []byte(fallback)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.store.DeleteCollection(r.Context(), db.PGUUID(id)); err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to delete")
		return
	}
	respond.JSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	row, err := h.store.GetCollection(r.Context(), db.PGUUID(id))
	if err != nil {
		respond.Error(w, http.StatusNotFound, "not found")
		return
	}
	respond.JSON(w, http.StatusOK, collectionFromGetCollectionRow(row))
}

func (h *Handler) Duplicate(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	row, err := h.store.GetCollection(r.Context(), db.PGUUID(id))
	if err != nil {
		respond.Error(w, http.StatusNotFound, "not found")
		return
	}
	src := collectionFromGetCollectionRow(row)
	newID, err := h.duplicateTree(r, src, src.ParentID, userID, src.Name+" (copy)")
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "duplicate failed")
		return
	}
	respond.JSON(w, http.StatusCreated, map[string]string{"id": newID.String()})
}

func (h *Handler) duplicateTree(r *http.Request, src Collection, parentID *string, userID uuid.UUID, name string) (uuid.UUID, error) {
	src.Name = name
	src.ParentID = parentID
	created, err := h.insert(r, src, userID)
	if err != nil {
		return uuid.Nil, err
	}
	newColID, _ := uuid.Parse(created.ID)
	oldColID, _ := uuid.Parse(src.ID)

	reqs, _ := h.store.ListRequestsForDuplicate(r.Context(), db.PGUUID(oldColID))
	for _, m := range reqs {
		_ = h.store.DuplicateRequest(r.Context(), sqlc.DuplicateRequestParams{
			CollectionID:     db.PGUUID(newColID),
			Name:             m.Name,
			Method:           m.Method,
			Url:              m.Url,
			Headers:          m.Headers,
			Params:           m.Params,
			PathVars:         m.PathVars,
			Body:             m.Body,
			Auth:             m.Auth,
			Settings:         m.Settings,
			PreRequestScript: m.PreRequestScript,
			TestScript:       m.TestScript,
			SortOrder:        m.SortOrder,
			Description:      m.Description,
			CreatedBy:        db.PGUUID(userID),
		})
	}

	children, _ := h.store.ListCollectionsByParent(r.Context(), db.PGUUID(oldColID))
	for _, child := range children {
		c := collectionFromListCollectionsByParentRow(child)
		pid := newColID.String()
		_, _ = h.duplicateTree(r, c, &pid, userID, child.Name)
	}
	return newColID, nil
}

func (h *Handler) Reorder(w http.ResponseWriter, r *http.Request) {
	var items []struct {
		ID        string  `json:"id"`
		ParentID  *string `json:"parent_id"`
		SortOrder int     `json:"sort_order"`
	}
	if err := json.NewDecoder(r.Body).Decode(&items); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid body")
		return
	}

	tx, q, err := h.store.Begin(r.Context())
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to reorder")
		return
	}
	defer tx.Rollback(r.Context())

	for _, item := range items {
		id, err := uuid.Parse(item.ID)
		if err != nil {
			respond.Error(w, http.StatusBadRequest, "invalid collection id")
			return
		}
		var parentID *uuid.UUID
		if item.ParentID != nil && *item.ParentID != "" {
			p, err := uuid.Parse(*item.ParentID)
			if err != nil {
				respond.Error(w, http.StatusBadRequest, "invalid parent id")
				return
			}
			if p == id {
				respond.Error(w, http.StatusBadRequest, "collection cannot be its own parent")
				return
			}
			parentID = &p
		}
		if err := q.ReorderCollection(r.Context(), sqlc.ReorderCollectionParams{
			ParentID:  db.PGUUIDPtr(parentID),
			SortOrder: int32(item.SortOrder),
			ID:        db.PGUUID(id),
		}); err != nil {
			respond.Error(w, http.StatusInternalServerError, "failed to reorder")
			return
		}
	}

	var cycles int
	err = tx.QueryRow(r.Context(), `
		WITH RECURSIVE walk AS (
			SELECT id, parent_id, ARRAY[id] AS path, false AS is_cycle
			FROM collections WHERE parent_id IS NOT NULL
			UNION ALL
			SELECT c.id, c.parent_id, w.path || c.id, c.id = ANY(w.path)
			FROM collections c JOIN walk w ON c.id = w.parent_id
			WHERE NOT w.is_cycle
		)
		SELECT count(*) FROM walk WHERE is_cycle
	`).Scan(&cycles)
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to validate hierarchy")
		return
	}
	if cycles > 0 {
		respond.Error(w, http.StatusBadRequest, "reorder would create a folder cycle")
		return
	}

	if err := tx.Commit(r.Context()); err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to reorder")
		return
	}
	respond.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) Docs(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	nd, _ := h.store.GetCollectionNameDescription(r.Context(), db.PGUUID(id))
	rows, err := h.store.ListCollectionDocEndpoints(r.Context(), db.PGUUID(id))
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "query failed")
		return
	}
	var endpoints []docEndpoint
	for _, ep := range rows {
		endpoint := docEndpoint{
			ID:          db.FromPGUUID(ep.ID).String(),
			Name:        ep.Name,
			Method:      ep.Method,
			URL:         ep.Url,
			Description: ep.Description,
			ApiDoc:      ep.ApiDoc,
		}
		if ep.SourceSpecID.Valid {
			s := db.FromPGUUID(ep.SourceSpecID).String()
			endpoint.SourceSpecID = &s
		}
		endpoints = append(endpoints, endpoint)
	}
	format := r.URL.Query().Get("format")
	if format == "json" {
		export := docExport{}
		export.Collection.ID = id.String()
		export.Collection.Name = nd.Name
		export.Collection.Description = nd.Description
		export.Endpoints = endpoints
		if export.Endpoints == nil {
			export.Endpoints = []docEndpoint{}
		}
		respond.JSON(w, http.StatusOK, export)
		return
	}
	md := buildDocsMarkdown(nd.Name, nd.Description, endpoints)
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.Write([]byte(md))
}

func collectionFromFields(
	id, wsID pgtype.UUID,
	parentID pgtype.UUID,
	name, description string,
	sortOrder int32,
	vars, headers, authB, presets, proxy, certs, secrets []byte,
	preRequestScript, testScript string,
) Collection {
	c := Collection{
		ID:                 db.FromPGUUID(id).String(),
		WorkspaceID:        db.FromPGUUID(wsID).String(),
		Name:               name,
		Description:        description,
		SortOrder:          int(sortOrder),
		Variables:          domain.ParseVariablesSpec(vars),
		Presets:            presets,
		Proxy:              proxy,
		ClientCertificates: certs,
		Secrets:            secrets,
		PreRequestScript:   preRequestScript,
		TestScript:         testScript,
	}
	if parentID.Valid {
		s := db.FromPGUUID(parentID).String()
		c.ParentID = &s
	}
	_ = json.Unmarshal(headers, &c.Headers)
	_ = json.Unmarshal(authB, &c.Auth)
	return c
}

func collectionFromGetCollectionRow(row sqlc.GetCollectionRow) Collection {
	return collectionFromFields(
		row.ID, row.WorkspaceID, row.ParentID,
		row.Name, row.Description, row.SortOrder,
		row.Variables, row.Headers, row.Auth, row.Presets, row.Proxy, row.ClientCertificates, row.Secrets,
		row.PreRequestScript, row.TestScript,
	)
}

func collectionFromListCollectionsByWorkspaceRow(row sqlc.ListCollectionsByWorkspaceRow) Collection {
	return collectionFromFields(
		row.ID, row.WorkspaceID, row.ParentID,
		row.Name, row.Description, row.SortOrder,
		row.Variables, row.Headers, row.Auth, row.Presets, row.Proxy, row.ClientCertificates, row.Secrets,
		row.PreRequestScript, row.TestScript,
	)
}

func collectionFromListCollectionsByParentRow(row sqlc.ListCollectionsByParentRow) Collection {
	return collectionFromFields(
		row.ID, row.WorkspaceID, row.ParentID,
		row.Name, row.Description, row.SortOrder,
		row.Variables, row.Headers, row.Auth, row.Presets, row.Proxy, row.ClientCertificates, row.Secrets,
		row.PreRequestScript, row.TestScript,
	)
}
