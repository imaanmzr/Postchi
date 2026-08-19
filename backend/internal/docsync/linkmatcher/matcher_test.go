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
