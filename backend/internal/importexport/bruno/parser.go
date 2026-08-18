package bruno

import (
	"regexp"
	"strings"
)

var sectionRe = regexp.MustCompile(`^\s*([a-z0-9:_-]+)\s*\{\s*$`)

type ParsedBru struct {
	Name     string
	Sections map[string]string
}

type BruRequest struct {
	Name             string
	Method           string
	URL              string
	Headers          []KV
	PreRequestScript string
	TestScript       string
	Body             string
	AuthType         string
	AuthToken        string
}

type KV struct {
	Key   string
	Value string
}

type BruVars struct {
	PreRequest   []KV
	PostResponse []KVExpr
}

type KVExpr struct {
	Key  string
	Expr string
}

func Parse(content string) ParsedBru {
	lines := strings.Split(content, "\n")
	var current string
	var buf strings.Builder
	sections := map[string]string{}
	name := ""

	for _, line := range lines {
		trim := strings.TrimSpace(line)
		if trim == "}" && current != "" {
			sections[current] = strings.TrimSpace(buf.String())
			buf.Reset()
			current = ""
			continue
		}
		if m := sectionRe.FindStringSubmatch(trim); len(m) == 2 {
			if current != "" {
				sections[current] = strings.TrimSpace(buf.String())
				buf.Reset()
			}
			current = m[1]
			continue
		}
		if current == "meta" {
			if strings.HasPrefix(trim, "name:") {
				name = strings.TrimSpace(strings.TrimPrefix(trim, "name:"))
			}
			continue
		}
		if current != "" {
			buf.WriteString(line)
			buf.WriteString("\n")
		}
	}
	if current != "" {
		sections[current] = strings.TrimSpace(buf.String())
	}
	return ParsedBru{Name: name, Sections: sections}
}

func ToRequest(parsed ParsedBru) BruRequest {
	req := BruRequest{Name: parsed.Name, Method: "GET"}
	for key, block := range parsed.Sections {
		switch key {
		case "get", "post", "put", "patch", "delete":
			req.Method = strings.ToUpper(key)
			req.URL = firstLine(block)
		case "headers":
			req.Headers = parseKVBlock(block)
		case "body":
			req.Body = block
		case "auth:bearer":
			req.AuthType = "bearer"
			req.AuthToken = parseSingleValue(block, "token")
		case "script:pre-request":
			req.PreRequestScript = block
		case "script:post-response", "tests":
			req.TestScript += block
		}
	}
	return req
}

func ToVars(pre, post string) BruVars {
	v := BruVars{}
	for _, line := range strings.Split(pre, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			v.PreRequest = append(v.PreRequest, KV{Key: strings.TrimSpace(parts[0]), Value: strings.TrimSpace(parts[1])})
		}
	}
	for _, line := range strings.Split(post, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			v.PostResponse = append(v.PostResponse, KVExpr{Key: strings.TrimSpace(parts[0]), Expr: strings.TrimSpace(parts[1])})
		}
	}
	return v
}

func ExportRequest(req BruRequest) string {
	var sb strings.Builder
	sb.WriteString("meta {\n  name: " + req.Name + "\n  type: http\n}\n\n")
	sb.WriteString(strings.ToLower(req.Method) + " {\n  url: " + req.URL + "\n}\n\n")
	if len(req.Headers) > 0 {
		sb.WriteString("headers {\n")
		for _, h := range req.Headers {
			sb.WriteString("  " + h.Key + ": " + h.Value + "\n")
		}
		sb.WriteString("}\n\n")
	}
	if req.Body != "" {
		sb.WriteString("body {\n" + req.Body + "\n}\n\n")
	}
	if req.PreRequestScript != "" {
		sb.WriteString("script:pre-request {\n" + req.PreRequestScript + "\n}\n\n")
	}
	if req.TestScript != "" {
		sb.WriteString("tests {\n" + req.TestScript + "\n}\n\n")
	}
	return sb.String()
}

func ExportCollectionMeta(name string, vars BruVars) string {
	var sb strings.Builder
	sb.WriteString("meta {\n  name: " + name + "\n  type: collection\n}\n\n")
	if len(vars.PreRequest) > 0 {
		sb.WriteString("vars:pre-request {\n")
		for _, v := range vars.PreRequest {
			sb.WriteString("  " + v.Key + ": " + v.Value + "\n")
		}
		sb.WriteString("}\n\n")
	}
	if len(vars.PostResponse) > 0 {
		sb.WriteString("vars:post-response {\n")
		for _, v := range vars.PostResponse {
			sb.WriteString("  " + v.Key + ": " + v.Expr + "\n")
		}
		sb.WriteString("}\n")
	}
	return sb.String()
}

func firstLine(s string) string {
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "url:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "url:"))
		}
		if line != "" {
			return line
		}
	}
	return strings.TrimSpace(s)
}

func parseKVBlock(block string) []KV {
	var out []KV
	for _, line := range strings.Split(block, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			out = append(out, KV{Key: strings.TrimSpace(parts[0]), Value: strings.TrimSpace(parts[1])})
		}
	}
	return out
}

func parseSingleValue(block, key string) string {
	for _, line := range strings.Split(block, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, key+":") {
			return strings.TrimSpace(strings.TrimPrefix(line, key+":"))
		}
	}
	return ""
}
