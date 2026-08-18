package docsync

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

type GitProvider string

const (
	GitProviderGitHub GitProvider = "github"
	GitProviderGitLab GitProvider = "gitlab"
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

func gitClientFromConfig(config map[string]any, token string) (*GitRepo, error) {
	repoURL, _ := config["repo_url"].(string)
	repoURL = strings.TrimSpace(repoURL)
	if repoURL == "" {
		return nil, fmt.Errorf("repo_url required")
	}
	parsed, err := parseRepoURLInput(repoURL)
	if err != nil {
		return nil, err
	}

	branch, _ := config["branch"].(string)
	if strings.TrimSpace(branch) == "" {
		branch = parsed.Branch
	}
	if strings.TrimSpace(branch) == "" {
		branch = "main"
	}

	pathPrefix, _ := config["path_prefix"].(string)
	pathPrefix = strings.Trim(strings.TrimSpace(pathPrefix), "/")
	if pathPrefix == "" {
		pathPrefix = strings.Trim(parsed.PathPrefix, "/")
	}

	apiBase := parsed.APIBase
	if parsed.Provider == GitProviderGitLab {
		apiBase = strings.TrimSuffix(apiBase, "/api/v4")
	}

	repoPath, err := parseRepoPath(parsed.RepoURL, parsed.Provider)
	if err != nil {
		return nil, err
	}

	return &GitRepo{
		Provider:   parsed.Provider,
		RepoPath:   repoPath,
		Branch:     branch,
		PathPrefix: pathPrefix,
		APIBaseURL: apiBase,
		Token:      strings.TrimSpace(token),
		HTTPClient: &http.Client{Timeout: 20 * time.Second},
	}, nil
}

func detectProvider(repoURL, apiBase string) GitProvider {
	parsed, err := parseRepoURLInput(repoURL)
	if err == nil {
		return parsed.Provider
	}
	lower := strings.ToLower(repoURL + " " + apiBase)
	if strings.Contains(lower, "gitlab") {
		return GitProviderGitLab
	}
	return GitProviderGitHub
}

func defaultAPIBase(provider GitProvider) string {
	if provider == GitProviderGitLab {
		return "https://gitlab.com"
	}
	return "https://api.github.com"
}

// normalizeRepoConfig infers provider and API host from a single repository URL.
func normalizeRepoConfig(config map[string]any) (map[string]any, error) {
	config = sanitizeSourceConfig(config)
	repoURL, _ := config["repo_url"].(string)
	repoURL = strings.TrimSpace(repoURL)
	if repoURL == "" {
		return config, fmt.Errorf("repo_url required")
	}
	parsed, err := parseRepoURLInput(repoURL)
	if err != nil {
		return nil, err
	}
	config["repo_url"] = parsed.RepoURL
	config["provider"] = string(parsed.Provider)
	if parsed.APIBase != "" {
		config["api_base_url"] = parsed.APIBase
	}
	if parsed.FromBrowseURL {
		if parsed.Branch != "" {
			config["branch"] = parsed.Branch
		}
		config["path_prefix"] = parsed.PathPrefix
	}
	return config, nil
}

type repoURLParseResult struct {
	Provider       GitProvider
	RepoURL        string
	APIBase        string
	Branch         string
	PathPrefix     string
	FromBrowseURL  bool
}

func parseRepoURLInput(raw string) (repoURLParseResult, error) {
	raw = strings.TrimSpace(raw)
	if !strings.Contains(raw, "://") {
		return repoURLParseResult{}, fmt.Errorf("repository URL must include https://")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return repoURLParseResult{}, fmt.Errorf("invalid repository URL")
	}
	if u.Host == "" {
		return repoURLParseResult{}, fmt.Errorf("invalid repository URL")
	}

	result := repoURLParseResult{}
	repoPath := strings.Trim(strings.TrimSuffix(u.Path, ".git"), "/")

	if browseRepo, branch, prefix, ok := splitGitLabBrowsePath(u.Path); ok {
		repoPath = browseRepo
		result.Branch = branch
		result.PathPrefix = strings.Trim(prefix, "/")
		result.FromBrowseURL = true
	}

	if repoPath == "" || !strings.Contains(repoPath, "/") {
		return repoURLParseResult{}, fmt.Errorf("repository URL must include owner/group and project name")
	}

	result.RepoURL = fmt.Sprintf("%s://%s/%s", u.Scheme, u.Host, repoPath)
	host := strings.ToLower(u.Hostname())

	if isGitLabHost(host, u.Path) {
		result.Provider = GitProviderGitLab
		result.APIBase = fmt.Sprintf("%s://%s", u.Scheme, u.Host)
		return result, nil
	}

	if host == "github.com" {
		result.Provider = GitProviderGitHub
		result.APIBase = "https://api.github.com"
		return result, nil
	}

	if strings.Contains(host, "github") {
		result.Provider = GitProviderGitHub
		result.APIBase = fmt.Sprintf("%s://%s/api/v3", u.Scheme, u.Host)
		return result, nil
	}

	result.Provider = GitProviderGitLab
	result.APIBase = fmt.Sprintf("%s://%s", u.Scheme, u.Host)
	return result, nil
}

func isGitLabHost(host, path string) bool {
	host = strings.ToLower(host)
	if strings.Contains(host, "gitlab") {
		return true
	}
	if strings.Contains(path, "/-/tree/") || strings.Contains(path, "/-/blob/") {
		return true
	}
	// Self-hosted GitLab often uses git.example.com (not github.com).
	if strings.HasPrefix(host, "git.") && !strings.Contains(host, "github") {
		return true
	}
	return false
}

func splitGitLabBrowsePath(path string) (repoPath, branch, browsePath string, ok bool) {
	path = strings.Trim(path, "/")
	idx := strings.Index(path, "/-/")
	if idx < 0 {
		return "", "", "", false
	}
	repoPath = strings.Trim(path[:idx], "/")
	rest := strings.Trim(path[idx+3:], "/")
	switch {
	case strings.HasPrefix(rest, "tree/"):
		rest = strings.TrimPrefix(rest, "tree/")
	case strings.HasPrefix(rest, "blob/"):
		rest = strings.TrimPrefix(rest, "blob/")
	default:
		return "", "", "", false
	}
	parts := strings.SplitN(rest, "/", 2)
	if len(parts) == 0 || parts[0] == "" {
		return "", "", "", false
	}
	branch = parts[0]
	if len(parts) == 2 {
		browsePath = strings.Trim(parts[1], "/")
		if strings.Contains(path, "/-/blob/") && strings.HasSuffix(browsePath, ".md") {
			if i := strings.LastIndex(browsePath, "/"); i >= 0 {
				browsePath = browsePath[:i]
			} else {
				browsePath = ""
			}
		}
	}
	return repoPath, branch, browsePath, true
}

func parseRepoURLInputLegacy(raw string) (GitProvider, string, string, error) {
	parsed, err := parseRepoURLInput(raw)
	if err != nil {
		return "", "", "", err
	}
	return parsed.Provider, parsed.RepoURL, parsed.APIBase, nil
}

func parseRepoPath(repoURL string, provider GitProvider) (string, error) {
	repoURL = strings.TrimSuffix(strings.TrimSpace(repoURL), ".git")
	if strings.Contains(repoURL, "://") {
		u, err := url.Parse(repoURL)
		if err != nil {
			return "", fmt.Errorf("invalid repo_url")
		}
		p := strings.Trim(u.Path, "/")
		if p == "" {
			return "", fmt.Errorf("invalid repo_url path")
		}
		return p, nil
	}
	if strings.Contains(repoURL, "/") {
		return strings.Trim(repoURL, "/"), nil
	}
	return "", fmt.Errorf("invalid repo_url")
}

func (g *GitRepo) ListMarkdownFiles(ctx context.Context) ([]string, error) {
	switch g.Provider {
	case GitProviderGitLab:
		return g.listGitLabMarkdownFiles(ctx)
	default:
		return g.listGitHubMarkdownFiles(ctx)
	}
}

func (g *GitRepo) FetchFile(ctx context.Context, filePath string) (string, error) {
	switch g.Provider {
	case GitProviderGitLab:
		return g.fetchGitLabFile(ctx, filePath)
	default:
		return g.fetchGitHubFile(ctx, filePath)
	}
}

func encodeGitLabPath(p string) string {
	parts := strings.Split(p, "/")
	for i, part := range parts {
		if part == "" {
			continue
		}
		escaped := url.PathEscape(part)
		escaped = strings.ReplaceAll(escaped, ".", "%2E")
		parts[i] = escaped
	}
	return strings.Join(parts, "%2F")
}

func encodeGitLabProjectPath(repoPath string) string {
	return encodeGitLabPath(repoPath)
}

func gitLabHTTPError(status int, body string) error {
	msg := fmt.Sprintf("gitlab API HTTP %d: %s", status, strings.TrimSpace(body))
	if status == 401 || status == 403 {
		msg += "; use a personal access token with read_api and read_repository scopes, and ensure your account has at least Reporter access to the project"
	}
	return fmt.Errorf("%s", msg)
}

func (g *GitRepo) doRequest(ctx context.Context, method, rawURL string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, rawURL, nil)
	if err != nil {
		return nil, err
	}
	switch g.Provider {
	case GitProviderGitLab:
		if g.Token != "" {
			req.Header.Set("PRIVATE-TOKEN", g.Token)
			req.Header.Set("Authorization", "Bearer "+g.Token)
		}
	default:
		req.Header.Set("Accept", "application/vnd.github+json")
		if g.Token != "" {
			req.Header.Set("Authorization", "Bearer "+g.Token)
		}
	}
	return g.HTTPClient.Do(req)
}

func (g *GitRepo) listGitHubMarkdownFiles(ctx context.Context) ([]string, error) {
	treeURL := fmt.Sprintf("%s/repos/%s/git/trees/%s?recursive=1", g.APIBaseURL, g.RepoPath, url.PathEscape(g.Branch))
	resp, err := g.doRequest(ctx, http.MethodGet, treeURL)
	if err != nil {
		return g.fallbackSingleFile()
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		if g.Token == "" {
			return g.fallbackSingleFile()
		}
		return nil, fmt.Errorf("github tree API HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var result struct {
		Tree []struct {
			Path string `json:"path"`
			Type string `json:"type"`
		} `json:"tree"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return g.fallbackSingleFile()
	}
	var paths []string
	for _, item := range result.Tree {
		if item.Type == "blob" && strings.HasSuffix(item.Path, ".md") {
			paths = append(paths, item.Path)
		}
	}
	return g.filterMarkdownPaths(paths), nil
}

const gitLabMaxTreePages = 50

func (g *GitRepo) listGitLabMarkdownFiles(ctx context.Context) ([]string, error) {
	if g.Token == "" {
		return nil, fmt.Errorf("access token required for GitLab repositories")
	}
	project := encodeGitLabProjectPath(g.RepoPath)
	apiRoot := strings.TrimSuffix(g.APIBaseURL, "/") + "/api/v4"
	nextURL := fmt.Sprintf("%s/projects/%s/repository/tree?recursive=true&ref=%s&per_page=100",
		apiRoot, project, url.QueryEscape(g.Branch))
	if g.PathPrefix != "" {
		nextURL += "&path=" + url.QueryEscape(g.PathPrefix)
	}

	var paths []string
	seenPages := map[string]bool{}
	for nextURL != "" && len(seenPages) < gitLabMaxTreePages {
		if seenPages[nextURL] {
			break
		}
		seenPages[nextURL] = true

		resp, err := g.doRequest(ctx, http.MethodGet, nextURL)
		if err != nil {
			return nil, fmt.Errorf("gitlab tree request failed: %w", err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode >= 400 {
			if g.Token == "" {
				return g.fallbackSingleFile()
			}
			return nil, gitLabHTTPError(resp.StatusCode, string(body))
		}
		var items []struct {
			Path string `json:"path"`
			Type string `json:"type"`
		}
		if err := json.Unmarshal(body, &items); err != nil {
			return nil, fmt.Errorf("gitlab tree response parse failed: %w", err)
		}
		if len(items) == 0 {
			break
		}
		for _, item := range items {
			if item.Type == "blob" && isMarkdownPath(item.Path) {
				paths = append(paths, g.normalizeTreeItemPath(item.Path))
			}
		}
		nextURL = parseGitLabNextLink(resp.Header.Get("Link"))
	}
	filtered := g.filterMarkdownPaths(paths)
	if len(filtered) == 0 {
		if g.Token != "" {
			if len(paths) > 0 && g.PathPrefix != "" {
				return nil, fmt.Errorf("no markdown files under path prefix %q (found %d .md file(s) elsewhere on branch %q)", g.PathPrefix, len(paths), g.Branch)
			}
			return nil, fmt.Errorf("no markdown files found on branch %q under path %q", g.Branch, displayPath(g.PathPrefix))
		}
		return g.fallbackSingleFile()
	}
	return filtered, nil
}

func parseGitLabNextLink(linkHeader string) string {
	if linkHeader == "" {
		return ""
	}
	for _, part := range strings.Split(linkHeader, ",") {
		part = strings.TrimSpace(part)
		if strings.Contains(part, `rel="next"`) {
			start := strings.Index(part, "<")
			end := strings.Index(part, ">")
			if start >= 0 && end > start {
				return part[start+1 : end]
			}
		}
	}
	return ""
}

func displayPath(prefix string) string {
	if prefix == "" {
		return "(repository root)"
	}
	return prefix
}

func (g *GitRepo) normalizeTreeItemPath(p string) string {
	p = strings.TrimPrefix(strings.TrimSpace(p), "/")
	if p == "" {
		return p
	}
	prefix := strings.Trim(g.PathPrefix, "/")
	if prefix == "" {
		return p
	}
	if p == prefix || strings.HasPrefix(p, prefix+"/") {
		return p
	}
	return path.Join(prefix, p)
}

func (g *GitRepo) filterMarkdownPaths(paths []string) []string {
	if g.PathPrefix == "" {
		return paths
	}
	prefix := g.PathPrefix
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	var filtered []string
	for _, p := range paths {
		if strings.HasPrefix(p, prefix) || p == strings.TrimSuffix(prefix, "/") {
			filtered = append(filtered, p)
		}
	}
	return filtered
}

func (g *GitRepo) fallbackSingleFile() ([]string, error) {
	p := g.PathPrefix
	if p == "" {
		p = "README.md"
	} else if !strings.HasSuffix(p, ".md") {
		p = path.Join(p, "README.md")
	}
	return []string{p}, nil
}

func (g *GitRepo) fetchGitHubFile(ctx context.Context, filePath string) (string, error) {
	if g.Token == "" {
		return g.fetchGitHubRawPublic(filePath)
	}
	apiURL := fmt.Sprintf("%s/repos/%s/contents/%s?ref=%s",
		g.APIBaseURL, g.RepoPath, escapeGitHubPath(filePath), url.QueryEscape(g.Branch))
	resp, err := g.doRequest(ctx, http.MethodGet, apiURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("github contents API HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var result struct {
		Content  string `json:"content"`
		Encoding string `json:"encoding"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if result.Encoding != "base64" {
		return "", fmt.Errorf("unsupported github encoding %q", result.Encoding)
	}
	raw, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(result.Content, "\n", ""))
	return string(raw), err
}

func (g *GitRepo) fetchGitHubRawPublic(filePath string) (string, error) {
	rawURL := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s", g.RepoPath, g.Branch, filePath)
	resp, err := g.HTTPClient.Get(rawURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	b, err := io.ReadAll(resp.Body)
	return string(b), err
}

func escapeGitHubPath(p string) string {
	parts := strings.Split(p, "/")
	for i, part := range parts {
		parts[i] = url.PathEscape(part)
	}
	return strings.Join(parts, "/")
}

func isMarkdownPath(p string) bool {
	lower := strings.ToLower(p)
	return strings.HasSuffix(lower, ".md") || strings.HasSuffix(lower, ".markdown")
}

func (g *GitRepo) fetchGitLabFile(ctx context.Context, filePath string) (string, error) {
	content, err := g.fetchGitLabFileJSON(ctx, filePath)
	if err == nil {
		return content, nil
	}
	if strings.Contains(err.Error(), "HTTP 404") {
		return g.fetchGitLabFileRaw(ctx, filePath)
	}
	return "", err
}

func (g *GitRepo) fetchGitLabFileRaw(ctx context.Context, filePath string) (string, error) {
	project := encodeGitLabProjectPath(g.RepoPath)
	file := encodeGitLabPath(filePath)
	apiRoot := strings.TrimSuffix(g.APIBaseURL, "/") + "/api/v4"
	rawURL := fmt.Sprintf("%s/projects/%s/repository/files/%s/raw?ref=%s",
		apiRoot, project, file, url.QueryEscape(g.Branch))
	resp, err := g.doRequest(ctx, http.MethodGet, rawURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return "", gitLabHTTPError(resp.StatusCode, string(body))
	}
	b, err := io.ReadAll(resp.Body)
	return string(b), err
}

func (g *GitRepo) fetchGitLabFileJSON(ctx context.Context, filePath string) (string, error) {
	project := encodeGitLabProjectPath(g.RepoPath)
	file := encodeGitLabPath(filePath)
	apiRoot := strings.TrimSuffix(g.APIBaseURL, "/") + "/api/v4"
	apiURL := fmt.Sprintf("%s/projects/%s/repository/files/%s?ref=%s",
		apiRoot, project, file, url.QueryEscape(g.Branch))
	resp, err := g.doRequest(ctx, http.MethodGet, apiURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return "", gitLabHTTPError(resp.StatusCode, string(body))
	}
	var result struct {
		Content  string `json:"content"`
		Encoding string `json:"encoding"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}
	if result.Encoding != "base64" {
		return "", fmt.Errorf("unsupported gitlab file encoding %q", result.Encoding)
	}
	raw, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(result.Content, "\n", ""))
	return string(raw), err
}

func sanitizeSourceConfig(config map[string]any) map[string]any {
	if config == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(config))
	for k, v := range config {
		if k == "access_token" || k == "token" {
			continue
		}
		out[k] = v
	}
	return out
}
