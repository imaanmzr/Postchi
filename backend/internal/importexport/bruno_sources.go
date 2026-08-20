package importexport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/imaanmzr/postchi/backend/internal/auth"
	"github.com/imaanmzr/postchi/backend/internal/db"
	"github.com/imaanmzr/postchi/backend/internal/db/sqlc"
	"github.com/imaanmzr/postchi/backend/internal/importexport/gitsync"
	"github.com/imaanmzr/postchi/backend/internal/importexport/model"
	"github.com/imaanmzr/postchi/backend/internal/shared/gitrepo"
	"github.com/imaanmzr/postchi/backend/internal/shared/operationid"
	"github.com/imaanmzr/postchi/backend/internal/shared/respond"
)

type BrunoSource struct {
	ID             string         `json:"id"`
	WorkspaceID    string         `json:"workspace_id"`
	CollectionID   *string        `json:"collection_id,omitempty"`
	Name           string         `json:"name"`
	Config         map[string]any `json:"config"`
	HasAccessToken bool           `json:"has_access_token"`
	LastSyncedAt   *string        `json:"last_synced_at,omitempty"`
	CreatedAt      string         `json:"created_at"`
}

type BrunoSyncResult struct {
	AddedCollections   int      `json:"added_collections"`
	UpdatedCollections int      `json:"updated_collections"`
	AddedRequests      int      `json:"added_requests"`
	UpdatedRequests    int      `json:"updated_requests"`
	RemovedRequests    int      `json:"removed_requests"`
	RemovedCollections int      `json:"removed_collections"`
	Errors             []string `json:"errors,omitempty"`
}

func (h *Handler) ListBrunoSources(w http.ResponseWriter, r *http.Request) {
	wsID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid workspace id")
		return
	}
	rows, err := h.store.ListBrunoSources(r.Context(), db.PGUUID(wsID))
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "query failed")
		return
	}
	list := make([]BrunoSource, 0, len(rows))
	for _, row := range rows {
		list = append(list, mapBrunoSourceListRow(row))
	}
	respond.JSON(w, http.StatusOK, list)
}

func (h *Handler) CreateBrunoSource(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.UserIDFromContext(r.Context())
	if err != nil {
		respond.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	wsID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid workspace id")
		return
	}
	var req struct {
		Name        string         `json:"name"`
		Config      map[string]any `json:"config"`
		AccessToken string         `json:"access_token"`
		importParentRequest
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.Name) == "" {
		respond.Error(w, http.StatusBadRequest, "name is required")
		return
	}
	normalized, err := normalizeBrunoRepoConfig(req.Config)
	if err != nil {
		respond.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	parentID, err := h.resolveImportParent(r.Context(), wsID, userID, req.importParentRequest)
	if err != nil {
		respond.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	setBrunoConfigParentID(normalized, parentID)
	cfg, _ := json.Marshal(normalized)
	tokenEnc, err := h.encryptToken(req.AccessToken)
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to store credentials")
		return
	}
	sourceID, err := h.store.CreateBrunoSource(r.Context(), sqlc.CreateBrunoSourceParams{
		WorkspaceID:          db.PGUUID(wsID),
		Name:                 strings.TrimSpace(req.Name),
		Config:               cfg,
		AccessTokenEncrypted: tokenEnc,
		CreatedBy:            db.PGUUID(userID),
	})
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to create source")
		return
	}
	sourceUUID := db.FromPGUUID(sourceID)
	result, syncErr := h.syncBrunoSource(r.Context(), wsID, sourceUUID, userID)
	if syncErr != nil {
		_ = h.store.DeleteBrunoSource(r.Context(), sqlc.DeleteBrunoSourceParams{
			ID:          db.PGUUID(sourceUUID),
			WorkspaceID: db.PGUUID(wsID),
		})
		writeGitImportError(w, syncErr)
		return
	}
	row, err := h.store.GetBrunoSource(r.Context(), sqlc.GetBrunoSourceParams{
		ID:          db.PGUUID(sourceUUID),
		WorkspaceID: db.PGUUID(wsID),
	})
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "source created but fetch failed")
		return
	}
	respond.JSON(w, http.StatusCreated, map[string]any{
		"source": mapBrunoSourceGetRow(row),
		"sync":   result,
	})
}

func (h *Handler) UpdateBrunoSource(w http.ResponseWriter, r *http.Request) {
	wsID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid workspace id")
		return
	}
	sourceID, err := uuid.Parse(chi.URLParam(r, "sourceId"))
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid source id")
		return
	}
	var req struct {
		Name        string         `json:"name"`
		Config      map[string]any `json:"config"`
		AccessToken string         `json:"access_token"`
		importParentRequest
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid body")
		return
	}
	if req.Name == "" && req.Config == nil && strings.TrimSpace(req.AccessToken) == "" && req.ParentID == nil && req.CreateParent == nil {
		respond.Error(w, http.StatusBadRequest, "nothing to update")
		return
	}
	params := sqlc.UpdateBrunoSourceParams{
		ID:          db.PGUUID(sourceID),
		WorkspaceID: db.PGUUID(wsID),
	}
	if req.Name != "" {
		params.Name = &req.Name
	}
	if req.Config != nil {
		normalized, err := normalizeBrunoRepoConfig(req.Config)
		if err != nil {
			respond.Error(w, http.StatusBadRequest, err.Error())
			return
		}
		cfg, _ := json.Marshal(normalized)
		params.Config = cfg
	}
	if req.ParentID != nil || req.CreateParent != nil {
		userID, err := auth.UserIDFromContext(r.Context())
		if err != nil {
			respond.Error(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		parentID, err := h.resolveImportParent(r.Context(), wsID, userID, req.importParentRequest)
		if err != nil {
			respond.Error(w, http.StatusBadRequest, err.Error())
			return
		}
		existingConfig := decodeBrunoConfig(params.Config)
		if existingConfig == nil {
			row, rowErr := h.store.GetBrunoSource(r.Context(), sqlc.GetBrunoSourceParams{
				ID:          db.PGUUID(sourceID),
				WorkspaceID: db.PGUUID(wsID),
			})
			if rowErr == nil {
				existingConfig = decodeBrunoConfig(row.Config)
			}
		}
		if existingConfig == nil {
			existingConfig = map[string]any{}
		}
		setBrunoConfigParentID(existingConfig, parentID)
		cfg, _ := json.Marshal(existingConfig)
		params.Config = cfg
	}
	if tok := strings.TrimSpace(req.AccessToken); tok != "" {
		tokenEnc, err := h.encryptToken(tok)
		if err != nil {
			respond.Error(w, http.StatusInternalServerError, "failed to store credentials")
			return
		}
		params.AccessTokenEncrypted = tokenEnc
	}
	if err := h.store.UpdateBrunoSource(r.Context(), params); err != nil {
		respond.Error(w, http.StatusInternalServerError, "update failed")
		return
	}
	row, err := h.store.GetBrunoSource(r.Context(), sqlc.GetBrunoSourceParams{
		ID:          db.PGUUID(sourceID),
		WorkspaceID: db.PGUUID(wsID),
	})
	if err != nil {
		respond.Error(w, http.StatusNotFound, "not found")
		return
	}
	respond.JSON(w, http.StatusOK, mapBrunoSourceGetRow(row))
}

func (h *Handler) DeleteBrunoSource(w http.ResponseWriter, r *http.Request) {
	wsID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid workspace id")
		return
	}
	sourceID, err := uuid.Parse(chi.URLParam(r, "sourceId"))
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid source id")
		return
	}
	if _, err := h.store.GetBrunoSource(r.Context(), sqlc.GetBrunoSourceParams{
		ID:          db.PGUUID(sourceID),
		WorkspaceID: db.PGUUID(wsID),
	}); err != nil {
		respond.Error(w, http.StatusNotFound, "not found")
		return
	}
	if err := h.store.DeleteBrunoSyncedCollectionsBySource(r.Context(), db.PGUUID(sourceID)); err != nil {
		respond.Error(w, http.StatusInternalServerError, "delete failed")
		return
	}
	if err := h.store.DeleteBrunoSource(r.Context(), sqlc.DeleteBrunoSourceParams{
		ID:          db.PGUUID(sourceID),
		WorkspaceID: db.PGUUID(wsID),
	}); err != nil {
		respond.Error(w, http.StatusInternalServerError, "delete failed")
		return
	}
	respond.JSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *Handler) SyncBrunoSource(w http.ResponseWriter, r *http.Request) {
	h.syncCollectionSource(w, r)
}

func (h *Handler) SyncCollectionSource(w http.ResponseWriter, r *http.Request) {
	h.syncCollectionSource(w, r)
}

func (h *Handler) syncCollectionSource(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.UserIDFromContext(r.Context())
	if err != nil {
		respond.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	sourceID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid source id")
		return
	}
	row, err := h.store.GetBrunoSourceForSync(r.Context(), db.PGUUID(sourceID))
	if err != nil {
		respond.Error(w, http.StatusNotFound, "not found")
		return
	}
	wsID := db.FromPGUUID(row.WorkspaceID)
	if err := h.validateWorkspaceEditor(r.Context(), wsID, userID); err != nil {
		respond.Error(w, http.StatusForbidden, err.Error())
		return
	}
	result, syncErr := h.syncBrunoSource(r.Context(), wsID, sourceID, userID)
	if syncErr != nil {
		writeGitImportError(w, syncErr)
		return
	}
	respond.JSON(w, http.StatusOK, result)
}

func (h *Handler) syncBrunoSource(ctx context.Context, wsID, sourceID, userID uuid.UUID) (BrunoSyncResult, error) {
	mu := h.sourceSyncMutex(sourceID)
	mu.Lock()
	defer mu.Unlock()

	result := BrunoSyncResult{}
	row, err := h.store.GetBrunoSourceForSync(ctx, db.PGUUID(sourceID))
	if err != nil {
		return result, err
	}
	var config map[string]any
	if err := json.Unmarshal(row.Config, &config); err != nil {
		return result, fmt.Errorf("invalid source config")
	}
	normalized, err := normalizeBrunoRepoConfig(config)
	if err != nil {
		return result, err
	}
	cfg, _ := json.Marshal(normalized)
	_ = h.store.UpdateBrunoSource(ctx, sqlc.UpdateBrunoSourceParams{
		Config:      cfg,
		ID:          db.PGUUID(sourceID),
		WorkspaceID: db.PGUUID(wsID),
	})
	config = normalized
	parentID := parentIDFromBrunoConfig(config)

	token := ""
	if row.AccessTokenEncrypted != nil && strings.TrimSpace(*row.AccessTokenEncrypted) != "" {
		if h.crypto == nil {
			return result, fmt.Errorf("encryption not configured")
		}
		plain, err := h.crypto.Decrypt(*row.AccessTokenEncrypted)
		if err != nil {
			return result, fmt.Errorf("failed to decrypt access token")
		}
		token = plain
	}

	syncCtx, cancel := context.WithTimeout(ctx, gitBrunoImportTimeout)
	defer cancel()
	client, err := gitrepo.New(gitrepo.Config{
		RepoURL:      stringFromConfig(config, "repo_url"),
		Branch:       stringFromConfig(config, "branch"),
		PathPrefix:   stringFromConfig(config, "path_prefix"),
		Token:        token,
		MaxTreeFiles: 10_000,
		MaxFileBytes: maxGitBrunoFileBytes,
	})
	if err != nil {
		return result, err
	}
	files, err := gitsync.FetchRepository(syncCtx, client)
	if err != nil {
		return result, err
	}
	roots := gitsync.Discover(files)
	parsed := parseRepositoryRoots(roots, row.Name)
	result.Errors = parsed.Errors
	if len(parsed.Collections) == 0 {
		msg := "repository contains no importable collections"
		if len(parsed.Errors) > 0 {
			msg = strings.Join(parsed.Errors, "; ")
		}
		return result, fmt.Errorf("%s", msg)
	}

	existingCols := map[string]sqlc.ListBrunoSyncedCollectionsRow{}
	colRows, err := h.store.ListBrunoSyncedCollections(ctx, db.PGUUID(sourceID))
	if err != nil {
		return result, err
	}
	for _, colRow := range colRows {
		existingCols[colRow.SourcePath] = colRow
	}
	existingReqs := map[string]sqlc.ListBrunoSyncedRequestsRow{}
	reqRows, err := h.store.ListBrunoSyncedRequests(ctx, db.PGUUID(sourceID))
	if err != nil {
		return result, err
	}
	for _, reqRow := range reqRows {
		existingReqs[reqRow.SourcePath] = reqRow
	}

	var rootColID uuid.UUID
	var colPaths, reqPaths []string
	err = h.store.WithTx(ctx, func(q *sqlc.Queries) error {
		for i, col := range parsed.Collections {
			stats, rootID, err := h.applyBrunoTree(ctx, q, wsID, sourceID, userID, col, parentID, existingCols, existingReqs)
			if err != nil {
				return err
			}
			if i == 0 {
				rootColID = rootID
			}
			result.AddedCollections += stats.AddedCollections
			result.UpdatedCollections += stats.UpdatedCollections
			result.AddedRequests += stats.AddedRequests
			result.UpdatedRequests += stats.UpdatedRequests
			cP, rP := collectBrunoSourcePaths(col)
			colPaths = append(colPaths, cP...)
			reqPaths = append(reqPaths, rP...)
		}
		removedReqs, err := q.DeleteBrunoSyncedRequestsNotInPaths(ctx, sqlc.DeleteBrunoSyncedRequestsNotInPathsParams{
			BrunoSourceID: db.PGUUID(sourceID),
			KeepPaths:     reqPaths,
		})
		if err != nil {
			return err
		}
		result.RemovedRequests = int(removedReqs)
		removedCols, err := q.DeleteBrunoSyncedCollectionsNotInPaths(ctx, sqlc.DeleteBrunoSyncedCollectionsNotInPathsParams{
			BrunoSourceID: db.PGUUID(sourceID),
			KeepPaths:     colPaths,
		})
		if err != nil {
			return err
		}
		result.RemovedCollections = int(removedCols)
		return q.UpdateBrunoSourceLastSynced(ctx, db.PGUUID(sourceID))
	})
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return result, fmt.Errorf("git sync timed out")
		}
		return result, err
	}

	if !row.CollectionID.Valid && rootColID != uuid.Nil {
		_ = h.store.UpdateBrunoSourceCollectionID(ctx, sqlc.UpdateBrunoSourceCollectionIDParams{
			CollectionID: db.PGUUID(rootColID),
			ID:           db.PGUUID(sourceID),
		})
	}
	return result, nil
}

func (h *Handler) applyBrunoTree(
	ctx context.Context,
	q *sqlc.Queries,
	wsID, sourceID, userID uuid.UUID,
	col model.Collection,
	parentID *uuid.UUID,
	existingCols map[string]sqlc.ListBrunoSyncedCollectionsRow,
	existingReqs map[string]sqlc.ListBrunoSyncedRequestsRow,
) (BrunoSyncResult, uuid.UUID, error) {
	stats := BrunoSyncResult{}
	colID, added, err := h.upsertBrunoCollection(ctx, q, wsID, sourceID, userID, col, parentID, existingCols)
	if err != nil {
		return stats, uuid.Nil, err
	}
	if added {
		stats.AddedCollections++
	} else if col.SourcePath != "" {
		stats.UpdatedCollections++
	}
	for _, req := range col.Requests {
		addedReq, updatedReq, err := h.upsertBrunoRequest(ctx, q, colID, sourceID, userID, req, existingReqs)
		if err != nil {
			return stats, uuid.Nil, err
		}
		if addedReq {
			stats.AddedRequests++
		} else if updatedReq {
			stats.UpdatedRequests++
		}
	}
	rootID := colID
	parent := colID
	for _, child := range col.Children {
		childStats, _, err := h.applyBrunoTree(ctx, q, wsID, sourceID, userID, child, &parent, existingCols, existingReqs)
		if err != nil {
			return stats, uuid.Nil, err
		}
		stats.AddedCollections += childStats.AddedCollections
		stats.UpdatedCollections += childStats.UpdatedCollections
		stats.AddedRequests += childStats.AddedRequests
		stats.UpdatedRequests += childStats.UpdatedRequests
	}
	if col.SourcePath != "" {
		existingCols[col.SourcePath] = sqlc.ListBrunoSyncedCollectionsRow{
			ID:         db.PGUUID(colID),
			ParentID:   db.PGUUIDPtr(parentID),
			SourcePath: col.SourcePath,
			Name:       col.Name,
		}
	}
	return stats, rootID, nil
}

func (h *Handler) upsertBrunoCollection(
	ctx context.Context,
	q *sqlc.Queries,
	wsID, sourceID, userID uuid.UUID,
	col model.Collection,
	parentID *uuid.UUID,
	existing map[string]sqlc.ListBrunoSyncedCollectionsRow,
) (uuid.UUID, bool, error) {
	vars, _ := json.Marshal(col.Variables)
	headers, _ := json.Marshal(col.Headers)
	authB, _ := json.Marshal(col.Auth)
	if ex, ok := existing[col.SourcePath]; ok && col.SourcePath != "" {
		err := q.UpdateBrunoSyncedCollection(ctx, sqlc.UpdateBrunoSyncedCollectionParams{
			ID:               ex.ID,
			BrunoSourceID:    db.PGUUID(sourceID),
			Name:             col.Name,
			Description:      col.Description,
			SortOrder:        int32(col.SortOrder),
			Variables:        vars,
			Headers:          headers,
			Auth:             authB,
			PreRequestScript: col.PreRequestScript,
			TestScript:       col.TestScript,
			ParentID:         db.PGUUIDPtr(parentID),
		})
		return db.FromPGUUID(ex.ID), false, err
	}
	id, err := q.InsertBrunoSyncedCollection(ctx, sqlc.InsertBrunoSyncedCollectionParams{
		WorkspaceID:      db.PGUUID(wsID),
		ParentID:         db.PGUUIDPtr(parentID),
		Name:             col.Name,
		Description:      col.Description,
		SortOrder:        int32(col.SortOrder),
		Variables:        vars,
		Headers:          headers,
		Auth:             authB,
		PreRequestScript: col.PreRequestScript,
		TestScript:       col.TestScript,
		SourcePath:       col.SourcePath,
		BrunoSourceID:    db.PGUUID(sourceID),
		CreatedBy:        db.PGUUID(userID),
	})
	if err != nil {
		return uuid.Nil, false, err
	}
	return db.FromPGUUID(id), true, nil
}

func (h *Handler) upsertBrunoRequest(
	ctx context.Context,
	q *sqlc.Queries,
	colID, sourceID, userID uuid.UUID,
	req model.Request,
	existing map[string]sqlc.ListBrunoSyncedRequestsRow,
) (added, updated bool, err error) {
	hash := brunoRequestHash(req)
	opID := operationid.CanonicalFromMethodURL(req.Method, req.URL)
	headers, _ := json.Marshal(req.Headers)
	params, _ := json.Marshal(req.Params)
	pathVars, _ := json.Marshal(req.PathVars)
	body, _ := json.Marshal(req.Body)
	authB, _ := json.Marshal(req.Auth)
	settings, _ := json.Marshal(req.Settings)
	if ex, ok := existing[req.SourcePath]; ok {
		if ex.SourceOpHash == hash {
			return false, false, nil
		}
		err = q.UpdateBrunoSyncedRequest(ctx, sqlc.UpdateBrunoSyncedRequestParams{
			ID:                ex.ID,
			BrunoSourceID:     db.PGUUID(sourceID),
			CollectionID:      db.PGUUID(colID),
			Name:              req.Name,
			Method:            req.Method,
			Url:               req.URL,
			Headers:           headers,
			Params:            params,
			PathVars:          pathVars,
			Body:              body,
			Auth:              authB,
			PreRequestScript:  req.PreRequestScript,
			TestScript:        req.TestScript,
			SortOrder:         int32(req.SortOrder),
			Description:       req.Description,
			SourceOpHash:      hash,
			SourceOperationID: opID,
		})
		return false, true, err
	}
	err = q.InsertBrunoSyncedRequest(ctx, sqlc.InsertBrunoSyncedRequestParams{
		CollectionID:      db.PGUUID(colID),
		Name:              req.Name,
		Method:            req.Method,
		Url:               req.URL,
		Headers:           headers,
		Params:            params,
		PathVars:          pathVars,
		Body:              body,
		Auth:              authB,
		Settings:          settings,
		PreRequestScript:  req.PreRequestScript,
		TestScript:        req.TestScript,
		SortOrder:         int32(req.SortOrder),
		Description:       req.Description,
		SourcePath:        req.SourcePath,
		BrunoSourceID:     db.PGUUID(sourceID),
		SourceOperationID: opID,
		SourceOpHash:      hash,
		CreatedBy:         db.PGUUID(userID),
	})
	return true, false, err
}

func normalizeBrunoRepoConfig(config map[string]any) (map[string]any, error) {
	if config == nil {
		return nil, fmt.Errorf("repo_url is required")
	}
	out := make(map[string]any, len(config))
	for key, value := range config {
		if key != "access_token" && key != "token" {
			out[key] = value
		}
	}
	repoURL, _ := out["repo_url"].(string)
	if strings.TrimSpace(repoURL) == "" {
		return nil, fmt.Errorf("repo_url is required")
	}
	parsed, err := gitrepo.ParseURL(repoURL)
	if err != nil {
		return nil, err
	}
	out["repo_url"] = parsed.RepoURL
	out["provider"] = string(parsed.Provider)
	out["api_base_url"] = parsed.APIBase
	if parsed.FromBrowseURL {
		if parsed.Branch != "" {
			out["branch"] = parsed.Branch
		}
		out["path_prefix"] = parsed.PathPrefix
	}
	if branch, _ := out["branch"].(string); strings.TrimSpace(branch) == "" {
		out["branch"] = "main"
	}
	return out, nil
}

func stringFromConfig(config map[string]any, key string) string {
	if v, ok := config[key].(string); ok {
		return strings.TrimSpace(v)
	}
	return ""
}

func (h *Handler) encryptToken(token string) (*string, error) {
	tok := strings.TrimSpace(token)
	if tok == "" {
		return nil, nil
	}
	if h.crypto == nil {
		return nil, fmt.Errorf("encryption not configured")
	}
	enc, err := h.crypto.Encrypt(tok)
	if err != nil {
		return nil, err
	}
	return &enc, nil
}

func mapBrunoSourceListRow(row sqlc.ListBrunoSourcesRow) BrunoSource {
	return BrunoSource{
		ID:             db.FromPGUUID(row.ID).String(),
		WorkspaceID:    db.FromPGUUID(row.WorkspaceID).String(),
		Name:           row.Name,
		Config:         decodeBrunoConfig(row.Config),
		HasAccessToken: row.AccessTokenEncrypted != nil && strings.TrimSpace(*row.AccessTokenEncrypted) != "",
		CreatedAt:      formatBrunoTime(row.CreatedAt),
		CollectionID:   pgUUIDToStringPtr(row.CollectionID),
		LastSyncedAt:   pgTimeToStringPtr(row.LastSyncedAt),
	}
}

func mapBrunoSourceGetRow(row sqlc.GetBrunoSourceRow) BrunoSource {
	return BrunoSource{
		ID:             db.FromPGUUID(row.ID).String(),
		WorkspaceID:    db.FromPGUUID(row.WorkspaceID).String(),
		Name:           row.Name,
		Config:         decodeBrunoConfig(row.Config),
		HasAccessToken: row.AccessTokenEncrypted != nil && strings.TrimSpace(*row.AccessTokenEncrypted) != "",
		CreatedAt:      formatBrunoTime(row.CreatedAt),
		CollectionID:   pgUUIDToStringPtr(row.CollectionID),
		LastSyncedAt:   pgTimeToStringPtr(row.LastSyncedAt),
	}
}

func decodeBrunoConfig(raw []byte) map[string]any {
	var config map[string]any
	_ = json.Unmarshal(raw, &config)
	return config
}

func pgUUIDToStringPtr(id pgtype.UUID) *string {
	if !id.Valid {
		return nil
	}
	s := db.FromPGUUID(id).String()
	return &s
}

func pgTimeToStringPtr(ts pgtype.Timestamptz) *string {
	if !ts.Valid {
		return nil
	}
	s := formatBrunoTime(ts)
	return &s
}

func formatBrunoTime(ts pgtype.Timestamptz) string {
	if !ts.Valid {
		return ""
	}
	return ts.Time.UTC().Format(time.RFC3339)
}
