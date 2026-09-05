package docsync

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/imaanmzr/postchi/backend/internal/docsync/linkmatcher"
	"github.com/imaanmzr/postchi/backend/internal/shared/gitrepo"
)

type GitProvider = gitrepo.Provider

const (
	GitProviderGitHub = gitrepo.ProviderGitHub
	GitProviderGitLab = gitrepo.ProviderGitLab
)

type GitRepo struct {
	Provider   GitProvider
	RepoPath   string
	Branch     string
	PathPrefix string
	APIBaseURL string
	Token      string
	HTTPClient *http.Client
}

type repoURLParseResult struct {
	Provider      GitProvider
	RepoURL       string
	APIBase       string
	Branch        string
	PathPrefix    string
	FromBrowseURL bool
}

func parseRepoURLInput(raw string) (repoURLParseResult, error) {
	parsed, err := gitrepo.ParseURL(raw)
	if err != nil {
		return repoURLParseResult{}, err
	}
	return repoURLParseResult{
		Provider:      parsed.Provider,
		RepoURL:       parsed.RepoURL,
		APIBase:       parsed.APIBase,
		Branch:        parsed.Branch,
		PathPrefix:    parsed.PathPrefix,
		FromBrowseURL: parsed.FromBrowseURL,
	}, nil
}

func normalizeRepoConfig(config map[string]any) (map[string]any, error) {
	config = sanitizeSourceConfig(config)
	repoURL, _ := config["repo_url"].(string)
	parsed, err := gitrepo.ParseURL(repoURL)
	if err != nil {
		return nil, err
	}
	config["repo_url"] = parsed.RepoURL
	config["provider"] = string(parsed.Provider)
	config["api_base_url"] = parsed.APIBase
	if parsed.FromBrowseURL {
		if parsed.Branch != "" {
			config["branch"] = parsed.Branch
		}
		config["path_prefix"] = gitrepo.NormalizePathPrefix(parsed.Branch, parsed.PathPrefix)
	}
	if branch, _ := config["branch"].(string); strings.TrimSpace(branch) != "" {
		if prefix, ok := config["path_prefix"].(string); ok {
			config["path_prefix"] = gitrepo.NormalizePathPrefix(branch, prefix)
		}
	}
	if tmpl, ok := config["link_template"].(string); ok {
		tmpl = strings.TrimSpace(tmpl)
		if tmpl != "" && !linkmatcher.ValidateLinkTemplate(tmpl) {
			return nil, fmt.Errorf("link_template must include {request_slug} or {request_name}")
		}
		config["link_template"] = tmpl
	}
	if branch, _ := config["branch"].(string); strings.TrimSpace(branch) != "" {
		if _, ok := gitrepo.SanitizeBranchName(branch); !ok {
			return nil, fmt.Errorf("invalid branch name")
		}
	}
	return config, nil
}

func gitClientFromConfig(config map[string]any, token string) (*GitRepo, error) {
	repoURL, _ := config["repo_url"].(string)
	branch, _ := config["branch"].(string)
	prefix, _ := config["path_prefix"].(string)
	client, err := gitrepo.New(gitrepo.Config{
		RepoURL:      repoURL,
		Branch:       branch,
		PathPrefix:   prefix,
		Token:        token,
		MaxTreeFiles: maxSyncFiles,
	})
	if err != nil {
		return nil, err
	}
	return &GitRepo{
		Provider:   client.Provider,
		RepoPath:   client.RepoPath,
		Branch:     client.Branch,
		PathPrefix: client.PathPrefix,
		APIBaseURL: client.APIBaseURL,
		Token:      client.Token,
		HTTPClient: client.HTTPClient,
	}, nil
}

func (g *GitRepo) client() (*gitrepo.Client, error) {
	repoURL := "https://github.com/" + g.RepoPath
	if g.Provider == GitProviderGitLab {
		repoURL = strings.TrimSuffix(g.APIBaseURL, "/") + "/" + g.RepoPath
	}
	return gitrepo.New(gitrepo.Config{
		RepoURL:      repoURL,
		Branch:       g.Branch,
		PathPrefix:   g.PathPrefix,
		Token:        g.Token,
		HTTPClient:   g.HTTPClient,
		APIBaseURL:   g.APIBaseURL,
		MaxTreeFiles: maxSyncFiles,
	})
}

func (g *GitRepo) ListMarkdownFiles(ctx context.Context) ([]string, error) {
	client, err := g.client()
	if err != nil {
		return nil, err
	}
	files, err := client.ListFiles(ctx)
	if err != nil {
		return nil, err
	}
	filtered := make([]string, 0, len(files))
	for _, file := range files {
		if isMarkdownPath(file) {
			filtered = append(filtered, file)
		}
	}
	if len(filtered) == 0 {
		return nil, fmt.Errorf("no markdown files found on branch %q under path %q", g.Branch, displayPath(g.PathPrefix))
	}
	return filtered, nil
}

func (g *GitRepo) FetchFile(ctx context.Context, filePath string) (string, error) {
	client, err := g.client()
	if err != nil {
		return "", err
	}
	content, err := client.FetchFile(ctx, filePath)
	return string(content), err
}

func (g *GitRepo) normalizeTreeItemPath(value string) string {
	value = strings.TrimPrefix(strings.TrimSpace(value), "/")
	prefix := strings.Trim(g.PathPrefix, "/")
	if prefix == "" || value == prefix || strings.HasPrefix(value, prefix+"/") {
		return value
	}
	return path.Join(prefix, value)
}

func encodeGitLabPath(value string) string {
	parts := strings.Split(value, "/")
	for i, part := range parts {
		parts[i] = strings.ReplaceAll(url.PathEscape(part), ".", "%2E")
	}
	return strings.Join(parts, "%2F")
}

func encodeGitLabProjectPath(value string) string {
	return encodeGitLabPath(value)
}

func gitLabHTTPError(status int, _ string) error {
	message := fmt.Sprintf("gitlab API HTTP %d", status)
	if status == http.StatusUnauthorized || status == http.StatusForbidden {
		message += "; use a personal access token with read_api and read_repository scopes, and ensure your account has at least Reporter access to the project"
	}
	return fmt.Errorf("%s", message)
}

func isMarkdownPath(value string) bool {
	lower := strings.ToLower(value)
	return strings.HasSuffix(lower, ".md") || strings.HasSuffix(lower, ".markdown")
}

func displayPath(prefix string) string {
	if prefix == "" {
		return "(repository root)"
	}
	return prefix
}

func sanitizeSourceConfig(config map[string]any) map[string]any {
	out := make(map[string]any, len(config))
	for key, value := range config {
		if key != "access_token" && key != "token" {
			out[key] = value
		}
	}
	return out
}
