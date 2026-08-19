package importexport

import (
	"strings"

	"github.com/imaanmzr/postchi/backend/internal/importexport/bruno"
	"github.com/imaanmzr/postchi/backend/internal/importexport/model"
	"github.com/imaanmzr/postchi/backend/internal/request"
	"github.com/imaanmzr/postchi/backend/internal/shared/domain"
)

func bruVarsToSpec(v bruno.BruVars) domain.VariablesSpec {
	spec := domain.EmptyVariablesSpec()
	for _, row := range v.PreRequest {
		spec.PreRequest = append(spec.PreRequest, domain.PreRequestVar{
			Enabled: true, Name: row.Key, Value: row.Value, Type: "string",
		})
	}
	for _, row := range v.PostResponse {
		spec.PostResponse = append(spec.PostResponse, domain.PostResponseVar{
			Enabled: true, Name: row.Key, Expr: row.Expr,
		})
	}
	return spec
}

func specToBruVars(spec domain.VariablesSpec) bruno.BruVars {
	v := bruno.BruVars{}
	for _, row := range spec.PreRequest {
		if row.Enabled {
			v.PreRequest = append(v.PreRequest, bruno.KV{Key: row.Name, Value: row.Value})
		}
	}
	for _, row := range spec.PostResponse {
		if row.Enabled {
			v.PostResponse = append(v.PostResponse, bruno.KVExpr{Key: row.Name, Expr: row.Expr})
		}
	}
	return v
}

func bruToNorm(b bruno.BruRequest) model.Request {
	req := model.Request{
		Name: b.Name, Method: b.Method, URL: b.URL,
		PreRequestScript: b.PreRequestScript, TestScript: b.TestScript,
		Body: request.BodySpec{Mode: "none"},
		Auth: request.AuthSpec{Type: "none"},
	}
	req.Body = bruBodyToSpec(b)
	if b.AuthType == "bearer" {
		req.Auth = request.AuthSpec{Type: "bearer", Config: map[string]any{"token": b.AuthToken}}
	}
	for _, h := range b.Headers {
		req.Headers = append(req.Headers, request.KVPair{Key: h.Key, Value: h.Value, Enabled: true})
	}
	return req
}

func bruBodyToSpec(b bruno.BruRequest) request.BodySpec {
	bodyType := strings.ToLower(strings.TrimSpace(b.BodyType))
	if (bodyType == "" || bodyType == "none") && b.Body == "" && b.GraphQLVars == "" {
		return request.BodySpec{Mode: "none"}
	}
	if bodyType == "" {
		bodyType = "json"
	}
	switch bodyType {
	case "json":
		return request.BodySpec{Mode: "json", Raw: b.Body, RawLang: "json"}
	case "text", "xml", "html", "sparql":
		return request.BodySpec{Mode: "raw", Raw: b.Body, RawLang: bodyType}
	case "form-urlencoded":
		return request.BodySpec{Mode: "urlencoded", URLEncoded: bruKVToPairs(b.Body)}
	case "multipart-form":
		return request.BodySpec{Mode: "form-data", FormData: bruFormFieldsFromBlock(b.Body)}
	case "graphql":
		spec := request.BodySpec{Mode: "graphql"}
		if b.Body != "" || b.GraphQLVars != "" {
			spec.GraphQL = &struct {
				Query     string `json:"query"`
				Variables string `json:"variables"`
			}{
				Query:     b.Body,
				Variables: b.GraphQLVars,
			}
		}
		return spec
	default:
		if b.Body != "" {
			return request.BodySpec{Mode: "raw", Raw: b.Body, RawLang: "json"}
		}
		return request.BodySpec{Mode: "none"}
	}
}

func bruKVToPairs(block string) []request.KVPair {
	var out []request.KVPair
	for _, kv := range bruno.ParseKVBlock(block) {
		out = append(out, request.KVPair{Key: kv.Key, Value: kv.Value, Enabled: true})
	}
	return out
}

func bruFormFieldsFromBlock(block string) []request.FormField {
	var out []request.FormField
	for _, kv := range bruno.ParseKVBlock(block) {
		out = append(out, request.FormField{
			Key: kv.Key, Value: kv.Value, Enabled: true, Type: "text",
		})
	}
	return out
}

func bruFromNorm(req model.Request) bruno.BruRequest {
	b := bruno.BruRequest{
		Name: req.Name, Method: req.Method, URL: req.URL,
		PreRequestScript: req.PreRequestScript, TestScript: req.TestScript,
	}
	b.BodyType, b.Body, b.GraphQLVars = bruBodyFromSpec(req.Body)
	if req.Auth.Type == "bearer" {
		b.AuthType = "bearer"
		if t, ok := req.Auth.Config["token"].(string); ok {
			b.AuthToken = t
		}
	}
	for _, h := range req.Headers {
		if h.Enabled {
			b.Headers = append(b.Headers, bruno.KV{Key: h.Key, Value: h.Value})
		}
	}
	return b
}

func bruBodyFromSpec(body request.BodySpec) (bodyType, content, graphqlVars string) {
	switch body.Mode {
	case "none", "":
		return "none", "", ""
	case "json":
		return "json", body.Raw, ""
	case "raw":
		lang := strings.ToLower(body.RawLang)
		if lang == "" {
			lang = "text"
		}
		return lang, body.Raw, ""
	case "urlencoded", "form":
		return "form-urlencoded", exportBrunoKVBlock(body.URLEncoded), ""
	case "form-data", "multipart":
		return "multipart-form", exportBrunoFormBlock(body.FormData), ""
	case "graphql":
		if body.GraphQL != nil {
			return "graphql", body.GraphQL.Query, body.GraphQL.Variables
		}
		return "graphql", "", ""
	default:
		if body.Raw != "" {
			return "json", body.Raw, ""
		}
		return "none", "", ""
	}
}

func exportBrunoKVBlock(pairs []request.KVPair) string {
	var lines []string
	for _, p := range pairs {
		if !p.Enabled || p.Key == "" {
			continue
		}
		lines = append(lines, "  "+p.Key+": "+p.Value)
	}
	return strings.Join(lines, "\n")
}

func exportBrunoFormBlock(fields []request.FormField) string {
	var lines []string
	for _, f := range fields {
		if !f.Enabled || f.Key == "" {
			continue
		}
		lines = append(lines, "  "+f.Key+": "+f.Value)
	}
	return strings.Join(lines, "\n")
}
