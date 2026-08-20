package importexport

import (
	"testing"

	"github.com/imaanmzr/postchi/backend/internal/importexport/gitsync"
)

func TestParseDiscoveredRootStripsRootPath(t *testing.T) {
	files := []gitsync.SourceFile{
		{Path: "api/collection.bru", Content: []byte("meta {\n  name: API\n  type: collection\n}\n")},
		{Path: "api/health.bru", Content: []byte("meta {\n  name: Health\n  type: http\n}\nget {\n  url: https://example.com/health\n}\n")},
		{Path: "api/Hub-Wallet/folder.bru", Content: []byte("meta {\n  name: Hub-Wallet\n}\n")},
		{Path: "api/Hub-Wallet/login.bru", Content: []byte("meta {\n  name: login\n  type: http\n}\npost {\n  url: https://example.com/login\n}\n")},
	}
	root := gitsync.Discover(files)[0]
	if root.RootPath != "api" {
		t.Fatalf("root=%q", root.RootPath)
	}
	col, err := parseDiscoveredRoot(root, "Imported")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	_, reqs := countTree(col)
	if reqs != 2 {
		t.Fatalf("requests=%d children=%d", reqs, len(col.Children))
	}
	hub := findChild(col, "Hub-Wallet")
	if hub == nil || len(hub.Requests) != 1 {
		t.Fatalf("hub=%+v", hub)
	}
	if findChild(col, "api") != nil {
		t.Fatal("unexpected extra api folder wrapper")
	}
}

func TestStrayRootCollectionBruPartialImport(t *testing.T) {
	files := []gitsync.SourceFile{
		{Path: "collection.bru", Content: []byte("meta {\n  name: Wrapper\n  type: collection\n}\n")},
		{Path: "health.bru", Content: []byte("meta {\n  name: Health\n  type: http\n}\nget {\n  url: https://example.com/health\n}\n")},
		{Path: "api/collection.bru", Content: []byte("meta {\n  name: API\n  type: collection\n}\n")},
		{Path: "api/Hub-Wallet/folder.bru", Content: []byte("meta {\n  name: Hub-Wallet\n}\n")},
		{Path: "api/Hub-Wallet/login.bru", Content: []byte("meta {\n  name: login\n  type: http\n}\npost {\n  url: https://example.com/login\n}\n")},
	}
	roots := gitsync.Discover(files)
	parsed := parseRepositoryRoots(roots, "Imported")
	if len(parsed.Collections) != 2 {
		t.Fatalf("collections=%d errors=%v", len(parsed.Collections), parsed.Errors)
	}
	var rootReqs, apiReqs int
	for _, col := range parsed.Collections {
		_, reqs := countTree(col)
		switch {
		case len(col.Children) == 0 && reqs == 1:
			rootReqs = reqs
		case findChild(col, "Hub-Wallet") != nil:
			apiReqs = reqs
		}
	}
	if rootReqs != 1 || apiReqs != 1 {
		t.Fatalf("rootReqs=%d apiReqs=%d collections=%+v errors=%v", rootReqs, apiReqs, parsed.Collections, parsed.Errors)
	}
}
