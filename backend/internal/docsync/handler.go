package docsync

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

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

type DocSource struct {
	ID             string         `json:"id"`
	WorkspaceID    string         `json:"workspace_id"`
	CollectionID   *string        `json:"collection_id,omitempty"`
	Name           string         `json:"name"`
	SourceType     string         `json:"source_type"`
	Config         map[string]any `json:"config"`
	HasAccessToken bool           `json:"has_access_token"`
	LastSyncedAt   *string        `json:"last_synced_at,omitempty"`
	CreatedAt      string         `json:"created_at"`
}

type WorkspaceDoc struct {
	ID                 string   `json:"id"`
	WorkspaceID        string   `json:"workspace_id"`
	Slug               string   `json:"slug"`
	Title              string   `json:"title"`
	ContentMD          string   `json:"content_md,omitempty"`
	SourcePath         string   `json:"source_path"`
	IsLocal            bool     `json:"is_local"`
	LinkedOperationIDs []string `json:"linked_operation_ids,omitempty"`
	UpdatedAt          string   `json:"updated_at"`
}

type WorkspaceDocSummary struct {
	ID          string `json:"id"`
	WorkspaceID string `json:"workspace_id"`
	Slug        string `json:"slug"`
	Title       string `json:"title"`
	SourcePath  string `json:"source_path"`
	IsLocal     bool   `json:"is_local"`
	UpdatedAt   string `json:"updated_at"`
}

func (h *Handler) ListSources(w http.ResponseWriter, r *http.Request) {
	wsID, _ := uuid.Parse(chi.URLParam(r, "id"))
	rows, err := h.store.ListDocSources(r.Context(), db.PGUUID(wsID))
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "query failed")
		return
	}
	list := make([]DocSource, 0, len(rows))
	for _, row := range rows {
		s := mapDocSourceRow(row)
		list = append(list, s)
	}
	respond.JSON(w, http.StatusOK, list)
}

func (h *Handler) CreateSource(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())
	wsID, _ := uuid.Parse(chi.URLParam(r, "id"))
	var req struct {
		Name         string         `json:"name"`
		SourceType   string         `json:"source_type"`
		CollectionID string         `json:"collection_id"`
		Config       map[string]any `json:"config"`
		AccessToken  string         `json:"access_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" || req.SourceType == "" {
		respond.Error(w, http.StatusBadRequest, "name and source_type required")
		return
	}
	colID := pgtype.UUID{}
	if req.CollectionID != "" {
		if cid, err := uuid.Parse(req.CollectionID); err == nil {
			colID = db.PGUUID(cid)
		}
	}
	normalized, err := normalizeRepoConfig(req.Config)
	if err != nil {
		respond.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	cfg, _ := json.Marshal(normalized)
	var tokenEnc *string
	if tok := strings.TrimSpace(req.AccessToken); tok != "" {
		enc, err := h.crypto.Encrypt(tok)
		if err != nil {
			respond.Error(w, http.StatusInternalServerError, "failed to store credentials")
			return
		}
		tokenEnc = &enc
	}
	id, err := h.store.CreateDocSource(r.Context(), sqlc.CreateDocSourceParams{
		WorkspaceID:          db.PGUUID(wsID),
		CollectionID:         colID,
		Name:                 req.Name,
		SourceType:           req.SourceType,
		Config:               cfg,
		AccessTokenEncrypted: tokenEnc,
		CreatedBy:            db.PGUUID(userID),
	})
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to create source")
		return
	}
	respond.JSON(w, http.StatusCreated, map[string]string{"id": db.FromPGUUID(id).String()})
}

func (h *Handler) UpdateSource(w http.ResponseWriter, r *http.Request) {
	wsID, _ := uuid.Parse(chi.URLParam(r, "id"))
	sourceID, _ := uuid.Parse(chi.URLParam(r, "sourceId"))
	var req struct {
		Name        string         `json:"name"`
		Config      map[string]any `json:"config"`
		AccessToken string         `json:"access_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid body")
		return
	}
	if req.Name == "" && req.Config == nil && strings.TrimSpace(req.AccessToken) == "" {
		respond.Error(w, http.StatusBadRequest, "nothing to update")
		return
	}
	params := sqlc.UpdateDocSourceParams{
		ID:          db.PGUUID(sourceID),
		WorkspaceID: db.PGUUID(wsID),
	}
	if req.Name != "" {
		params.Name = &req.Name
	}
	if req.Config != nil {
		normalized, err := normalizeRepoConfig(req.Config)
		if err != nil {
			respond.Error(w, http.StatusBadRequest, err.Error())
			return
		}
		cfg, _ := json.Marshal(normalized)
		params.Config = cfg
	}
	if tok := strings.TrimSpace(req.AccessToken); tok != "" {
		enc, err := h.crypto.Encrypt(tok)
		if err != nil {
			respond.Error(w, http.StatusInternalServerError, "failed to store credentials")
			return
		}
		params.AccessTokenEncrypted = &enc
	}
	if err := h.store.UpdateDocSource(r.Context(), params); err != nil {
		respond.Error(w, http.StatusInternalServerError, "update failed")
		return
	}
	row, err := h.store.GetDocSource(r.Context(), sqlc.GetDocSourceParams{
		ID:          db.PGUUID(sourceID),
		WorkspaceID: db.PGUUID(wsID),
	})
	if err != nil {
		respond.Error(w, http.StatusNotFound, "not found")
		return
	}
	respond.JSON(w, http.StatusOK, mapDocSourceGetRow(row))
}

func (h *Handler) DeleteSource(w http.ResponseWriter, r *http.Request) {
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
	ctx := r.Context()
	if _, err := h.store.GetDocSource(ctx, sqlc.GetDocSourceParams{
		ID:          db.PGUUID(sourceID),
		WorkspaceID: db.PGUUID(wsID),
	}); err != nil {
		respond.Error(w, http.StatusNotFound, "not found")
		return
	}
	if err := h.store.DeleteWorkspaceDocsByDocSource(ctx, sqlc.DeleteWorkspaceDocsByDocSourceParams{
		WorkspaceID:  db.PGUUID(wsID),
		DocSourceID:  db.PGUUID(sourceID),
	}); err != nil {
		respond.Error(w, http.StatusInternalServerError, "delete failed")
		return
	}
	if err := h.store.DeleteDocSource(ctx, sqlc.DeleteDocSourceParams{
		ID:          db.PGUUID(sourceID),
		WorkspaceID: db.PGUUID(wsID),
	}); err != nil {
		respond.Error(w, http.StatusInternalServerError, "delete failed")
		return
	}
	respond.JSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *Handler) SyncSource(w http.ResponseWriter, r *http.Request) {
	id, _ := uuid.Parse(chi.URLParam(r, "id"))
	row, err := h.store.GetDocSourceForSync(r.Context(), db.PGUUID(id))
	if err != nil {
		respond.Error(w, http.StatusNotFound, "not found")
		return
	}
	if row.SourceType != "git" {
		respond.Error(w, http.StatusBadRequest, "only git sources support sync")
		return
	}
	var config map[string]any
	_ = json.Unmarshal(row.Config, &config)
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
	defer cancel()
	count, meta, err := h.syncGitDocs(ctx, db.FromPGUUID(row.WorkspaceID), id, config, row.AccessTokenEncrypted)
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "sync failed: "+err.Error())
		return
	}
	_ = h.store.UpdateDocSourceLastSynced(r.Context(), db.PGUUID(id))
	respond.JSON(w, http.StatusOK, map[string]any{
		"synced": count,
		"total":  meta.total,
		"capped": meta.capped,
		"errors": meta.errors,
	})
}

func (h *Handler) GetDoc(w http.ResponseWriter, r *http.Request) {
	wsID, _ := uuid.Parse(chi.URLParam(r, "id"))
	slug := chi.URLParam(r, "slug")
	row, err := h.store.GetWorkspaceDocBySlug(r.Context(), sqlc.GetWorkspaceDocBySlugParams{
		WorkspaceID: db.PGUUID(wsID),
		Slug:        slug,
	})
	if err != nil {
		respond.Error(w, http.StatusNotFound, "not found")
		return
	}
	respond.JSON(w, http.StatusOK, mapWorkspaceDocFromRow(row))
}

func (h *Handler) UpdateDoc(w http.ResponseWriter, r *http.Request) {
	wsID, _ := uuid.Parse(chi.URLParam(r, "id"))
	slug := chi.URLParam(r, "slug")
	var req struct {
		Title     *string `json:"title"`
		ContentMD *string `json:"content_md"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid body")
		return
	}
	if req.Title == nil && req.ContentMD == nil {
		respond.Error(w, http.StatusBadRequest, "title or content_md required")
		return
	}
	existing, err := h.store.GetWorkspaceDocBySlug(r.Context(), sqlc.GetWorkspaceDocBySlugParams{
		WorkspaceID: db.PGUUID(wsID),
		Slug:        slug,
	})
	if err != nil {
		respond.Error(w, http.StatusNotFound, "not found")
		return
	}
	title := existing.Title
	content := existing.ContentMd
	if req.Title != nil {
		title = strings.TrimSpace(*req.Title)
	}
	if req.ContentMD != nil {
		content = *req.ContentMD
	}
	parsedTitle, ops, body := parseMarkdownDoc(content, slug+".md")
	if parsedTitle != slug+".md" && parsedTitle != slug && req.ContentMD != nil {
		if req.Title == nil {
			title = parsedTitle
		}
	}
	if body != "" {
		content = body
	}
	if title == "" {
		title = existing.Title
	}
	err = h.store.UpdateWorkspaceDoc(r.Context(), sqlc.UpdateWorkspaceDocParams{
		WorkspaceID:        db.PGUUID(wsID),
		Slug:               slug,
		Title:              title,
		ContentMd:          content,
		LinkedOperationIds: stringSliceOrEmpty(ops),
	})
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "update failed")
		return
	}
	row, err := h.store.GetWorkspaceDocBySlug(r.Context(), sqlc.GetWorkspaceDocBySlugParams{
		WorkspaceID: db.PGUUID(wsID),
		Slug:        slug,
	})
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "query failed")
		return
	}
	respond.JSON(w, http.StatusOK, mapWorkspaceDocFromRow(row))
}

func (h *Handler) CreateDoc(w http.ResponseWriter, r *http.Request) {
	wsID, _ := uuid.Parse(chi.URLParam(r, "id"))
	var req struct {
		Title      string `json:"title"`
		SourcePath string `json:"source_path"`
		ContentMD  string `json:"content_md"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid body")
		return
	}
	sourcePath := strings.Trim(strings.TrimSpace(req.SourcePath), "/")
	if sourcePath == "" {
		respond.Error(w, http.StatusBadRequest, "source_path required")
		return
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = pathBaseName(sourcePath)
	}
	slug := pathToSlug(sourcePath + ".md")
	if slug == "" {
		respond.Error(w, http.StatusBadRequest, "invalid source_path")
		return
	}
	content := req.ContentMD
	if content == "" {
		content = "# " + title + "\n"
	}
	row, err := h.store.CreateWorkspaceDoc(r.Context(), sqlc.CreateWorkspaceDocParams{
		WorkspaceID:        db.PGUUID(wsID),
		Slug:               slug,
		Title:              title,
		ContentMd:          content,
		SourcePath:         sourcePath,
		LinkedOperationIds: []string{},
	})
	if err != nil {
		respond.Error(w, http.StatusConflict, "doc already exists at this path")
		return
	}
	respond.JSON(w, http.StatusCreated, mapWorkspaceDocFromCreateRow(row))
}

func pathBaseName(p string) string {
	p = strings.TrimSuffix(p, "/")
	if i := strings.LastIndex(p, "/"); i >= 0 {
		return p[i+1:]
	}
	return p
}

func (h *Handler) ListDocs(w http.ResponseWriter, r *http.Request) {
	wsID, _ := uuid.Parse(chi.URLParam(r, "id"))
	opID := strings.TrimSpace(r.URL.Query().Get("operation_id"))
	summary := r.URL.Query().Get("summary") == "1" || r.URL.Query().Get("summary") == "true"

	if summary && opID == "" {
		rows, err := h.store.ListWorkspaceDocSummaries(r.Context(), db.PGUUID(wsID))
		if err != nil {
			respond.Error(w, http.StatusInternalServerError, "query failed")
			return
		}
		list := make([]WorkspaceDocSummary, 0, len(rows))
		for _, row := range rows {
			list = append(list, mapDocSummary(row))
		}
		respond.JSON(w, http.StatusOK, list)
		return
	}

	var docs []WorkspaceDoc
	if opID != "" {
		rows, err := h.store.ListWorkspaceDocs(r.Context(), db.PGUUID(wsID))
		if err != nil {
			respond.Error(w, http.StatusInternalServerError, "query failed")
			return
		}
		for _, row := range rows {
			if !containsOp(row.LinkedOperationIds, opID) {
				continue
			}
			docs = append(docs, mapWorkspaceDoc(row))
		}
	} else {
		rows, err := h.store.ListWorkspaceDocs(r.Context(), db.PGUUID(wsID))
		if err != nil {
			respond.Error(w, http.StatusInternalServerError, "query failed")
			return
		}
		for _, row := range rows {
			docs = append(docs, mapWorkspaceDoc(row))
		}
	}
	if docs == nil {
		docs = []WorkspaceDoc{}
	}
	respond.JSON(w, http.StatusOK, docs)
}

func mapWorkspaceDoc(row sqlc.ListWorkspaceDocsRow) WorkspaceDoc {
	return workspaceDocFields(row.ID, row.WorkspaceID, row.Slug, row.Title, row.ContentMd, row.SourcePath, row.IsLocal, row.LinkedOperationIds, row.UpdatedAt)
}

func mapWorkspaceDocFromRow(row sqlc.GetWorkspaceDocBySlugRow) WorkspaceDoc {
	return workspaceDocFields(row.ID, row.WorkspaceID, row.Slug, row.Title, row.ContentMd, row.SourcePath, row.IsLocal, row.LinkedOperationIds, row.UpdatedAt)
}

func mapWorkspaceDocFromCreateRow(row sqlc.CreateWorkspaceDocRow) WorkspaceDoc {
	return workspaceDocFields(row.ID, row.WorkspaceID, row.Slug, row.Title, row.ContentMd, row.SourcePath, row.IsLocal, row.LinkedOperationIds, row.UpdatedAt)
}

func mapDocSummary(row sqlc.ListWorkspaceDocSummariesRow) WorkspaceDocSummary {
	return WorkspaceDocSummary{
		ID:          db.FromPGUUID(row.ID).String(),
		WorkspaceID: db.FromPGUUID(row.WorkspaceID).String(),
		Slug:        row.Slug,
		Title:       row.Title,
		SourcePath:  row.SourcePath,
		IsLocal:     row.IsLocal,
		UpdatedAt:   row.UpdatedAt.Time.Format(time.RFC3339),
	}
}

func workspaceDocFields(id, wsID pgtype.UUID, slug, title, content, sourcePath string, isLocal bool, ops []string, updated pgtype.Timestamptz) WorkspaceDoc {
	return WorkspaceDoc{
		ID:                 db.FromPGUUID(id).String(),
		WorkspaceID:        db.FromPGUUID(wsID).String(),
		Slug:               slug,
		Title:              title,
		ContentMD:          content,
		SourcePath:         sourcePath,
		IsLocal:            isLocal,
		LinkedOperationIDs: ops,
		UpdatedAt:          updated.Time.Format(time.RFC3339),
	}
}

func containsOp(ops []string, opID string) bool {
	for _, op := range ops {
		if op == opID {
			return true
		}
	}
	return false
}

const (
	maxSyncFiles         = 2000
	maxConcurrentFetches = 8
)

type syncResultMeta struct {
	total  int
	capped int
	errors int
}

func (h *Handler) syncGitDocs(ctx context.Context, wsID, sourceID uuid.UUID, config map[string]any, tokenEnc *string) (int, syncResultMeta, error) {
	meta := syncResultMeta{}
	normalized, err := normalizeRepoConfig(config)
	if err != nil {
		return 0, meta, err
	}
	cfg, _ := json.Marshal(normalized)
	_ = h.store.UpdateDocSource(ctx, sqlc.UpdateDocSourceParams{
		Config:      cfg,
		ID:          db.PGUUID(sourceID),
		WorkspaceID: db.PGUUID(wsID),
	})
	config = normalized

	token := ""
	if tokenEnc != nil && strings.TrimSpace(*tokenEnc) != "" {
		plain, err := h.crypto.Decrypt(*tokenEnc)
		if err != nil {
			return 0, meta, fmt.Errorf("failed to decrypt access token")
		}
		token = plain
	}
	client, err := gitClientFromConfig(config, token)
	if err != nil {
		return 0, meta, err
	}
	files, err := client.ListMarkdownFiles(ctx)
	if err != nil {
		return 0, meta, err
	}
	meta.total = len(files)
	capped := 0
	if len(files) > maxSyncFiles {
		capped = len(files) - maxSyncFiles
		meta.capped = capped
		files = files[:maxSyncFiles]
	}

	type fetchResult struct {
		path    string
		content string
		err     error
	}
	results := make(chan fetchResult, len(files))
	sem := make(chan struct{}, maxConcurrentFetches)
	var wg sync.WaitGroup
	for _, filePath := range files {
		wg.Add(1)
		go func(p string) {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				results <- fetchResult{path: p, err: ctx.Err()}
				return
			}
			content, fetchErr := client.FetchFile(ctx, p)
			results <- fetchResult{path: p, content: content, err: fetchErr}
		}(filePath)
	}
	go func() {
		wg.Wait()
		close(results)
	}()

	count := 0
	fetchErrors := 0
	var firstFetchErr error
	var firstUpsertErr error
	for res := range results {
		if res.err != nil {
			fetchErrors++
			if firstFetchErr == nil {
				firstFetchErr = fmt.Errorf("%s: %w", res.path, res.err)
			}
			continue
		}
		if strings.TrimSpace(res.content) == "" {
			fetchErrors++
			if firstFetchErr == nil {
				firstFetchErr = fmt.Errorf("%s: file is empty", res.path)
			}
			continue
		}
		title, ops, body := parseMarkdownDoc(res.content, res.path)
		slug := pathToSlug(res.path)
		sourcePath := filePathToSourcePath(res.path)
		if slug == "" || sourcePath == "" {
			continue
		}
		_ = h.store.ClearWorkspaceDocSlugConflict(ctx, sqlc.ClearWorkspaceDocSlugConflictParams{
			WorkspaceID: db.PGUUID(wsID),
			Slug:        slug,
			SourcePath:  sourcePath,
			DocSourceID: db.PGUUID(sourceID),
		})
		err = h.store.UpsertWorkspaceDoc(ctx, sqlc.UpsertWorkspaceDocParams{
			WorkspaceID:        db.PGUUID(wsID),
			DocSourceID:        db.PGUUID(sourceID),
			Slug:               slug,
			Title:              title,
			ContentMd:          body,
			SourcePath:         sourcePath,
			IsLocal:            false,
			LinkedOperationIds: stringSliceOrEmpty(ops),
		})
		if err != nil {
			meta.errors++
			if firstUpsertErr == nil {
				firstUpsertErr = fmt.Errorf("%s: %w", res.path, err)
			}
			continue
		}
		count++
	}
	if ctx.Err() != nil {
		return count, meta, fmt.Errorf("sync timed out after fetching %d/%d files: %w", count, len(files), ctx.Err())
	}
	if count == 0 {
		if len(files) == 0 {
			return 0, meta, fmt.Errorf("no markdown files found; check branch and path prefix")
		}
		if fetchErrors > 0 {
			if firstFetchErr != nil {
				return 0, meta, fmt.Errorf("could not read markdown files: %v", firstFetchErr)
			}
			return 0, meta, fmt.Errorf("could not read markdown files; check access token scopes (GitLab: read_api and read_repository)")
		}
		if firstUpsertErr != nil {
			return 0, meta, fmt.Errorf("failed to save synced docs: %v", firstUpsertErr)
		}
		return 0, meta, fmt.Errorf("no markdown files synced from %d candidate file(s)", len(files))
	}
	meta.errors += fetchErrors
	return count, meta, nil
}

func stringSliceOrEmpty(v []string) []string {
	if v == nil {
		return []string{}
	}
	return v
}

func mapDocSourceRow(row sqlc.ListDocSourcesRow) DocSource {
	return mapDocSourceFields(
		row.ID, row.WorkspaceID, row.CollectionID, row.Name, row.SourceType,
		row.Config, row.AccessTokenEncrypted, row.LastSyncedAt, row.CreatedAt,
	)
}

func mapDocSourceGetRow(row sqlc.GetDocSourceRow) DocSource {
	return mapDocSourceFields(
		row.ID, row.WorkspaceID, row.CollectionID, row.Name, row.SourceType,
		row.Config, row.AccessTokenEncrypted, row.LastSyncedAt, row.CreatedAt,
	)
}

func mapDocSourceFields(
	id, wsID, colID pgtype.UUID,
	name, sourceType string,
	config []byte,
	tokenEnc *string,
	lastSynced, createdAt pgtype.Timestamptz,
) DocSource {
	s := DocSource{
		ID:             db.FromPGUUID(id).String(),
		WorkspaceID:    db.FromPGUUID(wsID).String(),
		Name:           name,
		SourceType:     sourceType,
		Config:         map[string]any{},
		HasAccessToken: tokenEnc != nil && strings.TrimSpace(*tokenEnc) != "",
		CreatedAt:      createdAt.Time.Format(time.RFC3339),
	}
	if colID.Valid {
		c := db.FromPGUUID(colID).String()
		s.CollectionID = &c
	}
	_ = json.Unmarshal(config, &s.Config)
	s.Config = sanitizeSourceConfig(s.Config)
	if lastSynced.Valid {
		t := lastSynced.Time.Format(time.RFC3339)
		s.LastSyncedAt = &t
	}
	return s
}

