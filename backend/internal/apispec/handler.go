package apispec

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/imaanmzr/postchi/backend/internal/auth"
	"github.com/imaanmzr/postchi/backend/internal/db"
	"github.com/imaanmzr/postchi/backend/internal/db/sqlc"
	openapiimport "github.com/imaanmzr/postchi/backend/internal/importexport/openapi"
	"github.com/imaanmzr/postchi/backend/internal/request"
	"github.com/imaanmzr/postchi/backend/internal/shared/config"
	"github.com/imaanmzr/postchi/backend/internal/shared/respond"
)

type Handler struct {
	store *db.Store
	cfg   *config.Config
}

func NewHandler(store *db.Store, cfg *config.Config) *Handler {
	return &Handler{store: store, cfg: cfg}
}

type ApiSpec struct {
	ID           string  `json:"id"`
	WorkspaceID  string  `json:"workspace_id"`
	CollectionID *string `json:"collection_id,omitempty"`
	Name         string  `json:"name"`
	SourceType   string  `json:"source_type"`
	SpecURL      string  `json:"spec_url"`
	SpecHash     string  `json:"spec_hash"`
	BaseURLVar   string  `json:"base_url_var"`
	LastSyncedAt *string `json:"last_synced_at,omitempty"`
	CreatedAt    string  `json:"created_at"`
	UpdatedAt    string  `json:"updated_at"`
}

type EnvURL struct {
	EnvironmentID string `json:"environment_id"`
	BaseURL       string `json:"base_url"`
}

type SyncDiff struct {
	Added   []SyncItem `json:"added"`
	Updated []SyncItem `json:"updated"`
	Removed []SyncItem `json:"removed"`
}

type SyncItem struct {
	OperationID string `json:"operation_id"`
	Name        string `json:"name"`
	Method      string `json:"method"`
	Path        string `json:"path"`
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())
	wsID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid workspace id")
		return
	}
	var req struct {
		Name         string `json:"name"`
		SpecURL      string `json:"spec_url"`
		CollectionID string `json:"collection_id"`
		BaseURLVar   string `json:"base_url_var"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" || req.SpecURL == "" {
		respond.Error(w, http.StatusBadRequest, "name and spec_url required")
		return
	}
	if req.BaseURLVar == "" {
		req.BaseURLVar = "baseUrl"
	}
	data, err := h.fetchSpecURL(req.SpecURL)
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "failed to fetch spec: "+err.Error())
		return
	}
	parsed, err := openapiimport.ParseWithHash(data, req.Name)
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "failed to parse spec: "+err.Error())
		return
	}
	var colID *uuid.UUID
	if req.CollectionID != "" {
		cid, err := uuid.Parse(req.CollectionID)
		if err == nil {
			colID = &cid
		}
	}
	if colID == nil {
		newColPg, err := h.store.CreateSpecCollection(r.Context(), sqlc.CreateSpecCollectionParams{
			WorkspaceID: pgUUID(wsID),
			Name:        req.Name,
			CreatedBy:   pgUUID(userID),
		})
		if err != nil {
			respond.Error(w, http.StatusInternalServerError, "failed to create collection")
			return
		}
		cid := uuidFromPg(newColPg)
		colID = &cid
	}
	specIDPg, err := h.store.CreateApiSpec(r.Context(), sqlc.CreateApiSpecParams{
		WorkspaceID:  pgUUID(wsID),
		CollectionID: pgUUIDPtr(colID),
		Name:         req.Name,
		SpecUrl:      req.SpecURL,
		SpecHash:     parsed.SpecHash,
		BaseUrlVar:   req.BaseURLVar,
		CreatedBy:    pgUUID(userID),
	})
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to create api spec")
		return
	}
	specID := uuidFromPg(specIDPg)
	specRow := &specRow{
		ID:           specID,
		WorkspaceID:  wsID,
		CollectionID: colID,
		Name:         req.Name,
		SpecURL:      req.SpecURL,
		SpecHash:     parsed.SpecHash,
		BaseURLVar:   req.BaseURLVar,
	}
	diff, err := h.computeDiff(r.Context(), specRow, parsed)
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to compute initial sync")
		return
	}
	if err := h.applyDiff(r.Context(), specRow, parsed, diff, userID); err != nil {
		_ = h.store.DeleteApiSpec(r.Context(), pgUUID(specID))
		respond.Error(w, http.StatusInternalServerError, "failed to import endpoints: "+err.Error())
		return
	}
	_ = h.store.UpdateApiSpecLastSynced(r.Context(), pgUUID(specID))
	spec, _ := h.load(r.Context(), specID)
	respond.JSON(w, http.StatusCreated, spec)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	wsID, _ := uuid.Parse(chi.URLParam(r, "id"))
	rows, err := h.store.ListApiSpecs(r.Context(), pgUUID(wsID))
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "query failed")
		return
	}
	list := make([]ApiSpec, 0, len(rows))
	for _, row := range rows {
		list = append(list, apiSpecFromListRow(row))
	}
	respond.JSON(w, http.StatusOK, list)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, _ := uuid.Parse(chi.URLParam(r, "id"))
	spec, err := h.load(r.Context(), id)
	if err != nil {
		respond.Error(w, http.StatusNotFound, "not found")
		return
	}
	respond.JSON(w, http.StatusOK, spec)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, _ := uuid.Parse(chi.URLParam(r, "id"))
	var req struct {
		Name       string `json:"name"`
		SpecURL    string `json:"spec_url"`
		BaseURLVar string `json:"base_url_var"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	idPg := pgUUID(id)
	if req.Name != "" {
		_ = h.store.UpdateApiSpecName(r.Context(), sqlc.UpdateApiSpecNameParams{Name: req.Name, ID: idPg})
	}
	if req.SpecURL != "" {
		_ = h.store.UpdateApiSpecURL(r.Context(), sqlc.UpdateApiSpecURLParams{SpecUrl: req.SpecURL, ID: idPg})
	}
	if req.BaseURLVar != "" {
		_ = h.store.UpdateApiSpecBaseURLVar(r.Context(), sqlc.UpdateApiSpecBaseURLVarParams{BaseUrlVar: req.BaseURLVar, ID: idPg})
	}
	spec, _ := h.load(r.Context(), id)
	respond.JSON(w, http.StatusOK, spec)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, _ := uuid.Parse(chi.URLParam(r, "id"))
	_ = h.store.DeleteApiSpec(r.Context(), pgUUID(id))
	respond.JSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *Handler) SetEnvironmentURLs(w http.ResponseWriter, r *http.Request) {
	id, _ := uuid.Parse(chi.URLParam(r, "id"))
	var items []EnvURL
	if err := json.NewDecoder(r.Body).Decode(&items); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid body")
		return
	}
	for _, item := range items {
		envID, err := uuid.Parse(item.EnvironmentID)
		if err != nil {
			continue
		}
		_ = h.store.UpsertApiSpecEnvironmentURL(r.Context(), sqlc.UpsertApiSpecEnvironmentURLParams{
			ApiSpecID:     pgUUID(id),
			EnvironmentID: pgUUID(envID),
			BaseUrl:       item.BaseURL,
		})
	}
	respond.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) Sync(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())
	id, _ := uuid.Parse(chi.URLParam(r, "id"))
	var req struct {
		Apply bool `json:"apply"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	spec, err := h.loadRow(r.Context(), id)
	if err != nil {
		respond.Error(w, http.StatusNotFound, "not found")
		return
	}
	data, err := h.fetchSpecData(r.Context(), spec)
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "failed to fetch spec: "+err.Error())
		return
	}
	parsed, err := openapiimport.ParseWithHash(data, spec.Name)
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "failed to parse spec: "+err.Error())
		return
	}
	diff, err := h.computeDiff(r.Context(), spec, parsed)
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "diff failed")
		return
	}
	if !req.Apply {
		respond.JSON(w, http.StatusOK, diff)
		return
	}
	if err := h.applyDiff(r.Context(), spec, parsed, diff, userID); err != nil {
		respond.Error(w, http.StatusInternalServerError, "apply failed: "+err.Error())
		return
	}
	_ = h.store.UpdateApiSpecAfterSync(r.Context(), sqlc.UpdateApiSpecAfterSyncParams{
		SpecHash: parsed.SpecHash,
		ID:       pgUUID(id),
	})
	respond.JSON(w, http.StatusOK, diff)
}

func (h *Handler) fetchSpecData(ctx context.Context, spec *specRow) ([]byte, error) {
	if spec.SourceType == "upload" || spec.SourceType == "push" {
		if len(spec.SpecContent) > 0 {
			return spec.SpecContent, nil
		}
		return nil, fmt.Errorf("no stored spec content")
	}
	return h.fetchSpecURL(spec.SpecURL)
}

func (h *Handler) fetchSpecURL(url string) ([]byte, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

type specRow struct {
	ID           uuid.UUID
	WorkspaceID  uuid.UUID
	CollectionID *uuid.UUID
	Name         string
	SourceType   string
	SpecURL      string
	SpecHash     string
	BaseURLVar   string
	SpecContent  []byte
}

func (h *Handler) loadRow(ctx context.Context, id uuid.UUID) (*specRow, error) {
	row, err := h.store.GetApiSpecRow(ctx, pgUUID(id))
	if err != nil {
		return nil, err
	}
	return specRowFromQuery(row), nil
}

func (h *Handler) load(ctx context.Context, id uuid.UUID) (ApiSpec, error) {
	row, err := h.store.GetApiSpec(ctx, pgUUID(id))
	if err != nil {
		return ApiSpec{}, err
	}
	return apiSpecFromGetRow(row), nil
}

func (h *Handler) computeDiff(ctx context.Context, spec *specRow, parsed openapiimport.ParseResult) (SyncDiff, error) {
	diff := SyncDiff{
		Added:   []SyncItem{},
		Updated: []SyncItem{},
		Removed: []SyncItem{},
	}
	existing := map[string]struct {
		id   uuid.UUID
		hash string
		name string
	}{}
	rows, _ := h.store.ListSyncedRequestsBySpec(ctx, pgUUID(spec.ID))
	for _, row := range rows {
		existing[row.SourceOperationID] = struct {
			id   uuid.UUID
			hash string
			name string
		}{id: uuidFromPg(row.ID), hash: row.SourceOpHash, name: row.Name}
	}
	seen := map[string]bool{}
	for _, op := range parsed.Operations {
		seen[op.OperationID] = true
		path := strings.TrimPrefix(op.Request.URL, "{{baseUrl}}")
		item := SyncItem{OperationID: op.OperationID, Name: op.Request.Name, Method: op.Request.Method, Path: path}
		if ex, ok := existing[op.OperationID]; !ok {
			diff.Added = append(diff.Added, item)
		} else if ex.hash != op.OpHash {
			diff.Updated = append(diff.Updated, item)
		}
	}
	for opID, ex := range existing {
		if !seen[opID] {
			diff.Removed = append(diff.Removed, SyncItem{OperationID: opID, Name: ex.name})
		}
	}
	return diff, nil
}

func (h *Handler) ensureSpecCollection(ctx context.Context, spec *specRow, userID uuid.UUID) error {
	if spec.CollectionID != nil {
		exists, err := h.store.CollectionExists(ctx, pgUUID(*spec.CollectionID))
		if err == nil && exists {
			return nil
		}
	}
	newColPg, err := h.store.CreateSpecCollection(ctx, sqlc.CreateSpecCollectionParams{
		WorkspaceID: pgUUID(spec.WorkspaceID),
		Name:        spec.Name,
		CreatedBy:   pgUUID(userID),
	})
	if err != nil {
		return fmt.Errorf("failed to create collection: %w", err)
	}
	newCol := uuidFromPg(newColPg)
	if err := h.store.UpdateApiSpecCollectionID(ctx, sqlc.UpdateApiSpecCollectionIDParams{
		CollectionID: pgUUID(newCol),
		ID:           pgUUID(spec.ID),
	}); err != nil {
		return fmt.Errorf("failed to link collection: %w", err)
	}
	spec.CollectionID = &newCol
	return nil
}

func (h *Handler) applyDiff(ctx context.Context, spec *specRow, parsed openapiimport.ParseResult, diff SyncDiff, userID uuid.UUID) error {
	if err := h.ensureSpecCollection(ctx, spec, userID); err != nil {
		return err
	}
	colID := *spec.CollectionID
	return h.store.WithTx(ctx, func(q *sqlc.Queries) error {
		opMap := map[string]openapiimport.Operation{}
		for _, op := range parsed.Operations {
			opMap[op.OperationID] = op
		}
		for _, item := range diff.Added {
			op := opMap[item.OperationID]
			if err := insertSyncedRequest(ctx, q, colID, spec.ID, op, userID); err != nil {
				return err
			}
		}
		for _, item := range diff.Updated {
			op := opMap[item.OperationID]
			headers, _ := json.Marshal(op.Request.Headers)
			params, _ := json.Marshal(op.Request.Params)
			pathVars, _ := json.Marshal(op.Request.PathVars)
			body, _ := json.Marshal(op.Request.Body)
			apiDoc := op.ApiDoc
			if len(apiDoc) == 0 {
				apiDoc = []byte("{}")
			}
			if err := q.UpdateSyncedRequest(ctx, sqlc.UpdateSyncedRequestParams{
				Method:            op.Request.Method,
				Url:               op.Request.URL,
				Headers:           headers,
				Params:            params,
				PathVars:          pathVars,
				SourceOpHash:      op.OpHash,
				ApiDoc:            apiDoc,
				Description:       op.Request.Description,
				Body:              body,
				SourceSpecID:      pgUUID(spec.ID),
				SourceOperationID: item.OperationID,
			}); err != nil {
				return err
			}
		}
		return nil
	})
}

func insertSyncedRequest(ctx context.Context, q *sqlc.Queries, colID, specID uuid.UUID, op openapiimport.Operation, userID uuid.UUID) error {
	return request.InsertSyncedRequest(ctx, q, request.SyncInsertParams{
		CollectionID: colID,
		SpecID:       specID,
		UserID:       userID,
		Name:         op.Request.Name,
		Method:       op.Request.Method,
		URL:          op.Request.URL,
		Headers:      toKVPairs(op.Request.Headers),
		Params:       toKVPairs(op.Request.Params),
		PathVars:     toKVPairs(op.Request.PathVars),
		Body:         op.Request.Body,
		Auth:         op.Request.Auth,
		Settings:     op.Request.Settings,
		SortOrder:    op.Request.SortOrder,
		Description:  op.Request.Description,
		OperationID:  op.OperationID,
		OpHash:       op.OpHash,
		ApiDoc:       op.ApiDoc,
	})
}

func toKVPairs(pairs []request.KVPair) []request.KVPair {
	if pairs == nil {
		return []request.KVPair{}
	}
	return pairs
}

func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())
	wsID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid workspace id")
		return
	}
	data, err := io.ReadAll(r.Body)
	if err != nil || len(data) == 0 {
		respond.Error(w, http.StatusBadRequest, "spec body required")
		return
	}
	name := r.URL.Query().Get("name")
	collectionID := r.URL.Query().Get("collection_id")
	parsed, err := openapiimport.ParseWithHash(data, name)
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "failed to parse spec: "+err.Error())
		return
	}
	if name == "" {
		name = parsed.Collection.Name
	}
	colID, err := h.ensureCollection(r.Context(), wsID, collectionID, name, userID)
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	specIDPg, err := h.store.CreateUploadedApiSpec(r.Context(), sqlc.CreateUploadedApiSpecParams{
		WorkspaceID:  pgUUID(wsID),
		CollectionID: pgUUIDPtr(colID),
		Name:         name,
		SpecHash:     parsed.SpecHash,
		SpecContent:  data,
		CreatedBy:    pgUUID(userID),
	})
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to create spec")
		return
	}
	specID := uuidFromPg(specIDPg)
	specRow := &specRow{ID: specID, WorkspaceID: wsID, CollectionID: colID, Name: name, SourceType: "upload", SpecHash: parsed.SpecHash, SpecContent: data}
	diff, _ := h.computeDiff(r.Context(), specRow, parsed)
	_ = h.applyDiff(r.Context(), specRow, parsed, diff, userID)
	_ = h.store.UpdateApiSpecLastSyncedOnly(r.Context(), pgUUID(specID))
	spec, _ := h.load(r.Context(), specID)
	respond.JSON(w, http.StatusCreated, spec)
}

func (h *Handler) Reupload(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())
	id, _ := uuid.Parse(chi.URLParam(r, "id"))
	data, err := io.ReadAll(r.Body)
	if err != nil || len(data) == 0 {
		respond.Error(w, http.StatusBadRequest, "spec body required")
		return
	}
	spec, err := h.loadRow(r.Context(), id)
	if err != nil {
		respond.Error(w, http.StatusNotFound, "not found")
		return
	}
	parsed, err := openapiimport.ParseWithHash(data, spec.Name)
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "failed to parse spec: "+err.Error())
		return
	}
	diff, _ := h.computeDiff(r.Context(), spec, parsed)
	apply := r.URL.Query().Get("apply") == "true"
	if !apply {
		respond.JSON(w, http.StatusOK, diff)
		return
	}
	_ = h.store.UpdateApiSpecReupload(r.Context(), sqlc.UpdateApiSpecReuploadParams{
		SpecContent: data,
		SpecHash:    parsed.SpecHash,
		ID:          pgUUID(id),
	})
	spec.SpecContent = data
	spec.SpecHash = parsed.SpecHash
	if err := h.applyDiff(r.Context(), spec, parsed, diff, userID); err != nil {
		respond.Error(w, http.StatusInternalServerError, "apply failed")
		return
	}
	_ = h.store.UpdateApiSpecLastSyncedOnly(r.Context(), pgUUID(id))
	respond.JSON(w, http.StatusOK, diff)
}

func (h *Handler) Push(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())
	wsID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid workspace id")
		return
	}
	data, err := io.ReadAll(r.Body)
	if err != nil || len(data) == 0 {
		respond.Error(w, http.StatusBadRequest, "spec body required")
		return
	}
	name := r.URL.Query().Get("name")
	if name == "" {
		name = "Pushed API Spec"
	}
	collectionID := r.URL.Query().Get("collection_id")
	apply := r.URL.Query().Get("apply") != "false"
	parsed, err := openapiimport.ParseWithHash(data, name)
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "failed to parse spec: "+err.Error())
		return
	}
	specIDPg, err := h.store.GetApiSpecIDByWorkspaceAndName(r.Context(), sqlc.GetApiSpecIDByWorkspaceAndNameParams{
		WorkspaceID: pgUUID(wsID),
		Name:        name,
	})
	var specID uuid.UUID
	if err != nil {
		colID, colErr := h.ensureCollection(r.Context(), wsID, collectionID, name, userID)
		if colErr != nil {
			respond.Error(w, http.StatusInternalServerError, colErr.Error())
			return
		}
		newSpecIDPg, createErr := h.store.CreatePushedApiSpec(r.Context(), sqlc.CreatePushedApiSpecParams{
			WorkspaceID:  pgUUID(wsID),
			CollectionID: pgUUIDPtr(colID),
			Name:         name,
			SpecHash:     parsed.SpecHash,
			SpecContent:  data,
			CreatedBy:    pgUUID(userID),
		})
		if createErr != nil {
			respond.Error(w, http.StatusInternalServerError, "failed to create spec")
			return
		}
		specID = uuidFromPg(newSpecIDPg)
	} else {
		specID = uuidFromPg(specIDPg)
		_ = h.store.UpdatePushedApiSpec(r.Context(), sqlc.UpdatePushedApiSpecParams{
			SpecContent: data,
			SpecHash:    parsed.SpecHash,
			ID:          pgUUID(specID),
		})
	}
	specRow, _ := h.loadRow(r.Context(), specID)
	diff, _ := h.computeDiff(r.Context(), specRow, parsed)
	if apply {
		_ = h.applyDiff(r.Context(), specRow, parsed, diff, userID)
		_ = h.store.UpdateApiSpecLastSyncedOnly(r.Context(), pgUUID(specID))
	}
	respond.JSON(w, http.StatusOK, diff)
}

func (h *Handler) ensureCollection(ctx context.Context, wsID uuid.UUID, collectionID, name string, userID uuid.UUID) (*uuid.UUID, error) {
	if collectionID != "" {
		cid, err := uuid.Parse(collectionID)
		if err == nil {
			return &cid, nil
		}
	}
	newColPg, err := h.store.CreateSpecCollection(ctx, sqlc.CreateSpecCollectionParams{
		WorkspaceID: pgUUID(wsID),
		Name:        name,
		CreatedBy:   pgUUID(userID),
	})
	if err != nil {
		return nil, err
	}
	cid := uuidFromPg(newColPg)
	return &cid, nil
}

func pgUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}

func pgUUIDPtr(id *uuid.UUID) pgtype.UUID {
	if id == nil {
		return pgtype.UUID{}
	}
	return pgtype.UUID{Bytes: *id, Valid: true}
}

func uuidFromPg(id pgtype.UUID) uuid.UUID {
	if !id.Valid {
		return uuid.Nil
	}
	return uuid.UUID(id.Bytes)
}

func uuidPtrFromPg(id pgtype.UUID) *uuid.UUID {
	if !id.Valid {
		return nil
	}
	u := uuid.UUID(id.Bytes)
	return &u
}

func specRowFromQuery(row sqlc.GetApiSpecRowRow) *specRow {
	return &specRow{
		ID:           uuidFromPg(row.ID),
		WorkspaceID:  uuidFromPg(row.WorkspaceID),
		CollectionID: uuidPtrFromPg(row.CollectionID),
		Name:         row.Name,
		SourceType:   row.SourceType,
		SpecURL:      row.SpecUrl,
		SpecHash:     row.SpecHash,
		BaseURLVar:   row.BaseUrlVar,
		SpecContent:  row.SpecContent,
	}
}

func apiSpecFromGetRow(row sqlc.GetApiSpecRow) ApiSpec {
	return apiSpecFromFields(row.ID, row.WorkspaceID, row.CollectionID, row.Name, row.SourceType, row.SpecUrl, row.SpecHash, row.BaseUrlVar, row.LastSyncedAt, row.CreatedAt, row.UpdatedAt)
}

func apiSpecFromListRow(row sqlc.ListApiSpecsRow) ApiSpec {
	return apiSpecFromFields(row.ID, row.WorkspaceID, row.CollectionID, row.Name, row.SourceType, row.SpecUrl, row.SpecHash, row.BaseUrlVar, row.LastSyncedAt, row.CreatedAt, row.UpdatedAt)
}

func apiSpecFromFields(id, wsID, colID pgtype.UUID, name, sourceType, specURL, specHash, baseURLVar string, lastSynced, created, updated pgtype.Timestamptz) ApiSpec {
	s := ApiSpec{
		ID:          uuidFromPg(id).String(),
		WorkspaceID: uuidFromPg(wsID).String(),
		Name:        name,
		SourceType:  sourceType,
		SpecURL:     specURL,
		SpecHash:    specHash,
		BaseURLVar:  baseURLVar,
	}
	if colID.Valid {
		c := uuidFromPg(colID).String()
		s.CollectionID = &c
	}
	if lastSynced.Valid {
		t := lastSynced.Time.Format(time.RFC3339)
		s.LastSyncedAt = &t
	}
	if created.Valid {
		s.CreatedAt = created.Time.Format(time.RFC3339)
	}
	if updated.Valid {
		s.UpdatedAt = updated.Time.Format(time.RFC3339)
	}
	return s
}
