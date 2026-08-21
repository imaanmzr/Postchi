package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/imaanmzr/postchi/backend/internal/db/sqlc"
	"github.com/imaanmzr/postchi/backend/internal/shared/respond"
)

const forgotPasswordMessage = "If an account exists for that email, a password reset link has been sent."

type resetPasswordPreview struct {
	Email     string `json:"email"`
	ExpiresAt string `json:"expires_at"`
	Expired   bool   `json:"expired"`
}

func resetPasswordURL(base, token string) string {
	return strings.TrimRight(base, "/") + "/reset-password/" + token
}

func (h *Handler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if req.Email == "" {
		respond.Error(w, http.StatusBadRequest, "email required")
		return
	}

	if h.cfg.SMTPConfigured() {
		user, err := h.store.GetUserAuthByEmail(r.Context(), req.Email)
		if err == nil && user.AuthProvider == "local" {
			rawToken, err := generateRandomToken(32)
			if err != nil {
				respond.Error(w, http.StatusInternalServerError, "failed to generate reset token")
				return
			}
			tokenHash := HashToken(rawToken)
			expiresAt := time.Now().Add(h.cfg.PasswordResetTTL)

			_ = h.store.DeletePasswordResetTokensByUserID(r.Context(), user.ID)
			if err := h.store.CreatePasswordResetToken(r.Context(), sqlc.CreatePasswordResetTokenParams{
				UserID:    user.ID,
				TokenHash: tokenHash,
				ExpiresAt: timeToPgTimestamptz(expiresAt),
			}); err != nil {
				respond.Error(w, http.StatusInternalServerError, "failed to create reset token")
				return
			}

			resetURL := resetPasswordURL(h.cfg.AppPublicURL, rawToken)
			if err := h.mailer.SendPasswordReset(req.Email, resetURL); err != nil {
				_ = h.store.DeletePasswordResetTokenByHash(r.Context(), tokenHash)
			}
		}
	}

	respond.JSON(w, http.StatusOK, map[string]string{"message": forgotPasswordMessage})
}

func (h *Handler) PreviewResetPassword(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	row, err := h.store.GetPasswordResetTokenByHash(r.Context(), HashToken(token))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			respond.Error(w, http.StatusNotFound, "reset link not found")
			return
		}
		respond.Error(w, http.StatusInternalServerError, "failed to load reset link")
		return
	}
	if row.AuthProvider != "local" {
		respond.Error(w, http.StatusBadRequest, "password reset is not available for this account")
		return
	}
	expired := !row.ExpiresAt.Valid || time.Now().After(row.ExpiresAt.Time)
	respond.JSON(w, http.StatusOK, resetPasswordPreview{
		Email:     row.Email,
		ExpiresAt: row.ExpiresAt.Time.Format(time.RFC3339),
		Expired:   expired,
	})
}

func (h *Handler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	tokenHash := HashToken(token)

	row, err := h.store.GetPasswordResetTokenByHash(r.Context(), tokenHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			respond.Error(w, http.StatusNotFound, "reset link not found")
			return
		}
		respond.Error(w, http.StatusInternalServerError, "failed to load reset link")
		return
	}
	if row.AuthProvider != "local" {
		respond.Error(w, http.StatusBadRequest, "password reset is not available for this account")
		return
	}
	if !row.ExpiresAt.Valid || time.Now().After(row.ExpiresAt.Time) {
		respond.Error(w, http.StatusGone, "reset link expired; request a new one")
		return
	}

	var req struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Password == "" {
		respond.Error(w, http.StatusBadRequest, "password required")
		return
	}

	hash, err := HashPassword(req.Password)
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to hash password")
		return
	}
	if err := h.store.UpdateUserPassword(r.Context(), sqlc.UpdateUserPasswordParams{
		ID:           row.UserID,
		PasswordHash: hash,
	}); err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to update password")
		return
	}
	_ = h.store.DeleteRefreshTokensByUserID(r.Context(), row.UserID)
	_ = h.store.DeletePasswordResetTokenByHash(r.Context(), tokenHash)

	respond.JSON(w, http.StatusOK, map[string]string{"status": "password_reset"})
}
