package gitbranches

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/imaanmzr/postchi/backend/internal/db"
	"github.com/imaanmzr/postchi/backend/internal/db/sqlc"
	"github.com/imaanmzr/postchi/backend/internal/shared/gitrepo"
)

const cacheTTL = 5 * time.Minute

// ListOptions controls branch listing behavior.
type ListOptions struct {
	Search  string
	Limit   int
	Refresh bool
}

// ListResponse is returned by branch list endpoints.
type ListResponse struct {
	Branches  []gitrepo.Branch `json:"branches"`
	Cached    bool             `json:"cached"`
	FetchedAt time.Time        `json:"fetched_at"`
}

// Service lists repository branches with Postgres caching.
type Service struct {
	store *db.Store
}

func NewService(store *db.Store) *Service {
	return &Service{store: store}
}

func (s *Service) ListForRepoURL(
	ctx context.Context,
	workspaceID uuid.UUID,
	repoURL string,
	token string,
	opts ListOptions,
) (ListResponse, error) {
	parsed, err := gitrepo.ParseRepoFromURL(repoURL)
	if err != nil {
		return ListResponse{}, err
	}
	client, err := gitrepo.ClientForRepo(parsed.RepoURL, token)
	if err != nil {
		return ListResponse{}, err
	}
	repoKey := gitrepo.RepoKey(client.Provider, client.APIBaseURL, client.RepoPath)
	return s.list(ctx, workspaceID, repoKey, client, opts)
}

func (s *Service) list(
	ctx context.Context,
	workspaceID uuid.UUID,
	repoKey string,
	client *gitrepo.Client,
	opts ListOptions,
) (ListResponse, error) {
	if !opts.Refresh {
		if cached, ok, err := s.loadCache(ctx, workspaceID, repoKey); err != nil {
			return ListResponse{}, err
		} else if ok {
			return ListResponse{
				Branches:  gitrepo.FilterBranches(cached.branches, opts.Search, opts.Limit),
				Cached:    true,
				FetchedAt: cached.fetchedAt,
			}, nil
		}
	}

	branches, err := client.ListBranches(ctx)
	if err != nil {
		return ListResponse{}, err
	}
	fetchedAt := time.Now().UTC()
	if err := s.saveCache(ctx, workspaceID, repoKey, branches); err != nil {
		return ListResponse{}, err
	}
	return ListResponse{
		Branches:  gitrepo.FilterBranches(branches, opts.Search, opts.Limit),
		Cached:    false,
		FetchedAt: fetchedAt,
	}, nil
}

type cachedBranches struct {
	branches  []gitrepo.Branch
	fetchedAt time.Time
}

func (s *Service) loadCache(ctx context.Context, workspaceID uuid.UUID, repoKey string) (cachedBranches, bool, error) {
	row, err := s.store.GetGitBranchCache(ctx, sqlc.GetGitBranchCacheParams{
		WorkspaceID: db.PGUUID(workspaceID),
		RepoKey:     repoKey,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return cachedBranches{}, false, nil
		}
		return cachedBranches{}, false, err
	}
	if !row.FetchedAt.Valid {
		return cachedBranches{}, false, nil
	}
	fetchedAt := row.FetchedAt.Time.UTC()
	if time.Since(fetchedAt) > cacheTTL {
		return cachedBranches{}, false, nil
	}
	var branches []gitrepo.Branch
	if err := json.Unmarshal(row.Branches, &branches); err != nil {
		return cachedBranches{}, false, err
	}
	return cachedBranches{branches: branches, fetchedAt: fetchedAt}, true, nil
}

func (s *Service) saveCache(ctx context.Context, workspaceID uuid.UUID, repoKey string, branches []gitrepo.Branch) error {
	payload, err := json.Marshal(branches)
	if err != nil {
		return err
	}
	return s.store.UpsertGitBranchCache(ctx, sqlc.UpsertGitBranchCacheParams{
		WorkspaceID: db.PGUUID(workspaceID),
		RepoKey:     repoKey,
		Branches:    payload,
	})
}

// HTTPStatus maps git provider errors to HTTP status codes.
func HTTPStatus(err error) int {
	switch gitrepo.Kind(err) {
	case gitrepo.ErrorInvalid:
		return 400
	case gitrepo.ErrorAuthentication:
		return 400
	case gitrepo.ErrorNotFound:
		return 404
	case gitrepo.ErrorRateLimited:
		return 429
	case gitrepo.ErrorTimeout:
		return 504
	default:
		return 502
	}
}
