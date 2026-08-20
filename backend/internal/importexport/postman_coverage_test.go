package importexport

import (
	"encoding/json"
	"testing"

	"github.com/imaanmzr/postchi/backend/internal/request"
)

func TestPostmanURLObjectAndAuthBodies(t *testing.T) {
	raw := `{
	  "info": {"name": "Rich", "schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"},
	  "auth": {"type": "bearer", "bearer": [{"key":"token","value":"t","type":"string"}]},
	  "item": [{
	    "name": "Create",
	    "request": {
	      "method": "POST",
	      "url": {"raw":"https://api.example.com:8080/v1/items","host":["api","example","com"],"port":"8080","path":["v1","items"],"query":[{"key":"q","value":"1","disabled":false}]},
	      "auth": {"type": "basic", "basic": [{"key":"username","value":"u","type":"string"},{"key":"password","value":"p","type":"string"}]},
	      "body": {
	        "mode": "formdata",
	        "formdata": [{"key":"file","value":"data","type":"file","disabled":false}]
	      }
	    }
	  }]
	}`
	col, err := ParsePostman([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	if len(col.Requests) != 1 {
		t.Fatalf("requests=%d", len(col.Requests))
	}
	req := col.Requests[0]
	if req.Auth.Type != "basic" {
		t.Fatalf("auth=%+v", req.Auth)
	}
	if req.Body.Mode != "form-data" || len(req.Body.FormData) != 1 {
		t.Fatalf("body=%+v", req.Body)
	}
	if len(req.Params) != 1 {
		t.Fatalf("params=%+v", req.Params)
	}
}

func TestPostmanGraphQLAndURLEncodedBody(t *testing.T) {
	raw := `{
	  "info": {"name": "Bodies", "schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"},
	  "item": [
	    {"name":"gql","request":{"method":"POST","url":"https://example.com","body":{"mode":"graphql","raw":"{ ping }"}}},
	    {"name":"url","request":{"method":"POST","url":"https://example.com","body":{"mode":"urlencoded","urlencoded":[{"key":"a","value":"1","disabled":false}]}}}
	  ]
	}`
	col, err := ParsePostman([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	if len(col.Requests) != 2 {
		t.Fatal(col.Requests)
	}
}

func TestBuildURLFromObjectPortVariants(t *testing.T) {
	url, params := parsePostmanURL(map[string]any{
		"host": []any{"api.example.com"},
		"path": []any{"v1", "users"},
		"port": float64(443),
		"query": []any{map[string]any{"key": "x", "value": "y", "disabled": false}},
	})
	if url == "" || len(params) != 1 {
		t.Fatalf("url=%q params=%+v", url, params)
	}
}

func TestJoinLinesAndExportItems(t *testing.T) {
	if got := joinLines([]string{"a", "", "b"}); got != "a\n\nb" {
		t.Fatalf("joinLines=%q", got)
	}
	if got := joinLines(nil); got != "" {
		t.Fatalf("joinLines empty=%q", got)
	}
	col, _ := ParsePostman([]byte(`{"info":{"name":"X","schema":"https://schema.getpostman.com/json/collection/v2.1.0/collection.json"},"item":[{"name":"R","request":{"method":"GET","url":"https://example.com"}}]}`))
	exported := ExportPostmanCollection(col)
	b, _ := json.Marshal(exported)
	if len(b) == 0 {
		t.Fatal("empty export")
	}
}

func TestPostmanBodyModesAndAuthObject(t *testing.T) {
	raw := `{
	  "info": {"name": "Modes", "schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"},
	  "item": [
	    {"name":"raw-default","request":{"method":"POST","url":"https://example.com","body":{"mode":"raw","raw":"{}"}}},
	    {"name":"raw-lang","request":{"method":"POST","url":"https://example.com","body":{"mode":"raw","raw":"<x/>","options":{"raw":{"language":"xml"}}}}},
	    {"name":"fallback","request":{"method":"POST","url":"https://example.com","body":{"mode":"unknown","raw":"{}"}}},
	    {"name":"apikey","request":{"method":"GET","url":"https://example.com","auth":{"type":"apikey","apikey":{"key":"x-api-key","value":"secret"}}}}
	  ]
	}`
	col, err := ParsePostman([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	if len(col.Requests) != 4 {
		t.Fatalf("requests=%d", len(col.Requests))
	}
}

func TestPostmanURLStringAndQueryInRaw(t *testing.T) {
	url, params := parsePostmanURL("https://example.com?q=1")
	if url != "https://example.com?q=1" || params != nil {
		t.Fatalf("url=%q params=%+v", url, params)
	}
	url, params = parsePostmanURL(map[string]any{
		"raw": "https://example.com?x=1",
		"query": []any{map[string]any{"key": "ignored", "value": "1"}},
	})
	if url == "" || params != nil {
		t.Fatalf("raw query url=%q params=%+v", url, params)
	}
}

func TestPortStringAndJoinURLSegments(t *testing.T) {
	if got := portString(json.Number("8080")); got != "8080" {
		t.Fatalf("port=%q", got)
	}
	if got := portString(float64(0)); got != "" {
		t.Fatalf("zero port=%q", got)
	}
	if got := joinURLSegments([]any{"a", 1, "b"}, "/"); got != "a/b" {
		t.Fatalf("segments=%q", got)
	}
	if got := joinURLSegments("host", "."); got != "host" {
		t.Fatalf("string segment=%q", got)
	}
}

func TestApplyPostmanAuthFieldsObject(t *testing.T) {
	out := request.AuthSpec{Type: "apikey", Config: map[string]any{}}
	applyPostmanAuthFields(&out, json.RawMessage(`{"key":"token","value":"abc"}`))
	if out.Config["token"] != "abc" {
		t.Fatalf("config=%+v", out.Config)
	}
}
