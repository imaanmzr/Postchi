package linkmatcher

import (
	"strings"
)

// CollectionInfo holds collection metadata for path template rendering.
type CollectionInfo struct {
	ID   string
	Name string
}

// TemplateVars builds placeholder values for a path template.
func TemplateVars(req Request, col CollectionInfo, pathPrefix string) map[string]string {
	return map[string]string{
		"request_slug":    RequestSlug(req),
		"request_name":    req.Name,
		"collection_slug": NormalizeSlug(col.Name),
		"collection_name": col.Name,
		"method":          strings.ToLower(strings.TrimSpace(req.Method)),
		"operation_id":    strings.TrimSpace(req.SourceOperationID),
		"path_prefix":     strings.Trim(strings.TrimSpace(pathPrefix), "/"),
	}
}

// RenderTemplate substitutes {placeholder} tokens in a link template.
func RenderTemplate(template string, vars map[string]string) string {
	out := template
	for key, val := range vars {
		out = strings.ReplaceAll(out, "{"+key+"}", val)
	}
	return out
}

// NormalizeSourcePath strips .md extension and normalizes slashes for comparison.
func NormalizeSourcePath(p string) string {
	p = strings.TrimSpace(p)
	p = strings.TrimPrefix(p, "./")
	p = strings.TrimPrefix(p, "/")
	p = strings.TrimSuffix(p, ".md")
	p = strings.TrimSuffix(p, ".markdown")
	p = strings.TrimSuffix(p, ".MD")
	p = strings.TrimSuffix(p, ".MARKDOWN")
	return strings.Trim(p, "/")
}

// MatchPathTemplate matches docs to requests by rendering a path template per request.
func MatchPathTemplate(template, pathPrefix string, docs []Doc, requests []Request, collections map[string]CollectionInfo) []Candidate {
	template = strings.TrimSpace(template)
	if template == "" {
		return nil
	}
	docByPath := make(map[string]Doc, len(docs))
	for _, doc := range docs {
		norm := NormalizeSourcePath(doc.SourcePath)
		if norm != "" {
			docByPath[norm] = doc
		}
	}
	var out []Candidate
	for _, req := range requests {
		col := collections[req.CollectionID]
		if col.ID == "" {
			col = CollectionInfo{ID: req.CollectionID, Name: req.CollectionName}
		}
		vars := TemplateVars(req, col, pathPrefix)
		rendered := RenderTemplate(template, vars)
		norm := NormalizeSourcePath(rendered)
		if norm == "" {
			continue
		}
		doc, ok := docByPath[norm]
		if !ok {
			continue
		}
		out = append(out, Candidate{
			DocID: doc.ID, RequestID: req.ID,
			Reason: "path_template", Confidence: "exact",
			Evidence: map[string]string{
				"template":    template,
				"source_path": norm,
			},
		})
	}
	return out
}

// ValidateLinkTemplate reports whether a link template is usable.
func ValidateLinkTemplate(template string) bool {
	template = strings.TrimSpace(template)
	if template == "" {
		return true
	}
	return strings.Contains(template, "{request_slug}") || strings.Contains(template, "{request_name}")
}
