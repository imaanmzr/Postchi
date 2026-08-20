package docsync

import (
	"regexp"
	"strings"

	"github.com/imaanmzr/postchi/backend/internal/docsync/linkmatcher"
	"github.com/imaanmzr/postchi/backend/internal/shared/operationid"
)

var frontmatterRe = regexp.MustCompile(`(?s)^---\s*\n(.*?)\n---\s*\n(.*)$`)

func parseMarkdownDoc(content, path string) (title string, ops []string, requestNames []string, body string) {
	title = strings.TrimSuffix(path, ".md")
	body = content
	m := frontmatterRe.FindStringSubmatch(content)
	if len(m) != 3 {
		return title, ops, requestNames, body
	}
	body = strings.TrimSpace(m[2])
	fm := m[1]
	title = parseFrontmatterTitle(fm, title)
	rawOps := parseFrontmatterKeyList(fm, "operations")
	ops = normalizeLinkedOperations(rawOps)
	rawRequests := parseFrontmatterRequests(fm)
	requestNames = normalizeLinkedRequestNames(rawRequests)
	return title, ops, requestNames, body
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

func parseFrontmatterRequests(fm string) []string {
	var out []string
	out = append(out, parseFrontmatterScalar(fm, "request")...)
	out = append(out, parseFrontmatterKeyList(fm, "requests")...)
	return out
}

func parseFrontmatterScalar(fm, key string) []string {
	prefix := key + ":"
	for _, line := range strings.Split(fm, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, prefix) {
			continue
		}
		val := strings.TrimSpace(strings.TrimPrefix(line, prefix))
		val = strings.Trim(val, `"'`)
		if val != "" {
			return []string{val}
		}
	}
	return nil
}

func parseFrontmatterKeyList(fm, key string) []string {
	prefix := key + ":"
	lines := strings.Split(fm, "\n")
	var out []string
	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if !strings.HasPrefix(line, prefix) {
			continue
		}
		rest := strings.TrimSpace(strings.TrimPrefix(line, prefix))
		if rest != "" {
			out = append(out, parseFrontmatterInlineList(rest)...)
			continue
		}
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

func parseFrontmatterInlineList(raw string) []string {
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

func normalizeLinkedRequestNames(raw []string) []string {
	seen := make(map[string]struct{})
	var out []string
	for _, r := range raw {
		n := linkmatcher.NormalizeSlug(r)
		if n == "" {
			continue
		}
		if _, ok := seen[n]; ok {
			continue
		}
		seen[n] = struct{}{}
		out = append(out, n)
	}
	return out
}
