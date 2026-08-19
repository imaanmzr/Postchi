package docsync

import "testing"

func TestBuildDocGraph(t *testing.T) {
	docs := []WorkspaceDoc{
		{Slug: "auth", Title: "Auth", ContentMD: "See [[users]]", LinkedOperationIDs: []string{"login"}},
		{Slug: "users", Title: "Users", ContentMD: "User docs"},
	}
	graph := buildDocGraph(docs, nil, nil, nil)
	if len(graph.Nodes) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(graph.Nodes))
	}
	if len(graph.Edges) != 2 {
		t.Fatalf("expected 2 edges, got %d", len(graph.Edges))
	}
}
