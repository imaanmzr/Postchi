package importexport

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/imaanmzr/postchi/backend/internal/importexport/model"
)

func countImportTree(col model.Collection) (collections, requests int) {
	collections = 1
	requests = len(col.Requests)
	for _, c := range col.Children {
		cc, cr := countImportTree(c)
		collections += cc
		requests += cr
	}
	return
}

func TestPostmanShowcaseCollection(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "..", "docs", "postchi-showcase.postman.json"))
	if err != nil {
		t.Fatal(err)
	}
	col, err := ParsePostman(data)
	if err != nil {
		t.Fatal(err)
	}
	if col.Name != "Postchi Showcase" {
		t.Fatalf("name=%q", col.Name)
	}
	collections, requests := countImportTree(col)
	if collections != 10 {
		t.Fatalf("collections=%d want 10", collections)
	}
	if requests != 24 {
		t.Fatalf("requests=%d want 24", requests)
	}
	if len(col.Variables.PreRequest) < 6 {
		t.Fatalf("collection vars: %+v", col.Variables.PreRequest)
	}
	if col.PreRequestScript == "" || col.TestScript == "" {
		t.Fatal("collection scripts missing")
	}
	foundBearer := false
	foundInherit := false
	var walk func(c model.Collection)
	walk = func(c model.Collection) {
		for _, r := range c.Requests {
			if r.Name == "Bearer token" && r.Auth.Type == "bearer" {
				if r.Auth.Config["token"] != "{{demoToken}}" {
					t.Fatalf("bearer config: %+v", r.Auth.Config)
				}
				foundBearer = true
			}
			if r.Name == "Basic Auth (inherit)" && r.Auth.Type == "inherit" {
				foundInherit = true
			}
		}
		for _, child := range c.Children {
			walk(child)
		}
	}
	walk(col)
	if !foundBearer {
		t.Fatal("bearer request not found")
	}
	if !foundInherit {
		t.Fatal("inherit auth request not found")
	}
}

func TestPostmanURLObject(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "postman", "url_object.json"))
	if err != nil {
		t.Fatal(err)
	}
	col, err := ParsePostman(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(col.Requests) != 2 {
		t.Fatalf("requests=%d", len(col.Requests))
	}
	if col.Requests[0].URL != "https://api.example.com:443/v1/users" {
		t.Fatalf("built url=%q", col.Requests[0].URL)
	}
	if len(col.Requests[0].Params) != 2 {
		t.Fatalf("params=%+v", col.Requests[0].Params)
	}
	if col.Requests[0].Params[0].Key != "limit" || !col.Requests[0].Params[0].Enabled {
		t.Fatalf("limit param: %+v", col.Requests[0].Params[0])
	}
	if col.Requests[0].Params[1].Enabled {
		t.Fatal("disabled query param should be disabled")
	}
	if col.Requests[1].URL != "https://api.example.com/search?q=hello" {
		t.Fatalf("raw url=%q", col.Requests[1].URL)
	}
	if len(col.Requests[1].Params) != 0 {
		t.Fatalf("raw url should not duplicate query params: %+v", col.Requests[1].Params)
	}
}

func TestPostmanAuthAndBody(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "postman", "auth_and_body.json"))
	if err != nil {
		t.Fatal(err)
	}
	col, err := ParsePostman(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(col.Requests) != 3 {
		t.Fatalf("requests=%d", len(col.Requests))
	}
	if col.Requests[0].Auth.Type != "bearer" || col.Requests[0].Auth.Config["token"] != "{{token}}" {
		t.Fatalf("bearer: %+v", col.Requests[0].Auth)
	}
	if col.Requests[1].Auth.Type != "inherit" {
		t.Fatalf("inherit auth: %+v", col.Requests[1].Auth)
	}
	body := col.Requests[2].Body
	if body.Mode != "urlencoded" || len(body.URLEncoded) != 2 {
		t.Fatalf("urlencoded body: %+v", body)
	}
	if body.URLEncoded[1].Enabled {
		t.Fatal("disabled form field should be disabled")
	}
}
