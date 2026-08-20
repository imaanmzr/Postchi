package linkmatcher

import "testing"

func TestRenderTemplate(t *testing.T) {
	vars := map[string]string{
		"request_slug":    "get-user",
		"collection_slug": "users-api",
		"path_prefix":     "docs",
	}
	got := RenderTemplate("docs/{collection_slug}/{request_slug}.md", vars)
	if got != "docs/users-api/get-user.md" {
		t.Fatalf("got %q", got)
	}
}

func TestMatchPathTemplate(t *testing.T) {
	docs := []Doc{{ID: "d1", SourcePath: "docs/users-api/get-user"}}
	requests := []Request{{
		ID: "r1", Name: "get-user", CollectionID: "c1", CollectionName: "Users API",
	}}
	collections := map[string]CollectionInfo{
		"c1": {ID: "c1", Name: "Users API"},
	}
	candidates := MatchPathTemplate("docs/{collection_slug}/{request_slug}.md", "", docs, requests, collections)
	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(candidates))
	}
	if candidates[0].Reason != "path_template" {
		t.Fatalf("unexpected reason: %s", candidates[0].Reason)
	}
}

func TestValidateLinkTemplate(t *testing.T) {
	if !ValidateLinkTemplate("docs/{request_slug}.md") {
		t.Fatal("expected valid template")
	}
	if ValidateLinkTemplate("docs/static.md") {
		t.Fatal("expected invalid template")
	}
	if !ValidateLinkTemplate("") {
		t.Fatal("empty template should be valid")
	}
}
