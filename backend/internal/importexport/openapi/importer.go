package openapi

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"

	"github.com/imaanmzr/postchi/backend/internal/importexport/model"
	"github.com/imaanmzr/postchi/backend/internal/request"
	"github.com/imaanmzr/postchi/backend/internal/shared/domain"
)

type Operation struct {
	Request     model.Request
	OperationID string
	OpHash      string
	ApiDoc      json.RawMessage
}

type ParseResult struct {
	Collection model.Collection
	SpecHash   string
	Operations []Operation
}

func ParseWithHash(data []byte, name string) (ParseResult, error) {
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromData(data)
	if err != nil {
		return ParseResult{}, err
	}
	specHash := sha256.Sum256(data)
	if name == "" {
		name = doc.Info.Title
		if name == "" {
			name = "Imported OpenAPI"
		}
	}
	col := model.Collection{Name: name, Variables: domain.EmptyVariablesSpec()}
	if len(doc.Servers) > 0 {
		col.Variables.PreRequest = append(col.Variables.PreRequest, domain.PreRequestVar{
			Enabled: true, Name: "baseUrl", Value: strings.TrimRight(doc.Servers[0].URL, "/"), Type: "string",
		})
	}
	var ops []Operation
	i := 0
	for path, item := range doc.Paths.Map() {
		for method, op := range item.Operations() {
			if op == nil {
				continue
			}
			req := model.Request{
				Name:      op.OperationID,
				Method:    strings.ToUpper(method),
				URL:       "{{baseUrl}}" + path,
				SortOrder: i,
				Body:      request.BodySpec{Mode: "none"},
				Auth:      request.AuthSpec{Type: "none"},
			}
			opID := op.OperationID
			if opID == "" {
				opID = method + " " + path
				req.Name = opID
			}
			if req.Name == "" {
				req.Name = method + " " + path
			}
			desc := op.Description
			if desc == "" {
				desc = op.Summary
			}
			if desc != "" {
				req.Description = desc
			}
			for _, p := range op.Parameters {
				if p.Value == nil {
					continue
				}
				pv := p.Value
				switch pv.In {
				case "query":
					req.Params = append(req.Params, request.KVPair{Key: pv.Name, Value: "", Enabled: true})
				case "header":
					req.Headers = append(req.Headers, request.KVPair{Key: pv.Name, Value: "", Enabled: true})
				case "path":
					req.PathVars = append(req.PathVars, request.KVPair{Key: pv.Name, Value: "", Enabled: true})
				}
			}
			if op.RequestBody != nil && op.RequestBody.Value != nil {
				req.Body = requestBodyFromSpec(op.RequestBody.Value)
			}
			apiDoc := extractApiDoc(op)
			apiDocBytes, _ := json.Marshal(apiDoc)
			hash := operationHash(method, path, op, apiDoc)
			ops = append(ops, Operation{Request: req, OperationID: opID, OpHash: hash, ApiDoc: apiDocBytes})
			col.Requests = append(col.Requests, req)
			i++
		}
	}
	return ParseResult{
		Collection: col,
		SpecHash:   hex.EncodeToString(specHash[:]),
		Operations: ops,
	}, nil
}

func requestBodyFromSpec(rb *openapi3.RequestBody) request.BodySpec {
	if rb == nil || len(rb.Content) == 0 {
		return request.BodySpec{Mode: "none"}
	}

	// Prefer JSON content types, then fall back to first media type.
	mediaTypes := sortedContentTypes(rb.Content)
	for _, mt := range mediaTypes {
		media := rb.Content[mt]
		if media == nil {
			continue
		}
		if strings.Contains(mt, "json") {
			body := request.BodySpec{Mode: "raw", RawLang: "json"}
			if raw := exampleFromMedia(media); raw != "" {
				body.Raw = raw
				return body
			}
			if media.Schema != nil && media.Schema.Value != nil {
				if sample, err := json.Marshal(sampleFromSchema(media.Schema.Value)); err == nil && string(sample) != "null" {
					body.Raw = string(sample)
					return body
				}
			}
			body.Raw = "{}"
			return body
		}
		if mt == "application/x-www-form-urlencoded" {
			return request.BodySpec{Mode: "urlencoded", Raw: formBodyFromSchema(media)}
		}
		if strings.HasPrefix(mt, "multipart/") {
			return request.BodySpec{Mode: "multipart", Raw: formBodyFromSchema(media)}
		}
	}

	return request.BodySpec{Mode: "none"}
}

func sortedContentTypes(content openapi3.Content) []string {
	types := make([]string, 0, len(content))
	for mt := range content {
		types = append(types, mt)
	}
	sort.Strings(types)
	return types
}

func exampleFromMedia(media *openapi3.MediaType) string {
	if media == nil {
		return ""
	}
	if media.Example != nil {
		if b, err := json.Marshal(media.Example); err == nil {
			return string(b)
		}
	}
	if media.Schema != nil && media.Schema.Value != nil && media.Schema.Value.Example != nil {
		if b, err := json.Marshal(media.Schema.Value.Example); err == nil {
			return string(b)
		}
	}
	for _, ex := range media.Examples {
		if ex != nil && ex.Value != nil && ex.Value.Value != nil {
			if b, err := json.Marshal(ex.Value.Value); err == nil {
				return string(b)
			}
		}
	}
	return ""
}

func formBodyFromSchema(media *openapi3.MediaType) string {
	if media == nil || media.Schema == nil || media.Schema.Value == nil {
		return ""
	}
	pairs := make([]string, 0)
	props := media.Schema.Value.Properties
	if len(props) == 0 {
		return ""
	}
	keys := make([]string, 0, len(props))
	for k := range props {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		ref := props[k]
		val := ""
		if ref != nil && ref.Value != nil {
			if s := sampleFromSchema(ref.Value); s != nil {
				val = fmt.Sprint(s)
			}
		}
		pairs = append(pairs, k+"="+val)
	}
	return strings.Join(pairs, "&")
}

func sampleFromSchema(s *openapi3.Schema) any {
	if s == nil {
		return nil
	}
	if s.Example != nil {
		return s.Example
	}
	if s.Default != nil {
		return s.Default
	}
	if len(s.Enum) > 0 {
		return s.Enum[0]
	}
	if len(s.AllOf) > 0 {
		merged := map[string]any{}
		for _, ref := range s.AllOf {
			if ref == nil || ref.Value == nil {
				continue
			}
			if obj, ok := sampleFromSchema(ref.Value).(map[string]any); ok {
				for k, v := range obj {
					merged[k] = v
				}
			}
		}
		if len(merged) > 0 {
			return merged
		}
	}
	if len(s.OneOf) > 0 && s.OneOf[0] != nil && s.OneOf[0].Value != nil {
		return sampleFromSchema(s.OneOf[0].Value)
	}
	if len(s.AnyOf) > 0 && s.AnyOf[0] != nil && s.AnyOf[0].Value != nil {
		return sampleFromSchema(s.AnyOf[0].Value)
	}
	if s.Type != nil && len(*s.Type) > 0 {
		switch (*s.Type)[0] {
		case "object":
			obj := map[string]any{}
			keys := make([]string, 0, len(s.Properties))
			for k := range s.Properties {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				ref := s.Properties[k]
				if ref == nil || ref.Value == nil {
					continue
				}
				obj[k] = sampleFromSchema(ref.Value)
			}
			return obj
		case "array":
			if s.Items != nil && s.Items.Value != nil {
				return []any{sampleFromSchema(s.Items.Value)}
			}
			return []any{}
		case "string":
			switch s.Format {
			case "uuid":
				return "00000000-0000-0000-0000-000000000000"
			case "email":
				return "user@example.com"
			case "date-time":
				return "2024-01-01T00:00:00Z"
			case "date":
				return "2024-01-01"
			case "password":
				return "password"
			default:
				return ""
			}
		case "integer":
			return 0
		case "number":
			return 0.0
		case "boolean":
			return false
		}
	}
	return nil
}

func extractApiDoc(op *openapi3.Operation) map[string]any {
	if op == nil {
		return map[string]any{}
	}
	doc := map[string]any{}
	if op.Summary != "" {
		doc["summary"] = op.Summary
	}
	if op.Description != "" {
		doc["description"] = op.Description
	}
	if len(op.Tags) > 0 {
		doc["tags"] = op.Tags
	}
	if op.Deprecated {
		doc["deprecated"] = true
	}
	if len(op.Parameters) > 0 {
		params := make([]map[string]any, 0, len(op.Parameters))
		for _, p := range op.Parameters {
			if p.Value == nil {
				continue
			}
			pv := p.Value
			entry := map[string]any{
				"name":     pv.Name,
				"in":       pv.In,
				"required": pv.Required,
			}
			if pv.Description != "" {
				entry["description"] = pv.Description
			}
			if pv.Schema != nil && pv.Schema.Value != nil {
				entry["schema"] = schemaRefHash(pv.Schema.Value)
			}
			params = append(params, entry)
		}
		doc["parameters"] = params
	}
	if op.RequestBody != nil && op.RequestBody.Value != nil {
		doc["requestBody"] = requestBodyDoc(op.RequestBody.Value)
	}
	if op.Responses != nil {
		responses := map[string]any{}
		for code, respRef := range op.Responses.Map() {
			if respRef.Value == nil {
				continue
			}
			responses[code] = responseDoc(respRef.Value)
		}
		if len(responses) > 0 {
			doc["responses"] = responses
		}
	}
	return doc
}

func requestBodyDoc(rb *openapi3.RequestBody) map[string]any {
	out := map[string]any{}
	if rb.Description != "" {
		out["description"] = rb.Description
	}
	if rb.Required {
		out["required"] = true
	}
	content := map[string]any{}
	for mt, media := range rb.Content {
		content[mt] = mediaDoc(media)
	}
	if len(content) > 0 {
		out["content"] = content
	}
	return out
}

func responseDoc(resp *openapi3.Response) map[string]any {
	out := map[string]any{}
	if resp.Description != nil && *resp.Description != "" {
		out["description"] = *resp.Description
	}
	content := map[string]any{}
	for mt, media := range resp.Content {
		content[mt] = mediaDoc(media)
	}
	if len(content) > 0 {
		out["content"] = content
	}
	return out
}

func mediaDoc(media *openapi3.MediaType) map[string]any {
	out := map[string]any{}
	if media.Schema != nil && media.Schema.Value != nil {
		out["schema"] = schemaRefHash(media.Schema.Value)
	}
	if media.Example != nil {
		out["example"] = media.Example
	}
	if len(media.Examples) > 0 {
		examples := map[string]any{}
		for k, ex := range media.Examples {
			if ex.Value != nil && ex.Value.Value != nil {
				examples[k] = ex.Value.Value
			}
		}
		if len(examples) > 0 {
			out["examples"] = examples
		}
	}
	return out
}

func operationHash(method, path string, op *openapi3.Operation, apiDoc map[string]any) string {
	canon := map[string]any{
		"method": strings.ToUpper(method),
		"path":   path,
	}
	if op != nil {
		if op.Summary != "" {
			canon["summary"] = op.Summary
		}
		if op.Description != "" {
			canon["description"] = op.Description
		}
	}
	if apiDoc != nil {
		canon["api_doc"] = apiDoc
	}
	b, _ := json.Marshal(canon)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func schemaRefHash(s *openapi3.Schema) map[string]any {
	if s == nil {
		return nil
	}
	out := map[string]any{}
	if s.Type != nil && len(*s.Type) > 0 {
		out["type"] = (*s.Type)[0]
	}
	if s.Format != "" {
		out["format"] = s.Format
	}
	if len(s.Required) > 0 {
		req := append([]string(nil), s.Required...)
		sort.Strings(req)
		out["required"] = req
	}
	if s.Properties != nil {
		keys := make([]string, 0, len(s.Properties))
		for k := range s.Properties {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		props := map[string]any{}
		for _, k := range keys {
			ref := s.Properties[k]
			if ref != nil && ref.Value != nil {
				props[k] = schemaRefHash(ref.Value)
			}
		}
		out["properties"] = props
	}
	if s.Items != nil && s.Items.Value != nil {
		out["items"] = schemaRefHash(s.Items.Value)
	}
	if s.Example != nil {
		out["example"] = s.Example
	}
	if len(s.Enum) > 0 {
		out["enum"] = s.Enum
	}
	return out
}

// Parse keeps backward compatibility for one-shot import.
func Parse(data []byte, name string) (model.Collection, error) {
	res, err := ParseWithHash(data, name)
	if err != nil {
		return model.Collection{}, err
	}
	return res.Collection, nil
}
