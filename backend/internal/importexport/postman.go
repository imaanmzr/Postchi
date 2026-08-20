package importexport

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/imaanmzr/postchi/backend/internal/importexport/model"
	"github.com/imaanmzr/postchi/backend/internal/request"
	"github.com/imaanmzr/postchi/backend/internal/shared/domain"
)

type postmanCollection struct {
	Info struct {
		Name   string `json:"name"`
		Schema string `json:"schema"`
	} `json:"info"`
	Item     []postmanItem  `json:"item"`
	Auth     *postmanAuth   `json:"auth,omitempty"`
	Variable []postmanVar   `json:"variable,omitempty"`
	Event    []postmanEvent `json:"event,omitempty"`
}

type postmanItem struct {
	Name    string          `json:"name"`
	Item    []postmanItem   `json:"item,omitempty"`
	Request *postmanRequest `json:"request,omitempty"`
	Event   []postmanEvent  `json:"event,omitempty"`
	Auth    *postmanAuth    `json:"auth,omitempty"`
}

type postmanRequest struct {
	Method      string          `json:"method"`
	Header      []postmanHeader `json:"header,omitempty"`
	URL         any             `json:"url"`
	Body        *postmanBody    `json:"body,omitempty"`
	Auth        *postmanAuth    `json:"auth,omitempty"`
	Description string          `json:"description,omitempty"`
}

type postmanHeader struct {
	Key      string `json:"key"`
	Value    string `json:"value"`
	Disabled bool   `json:"disabled"`
}

type postmanFormField struct {
	Key      string `json:"key"`
	Value    string `json:"value"`
	Type     string `json:"type"`
	Disabled bool   `json:"disabled"`
}

type postmanBody struct {
	Mode       string             `json:"mode"`
	Raw        string             `json:"raw"`
	Formdata   []postmanFormField `json:"formdata,omitempty"`
	Urlencoded []postmanFormField `json:"urlencoded,omitempty"`
	Options    *struct {
		Raw map[string]string `json:"raw"`
	} `json:"options,omitempty"`
}

type postmanAuth struct {
	Type   string          `json:"type"`
	Bearer json.RawMessage `json:"bearer,omitempty"`
	Basic  json.RawMessage `json:"basic,omitempty"`
	APIKey json.RawMessage `json:"apikey,omitempty"`
}

type postmanVar struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type postmanEvent struct {
	Listen string `json:"listen"`
	Script struct {
		Exec []string `json:"exec"`
	} `json:"script"`
}

type PostmanParseResult struct {
	Collection model.Collection
	Warnings   []string
}

func ParsePostman(data []byte) (model.Collection, error) {
	result, err := ParsePostmanWithWarnings(data)
	if err != nil {
		return model.Collection{}, err
	}
	return result.Collection, nil
}

func ParsePostmanWithWarnings(data []byte) (PostmanParseResult, error) {
	var col postmanCollection
	if err := json.Unmarshal(data, &col); err != nil {
		return PostmanParseResult{}, err
	}
	out := model.Collection{Name: col.Info.Name, Variables: domain.EmptyVariablesSpec()}
	for _, v := range col.Variable {
		out.Variables.PreRequest = append(out.Variables.PreRequest, domain.PreRequestVar{
			Enabled: true, Name: v.Key, Value: v.Value, Type: "string",
		})
	}
	for _, ev := range col.Event {
		script := joinLines(ev.Script.Exec)
		if ev.Listen == "prerequest" {
			out.PreRequestScript = script
		} else if ev.Listen == "test" {
			out.TestScript = script
		}
	}
	out.Auth = mapPostmanAuth(col.Auth)
	var warnings []string
	convertItems(col.Item, &out, &warnings)
	return PostmanParseResult{Collection: out, Warnings: warnings}, nil
}

func convertItems(items []postmanItem, parent *model.Collection, warnings *[]string) {
	for i, item := range items {
		if item.Request == nil {
			child := model.Collection{
				Name:      item.Name,
				SortOrder: i,
				Variables: domain.EmptyVariablesSpec(),
			}
			child.Auth = mapPostmanAuth(item.Auth)
			for _, ev := range item.Event {
				script := joinLines(ev.Script.Exec)
				if ev.Listen == "prerequest" {
					child.PreRequestScript = script
				} else if ev.Listen == "test" {
					child.TestScript = script
				}
			}
			if len(item.Item) > 0 {
				convertItems(item.Item, &child, warnings)
			}
			parent.Children = append(parent.Children, child)
			continue
		}
		if item.Request.Method == "" {
			*warnings = append(*warnings, fmt.Sprintf("skipped item %q: missing HTTP method", item.Name))
			continue
		}
		url, urlParams := parsePostmanURL(item.Request.URL)
		req := model.Request{
			Name:        item.Name,
			Method:      strings.ToUpper(item.Request.Method),
			URL:         url,
			Params:      urlParams,
			Description: item.Request.Description,
			SortOrder:   i,
			Body:        request.BodySpec{Mode: "none"},
		}
		if item.Request.Auth != nil {
			req.Auth = mapPostmanAuth(item.Request.Auth)
		} else {
			req.Auth = request.AuthSpec{Type: "inherit"}
		}
		for _, h := range item.Request.Header {
			req.Headers = append(req.Headers, request.KVPair{Key: h.Key, Value: h.Value, Enabled: !h.Disabled})
		}
		if item.Request.Body != nil {
			req.Body = mapPostmanBody(item.Request.Body)
		}
		for _, ev := range item.Event {
			script := joinLines(ev.Script.Exec)
			if ev.Listen == "prerequest" {
				req.PreRequestScript = script
			} else if ev.Listen == "test" {
				req.TestScript = script
			}
		}
		parent.Requests = append(parent.Requests, req)
	}
}

func parsePostmanURL(u any) (string, []request.KVPair) {
	switch v := u.(type) {
	case string:
		return v, nil
	case map[string]any:
		if raw, ok := v["raw"].(string); ok && strings.TrimSpace(raw) != "" {
			return raw, queryParamsFromURLObject(v)
		}
		return buildURLFromObject(v), queryParamsFromURLObject(v)
	}
	return "", nil
}

func queryParamsFromURLObject(m map[string]any) []request.KVPair {
	raw, _ := m["raw"].(string)
	if raw != "" && strings.Contains(raw, "?") {
		return nil
	}
	q, ok := m["query"].([]any)
	if !ok {
		return nil
	}
	var params []request.KVPair
	for _, item := range q {
		kv, ok := item.(map[string]any)
		if !ok {
			continue
		}
		key, _ := kv["key"].(string)
		if key == "" {
			continue
		}
		disabled, _ := kv["disabled"].(bool)
		val, _ := kv["value"].(string)
		params = append(params, request.KVPair{Key: key, Value: val, Enabled: !disabled})
	}
	return params
}

func buildURLFromObject(m map[string]any) string {
	protocol, _ := m["protocol"].(string)
	host := joinURLSegments(m["host"], ".")
	path := joinURLSegments(m["path"], "/")
	if path != "" && !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	port := portString(m["port"])
	var b strings.Builder
	if protocol != "" {
		b.WriteString(protocol)
		b.WriteString("://")
	}
	b.WriteString(host)
	if port != "" && !strings.Contains(host, ":") {
		b.WriteString(":")
		b.WriteString(port)
	}
	b.WriteString(path)
	return b.String()
}

func portString(v any) string {
	switch p := v.(type) {
	case string:
		return strings.TrimSpace(p)
	case float64:
		if p > 0 {
			return fmt.Sprintf("%d", int(p))
		}
	case json.Number:
		if s := strings.TrimSpace(p.String()); s != "" && s != "0" {
			return s
		}
	}
	return ""
}

func joinURLSegments(v any, sep string) string {
	switch t := v.(type) {
	case string:
		return t
	case []any:
		parts := make([]string, 0, len(t))
		for _, p := range t {
			if s, ok := p.(string); ok && s != "" {
				parts = append(parts, s)
			}
		}
		return strings.Join(parts, sep)
	case []string:
		return strings.Join(t, sep)
	}
	return ""
}

func mapPostmanBody(b *postmanBody) request.BodySpec {
	if b == nil {
		return request.BodySpec{Mode: "none"}
	}
	switch b.Mode {
	case "raw", "json", "xml", "html", "javascript", "text":
		lang := b.Mode
		if lang == "raw" && b.Options != nil {
			if l, ok := b.Options.Raw["language"]; ok && l != "" {
				lang = l
			} else {
				lang = "json"
			}
		}
		if lang == "raw" {
			lang = "json"
		}
		return request.BodySpec{Mode: "raw", Raw: b.Raw, RawLang: lang}
	case "formdata":
		return request.BodySpec{Mode: "form-data", FormData: mapPostmanFormFields(b.Formdata)}
	case "urlencoded":
		return request.BodySpec{Mode: "urlencoded", URLEncoded: mapPostmanKVPairs(b.Urlencoded)}
	case "graphql":
		return request.BodySpec{Mode: "raw", Raw: b.Raw, RawLang: "json"}
	default:
		if b.Raw != "" {
			return request.BodySpec{Mode: "raw", Raw: b.Raw, RawLang: "json"}
		}
		return request.BodySpec{Mode: "none"}
	}
}

func mapPostmanKVPairs(fields []postmanFormField) []request.KVPair {
	out := make([]request.KVPair, 0, len(fields))
	for _, f := range fields {
		out = append(out, request.KVPair{
			Key: f.Key, Value: f.Value, Enabled: !f.Disabled,
		})
	}
	return out
}

func mapPostmanFormFields(fields []postmanFormField) []request.FormField {
	out := make([]request.FormField, 0, len(fields))
	for _, f := range fields {
		out = append(out, request.FormField{
			Key: f.Key, Value: f.Value, Enabled: !f.Disabled, Type: f.Type,
		})
	}
	return out
}

func mapPostmanAuth(a *postmanAuth) request.AuthSpec {
	if a == nil || a.Type == "" || a.Type == "noauth" {
		return request.AuthSpec{Type: "none"}
	}
	if a.Type == "inherit" {
		return request.AuthSpec{Type: "inherit"}
	}
	out := request.AuthSpec{Type: a.Type, Config: map[string]any{}}
	switch a.Type {
	case "bearer":
		applyPostmanAuthFields(&out, a.Bearer)
	case "basic":
		applyPostmanAuthFields(&out, a.Basic)
	case "apikey":
		applyPostmanAuthFields(&out, a.APIKey)
	}
	return out
}

func applyPostmanAuthFields(out *request.AuthSpec, raw json.RawMessage) {
	if len(raw) == 0 {
		return
	}
	trim := strings.TrimSpace(string(raw))
	if strings.HasPrefix(trim, "[") {
		var items []map[string]string
		if json.Unmarshal(raw, &items) == nil {
			for _, kv := range items {
				out.Config[kv["key"]] = kv["value"]
			}
		}
		return
	}
	var obj map[string]string
	if json.Unmarshal(raw, &obj) == nil {
		if key, value := obj["key"], obj["value"]; key != "" {
			out.Config[key] = value
			return
		}
		for k, v := range obj {
			out.Config[k] = v
		}
	}
}

func joinLines(lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	out := lines[0]
	for i := 1; i < len(lines); i++ {
		out += "\n" + lines[i]
	}
	return out
}

func ExportPostmanCollection(col model.Collection) map[string]any {
	return map[string]any{
		"info": map[string]any{
			"name":   col.Name,
			"schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json",
		},
		"item": exportPostmanItems(col),
	}
}

func exportPostmanItems(col model.Collection) []any {
	var items []any
	for _, child := range col.Children {
		items = append(items, map[string]any{
			"name": child.Name,
			"item": exportPostmanItems(child),
		})
	}
	for _, req := range col.Requests {
		headers := []any{}
		for _, h := range req.Headers {
			headers = append(headers, map[string]any{"key": h.Key, "value": h.Value, "disabled": !h.Enabled})
		}
		rmap := map[string]any{
			"method": req.Method,
			"header": headers,
			"url":    req.URL,
		}
		if req.Body.Mode != "" && req.Body.Mode != "none" {
			rmap["body"] = map[string]any{"mode": req.Body.Mode, "raw": req.Body.Raw}
		}
		item := map[string]any{"name": req.Name, "request": rmap}
		if req.PreRequestScript != "" || req.TestScript != "" {
			events := []any{}
			if req.PreRequestScript != "" {
				events = append(events, map[string]any{"listen": "prerequest", "script": map[string]any{"exec": []string{req.PreRequestScript}}})
			}
			if req.TestScript != "" {
				events = append(events, map[string]any{"listen": "test", "script": map[string]any{"exec": []string{req.TestScript}}})
			}
			item["event"] = events
		}
		items = append(items, item)
	}
	return items
}
