package validate

import (
	"testing"

	"github.com/imaanmzr/postchi/backend/internal/importexport/bruno"
	"github.com/imaanmzr/postchi/backend/internal/importexport/model"
)

func TestCollectionRequiresRequests(t *testing.T) {
	if err := Collection(model.Collection{Name: "empty"}); err == nil {
		t.Fatal("expected error for empty collection")
	}
	col := model.Collection{
		Name:     "root",
		Requests: []model.Request{{Name: "r", Method: "GET", URL: "https://example.com"}},
	}
	if err := Collection(col); err != nil {
		t.Fatal(err)
	}
}

func TestBrunoClassification(t *testing.T) {
	request := bruno.Parse("meta {\n  name: Req\n  type: http\n}\n\nget {\n  url: https://example.com\n}\n")
	if !HasHTTPMethodBlock(request) {
		t.Fatal("expected HTTP method block")
	}
	meta := bruno.Parse("meta {\n  name: Collection\n  type: collection\n}\n")
	if !IsCollectionOrFolderMeta(meta) {
		t.Fatal("expected collection meta")
	}
}

func TestBrunoRequestValidation(t *testing.T) {
	parsed := bruno.Parse("meta {\n  name: Ping\n}\nget {\n  url: https://example.com\n}\n")
	if err := BrunoRequest(parsed); err != nil {
		t.Fatal(err)
	}
	broken := bruno.Parse("meta {\n  name: Broken\n}\n")
	if err := BrunoRequest(broken); err == nil {
		t.Fatal("expected validation error")
	}
}
