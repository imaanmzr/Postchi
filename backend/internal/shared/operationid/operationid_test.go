package operationid

import (
	"slices"
	"testing"
)

func TestCanonicalFromMethodURL(t *testing.T) {
	tests := []struct {
		method string
		url    string
		want   string
	}{
		{"GET", "{{baseUrl}}/users", "get-/users"},
		{"POST", "{{baseUrl}}/users/:id", "post-/users/{id}"},
		{"get", "https://api.example.com/v1/items?page=1", "get-/v1/items"},
		{"DELETE", "/users/{id}/", "delete-/users/{id}"},
		{"", "{{baseUrl}}/x", ""},
	}
	for _, tt := range tests {
		got := CanonicalFromMethodURL(tt.method, tt.url)
		if got != tt.want {
			t.Errorf("CanonicalFromMethodURL(%q, %q) = %q, want %q", tt.method, tt.url, got, tt.want)
		}
	}
}

func TestAliasesForRequest(t *testing.T) {
	aliases := AliasesForRequest("GET", "{{baseUrl}}/users/{id}", "getUserById")
	if !slices.Contains(aliases, "getUserById") {
		t.Fatalf("missing operationId alias: %v", aliases)
	}
	if !slices.Contains(aliases, "get-/users/{id}") {
		t.Fatalf("missing canonical alias: %v", aliases)
	}
}

func TestNormalizeFrontmatterOp(t *testing.T) {
	tests := []struct {
		raw  string
		want []string
	}{
		{"get-/users/{id}", []string{"get-/users/{id}", "get /users/{id}"}},
		{"get /users/{id}", []string{"get /users/{id}", "get-/users/{id}"}},
		{"createUser", []string{"createUser"}},
	}
	for _, tt := range tests {
		got := NormalizeFrontmatterOp(tt.raw)
		for _, w := range tt.want {
			if !slices.Contains(got, w) {
				t.Errorf("NormalizeFrontmatterOp(%q) = %v, want to contain %q", tt.raw, got, w)
			}
		}
	}
}

func TestMatches(t *testing.T) {
	linked := []string{"get-/users/{id}", "createUser"}
	aliases := AliasesForRequest("GET", "{{baseUrl}}/users/{id}", "getUserById")
	if Matches(linked, []string{"getUserById"}) {
		t.Fatal("should not match unrelated alias")
	}
	aliases = append(aliases, "get-/users/{id}")
	if !Matches(linked, aliases) {
		t.Fatal("expected match on canonical id")
	}
}
