package importexport

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/imaanmzr/postchi/backend/internal/auth"
	"github.com/imaanmzr/postchi/backend/internal/importexport/gitsync"
	"github.com/imaanmzr/postchi/backend/internal/shared/gitrepo"
	"github.com/imaanmzr/postchi/backend/internal/shared/respond"
)

type gitCollectionImportRequest struct {
	Name        string `json:"name"`
	RepoURL     string `json:"repo_url"`
	Branch      string `json:"branch"`
	PathPrefix  string `json:"path_prefix"`
	AccessToken string `json:"access_token"`
	importParentRequest
}

func (h *Handler) ImportCollectionGit(w http.ResponseWriter, r *http.Request) {
	h.importCollectionFromGit(w, r)
}

func (h *Handler) ImportBrunoGit(w http.ResponseWriter, r *http.Request) {
	h.importCollectionFromGit(w, r)
}

func (h *Handler) importCollectionFromGit(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.UserIDFromContext(r.Context())
	if err != nil {
		respond.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	workspaceID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid workspace id")
		return
	}
	if err := h.validateWorkspaceEditor(r.Context(), workspaceID, userID); err != nil {
		respond.Error(w, http.StatusForbidden, err.Error())
		return
	}
	var request gitCollectionImportRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	request.Name = strings.TrimSpace(request.Name)
	request.RepoURL = strings.TrimSpace(request.RepoURL)
	request.Branch = strings.TrimSpace(request.Branch)
	request.PathPrefix = strings.Trim(strings.TrimSpace(request.PathPrefix), "/")
	if request.Name == "" || request.RepoURL == "" {
		respond.Error(w, http.StatusBadRequest, "name and repo_url are required")
		return
	}
	parentID, err := h.resolveImportParent(r.Context(), workspaceID, userID, request.importParentRequest)
	if err != nil {
		respond.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), gitBrunoImportTimeout)
	defer cancel()
	client, err := gitrepo.New(gitrepo.Config{
		RepoURL:      request.RepoURL,
		Branch:       request.Branch,
		PathPrefix:   request.PathPrefix,
		Token:        request.AccessToken,
		MaxTreeFiles: 10_000,
		MaxFileBytes: maxGitBrunoFileBytes,
	})
	request.AccessToken = ""
	if err != nil {
		writeGitImportError(w, err)
		return
	}

	files, err := gitsync.FetchRepository(ctx, client)
	if err != nil {
		writeGitImportError(w, err)
		return
	}
	roots := gitsync.Discover(files)
	parsed := parseRepositoryRoots(roots, request.Name)
	if len(parsed.Collections) == 0 {
		msg := "repository contains no importable collections"
		if len(parsed.Errors) > 0 {
			msg = strings.Join(parsed.Errors, "; ")
		}
		respond.Error(w, http.StatusUnprocessableEntity, msg)
		return
	}

	result := ImportResult{Errors: parsed.Errors}
	var firstColID uuid.UUID
	for i, col := range parsed.Collections {
		colID, partial, err := h.persistCollection(ctx, workspaceID, userID, col, parentID)
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
				respond.Error(w, http.StatusGatewayTimeout, "Git import timed out")
				return
			}
			respond.Error(w, http.StatusInternalServerError, "import failed: "+err.Error())
			return
		}
		if i == 0 {
			firstColID = colID
		}
		result.Collections += partial.Collections
		result.Requests += partial.Requests
	}
	result.CollectionID = firstColID.String()
	respond.JSON(w, http.StatusCreated, result)
}
