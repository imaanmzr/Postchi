package validate

import (
	"fmt"
	"strings"

	"github.com/imaanmzr/postchi/backend/internal/importexport/bruno"
	"github.com/imaanmzr/postchi/backend/internal/importexport/model"
)

func CountRequests(col model.Collection) int {
	total := len(col.Requests)
	for _, child := range col.Children {
		total += CountRequests(child)
	}
	return total
}

func Collection(col model.Collection) error {
	if CountRequests(col) == 0 {
		return fmt.Errorf("collection %q contains no requests", col.Name)
	}
	return nil
}

func HasHTTPMethodBlock(parsed bruno.ParsedBru) bool {
	for _, method := range []string{"get", "post", "put", "patch", "delete", "head", "options"} {
		if block, ok := parsed.Sections[method]; ok && strings.TrimSpace(block) != "" {
			return true
		}
	}
	return false
}

func IsCollectionOrFolderMeta(parsed bruno.ParsedBru) bool {
	if HasHTTPMethodBlock(parsed) {
		return false
	}
	if _, ok := parsed.Sections["meta"]; ok {
		return true
	}
	return parsed.Name != "" && len(parsed.Sections) > 0
}

func BrunoRequest(parsed bruno.ParsedBru) error {
	if _, ok := parsed.Sections["meta"]; !ok {
		return fmt.Errorf("malformed or missing meta block")
	}
	for _, method := range []string{"get", "post", "put", "patch", "delete"} {
		if block, ok := parsed.Sections[method]; ok {
			if strings.TrimSpace(bruno.ToRequest(parsed).URL) == "" || strings.TrimSpace(block) == "" {
				return fmt.Errorf("request URL is missing")
			}
			return nil
		}
	}
	return fmt.Errorf("HTTP method block is missing")
}
