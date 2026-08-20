package bruno

import (
	"strings"
	"testing"
)

func TestToVarsAndExportMeta(t *testing.T) {
	vars := ToVars("host: localhost\n", "token: res.body.token\n")
	if len(vars.PreRequest) != 1 || len(vars.PostResponse) != 1 {
		t.Fatalf("vars=%+v", vars)
	}
	meta := ExportCollectionMeta("API", vars)
	if !strings.Contains(meta, "vars:pre-request") || !strings.Contains(meta, "type: collection") {
		t.Fatalf("meta=%q", meta)
	}
}

func TestExportRequestBodyVariants(t *testing.T) {
	cases := []BruRequest{
		{Name: "JSON", Method: "POST", URL: "https://example.com", BodyType: "json", Body: `{"ok":true}`},
		{Name: "DefaultBody", Method: "POST", URL: "https://example.com", Body: `{"ok":true}`},
		{Name: "GraphQL", Method: "POST", URL: "https://example.com", BodyType: "graphql", Body: "{ ping }", GraphQLVars: `{}`},
		{Name: "File", Method: "POST", URL: "https://example.com", BodyType: "file", Body: "payload"},
		{Name: "Custom", Method: "POST", URL: "https://example.com", BodyType: "custom", Body: "plain"},
		{Name: "Headers", Method: "GET", URL: "https://example.com", Headers: []KV{{Key: "Accept", Value: "application/json"}}},
		{Name: "Scripts", Method: "GET", URL: "https://example.com", PreRequestScript: "console.log(1)", TestScript: "test('ok')"},
	}
	for _, req := range cases {
		out := ExportRequest(req)
		if out == "" || !strings.Contains(out, req.Name) {
			t.Fatalf("export for %q empty or missing name:\n%s", req.Name, out)
		}
	}
}
