package importexport

import (
	"testing"

	"github.com/imaanmzr/postchi/backend/internal/importexport/bruno"
	"github.com/imaanmzr/postchi/backend/internal/importexport/model"
	"github.com/imaanmzr/postchi/backend/internal/request"
	"github.com/imaanmzr/postchi/backend/internal/shared/domain"
)

func TestBruBodyRoundTripAllModes(t *testing.T) {
	cases := []struct {
		name       string
		req        model.Request
		expectMode string
	}{
		{
			name: "urlencoded",
			req: model.Request{
				Name: "Form", Method: "POST", URL: "https://example.com",
				Body: request.BodySpec{Mode: "urlencoded", URLEncoded: []request.KVPair{{Key: "a", Value: "1", Enabled: true}}},
			},
		},
		{
			name: "multipart",
			req: model.Request{
				Name: "Upload", Method: "POST", URL: "https://example.com",
				Body: request.BodySpec{Mode: "form-data", FormData: []request.FormField{{Key: "file", Value: "x", Enabled: true, Type: "text"}}},
			},
		},
		{
			name: "graphql",
			req: model.Request{
				Name: "GQL", Method: "POST", URL: "https://example.com",
				Body: request.BodySpec{Mode: "graphql", GraphQL: &struct {
					Query     string `json:"query"`
					Variables string `json:"variables"`
				}{Query: "{ ping }", Variables: `{}`}},
			},
		},
		{
			name: "raw xml",
			req: model.Request{
				Name: "XML", Method: "POST", URL: "https://example.com",
				Body: request.BodySpec{Mode: "raw", Raw: "<ok/>", RawLang: "xml"},
			},
		},
		{
			name: "default raw fallback",
			req: model.Request{
				Name: "Other", Method: "POST", URL: "https://example.com",
				Body: request.BodySpec{Mode: "custom", Raw: `{"ok":true}`},
			},
			expectMode: "json",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b := bruFromNorm(tc.req)
			got := bruToNorm(b)
			wantMode := tc.expectMode
			if wantMode == "" {
				wantMode = tc.req.Body.Mode
			}
			if got.Body.Mode != wantMode {
				t.Fatalf("mode got %q want %q body=%+v", got.Body.Mode, wantMode, got.Body)
			}
		})
	}
}

func TestBruBodyToSpecBranches(t *testing.T) {
	b := bruno.BruRequest{
		Name: "All", Method: "POST", URL: "https://example.com",
		BodyType: "form-urlencoded", Body: "a: 1",
	}
	spec := bruBodyToSpec(b)
	if len(spec.URLEncoded) != 1 {
		t.Fatalf("urlencoded=%+v", spec.URLEncoded)
	}
	b.BodyType = "multipart-form"
	b.Body = "file: data"
	spec = bruBodyToSpec(b)
	if len(spec.FormData) != 1 {
		t.Fatalf("form=%+v", spec.FormData)
	}
	b.BodyType = "graphql"
	b.Body = "{ ping }"
	b.GraphQLVars = `{}`
	spec = bruBodyToSpec(b)
	if spec.GraphQL == nil {
		t.Fatal("expected graphql body")
	}
}

func TestBruBodyFromSpecNoneAndDisabledPairs(t *testing.T) {
	bodyType, content, vars := bruBodyFromSpec(request.BodySpec{Mode: "none"})
	if bodyType != "none" || content != "" || vars != "" {
		t.Fatalf("none: %s %s %s", bodyType, content, vars)
	}
	bodyType, content, _ = bruBodyFromSpec(request.BodySpec{
		Mode: "urlencoded",
		URLEncoded: []request.KVPair{
			{Key: "", Value: "skip", Enabled: true},
			{Key: "a", Value: "1", Enabled: false},
			{Key: "b", Value: "2", Enabled: true},
		},
	})
	if bodyType != "form-urlencoded" || content == "" {
		t.Fatalf("urlencoded export: %s %q", bodyType, content)
	}
}

func TestSpecToBruVarsDisabledSkipped(t *testing.T) {
	spec := domain.VariablesSpec{
		PreRequest:  []domain.PreRequestVar{{Enabled: false, Name: "x", Value: "y"}},
		PostResponse: []domain.PostResponseVar{{Enabled: false, Name: "z", Expr: "1"}},
	}
	v := specToBruVars(spec)
	if len(v.PreRequest) != 0 || len(v.PostResponse) != 0 {
		t.Fatalf("expected disabled vars skipped: %+v", v)
	}
}
