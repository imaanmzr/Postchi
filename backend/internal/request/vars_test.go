package request

import (
	"encoding/json"
	"testing"

	"github.com/imaanmzr/postchi/backend/internal/shared/domain"
)

func TestMergeCollectionVarsChain(t *testing.T) {
	parent := domain.VariablesSpec{
		PreRequest: []domain.PreRequestVar{
			{Enabled: true, Name: "baseUrl", Value: "https://parent.example.com", Type: "string"},
			{Enabled: true, Name: "shared", Value: "parent", Type: "string"},
		},
	}
	child := domain.VariablesSpec{
		PreRequest: []domain.PreRequestVar{
			{Enabled: true, Name: "shared", Value: "child", Type: "string"},
			{Enabled: true, Name: "token", Value: "abc", Type: "string"},
		},
	}
	parentData, _ := json.Marshal(parent)
	childData, _ := json.Marshal(child)

	vars := map[string]string{}
	mergeCollectionVars(vars, parentData)
	mergeCollectionVars(vars, childData)

	if vars["baseUrl"] != "https://parent.example.com" {
		t.Fatalf("baseUrl=%q", vars["baseUrl"])
	}
	if vars["shared"] != "child" {
		t.Fatalf("shared=%q want child override", vars["shared"])
	}
	if vars["token"] != "abc" {
		t.Fatalf("token=%q", vars["token"])
	}
}

func TestEvalPostResponseExprs(t *testing.T) {
	exprs := []domain.PostResponseVar{
		{Enabled: true, Name: "status", Expr: "res.status"},
		{Enabled: true, Name: "id", Expr: "res.body.id"},
	}
	out := evalPostResponseExprs(exprs, 200, `{"id":"xyz"}`)
	if out["status"] != "200" {
		t.Fatalf("status=%q", out["status"])
	}
	if out["id"] != "xyz" {
		t.Fatalf("id=%q", out["id"])
	}
}

func TestFormatResponseBodyJSON(t *testing.T) {
	raw := `{"ok":true,"items":[1,2]}`
	got := formatResponseBody(raw)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(got), &parsed); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if parsed["ok"] != true {
		t.Fatalf("ok=%v", parsed["ok"])
	}
	items, ok := parsed["items"].([]any)
	if !ok || len(items) != 2 {
		t.Fatalf("items=%v", parsed["items"])
	}
}

func TestFormatResponseBodyPlainText(t *testing.T) {
	got := formatResponseBody("plain text")
	if got != "plain text" {
		t.Fatalf("got %q", got)
	}
}
