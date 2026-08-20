package gitsync

import "testing"

func TestNestedBrunoRoots(t *testing.T) {
	files := []SourceFile{
		{Path: "api/collection.bru", Content: []byte("meta { name: API type: collection }")},
		{Path: "api/ping.bru", Content: []byte("meta { name: Ping type: http } get { url: https://example.com/ping }")},
		{Path: "api/orders/collection.bru", Content: []byte("meta { name: Orders type: collection }")},
		{Path: "api/orders/list.bru", Content: []byte("meta { name: List type: http } get { url: https://example.com/orders }")},
	}
	roots := Discover(files)
	if len(roots) != 2 {
		t.Fatalf("roots=%d", len(roots))
	}
}

func TestFileBelongsToBrunoRoot(t *testing.T) {
	allRoots := []string{"api", "api/orders"}
	if !fileBelongsToBrunoRoot("api/ping.bru", "api", allRoots) {
		t.Fatal("expected api/ping.bru under api root")
	}
	if fileBelongsToBrunoRoot("api/orders/list.bru", "api", allRoots) {
		t.Fatal("nested root file should not belong to parent root")
	}
	if !fileBelongsToBrunoRoot("ping.bru", "", []string{"api"}) {
		t.Fatal("top-level file expected")
	}
}

func TestContainsPathSegment(t *testing.T) {
	if containsPathSegment([]string{"foo", "Environments"}, "environments") != true {
		t.Fatal("expected case-insensitive match")
	}
	if containsPathSegment([]string{"api"}, "environments") {
		t.Fatal("expected no match")
	}
}

func TestRelativeRepositoryPathEdgeCases(t *testing.T) {
	if got := relativeRepositoryPath("api/orders/list.bru", ""); got != "api/orders/list.bru" {
		t.Fatalf("got %q", got)
	}
	if got := relativeRepositoryPath("api", "api"); got != "" {
		t.Fatalf("got %q", got)
	}
}
