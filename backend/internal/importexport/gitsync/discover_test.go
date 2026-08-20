package gitsync

import (
	"testing"

	"github.com/imaanmzr/postchi/backend/internal/importexport/detect"
)

func TestDiscoverMixedRepository(t *testing.T) {
	files := []SourceFile{
		{Path: "bruno/collection.bru", Content: []byte("meta { name: API type: collection }")},
		{Path: "bruno/ping.bru", Content: []byte("meta { name: Ping type: http } get { url: https://example.com/ping }")},
		{Path: "postman/api.postman_collection.json", Content: []byte(`{"info":{"name":"API","schema":"https://schema.getpostman.com/json/collection/v2.1.0/collection.json"},"item":[{"name":"Health","request":{"method":"GET","url":"https://example.com/health"}}]}`)},
		{Path: "specs/petstore.yaml", Content: []byte("openapi: 3.0.0\ninfo:\n  title: Pets\npaths:\n  /pets:\n    get:\n      operationId: listPets\n      responses:\n        '200':\n          description: ok\n")},
		{Path: "README.md", Content: []byte("# docs")},
	}
	roots := Discover(files)
	if len(roots) != 3 {
		t.Fatalf("expected 3 roots, got %d", len(roots))
	}
	formats := map[detect.Format]int{}
	for _, root := range roots {
		formats[root.Format]++
	}
	if formats[detect.FormatBruno] != 1 || formats[detect.FormatPostman] != 1 || formats[detect.FormatOpenAPI] != 1 {
		t.Fatalf("unexpected formats: %+v", formats)
	}
}

func TestRelativeRepositoryPath(t *testing.T) {
	if got := RelativeRepositoryPath("collections/api/orders/list.bru", "collections/api"); got != "orders/list.bru" {
		t.Fatalf("got %q", got)
	}
}
