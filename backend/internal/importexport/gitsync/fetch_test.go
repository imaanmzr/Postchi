package gitsync

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/imaanmzr/postchi/backend/internal/shared/gitrepo"
)

func TestFetchRepositoryGitHub(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/git/trees/") {
			_, _ = w.Write([]byte(`{"tree":[{"path":"api/collection.bru","type":"blob"},{"path":"api/ping.json","type":"blob"}]}`))
			return
		}
		content := base64.StdEncoding.EncodeToString([]byte(`{"info":{"name":"API","schema":"https://schema.getpostman.com/json/collection/v2.1.0/collection.json"},"item":[]}`))
		_, _ = w.Write([]byte(`{"content":"` + content + `","encoding":"base64"}`))
	}))
	defer srv.Close()

	client, err := gitrepo.New(gitrepo.Config{
		RepoURL:    "https://github.com/acme/repository",
		PathPrefix: "api",
		HTTPClient: srv.Client(),
		APIBaseURL: srv.URL,
	})
	if err != nil {
		t.Fatal(err)
	}
	files, err := FetchRepository(context.Background(), client)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 {
		t.Fatalf("files = %+v", files)
	}
}
