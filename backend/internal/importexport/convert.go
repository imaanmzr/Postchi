package importexport

import (
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
	if b.Body != "" {
		req.Body = request.BodySpec{Mode: "raw", Raw: b.Body, RawLang: "json"}
	}
	if b.AuthType == "bearer" {
		req.Auth = request.AuthSpec{Type: "bearer", Config: map[string]any{"token": b.AuthToken}}
	}
	for _, h := range b.Headers {
		req.Headers = append(req.Headers, request.KVPair{Key: h.Key, Value: h.Value, Enabled: true})
	}
	return req
}

func bruFromNorm(req model.Request) bruno.BruRequest {
	b := bruno.BruRequest{
		Name: req.Name, Method: req.Method, URL: req.URL,
		PreRequestScript: req.PreRequestScript, TestScript: req.TestScript,
	}
	if req.Body.Mode == "raw" {
		b.Body = req.Body.Raw
	}
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
