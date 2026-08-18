package history

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

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

type Entry struct {
	ID              string         `json:"id"`
	WorkspaceID     string         `json:"workspace_id"`
	RequestID       *string        `json:"request_id,omitempty"`
	Snapshot        map[string]any `json:"snapshot"`
	Response        map[string]any `json:"response"`
	TestResults     any            `json:"test_results,omitempty"`
	ExecutedBy      string         `json:"executed_by"`
	ExecutedByName  string         `json:"executed_by_name,omitempty"`
	ExecutedByEmail string         `json:"executed_by_email,omitempty"`
	ExecutedAt      any            `json:"executed_at"`
	DurationMS      int64          `json:"duration_ms"`
	StatusCode      int            `json:"status_code"`
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.UserIDFromContext(r.Context())
	if err != nil {
		respond.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	wsIDStr := r.URL.Query().Get("workspace_id")
	reqID := r.URL.Query().Get("request_id")
	if wsIDStr == "" {
		respond.Error(w, http.StatusBadRequest, "workspace_id required")
		return
	}
	wsID, err := uuid.Parse(wsIDStr)
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid workspace_id")
		return
	}
	if !h.hasMinRole(r.Context(), wsID, userID, "viewer") {
		respond.Error(w, http.StatusForbidden, "not a workspace member")
		return
	}

	pgWS := db.PGUUID(wsID)
	var list []Entry
	if reqID != "" {
		reqUUID, err := uuid.Parse(reqID)
		if err != nil {
			respond.Error(w, http.StatusBadRequest, "invalid request_id")
			return
		}
		rows, err := h.store.ListHistoryByWorkspaceAndRequest(r.Context(), sqlc.ListHistoryByWorkspaceAndRequestParams{
			WorkspaceID: pgWS,
			RequestID:   db.PGUUID(reqUUID),
		})
		if err != nil {
			respond.Error(w, http.StatusInternalServerError, "query failed")
			return
		}
		list = make([]Entry, 0, len(rows))
		for _, row := range rows {
			list = append(list, entryFromFields(
				row.ID, row.WorkspaceID, row.RequestID, row.Snapshot, row.Response, row.TestResults,
				row.ExecutedBy, row.DisplayName, row.Email, row.ExecutedAt, row.DurationMs, row.StatusCode,
			))
		}
	} else {
		rows, err := h.store.ListHistoryByWorkspace(r.Context(), pgWS)
		if err != nil {
			respond.Error(w, http.StatusInternalServerError, "query failed")
			return
		}
		list = make([]Entry, 0, len(rows))
		for _, row := range rows {
			list = append(list, entryFromFields(
				row.ID, row.WorkspaceID, row.RequestID, row.Snapshot, row.Response, row.TestResults,
				row.ExecutedBy, row.DisplayName, row.Email, row.ExecutedAt, row.DurationMs, row.StatusCode,
			))
		}
	}
	if list == nil {
		list = []Entry{}
	}
	respond.JSON(w, http.StatusOK, list)
}

func entryFromFields(
	id, ws, reqID pgtype.UUID,
	snap, resp, testResults []byte,
	execBy pgtype.UUID,
	execName, execEmail string,
	executedAt pgtype.Timestamptz,
	durationMS int64,
	statusCode int32,
) Entry {
	e := Entry{
		ID:              db.FromPGUUID(id).String(),
		WorkspaceID:     db.FromPGUUID(ws).String(),
		ExecutedBy:      db.FromPGUUID(execBy).String(),
		ExecutedByName:  execName,
		ExecutedByEmail: execEmail,
		DurationMS:      durationMS,
		StatusCode:      int(statusCode),
	}
	if reqID.Valid {
		s := db.FromPGUUID(reqID).String()
		e.RequestID = &s
	}
	if t := db.FromPGTimestamptz(executedAt); t != nil {
		e.ExecutedAt = *t
	}
	_ = json.Unmarshal(snap, &e.Snapshot)
	_ = json.Unmarshal(resp, &e.Response)
	if len(testResults) > 0 {
		_ = json.Unmarshal(testResults, &e.TestResults)
	}
	return e
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
