package importexport

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/imaanmzr/postchi/backend/internal/importexport/bruno"
	openapiimport "github.com/imaanmzr/postchi/backend/internal/importexport/openapi"
)

func TestBrunoParseRequest(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "bruno", "sample.bru"))
	if err != nil {
		t.Fatal(err)
	}
	parsed := bruno.Parse(string(data))
	if parsed.Name != "Get Users" {
		t.Fatalf("name=%q", parsed.Name)
	}
	req := bruToNorm(bruno.ToRequest(parsed))
	if req.Method != "GET" || req.URL != "{{baseUrl}}/users" {
		t.Fatalf("method/url: %s %s", req.Method, req.URL)
	}
	if len(req.Headers) != 1 || req.Headers[0].Key != "Accept" {
		t.Fatalf("headers: %+v", req.Headers)
	}
	if req.Auth.Type != "bearer" {
		t.Fatalf("auth: %+v", req.Auth)
	}
	if req.PreRequestScript == "" || req.TestScript == "" {
		t.Fatal("scripts missing")
	}
	exported := bruno.ExportRequest(bruFromNorm(req))
	reParsed := bruno.Parse(exported)
	reReq := bruToNorm(bruno.ToRequest(reParsed))
	if reReq.Method != req.Method || reReq.URL != req.URL {
		t.Fatalf("round-trip failed: %+v", reReq)
	}
}

func TestBrunoCollectionVars(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "bruno", "collection.bru"))
	if err != nil {
		t.Fatal(err)
	}
	parsed := bruno.Parse(string(data))
	spec := bruVarsToSpec(bruno.ToVars(parsed.Sections["vars:pre-request"], parsed.Sections["vars:post-response"]))
	if len(spec.PreRequest) != 1 || spec.PreRequest[0].Name != "token" {
		t.Fatalf("pre: %+v", spec.PreRequest)
	}
	if len(spec.PostResponse) != 1 || spec.PostResponse[0].Expr != "res.body.id" {
		t.Fatalf("post: %+v", spec.PostResponse)
	}
	back := specToBruVars(spec)
	exported := bruno.ExportCollectionMeta(parsed.Name, back)
	reParsed := bruno.Parse(exported)
	reSpec := bruVarsToSpec(bruno.ToVars(reParsed.Sections["vars:pre-request"], reParsed.Sections["vars:post-response"]))
	if len(reSpec.PreRequest) != 1 || reSpec.PreRequest[0].Name != "token" {
		t.Fatalf("round-trip pre: %+v", reSpec.PreRequest)
	}
}

func TestPostmanImportExport(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "postman", "nested.json"))
	if err != nil {
		t.Fatal(err)
	}
	col, err := ParsePostman(data)
	if err != nil {
		t.Fatal(err)
	}
	if col.Name != "Nested Collection" {
		t.Fatalf("name=%q", col.Name)
	}
	if len(col.Variables.PreRequest) != 1 {
		t.Fatalf("vars: %+v", col.Variables)
	}
	if len(col.Children) != 1 || len(col.Children[0].Requests) != 1 {
		t.Fatalf("tree: children=%d", len(col.Children))
	}
	req := col.Children[0].Requests[0]
	if req.Method != "GET" || req.Name != "Get Users" {
		t.Fatalf("req: %+v", req)
	}
	exported := ExportPostmanCollection(col)
	items, ok := exported["item"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("export items: %T %+v", exported["item"], exported["item"])
	}
}

func TestPostmanMixedOrderAndEmptyFolders(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "postman", "mixed_order.json"))
	if err != nil {
		t.Fatal(err)
	}
	col, err := ParsePostman(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(col.Children) != 2 {
		t.Fatalf("expected 2 child folders, got %d", len(col.Children))
	}
	if len(col.Requests) != 2 {
		t.Fatalf("expected 2 root requests, got %d", len(col.Requests))
	}
	if col.Requests[0].Name != "Request First" || col.Requests[0].SortOrder != 0 {
		t.Fatalf("first request: %+v", col.Requests[0])
	}
	if col.Requests[1].Name != "Request Last" || col.Requests[1].SortOrder != 3 {
		t.Fatalf("last request: %+v", col.Requests[1])
	}
	if col.Children[0].Name != "Folder B" || col.Children[0].SortOrder != 1 {
		t.Fatalf("folder B: %+v", col.Children[0])
	}
	if col.Children[1].Name != "Empty Folder" || col.Children[1].SortOrder != 2 {
		t.Fatalf("empty folder: %+v", col.Children[1])
	}
	if len(col.Children[0].Requests) != 1 {
		t.Fatalf("nested requests: %d", len(col.Children[0].Requests))
	}
}

func TestOpenAPIImport(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "openapi", "petstore.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	col, err := openapiimport.Parse(data, "")
	if err != nil {
		t.Fatal(err)
	}
	if col.Name != "Petstore" {
		t.Fatalf("name=%q", col.Name)
	}
	if len(col.Variables.PreRequest) != 1 || col.Variables.PreRequest[0].Name != "baseUrl" {
		t.Fatalf("baseUrl var: %+v", col.Variables.PreRequest)
	}
	if len(col.Requests) < 3 {
		t.Fatalf("expected 3+ requests, got %d", len(col.Requests))
	}
	found := false
	for _, r := range col.Requests {
		if r.Method == "GET" && r.Name == "listPets" {
			found = true
			if len(r.Params) != 1 || r.Params[0].Key != "limit" {
				t.Fatalf("params: %+v", r.Params)
			}
		}
	}
	if !found {
		t.Fatal("listPets not found")
	}
}
