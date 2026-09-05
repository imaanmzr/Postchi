package gitrepo

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGitLabTreeNotFoundBranchMissing(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/repository/tree"):
			http.Error(w, "404", http.StatusNotFound)
		case strings.Contains(r.URL.Path, "/repository/branches/"):
			http.Error(w, "404", http.StatusNotFound)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	client, err := New(Config{
		RepoURL:    "https://gitlab.com/acme/repo",
		Branch:     "feature/missing",
		PathPrefix: "bruno-collection",
		Token:      "token",
		HTTPClient: srv.Client(),
		APIBaseURL: srv.URL,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.ListFiles(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "branch") || !strings.Contains(err.Error(), "feature/missing") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGitLabTreeNotFoundPathMissing(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/repository/tree"):
			http.Error(w, "404", http.StatusNotFound)
		case strings.Contains(r.URL.Path, "/repository/branches/"):
			_, _ = w.Write([]byte(`{"name":"develop"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	client, err := New(Config{
		RepoURL:    "https://gitlab.com/acme/repo",
		Branch:     "develop",
		PathPrefix: "wrong/path",
		Token:      "token",
		HTTPClient: srv.Client(),
		APIBaseURL: srv.URL,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.ListFiles(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "path") || !strings.Contains(err.Error(), "wrong/path") {
		t.Fatalf("unexpected error: %v", err)
	}
}
