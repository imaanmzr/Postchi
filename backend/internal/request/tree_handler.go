package request

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/imaanmzr/postchi/backend/internal/auth"
	"github.com/imaanmzr/postchi/backend/internal/db"
	"github.com/imaanmzr/postchi/backend/internal/db/sqlc"
	"github.com/imaanmzr/postchi/backend/internal/shared/respond"
)

func (h *Handler) CreateChild(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())
	templateID, _ := uuid.Parse(chi.URLParam(r, "id"))
	template, err := h.loadModelRaw(r, templateID)
	if err != nil {
		respond.Error(w, http.StatusNotFound, "template not found")
		return
	}
	var req struct {
		Name      string                     `json:"name"`
		Overrides map[string]json.RawMessage `json:"overrides"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid body")
		return
	}
	name := req.Name
	if name == "" {
		name = template.Name + " variant"
	}
	child := template
	child.Name = name
	child.TemplateID = &template.ID
	child.IsTemplate = false
	if req.Overrides != nil {
		child, overridden := applyOverrides(child, req.Overrides)
		child.OverriddenFields = overridden
	}
	colID, _ := uuid.Parse(template.CollectionID)
	newID, err := h.insertModel(r, colID, child, userID)
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to create variant")
		return
	}
	child.ID = newID.String()
	respond.JSON(w, http.StatusCreated, child)
}

func (h *Handler) ListChildren(w http.ResponseWriter, r *http.Request) {
	templateID, _ := uuid.Parse(chi.URLParam(r, "id"))
	ids, err := h.store.ListRequestIDsByTemplate(r.Context(), db.PGUUID(templateID))
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "query failed")
		return
	}
	type childSummary struct {
		ID               string   `json:"id"`
		Name             string   `json:"name"`
		Method           string   `json:"method"`
		OverriddenFields []string `json:"overridden_fields"`
	}
	var list []childSummary
	for _, idpg := range ids {
		row, err := h.store.GetRequest(r.Context(), idpg)
		if err != nil {
			continue
		}
		m := modelFromGetRequestRow(row)
		list = append(list, childSummary{
			ID:               m.ID,
			Name:             m.Name,
			Method:           m.Method,
			OverriddenFields: m.OverriddenFields,
		})
	}
	if list == nil {
		list = []childSummary{}
	}
	respond.JSON(w, http.StatusOK, list)
}

func (h *Handler) ResetField(w http.ResponseWriter, r *http.Request) {
	id, _ := uuid.Parse(chi.URLParam(r, "id"))
	var req struct {
		Field string `json:"field"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Field == "" {
		respond.Error(w, http.StatusBadRequest, "field required")
		return
	}
	child, err := h.loadModelRaw(r, id)
	if err != nil || child.TemplateID == nil {
		respond.Error(w, http.StatusBadRequest, "not a template child")
		return
	}
	child.OverriddenFields = removeFieldFromList(child.OverriddenFields, req.Field)
	err = h.store.UpdateRequestOverriddenFields(r.Context(), sqlc.UpdateRequestOverriddenFieldsParams{
		OverriddenFields: child.OverriddenFields,
		ID:               db.PGUUID(id),
	})
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to reset field")
		return
	}
	respond.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) PromoteToTemplate(w http.ResponseWriter, r *http.Request) {
	id, _ := uuid.Parse(chi.URLParam(r, "id"))
	if err := h.store.PromoteRequestToTemplate(r.Context(), db.PGUUID(id)); err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to promote")
		return
	}
	respond.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) PushToChildren(w http.ResponseWriter, r *http.Request) {
	templateID, _ := uuid.Parse(chi.URLParam(r, "id"))
	var req struct {
		Fields []string `json:"fields"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	if len(req.Fields) == 0 {
		respond.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
		return
	}
	rows, _ := h.store.ListTemplateChildOverriddenFields(r.Context(), db.PGUUID(templateID))
	for _, row := range rows {
		fields := row.OverriddenFields
		for _, f := range req.Fields {
			fields = removeFieldFromList(fields, f)
		}
		_ = h.store.UpdateRequestOverriddenFields(r.Context(), sqlc.UpdateRequestOverriddenFieldsParams{
			OverriddenFields: fields,
			ID:               row.ID,
		})
	}
	respond.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) loadModelRaw(r *http.Request, id uuid.UUID) (Model, error) {
	row, err := h.store.GetRequest(r.Context(), db.PGUUID(id))
	if err != nil {
		return Model{}, err
	}
	return modelFromGetRequestRow(row), nil
}

func (h *Handler) loadModelMerged(r *http.Request, id uuid.UUID) (Model, error) {
	m, err := h.loadModelRaw(r, id)
	if err != nil {
		return m, err
	}
	if m.TemplateID != nil && *m.TemplateID != "" {
		tid, _ := uuid.Parse(*m.TemplateID)
		template, err := h.loadModelRaw(r, tid)
		if err == nil {
			m = mergeWithTemplate(m, template)
		}
	}
	return m, nil
}

func (h *Handler) insertModel(r *http.Request, colID uuid.UUID, m Model, userID uuid.UUID) (uuid.UUID, error) {
	headers, _ := json.Marshal(m.Headers)
	params, _ := json.Marshal(m.Params)
	pathVars, _ := json.Marshal(m.PathVars)
	body, _ := json.Marshal(m.Body)
	authSpec, _ := json.Marshal(m.Auth)
	settings, _ := json.Marshal(m.Settings)
	var templateID *uuid.UUID
	if m.TemplateID != nil && *m.TemplateID != "" {
		tid, err := uuid.Parse(*m.TemplateID)
		if err == nil {
			templateID = &tid
		}
	}
	var sourceSpecID *uuid.UUID
	if m.SourceSpecID != nil && *m.SourceSpecID != "" {
		sid, err := uuid.Parse(*m.SourceSpecID)
		if err == nil {
			sourceSpecID = &sid
		}
	}
	overridden := m.OverriddenFields
	if overridden == nil {
		overridden = []string{}
	}
	id, err := h.store.CreateRequest(r.Context(), sqlc.CreateRequestParams{
		CollectionID:      db.PGUUID(colID),
		Name:              m.Name,
		Method:            m.Method,
		Url:               m.URL,
		Headers:           headers,
		Params:            params,
		PathVars:          pathVars,
		Body:              body,
		Auth:              authSpec,
		Settings:          settings,
		PreRequestScript:  m.PreRequestScript,
		TestScript:        m.TestScript,
		SortOrder:         int32(m.SortOrder),
		Description:       m.Description,
		TemplateID:        db.PGUUIDPtr(templateID),
		IsTemplate:        m.IsTemplate,
		OverriddenFields:  overridden,
		SourceSpecID:      db.PGUUIDPtr(sourceSpecID),
		SourceOperationID: m.SourceOperationID,
		SourceOpHash:      m.SourceOpHash,
		ApiDoc:            nullRawJSON(m.ApiDoc),
		DocsOverridden:    m.DocsOverridden,
		CreatedBy:         db.PGUUID(userID),
	})
	return db.FromPGUUID(id), err
}

func modelFromGetRequestRow(row sqlc.GetRequestRow) Model {
	return modelFromRequestFields(
		row.ID, row.CollectionID, row.Name, row.Method, row.Url,
		row.Headers, row.Params, row.PathVars, row.Body, row.Auth, row.Settings, row.ApiDoc,
		row.PreRequestScript, row.TestScript, row.SortOrder, row.Description,
		row.TemplateID, row.IsTemplate, row.OverriddenFields,
		row.SourceSpecID, row.SourceOperationID, row.SourceOpHash, row.DocsOverridden,
	)
}

func modelFromListRequestsByCollectionRow(row sqlc.ListRequestsByCollectionRow) Model {
	return modelFromRequestFields(
		row.ID, row.CollectionID, row.Name, row.Method, row.Url,
		row.Headers, row.Params, row.PathVars, row.Body, row.Auth, row.Settings, row.ApiDoc,
		row.PreRequestScript, row.TestScript, row.SortOrder, row.Description,
		row.TemplateID, row.IsTemplate, row.OverriddenFields,
		row.SourceSpecID, row.SourceOperationID, row.SourceOpHash, row.DocsOverridden,
	)
}

func modelFromListRequestsByWorkspaceRow(row sqlc.ListRequestsByWorkspaceRow) Model {
	return modelFromRequestFields(
		row.ID, row.CollectionID, row.Name, row.Method, row.Url,
		row.Headers, row.Params, row.PathVars, row.Body, row.Auth, row.Settings, row.ApiDoc,
		row.PreRequestScript, row.TestScript, row.SortOrder, row.Description,
		row.TemplateID, row.IsTemplate, row.OverriddenFields,
		row.SourceSpecID, row.SourceOperationID, row.SourceOpHash, row.DocsOverridden,
	)
}

func modelFromRequestFields(
	id, colID pgtype.UUID,
	name, method, url string,
	headers, params, pathVars, body, authSpec, settings, apiDoc []byte,
	preRequestScript, testScript string,
	sortOrder int32,
	description string,
	templateID pgtype.UUID,
	isTemplate bool,
	overridden []string,
	sourceSpecID pgtype.UUID,
	sourceOperationID, sourceOpHash string,
	docsOverridden bool,
) Model {
	m := Model{
		ID:                db.FromPGUUID(id).String(),
		CollectionID:      db.FromPGUUID(colID).String(),
		Name:              name,
		Method:            method,
		URL:               url,
		PreRequestScript:  preRequestScript,
		TestScript:        testScript,
		SortOrder:         int(sortOrder),
		Description:       description,
		IsTemplate:        isTemplate,
		OverriddenFields:  overridden,
		SourceOperationID: sourceOperationID,
		SourceOpHash:      sourceOpHash,
		ApiDoc:            apiDoc,
		DocsOverridden:    docsOverridden,
	}
	_ = json.Unmarshal(headers, &m.Headers)
	_ = json.Unmarshal(params, &m.Params)
	_ = json.Unmarshal(pathVars, &m.PathVars)
	_ = json.Unmarshal(body, &m.Body)
	_ = json.Unmarshal(authSpec, &m.Auth)
	_ = json.Unmarshal(settings, &m.Settings)
	if templateID.Valid {
		s := db.FromPGUUID(templateID).String()
		m.TemplateID = &s
	}
	if sourceSpecID.Valid {
		s := db.FromPGUUID(sourceSpecID).String()
		m.SourceSpecID = &s
	}
	return m
}

func nullRawJSON(raw json.RawMessage) []byte {
	if len(raw) == 0 {
		return []byte("{}")
	}
	return raw
}
