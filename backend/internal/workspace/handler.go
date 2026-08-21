package workspace

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/imaanmzr/postchi/backend/internal/auth"
	"github.com/imaanmzr/postchi/backend/internal/db"
	"github.com/imaanmzr/postchi/backend/internal/db/sqlc"
	"github.com/imaanmzr/postchi/backend/internal/shared/respond"
)

var errWorkspaceNameTaken = errors.New("workspace name already in use")

type Handler struct {
	store *db.Store
}

func NewHandler(store *db.Store) *Handler {
	return &Handler{store: store}
}

type Workspace struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Variables   map[string]any `json:"variables,omitempty"`
	Role        string         `json:"role,omitempty"`
	CreatedAt   string         `json:"created_at"`
}

type Member struct {
	UserID      string `json:"user_id"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
}

func pgUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}

func fromPgUUID(id pgtype.UUID) uuid.UUID {
	if !id.Valid {
		return uuid.Nil
	}
	return uuid.UUID(id.Bytes)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.UserIDFromContext(r.Context())
	if err != nil {
		respond.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	rows, err := h.store.ListWorkspacesByUser(r.Context(), pgUUID(userID))
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to list workspaces")
		return
	}

	list := make([]Workspace, 0, len(rows))
	for _, row := range rows {
		var ws Workspace
		ws.ID = fromPgUUID(row.ID).String()
		ws.Name = row.Name
		ws.Description = row.Description
		ws.Role = row.WmRole
		_ = json.Unmarshal(row.Variables, &ws.Variables)
		list = append(list, ws)
	}
	respond.JSON(w, http.StatusOK, list)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.UserIDFromContext(r.Context())
	if err != nil {
		respond.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.Name) == "" {
		respond.Error(w, http.StatusBadRequest, "name required")
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Description != "" {
		req.Description = strings.TrimSpace(req.Description)
	}

	taken, err := h.store.UserHasWorkspaceNamed(r.Context(), sqlc.UserHasWorkspaceNamedParams{
		UserID: pgUUID(userID),
		Name:   req.Name,
	})
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to create workspace")
		return
	}
	if taken {
		respond.Error(w, http.StatusConflict, errWorkspaceNameTaken.Error())
		return
	}

	var wsID uuid.UUID
	err = h.store.WithTx(r.Context(), func(q *sqlc.Queries) error {
		pgUserID := pgUUID(userID)
		id, err := q.CreateWorkspace(r.Context(), sqlc.CreateWorkspaceParams{
			Name:        req.Name,
			Description: req.Description,
			CreatedBy:   pgUserID,
		})
		if err != nil {
			return err
		}
		wsID = fromPgUUID(id)
		if err := q.AddWorkspaceOwner(r.Context(), sqlc.AddWorkspaceOwnerParams{
			WorkspaceID: id,
			UserID:      pgUserID,
		}); err != nil {
			return err
		}
		return q.CreateDefaultCollection(r.Context(), sqlc.CreateDefaultCollectionParams{
			WorkspaceID: id,
			CreatedBy:   pgUserID,
		})
	})
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to create workspace")
		return
	}
	_ = h.logActivity(r, wsID, userID, "create", "workspace", wsID, nil)
	respond.JSON(w, http.StatusCreated, Workspace{ID: wsID.String(), Name: req.Name, Description: req.Description, Role: "owner"})
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	wsID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	row, err := h.store.GetWorkspaceByID(r.Context(), pgUUID(wsID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			respond.Error(w, http.StatusNotFound, "workspace not found")
			return
		}
		respond.Error(w, http.StatusInternalServerError, "failed to get workspace")
		return
	}
	var ws Workspace
	ws.ID = fromPgUUID(row.ID).String()
	ws.Name = row.Name
	ws.Description = row.Description
	_ = json.Unmarshal(row.Variables, &ws.Variables)
	if wsCtx, ok := auth.WorkspaceFromContext(r.Context()); ok {
		ws.Role = wsCtx.Role
	}
	respond.JSON(w, http.StatusOK, ws)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	wsID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	var req struct {
		Name        *string        `json:"name"`
		Description *string        `json:"description"`
		Variables   map[string]any `json:"variables"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid body")
		return
	}
	pgWsID := pgUUID(wsID)
	userID, err := auth.UserIDFromContext(r.Context())
	if err != nil {
		respond.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			respond.Error(w, http.StatusBadRequest, "name required")
			return
		}
		taken, err := h.store.UserHasWorkspaceNamedExcluding(r.Context(), sqlc.UserHasWorkspaceNamedExcludingParams{
			UserID:             pgUUID(userID),
			Name:               name,
			ExcludeWorkspaceID: pgWsID,
		})
		if err != nil {
			respond.Error(w, http.StatusInternalServerError, "failed to update workspace")
			return
		}
		if taken {
			respond.Error(w, http.StatusConflict, errWorkspaceNameTaken.Error())
			return
		}
		if err := h.store.UpdateWorkspaceName(r.Context(), sqlc.UpdateWorkspaceNameParams{
			Name: name,
			ID:   pgWsID,
		}); err != nil {
			respond.Error(w, http.StatusInternalServerError, "failed to update workspace")
			return
		}
	}
	if req.Description != nil {
		desc := strings.TrimSpace(*req.Description)
		if err := h.store.UpdateWorkspaceDescription(r.Context(), sqlc.UpdateWorkspaceDescriptionParams{
			Description: desc,
			ID:          pgWsID,
		}); err != nil {
			respond.Error(w, http.StatusInternalServerError, "failed to update workspace")
			return
		}
	}
	if req.Variables != nil {
		b, _ := json.Marshal(req.Variables)
		if err := h.store.UpdateWorkspaceVariables(r.Context(), sqlc.UpdateWorkspaceVariablesParams{
			Variables: b,
			ID:        pgWsID,
		}); err != nil {
			respond.Error(w, http.StatusInternalServerError, "failed to update workspace")
			return
		}
	}
	h.Get(w, r)
}

func (h *Handler) ListMembers(w http.ResponseWriter, r *http.Request) {
	wsID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	rows, err := h.store.ListWorkspaceMembers(r.Context(), pgUUID(wsID))
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to list members")
		return
	}
	members := make([]Member, 0, len(rows))
	for _, row := range rows {
		members = append(members, Member{
			UserID:      fromPgUUID(row.ID).String(),
			Email:       row.Email,
			DisplayName: row.DisplayName,
			Role:        row.WmRole,
		})
	}
	respond.JSON(w, http.StatusOK, members)
}

func (h *Handler) AddMember(w http.ResponseWriter, r *http.Request) {
	wsID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	var req struct {
		Email string `json:"email"`
		Role  string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Email == "" {
		respond.Error(w, http.StatusBadRequest, "email required")
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if req.Role == "" {
		req.Role = "viewer"
	}
	userID, err := h.store.GetUserIDByEmail(r.Context(), req.Email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			respond.Error(w, http.StatusNotFound, "user not found")
			return
		}
		respond.Error(w, http.StatusInternalServerError, "failed to look up user")
		return
	}
	err = h.store.UpsertWorkspaceMember(r.Context(), sqlc.UpsertWorkspaceMemberParams{
		WorkspaceID: pgUUID(wsID),
		UserID:      userID,
		Role:        sqlc.WorkspaceRole(req.Role),
	})
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to add member")
		return
	}
	respond.JSON(w, http.StatusCreated, Member{UserID: fromPgUUID(userID).String(), Email: req.Email, Role: req.Role})
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	wsID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.store.DeleteWorkspace(r.Context(), pgUUID(wsID)); err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to delete workspace")
		return
	}
	respond.JSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *Handler) UpdateMember(w http.ResponseWriter, r *http.Request) {
	wsID, _ := uuid.Parse(chi.URLParam(r, "id"))
	memberID, err := uuid.Parse(chi.URLParam(r, "userId"))
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid user id")
		return
	}
	var req struct {
		Role string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Role == "" {
		respond.Error(w, http.StatusBadRequest, "role required")
		return
	}
	pgWsID := pgUUID(wsID)
	pgMemberID := pgUUID(memberID)
	ownerCount, _ := h.store.CountWorkspaceOwners(r.Context(), pgWsID)
	memberRole, _ := h.store.GetWorkspaceMemberRole(r.Context(), sqlc.GetWorkspaceMemberRoleParams{
		WorkspaceID: pgWsID,
		UserID:      pgMemberID,
	})
	if memberRole == "owner" && ownerCount <= 1 && req.Role != "owner" {
		respond.Error(w, http.StatusBadRequest, "cannot demote the last owner")
		return
	}
	err = h.store.UpdateWorkspaceMemberRole(r.Context(), sqlc.UpdateWorkspaceMemberRoleParams{
		Role:        sqlc.WorkspaceRole(req.Role),
		WorkspaceID: pgWsID,
		UserID:      pgMemberID,
	})
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to update member")
		return
	}
	respond.JSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (h *Handler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	wsID, _ := uuid.Parse(chi.URLParam(r, "id"))
	memberID, err := uuid.Parse(chi.URLParam(r, "userId"))
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid user id")
		return
	}
	pgWsID := pgUUID(wsID)
	pgMemberID := pgUUID(memberID)
	ownerCount, _ := h.store.CountWorkspaceOwners(r.Context(), pgWsID)
	memberRole, _ := h.store.GetWorkspaceMemberRole(r.Context(), sqlc.GetWorkspaceMemberRoleParams{
		WorkspaceID: pgWsID,
		UserID:      pgMemberID,
	})
	if memberRole == "owner" && ownerCount <= 1 {
		respond.Error(w, http.StatusBadRequest, "cannot remove the last owner")
		return
	}
	err = h.store.DeleteWorkspaceMember(r.Context(), sqlc.DeleteWorkspaceMemberParams{
		WorkspaceID: pgWsID,
		UserID:      pgMemberID,
	})
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to remove member")
		return
	}
	respond.JSON(w, http.StatusOK, map[string]string{"status": "removed"})
}

func (h *Handler) Activity(w http.ResponseWriter, r *http.Request) {
	wsID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	rows, err := h.store.ListWorkspaceActivity(r.Context(), pgUUID(wsID))
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to list activity")
		return
	}
	type entry struct {
		ID         string         `json:"id"`
		Action     string         `json:"action"`
		EntityType string         `json:"entity_type"`
		EntityID   *string        `json:"entity_id,omitempty"`
		Metadata   map[string]any `json:"metadata"`
		ActorEmail string         `json:"actor_email"`
		CreatedAt  any            `json:"created_at"`
	}
	list := make([]entry, 0, len(rows))
	for _, row := range rows {
		e := entry{
			ID:         fromPgUUID(row.ID).String(),
			Action:     row.Action,
			EntityType: row.EntityType,
			ActorEmail: row.Email,
		}
		if row.EntityID.Valid {
			s := fromPgUUID(row.EntityID).String()
			e.EntityID = &s
		}
		if row.CreatedAt.Valid {
			e.CreatedAt = row.CreatedAt.Time
		}
		_ = json.Unmarshal(row.Metadata, &e.Metadata)
		list = append(list, e)
	}
	respond.JSON(w, http.StatusOK, list)
}

func (h *Handler) logActivity(r *http.Request, wsID, actorID uuid.UUID, action, entityType string, entityID uuid.UUID, meta map[string]any) error {
	b, _ := json.Marshal(meta)
	return h.store.CreateActivityLog(r.Context(), sqlc.CreateActivityLogParams{
		WorkspaceID: pgUUID(wsID),
		ActorID:     pgUUID(actorID),
		Action:      action,
		EntityType:  entityType,
		EntityID:    pgUUID(entityID),
		Metadata:    b,
	})
}

func (h *Handler) RequireRole(minRole string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
			role, err := h.store.GetWorkspaceMemberRole(r.Context(), sqlc.GetWorkspaceMemberRoleParams{
				WorkspaceID: pgUUID(wsID),
				UserID:      pgUUID(userID),
			})
			if err != nil {
				respond.Error(w, http.StatusForbidden, "not a workspace member")
				return
			}
			ranks := map[string]int{"viewer": 1, "editor": 2, "owner": 3}
			if ranks[role] < ranks[minRole] {
				respond.Error(w, http.StatusForbidden, "insufficient permissions")
				return
			}
			ctx := auth.WithWorkspaceContext(r.Context(), wsID, role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
