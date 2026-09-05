package gitrepo

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const maxBranchPages = 10

// Branch describes a repository branch ref.
type Branch struct {
	Name      string `json:"name"`
	IsDefault bool   `json:"is_default,omitempty"`
	Protected bool   `json:"protected,omitempty"`
}

func (c *Client) ListBranches(ctx context.Context) ([]Branch, error) {
	if c.Provider == ProviderGitLab {
		return c.listGitLabBranches(ctx)
	}
	return c.listGitHubBranches(ctx)
}

func (c *Client) listGitHubBranches(ctx context.Context) ([]Branch, error) {
	defaultBranch, _ := c.fetchGitHubDefaultBranch(ctx)
	apiRoot := strings.TrimSuffix(c.APIBaseURL, "/")
	var branches []Branch
	for page := 1; page <= maxBranchPages; page++ {
		listURL := fmt.Sprintf("%s/repos/%s/branches?per_page=100&page=%d", apiRoot, c.RepoPath, page)
		resp, err := c.doRequest(ctx, http.MethodGet, listURL)
		if err != nil {
			return nil, requestError("GitHub branches request failed", err)
		}
		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			return nil, requestError("GitHub branches response failed", readErr)
		}
		if resp.StatusCode >= 400 {
			return nil, responseErrorBody("GitHub", resp.StatusCode, body)
		}
		var items []struct {
			Name       string `json:"name"`
			Protected  bool   `json:"protected"`
		}
		if err := json.Unmarshal(body, &items); err != nil {
			return nil, &Error{Kind: ErrorProvider, Message: "invalid GitHub branches response"}
		}
		for _, item := range items {
			if safe, ok := SanitizeBranchName(item.Name); ok {
				branches = append(branches, Branch{
					Name:      safe,
					IsDefault: defaultBranch != "" && safe == defaultBranch,
					Protected: item.Protected,
				})
			}
		}
		if len(items) < 100 {
			break
		}
	}
	if len(branches) == 0 {
		return nil, &Error{Kind: ErrorNotFound, Message: "no branches found for repository"}
	}
	return branches, nil
}

func (c *Client) fetchGitHubDefaultBranch(ctx context.Context) (string, error) {
	apiRoot := strings.TrimSuffix(c.APIBaseURL, "/")
	repoURL := fmt.Sprintf("%s/repos/%s", apiRoot, c.RepoPath)
	resp, err := c.doRequest(ctx, http.MethodGet, repoURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return "", nil
	}
	var result struct {
		DefaultBranch string `json:"default_branch"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	return strings.TrimSpace(result.DefaultBranch), nil
}

func (c *Client) listGitLabBranches(ctx context.Context) ([]Branch, error) {
	if c.Token == "" {
		return nil, &Error{Kind: ErrorAuthentication, Message: "access token required for GitLab repositories"}
	}
	apiRoot := strings.TrimSuffix(c.APIBaseURL, "/") + "/api/v4"
	nextURL := fmt.Sprintf("%s/projects/%s/repository/branches?per_page=100",
		apiRoot, encodeGitLabPath(c.RepoPath))
	var branches []Branch
	seenPages := map[string]bool{}
	for nextURL != "" && len(seenPages) < maxBranchPages {
		if seenPages[nextURL] {
			return nil, &Error{Kind: ErrorProvider, Message: "GitLab returned a pagination loop"}
		}
		seenPages[nextURL] = true
		resp, err := c.doRequest(ctx, http.MethodGet, nextURL)
		if err != nil {
			return nil, requestError("GitLab branches request failed", err)
		}
		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			return nil, requestError("GitLab branches response failed", readErr)
		}
		if resp.StatusCode >= 400 {
			return nil, responseErrorBody("GitLab", resp.StatusCode, body)
		}
		var items []struct {
			Name      string `json:"name"`
			Default   bool   `json:"default"`
			Protected bool   `json:"protected"`
		}
		if err := json.Unmarshal(body, &items); err != nil {
			return nil, &Error{Kind: ErrorProvider, Message: "invalid GitLab branches response"}
		}
		for _, item := range items {
			if safe, ok := SanitizeBranchName(item.Name); ok {
				branches = append(branches, Branch{
					Name:      safe,
					IsDefault: item.Default,
					Protected: item.Protected,
				})
			}
		}
		if len(items) < 100 {
			break
		}
		nextURL = parseNextLink(resp.Header.Get("Link"))
	}
	if len(branches) == 0 {
		return nil, &Error{Kind: ErrorNotFound, Message: "no branches found for repository"}
	}
	return branches, nil
}

// RepoKey returns a stable cache key for a repository identity.
func RepoKey(provider Provider, apiBaseURL, repoPath string) string {
	return fmt.Sprintf("%s|%s|%s",
		provider,
		strings.TrimSuffix(strings.TrimSpace(apiBaseURL), "/"),
		strings.Trim(strings.TrimSpace(repoPath), "/"),
	)
}

// ClientForRepo builds a branch-listing client from repo URL and optional token.
func ClientForRepo(repoURL, token string) (*Client, error) {
	return New(Config{
		RepoURL: repoURL,
		Branch:  "main",
		Token:   token,
	})
}

// FilterBranches applies case-insensitive substring search and limit.
func FilterBranches(branches []Branch, search string, limit int) []Branch {
	if limit <= 0 {
		limit = 100
	}
	if limit > 200 {
		limit = 200
	}
	search = strings.TrimSpace(strings.ToLower(search))
	filtered := make([]Branch, 0, len(branches))
	for _, branch := range branches {
		if search != "" && !strings.Contains(strings.ToLower(branch.Name), search) {
			continue
		}
		filtered = append(filtered, branch)
		if len(filtered) >= limit {
			break
		}
	}
	return filtered
}

// ParseRepoFromURL is a convenience wrapper around ParseURL for handlers.
func ParseRepoFromURL(raw string) (ParseResult, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ParseResult{}, invalidError("repository URL is required")
	}
	return ParseURL(raw)
}
