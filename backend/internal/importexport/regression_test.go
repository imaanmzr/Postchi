package importexport

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/imaanmzr/postchi/backend/internal/importexport/detect"
	"github.com/imaanmzr/postchi/backend/internal/importexport/gitsync"
)

func TestParseBrunoSingleFileRequest(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "bruno", "my-collection-request.bru"))
	if err != nil {
		t.Fatal(err)
	}
	col, err := parseBrunoSingleFile(data, "my-collection-request.bru")
	if err != nil {
		t.Fatal(err)
	}
	if len(col.Requests) != 1 {
		t.Fatalf("expected request import, got %+v", col)
	}
}

func TestParseBrunoZipRequiresRootMeta(t *testing.T) {
	files := []brunoSourceFile{{Path: "ping.bru", Content: []byte("meta { name: Ping type: http } get { url: https://example.com }")}}
	_, err := parseBrunoFiles(files, defaultBrunoParseOptions())
	if err == nil {
		t.Fatal("expected missing collection.bru error")
	}
}

func TestPostmanEmptyMethodWarnings(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "postman", "empty-method.json"))
	if err != nil {
		t.Fatal(err)
	}
	result, err := ParsePostmanWithWarnings(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Warnings) != 1 {
		t.Fatalf("warnings = %+v", result.Warnings)
	}
	if len(result.Collection.Requests) != 1 {
		t.Fatalf("requests = %d", len(result.Collection.Requests))
	}
}

func TestParseRepositoryRootsMixed(t *testing.T) {
	roots := []gitsync.DiscoveredRoot{
		{
			Format:   detect.FormatPostman,
			FilePath: "api.postman_collection.json",
			Content:  []byte(`{"info":{"name":"API","schema":"https://schema.getpostman.com/json/collection/v2.1.0/collection.json"},"item":[{"name":"Health","request":{"method":"GET","url":"https://example.com/health"}}]}`),
		},
	}
	parsed := parseRepositoryRoots(roots, "Git Import")
	if len(parsed.Collections) != 1 || len(parsed.Errors) != 0 {
		t.Fatalf("parsed = %+v", parsed)
	}
	if parsed.Collections[0].SourcePath == "" {
		t.Fatal("expected source path on collection")
	}
}

func TestParseDiscoveredOpenAPI(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "openapi", "minimal.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	roots := []gitsync.DiscoveredRoot{{
		Format:   detect.FormatOpenAPI,
		FilePath: "specs/petstore.yaml",
		Content:  data,
	}}
	parsed := parseRepositoryRoots(roots, "Pets")
	if len(parsed.Collections) != 1 || parsed.Collections[0].Requests == nil {
		t.Fatalf("parsed = %+v errors=%v", parsed.Collections, parsed.Errors)
	}
	if parsed.Collections[0].Requests[0].SourcePath == "" {
		t.Fatal("expected operation source path")
	}
}
