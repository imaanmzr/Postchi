package docsync

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/imaanmzr/postchi/backend/internal/db"
	"github.com/imaanmzr/postchi/backend/internal/db/sqlc"
	"github.com/imaanmzr/postchi/backend/internal/gitbranches"
	"github.com/imaanmzr/postchi/backend/internal/shared/respond"
)

func (h *Handler) ListSourceBranches(w http.ResponseWriter, r *http.Request) {
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
	row, err := h.store.GetDocSource(r.Context(), sqlc.GetDocSourceParams{
		ID:          db.PGUUID(sourceID),
		WorkspaceID: db.PGUUID(wsID),
	})
	if err != nil {
		respond.Error(w, http.StatusNotFound, "not found")
		return
	}
	var config map[string]any
	if err := json.Unmarshal(row.Config, &config); err != nil {
		respond.Error(w, http.StatusInternalServerError, "invalid source config")
		return
	}
	repoURL, _ := config["repo_url"].(string)
	if strings.TrimSpace(repoURL) == "" {
		respond.Error(w, http.StatusBadRequest, "source has no repository URL")
		return
	}
	token, err := h.decryptOptionalToken(row.AccessTokenEncrypted)
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to read credentials")
		return
	}
	svc := gitbranches.NewService(h.store)
	result, err := svc.ListForRepoURL(r.Context(), wsID, repoURL, token, gitbranches.ParseListOptions(r))
	if err != nil {
		respond.Error(w, gitbranches.HTTPStatus(err), err.Error())
		return
	}
	respond.JSON(w, http.StatusOK, result)
}

func (h *Handler) PreviewBranches(w http.ResponseWriter, r *http.Request) {
	wsID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid workspace id")
		return
	}
	var req struct {
		RepoURL     string `json:"repo_url"`
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.RepoURL) == "" {
		respond.Error(w, http.StatusBadRequest, "repo_url is required")
		return
	}
	svc := gitbranches.NewService(h.store)
	result, err := svc.ListForRepoURL(
		r.Context(),
		wsID,
		req.RepoURL,
		strings.TrimSpace(req.AccessToken),
		gitbranches.ParseListOptions(r),
	)
	if err != nil {
		respond.Error(w, gitbranches.HTTPStatus(err), err.Error())
		return
	}
	respond.JSON(w, http.StatusOK, result)
}

func (h *Handler) decryptOptionalToken(tokenEnc *string) (string, error) {
	if tokenEnc == nil || strings.TrimSpace(*tokenEnc) == "" {
		return "", nil
	}
	return h.crypto.Decrypt(*tokenEnc)
}
