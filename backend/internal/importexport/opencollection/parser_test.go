package opencollection

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseYAMLFixture(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "testdata", "n2.yml"))
	if err != nil {
		t.Fatal(err)
	}
	col, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if col.Name != "Acme API - Staging" {
		t.Fatalf("name = %q", col.Name)
	}
	if len(col.Children) == 0 {
		t.Fatal("expected nested folders")
	}
}

func TestParseJSONFixture(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "testdata", "opencollection", "collection.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !IsOpenCollection(data) {
		t.Fatal("expected open collection probe")
	}
	col, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(col.Requests) != 1 {
		t.Fatalf("requests = %d", len(col.Requests))
	}
}

func TestParseInvalid(t *testing.T) {
	_, err := Parse([]byte("not: valid: yaml: ["))
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestParseJSONWithBodiesAndAuth(t *testing.T) {
	data := []byte(`{
	  "opencollection": "1.0.0",
	  "info": {"name": "Rich"},
	  "items": [{
	    "info": {"name": "Create", "type": "http", "seq": 1},
	    "http": {
	      "method": "POST",
	      "url": "https://example.com/items",
	      "auth": {"type": "bearer", "token": "t"},
	      "body": {"type": "json", "data": {"ok": true}},
	      "params": [{"name": "id", "value": "1", "type": "path"}]
	    }
	  }]
	}`)
	col, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(col.Requests) != 1 || col.Requests[0].Body.Mode != "raw" {
		t.Fatalf("col = %+v", col.Requests)
	}
}
