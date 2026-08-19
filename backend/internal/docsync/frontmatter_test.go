package docsync

import "testing"

func TestParseMarkdownDoc_multilineOperations(t *testing.T) {
	content := `---
title: Users API
operations:
  - get-/users/{id}
  - post-/users
---

# Users
`
	title, ops, body := parseMarkdownDoc(content, "users.md")
	if title != "Users API" {
		t.Fatalf("title = %q", title)
	}
	if len(ops) < 2 {
		t.Fatalf("ops = %v", ops)
	}
	if body != "# Users" {
		t.Fatalf("body = %q", body)
	}
}

func TestParseMarkdownDoc_inlineOperations(t *testing.T) {
	content := `---
operations: [get-/users, "post-/users"]
---
Body`
	_, ops, _ := parseMarkdownDoc(content, "x.md")
	if len(ops) < 2 {
		t.Fatalf("ops = %v", ops)
	}
}

func TestNormalizeLinkedOperations_aliases(t *testing.T) {
	ops := normalizeLinkedOperations([]string{"get /users/{id}"})
	found := false
	for _, o := range ops {
		if o == "get-/users/{id}" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected canonical alias, got %v", ops)
	}
}
