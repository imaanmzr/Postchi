package importexport

import (
	"archive/zip"
	"bytes"
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
