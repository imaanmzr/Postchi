package gitrepo

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

type Provider string

const (
	ProviderGitHub Provider = "github"
	ProviderGitLab Provider = "gitlab"
)

type ErrorKind string

const (
	ErrorInvalid        ErrorKind = "invalid"
	ErrorAuthentication ErrorKind = "authentication"
	ErrorNotFound       ErrorKind = "not_found"
	ErrorRateLimited    ErrorKind = "rate_limited"
	ErrorTimeout        ErrorKind = "timeout"
	ErrorLimit          ErrorKind = "limit"
	ErrorProvider       ErrorKind = "provider"
)

type Error struct {
	Kind    ErrorKind
	Message string
	Status  int
}

func (e *Error) Error() string { return e.Message }

func Kind(err error) ErrorKind {
	var repoErr *Error
	if errors.As(err, &repoErr) {
		return repoErr.Kind
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return ErrorTimeout
	}
	return ErrorProvider
}

type ParseResult struct {
	Provider      Provider
	RepoURL       string
	APIBase       string
	RepoPath      string
	Branch        string
	PathPrefix    string
	FromBrowseURL bool
}

type Config struct {
	RepoURL      string
	Branch       string
	PathPrefix   string
	Token        string
	HTTPClient   *http.Client
	APIBaseURL   string
	MaxTreeFiles int
	MaxFileBytes int64
}

type Client struct {
	Provider     Provider
	RepoPath     string
	Branch       string
	PathPrefix   string
	APIBaseURL   string
	Token        string
	HTTPClient   *http.Client
	MaxTreeFiles int
	MaxFileBytes int64
}

func New(config Config) (*Client, error) {
	parsed, err := ParseURL(config.RepoURL)
	if err != nil {
		return nil, err
	}
	branch := strings.TrimSpace(config.Branch)
	if branch == "" {
		branch = parsed.Branch
	}
	if branch == "" {
		branch = "main"
	}
	prefix := strings.Trim(strings.TrimSpace(config.PathPrefix), "/")
	if prefix == "" {
		prefix = parsed.PathPrefix
	}
	apiBase := parsed.APIBase
	if strings.TrimSpace(config.APIBaseURL) != "" {
		apiBase = strings.TrimSuffix(strings.TrimSpace(config.APIBaseURL), "/")
	}
	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 20 * time.Second}
	}
	maxTreeFiles := config.MaxTreeFiles
	if maxTreeFiles <= 0 {
		maxTreeFiles = 10_000
	}
	maxFileBytes := config.MaxFileBytes
	if maxFileBytes <= 0 {
		maxFileBytes = 2 << 20
	}
	return &Client{
		Provider:     parsed.Provider,
		RepoPath:     parsed.RepoPath,
		Branch:       branch,
		PathPrefix:   prefix,
		APIBaseURL:   apiBase,
		Token:        strings.TrimSpace(config.Token),
		HTTPClient:   httpClient,
		MaxTreeFiles: maxTreeFiles,
		MaxFileBytes: maxFileBytes,
	}, nil
}

func ParseURL(raw string) (ParseResult, error) {
	raw = strings.TrimSpace(raw)
	if !strings.Contains(raw, "://") {
		return ParseResult{}, invalidError("repository URL must include https://")
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") {
		return ParseResult{}, invalidError("invalid repository URL")
	}

	result := ParseResult{}
	repoPath := strings.Trim(strings.TrimSuffix(u.Path, ".git"), "/")
	if browseRepo, branch, prefix, ok := splitGitLabBrowsePath(u.Path); ok {
		repoPath = browseRepo
		result.Branch = branch
		result.PathPrefix = strings.Trim(prefix, "/")
		result.FromBrowseURL = true
	}
	if repoPath == "" || !strings.Contains(repoPath, "/") {
		return ParseResult{}, invalidError("repository URL must include owner/group and project name")
	}

	result.RepoPath = repoPath
	result.RepoURL = fmt.Sprintf("%s://%s/%s", u.Scheme, u.Host, repoPath)
	host := strings.ToLower(u.Hostname())
	switch {
	case isGitLabHost(host, u.Path):
		result.Provider = ProviderGitLab
		result.APIBase = fmt.Sprintf("%s://%s", u.Scheme, u.Host)
	case host == "github.com":
		result.Provider = ProviderGitHub
		result.APIBase = "https://api.github.com"
	case strings.Contains(host, "github"):
		result.Provider = ProviderGitHub
		result.APIBase = fmt.Sprintf("%s://%s/api/v3", u.Scheme, u.Host)
	default:
		result.Provider = ProviderGitLab
		result.APIBase = fmt.Sprintf("%s://%s", u.Scheme, u.Host)
	}
	return result, nil
}

func (c *Client) ListFiles(ctx context.Context) ([]string, error) {
	if c.Provider == ProviderGitLab {
		return c.listGitLabFiles(ctx)
	}
	return c.listGitHubFiles(ctx)
}

func (c *Client) FetchFile(ctx context.Context, filePath string) ([]byte, error) {
	if c.Provider == ProviderGitLab {
		return c.fetchGitLabFile(ctx, filePath)
	}
	return c.fetchGitHubFile(ctx, filePath)
}

func (c *Client) listGitHubFiles(ctx context.Context) ([]string, error) {
	treeURL := fmt.Sprintf("%s/repos/%s/git/trees/%s?recursive=1", strings.TrimSuffix(c.APIBaseURL, "/"), c.RepoPath, url.PathEscape(c.Branch))
	resp, err := c.doRequest(ctx, http.MethodGet, treeURL)
	if err != nil {
		return nil, requestError("GitHub tree request failed", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, responseError("GitHub", resp)
	}
	var result struct {
		Tree []struct {
			Path string `json:"path"`
			Type string `json:"type"`
		} `json:"tree"`
		Truncated bool `json:"truncated"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, &Error{Kind: ErrorProvider, Message: "invalid GitHub tree response"}
	}
	if result.Truncated {
		return nil, &Error{Kind: ErrorLimit, Message: "repository tree is too large; use a narrower path prefix"}
	}
	paths := make([]string, 0, len(result.Tree))
	for _, item := range result.Tree {
		if item.Type == "blob" && c.inPathPrefix(item.Path) {
			paths = append(paths, item.Path)
			if len(paths) > c.MaxTreeFiles {
				return nil, &Error{Kind: ErrorLimit, Message: "repository contains too many files; use a narrower path prefix"}
			}
		}
	}
	if len(paths) == 0 {
		return nil, &Error{Kind: ErrorNotFound, Message: fmt.Sprintf("no files found on branch %q under path %q", c.Branch, displayPath(c.PathPrefix))}
	}
	return paths, nil
}

const gitLabMaxTreePages = 50

func (c *Client) listGitLabFiles(ctx context.Context) ([]string, error) {
	if c.Token == "" {
		return nil, &Error{Kind: ErrorAuthentication, Message: "access token required for GitLab repositories"}
	}
	apiRoot := strings.TrimSuffix(c.APIBaseURL, "/") + "/api/v4"
	nextURL := fmt.Sprintf("%s/projects/%s/repository/tree?recursive=true&ref=%s&per_page=100",
		apiRoot, encodeGitLabPath(c.RepoPath), url.QueryEscape(c.Branch))
	if c.PathPrefix != "" {
		nextURL += "&path=" + url.QueryEscape(c.PathPrefix)
	}

	var paths []string
	seen := map[string]bool{}
	for nextURL != "" {
		if len(seen) >= gitLabMaxTreePages {
			return nil, &Error{Kind: ErrorLimit, Message: "GitLab repository tree exceeded the page limit"}
		}
		if seen[nextURL] {
			return nil, &Error{Kind: ErrorProvider, Message: "GitLab returned a pagination loop"}
		}
		seen[nextURL] = true
		resp, err := c.doRequest(ctx, http.MethodGet, nextURL)
		if err != nil {
			return nil, requestError("GitLab tree request failed", err)
		}
		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			return nil, requestError("GitLab tree response failed", readErr)
		}
		if resp.StatusCode >= 400 {
			return nil, responseErrorBody("GitLab", resp.StatusCode, body)
		}
		var items []struct {
			Path string `json:"path"`
			Type string `json:"type"`
		}
		if err := json.Unmarshal(body, &items); err != nil {
			return nil, &Error{Kind: ErrorProvider, Message: "invalid GitLab tree response"}
		}
		for _, item := range items {
			if item.Type != "blob" {
				continue
			}
			itemPath := c.normalizeTreeItemPath(item.Path)
			if c.inPathPrefix(itemPath) {
				paths = append(paths, itemPath)
				if len(paths) > c.MaxTreeFiles {
					return nil, &Error{Kind: ErrorLimit, Message: "repository contains too many files; use a narrower path prefix"}
				}
			}
		}
		nextURL = parseNextLink(resp.Header.Get("Link"))
	}
	if len(paths) == 0 {
		return nil, &Error{Kind: ErrorNotFound, Message: fmt.Sprintf("no files found on branch %q under path %q", c.Branch, displayPath(c.PathPrefix))}
	}
	return paths, nil
}

func (c *Client) fetchGitHubFile(ctx context.Context, filePath string) ([]byte, error) {
	apiURL := fmt.Sprintf("%s/repos/%s/contents/%s?ref=%s",
		strings.TrimSuffix(c.APIBaseURL, "/"), c.RepoPath, escapePath(filePath), url.QueryEscape(c.Branch))
	resp, err := c.doRequest(ctx, http.MethodGet, apiURL)
	if err != nil {
		return nil, requestError("GitHub file request failed", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, responseError("GitHub", resp)
	}
	var result struct {
		Content  string `json:"content"`
		Encoding string `json:"encoding"`
		Size     int64  `json:"size"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, c.MaxFileBytes*2+1)).Decode(&result); err != nil {
		return nil, &Error{Kind: ErrorProvider, Message: fmt.Sprintf("invalid GitHub response for %q", filePath)}
	}
	if result.Size > c.MaxFileBytes {
		return nil, &Error{Kind: ErrorLimit, Message: fmt.Sprintf("file %q exceeds the size limit", filePath)}
	}
	if result.Encoding != "base64" {
		return nil, &Error{Kind: ErrorProvider, Message: fmt.Sprintf("unsupported GitHub encoding %q", result.Encoding)}
	}
	raw, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(result.Content, "\n", ""))
	if err != nil {
		return nil, &Error{Kind: ErrorProvider, Message: fmt.Sprintf("invalid GitHub content for %q", filePath)}
	}
	if int64(len(raw)) > c.MaxFileBytes {
		return nil, &Error{Kind: ErrorLimit, Message: fmt.Sprintf("file %q exceeds the size limit", filePath)}
	}
	return raw, nil
}

func (c *Client) fetchGitLabFile(ctx context.Context, filePath string) ([]byte, error) {
	apiRoot := strings.TrimSuffix(c.APIBaseURL, "/") + "/api/v4"
	rawURL := fmt.Sprintf("%s/projects/%s/repository/files/%s/raw?ref=%s",
		apiRoot, encodeGitLabPath(c.RepoPath), encodeGitLabPath(filePath), url.QueryEscape(c.Branch))
	resp, err := c.doRequest(ctx, http.MethodGet, rawURL)
	if err != nil {
		return nil, requestError("GitLab file request failed", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, responseError("GitLab", resp)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, c.MaxFileBytes+1))
	if err != nil {
		return nil, requestError("GitLab file response failed", err)
	}
	if int64(len(body)) > c.MaxFileBytes {
		return nil, &Error{Kind: ErrorLimit, Message: fmt.Sprintf("file %q exceeds the size limit", filePath)}
	}
	return body, nil
}

func (c *Client) doRequest(ctx context.Context, method, rawURL string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, rawURL, nil)
	if err != nil {
		return nil, err
	}
	if c.Provider == ProviderGitLab {
		if c.Token != "" {
			req.Header.Set("PRIVATE-TOKEN", c.Token)
			req.Header.Set("Authorization", "Bearer "+c.Token)
		}
	} else {
		req.Header.Set("Accept", "application/vnd.github+json")
		if c.Token != "" {
			req.Header.Set("Authorization", "Bearer "+c.Token)
		}
	}
	return c.HTTPClient.Do(req)
}

func (c *Client) inPathPrefix(filePath string) bool {
	prefix := strings.Trim(c.PathPrefix, "/")
	return prefix == "" || filePath == prefix || strings.HasPrefix(filePath, prefix+"/")
}

func (c *Client) normalizeTreeItemPath(filePath string) string {
	filePath = strings.TrimPrefix(strings.TrimSpace(filePath), "/")
	prefix := strings.Trim(c.PathPrefix, "/")
	if prefix == "" || filePath == prefix || strings.HasPrefix(filePath, prefix+"/") {
		return filePath
	}
	return path.Join(prefix, filePath)
}

func splitGitLabBrowsePath(rawPath string) (repoPath, branch, browsePath string, ok bool) {
	rawPath = strings.Trim(rawPath, "/")
	idx := strings.Index(rawPath, "/-/")
	if idx < 0 {
		return "", "", "", false
	}
	repoPath = strings.Trim(rawPath[:idx], "/")
	rest := strings.Trim(rawPath[idx+3:], "/")
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
		if strings.Contains(rawPath, "/-/blob/") {
			if i := strings.LastIndex(browsePath, "/"); i >= 0 {
				browsePath = browsePath[:i]
			} else {
				browsePath = ""
			}
		}
	}
	return repoPath, branch, browsePath, true
}

func isGitLabHost(host, rawPath string) bool {
	host = strings.ToLower(host)
	return strings.Contains(host, "gitlab") ||
		strings.Contains(rawPath, "/-/tree/") ||
		strings.Contains(rawPath, "/-/blob/") ||
		(strings.HasPrefix(host, "git.") && !strings.Contains(host, "github"))
}

func encodeGitLabPath(value string) string {
	parts := strings.Split(value, "/")
	for i, part := range parts {
		escaped := url.PathEscape(part)
		parts[i] = strings.ReplaceAll(escaped, ".", "%2E")
	}
	return strings.Join(parts, "%2F")
}

func escapePath(value string) string {
	parts := strings.Split(value, "/")
	for i, part := range parts {
		parts[i] = url.PathEscape(part)
	}
	return strings.Join(parts, "/")
}

func parseNextLink(header string) string {
	for _, part := range strings.Split(header, ",") {
		if !strings.Contains(part, `rel="next"`) {
			continue
		}
		start, end := strings.Index(part, "<"), strings.Index(part, ">")
		if start >= 0 && end > start {
			return part[start+1 : end]
		}
	}
	return ""
}

func responseError(provider string, resp *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	return responseErrorBody(provider, resp.StatusCode, body)
}

func responseErrorBody(provider string, status int, body []byte) error {
	kind := ErrorProvider
	message := fmt.Sprintf("%s API HTTP %d", provider, status)
	switch status {
	case http.StatusUnauthorized, http.StatusForbidden:
		kind = ErrorAuthentication
		message = provider + " authentication failed; check the personal access token and repository permissions"
	case http.StatusNotFound:
		kind = ErrorNotFound
		message = provider + " repository, branch, path, or file was not found"
	case http.StatusTooManyRequests:
		kind = ErrorRateLimited
		message = provider + " API rate limit exceeded; try again later or provide an access token"
	}
	if status == http.StatusForbidden && strings.Contains(strings.ToLower(string(body)), "rate limit") {
		kind = ErrorRateLimited
		message = provider + " API rate limit exceeded; try again later or provide an access token"
	}
	return &Error{Kind: kind, Message: message, Status: status}
}

func requestError(prefix string, err error) error {
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return &Error{Kind: ErrorTimeout, Message: prefix + ": request timed out"}
	}
	var netErr interface{ Timeout() bool }
	if errors.As(err, &netErr) && netErr.Timeout() {
		return &Error{Kind: ErrorTimeout, Message: prefix + ": request timed out"}
	}
	return &Error{Kind: ErrorProvider, Message: prefix + ": " + err.Error()}
}

func invalidError(message string) error {
	return &Error{Kind: ErrorInvalid, Message: message}
}

func displayPath(prefix string) string {
	if prefix == "" {
		return "(repository root)"
	}
	return prefix
}
