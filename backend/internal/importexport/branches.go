package importexport

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

func (h *Handler) ListBrunoSourceBranches(w http.ResponseWriter, r *http.Request) {
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
	row, err := h.store.GetBrunoSource(r.Context(), sqlc.GetBrunoSourceParams{
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
	token := ""
	if row.AccessTokenEncrypted != nil && strings.TrimSpace(*row.AccessTokenEncrypted) != "" {
		if h.crypto == nil {
			respond.Error(w, http.StatusInternalServerError, "encryption not configured")
			return
		}
		plain, err := h.crypto.Decrypt(*row.AccessTokenEncrypted)
		if err != nil {
			respond.Error(w, http.StatusInternalServerError, "failed to read credentials")
			return
		}
		token = plain
	}
	svc := gitbranches.NewService(h.store)
	result, err := svc.ListForRepoURL(r.Context(), wsID, repoURL, token, gitbranches.ParseListOptions(r))
	if err != nil {
		respond.Error(w, gitbranches.HTTPStatus(err), err.Error())
		return
	}
	respond.JSON(w, http.StatusOK, result)
}
