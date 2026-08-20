package opencollection

import "testing"

func TestParseBodyAndAuthVariants(t *testing.T) {
	data := []byte(`{
	  "opencollection": "1.0.0",
	  "info": {"name": "Bodies"},
	  "request": {
	    "auth": "inherit",
	    "variables": [{"name": "baseUrl", "value": "https://example.com", "enabled": false}]
	  },
	  "items": [
	    {"info": {"name": "Text", "type": "http"}, "http": {"method": "POST", "url": "https://example.com", "body": {"type": "text", "data": "hello"}}},
	    {"info": {"name": "Form", "type": "http"}, "http": {"method": "POST", "url": "https://example.com", "body": {"type": "form-urlencoded", "data": "a=1"}}},
	    {"info": {"name": "Multipart", "type": "http"}, "http": {"method": "POST", "url": "https://example.com", "body": {"type": "multipart-form", "data": {"file": "x"}}}},
	    {"info": {"name": "Basic", "type": "http"}, "http": {"method": "GET", "url": "https://example.com", "auth": {"type": "basic", "username": "u", "password": "p"}}},
	    {"info": {"name": "Folder", "type": "folder"}, "items": [
	      {"info": {"name": "Nested", "type": "http"}, "http": {"method": "GET", "url": "https://example.com/nested", "auth": "inherit"}}
	    ]}
	  ]
	}`)
	col, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(col.Requests) != 4 || len(col.Children) != 1 {
		t.Fatalf("requests=%d children=%d", len(col.Requests), len(col.Children))
	}
	if col.Requests[0].Body.RawLang != "text" {
		t.Fatalf("text body=%+v", col.Requests[0].Body)
	}
	if col.Requests[3].Auth.Type != "basic" {
		t.Fatalf("auth=%+v", col.Requests[3].Auth)
	}
}

func TestIsOpenCollectionYAML(t *testing.T) {
	if !IsOpenCollection([]byte("opencollection: 1.0.0\ninfo:\n  name: x\n")) {
		t.Fatal("expected yaml probe true")
	}
	if IsOpenCollection([]byte("foo: bar\n")) {
		t.Fatal("expected yaml probe false")
	}
}
