package importexport

import (
	"context"
	"testing"

	"github.com/imaanmzr/postchi/backend/internal/importexport/gitsync"
	"github.com/imaanmzr/postchi/backend/internal/shared/gitrepo"
)

func TestGitImportPipelineWithMockServer(t *testing.T) {
	gitServer := newGitLabFileServer(t, map[string]string{
		"collection.bru":    "meta {\n  name: Repository API\n  type: collection\n}\n",
		"Orders/folder.bru": "meta {\n  name: Orders\n  seq: 1\n}\n",
		"Orders/list.bru":   "meta {\n  name: List orders\n  type: http\n  seq: 1\n}\n\nget {\n  url: https://example.com/orders\n}\n",
		"health.bru":        "meta {\n  name: Health\n  type: http\n  seq: 1\n}\n\nget {\n  url: https://example.com/health\n}\n",
	})
	defer gitServer.Close()

	client, err := gitrepo.New(gitrepo.Config{
		RepoURL:      gitServer.URL + "/group/repository",
		Branch:       "main",
		Token:        "gitlab-token",
		HTTPClient:   gitServer.Client(),
		APIBaseURL:   gitServer.URL,
		MaxTreeFiles: 10_000,
		MaxFileBytes: maxGitBrunoFileBytes,
	})
	if err != nil {
		t.Fatal(err)
	}
	files, err := gitsync.FetchRepository(context.Background(), client)
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("no files fetched")
	}
	roots := gitsync.Discover(files)
	if len(roots) == 0 {
		t.Fatalf("no roots discovered from %+v", files)
	}
	parsed := parseRepositoryRoots(roots, "Imported Git API")
	if len(parsed.Collections) == 0 {
		t.Fatalf("no collections parsed, errors=%v files=%d roots=%d", parsed.Errors, len(files), len(roots))
	}
}
