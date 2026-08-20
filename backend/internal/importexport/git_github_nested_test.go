package importexport

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/imaanmzr/postchi/backend/internal/importexport/gitsync"
	"github.com/imaanmzr/postchi/backend/internal/shared/gitrepo"
)

func TestGitHubNestedWithPathPrefix(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/git/trees/"):
			_, _ = w.Write([]byte(`{"tree":[
				{"path":"bruno/collection.bru","type":"blob"},
				{"path":"bruno/Hub-Wallet/folder.bru","type":"blob"},
				{"path":"bruno/Hub-Wallet/login.bru","type":"blob"},
				{"path":"bruno/health.bru","type":"blob"}
			]}`))
		default:
			content := base64.StdEncoding.EncodeToString([]byte("meta {\n  name: x\n  type: http\n}\nget {\n  url: https://example.com\n}\n"))
			_, _ = w.Write([]byte(`{"content":"` + content + `","encoding":"base64","size":24}`))
		}
	}))
	defer srv.Close()

	client, err := gitrepo.New(gitrepo.Config{
		RepoURL:      "https://github.com/acme/repo",
		PathPrefix:   "bruno",
		Token:        "token",
		HTTPClient:   srv.Client(),
		APIBaseURL:   srv.URL,
		MaxTreeFiles: 10000,
		MaxFileBytes: maxGitBrunoFileBytes,
	})
	if err != nil {
		t.Fatal(err)
	}
	files, err := gitsync.FetchRepository(context.Background(), client)
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if len(files) != 4 {
		t.Fatalf("files=%d %+v", len(files), files)
	}
	roots := gitsync.Discover(files)
	parsed := parseRepositoryRoots(roots, "Imported")
	if len(parsed.Collections) != 1 {
		t.Fatalf("collections=%d errors=%v", len(parsed.Collections), parsed.Errors)
	}
	_, reqs := countTree(parsed.Collections[0])
	if reqs < 2 {
		t.Fatalf("expected nested requests, got %d", reqs)
	}
}

func TestGitHubNestedAtRepoRoot(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/git/trees/"):
			_, _ = w.Write([]byte(`{"tree":[
				{"path":"collection.bru","type":"blob"},
				{"path":"Hub-Wallet/folder.bru","type":"blob"},
				{"path":"Hub-Wallet/login.bru","type":"blob"},
				{"path":"health.bru","type":"blob"}
			]}`))
		default:
			content := base64.StdEncoding.EncodeToString([]byte("meta {\n  name: x\n  type: http\n}\nget {\n  url: https://example.com\n}\n"))
			_, _ = w.Write([]byte(`{"content":"` + content + `","encoding":"base64","size":24}`))
		}
	}))
	defer srv.Close()

	client, err := gitrepo.New(gitrepo.Config{
		RepoURL:      "https://github.com/acme/repo",
		Token:        "token",
		HTTPClient:   srv.Client(),
		APIBaseURL:   srv.URL,
		MaxTreeFiles: 10000,
		MaxFileBytes: maxGitBrunoFileBytes,
	})
	if err != nil {
		t.Fatal(err)
	}
	files, err := gitsync.FetchRepository(context.Background(), client)
	if err != nil {
		t.Fatal(err)
	}
	roots := gitsync.Discover(files)
	parsed := parseRepositoryRoots(roots, "Imported")
	_, reqs := countTree(parsed.Collections[0])
	if reqs < 2 {
		t.Fatalf("expected nested requests, got %d roots=%+v files=%+v", reqs, roots, files)
	}
}
