package docsync

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNormalizeTreeItemPath(t *testing.T) {
	g := &GitRepo{PathPrefix: "docs"}
	if got := g.normalizeTreeItemPath("guide.md"); got != "docs/guide.md" {
		t.Fatalf("relative: %s", got)
	}
	if got := g.normalizeTreeItemPath("docs/guide.md"); got != "docs/guide.md" {
		t.Fatalf("absolute: %s", got)
	}
}

func TestEncodeGitLabProjectPath(t *testing.T) {
	got := encodeGitLabProjectPath("acme/apps/platform-api")
	if got != "acme%2Fapps%2Fplatform-api" {
		t.Fatalf("encoded: %s", got)
	}
	got = encodeGitLabPath("docs/api/auth.md")
	if got != "docs%2Fapi%2Fauth%2Emd" {
		t.Fatalf("file encoded: %s", got)
	}
}

func TestGitLabHTTPError(t *testing.T) {
	err := gitLabHTTPError(403, `{"message":"403 Forbidden"}`)
	if err == nil || !strings.Contains(err.Error(), "read_repository") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseRepoURLInput(t *testing.T) {
	parsed, err := parseRepoURLInput("https://github.com/acme/docs.git")
	if err != nil || parsed.Provider != GitProviderGitHub || parsed.APIBase != "https://api.github.com" {
		t.Fatalf("github.com: %+v %v", parsed, err)
	}

	parsed, err = parseRepoURLInput("https://gitlab.com/acme/docs")
	if err != nil || parsed.Provider != GitProviderGitLab || parsed.APIBase != "https://gitlab.com" {
		t.Fatalf("gitlab.com: %+v %v", parsed, err)
	}

	parsed, err = parseRepoURLInput("https://gitlab.internal.example/acme/docs")
	if err != nil || parsed.Provider != GitProviderGitLab || parsed.APIBase != "https://gitlab.internal.example" {
		t.Fatalf("gitlab internal: %+v %v", parsed, err)
	}

	parsed, err = parseRepoURLInput("https://github.corp.example/acme/docs")
	if err != nil || parsed.Provider != GitProviderGitHub || parsed.APIBase != "https://github.corp.example/api/v3" {
		t.Fatalf("ghe: %+v %v", parsed, err)
	}

	parsed, err = parseRepoURLInput("https://git.example.internal/acme/apps/platform-api/-/tree/main/docs?ref_type=heads")
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Provider != GitProviderGitLab {
		t.Fatalf("provider: %s", parsed.Provider)
	}
	if parsed.RepoURL != "https://git.example.internal/acme/apps/platform-api" {
		t.Fatalf("repo url: %s", parsed.RepoURL)
	}
	if parsed.APIBase != "https://git.example.internal" {
		t.Fatalf("api base: %s", parsed.APIBase)
	}
	if parsed.Branch != "main" || parsed.PathPrefix != "docs" {
		t.Fatalf("branch/prefix: %s %s", parsed.Branch, parsed.PathPrefix)
	}

	parsed, err = parseRepoURLInput("https://git.example.internal/acme/apps/platform-api")
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Provider != GitProviderGitLab || parsed.APIBase != "https://git.example.internal" {
		t.Fatalf("clean git host: %+v", parsed)
	}
}

func TestNormalizeRepoConfigGitLabBrowseURL(t *testing.T) {
	out, err := normalizeRepoConfig(map[string]any{
		"repo_url": "https://git.example.internal/acme/apps/platform-api/-/tree/main/docs",
	})
	if err != nil {
		t.Fatal(err)
	}
	if out["provider"] != "gitlab" {
		t.Fatalf("provider: %v", out["provider"])
	}
	if out["branch"] != "main" || out["path_prefix"] != "docs" {
		t.Fatalf("branch/prefix: %v %v", out["branch"], out["path_prefix"])
	}
}

func TestGitClientFromConfigIgnoresStaleProvider(t *testing.T) {
	client, err := gitClientFromConfig(map[string]any{
		"provider":     "github",
		"repo_url":     "https://git.example.internal/acme/apps/platform-api",
		"api_base_url": "https://git.example.internal/api/v3",
		"branch":       "main",
		"path_prefix":  "docs",
	}, "token")
	if err != nil {
		t.Fatal(err)
	}
	if client.Provider != GitProviderGitLab {
		t.Fatalf("provider: %s", client.Provider)
	}
	if client.APIBaseURL != "https://git.example.internal" {
		t.Fatalf("api base: %s", client.APIBaseURL)
	}
}

func TestGitClientFromConfig(t *testing.T) {
	client, err := gitClientFromConfig(map[string]any{
		"repo_url":    "https://gitlab.internal.example/acme/docs",
		"branch":      "develop",
		"path_prefix": "docs/api",
	}, "secret")
	if err != nil {
		t.Fatal(err)
	}
	if client.Provider != GitProviderGitLab {
		t.Fatalf("provider: %s", client.Provider)
	}
	if client.RepoPath != "acme/docs" {
		t.Fatalf("repo path: %s", client.RepoPath)
	}
	if client.APIBaseURL != "https://gitlab.internal.example" {
		t.Fatalf("api base: %s", client.APIBaseURL)
	}
}

func TestGitHubPrivateListAndFetch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		switch r.URL.Path {
		case "/repos/acme/private/git/trees/main":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"tree":[{"path":"docs/api/auth.md","type":"blob"}]}`))
		case "/repos/acme/private/contents/docs/api/auth.md":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"content":"IyBBdXRo","encoding":"base64"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	client := &GitRepo{
		Provider:   GitProviderGitHub,
		RepoPath:   "acme/private",
		Branch:     "main",
		PathPrefix: "docs/api",
		APIBaseURL: srv.URL,
		Token:      "test-token",
		HTTPClient: srv.Client(),
	}
	files, err := client.ListMarkdownFiles(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0] != "docs/api/auth.md" {
		t.Fatalf("files: %v", files)
	}
	content, err := client.FetchFile(context.Background(), "docs/api/auth.md")
	if err != nil {
		t.Fatal(err)
	}
	if content != "# Auth" {
		t.Fatalf("content: %q", content)
	}
}

func TestGitLabPrivateListAndFetch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Private-Token") != "gl-token" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		switch {
		case strings.Contains(r.URL.Path, "/repository/tree"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[{"path":"docs/api/users.md","type":"blob"}]`))
		case strings.Contains(r.URL.Path, "/repository/files/") && strings.HasSuffix(r.URL.Path, "/raw"):
			_, _ = w.Write([]byte("# Users"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	client := &GitRepo{
		Provider:   GitProviderGitLab,
		RepoPath:   "acme/docs",
		Branch:     "main",
		PathPrefix: "",
		APIBaseURL: srv.URL,
		Token:      "gl-token",
		HTTPClient: srv.Client(),
	}
	files, err := client.ListMarkdownFiles(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Fatalf("files: %v", files)
	}
	content, err := client.FetchFile(context.Background(), "docs/api/users.md")
	if err != nil {
		t.Fatal(err)
	}
	if content != "# Users" {
		t.Fatalf("content: %q", content)
	}
}

func TestSanitizeSourceConfig(t *testing.T) {
	out := sanitizeSourceConfig(map[string]any{
		"repo_url":     "https://github.com/a/b",
		"access_token": "secret",
	})
	if _, ok := out["access_token"]; ok {
		t.Fatal("token should be stripped")
	}
}
