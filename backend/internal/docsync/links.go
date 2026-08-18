package docsync

import (
	"regexp"
	"strings"
)

var (
	wikilinkRe = regexp.MustCompile(`\[\[([^\]|]+)(?:\|[^\]]+)?\]\]`)
	mdLinkRe   = regexp.MustCompile(`\[[^\]]*\]\(([^)]+)\)`)
)

// filePathToSourcePath strips markdown extensions from a git file path.
func filePathToSourcePath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.TrimPrefix(path, "./")
	path = strings.TrimPrefix(path, "/")
	lower := strings.ToLower(path)
	for _, ext := range []string{".markdown", ".md"} {
		if strings.HasSuffix(lower, ext) {
			return path[:len(path)-len(ext)]
		}
	}
	return path
}

// pathToSlug converts a git file path or link target to a workspace doc slug.
func pathToSlug(path string) string {
	path = strings.TrimSpace(path)
	path = strings.TrimPrefix(path, "./")
	path = strings.TrimPrefix(path, "/")
	if idx := strings.Index(path, "#"); idx >= 0 {
		path = path[:idx]
	}
	path = strings.TrimSuffix(path, ".md")
	path = strings.TrimSuffix(path, ".MD")
	path = strings.TrimSuffix(path, ".markdown")
	path = strings.TrimSuffix(path, ".MARKDOWN")
	return strings.ReplaceAll(path, "/", "-")
}

// slugify converts a human-readable title to a slug for fuzzy matching.
func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, " ", "-")
	return s
}

type docIndex struct {
	slugs  map[string]struct{}
	titles map[string]string // lower(title) -> slug
}

func buildDocIndex(docs []WorkspaceDoc) docIndex {
	idx := docIndex{
		slugs:  make(map[string]struct{}, len(docs)),
		titles: make(map[string]string, len(docs)),
	}
	for _, d := range docs {
		idx.slugs[d.Slug] = struct{}{}
		idx.titles[strings.ToLower(d.Title)] = d.Slug
	}
	return idx
}

func (idx docIndex) resolve(target string) string {
	target = strings.TrimSpace(target)
	if target == "" {
		return ""
	}
	if _, ok := idx.slugs[target]; ok {
		return target
	}
	slug := pathToSlug(target)
	if _, ok := idx.slugs[slug]; ok {
		return slug
	}
	slug = slugify(target)
	if _, ok := idx.slugs[slug]; ok {
		return slug
	}
	if s, ok := idx.titles[strings.ToLower(target)]; ok {
		return s
	}
	return ""
}

// extractDocLinks returns unique target doc slugs referenced in markdown content.
func extractDocLinks(content string, idx docIndex) []string {
	seen := make(map[string]struct{})
	var links []string
	add := func(slug string) {
		if slug == "" {
			return
		}
		if _, ok := seen[slug]; ok {
			return
		}
		seen[slug] = struct{}{}
		links = append(links, slug)
	}

	for _, m := range wikilinkRe.FindAllStringSubmatch(content, -1) {
		add(idx.resolve(m[1]))
	}
	for _, m := range mdLinkRe.FindAllStringSubmatch(content, -1) {
		href := strings.TrimSpace(m[1])
		if strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://") || strings.HasPrefix(href, "mailto:") {
			continue
		}
		if strings.HasSuffix(href, ".md") || !strings.Contains(href, "://") {
			add(idx.resolve(href))
		}
	}
	return links
}
