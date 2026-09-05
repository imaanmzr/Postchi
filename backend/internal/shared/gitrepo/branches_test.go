package gitrepo

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestListGitHubBranchesPagination(t *testing.T) {
	page := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/repos/acme/repo"):
			_, _ = w.Write([]byte(`{"default_branch":"main"}`))
		case strings.Contains(r.URL.Path, "/branches"):
			page++
			if page == 1 {
				items := make([]string, 0, 100)
				for i := 0; i < 100; i++ {
					items = append(items, `{"name":"branch-`+string(rune('a'+i%26))+`","protected":false}`)
				}
				_, _ = w.Write([]byte("[" + strings.Join(items, ",") + "]"))
				return
			}
			_, _ = w.Write([]byte(`[{"name":"develop","protected":false},{"name":"main","protected":true}]`))
		default:
			http.NotFound(w, r)
		}
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
	branches, err := client.ListBranches(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(branches) < 102 {
		t.Fatalf("expected at least 102 branches, got %d", len(branches))
	}
	foundMain := false
	for _, branch := range branches {
		if branch.Name == "main" && branch.IsDefault {
			foundMain = true
		}
	}
	if !foundMain {
		t.Fatal("expected main to be marked default")
	}
}

func TestListGitLabBranchesRequiresToken(t *testing.T) {
	client, err := New(Config{
		RepoURL: "https://gitlab.com/acme/repo",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.ListBranches(context.Background())
	if Kind(err) != ErrorAuthentication {
		t.Fatalf("expected auth error, got %v", err)
	}
}

func TestListGitLabBranches(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Private-Token") != "token" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if !strings.Contains(r.URL.Path, "/repository/branches") {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(`[
			{"name":"develop","default":false,"protected":false},
			{"name":"main","default":true,"protected":true}
		]`))
	}))
	defer srv.Close()

	client, err := New(Config{
		RepoURL:    "https://gitlab.com/acme/repo",
		Token:      "token",
		HTTPClient: srv.Client(),
		APIBaseURL: srv.URL,
	})
	if err != nil {
		t.Fatal(err)
	}
	branches, err := client.ListBranches(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(branches) != 2 || !branches[1].IsDefault {
		t.Fatalf("unexpected branches: %+v", branches)
	}
}

func TestFilterBranches(t *testing.T) {
	branches := []Branch{
		{Name: "main"},
		{Name: "develop"},
		{Name: "feature/pay-482"},
		{Name: "uat/release"},
	}
	filtered := FilterBranches(branches, "feat", 10)
	if len(filtered) != 1 || filtered[0].Name != "feature/pay-482" {
		t.Fatalf("unexpected filter result: %+v", filtered)
	}
	limit := FilterBranches(branches, "", 2)
	if len(limit) != 2 {
		t.Fatalf("expected limit 2, got %d", len(limit))
	}
}
