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

func TestFetchBrunoRepositoryFiltersFiles(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/git/trees/") {
			_, _ = w.Write([]byte(`{"tree":[
				{"path":"api/collection.bru","type":"blob"},
				{"path":"api/orders/list.bru","type":"blob"},
				{"path":"api/environments/local.bru","type":"blob"},
				{"path":"api/README.md","type":"blob"}
			]}`))
			return
		}
		content := base64.StdEncoding.EncodeToString([]byte("meta {\n  name: File\n}\n"))
		_, _ = w.Write([]byte(`{"content":"` + content + `","encoding":"base64","size":23}`))
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
	files, err := fetchBrunoRepository(context.Background(), client)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, file := range files {
		got[file.Path] = true
	}
	if len(files) != 2 || !got["collection.bru"] || !got["orders/list.bru"] {
		t.Fatalf("unexpected files: %+v", files)
	}
}

func TestRelativeRepositoryPath(t *testing.T) {
	if got := gitsync.RelativeRepositoryPath("collections/api/orders/list.bru", "collections/api"); got != "orders/list.bru" {
		t.Fatalf("relative path = %q", got)
	}
}
