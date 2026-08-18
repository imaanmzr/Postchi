package importexport

import (
	"encoding/json"
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

type postmanBody struct {
	Mode string `json:"mode"`
	Raw  string `json:"raw"`
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

func ParsePostman(data []byte) (model.Collection, error) {
	var col postmanCollection
	if err := json.Unmarshal(data, &col); err != nil {
		return model.Collection{}, err
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
	convertItems(col.Item, &out)
	return out, nil
}

func convertItems(items []postmanItem, parent *model.Collection) {
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
				convertItems(item.Item, &child)
			}
			parent.Children = append(parent.Children, child)
			continue
		}
		if item.Request.Method == "" {
			continue
		}
		req := model.Request{
			Name:        item.Name,
			Method:      item.Request.Method,
			URL:         extractURL(item.Request.URL),
			Description: item.Request.Description,
			SortOrder:   i,
			Auth:        mapPostmanAuth(item.Request.Auth),
			Body:        request.BodySpec{Mode: "none"},
		}
		for _, h := range item.Request.Header {
			req.Headers = append(req.Headers, request.KVPair{Key: h.Key, Value: h.Value, Enabled: !h.Disabled})
		}
		if item.Request.Body != nil {
			req.Body = request.BodySpec{Mode: item.Request.Body.Mode, Raw: item.Request.Body.Raw, RawLang: "json"}
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

func extractURL(u any) string {
	switch v := u.(type) {
	case string:
		return v
	case map[string]any:
		if raw, ok := v["raw"].(string); ok {
			return raw
		}
	}
	return ""
}

func mapPostmanAuth(a *postmanAuth) request.AuthSpec {
	if a == nil || a.Type == "" || a.Type == "noauth" {
		return request.AuthSpec{Type: "none"}
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
