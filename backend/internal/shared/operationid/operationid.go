package operationid

import (
	"net/url"
	"regexp"
	"strings"
)

var brunoPathParamRe = regexp.MustCompile(`:([a-zA-Z_][a-zA-Z0-9_]*)`)

// CanonicalFromMethodURL builds the canonical method-path operation ID used for
// Bruno imports and doc frontmatter (e.g. get-/users/{id}).
func CanonicalFromMethodURL(method, rawURL string) string {
	method = strings.ToLower(strings.TrimSpace(method))
	path := normalizeURLPath(rawURL)
	if method == "" || path == "" {
		return ""
	}
	return method + "-" + path
}

// AliasesForRequest returns deduplicated operation ID aliases for matching docs.
func AliasesForRequest(method, rawURL, sourceOperationID string) []string {
	seen := make(map[string]struct{})
	add := func(ids []string, v string) []string {
		v = strings.TrimSpace(v)
		if v == "" {
			return ids
		}
		if _, ok := seen[v]; ok {
			return ids
		}
		seen[v] = struct{}{}
		return append(ids, v)
	}

	var out []string
	out = add(out, sourceOperationID)
	out = add(out, CanonicalFromMethodURL(method, rawURL))

	// Legacy OpenAPI fallback format: "get /users/{id}" (space instead of hyphen).
	if canonical := CanonicalFromMethodURL(method, rawURL); canonical != "" {
		parts := strings.SplitN(canonical, "-", 2)
		if len(parts) == 2 {
			out = add(out, parts[0]+" "+parts[1])
		}
	}

	// If sourceOperationID is method-path with space, also add canonical hyphen form.
	if sourceOperationID != "" {
		for _, sep := range []string{" ", "-"} {
			if idx := strings.Index(sourceOperationID, sep); idx > 0 {
				m := strings.ToLower(strings.TrimSpace(sourceOperationID[:idx]))
				p := strings.TrimSpace(sourceOperationID[idx+len(sep):])
				if strings.HasPrefix(p, "/") {
					out = add(out, CanonicalFromMethodURL(m, p))
				}
			}
		}
	}

	return out
}

// NormalizeFrontmatterOp expands a frontmatter operation entry into stored aliases.
func NormalizeFrontmatterOp(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	seen := make(map[string]struct{})
	add := func(ids []string, v string) []string {
		v = strings.TrimSpace(v)
		if v == "" {
			return ids
		}
		if _, ok := seen[v]; ok {
			return ids
		}
		seen[v] = struct{}{}
		return append(ids, v)
	}

	var out []string
	out = add(out, raw)

	// Already canonical method-path: get-/users/{id}
	if idx := strings.Index(raw, "-/"); idx > 0 {
		method := strings.ToLower(raw[:idx])
		path := raw[idx+1:]
		out = add(out, CanonicalFromMethodURL(method, path))
		out = add(out, method+" "+path)
		return out
	}

	// Legacy space form: get /users/{id}
	if idx := strings.Index(raw, " /"); idx > 0 {
		method := strings.ToLower(strings.TrimSpace(raw[:idx]))
		path := strings.TrimSpace(raw[idx:])
		out = add(out, CanonicalFromMethodURL(method, path))
		return out
	}

	// Bare OpenAPI operationId with no path — keep as-is only.
	return out
}

// Matches reports whether any linked operation ID intersects request aliases.
func Matches(linkedOps []string, requestAliases []string) bool {
	if len(linkedOps) == 0 || len(requestAliases) == 0 {
		return false
	}
	set := make(map[string]struct{}, len(linkedOps))
	for _, op := range linkedOps {
		set[strings.TrimSpace(op)] = struct{}{}
	}
	for _, alias := range requestAliases {
		if _, ok := set[strings.TrimSpace(alias)]; ok {
			return true
		}
	}
	return false
}

func normalizeURLPath(rawURL string) string {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return ""
	}

	// Strip Bruno/OpenAPI template variables like {{baseUrl}}.
	for {
		start := strings.Index(rawURL, "{{")
		if start < 0 {
			break
		}
		end := strings.Index(rawURL[start:], "}}")
		if end < 0 {
			break
		}
		rawURL = rawURL[:start] + rawURL[start+end+2:]
	}
	rawURL = strings.TrimSpace(rawURL)

	// If no scheme, ensure path form for parsing.
	parseTarget := rawURL
	if !strings.Contains(parseTarget, "://") && !strings.HasPrefix(parseTarget, "/") {
		if strings.Contains(parseTarget, "/") {
			parseTarget = "http://placeholder/" + strings.TrimLeft(parseTarget, "/")
		} else {
			parseTarget = "http://placeholder/" + parseTarget
		}
	} else if strings.HasPrefix(parseTarget, "/") {
		parseTarget = "http://placeholder" + parseTarget
	}

	u, err := url.Parse(parseTarget)
	if err != nil {
		path := rawURL
		if i := strings.Index(path, "?"); i >= 0 {
			path = path[:i]
		}
		return normalizePath(path)
	}
	return normalizePath(u.Path)
}

func normalizePath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	// Bruno :id → OpenAPI {id}
	path = brunoPathParamRe.ReplaceAllString(path, `{$1}`)
	// Collapse duplicate slashes.
	for strings.Contains(path, "//") {
		path = strings.ReplaceAll(path, "//", "/")
	}
	// Trim trailing slash except root.
	if len(path) > 1 {
		path = strings.TrimRight(path, "/")
	}
	return path
}
