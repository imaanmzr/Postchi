package domain

import (
	"regexp"
	"sort"
	"strings"
)

var placeholderPattern = regexp.MustCompile(`\{\{([^}]+)\}\}`)

func ExtractPlaceholderNames(texts ...string) []string {
	seen := map[string]bool{}
	for _, text := range texts {
		for _, match := range placeholderPattern.FindAllStringSubmatch(text, -1) {
			if len(match) < 2 {
				continue
			}
			name := strings.TrimSpace(match[1])
			if name != "" {
				seen[name] = true
			}
		}
	}
	out := make([]string, 0, len(seen))
	for k := range seen {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
