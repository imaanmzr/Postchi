package domain

import "testing"

func TestExtractPlaceholderNames(t *testing.T) {
	got := ExtractPlaceholderNames("https://{{host}}/v1?key={{token}}", "{{apiKey}}")
	if len(got) != 3 {
		t.Fatalf("expected 3 names, got %d: %v", len(got), got)
	}
}

func TestExtractPlaceholderNamesDedup(t *testing.T) {
	got := ExtractPlaceholderNames("{{foo}} and {{foo}}")
	if len(got) != 1 || got[0] != "foo" {
		t.Fatalf("expected single foo, got %v", got)
	}
}
