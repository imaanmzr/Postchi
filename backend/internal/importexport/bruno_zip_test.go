package importexport

import (
	"archive/zip"
	"bytes"
	"strings"
	"testing"

	"github.com/imaanmzr/postchi/backend/internal/importexport/model"
)

func TestParseBrunoZipFolderHierarchy(t *testing.T) {
	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)

	writeZipFile := func(name, content string) {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		_, err = w.Write([]byte(content))
		if err != nil {
			t.Fatal(err)
		}
	}

	writeZipFile("collection.bru", `meta {
  name: My API
  type: collection
}
`)
	writeZipFile("Invoices/folder.bru", `meta {
  name: Invoices
  seq: 2
}
`)
	writeZipFile("Invoices/Invoices/folder.bru", `meta {
  name: Invoices
  seq: 1
}
`)
	writeZipFile("Invoices/Invoices/list.bru", `meta {
  name: list invoices
  type: http
}

get {
  url: {{baseUrl}}/invoices
}
`)
	writeZipFile("Gateway.Api/folder.bru", `meta {
  name: Gateway.Api
  seq: 1
}
`)
	writeZipFile("Gateway.Api/UserGroups/folder.bru", `meta {
  name: UserGroups
  seq: 1
}
`)
	writeZipFile("Gateway.Api/UserGroups/search.bru", `meta {
  name: search-groups
  type: http
}

get {
  url: {{baseUrl}}/groups
}
`)

	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}

	col, err := parseBrunoZip(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if col.Name != "My API" {
		t.Fatalf("root name=%q", col.Name)
	}
	if len(col.Children) != 2 {
		t.Fatalf("root children=%d", len(col.Children))
	}

	gateway := findChild(col, "Gateway.Api")
	if gateway == nil {
		t.Fatal("missing Gateway.Api folder")
	}
	groups := findChild(*gateway, "UserGroups")
	if groups == nil {
		t.Fatal("missing UserGroups folder")
	}
	if len(groups.Requests) != 1 || groups.Requests[0].Name != "search-groups" {
		t.Fatalf("group requests=%+v", groups.Requests)
	}

	invoices := findChild(col, "Invoices")
	if invoices == nil {
		t.Fatal("missing Invoices folder")
	}
	if len(invoices.Children) != 1 {
		t.Fatalf("Invoices subfolders=%d", len(invoices.Children))
	}
	nested := invoices.Children[0]
	if nested.Name != "Invoices" {
		t.Fatalf("nested folder=%q", nested.Name)
	}
	if len(nested.Requests) != 1 || nested.Requests[0].Name != "list invoices" {
		t.Fatalf("nested requests=%+v", nested.Requests)
	}
}

func findChild(col model.Collection, name string) *model.Collection {
	for _, child := range col.Children {
		if child.Name == name {
			return &child
		}
	}
	return nil
}

func TestParseBrunoFilesAllowsCollectionWithoutMeta(t *testing.T) {
	files := []brunoSourceFile{
		{Path: "collection.bru", Content: []byte(`headers {
  accept: application/json
}

vars:pre-request {
  baseUrl: https://example.com
}
`)},
		{Path: "health.bru", Content: []byte(`meta {
  name: Health
  type: http
}

get {
  url: {{baseUrl}}/health
}
`)},
	}

	collection, err := parseBrunoFiles(files, brunoParseOptions{
		RootName:         "Imported API",
		RequireRootMeta:  true,
		ValidateRequests: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if collection.Name != "Imported API" {
		t.Fatalf("root name = %q", collection.Name)
	}
	if len(collection.Requests) != 1 || collection.Requests[0].Name != "Health" {
		t.Fatalf("requests = %+v", collection.Requests)
	}
}

func TestParseBrunoFilesForGit(t *testing.T) {
	files := []brunoSourceFile{
		{Path: "collection.bru", Content: []byte("meta {\n  name: Repository name\n  type: collection\n}\n")},
		{Path: "orders/folder.bru", Content: []byte("meta {\n  name: Orders\n  seq: 2\n}\n")},
		{Path: "orders/second.bru", Content: []byte("meta {\n  name: Second\n  type: http\n  seq: 2\n}\n\nget {\n  url: https://example.com/second\n}\n")},
		{Path: "orders/first.bru", Content: []byte("meta {\n  name: First\n  type: http\n  seq: 1\n}\n\nget {\n  url: https://example.com/first\n}\n")},
		{Path: "orders/readme.md", Content: []byte("ignored")},
		{Path: "orders/environments/local.bru", Content: []byte("not a request")},
	}

	collection, err := parseBrunoFiles(files, brunoParseOptions{
		RootName:         "Imported source",
		RequireRootMeta:  true,
		ValidateRequests: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if collection.Name != "Imported source" {
		t.Fatalf("root name = %q", collection.Name)
	}
	orders := findChild(collection, "Orders")
	if orders == nil || len(orders.Requests) != 2 {
		t.Fatalf("orders = %+v", orders)
	}
	if orders.SortOrder != 2 || orders.Requests[0].Name != "First" || orders.Requests[1].Name != "Second" {
		t.Fatalf("ordering was not preserved: %+v", orders)
	}
}

func TestParseBrunoFilesRequiresCollectionMetadata(t *testing.T) {
	_, err := parseBrunoFiles([]brunoSourceFile{{
		Path:    "request.bru",
		Content: []byte("meta {\n  name: Request\n}\n\nget {\n  url: https://example.com\n}\n"),
	}}, brunoParseOptions{RequireRootMeta: true, ValidateRequests: true})
	if err == nil || !strings.Contains(err.Error(), "collection.bru") {
		t.Fatalf("expected missing collection.bru error, got %v", err)
	}
}

func TestParseBrunoFilesRejectsMalformedRequest(t *testing.T) {
	_, err := parseBrunoFiles([]brunoSourceFile{
		{Path: "collection.bru", Content: []byte("meta {\n  name: API\n}\n")},
		{Path: "broken.bru", Content: []byte("meta {\n  name: Broken\n}\n")},
	}, brunoParseOptions{RequireRootMeta: true, ValidateRequests: true})
	if err == nil || !strings.Contains(err.Error(), "HTTP method block") {
		t.Fatalf("expected malformed request error, got %v", err)
	}
}
