package diagram

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/imaanmzr/postchi/backend/internal/auth"
	"github.com/imaanmzr/postchi/backend/internal/db"
	"github.com/imaanmzr/postchi/backend/internal/db/sqlc"
	"github.com/imaanmzr/postchi/backend/internal/shared/respond"
)

var slugRe = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

type Handler struct {
	store *db.Store
}

func NewHandler(store *db.Store) *Handler {
	return &Handler{store: store}
}

type DiagramSummary struct {
	ID          string `json:"id"`
	WorkspaceID string `json:"workspace_id"`
	Slug        string `json:"slug"`
	Title       string `json:"title"`
	UpdatedAt   string `json:"updated_at"`
}

type LinkedRequest struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Method        string `json:"method"`
	URL           string `json:"url"`
	WorkspaceID   string `json:"workspace_id"`
	WorkspaceName string `json:"workspace_name"`
}

type Diagram struct {
	ID          string           `json:"id"`
	WorkspaceID string           `json:"workspace_id"`
	Slug        string           `json:"slug"`
	Title       string           `json:"title"`
	Content     map[string]any   `json:"content,omitempty"`
	Requests    []LinkedRequest  `json:"requests,omitempty"`
	UpdatedAt   string           `json:"updated_at"`
}

func defaultContent() []byte {
	return []byte(`{"type":"excalidraw","version":2,"source":"postchi","elements":[],"appState":{},"files":{}}`)
}

func slugify(title string) string {
	s := strings.ToLower(strings.TrimSpace(title))
	s = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if s == "" {
		return "diagram"
	}
	return s
}

func uniqueSlug(base string, exists func(string) bool) string {
	if !exists(base) {
		return base
	}
	for i := 2; i < 1000; i++ {
		candidate := fmt.Sprintf("%s-%d", base, i)
		if !exists(candidate) {
			return candidate
		}
	}
	return base + "-" + uuid.New().String()[:8]
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	wsID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid workspace id")
		return
	}
	rows, err := h.store.ListWorkspaceDiagrams(r.Context(), db.PGUUID(wsID))
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to list diagrams")
		return
	}
	list := make([]DiagramSummary, 0, len(rows))
	for _, row := range rows {
		list = append(list, DiagramSummary{
			ID:          db.FromPGUUID(row.ID).String(),
			WorkspaceID: db.FromPGUUID(row.WorkspaceID).String(),
			Slug:        row.Slug,
			Title:       row.Title,
			UpdatedAt:   row.UpdatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
		})
	}
	respond.JSON(w, http.StatusOK, list)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	wsID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid workspace id")
		return
	}
	slug := chi.URLParam(r, "slug")
	row, err := h.store.GetWorkspaceDiagramBySlug(r.Context(), sqlc.GetWorkspaceDiagramBySlugParams{
		WorkspaceID: db.PGUUID(wsID),
		Slug:        slug,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			respond.Error(w, http.StatusNotFound, "diagram not found")
			return
		}
		respond.Error(w, http.StatusInternalServerError, "failed to get diagram")
		return
	}
	respond.JSON(w, http.StatusOK, h.mapDiagramWithLinks(r.Context(), row))
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
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
		Title   string         `json:"title"`
		Slug    string         `json:"slug"`
		Content map[string]any `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.Title) == "" {
		respond.Error(w, http.StatusBadRequest, "title required")
		return
	}
	req.Title = strings.TrimSpace(req.Title)
	slug := strings.TrimSpace(req.Slug)
	if slug == "" {
		slug = slugify(req.Title)
	}
	if !slugRe.MatchString(slug) {
		respond.Error(w, http.StatusBadRequest, "invalid slug")
		return
	}
	exists := func(s string) bool {
		ok, _ := h.store.WorkspaceDiagramSlugExists(r.Context(), sqlc.WorkspaceDiagramSlugExistsParams{
			WorkspaceID: db.PGUUID(wsID),
			Slug:        s,
		})
		return ok
	}
	slug = uniqueSlug(slug, exists)

	content := defaultContent()
	if req.Content != nil {
		content, _ = json.Marshal(req.Content)
	}

	row, err := h.store.CreateWorkspaceDiagram(r.Context(), sqlc.CreateWorkspaceDiagramParams{
		WorkspaceID: db.PGUUID(wsID),
		Slug:        slug,
		Title:       req.Title,
		Content:     content,
		CreatedBy:   db.PGUUID(userID),
	})
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to create diagram")
		return
	}
	respond.JSON(w, http.StatusCreated, mapDiagram(row))
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	wsID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid workspace id")
		return
	}
	slug := chi.URLParam(r, "slug")
	var req struct {
		Title   *string        `json:"title"`
		Content map[string]any `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid body")
		return
	}
	params := sqlc.UpdateWorkspaceDiagramParams{
		WorkspaceID: db.PGUUID(wsID),
		Slug:        slug,
	}
	if req.Title != nil {
		title := strings.TrimSpace(*req.Title)
		params.Title = &title
	}
	if req.Content != nil {
		b, _ := json.Marshal(req.Content)
		params.Content = b
	}
	row, err := h.store.UpdateWorkspaceDiagram(r.Context(), params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			respond.Error(w, http.StatusNotFound, "diagram not found")
			return
		}
		respond.Error(w, http.StatusInternalServerError, "failed to update diagram")
		return
	}
	respond.JSON(w, http.StatusOK, mapDiagram(row))
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	wsID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid workspace id")
		return
	}
	slug := chi.URLParam(r, "slug")
	if err := h.store.DeleteWorkspaceDiagram(r.Context(), sqlc.DeleteWorkspaceDiagramParams{
		WorkspaceID: db.PGUUID(wsID),
		Slug:        slug,
	}); err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to delete diagram")
		return
	}
	respond.JSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *Handler) AddRequestLink(w http.ResponseWriter, r *http.Request) {
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
	slug := chi.URLParam(r, "slug")
	reqID, err := uuid.Parse(chi.URLParam(r, "requestId"))
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid request id")
		return
	}
	row, err := h.store.GetWorkspaceDiagramBySlug(r.Context(), sqlc.GetWorkspaceDiagramBySlugParams{
		WorkspaceID: db.PGUUID(wsID),
		Slug:        slug,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			respond.Error(w, http.StatusNotFound, "diagram not found")
			return
		}
		respond.Error(w, http.StatusInternalServerError, "failed to get diagram")
		return
	}
	if _, err := h.store.VerifyRequestAccessibleToUser(r.Context(), sqlc.VerifyRequestAccessibleToUserParams{
		RequestID: db.PGUUID(reqID),
		UserID:    db.PGUUID(userID),
	}); err != nil {
		respond.Error(w, http.StatusNotFound, "request not found")
		return
	}
	if err := h.store.AddDiagramRequestLink(r.Context(), sqlc.AddDiagramRequestLinkParams{
		DiagramID: row.ID,
		RequestID: db.PGUUID(reqID),
	}); err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to link request")
		return
	}
	respond.JSON(w, http.StatusCreated, map[string]string{"status": "linked"})
}

func (h *Handler) RemoveRequestLink(w http.ResponseWriter, r *http.Request) {
	wsID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid workspace id")
		return
	}
	slug := chi.URLParam(r, "slug")
	reqID, err := uuid.Parse(chi.URLParam(r, "requestId"))
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid request id")
		return
	}
	row, err := h.store.GetWorkspaceDiagramBySlug(r.Context(), sqlc.GetWorkspaceDiagramBySlugParams{
		WorkspaceID: db.PGUUID(wsID),
		Slug:        slug,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			respond.Error(w, http.StatusNotFound, "diagram not found")
			return
		}
		respond.Error(w, http.StatusInternalServerError, "failed to get diagram")
		return
	}
	if err := h.store.RemoveDiagramRequestLink(r.Context(), sqlc.RemoveDiagramRequestLinkParams{
		DiagramID: row.ID,
		RequestID: db.PGUUID(reqID),
	}); err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to unlink request")
		return
	}
	respond.JSON(w, http.StatusOK, map[string]string{"status": "unlinked"})
}

func (h *Handler) mapDiagramWithLinks(ctx context.Context, row sqlc.WorkspaceDiagram) Diagram {
	d := mapDiagram(row)
	links, _ := h.store.ListDiagramRequestLinks(ctx, row.ID)
	d.Requests = mapDiagramLinkedRequests(links)
	return d
}

func mapDiagramLinkedRequests(rows []sqlc.ListDiagramRequestLinksRow) []LinkedRequest {
	out := make([]LinkedRequest, 0, len(rows))
	for _, link := range rows {
		out = append(out, LinkedRequest{
			ID:            db.FromPGUUID(link.ID).String(),
			Name:          link.Name,
			Method:        link.Method,
			URL:           link.Url,
			WorkspaceID:   db.FromPGUUID(link.WorkspaceID).String(),
			WorkspaceName: link.WorkspaceName,
		})
	}
	return out
}

func mapDiagram(row sqlc.WorkspaceDiagram) Diagram {
	d := Diagram{
		ID:          db.FromPGUUID(row.ID).String(),
		WorkspaceID: db.FromPGUUID(row.WorkspaceID).String(),
		Slug:        row.Slug,
		Title:       row.Title,
		UpdatedAt:   row.UpdatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
	}
	_ = json.Unmarshal(row.Content, &d.Content)
	return d
}
