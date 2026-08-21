package invite

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

	"github.com/imaanmzr/postchi/backend/internal/auth"
	"github.com/imaanmzr/postchi/backend/internal/db"
	"github.com/imaanmzr/postchi/backend/internal/db/sqlc"
	"github.com/imaanmzr/postchi/backend/internal/shared/config"
	"github.com/imaanmzr/postchi/backend/internal/shared/email"
	"github.com/imaanmzr/postchi/backend/internal/shared/respond"
)

var (
	ErrNotFound = errors.New("invite not found")
	ErrExpired  = errors.New("invite expired")
)

type Handler struct {
	store  *db.Store
	cfg    *config.Config
	mailer *email.Sender
	tokens *auth.Service
}

func NewHandler(store *db.Store, cfg *config.Config, tokens *auth.Service) *Handler {
	return &Handler{store: store, cfg: cfg, mailer: email.NewSender(cfg), tokens: tokens}
}

type Invite struct {
	ID          string `json:"id"`
	WorkspaceID string `json:"workspace_id"`
	Email       string `json:"email"`
	Role        string `json:"role"`
	ExpiresAt   string `json:"expires_at"`
	CreatedAt   string `json:"created_at,omitempty"`
	InviteURL   string `json:"invite_url,omitempty"`
}

type Member struct {
	UserID      string `json:"user_id"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
}

func inviteURL(base, token string) string {
	return strings.TrimRight(base, "/") + "/invite/" + token
}

type InvitePreview struct {
	Email         string `json:"email"`
	WorkspaceName string `json:"workspace_name"`
	ExpiresAt     string `json:"expires_at"`
	UserExists    bool   `json:"user_exists"`
	Expired       bool   `json:"expired"`
}

func newToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	wsID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid workspace id")
		return
	}
	userID, err := auth.UserIDFromContext(r.Context())
	if err != nil {
		respond.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req struct {
		Email     string `json:"email"`
		Role      string `json:"role"`
		SendEmail *bool  `json:"send_email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Email == "" {
		respond.Error(w, http.StatusBadRequest, "email required")
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if req.Role == "" {
		req.Role = "viewer"
	}

	userRow, err := h.store.GetUserByEmail(r.Context(), req.Email)
	if err == nil {
		memberUserID := db.FromPGUUID(userRow.ID)
		err = h.store.UpsertWorkspaceMember(r.Context(), sqlc.UpsertWorkspaceMemberParams{
			WorkspaceID: db.PGUUID(wsID),
			UserID:      userRow.ID,
			Role:        sqlc.WorkspaceRole(req.Role),
		})
		if err != nil {
			respond.Error(w, http.StatusInternalServerError, "failed to add member")
			return
		}
		respond.JSON(w, http.StatusCreated, map[string]any{
			"outcome": "added",
			"member": Member{
				UserID:      memberUserID.String(),
				Email:       req.Email,
				DisplayName: userRow.DisplayName,
				Role:        req.Role,
			},
		})
		return
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		respond.Error(w, http.StatusInternalServerError, "failed to look up user")
		return
	}

	token, err := newToken()
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to create invite")
		return
	}
	expires := time.Now().Add(h.cfg.InviteTTL)

	wsName, _ := h.store.GetWorkspaceName(r.Context(), db.PGUUID(wsID))

	inviteID, err := h.store.UpsertWorkspaceInvite(r.Context(), sqlc.UpsertWorkspaceInviteParams{
		WorkspaceID: db.PGUUID(wsID),
		Email:       req.Email,
		Role:        sqlc.WorkspaceRole(req.Role),
		Token:       token,
		ExpiresAt:   db.PGTimestamptz(expires),
		CreatedBy:   db.PGUUID(userID),
	})
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to save invite")
		return
	}

	url := inviteURL(h.cfg.AppPublicURL, token)
	sendEmail := h.cfg.SMTPConfigured()
	if req.SendEmail != nil {
		sendEmail = *req.SendEmail
	}
	emailSent := false
	if sendEmail && h.cfg.SMTPConfigured() {
		if err := h.mailer.SendInvite(req.Email, wsName, url); err != nil {
			respond.Error(w, http.StatusInternalServerError, "failed to send invite email: "+err.Error())
			return
		}
		emailSent = true
	}

	invite := Invite{
		ID:          db.FromPGUUID(inviteID).String(),
		WorkspaceID: wsID.String(),
		Email:       req.Email,
		Role:        req.Role,
		ExpiresAt:   expires.Format(time.RFC3339),
		InviteURL:   url,
	}
	respond.JSON(w, http.StatusCreated, map[string]any{
		"outcome":     "invited",
		"invite":      invite,
		"invite_url":  url,
		"email_sent":  emailSent,
	})
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	wsID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid workspace id")
		return
	}
	rows, err := h.store.ListPendingWorkspaceInvites(r.Context(), db.PGUUID(wsID))
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to list invites")
		return
	}
	list := make([]Invite, 0, len(rows))
	for _, row := range rows {
		list = append(list, Invite{
			ID:          db.FromPGUUID(row.ID).String(),
			WorkspaceID: db.FromPGUUID(row.WorkspaceID).String(),
			Email:       row.Email,
			Role:        row.Role,
			ExpiresAt:   row.ExpiresAt.Time.Format(time.RFC3339),
			CreatedAt:   row.CreatedAt.Time.Format(time.RFC3339),
			InviteURL:   inviteURL(h.cfg.AppPublicURL, row.Token),
		})
	}
	respond.JSON(w, http.StatusOK, list)
}

func (h *Handler) Revoke(w http.ResponseWriter, r *http.Request) {
	wsID, _ := uuid.Parse(chi.URLParam(r, "id"))
	inviteID, err := uuid.Parse(chi.URLParam(r, "inviteId"))
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid invite id")
		return
	}
	err = h.store.DeleteWorkspaceInvite(r.Context(), sqlc.DeleteWorkspaceInviteParams{
		ID:          db.PGUUID(inviteID),
		WorkspaceID: db.PGUUID(wsID),
	})
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to revoke invite")
		return
	}
	respond.JSON(w, http.StatusOK, map[string]string{"status": "revoked"})
}

func (h *Handler) Preview(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	inv, err := h.loadByToken(r.Context(), token)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			respond.Error(w, http.StatusNotFound, "invite not found")
			return
		}
		respond.Error(w, http.StatusInternalServerError, "failed to load invite")
		return
	}
	wsName, _ := h.store.GetWorkspaceName(r.Context(), db.PGUUID(inv.WorkspaceID))
	exists, _ := h.store.UserExistsByEmail(r.Context(), inv.Email)
	respond.JSON(w, http.StatusOK, InvitePreview{
		Email: inv.Email, WorkspaceName: wsName, ExpiresAt: inv.ExpiresAt.Format(time.RFC3339),
		UserExists: exists, Expired: time.Now().After(inv.ExpiresAt),
	})
}

func (h *Handler) Accept(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	inv, err := h.loadByToken(r.Context(), token)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			respond.Error(w, http.StatusNotFound, "invite not found")
			return
		}
		respond.Error(w, http.StatusInternalServerError, "failed to load invite")
		return
	}
	if time.Now().After(inv.ExpiresAt) {
		respond.Error(w, http.StatusGone, "invite expired; ask the workspace owner for a new invite")
		return
	}
	if inv.AcceptedAt != nil {
		respond.Error(w, http.StatusConflict, "invite already accepted")
		return
	}

	var req struct {
		Password    string `json:"password"`
		DisplayName string `json:"display_name"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	userRow, err := h.store.GetUserByEmail(r.Context(), inv.Email)
	var userID uuid.UUID
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			respond.Error(w, http.StatusInternalServerError, "failed to lookup user")
			return
		}
		if req.Password == "" {
			respond.JSON(w, http.StatusOK, map[string]any{"requires_password": true, "email": inv.Email})
			return
		}
		hash, err := auth.HashPassword(req.Password)
		if err != nil {
			respond.Error(w, http.StatusInternalServerError, "failed to hash password")
			return
		}
		displayName := req.DisplayName
		if displayName == "" {
			displayName = inv.Email
		}
		newID, err := h.store.CreateUser(r.Context(), sqlc.CreateUserParams{
			Email:        inv.Email,
			PasswordHash: hash,
			DisplayName:  displayName,
		})
		if err != nil {
			respond.Error(w, http.StatusConflict, "could not create user")
			return
		}
		userID = db.FromPGUUID(newID)
	} else {
		userID = db.FromPGUUID(userRow.ID)
		if err := h.addMember(r, inv, userID); err != nil {
			respond.Error(w, http.StatusInternalServerError, "failed to add member")
			return
		}
		respond.JSON(w, http.StatusOK, map[string]any{
			"requires_login": true,
			"workspace_id":   inv.WorkspaceID.String(),
			"email":          inv.Email,
		})
		return
	}

	if err := h.addMember(r, inv, userID); err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to add member")
		return
	}

	pair, _, _, err := h.tokens.GenerateTokenPair(userID.String(), inv.Email)
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to create session")
		return
	}
	respond.JSON(w, http.StatusOK, map[string]any{
		"workspace_id": inv.WorkspaceID.String(),
		"tokens":       pair,
		"user": map[string]string{
			"id": userID.String(), "email": inv.Email, "display_name": req.DisplayName,
		},
	})
}

type inviteRow struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	Email       string
	Role        string
	ExpiresAt   time.Time
	AcceptedAt  *time.Time
}

func (h *Handler) loadByToken(ctx context.Context, token string) (*inviteRow, error) {
	row, err := h.store.GetWorkspaceInviteByToken(ctx, token)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	inv := inviteRow{
		ID:          db.FromPGUUID(row.ID),
		WorkspaceID: db.FromPGUUID(row.WorkspaceID),
		Email:       row.Email,
		Role:        row.Role,
		ExpiresAt:   row.ExpiresAt.Time,
	}
	if row.AcceptedAt.Valid {
		t := row.AcceptedAt.Time
		inv.AcceptedAt = &t
	}
	return &inv, nil
}

func (h *Handler) addMember(r *http.Request, inv *inviteRow, userID uuid.UUID) error {
	if err := h.store.UpsertWorkspaceMember(r.Context(), sqlc.UpsertWorkspaceMemberParams{
		WorkspaceID: db.PGUUID(inv.WorkspaceID),
		UserID:      db.PGUUID(userID),
		Role:        sqlc.WorkspaceRole(inv.Role),
	}); err != nil {
		return err
	}
	return h.store.MarkWorkspaceInviteAccepted(r.Context(), db.PGUUID(inv.ID))
}
