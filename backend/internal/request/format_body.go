package request

import (
	"encoding/json"
	"strings"
)

// formatResponseBody mirrors frontend JSON prettify for response bodies.
func formatResponseBody(body string) string {
	trimmed := strings.TrimSpace(body)
	if trimmed == "" {
		return ""
	}
	if trimmed[0] != '{' && trimmed[0] != '[' {
		return body
	}
	var parsed any
	if err := json.Unmarshal([]byte(trimmed), &parsed); err != nil {
		return body
	}
	out, err := json.MarshalIndent(parsed, "", "  ")
	if err != nil {
		return body
	}
	return string(out)
}
