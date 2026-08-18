package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/imaanmzr/postchi/backend/internal/db"
	"github.com/imaanmzr/postchi/backend/internal/db/sqlc"
	"github.com/imaanmzr/postchi/backend/internal/shared/config"
	"github.com/imaanmzr/postchi/backend/internal/shared/respond"
)

type Handler struct {
	store  *db.Store
	tokens *Service
	cfg    *config.Config
}

func NewHandler(store *db.Store, tokens *Service, cfg *config.Config) *Handler {
	return &Handler{store: store, tokens: tokens, cfg: cfg}
}

type registerRequest struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type authResponse struct {
	User      userResponse `json:"user"`
	TokenPair *TokenPair   `json:"tokens"`
}

type userResponse struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Email == "" || req.Password == "" {
		respond.Error(w, http.StatusBadRequest, "email and password required")
		return
	}
	hash, err := HashPassword(req.Password)
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to hash password")
		return
	}
	if req.DisplayName == "" {
		req.DisplayName = req.Email
	}

	userID, err := h.store.CreateUser(r.Context(), sqlc.CreateUserParams{
		Email:        req.Email,
		PasswordHash: hash,
		DisplayName:  req.DisplayName,
	})
	if err != nil {
		respond.Error(w, http.StatusConflict, "email already registered")
		return
	}

	uid := pgUUIDToUUID(userID)
	pair, refreshHash, expiresAt, err := h.tokens.GenerateTokenPair(uid.String(), req.Email)
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to generate tokens")
		return
	}
	_ = h.store.CreateRefreshToken(r.Context(), sqlc.CreateRefreshTokenParams{
		UserID:    userID,
		TokenHash: refreshHash,
		ExpiresAt: timeToPgTimestamptz(expiresAt),
	})

	respond.JSON(w, http.StatusCreated, authResponse{
		User:      userResponse{ID: uid.String(), Email: req.Email, DisplayName: req.DisplayName},
		TokenPair: pair,
	})
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, err := h.store.GetUserByEmail(r.Context(), req.Email)
	if err != nil || !VerifyPassword(user.PasswordHash, req.Password) {
		respond.Error(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	uid := pgUUIDToUUID(user.ID)
	pair, refreshHash, expiresAt, err := h.tokens.GenerateTokenPair(uid.String(), req.Email)
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to generate tokens")
		return
	}
	_ = h.store.CreateRefreshToken(r.Context(), sqlc.CreateRefreshTokenParams{
		UserID:    user.ID,
		TokenHash: refreshHash,
		ExpiresAt: timeToPgTimestamptz(expiresAt),
	})

	respond.JSON(w, http.StatusOK, authResponse{
		User:      userResponse{ID: uid.String(), Email: req.Email, DisplayName: user.DisplayName},
		TokenPair: pair,
	})
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	hash := HashToken(req.RefreshToken)

	rt, err := h.store.GetRefreshTokenWithUser(r.Context(), hash)
	if err != nil || !rt.ExpiresAt.Valid || time.Now().After(rt.ExpiresAt.Time) {
		respond.Error(w, http.StatusUnauthorized, "invalid refresh token")
		return
	}

	_ = h.store.DeleteRefreshTokenByHash(r.Context(), hash)

	userID := pgUUIDToUUID(rt.UserID)
	pair, refreshHash, newExpires, err := h.tokens.GenerateTokenPair(userID.String(), rt.Email)
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to generate tokens")
		return
	}
	_ = h.store.CreateRefreshToken(r.Context(), sqlc.CreateRefreshTokenParams{
		UserID:    rt.UserID,
		TokenHash: refreshHash,
		ExpiresAt: timeToPgTimestamptz(newExpires),
	})

	respond.JSON(w, http.StatusOK, pair)
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.RefreshToken != "" {
		_ = h.store.DeleteRefreshTokenByHash(r.Context(), HashToken(req.RefreshToken))
	}
	respond.JSON(w, http.StatusOK, map[string]string{"status": "logged out"})
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	userID, err := UserIDFromContext(r.Context())
	if err != nil {
		respond.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	user, err := h.store.GetUserByID(r.Context(), uuidToPgUUID(userID))
	if err != nil {
		respond.Error(w, http.StatusNotFound, "user not found")
		return
	}
	respond.JSON(w, http.StatusOK, userResponse{ID: userID.String(), Email: user.Email, DisplayName: user.DisplayName})
}

func (h *Handler) RequireWorkspaceRole(minRole string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, err := UserIDFromContext(r.Context())
			if err != nil {
				respond.Error(w, http.StatusUnauthorized, "unauthorized")
				return
			}
			workspaceID, err := parseWorkspaceID(r)
			if err != nil {
				respond.Error(w, http.StatusBadRequest, "invalid workspace id")
				return
			}
			role, err := h.store.GetWorkspaceMemberRole(r.Context(), sqlc.GetWorkspaceMemberRoleParams{
				WorkspaceID: uuidToPgUUID(workspaceID),
				UserID:      uuidToPgUUID(userID),
			})
			if err != nil {
				respond.Error(w, http.StatusForbidden, "not a workspace member")
				return
			}
			if !hasMinRole(role, minRole) {
				respond.Error(w, http.StatusForbidden, "insufficient permissions")
				return
			}
			ctx := context.WithValue(r.Context(), workspaceRoleKey{}, role)
			ctx = context.WithValue(ctx, workspaceIDKey{}, workspaceID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

type workspaceRoleKey struct{}
type workspaceIDKey struct{}

func WorkspaceRole(ctx context.Context) string {
	v, _ := ctx.Value(workspaceRoleKey{}).(string)
	return v
}

func WorkspaceID(ctx context.Context) uuid.UUID {
	v, _ := ctx.Value(workspaceIDKey{}).(uuid.UUID)
	return v
}

func parseWorkspaceID(r *http.Request) (uuid.UUID, error) {
	idStr := r.URL.Query().Get("workspace_id")
	if idStr == "" {
		// chi URL param
		if v := r.Context().Value("workspaceID"); v != nil {
			if s, ok := v.(string); ok {
				idStr = s
			}
		}
	}
	return uuid.Parse(idStr)
}

func hasMinRole(actual, required string) bool {
	rank := map[string]int{"viewer": 1, "editor": 2, "owner": 3}
	return rank[actual] >= rank[required]
}

func uuidToPgUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}

func pgUUIDToUUID(id pgtype.UUID) uuid.UUID {
	return uuid.UUID(id.Bytes)
}

func timeToPgTimestamptz(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}
