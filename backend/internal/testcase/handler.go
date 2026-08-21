package testcase

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/imaanmzr/postchi/backend/internal/auth"
	"github.com/imaanmzr/postchi/backend/internal/db"
	"github.com/imaanmzr/postchi/backend/internal/db/sqlc"
	"github.com/imaanmzr/postchi/backend/internal/shared/respond"
)

type Handler struct {
	store *db.Store
}

func NewHandler(store *db.Store) *Handler {
	return &Handler{store: store}
}

type LinkedRequest struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Method        string `json:"method"`
	URL           string `json:"url"`
	WorkspaceID   string `json:"workspace_id"`
	WorkspaceName string `json:"workspace_name"`
}

type TestCase struct {
	ID          string          `json:"id"`
	WorkspaceID string          `json:"workspace_id"`
	Title       string          `json:"title"`
	Description string          `json:"description"`
	SortOrder   int32           `json:"sort_order"`
	Requests    []LinkedRequest `json:"requests,omitempty"`
	UpdatedAt   string          `json:"updated_at"`
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	wsID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid workspace id")
		return
	}
	rows, err := h.store.ListTestCases(r.Context(), db.PGUUID(wsID))
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to list test cases")
		return
	}
	list := make([]TestCase, 0, len(rows))
	for _, row := range rows {
		tc := mapTestCase(row)
		links, _ := h.store.ListTestCaseRequestLinks(r.Context(), row.ID)
		tc.Requests = mapLinkedRequests(links)
		list = append(list, tc)
	}
	respond.JSON(w, http.StatusOK, list)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	wsID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid workspace id")
		return
	}
	tcID, err := uuid.Parse(chi.URLParam(r, "testCaseId"))
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid test case id")
		return
	}
	row, err := h.store.GetTestCase(r.Context(), sqlc.GetTestCaseParams{
		ID:          db.PGUUID(tcID),
		WorkspaceID: db.PGUUID(wsID),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			respond.Error(w, http.StatusNotFound, "test case not found")
			return
		}
		respond.Error(w, http.StatusInternalServerError, "failed to get test case")
		return
	}
	tc := mapTestCase(row)
	links, _ := h.store.ListTestCaseRequestLinks(r.Context(), db.PGUUID(tcID))
	tc.Requests = mapLinkedRequests(links)
	respond.JSON(w, http.StatusOK, tc)
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
		Title       string `json:"title"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.Title) == "" {
		respond.Error(w, http.StatusBadRequest, "title required")
		return
	}
	row, err := h.store.CreateTestCase(r.Context(), sqlc.CreateTestCaseParams{
		WorkspaceID: db.PGUUID(wsID),
		Title:       strings.TrimSpace(req.Title),
		Description: strings.TrimSpace(req.Description),
		SortOrder:   0,
		CreatedBy:   db.PGUUID(userID),
	})
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to create test case")
		return
	}
	respond.JSON(w, http.StatusCreated, mapTestCase(row))
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	wsID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid workspace id")
		return
	}
	tcID, err := uuid.Parse(chi.URLParam(r, "testCaseId"))
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid test case id")
		return
	}
	var req struct {
		Title       *string `json:"title"`
		Description *string `json:"description"`
		SortOrder   *int32  `json:"sort_order"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid body")
		return
	}
	params := sqlc.UpdateTestCaseParams{
		ID:          db.PGUUID(tcID),
		WorkspaceID: db.PGUUID(wsID),
	}
	if req.Title != nil {
		title := strings.TrimSpace(*req.Title)
		params.Title = &title
	}
	if req.Description != nil {
		desc := strings.TrimSpace(*req.Description)
		params.Description = &desc
	}
	if req.SortOrder != nil {
		params.SortOrder = req.SortOrder
	}
	row, err := h.store.UpdateTestCase(r.Context(), params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			respond.Error(w, http.StatusNotFound, "test case not found")
			return
		}
		respond.Error(w, http.StatusInternalServerError, "failed to update test case")
		return
	}
	respond.JSON(w, http.StatusOK, mapTestCase(row))
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	wsID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid workspace id")
		return
	}
	tcID, err := uuid.Parse(chi.URLParam(r, "testCaseId"))
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid test case id")
		return
	}
	if err := h.store.DeleteTestCase(r.Context(), sqlc.DeleteTestCaseParams{
		ID:          db.PGUUID(tcID),
		WorkspaceID: db.PGUUID(wsID),
	}); err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to delete test case")
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
	tcID, err := uuid.Parse(chi.URLParam(r, "testCaseId"))
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid test case id")
		return
	}
	reqID, err := uuid.Parse(chi.URLParam(r, "requestId"))
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid request id")
		return
	}
	ok, err := h.store.VerifyTestCaseInWorkspace(r.Context(), sqlc.VerifyTestCaseInWorkspaceParams{
		ID:          db.PGUUID(tcID),
		WorkspaceID: db.PGUUID(wsID),
	})
	if err != nil || !ok {
		respond.Error(w, http.StatusNotFound, "test case not found")
		return
	}
	if _, err := h.store.VerifyRequestAccessibleToUser(r.Context(), sqlc.VerifyRequestAccessibleToUserParams{
		RequestID: db.PGUUID(reqID),
		UserID:    db.PGUUID(userID),
	}); err != nil {
		respond.Error(w, http.StatusNotFound, "request not found")
		return
	}
	if err := h.store.AddTestCaseRequestLink(r.Context(), sqlc.AddTestCaseRequestLinkParams{
		TestCaseID: db.PGUUID(tcID),
		RequestID:  db.PGUUID(reqID),
	}); err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to link request")
		return
	}
	respond.JSON(w, http.StatusCreated, map[string]string{"status": "linked"})
}

func (h *Handler) RemoveRequestLink(w http.ResponseWriter, r *http.Request) {
	tcID, err := uuid.Parse(chi.URLParam(r, "testCaseId"))
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid test case id")
		return
	}
	reqID, err := uuid.Parse(chi.URLParam(r, "requestId"))
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid request id")
		return
	}
	if err := h.store.RemoveTestCaseRequestLink(r.Context(), sqlc.RemoveTestCaseRequestLinkParams{
		TestCaseID: db.PGUUID(tcID),
		RequestID:  db.PGUUID(reqID),
	}); err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to unlink request")
		return
	}
	respond.JSON(w, http.StatusOK, map[string]string{"status": "unlinked"})
}

func mapTestCase(row sqlc.TestCase) TestCase {
	return TestCase{
		ID:          db.FromPGUUID(row.ID).String(),
		WorkspaceID: db.FromPGUUID(row.WorkspaceID).String(),
		Title:       row.Title,
		Description: row.Description,
		SortOrder:   row.SortOrder,
		UpdatedAt:   row.UpdatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func mapLinkedRequests(rows []sqlc.ListTestCaseRequestLinksRow) []LinkedRequest {
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
