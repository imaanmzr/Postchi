package importexport

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/imaanmzr/postchi/backend/internal/importexport/detect"
	"github.com/imaanmzr/postchi/backend/internal/importexport/gitsync"
	"github.com/imaanmzr/postchi/backend/internal/importexport/model"
)

func TestParseRepositoryRootsAllFormats(t *testing.T) {
	postmanData, _ := os.ReadFile(filepath.Join("testdata", "postman", "nested.json"))
	openAPIData, _ := os.ReadFile(filepath.Join("testdata", "openapi", "minimal.yaml"))
	openCollectionData, _ := os.ReadFile(filepath.Join("testdata", "opencollection", "collection.json"))

	roots := []gitsync.DiscoveredRoot{
		{
			Format: detect.FormatPostman,
			FilePath: "api.postman_collection.json",
			Content: postmanData,
		},
		{
			Format: detect.FormatOpenAPI,
			FilePath: "specs/api.yaml",
			Content: openAPIData,
		},
		{
			Format: detect.FormatOpenCollection,
			FilePath: "collection.json",
			Content: openCollectionData,
		},
		{
			Format: detect.FormatBruno,
			RootPath: "bruno-api",
			Files: []gitsync.SourceFile{
				{Path: "collection.bru", Content: []byte("meta {\n  name: Bruno API\n  type: collection\n}\n")},
				{Path: "ping.bru", Content: []byte("meta {\n  name: Ping\n  type: http\n}\nget {\n  url: https://example.com/ping\n}\n")},
			},
		},
	}
	parsed := parseRepositoryRoots(roots, "fallback")
	if len(parsed.Collections) != 4 {
		t.Fatalf("collections=%d errors=%v", len(parsed.Collections), parsed.Errors)
	}
}

func TestParseRepositoryRootsCollectsErrors(t *testing.T) {
	roots := []gitsync.DiscoveredRoot{
		{Format: detect.FormatPostman, FilePath: "broken.json", Content: []byte("{")},
		{Format: detect.FormatOpenAPI, FilePath: "bad.yaml", Content: []byte("not: openapi")},
		{Format: detect.FormatBruno, RootPath: "empty", Files: []gitsync.SourceFile{
			{Path: "collection.bru", Content: []byte("meta {\n  name: Empty\n  type: collection\n}\n")},
		}},
		{Format: detect.Format("unknown"), FilePath: "x.txt", Content: []byte("x")},
	}
	parsed := parseRepositoryRoots(roots, "fallback")
	if len(parsed.Collections) != 0 {
		t.Fatalf("expected no collections, got %d", len(parsed.Collections))
	}
	if len(parsed.Errors) < 3 {
		t.Fatalf("errors=%v", parsed.Errors)
	}
}

func TestTagSourcePathsNestedChildren(t *testing.T) {
	col := modelCollectionWithChild()
	tagSourcePaths(&col, "root.json")
	if col.SourcePath != "root.json" {
		t.Fatalf("source path=%q", col.SourcePath)
	}
	if col.Children[0].Requests[0].SourcePath == "" {
		t.Fatal("expected child request source path")
	}
}

func modelCollectionWithChild() model.Collection {
	return model.Collection{
		Name: "Root",
		Children: []model.Collection{{
			Name: "Child",
			Requests: []model.Request{{Name: "Ping"}},
		}},
	}
}

func TestFilepathHelpersEdgeCases(t *testing.T) {
	if got := filepathBaseName(""); got != "" {
		t.Fatalf("base empty=%q", got)
	}
	if got := filepathBaseName("/only/"); got != "only" {
		t.Fatalf("base trailing=%q", got)
	}
	if got := filepathExt("noext"); got != "" {
		t.Fatalf("ext=%q", got)
	}
}
