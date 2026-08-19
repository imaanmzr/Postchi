package share

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/imaanmzr/postchi/backend/internal/auth"
	"github.com/imaanmzr/postchi/backend/internal/db"
	"github.com/imaanmzr/postchi/backend/internal/db/sqlc"
	"github.com/imaanmzr/postchi/backend/internal/request"
	"github.com/imaanmzr/postchi/backend/internal/shared/config"
	"github.com/imaanmzr/postchi/backend/internal/shared/respond"
)

var (
	ErrNotFound = errors.New("share not found")
	ErrGone     = errors.New("share unavailable")
)

type Handler struct {
	store *db.Store
	cfg   *config.Config
}

func NewHandler(store *db.Store, cfg *config.Config) *Handler {
	return &Handler{store: store, cfg: cfg}
}

type Share struct {
	ID          string         `json:"id"`
	WorkspaceID string         `json:"workspace_id"`
	Kind        string         `json:"kind"`
	SourceID    string         `json:"source_id"`
	Token       string         `json:"token"`
	Title       string         `json:"title"`
	Snapshot    map[string]any `json:"snapshot"`
	Visibility  string         `json:"visibility"`
	ExpiresAt   *string        `json:"expires_at,omitempty"`
	CreatedBy   string         `json:"created_by"`
	CreatedAt   string         `json:"created_at"`
	ShareURL    string         `json:"share_url,omitempty"`
}

func newToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.UserIDFromContext(r.Context())
	if err != nil {
		respond.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req struct {
		Kind             string `json:"kind"`
		SourceID         string `json:"source_id"`
		WorkspaceID      string `json:"workspace_id"`
		LandingRequestID string `json:"landing_request_id"`
		Title            string `json:"title"`
		Visibility       string `json:"visibility"`
		TTLHours         *int   `json:"ttl_hours"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Kind == "" || req.SourceID == "" || req.WorkspaceID == "" {
		respond.Error(w, http.StatusBadRequest, "kind, source_id, and workspace_id required")
		return
	}
	if req.Kind != "request" && req.Kind != "history" && req.Kind != "catalog" {
		respond.Error(w, http.StatusBadRequest, "invalid kind")
		return
	}
	if req.Visibility == "" {
		req.Visibility = "workspace"
	}
	if req.Visibility != "workspace" && req.Visibility != "link" {
		respond.Error(w, http.StatusBadRequest, "invalid visibility")
		return
	}
	wsID, err := uuid.Parse(req.WorkspaceID)
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid workspace_id")
		return
	}
	if !h.hasMinRole(r.Context(), wsID, userID, "editor") {
		respond.Error(w, http.StatusForbidden, "insufficient permissions")
		return
	}
	sourceID, err := uuid.Parse(req.SourceID)
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid source_id")
		return
	}
	landingRequestID := uuid.Nil
	if req.LandingRequestID != "" {
		if req.Kind != "catalog" {
			respond.Error(w, http.StatusBadRequest, "landing_request_id is only valid for catalog shares")
			return
		}
		landingRequestID, err = uuid.Parse(req.LandingRequestID)
		if err != nil {
			respond.Error(w, http.StatusBadRequest, "invalid landing_request_id")
			return
		}
	}

	snapshot, title, err := h.buildSnapshot(r.Context(), req.Kind, sourceID, wsID, landingRequestID)
	if err != nil {
		respond.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.Title != "" {
		title = req.Title
	}

	token, err := newToken()
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to create token")
		return
	}
	var expiresAt pgtype.Timestamptz
	if req.TTLHours != nil && *req.TTLHours > 0 {
		t := time.Now().Add(time.Duration(*req.TTLHours) * time.Hour)
		expiresAt = db.PGTimestamptz(t)
	}
	snapJSON, _ := json.Marshal(snapshot)

	shareID, err := h.store.CreateShare(r.Context(), sqlc.CreateShareParams{
		WorkspaceID: db.PGUUID(wsID),
		Kind:        sqlc.ShareKind(req.Kind),
		SourceID:    db.PGUUID(sourceID),
		Token:       token,
		Title:       title,
		Snapshot:    snapJSON,
		Visibility:  req.Visibility,
		ExpiresAt:   expiresAt,
		CreatedBy:   db.PGUUID(userID),
	})
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to create share")
		return
	}

	shareURL := strings.TrimRight(h.cfg.AppPublicURL, "/") + "/share/" + token
	respond.JSON(w, http.StatusCreated, Share{
		ID:          db.FromPGUUID(shareID).String(),
		WorkspaceID: wsID.String(),
		Kind:        req.Kind,
		SourceID:    sourceID.String(),
		Token:       token,
		Title:       title,
		Snapshot:    snapshot,
		Visibility:  req.Visibility,
		ExpiresAt:   formatPGTimePtr(expiresAt),
		CreatedBy:   userID.String(),
		ShareURL:    shareURL,
	})
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	wsID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid workspace id")
		return
	}
	rows, err := h.store.ListActiveShares(r.Context(), db.PGUUID(wsID))
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to list shares")
		return
	}
	list := make([]Share, 0, len(rows))
	for _, row := range rows {
		s := Share{
			ID:          db.FromPGUUID(row.ID).String(),
			WorkspaceID: db.FromPGUUID(row.WorkspaceID).String(),
			Kind:        row.Kind,
			SourceID:    db.FromPGUUID(row.SourceID).String(),
			Token:       row.Token,
			Title:       row.Title,
			Visibility:  row.Visibility,
			ExpiresAt:   formatPGTimePtr(row.ExpiresAt),
			CreatedBy:   db.FromPGUUID(row.CreatedBy).String(),
			CreatedAt:   row.CreatedAt.Time.Format(time.RFC3339),
			ShareURL:    strings.TrimRight(h.cfg.AppPublicURL, "/") + "/share/" + row.Token,
		}
		list = append(list, s)
	}
	respond.JSON(w, http.StatusOK, list)
}

func (h *Handler) Revoke(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.UserIDFromContext(r.Context())
	if err != nil {
		respond.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	shareID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	meta, err := h.store.GetShareWorkspaceAndCreator(r.Context(), db.PGUUID(shareID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			respond.Error(w, http.StatusNotFound, "not found")
			return
		}
		respond.Error(w, http.StatusInternalServerError, "failed to load share")
		return
	}
	wsID := db.FromPGUUID(meta.WorkspaceID)
	createdBy := db.FromPGUUID(meta.CreatedBy)
	if createdBy != userID && !h.hasMinRole(r.Context(), wsID, userID, "editor") {
		respond.Error(w, http.StatusForbidden, "insufficient permissions")
		return
	}
	_ = h.store.RevokeShare(r.Context(), db.PGUUID(shareID))
	respond.JSON(w, http.StatusOK, map[string]string{"status": "revoked"})
}

func (h *Handler) GetByToken(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	row, err := h.loadByToken(r.Context(), token)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			respond.Error(w, http.StatusNotFound, "share not found")
			return
		}
		if errors.Is(err, ErrGone) {
			respond.Error(w, http.StatusGone, "this share is no longer available")
			return
		}
		respond.Error(w, http.StatusInternalServerError, "failed to load share")
		return
	}
	if row.Visibility == "workspace" {
		userID, err := auth.UserIDFromContext(r.Context())
		if err != nil {
			respond.Error(w, http.StatusUnauthorized, "authentication required")
			return
		}
		if !h.hasMinRole(r.Context(), row.WorkspaceID, userID, "viewer") {
			respond.Error(w, http.StatusForbidden, "not a workspace member")
			return
		}
	}
	var snapshot map[string]any
	_ = json.Unmarshal(row.Snapshot, &snapshot)
	respond.JSON(w, http.StatusOK, Share{
		ID:          row.ID.String(),
		WorkspaceID: row.WorkspaceID.String(),
		Kind:        row.Kind,
		SourceID:    row.SourceID.String(),
		Token:       row.Token,
		Title:       row.Title,
		Snapshot:    snapshot,
		Visibility:  row.Visibility,
		ExpiresAt:   formatTimePtr(row.ExpiresAt),
		CreatedBy:   row.CreatedBy.String(),
		CreatedAt:   row.CreatedAt.Format(time.RFC3339),
	})
}

func (h *Handler) Import(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.UserIDFromContext(r.Context())
	if err != nil {
		respond.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	token := chi.URLParam(r, "token")
	row, err := h.loadByToken(r.Context(), token)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			respond.Error(w, http.StatusNotFound, "share not found")
			return
		}
		if errors.Is(err, ErrGone) {
			respond.Error(w, http.StatusGone, "this share is no longer available")
			return
		}
		respond.Error(w, http.StatusInternalServerError, "failed to load share")
		return
	}
	if row.Visibility == "workspace" {
		if !h.hasMinRole(r.Context(), row.WorkspaceID, userID, "viewer") {
			respond.Error(w, http.StatusForbidden, "not a workspace member")
			return
		}
	}
	if row.Kind == "catalog" {
		respond.Error(w, http.StatusBadRequest, "catalog shares are read-only")
		return
	}
	var req struct {
		WorkspaceID         string `json:"workspace_id"`
		CollectionID        string `json:"collection_id"`
		TargetEnvironmentID string `json:"target_environment_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.WorkspaceID == "" || req.CollectionID == "" {
		respond.Error(w, http.StatusBadRequest, "workspace_id and collection_id required")
		return
	}
	wsID, err := uuid.Parse(req.WorkspaceID)
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid workspace_id")
		return
	}
	if !h.hasMinRole(r.Context(), wsID, userID, "editor") {
		respond.Error(w, http.StatusForbidden, "insufficient permissions")
		return
	}
	colID, err := uuid.Parse(req.CollectionID)
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid collection_id")
		return
	}

	var snapshot map[string]any
	_ = json.Unmarshal(row.Snapshot, &snapshot)

	var reqModel request.Model
	if row.Kind == "history" {
		snapPart, ok := snapshot["snapshot"]
		if !ok {
			respond.Error(w, http.StatusBadRequest, "invalid history snapshot")
			return
		}
		b, _ := json.Marshal(snapPart)
		_ = json.Unmarshal(b, &reqModel)
	} else {
		b, _ := json.Marshal(snapshot)
		_ = json.Unmarshal(b, &reqModel)
	}
	if reqModel.Name == "" {
		reqModel.Name = row.Title
	}
	if reqModel.Name == "" {
		reqModel.Name = "Imported request"
	}

	headers, _ := json.Marshal(reqModel.Headers)
	params, _ := json.Marshal(reqModel.Params)
	pathVars, _ := json.Marshal(reqModel.PathVars)
	body, _ := json.Marshal(reqModel.Body)
	authSpec, _ := json.Marshal(reqModel.Auth)
	settings, _ := json.Marshal(reqModel.Settings)

	newID, err := h.store.ImportSharedRequest(r.Context(), sqlc.ImportSharedRequestParams{
		CollectionID:     db.PGUUID(colID),
		Name:             reqModel.Name,
		Method:           reqModel.Method,
		Url:              reqModel.URL,
		Headers:          headers,
		Params:           params,
		PathVars:         pathVars,
		Body:             body,
		Auth:             authSpec,
		Settings:         settings,
		PreRequestScript: reqModel.PreRequestScript,
		TestScript:       reqModel.TestScript,
		SortOrder:        int32(reqModel.SortOrder),
		Description:      reqModel.Description,
		CreatedBy:        db.PGUUID(userID),
	})
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to import request")
		return
	}
	newUUID := db.FromPGUUID(newID)

	if row.Kind == "history" {
		if resp, ok := snapshot["response"]; ok {
			respJSON, _ := json.Marshal(resp)
			_ = h.store.CreateSharedResponseExample(r.Context(), sqlc.CreateSharedResponseExampleParams{
				RequestID: db.PGUUID(newUUID),
				Name:      "Shared response",
				Response:  respJSON,
				CreatedBy: db.PGUUID(userID),
			})
		}
	}

	respond.JSON(w, http.StatusCreated, map[string]string{
		"id":             newUUID.String(),
		"request_id":     newUUID.String(),
		"collection_id":  colID.String(),
		"workspace_id":   wsID.String(),
		"environment_id": req.TargetEnvironmentID,
	})
}

type shareRow struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	Kind        string
	SourceID    uuid.UUID
	Token       string
	Title       string
	Snapshot    []byte
	Visibility  string
	ExpiresAt   *time.Time
	RevokedAt   *time.Time
	CreatedBy   uuid.UUID
	CreatedAt   time.Time
}

func (h *Handler) loadByToken(ctx context.Context, token string) (*shareRow, error) {
	row, err := h.store.GetShareByToken(ctx, token)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if row.RevokedAt.Valid {
		return nil, ErrGone
	}
	var expiresAt *time.Time
	if row.ExpiresAt.Valid {
		t := row.ExpiresAt.Time
		expiresAt = &t
		if time.Now().After(t) {
			return nil, ErrGone
		}
	}
	return &shareRow{
		ID:          db.FromPGUUID(row.ID),
		WorkspaceID: db.FromPGUUID(row.WorkspaceID),
		Kind:        row.Kind,
		SourceID:    db.FromPGUUID(row.SourceID),
		Token:       row.Token,
		Title:       row.Title,
		Snapshot:    row.Snapshot,
		Visibility:  row.Visibility,
		ExpiresAt:   expiresAt,
		CreatedBy:   db.FromPGUUID(row.CreatedBy),
		CreatedAt:   row.CreatedAt.Time,
	}, nil
}

func (h *Handler) buildSnapshot(ctx context.Context, kind string, sourceID, wsID, landingRequestID uuid.UUID) (map[string]any, string, error) {
	if kind == "catalog" {
		if sourceID == wsID {
			collections, err := h.store.ListCatalogCollections(ctx, db.PGUUID(wsID))
			if err != nil {
				return nil, "", err
			}
			collectionNames := make(map[string]string, len(collections))
			snapshotCollections := make([]map[string]any, 0, len(collections))
			for _, col := range collections {
				id := db.FromPGUUID(col.ID).String()
				collectionNames[id] = col.Name
				snapshotCollections = append(snapshotCollections, map[string]any{
					"id":          id,
					"name":        col.Name,
					"description": col.Description,
				})
			}

			rows, err := h.store.ListRequestsByWorkspace(ctx, db.PGUUID(wsID))
			if err != nil {
				return nil, "", err
			}
			endpoints := make([]map[string]any, 0, len(rows))
			landingRequestFound := landingRequestID == uuid.Nil
			for _, r := range rows {
				requestID := db.FromPGUUID(r.ID)
				if requestID == landingRequestID {
					landingRequestFound = true
				}
				collectionID := db.FromPGUUID(r.CollectionID).String()
				var doc any
				_ = json.Unmarshal(r.ApiDoc, &doc)
				endpoints = append(endpoints, map[string]any{
					"id":              requestID.String(),
					"collection_id":   collectionID,
					"collection_name": collectionNames[collectionID],
					"name":            r.Name,
					"method":          r.Method,
					"url":             r.Url,
					"description":     r.Description,
					"api_doc":         doc,
				})
			}
			if !landingRequestFound {
				return nil, "", errors.New("landing request not found in catalog")
			}
			snapshot := map[string]any{
				"collections": snapshotCollections,
				"endpoints":   endpoints,
			}
			if landingRequestID != uuid.Nil {
				snapshot["landing_request_id"] = landingRequestID.String()
			}
			return snapshot, "Workspace API Catalog", nil
		}

		col, err := h.store.GetCollectionForCatalogShare(ctx, sqlc.GetCollectionForCatalogShareParams{
			ID:          db.PGUUID(sourceID),
			WorkspaceID: db.PGUUID(wsID),
		})
		if err != nil {
			return nil, "", errors.New("collection not found")
		}
		rows, err := h.store.ListRequestsForCatalogShare(ctx, db.PGUUID(sourceID))
		if err != nil {
			return nil, "", err
		}
		var endpoints []map[string]any
		landingRequestFound := landingRequestID == uuid.Nil
		for _, r := range rows {
			requestID := db.FromPGUUID(r.ID)
			if requestID == landingRequestID {
				landingRequestFound = true
			}
			ep := map[string]any{
				"id": requestID.String(), "name": r.Name, "method": r.Method,
				"url": r.Url, "description": r.Description,
			}
			var doc any
			_ = json.Unmarshal(r.ApiDoc, &doc)
			ep["api_doc"] = doc
			endpoints = append(endpoints, ep)
		}
		if !landingRequestFound {
			return nil, "", errors.New("landing request not found in catalog")
		}
		snapshot := map[string]any{
			"collection": map[string]any{
				"id": sourceID.String(), "name": col.Name, "description": col.Description,
			},
			"endpoints": endpoints,
		}
		if landingRequestID != uuid.Nil {
			snapshot["landing_request_id"] = landingRequestID.String()
		}
		return snapshot, col.Name + " API Catalog", nil
	}
	if kind == "request" {
		row, err := h.store.GetRequestForShareSnapshot(ctx, sqlc.GetRequestForShareSnapshotParams{
			ID:          db.PGUUID(sourceID),
			WorkspaceID: db.PGUUID(wsID),
		})
		if err != nil {
			return nil, "", errors.New("request not found")
		}
		m := requestModelFromShareRow(row)
		b, _ := json.Marshal(m)
		var snap map[string]any
		_ = json.Unmarshal(b, &snap)
		return snap, m.Name, nil
	}
	row, err := h.store.GetHistoryForShareSnapshot(ctx, sqlc.GetHistoryForShareSnapshotParams{
		ID:          db.PGUUID(sourceID),
		WorkspaceID: db.PGUUID(wsID),
	})
	if err != nil {
		return nil, "", errors.New("history entry not found")
	}
	var reqSnap, resp, testResults any
	_ = json.Unmarshal(row.Snapshot, &reqSnap)
	_ = json.Unmarshal(row.Response, &resp)
	if len(row.TestResults) > 0 {
		_ = json.Unmarshal(row.TestResults, &testResults)
	}
	snapshot := map[string]any{
		"snapshot":          reqSnap,
		"response":          resp,
		"test_results":      testResults,
		"duration_ms":       row.DurationMs,
		"status_code":       row.StatusCode,
		"executed_at":       row.ExecutedAt.Time.Format(time.RFC3339),
		"executed_by":       db.FromPGUUID(row.ExecutedBy).String(),
		"executed_by_name":  row.DisplayName,
		"executed_by_email": row.Email,
	}
	title := "Shared execution"
	if m, ok := reqSnap.(map[string]any); ok {
		if n, ok := m["name"].(string); ok && n != "" {
			title = n
		}
	}
	return snapshot, title, nil
}

func requestModelFromShareRow(row sqlc.GetRequestForShareSnapshotRow) request.Model {
	m := request.Model{
		Name:             row.Name,
		Method:           row.Method,
		URL:              row.Url,
		PreRequestScript: row.PreRequestScript,
		TestScript:       row.TestScript,
		SortOrder:        int(row.SortOrder),
		Description:      row.Description,
	}
	m.ID = db.FromPGUUID(row.ID).String()
	m.CollectionID = db.FromPGUUID(row.CollectionID).String()
	_ = json.Unmarshal(row.Headers, &m.Headers)
	_ = json.Unmarshal(row.Params, &m.Params)
	_ = json.Unmarshal(row.PathVars, &m.PathVars)
	_ = json.Unmarshal(row.Body, &m.Body)
	_ = json.Unmarshal(row.Auth, &m.Auth)
	_ = json.Unmarshal(row.Settings, &m.Settings)
	return m
}

func (h *Handler) hasMinRole(ctx context.Context, wsID, userID uuid.UUID, minRole string) bool {
	role, err := h.store.GetWorkspaceMemberRole(ctx, sqlc.GetWorkspaceMemberRoleParams{
		WorkspaceID: db.PGUUID(wsID),
		UserID:      db.PGUUID(userID),
	})
	if err != nil {
		return false
	}
	ranks := map[string]int{"viewer": 1, "editor": 2, "owner": 3}
	return ranks[role] >= ranks[minRole]
}

func formatTimePtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format(time.RFC3339)
	return &s
}

func formatPGTimePtr(t pgtype.Timestamptz) *string {
	if !t.Valid {
		return nil
	}
	s := t.Time.Format(time.RFC3339)
	return &s
}
