package validate

import (
	"testing"

	"github.com/imaanmzr/postchi/backend/internal/importexport/bruno"
	"github.com/imaanmzr/postchi/backend/internal/importexport/model"
)

func TestCountRequestsNested(t *testing.T) {
	col := model.Collection{
		Children: []model.Collection{{
			Requests: []model.Request{{Name: "a"}, {Name: "b"}},
			Children: []model.Collection{{
				Requests: []model.Request{{Name: "c"}},
			}},
		}},
		Requests: []model.Request{{Name: "root"}},
	}
	if got := CountRequests(col); got != 4 {
		t.Fatalf("count=%d", got)
	}
}

func TestIsCollectionOrFolderMetaVariants(t *testing.T) {
	folder := bruno.Parse("meta {\n  name: Folder\n}\n")
	if !IsCollectionOrFolderMeta(folder) {
		t.Fatal("folder meta without type")
	}
	withMethod := bruno.Parse("meta {\n  name: Ping\n  type: http\n}\nget {\n  url: https://example.com\n}\n")
	if IsCollectionOrFolderMeta(withMethod) {
		t.Fatal("request with method block should not be folder meta")
	}
}

func TestBrunoRequestMissingMethod(t *testing.T) {
	parsed := bruno.Parse("meta {\n  name: Broken\n  type: http\n}\n")
	if err := BrunoRequest(parsed); err == nil {
		t.Fatal("expected missing method/url error")
	}
}
