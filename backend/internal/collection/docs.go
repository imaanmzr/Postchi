package collection

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

type docEndpoint struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Method      string          `json:"method"`
	URL         string          `json:"url"`
	Description string          `json:"description"`
	ApiDoc      json.RawMessage `json:"api_doc"`
	SourceSpecID *string        `json:"source_spec_id,omitempty"`
}

type docExport struct {
	Collection struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
	} `json:"collection"`
	Endpoints []docEndpoint `json:"endpoints"`
}

func buildDocsMarkdown(name, desc string, endpoints []docEndpoint) string {
	var b strings.Builder
	b.WriteString("# " + name + "\n\n")
	if desc != "" {
		b.WriteString(desc + "\n\n")
	}
	b.WriteString("## Endpoints\n\n")
	for _, ep := range endpoints {
		b.WriteString("### " + ep.Name + "\n\n")
		b.WriteString("**" + ep.Method + "** `" + ep.URL + "`\n\n")
		if ep.Description != "" {
			b.WriteString(ep.Description + "\n\n")
		}
		if ep.SourceSpecID != nil && *ep.SourceSpecID != "" {
			b.WriteString("_Synced from OpenAPI spec_\n\n")
		}
		b.WriteString(formatApiDocMarkdown(ep.ApiDoc))
	}
	return b.String()
}

func formatApiDocMarkdown(apiDocRaw json.RawMessage) string {
	if len(apiDocRaw) == 0 {
		return ""
	}
	var doc map[string]any
	if err := json.Unmarshal(apiDocRaw, &doc); err != nil {
		return ""
	}
	var b strings.Builder
	if summary, ok := doc["summary"].(string); ok && summary != "" {
		b.WriteString("**Summary:** " + summary + "\n\n")
	}
	if tags, ok := doc["tags"].([]any); ok && len(tags) > 0 {
		parts := make([]string, 0, len(tags))
		for _, t := range tags {
			if s, ok := t.(string); ok {
				parts = append(parts, s)
			}
		}
		b.WriteString("**Tags:** " + strings.Join(parts, ", ") + "\n\n")
	}
	if params, ok := doc["parameters"].([]any); ok && len(params) > 0 {
		b.WriteString("#### Parameters\n\n")
		b.WriteString("| Name | In | Required | Description |\n")
		b.WriteString("|------|-----|----------|-------------|\n")
		for _, p := range params {
			pm, ok := p.(map[string]any)
			if !ok {
				continue
			}
			b.WriteString(fmt.Sprintf("| %s | %s | %v | %s |\n",
				strVal(pm, "name"), strVal(pm, "in"), pm["required"], strVal(pm, "description")))
		}
		b.WriteString("\n")
	}
	if rb, ok := doc["requestBody"].(map[string]any); ok {
		b.WriteString("#### Request Body\n\n")
		if d := strVal(rb, "description"); d != "" {
			b.WriteString(d + "\n\n")
		}
		b.WriteString(formatContentMarkdown(rb["content"]))
	}
	if responses, ok := doc["responses"].(map[string]any); ok && len(responses) > 0 {
		b.WriteString("#### Responses\n\n")
		codes := make([]string, 0, len(responses))
		for code := range responses {
			codes = append(codes, code)
		}
		sort.Strings(codes)
		b.WriteString("| Status | Description | Content |\n")
		b.WriteString("|--------|-------------|--------|\n")
		for _, code := range codes {
			rm, _ := responses[code].(map[string]any)
			desc := strVal(rm, "description")
			contentTypes := contentTypeSummary(rm["content"])
			b.WriteString(fmt.Sprintf("| %s | %s | %s |\n", code, desc, contentTypes))
		}
		b.WriteString("\n")
		for _, code := range codes {
			rm, _ := responses[code].(map[string]any)
			if content := rm["content"]; content != nil {
				b.WriteString(fmt.Sprintf("**%s response body:**\n\n", code))
				b.WriteString(formatContentMarkdown(content))
			}
		}
	}
	return b.String()
}

func formatContentMarkdown(content any) string {
	cm, ok := content.(map[string]any)
	if !ok {
		return ""
	}
	var b strings.Builder
	types := make([]string, 0, len(cm))
	for mt := range cm {
		types = append(types, mt)
	}
	sort.Strings(types)
	for _, mt := range types {
		media, _ := cm[mt].(map[string]any)
		b.WriteString("**" + mt + "**\n\n")
		if schema, ok := media["schema"]; ok {
			if sb, err := json.MarshalIndent(schema, "", "  "); err == nil {
				b.WriteString("```json\n" + string(sb) + "\n```\n\n")
			}
		}
		if ex, ok := media["example"]; ok {
			if eb, err := json.MarshalIndent(ex, "", "  "); err == nil {
				b.WriteString("Example:\n\n```json\n" + string(eb) + "\n```\n\n")
			}
		}
	}
	return b.String()
}

func contentTypeSummary(content any) string {
	cm, ok := content.(map[string]any)
	if !ok || len(cm) == 0 {
		return "-"
	}
	types := make([]string, 0, len(cm))
	for mt := range cm {
		types = append(types, mt)
	}
	sort.Strings(types)
	return strings.Join(types, ", ")
}

func strVal(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func isDocumented(ep docEndpoint) bool {
	if ep.Description != "" {
		return true
	}
	if len(ep.ApiDoc) == 0 {
		return false
	}
	var doc map[string]any
	if json.Unmarshal(ep.ApiDoc, &doc) != nil {
		return false
	}
	if responses, ok := doc["responses"].(map[string]any); ok && len(responses) > 0 {
		return true
	}
	return false
}
