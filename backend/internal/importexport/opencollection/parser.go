package opencollection

import (
	"encoding/json"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/imaanmzr/postchi/backend/internal/importexport/model"
	"github.com/imaanmzr/postchi/backend/internal/request"
	"github.com/imaanmzr/postchi/backend/internal/shared/domain"
)

type rootDoc struct {
	OpenCollection string      `yaml:"opencollection" json:"opencollection"`
	Info           infoBlock   `yaml:"info" json:"info"`
	Items          []itemBlock `yaml:"items" json:"items"`
	Request        *metaBlock  `yaml:"request" json:"request"`
}

type infoBlock struct {
	Name string `yaml:"name" json:"name"`
}

type itemBlock struct {
	Info    itemInfo    `yaml:"info" json:"info"`
	HTTP    *httpBlock  `yaml:"http" json:"http"`
	Items   []itemBlock `yaml:"items" json:"items"`
	Request *metaBlock  `yaml:"request" json:"request"`
}

type itemInfo struct {
	Name string `yaml:"name" json:"name"`
	Type string `yaml:"type" json:"type"`
	Seq  int    `yaml:"seq" json:"seq"`
}

type httpBlock struct {
	Method  string      `yaml:"method" json:"method"`
	URL     string      `yaml:"url" json:"url"`
	Body    *bodyBlock  `yaml:"body" json:"body"`
	Auth    any         `yaml:"auth" json:"auth"`
	Params  []paramItem `yaml:"params" json:"params"`
	Headers []kvItem    `yaml:"headers" json:"headers"`
}

type bodyBlock struct {
	Type string `yaml:"type" json:"type"`
	Data any    `yaml:"data" json:"data"`
}

type paramItem struct {
	Name    string `yaml:"name" json:"name"`
	Value   string `yaml:"value" json:"value"`
	Type    string `yaml:"type" json:"type"`
	Enabled *bool  `yaml:"enabled" json:"enabled"`
}

type kvItem struct {
	Name    string `yaml:"name" json:"name"`
	Value   string `yaml:"value" json:"value"`
	Enabled *bool  `yaml:"enabled" json:"enabled"`
}

type metaBlock struct {
	Auth      any     `yaml:"auth" json:"auth"`
	Variables []kvItem `yaml:"variables" json:"variables"`
}

type authBlock struct {
	Type     string `yaml:"type" json:"type"`
	Token    string `yaml:"token" json:"token"`
	Username string `yaml:"username" json:"username"`
	Password string `yaml:"password" json:"password"`
}

func IsOpenCollection(data []byte) bool {
	trimmed := strings.TrimSpace(string(data))
	if strings.HasPrefix(trimmed, "{") {
		var probe struct {
			OpenCollection string `json:"opencollection"`
		}
		if json.Unmarshal(data, &probe) == nil && probe.OpenCollection != "" {
			return true
		}
		return false
	}
	var probe struct {
		OpenCollection string `yaml:"opencollection"`
	}
	if yaml.Unmarshal(data, &probe) == nil && probe.OpenCollection != "" {
		return true
	}
	return false
}

func Parse(data []byte) (model.Collection, error) {
	var doc rootDoc
	trimmed := strings.TrimSpace(string(data))
	if strings.HasPrefix(trimmed, "{") {
		if err := json.Unmarshal(data, &doc); err != nil {
			return model.Collection{}, err
		}
	} else if err := yaml.Unmarshal(data, &doc); err != nil {
		return model.Collection{}, err
	}
	col := model.Collection{
		Name:      doc.Info.Name,
		Variables: domain.EmptyVariablesSpec(),
		Auth:      request.AuthSpec{Type: "none"},
	}
	if doc.Request != nil {
		col.Auth = mapAuthValue(doc.Request.Auth)
		for _, v := range doc.Request.Variables {
			col.Variables.PreRequest = append(col.Variables.PreRequest, domain.PreRequestVar{
				Enabled: enabled(v.Enabled), Name: v.Name, Value: v.Value, Type: "string",
			})
		}
	}
	convertItems(doc.Items, &col, col.Auth)
	if col.Name == "" {
		col.Name = "Imported OpenCollection"
	}
	return col, nil
}

func convertItems(items []itemBlock, parent *model.Collection, inherited request.AuthSpec) {
	for _, item := range items {
		folderAuth := inherited
		if item.Request != nil {
			if mapped := mapAuthValue(item.Request.Auth); mapped.Type != "inherit" && mapped.Type != "none" {
				folderAuth = mapped
			}
		}
		if item.Info.Type == "folder" || (item.Info.Type == "" && len(item.Items) > 0) {
			child := model.Collection{
				Name:      item.Info.Name,
				Variables: domain.EmptyVariablesSpec(),
				Auth:      folderAuth,
			}
			convertItems(item.Items, &child, folderAuth)
			parent.Children = append(parent.Children, child)
			continue
		}
		if item.HTTP == nil || item.HTTP.Method == "" {
			if len(item.Items) > 0 {
				child := model.Collection{Name: item.Info.Name, Variables: domain.EmptyVariablesSpec(), Auth: folderAuth}
				convertItems(item.Items, &child, folderAuth)
				parent.Children = append(parent.Children, child)
			}
			continue
		}
		reqAuth := folderAuth
		if mapped := mapHTTPAuth(item.HTTP.Auth, folderAuth); mapped.Type != "inherit" {
			reqAuth = mapped
		}
		req := model.Request{
			Name:      item.Info.Name,
			Method:    item.HTTP.Method,
			URL:       item.HTTP.URL,
			SortOrder: item.Info.Seq,
			Auth:      reqAuth,
			Body:      request.BodySpec{Mode: "none"},
		}
		for _, h := range item.HTTP.Headers {
			req.Headers = append(req.Headers, request.KVPair{Key: h.Name, Value: h.Value, Enabled: enabled(h.Enabled)})
		}
		for _, p := range item.HTTP.Params {
			kv := request.KVPair{Key: p.Name, Value: p.Value, Enabled: enabled(p.Enabled)}
			switch strings.ToLower(p.Type) {
			case "path":
				req.PathVars = append(req.PathVars, kv)
			default:
				req.Params = append(req.Params, kv)
			}
		}
		if item.HTTP.Body != nil {
			req.Body = mapBody(item.HTTP.Body)
		}
		parent.Requests = append(parent.Requests, req)
	}
}

func mapBody(b *bodyBlock) request.BodySpec {
	mode := strings.ToLower(b.Type)
	switch mode {
	case "json":
		return request.BodySpec{Mode: "raw", Raw: asString(b.Data), RawLang: "json"}
	case "text", "xml", "html":
		return request.BodySpec{Mode: "raw", Raw: asString(b.Data), RawLang: mode}
	case "form", "form-urlencoded":
		return request.BodySpec{Mode: "form", Raw: asString(b.Data)}
	case "multipart", "multipart-form":
		return request.BodySpec{Mode: "multipart", Raw: asJSON(b.Data)}
	default:
		raw := asString(b.Data)
		if raw == "" && b.Data == nil {
			return request.BodySpec{Mode: "none"}
		}
		return request.BodySpec{Mode: "raw", Raw: raw, RawLang: b.Type}
	}
}

func asString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case nil:
		return ""
	default:
		return asJSON(v)
	}
}

func asJSON(v any) string {
	if v == nil {
		return ""
	}
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(b)
}

func mapHTTPAuth(auth any, inherited request.AuthSpec) request.AuthSpec {
	if auth == nil {
		return inherited
	}
	if s, ok := auth.(string); ok && s == "inherit" {
		return inherited
	}
	mapped := mapAuthValue(auth)
	if mapped.Type == "inherit" || mapped.Type == "none" {
		return inherited
	}
	return mapped
}

func mapAuthValue(auth any) request.AuthSpec {
	if auth == nil {
		return request.AuthSpec{Type: "none"}
	}
	if s, ok := auth.(string); ok {
		if s == "inherit" {
			return request.AuthSpec{Type: "inherit"}
		}
		return request.AuthSpec{Type: "none"}
	}
	b, err := json.Marshal(auth)
	if err != nil {
		return request.AuthSpec{Type: "none"}
	}
	var block authBlock
	if err := json.Unmarshal(b, &block); err != nil {
		return request.AuthSpec{Type: "none"}
	}
	return mapAuth(&block)
}

func mapAuth(a *authBlock) request.AuthSpec {
	if a == nil || a.Type == "" || a.Type == "none" || a.Type == "inherit" {
		return request.AuthSpec{Type: "none"}
	}
	out := request.AuthSpec{Type: a.Type, Config: map[string]any{}}
	switch a.Type {
	case "bearer":
		out.Config["token"] = a.Token
	case "basic":
		out.Config["username"] = a.Username
		out.Config["password"] = a.Password
	}
	return out
}

func enabled(v *bool) bool {
	if v == nil {
		return true
	}
	return *v
}
