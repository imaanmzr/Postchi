package importexport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/imaanmzr/postchi/backend/internal/auth"
	"github.com/imaanmzr/postchi/backend/internal/shared/gitrepo"
	"github.com/imaanmzr/postchi/backend/internal/shared/respond"
)

const (
	maxGitBrunoFiles       = 2_000
	maxGitBrunoFileBytes   = 2 << 20
	maxGitBrunoTotalBytes  = 32 << 20
	maxGitBrunoConcurrency = 8
	gitBrunoImportTimeout  = 5 * time.Minute
)

type gitBrunoImportRequest struct {
	Name        string `json:"name"`
	RepoURL     string `json:"repo_url"`
	Branch      string `json:"branch"`
	PathPrefix  string `json:"path_prefix"`
	AccessToken string `json:"access_token"`
}

func (h *Handler) ImportBrunoGit(w http.ResponseWriter, r *http.Request) {
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
	var request gitBrunoImportRequest
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

	files, err := fetchBrunoRepository(ctx, client)
	if err != nil {
		writeGitImportError(w, err)
		return
	}
	collection, err := parseBrunoFiles(files, brunoParseOptions{
		RootName:         request.Name,
		RequireRootMeta:  true,
		ValidateRequests: true,
	})
	if err != nil {
		respond.Error(w, http.StatusUnprocessableEntity, "invalid Bruno repository: "+err.Error())
		return
	}
	_, requests := countCollectionTree(collection)
	if requests == 0 {
		respond.Error(w, http.StatusUnprocessableEntity, "repository contains no Bruno request files")
		return
	}

	_, result, err := h.persistCollection(ctx, workspaceID, userID, collection, nil)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
			respond.Error(w, http.StatusGatewayTimeout, "Git import timed out")
			return
		}
		respond.Error(w, http.StatusInternalServerError, "Bruno import failed")
		return
	}
	respond.JSON(w, http.StatusCreated, result)
}

func fetchBrunoRepository(ctx context.Context, client *gitrepo.Client) ([]brunoSourceFile, error) {
	paths, err := client.ListFiles(ctx)
	if err != nil {
		return nil, err
	}
	candidates := make([]string, 0)
	for _, filePath := range paths {
		relative := relativeRepositoryPath(filePath, client.PathPrefix)
		parts := strings.Split(relative, "/")
		if relative == "" || !strings.HasSuffix(strings.ToLower(relative), ".bru") ||
			containsPathSegment(parts[:len(parts)-1], "environments") {
			continue
		}
		candidates = append(candidates, filePath)
		if len(candidates) > maxGitBrunoFiles {
			return nil, &gitrepo.Error{
				Kind:    gitrepo.ErrorLimit,
				Message: "repository contains more than 2,000 Bruno files; use a narrower path prefix",
			}
		}
	}
	if len(candidates) == 0 {
		return nil, &gitrepo.Error{Kind: gitrepo.ErrorNotFound, Message: "repository contains no Bruno request files"}
	}

	type fetchResult struct {
		file brunoSourceFile
		err  error
	}
	results := make(chan fetchResult, len(candidates))
	semaphore := make(chan struct{}, maxGitBrunoConcurrency)
	var group sync.WaitGroup
	for _, filePath := range candidates {
		group.Add(1)
		go func(repoPath string) {
			defer group.Done()
			select {
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()
			case <-ctx.Done():
				results <- fetchResult{err: ctx.Err()}
				return
			}
			content, fetchErr := client.FetchFile(ctx, repoPath)
			results <- fetchResult{
				file: brunoSourceFile{Path: relativeRepositoryPath(repoPath, client.PathPrefix), Content: content},
				err:  fetchErr,
			}
		}(filePath)
	}
	go func() {
		group.Wait()
		close(results)
	}()

	files := make([]brunoSourceFile, 0, len(candidates))
	totalBytes := 0
	for result := range results {
		if result.err != nil {
			return nil, result.err
		}
		totalBytes += len(result.file.Content)
		if totalBytes > maxGitBrunoTotalBytes {
			return nil, &gitrepo.Error{
				Kind:    gitrepo.ErrorLimit,
				Message: "Bruno files exceed the 32 MiB total import limit",
			}
		}
		files = append(files, result.file)
	}
	return files, nil
}

func relativeRepositoryPath(filePath, prefix string) string {
	cleanPath := strings.Trim(path.Clean("/"+filePath), "/")
	cleanPrefix := strings.Trim(path.Clean("/"+prefix), "/")
	if cleanPrefix == "" || cleanPrefix == "." {
		return cleanPath
	}
	if cleanPath == cleanPrefix {
		return ""
	}
	return strings.TrimPrefix(cleanPath, cleanPrefix+"/")
}

func writeGitImportError(w http.ResponseWriter, err error) {
	status := http.StatusBadGateway
	var repoError *gitrepo.Error
	_ = errors.As(err, &repoError)
	switch gitrepo.Kind(err) {
	case gitrepo.ErrorInvalid:
		status = http.StatusBadRequest
	case gitrepo.ErrorAuthentication:
		status = http.StatusUnauthorized
		if repoError != nil && repoError.Status == http.StatusForbidden {
			status = http.StatusForbidden
		}
	case gitrepo.ErrorNotFound:
		status = http.StatusUnprocessableEntity
		if repoError != nil && repoError.Status == http.StatusNotFound {
			status = http.StatusNotFound
		}
	case gitrepo.ErrorRateLimited:
		status = http.StatusTooManyRequests
	case gitrepo.ErrorTimeout:
		status = http.StatusGatewayTimeout
	case gitrepo.ErrorLimit:
		status = http.StatusRequestEntityTooLarge
	}
	message := strings.TrimSpace(err.Error())
	if message == "" {
		message = fmt.Sprintf("Git provider request failed (%s)", gitrepo.Kind(err))
	}
	respond.Error(w, status, message)
}
