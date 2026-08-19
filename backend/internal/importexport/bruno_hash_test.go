package importexport

import (
	"testing"

	"github.com/imaanmzr/postchi/backend/internal/importexport/model"
	"github.com/imaanmzr/postchi/backend/internal/request"
)

func TestBrunoRequestHashStable(t *testing.T) {
	req := model.Request{
		Name: "Create", Method: "POST", URL: "{{base}}/items",
		Body: request.BodySpec{Mode: "json", Raw: `{"name":"x"}`, RawLang: "json"},
	}
	h1 := brunoRequestHash(req)
	h2 := brunoRequestHash(req)
	if h1 == "" || h1 != h2 {
		t.Fatalf("hash=%q %q", h1, h2)
	}
	req.Body.Raw = `{"name":"y"}`
	if brunoRequestHash(req) == h1 {
		t.Fatal("expected hash to change when body changes")
	}
}

func TestCollectBrunoSourcePaths(t *testing.T) {
	tree := model.Collection{
		SourcePath: "collection.bru",
		Children: []model.Collection{{
			SourcePath: "orders/",
			Requests: []model.Request{{
				SourcePath: "orders/get.bru",
			}},
		}},
	}
	colPaths, reqPaths := collectBrunoSourcePaths(tree)
	if len(colPaths) != 2 || len(reqPaths) != 1 {
		t.Fatalf("paths=%v %v", colPaths, reqPaths)
	}
}
