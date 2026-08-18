package request

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/imaanmzr/postchi/backend/internal/auth"
	"github.com/imaanmzr/postchi/backend/internal/db"
	"github.com/imaanmzr/postchi/backend/internal/db/sqlc"
	"github.com/imaanmzr/postchi/backend/internal/shared/config"
	"github.com/imaanmzr/postchi/backend/internal/shared/crypto"
	"github.com/imaanmzr/postchi/backend/internal/shared/respond"
)

type Handler struct {
	store    *db.Store
	executor *Executor
	cfg      *config.Config
}

func NewHandler(store *db.Store, cfg *config.Config, cryptoSvc *crypto.Service) *Handler {
	return &Handler{store: store, executor: NewExecutor(cfg, store, cryptoSvc), cfg: cfg}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())
	var req Model
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" || req.CollectionID == "" {
		respond.Error(w, http.StatusBadRequest, "invalid request")
		return
	}
	colID, _ := uuid.Parse(req.CollectionID)
	id, err := h.insertModel(r, colID, req, userID)
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to create request")
		return
	}
	req.ID = id.String()
	respond.JSON(w, http.StatusCreated, req)
}

func (h *Handler) loadModel(r *http.Request, id uuid.UUID) (Model, error) {
	return h.loadModelMerged(r, id)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	existing, err := h.loadModelRaw(r, id)
	if err != nil {
		respond.Error(w, http.StatusNotFound, "not found")
		return
	}
	var req Model
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid body")
		return
	}
	overridden := existing.OverriddenFields
	if existing.TemplateID != nil && *existing.TemplateID != "" {
		tid, _ := uuid.Parse(*existing.TemplateID)
		template, err := h.loadModelRaw(r, tid)
		if err == nil {
			overridden = diffOverriddenFields(existing, req, template)
		}
	}
	headers, _ := json.Marshal(req.Headers)
	params, _ := json.Marshal(req.Params)
	pathVars, _ := json.Marshal(req.PathVars)
	body, _ := json.Marshal(req.Body)
	authSpec, _ := json.Marshal(req.Auth)
	settings, _ := json.Marshal(req.Settings)
	err = h.store.UpdateRequest(r.Context(), sqlc.UpdateRequestParams{
		Name:             req.Name,
		Method:           req.Method,
		Url:              req.URL,
		Headers:          headers,
		Params:           params,
		PathVars:         pathVars,
		Body:             body,
		Auth:             authSpec,
		Settings:         settings,
		PreRequestScript: req.PreRequestScript,
		TestScript:       req.TestScript,
		SortOrder:        int32(req.SortOrder),
		Description:      req.Description,
		OverriddenFields: overridden,
		ApiDoc:           nullRawJSON(req.ApiDoc),
		DocsOverridden:   req.DocsOverridden,
		ID:               db.PGUUID(id),
	})
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to update")
		return
	}
	req.ID = id.String()
	req.OverriddenFields = overridden
	respond.JSON(w, http.StatusOK, req)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, _ := uuid.Parse(chi.URLParam(r, "id"))
	template, err := h.loadModelRaw(r, id)
	if err == nil && template.IsTemplate {
		childIDs, _ := h.store.ListRequestIDsByTemplate(r.Context(), db.PGUUID(id))
		for _, childIDpg := range childIDs {
			childID := db.FromPGUUID(childIDpg)
			child, err := h.loadModelRaw(r, childID)
			if err != nil {
				continue
			}
			tpl, err := h.loadModelRaw(r, id)
			if err != nil {
				continue
			}
			merged := snapshotMergedChild(child, tpl)
			headers, _ := json.Marshal(merged.Headers)
			params, _ := json.Marshal(merged.Params)
			pathVars, _ := json.Marshal(merged.PathVars)
			body, _ := json.Marshal(merged.Body)
			authSpec, _ := json.Marshal(merged.Auth)
			settings, _ := json.Marshal(merged.Settings)
			_ = h.store.SnapshotTemplateChild(r.Context(), sqlc.SnapshotTemplateChildParams{
				Method:           merged.Method,
				Url:              merged.URL,
				Headers:          headers,
				Params:           params,
				PathVars:         pathVars,
				Body:             body,
				Auth:             authSpec,
				Settings:         settings,
				PreRequestScript: merged.PreRequestScript,
				TestScript:       merged.TestScript,
				OverriddenFields: []string{},
				ID:               db.PGUUID(childID),
			})
		}
	}
	_ = h.store.DeleteRequest(r.Context(), db.PGUUID(id))
	respond.JSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, _ := uuid.Parse(chi.URLParam(r, "id"))
	req, err := h.loadModelMerged(r, id)
	if err != nil {
		respond.Error(w, http.StatusNotFound, "not found")
		return
	}
	respond.JSON(w, http.StatusOK, req)
}

func (h *Handler) ListByCollection(w http.ResponseWriter, r *http.Request) {
	colID, err := uuid.Parse(r.URL.Query().Get("collection_id"))
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "collection_id required")
		return
	}
	rows, err := h.store.ListRequestsByCollection(r.Context(), db.PGUUID(colID))
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "query failed")
		return
	}
	list := make([]Model, 0, len(rows))
	for _, row := range rows {
		list = append(list, modelFromListRequestsByCollectionRow(row))
	}
	respond.JSON(w, http.StatusOK, list)
}

func (h *Handler) ListByWorkspace(w http.ResponseWriter, r *http.Request) {
	wsID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid workspace id")
		return
	}
	rows, err := h.store.ListRequestsByWorkspace(r.Context(), db.PGUUID(wsID))
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "query failed")
		return
	}
	list := make([]Model, 0, len(rows))
	for _, row := range rows {
		list = append(list, modelFromListRequestsByWorkspaceRow(row))
	}
	respond.JSON(w, http.StatusOK, list)
}

func (h *Handler) Duplicate(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())
	id, _ := uuid.Parse(chi.URLParam(r, "id"))
	req, err := h.loadModelMerged(r, id)
	if err != nil {
		respond.Error(w, http.StatusNotFound, "not found")
		return
	}
	req.Name += " (copy)"
	req.TemplateID = nil
	req.IsTemplate = false
	req.OverriddenFields = nil
	req.SourceSpecID = nil
	req.SourceOperationID = ""
	req.SourceOpHash = ""
	colID, _ := uuid.Parse(req.CollectionID)
	newID, err := h.insertModel(r, colID, req, userID)
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "duplicate failed")
		return
	}
	req.ID = newID.String()
	respond.JSON(w, http.StatusCreated, req)
}

func (h *Handler) Execute(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())
	id, _ := uuid.Parse(chi.URLParam(r, "id"))
	req, err := h.loadModelMerged(r, id)
	if err != nil {
		respond.Error(w, http.StatusNotFound, "not found")
		return
	}

	var execReq struct {
		EnvironmentID string            `json:"environment_id"`
		Variables     map[string]string `json:"variables"`
		Request       *Model            `json:"request"`
	}
	_ = json.NewDecoder(r.Body).Decode(&execReq)

	if execReq.Request != nil {
		override := *execReq.Request
		override.ID = req.ID
		override.CollectionID = req.CollectionID
		override.TemplateID = req.TemplateID
		override.OverriddenFields = req.OverriddenFields
		req = override
	}

	wsID, colID, inheritedPre, inheritedTest := h.getCollectionContext(r, uuid.MustParse(req.CollectionID))
	envID := uuid.Nil
	if execReq.EnvironmentID != "" {
		envID, _ = uuid.Parse(execReq.EnvironmentID)
	}
	vars := h.executor.BuildVariablesForRequest(r.Context(), wsID, colID, envID, execReq.Variables, req)

	result, err := h.executor.Execute(r.Context(), req, vars, inheritedPre, inheritedTest)
	if err != nil {
		respond.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	snapshot, _ := json.Marshal(req)
	response, _ := json.Marshal(result)
	testResults, _ := json.Marshal(result.TestResults)
	historyID, _ := h.store.CreateHistoryEntry(r.Context(), sqlc.CreateHistoryEntryParams{
		WorkspaceID: db.PGUUID(wsID),
		RequestID:   db.PGUUID(id),
		Snapshot:    snapshot,
		Response:    response,
		TestResults: testResults,
		ExecutedBy:  db.PGUUID(userID),
		DurationMs:  result.Timing.Total,
		StatusCode:  int32(result.StatusCode),
	})
	result.HistoryID = db.FromPGUUID(historyID).String()
	respond.JSON(w, http.StatusOK, result)
}

func (h *Handler) Snippet(w http.ResponseWriter, r *http.Request) {
	id, _ := uuid.Parse(chi.URLParam(r, "id"))
	req, err := h.loadModelMerged(r, id)
	if err != nil {
		respond.Error(w, http.StatusNotFound, "not found")
		return
	}
	lang := r.URL.Query().Get("lang")
	if lang == "" {
		lang = "curl"
	}
	vars := map[string]string{}
	if r.URL.Query().Get("interpolate") != "false" {
		wsID, colID, _, _ := h.getCollectionContext(r, uuid.MustParse(req.CollectionID))
		envID := uuid.Nil
		if eid := r.URL.Query().Get("environment_id"); eid != "" {
			envID, _ = uuid.Parse(eid)
		}
		vars = h.executor.BuildVariablesForRequest(r.Context(), wsID, colID, envID, nil, req)
	}
	snippetReq := req
	snippetReq.Auth = h.executor.ResolveAuth(r.Context(), req.CollectionID, req.Auth)
	respond.JSON(w, http.StatusOK, map[string]string{"snippet": GenerateSnippet(snippetReq, lang, vars)})
}

func (h *Handler) SaveExample(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())
	id, _ := uuid.Parse(chi.URLParam(r, "id"))
	var req struct {
		Name     string `json:"name"`
		Response any    `json:"response"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	resp, _ := json.Marshal(req.Response)
	exID, _ := h.store.CreateExample(r.Context(), sqlc.CreateExampleParams{
		RequestID: db.PGUUID(id),
		Name:      req.Name,
		Response:  resp,
		CreatedBy: db.PGUUID(userID),
	})
	respond.JSON(w, http.StatusCreated, map[string]string{"id": db.FromPGUUID(exID).String()})
}

func (h *Handler) getCollectionContext(r *http.Request, colID uuid.UUID) (wsID, collectionID uuid.UUID, pre, test string) {
	chain := h.collectionAncestorIDs(r.Context(), colID)
	if len(chain) == 0 {
		return uuid.Nil, colID, "", ""
	}
	var pres, tests []string
	for _, id := range chain {
		ctxRow, err := h.store.GetCollectionContext(r.Context(), db.PGUUID(id))
		if err != nil {
			continue
		}
		if wsID == uuid.Nil {
			wsID = db.FromPGUUID(ctxRow.WorkspaceID)
		}
		if ctxRow.PreRequestScript != "" {
			pres = append(pres, ctxRow.PreRequestScript)
		}
		if ctxRow.TestScript != "" {
			tests = append(tests, ctxRow.TestScript)
		}
	}
	return wsID, colID, strings.Join(pres, "\n"), strings.Join(tests, "\n")
}

func (h *Handler) collectionAncestorIDs(ctx context.Context, collectionID uuid.UUID) []uuid.UUID {
	var chain []uuid.UUID
	seen := make(map[uuid.UUID]bool)
	cur := collectionID
	for {
		if seen[cur] {
			break
		}
		seen[cur] = true
		parentPg, err := h.store.GetCollectionParentID(ctx, db.PGUUID(cur))
		if err != nil {
			break
		}
		chain = append([]uuid.UUID{cur}, chain...)
		if !parentPg.Valid {
			break
		}
		cur = db.FromPGUUID(parentPg)
	}
	return chain
}

func (h *Handler) Move(w http.ResponseWriter, r *http.Request) {
	id, _ := uuid.Parse(chi.URLParam(r, "id"))
	var req struct {
		CollectionID string `json:"collection_id"`
		SortOrder    int    `json:"sort_order"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.CollectionID == "" {
		respond.Error(w, http.StatusBadRequest, "collection_id required")
		return
	}
	colID, _ := uuid.Parse(req.CollectionID)
	err := h.store.MoveRequest(r.Context(), sqlc.MoveRequestParams{
		CollectionID: db.PGUUID(colID),
		SortOrder:    int32(req.SortOrder),
		ID:           db.PGUUID(id),
	})
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "move failed")
		return
	}
	respond.JSON(w, http.StatusOK, map[string]string{"status": "moved"})
}

func (h *Handler) Reorder(w http.ResponseWriter, r *http.Request) {
	var items []struct {
		ID        string `json:"id"`
		SortOrder int    `json:"sort_order"`
	}
	if err := json.NewDecoder(r.Body).Decode(&items); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid body")
		return
	}
	for _, item := range items {
		rid, _ := uuid.Parse(item.ID)
		_ = h.store.UpdateRequestSortOrder(r.Context(), sqlc.UpdateRequestSortOrderParams{
			SortOrder: int32(item.SortOrder),
			ID:        db.PGUUID(rid),
		})
	}
	respond.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Runner orchestrates collection execution
type Runner struct {
	handler *Handler
	hub     interface{ Broadcast(workspaceID string, msg any) }
}

func NewRunner(h *Handler) *Runner {
	return &Runner{handler: h}
}

func (h *Handler) RunCollection(w http.ResponseWriter, r *http.Request) {
	colID, _ := uuid.Parse(chi.URLParam(r, "id"))
	var req struct {
		EnvironmentID string              `json:"environment_id"`
		DataRows      []map[string]string `json:"data_rows"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	type runResult struct {
		RequestID string `json:"request_id"`
		Name      string `json:"name"`
		Passed    bool   `json:"passed"`
		Status    int    `json:"status_code"`
		Error     string `json:"error,omitempty"`
	}
	var results []runResult
	var passed, failed int

	dataRows := req.DataRows
	if len(dataRows) == 0 {
		dataRows = []map[string]string{{}}
	}

	for _, dataRow := range dataRows {
		requests, _ := h.store.ListRequestIDAndNameByCollection(r.Context(), db.PGUUID(colID))
		for _, row := range requests {
			reqID := db.FromPGUUID(row.ID)
			model, err := h.loadModelMerged(r, reqID)
			if err != nil {
				continue
			}
			reqColID, _ := uuid.Parse(model.CollectionID)
			wsID, _, pre, test := h.getCollectionContext(r, reqColID)
			envID := uuid.Nil
			if req.EnvironmentID != "" {
				envID, _ = uuid.Parse(req.EnvironmentID)
			}
			vars := h.executor.BuildVariablesForRequest(r.Context(), wsID, reqColID, envID, dataRow, model)
			result, _ := h.executor.Execute(r.Context(), model, vars, pre, test)
			ok := result.Error == ""
			for _, t := range result.TestResults {
				if !t.Passed {
					ok = false
				}
			}
			if ok {
				passed++
			} else {
				failed++
			}
			results = append(results, runResult{
				RequestID: reqID.String(),
				Name:      row.Name,
				Passed:    ok,
				Status:    result.StatusCode,
				Error:     result.Error,
			})
		}
	}

	respond.JSON(w, http.StatusOK, map[string]any{
		"passed":  passed,
		"failed":  failed,
		"total":   passed + failed,
		"results": results,
	})
}
