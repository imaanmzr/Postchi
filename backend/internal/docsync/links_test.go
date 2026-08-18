package docsync

import "testing"

func TestExtractDocLinks(t *testing.T) {
	docs := []WorkspaceDoc{
		{Slug: "api-auth", Title: "API Auth"},
		{Slug: "getting-started", Title: "Getting Started"},
	}
	idx := buildDocIndex(docs)

	content := `
See [[api-auth]] and [[Getting Started|start here]].
Also [link](./getting-started.md) and [ext](https://example.com).
`
	links := extractDocLinks(content, idx)
	if len(links) != 2 {
		t.Fatalf("expected 2 links, got %d: %v", len(links), links)
	}
}

func TestPathToSlug(t *testing.T) {
	if got := pathToSlug("docs/api/auth.md"); got != "docs-api-auth" {
		t.Fatalf("unexpected slug: %s", got)
	}
	if got := pathToSlug("./foo.md#section"); got != "foo" {
		t.Fatalf("unexpected slug: %s", got)
	}
}

func TestFilePathToSourcePath(t *testing.T) {
	if got := filePathToSourcePath("docs/api/auth.md"); got != "docs/api/auth" {
		t.Fatalf("got %q", got)
	}
	if got := filePathToSourcePath("README.MD"); got != "README" {
		t.Fatalf("got %q", got)
	}
}
