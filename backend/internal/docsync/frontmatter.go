package docsync

import (
	"regexp"
	"strings"

	"github.com/imaanmzr/postchi/backend/internal/shared/operationid"
)

var frontmatterRe = regexp.MustCompile(`(?s)^---\s*\n(.*?)\n---\s*\n(.*)$`)

func parseMarkdownDoc(content, path string) (title string, ops []string, body string) {
	title = strings.TrimSuffix(path, ".md")
	body = content
	m := frontmatterRe.FindStringSubmatch(content)
	if len(m) != 3 {
		return title, ops, body
	}
	body = strings.TrimSpace(m[2])
	fm := m[1]
	title = parseFrontmatterTitle(fm, title)
	rawOps := parseFrontmatterOperations(fm)
	ops = normalizeLinkedOperations(rawOps)
	return title, ops, body
}

func parseFrontmatterTitle(fm, fallback string) string {
	for _, line := range strings.Split(fm, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "title:") {
			val := strings.TrimSpace(strings.TrimPrefix(line, "title:"))
			val = strings.Trim(val, `"'`)
			if val != "" {
				return val
			}
		}
	}
	return fallback
}

func parseFrontmatterOperations(fm string) []string {
	lines := strings.Split(fm, "\n")
	var out []string
	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if !strings.HasPrefix(line, "operations:") {
			continue
		}
		rest := strings.TrimSpace(strings.TrimPrefix(line, "operations:"))
		if rest != "" {
			out = append(out, parseOperationsInline(rest)...)
			continue
		}
		// Multi-line YAML array.
		for i++; i < len(lines); i++ {
			item := strings.TrimSpace(lines[i])
			if item == "" {
				continue
			}
			if !strings.HasPrefix(item, "-") {
				i--
				break
			}
			val := strings.TrimSpace(strings.TrimPrefix(item, "-"))
			val = strings.Trim(val, `"'`)
			if val != "" {
				out = append(out, val)
			}
		}
	}
	return out
}

func parseOperationsInline(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	if strings.HasPrefix(raw, "[") {
		raw = strings.Trim(raw, "[]")
	}
	var out []string
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		part = strings.Trim(part, `"'`)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func normalizeLinkedOperations(raw []string) []string {
	seen := make(map[string]struct{})
	var out []string
	for _, r := range raw {
		for _, n := range operationid.NormalizeFrontmatterOp(r) {
			if _, ok := seen[n]; ok {
				continue
			}
			seen[n] = struct{}{}
			out = append(out, n)
		}
	}
	return out
}
