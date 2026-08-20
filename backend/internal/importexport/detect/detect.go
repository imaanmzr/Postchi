package detect

import (
	"encoding/json"
	"strings"
)

type Format string

const (
	FormatBruno          Format = "bruno"
	FormatPostman        Format = "postman"
	FormatOpenCollection Format = "opencollection"
	FormatOpenAPI        Format = "openapi"
	FormatUnknown        Format = "unknown"
)

// DetectFormat identifies a collection file format from filename and content.
// YAML content is inspected before filename extension.
func DetectFormat(filename string, content []byte) Format {
	lower := strings.ToLower(strings.TrimSpace(filename))
	if strings.HasSuffix(lower, ".zip") || strings.HasSuffix(lower, ".bru") {
		return FormatBruno
	}

	trimmed := strings.TrimSpace(string(content))
	if trimmed == "" {
		return detectByExtension(lower)
	}

	if strings.HasPrefix(trimmed, "{") {
		var probe map[string]json.RawMessage
		if json.Unmarshal(content, &probe) == nil {
			if _, ok := probe["opencollection"]; ok {
				return FormatOpenCollection
			}
			if info, ok := probe["info"]; ok {
				var infoBlock struct {
					Schema string `json:"schema"`
				}
				if json.Unmarshal(info, &infoBlock) == nil &&
					strings.Contains(infoBlock.Schema, "postman.com/json/collection") {
					return FormatPostman
				}
			}
			if _, ok := probe["openapi"]; ok {
				return FormatOpenAPI
			}
			if _, ok := probe["swagger"]; ok {
				return FormatOpenAPI
			}
		}
	}

	if matchYAMLMarker(trimmed, "opencollection:") {
		return FormatOpenCollection
	}
	if matchYAMLMarker(trimmed, "openapi:") || matchYAMLMarker(trimmed, "swagger:") {
		return FormatOpenAPI
	}

	if strings.HasSuffix(lower, ".yml") || strings.HasSuffix(lower, ".yaml") {
		return FormatOpenCollection
	}
	if strings.HasSuffix(lower, ".json") {
		return FormatPostman
	}
	return FormatUnknown
}

func detectByExtension(lower string) Format {
	switch {
	case strings.HasSuffix(lower, ".yml"), strings.HasSuffix(lower, ".yaml"):
		return FormatOpenCollection
	case strings.HasSuffix(lower, ".json"):
		return FormatPostman
	default:
		return FormatUnknown
	}
}

func matchYAMLMarker(text, marker string) bool {
	for _, line := range strings.Split(text, "\n") {
		trim := strings.TrimSpace(line)
		if trim == "" || strings.HasPrefix(trim, "#") {
			continue
		}
		return strings.HasPrefix(strings.ToLower(trim), marker)
	}
	return false
}
