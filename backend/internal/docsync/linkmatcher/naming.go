package linkmatcher

import (
	"strings"
)

// NormalizeSlug lowercases and converts spaces/underscores to hyphens for name matching.
func NormalizeSlug(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, "_", "-")
	s = strings.ReplaceAll(s, " ", "-")
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	return strings.Trim(s, "-")
}

// RequestSlug returns the normalized slug for a request display name.
func RequestSlug(req Request) string {
	return NormalizeSlug(req.Name)
}

// DocMatchKeys returns normalized keys used to match a doc against request names.
func DocMatchKeys(doc Doc) []string {
	seen := make(map[string]struct{})
	add := func(keys []string, v string) []string {
		v = NormalizeSlug(v)
		if v == "" {
			return keys
		}
		if _, ok := seen[v]; ok {
			return keys
		}
		seen[v] = struct{}{}
		return append(keys, v)
	}
	var keys []string
	if base := pathBaseName(doc.SourcePath); base != "" {
		keys = add(keys, base)
	}
	if seg := lastSlugSegment(doc.Slug); seg != "" {
		keys = add(keys, seg)
	}
	if doc.Title != "" {
		keys = add(keys, doc.Title)
	}
	return keys
}

func pathBaseName(p string) string {
	p = strings.TrimSuffix(strings.TrimSpace(p), "/")
	if p == "" {
		return ""
	}
	if i := strings.LastIndex(p, "/"); i >= 0 {
		return p[i+1:]
	}
	return p
}

func lastSlugSegment(slug string) string {
	slug = strings.Trim(slug, "-")
	if slug == "" {
		return ""
	}
	parts := strings.Split(slug, "-")
	return parts[len(parts)-1]
}

// MatchExactName returns candidates where a request slug exactly matches a doc key.
func MatchExactName(docs []Doc, requests []Request) []Candidate {
	var out []Candidate
	for _, doc := range docs {
		docKeys := DocMatchKeys(doc)
		if len(docKeys) == 0 {
			continue
		}
		keySet := make(map[string]struct{}, len(docKeys))
		for _, k := range docKeys {
			keySet[k] = struct{}{}
		}
		for _, req := range requests {
			reqSlug := RequestSlug(req)
			if reqSlug == "" {
				continue
			}
			if _, ok := keySet[reqSlug]; !ok {
				continue
			}
			out = append(out, Candidate{
				DocID: doc.ID, RequestID: req.ID,
				Reason: "exact_name", Confidence: "exact",
				Evidence: map[string]string{"request_slug": reqSlug},
			})
		}
	}
	return out
}

// PartitionUnique splits candidates into 1:1 pairs (auto) vs ambiguous pairs.
func PartitionUnique(candidates []Candidate) (auto, ambiguous []Candidate) {
	if len(candidates) == 0 {
		return nil, nil
	}
	byRequest := make(map[string][]Candidate)
	byDoc := make(map[string][]Candidate)
	for _, c := range candidates {
		byRequest[c.RequestID] = append(byRequest[c.RequestID], c)
		byDoc[c.DocID] = append(byDoc[c.DocID], c)
	}
	seen := make(map[string]struct{})
	for _, c := range candidates {
		key := c.DocID + ":" + c.RequestID
		if _, ok := seen[key]; ok {
			continue
		}
		if len(byRequest[c.RequestID]) == 1 && len(byDoc[c.DocID]) == 1 {
			auto = append(auto, c)
		} else {
			ambiguous = append(ambiguous, c)
		}
		seen[key] = struct{}{}
	}
	return auto, ambiguous
}
