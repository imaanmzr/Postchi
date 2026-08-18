package workspacetoken

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
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
	"github.com/imaanmzr/postchi/backend/internal/shared/respond"
)

type Handler struct {
	store *db.Store
}

func NewHandler(store *db.Store) *Handler {
	return &Handler{store: store}
}

type TokenInfo struct {
	ID          string   `json:"id"`
	WorkspaceID string   `json:"workspace_id"`
	Name        string   `json:"name"`
	TokenPrefix string   `json:"token_prefix"`
	Scopes      []string `json:"scopes"`
	CreatedAt   string   `json:"created_at"`
	RevokedAt   *string  `json:"revoked_at,omitempty"`
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	wsID, _ := uuid.Parse(chi.URLParam(r, "id"))
	rows, err := h.store.ListWorkspaceApiTokens(r.Context(), db.PGUUID(wsID))
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "query failed")
		return
	}
	list := make([]TokenInfo, 0, len(rows))
	for _, row := range rows {
		t := TokenInfo{
			ID:          db.FromPGUUID(row.ID).String(),
			WorkspaceID: db.FromPGUUID(row.WorkspaceID).String(),
			Name:        row.Name,
			TokenPrefix: row.TokenPrefix,
			Scopes:      row.Scopes,
			CreatedAt:   row.CreatedAt.Time.Format(time.RFC3339),
		}
		if row.RevokedAt.Valid {
			s := row.RevokedAt.Time.Format(time.RFC3339)
			t.RevokedAt = &s
		}
		list = append(list, t)
	}
	respond.JSON(w, http.StatusOK, list)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())
	wsID, _ := uuid.Parse(chi.URLParam(r, "id"))
	var req struct {
		Name   string   `json:"name"`
		Scopes []string `json:"scopes"`
	}
	if err := jsonDecode(r, &req); err != nil || req.Name == "" {
		respond.Error(w, http.StatusBadRequest, "name required")
		return
	}
	if len(req.Scopes) == 0 {
		req.Scopes = []string{"spec:push"}
	}
	raw := make([]byte, 32)
	_, _ = rand.Read(raw)
	plain := "pst_" + hex.EncodeToString(raw)
	hash := hashToken(plain)
	prefix := plain[:12]
	id, err := h.store.CreateWorkspaceApiToken(r.Context(), sqlc.CreateWorkspaceApiTokenParams{
		WorkspaceID: db.PGUUID(wsID),
		Name:        req.Name,
		TokenHash:   hash,
		TokenPrefix: prefix,
		Scopes:      req.Scopes,
		CreatedBy:   db.PGUUID(userID),
	})
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to create token")
		return
	}
	respond.JSON(w, http.StatusCreated, map[string]any{
		"id": db.FromPGUUID(id).String(), "token": plain, "token_prefix": prefix, "scopes": req.Scopes,
	})
}

func (h *Handler) Revoke(w http.ResponseWriter, r *http.Request) {
	id, _ := uuid.Parse(chi.URLParam(r, "tokenId"))
	_ = h.store.RevokeWorkspaceApiToken(r.Context(), db.PGUUID(id))
	respond.JSON(w, http.StatusOK, map[string]string{"status": "revoked"})
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

type tokenContextKey struct{}

type TokenAuth struct {
	WorkspaceID uuid.UUID
	Scopes      []string
}

func ContextFromRequest(ctx context.Context) (*TokenAuth, bool) {
	v, ok := ctx.Value(tokenContextKey{}).(*TokenAuth)
	return v, ok
}

func (h *Handler) RequireScope(scope string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if !strings.HasPrefix(authHeader, "Bearer pst_") {
				respond.Error(w, http.StatusUnauthorized, "workspace token required")
				return
			}
			plain := strings.TrimPrefix(authHeader, "Bearer ")
			hash := hashToken(plain)
			row, err := h.store.GetWorkspaceApiTokenByHash(r.Context(), hash)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					respond.Error(w, http.StatusUnauthorized, "invalid token")
					return
				}
				respond.Error(w, http.StatusInternalServerError, "token lookup failed")
				return
			}
			if row.RevokedAt.Valid {
				respond.Error(w, http.StatusUnauthorized, "invalid token")
				return
			}
			wsID := db.FromPGUUID(row.WorkspaceID)
			wsParam := chi.URLParam(r, "id")
			if wsParam != "" {
				paramID, err := uuid.Parse(wsParam)
				if err != nil || paramID != wsID {
					respond.Error(w, http.StatusForbidden, "token workspace mismatch")
					return
				}
			}
			hasScope := false
			for _, s := range row.Scopes {
				if s == scope {
					hasScope = true
					break
				}
			}
			if !hasScope {
				respond.Error(w, http.StatusForbidden, "insufficient token scope")
				return
			}
			ctx := context.WithValue(r.Context(), tokenContextKey{}, &TokenAuth{WorkspaceID: wsID, Scopes: row.Scopes})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func jsonDecode(r *http.Request, v any) error {
	dec := json.NewDecoder(r.Body)
	return dec.Decode(v)
}
