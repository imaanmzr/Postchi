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
	title, ops, requestNames, body := parseMarkdownDoc(content, "users.md")
	if title != "Users API" {
		t.Fatalf("title = %q", title)
	}
	if len(ops) < 2 {
		t.Fatalf("ops = %v", ops)
	}
	if len(requestNames) != 0 {
		t.Fatalf("requestNames = %v", requestNames)
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
	_, ops, _, _ := parseMarkdownDoc(content, "x.md")
	if len(ops) < 2 {
		t.Fatalf("ops = %v", ops)
	}
}

func TestParseMarkdownDoc_requestField(t *testing.T) {
	content := `---
title: Get User
request: get-user
operations:
  - get-/users/{id}
---
Body`
	_, ops, requestNames, body := parseMarkdownDoc(content, "get-user.md")
	if len(ops) == 0 {
		t.Fatalf("expected operations")
	}
	if len(requestNames) != 1 || requestNames[0] != "get-user" {
		t.Fatalf("requestNames = %v", requestNames)
	}
	if body != "Body" {
		t.Fatalf("body = %q", body)
	}
}

func TestParseMarkdownDoc_requestsList(t *testing.T) {
	content := `---
requests:
  - get-user
  - get-user-profile
---
Body`
	_, _, requestNames, _ := parseMarkdownDoc(content, "x.md")
	if len(requestNames) != 2 {
		t.Fatalf("requestNames = %v", requestNames)
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

func TestNormalizeLinkedRequestNames(t *testing.T) {
	names := normalizeLinkedRequestNames([]string{"Get User", "get-user", "Get User"})
	if len(names) != 1 || names[0] != "get-user" {
		t.Fatalf("names = %v", names)
	}
}
