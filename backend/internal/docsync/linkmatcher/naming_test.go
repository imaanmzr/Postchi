package linkmatcher

import "testing"

func TestNormalizeSlug(t *testing.T) {
	if got := NormalizeSlug("Get User"); got != "get-user" {
		t.Fatalf("got %q", got)
	}
	if got := NormalizeSlug("get_user"); got != "get-user" {
		t.Fatalf("got %q", got)
	}
}

func TestMatchExactName(t *testing.T) {
	docs := []Doc{{
		ID: "d1", Slug: "docs-api-get-user", Title: "Get User", SourcePath: "docs/api/get-user",
	}}
	requests := []Request{{ID: "r1", Name: "get-user"}}
	candidates := MatchExactName(docs, requests)
	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(candidates))
	}
	if candidates[0].Reason != "exact_name" || candidates[0].Confidence != "exact" {
		t.Fatalf("unexpected candidate: %+v", candidates[0])
	}
}

func TestMatchExactName_noSubstringMatch(t *testing.T) {
	docs := []Doc{{ID: "d1", SourcePath: "docs/user-profile", Slug: "user-profile", Title: "User Profile"}}
	requests := []Request{{ID: "r1", Name: "get-user"}}
	if len(MatchExactName(docs, requests)) != 0 {
		t.Fatal("expected no match")
	}
}

func TestPartitionUnique(t *testing.T) {
	candidates := []Candidate{
		{DocID: "d1", RequestID: "r1", Reason: "exact_name", Confidence: "exact"},
		{DocID: "d2", RequestID: "r1", Reason: "exact_name", Confidence: "exact"},
		{DocID: "d3", RequestID: "r3", Reason: "exact_name", Confidence: "exact"},
	}
	auto, ambiguous := PartitionUnique(candidates)
	if len(auto) != 1 || auto[0].DocID != "d3" {
		t.Fatalf("auto = %+v", auto)
	}
	if len(ambiguous) != 2 {
		t.Fatalf("ambiguous = %+v", ambiguous)
	}
}
