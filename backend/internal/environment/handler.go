package environment

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/imaanmzr/postchi/backend/internal/auth"
	"github.com/imaanmzr/postchi/backend/internal/db"
	"github.com/imaanmzr/postchi/backend/internal/db/sqlc"
	"github.com/imaanmzr/postchi/backend/internal/shared/crypto"
	"github.com/imaanmzr/postchi/backend/internal/shared/respond"
)

type Handler struct {
	store  *db.Store
	crypto *crypto.Service
}

func NewHandler(store *db.Store, cryptoSvc *crypto.Service) *Handler {
	return &Handler{store: store, crypto: cryptoSvc}
}

type Environment struct {
	ID          string     `json:"id"`
	WorkspaceID string     `json:"workspace_id"`
	Name        string     `json:"name"`
	Stage       string     `json:"stage"`
	Variables   []Variable `json:"variables,omitempty"`
}

var validStages = map[string]bool{
	"local": true, "dev": true, "uat": true, "staging": true, "prod": true, "custom": true,
}

func normalizeStage(stage string) string {
	if stage == "" || !validStages[stage] {
		return "custom"
	}
	return stage
}

type Variable struct {
	ID          string `json:"id,omitempty"`
	Key         string `json:"key"`
	Value       string `json:"value"`
	Expr        string `json:"expr,omitempty"`
	Phase       string `json:"phase"`
	Enabled     bool   `json:"enabled"`
	Type        string `json:"type"`
	Description string `json:"description"`
	IsSecret    bool   `json:"is_secret"`
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	wsID, err := uuid.Parse(r.URL.Query().Get("workspace_id"))
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "workspace_id required")
		return
	}
	rows, err := h.store.ListEnvironments(r.Context(), db.PGUUID(wsID))
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "query failed")
		return
	}
	list := make([]Environment, 0, len(rows))
	for _, row := range rows {
		list = append(list, Environment{
			ID:          db.FromPGUUID(row.ID).String(),
			WorkspaceID: db.FromPGUUID(row.WorkspaceID).String(),
			Name:        row.Name,
			Stage:       row.Stage,
		})
	}
	respond.JSON(w, http.StatusOK, list)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, _ := uuid.Parse(chi.URLParam(r, "id"))
	e, err := h.load(r, id, true)
	if err != nil {
		respond.Error(w, http.StatusNotFound, "not found")
		return
	}
	respond.JSON(w, http.StatusOK, e)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())
	var req Environment
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" || req.WorkspaceID == "" {
		respond.Error(w, http.StatusBadRequest, "invalid request")
		return
	}
	wsID, _ := uuid.Parse(req.WorkspaceID)
	stage := normalizeStage(req.Stage)
	id, err := h.store.CreateEnvironment(r.Context(), sqlc.CreateEnvironmentParams{
		WorkspaceID: db.PGUUID(wsID),
		Name:        req.Name,
		Stage:       stage,
		CreatedBy:   db.PGUUID(userID),
	})
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "create failed")
		return
	}
	envID := db.FromPGUUID(id)
	h.replaceVars(r, envID, req.Variables)
	e, _ := h.load(r, envID, true)
	respond.JSON(w, http.StatusCreated, e)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, _ := uuid.Parse(chi.URLParam(r, "id"))
	var req Environment
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid body")
		return
	}
	pgID := db.PGUUID(id)
	if req.Name != "" {
		_ = h.store.UpdateEnvironmentName(r.Context(), sqlc.UpdateEnvironmentNameParams{Name: req.Name, ID: pgID})
	}
	if req.Stage != "" {
		_ = h.store.UpdateEnvironmentStage(r.Context(), sqlc.UpdateEnvironmentStageParams{Stage: normalizeStage(req.Stage), ID: pgID})
	}
	if req.Variables != nil {
		h.replaceVars(r, id, req.Variables)
	}
	e, _ := h.load(r, id, true)
	respond.JSON(w, http.StatusOK, e)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, _ := uuid.Parse(chi.URLParam(r, "id"))
	_ = h.store.DeleteEnvironment(r.Context(), db.PGUUID(id))
	respond.JSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *Handler) replaceVars(r *http.Request, envID uuid.UUID, vars []Variable) {
	_ = h.store.DeleteEnvironmentVariables(r.Context(), db.PGUUID(envID))
	for _, v := range vars {
		if v.Key == "" {
			continue
		}
		h.upsertVar(r, envID, v)
	}
}

func (h *Handler) upsertVar(r *http.Request, envID uuid.UUID, v Variable) {
	phase := v.Phase
	if phase == "" {
		phase = "pre_request"
	}
	enc := v.Value
	if v.IsSecret && v.Value != "" && !strings.HasPrefix(v.Value, "****") {
		var err error
		enc, err = h.crypto.Encrypt(v.Value)
		if err != nil {
			return
		}
	}
	typ := v.Type
	if typ == "" {
		typ = "string"
	}
	_ = h.store.UpsertEnvironmentVariable(r.Context(), sqlc.UpsertEnvironmentVariableParams{
		EnvironmentID:  db.PGUUID(envID),
		Key:            v.Key,
		ValueEncrypted: enc,
		IsSecret:       v.IsSecret,
		Phase:          phase,
		Enabled:        v.Enabled,
		Type:           typ,
		Description:    v.Description,
		Expr:           v.Expr,
	})
}

func (h *Handler) load(r *http.Request, id uuid.UUID, maskSecrets bool) (Environment, error) {
	var e Environment
	row, err := h.store.GetEnvironment(r.Context(), db.PGUUID(id))
	if err != nil {
		return e, err
	}
	e.ID = db.FromPGUUID(row.ID).String()
	e.WorkspaceID = db.FromPGUUID(row.WorkspaceID).String()
	e.Name = row.Name
	e.Stage = row.Stage

	rows, err := h.store.ListEnvironmentVariables(r.Context(), db.PGUUID(id))
	if err != nil {
		return e, err
	}
	for _, row := range rows {
		v := Variable{
			ID:          db.FromPGUUID(row.ID).String(),
			Key:         row.Key,
			IsSecret:    row.IsSecret,
			Phase:       row.Phase,
			Enabled:     row.Enabled,
			Type:        row.Type,
			Description: row.Description,
			Expr:        row.Expr,
		}
		if v.IsSecret && maskSecrets {
			plain, err := h.crypto.Decrypt(row.ValueEncrypted)
			if err == nil {
				v.Value = crypto.MaskSecret(plain)
			} else {
				v.Value = "****"
			}
		} else if v.IsSecret {
			v.Value, _ = h.crypto.Decrypt(row.ValueEncrypted)
		} else {
			v.Value = row.ValueEncrypted
		}
		e.Variables = append(e.Variables, v)
	}
	return e, nil
}

func (h *Handler) DecryptVariables(r *http.Request, envID uuid.UUID) (map[string]string, []Variable) {
	out := map[string]string{}
	var postRows []Variable
	rows, _ := h.store.ListEnvironmentVariablesForDecrypt(r.Context(), db.PGUUID(envID))
	for _, row := range rows {
		if !row.Enabled {
			continue
		}
		if row.Phase == "post_response" {
			postRows = append(postRows, Variable{Key: row.Key, Expr: row.Expr, Phase: row.Phase, Enabled: row.Enabled})
			continue
		}
		if row.IsSecret {
			if plain, err := h.crypto.Decrypt(row.ValueEncrypted); err == nil {
				out[row.Key] = plain
			}
		} else {
			out[row.Key] = row.ValueEncrypted
		}
	}
	return out, postRows
}

func (h *Handler) ResolveVariables(w http.ResponseWriter, r *http.Request) {
	id, _ := uuid.Parse(chi.URLParam(r, "id"))
	var req struct {
		Names []string `json:"names"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid body")
		return
	}
	existingMap, _ := h.DecryptVariables(r, id)
	existing := []string{}
	missing := []string{}
	for _, name := range req.Names {
		if _, ok := existingMap[name]; ok {
			existing = append(existing, name)
		} else {
			missing = append(missing, name)
		}
	}
	respond.JSON(w, http.StatusOK, map[string][]string{
		"existing": existing,
		"missing":  missing,
	})
}

func (h *Handler) BulkSetVariables(w http.ResponseWriter, r *http.Request) {
	id, _ := uuid.Parse(chi.URLParam(r, "id"))
	var vars []Variable
	if err := json.NewDecoder(r.Body).Decode(&vars); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid body")
		return
	}
	for _, v := range vars {
		if v.Key == "" {
			continue
		}
		h.upsertVar(r, id, v)
	}
	e, _ := h.load(r, id, true)
	respond.JSON(w, http.StatusOK, e)
}
