package gitrepo

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParseURL(t *testing.T) {
	t.Run("GitHub", func(t *testing.T) {
		got, err := ParseURL("https://github.com/acme/bruno.git")
		if err != nil {
			t.Fatal(err)
		}
		if got.Provider != ProviderGitHub || got.RepoPath != "acme/bruno" || got.APIBase != "https://api.github.com" {
			t.Fatalf("unexpected result: %+v", got)
		}
	})

	t.Run("GitLab tree URL", func(t *testing.T) {
		got, err := ParseURL("https://gitlab.com/acme/platform/api/-/tree/develop/collections/orders")
		if err != nil {
			t.Fatal(err)
		}
		if got.Provider != ProviderGitLab || got.RepoPath != "acme/platform/api" {
			t.Fatalf("unexpected result: %+v", got)
		}
		if got.Branch != "develop" || got.PathPrefix != "collections/orders" {
			t.Fatalf("unexpected browse hints: %+v", got)
		}
	})
}

func TestGitHubListAndFetch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/git/trees/"):
			_, _ = w.Write([]byte(`{"tree":[
				{"path":"collections/collection.bru","type":"blob"},
				{"path":"collections/nested/get.bru","type":"blob"},
				{"path":"other/ignored.bru","type":"blob"}
			]}`))
		case strings.Contains(r.URL.Path, "/contents/"):
			if r.Header.Get("Authorization") != "Bearer token" {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			content := base64.StdEncoding.EncodeToString([]byte("meta {\n  name: request\n}"))
			_, _ = w.Write([]byte(`{"content":"` + content + `","encoding":"base64","size":24}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	client, err := New(Config{
		RepoURL:    "https://github.com/acme/repo",
		PathPrefix: "collections",
		Token:      "token",
		HTTPClient: srv.Client(),
		APIBaseURL: srv.URL,
	})
	if err != nil {
		t.Fatal(err)
	}
	files, err := client.ListFiles(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 || files[0] != "collections/collection.bru" || files[1] != "collections/nested/get.bru" {
		t.Fatalf("unexpected files: %v", files)
	}
	if _, err := client.FetchFile(context.Background(), files[1]); err != nil {
		t.Fatal(err)
	}
}

func TestGitLabListNormalizesPrefixAndIgnoresNonBlobs(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Private-Token") != "token" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		_, _ = w.Write([]byte(`[
			{"path":"collection.bru","type":"blob"},
			{"path":"folder","type":"tree"},
			{"path":"nested/get.bru","type":"blob"}
		]`))
	}))
	defer srv.Close()

	client, err := New(Config{
		RepoURL:    "https://gitlab.com/acme/repo",
		PathPrefix: "bruno",
		Token:      "token",
		HTTPClient: srv.Client(),
		APIBaseURL: srv.URL,
	})
	if err != nil {
		t.Fatal(err)
	}
	files, err := client.ListFiles(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 || files[0] != "bruno/collection.bru" || files[1] != "bruno/nested/get.bru" {
		t.Fatalf("unexpected files: %v", files)
	}
}

func TestProviderErrorsAreClassified(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "API rate limit exceeded", http.StatusForbidden)
	}))
	defer srv.Close()

	client, err := New(Config{
		RepoURL:    "https://github.com/acme/repo",
		HTTPClient: srv.Client(),
		APIBaseURL: srv.URL,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.ListFiles(context.Background())
	if Kind(err) != ErrorRateLimited {
		t.Fatalf("expected rate limit error, got %v (%s)", err, Kind(err))
	}
}
