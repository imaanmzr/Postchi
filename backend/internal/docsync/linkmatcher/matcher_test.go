package linkmatcher

import "testing"

func TestMatchContentMethodPath(t *testing.T) {
	doc := Doc{
		ID: "d1", Slug: "users", Title: "Users",
		ContentMD: "Call GET /users/{id} to fetch a user.",
	}
	req := Request{ID: "r1", Method: "GET", URL: "{{baseUrl}}/users/:id", Name: "Get user"}
	c, ok := matchContentMethodPath(doc, req)
	if !ok {
		t.Fatal("expected content match")
	}
	if c.Confidence != "high" || c.Reason != "content_method_path" {
		t.Fatalf("unexpected candidate: %+v", c)
	}
}

func TestAnalyzeSkipsViaCallback(t *testing.T) {
	docs := []Doc{{ID: "d1", Title: "Users", ContentMD: "GET /users"}}
	reqs := []Request{{ID: "r1", Method: "GET", URL: "/users", Name: "Users"}}
	out := Analyze(docs, reqs, func(docID, requestID string) bool { return true })
	if len(out) != 0 {
		t.Fatalf("expected no candidates, got %d", len(out))
	}
}

func TestMatchPathAlignment_exactOnly(t *testing.T) {
	doc := Doc{ID: "d1", SourcePath: "docs/user-profile", Slug: "user-profile", Title: "User Profile"}
	req := Request{ID: "r1", Name: "get-user"}
	if _, ok := matchPathAlignment(doc, req); ok {
		t.Fatal("expected no substring path alignment match")
	}
}

func TestMatchTitleSimilarity_requiresFullOverlap(t *testing.T) {
	doc := Doc{ID: "d1", Title: "Get User Profile", Slug: "get-user-profile"}
	req := Request{ID: "r1", Name: "Get User Settings"}
	if _, ok := matchTitleSimilarity(doc, req); ok {
		t.Fatal("expected no partial title match when not all request tokens are present")
	}
	doc2 := Doc{ID: "d2", Title: "Get User", Slug: "get-user"}
	req2 := Request{ID: "r2", Name: "Get"}
	if _, ok := matchTitleSimilarity(doc2, req2); ok {
		t.Fatal("expected no match with fewer than 2 tokens")
	}
}
