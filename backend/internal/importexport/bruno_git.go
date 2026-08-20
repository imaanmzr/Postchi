package importexport

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/imaanmzr/postchi/backend/internal/importexport/gitsync"
	"github.com/imaanmzr/postchi/backend/internal/shared/gitrepo"
	"github.com/imaanmzr/postchi/backend/internal/shared/respond"
)

const (
	maxGitBrunoFiles       = gitsync.MaxFiles
	maxGitBrunoFileBytes   = gitsync.MaxFileBytes
	maxGitBrunoTotalBytes  = gitsync.MaxTotalBytes
	maxGitBrunoConcurrency = gitsync.MaxConcurrency
	gitBrunoImportTimeout  = 5 * time.Minute
)

func fetchBrunoRepository(ctx context.Context, client *gitrepo.Client) ([]brunoSourceFile, error) {
	files, err := gitsync.FetchRepository(ctx, client)
	if err != nil {
		return nil, err
	}
	out := make([]brunoSourceFile, 0, len(files))
	for _, file := range files {
		if !strings.HasSuffix(strings.ToLower(file.Path), ".bru") {
			continue
		}
		out = append(out, brunoSourceFile{Path: file.Path, Content: file.Content})
	}
	if len(out) == 0 {
		return nil, &gitrepo.Error{Kind: gitrepo.ErrorNotFound, Message: "repository contains no Bruno request files"}
	}
	return out, nil
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
