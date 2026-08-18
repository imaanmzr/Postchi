package request

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/imaanmzr/postchi/backend/internal/db"
	"github.com/imaanmzr/postchi/backend/internal/db/sqlc"
	"github.com/imaanmzr/postchi/backend/internal/script"
	"github.com/imaanmzr/postchi/backend/internal/shared/config"
	"github.com/imaanmzr/postchi/backend/internal/shared/crypto"
	"github.com/imaanmzr/postchi/backend/internal/shared/domain"
)

var varPattern = regexp.MustCompile(`\{\{([^}]+)\}\}`)

type KVPair struct {
	Key     string `json:"key"`
	Value   string `json:"value"`
	Enabled bool   `json:"enabled"`
}

type FormField struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	Enabled     bool   `json:"enabled"`
	Type        string `json:"type,omitempty"` // text | file
	FileName    string `json:"file_name,omitempty"`
	ContentType string `json:"content_type,omitempty"`
	FileData    string `json:"file_data,omitempty"` // base64-encoded
}

type BodySpec struct {
	Mode       string      `json:"mode"`
	Raw        string      `json:"raw"`
	RawLang    string      `json:"raw_lang"`
	GraphQL    *struct {
		Query     string `json:"query"`
		Variables string `json:"variables"`
	} `json:"graphql,omitempty"`
	FormData   []FormField `json:"form_data,omitempty"`
	URLEncoded []KVPair    `json:"urlencoded,omitempty"`
}

type AuthSpec struct {
	Type   string         `json:"type"`
	Config map[string]any `json:"config,omitempty"`
}

type Settings struct {
	TimeoutMS       int  `json:"timeout_ms"`
	FollowRedirects bool `json:"follow_redirects"`
	VerifySSL       bool `json:"verify_ssl"`
}

type Model struct {
	ID                 string         `json:"id"`
	CollectionID       string         `json:"collection_id"`
	Name               string         `json:"name"`
	Method             string         `json:"method"`
	URL                string         `json:"url"`
	Headers            []KVPair       `json:"headers"`
	Params             []KVPair       `json:"params"`
	PathVars           []KVPair       `json:"path_vars"`
	Body               BodySpec       `json:"body"`
	Auth               AuthSpec       `json:"auth"`
	Settings           Settings       `json:"settings"`
	PreRequestScript   string         `json:"pre_request_script"`
	TestScript         string         `json:"test_script"`
	SortOrder          int            `json:"sort_order"`
	Description        string         `json:"description"`
	TemplateID         *string        `json:"template_id,omitempty"`
	IsTemplate         bool           `json:"is_template"`
	OverriddenFields   []string       `json:"overridden_fields,omitempty"`
	SourceSpecID       *string        `json:"source_spec_id,omitempty"`
	SourceOperationID  string         `json:"source_operation_id,omitempty"`
	SourceOpHash       string         `json:"source_op_hash,omitempty"`
	ApiDoc             json.RawMessage `json:"api_doc,omitempty"`
	DocsOverridden     bool           `json:"docs_overridden"`
}

type Timing struct {
	DNS      int64 `json:"dns_ms"`
	Connect  int64 `json:"connect_ms"`
	TLS      int64 `json:"tls_ms"`
	TTFB     int64 `json:"ttfb_ms"`
	Download int64 `json:"download_ms"`
	Total    int64 `json:"total_ms"`
}

type ExecuteResult struct {
	StatusCode  int              `json:"status_code"`
	Headers     map[string]string `json:"headers"`
	Body        string           `json:"body"`
	BodySize    int64            `json:"body_size"`
	Timing      Timing           `json:"timing"`
	TestResults []script.TestResult `json:"test_results"`
	Console     []string         `json:"console"`
	Error       string           `json:"error,omitempty"`
	HistoryID   string           `json:"history_id,omitempty"`
}

type Executor struct {
	cfg    *config.Config
	store  *db.Store
	script *script.Sandbox
	crypto *crypto.Service
}

func NewExecutor(cfg *config.Config, store *db.Store, cryptoSvc *crypto.Service) *Executor {
	return &Executor{cfg: cfg, store: store, script: script.NewSandbox(), crypto: cryptoSvc}
}

// Variable precedence (highest wins): local > data file > environment > collection > workspace > built-in
func (e *Executor) BuildVariables(ctx context.Context, workspaceID, collectionID, environmentID uuid.UUID, local map[string]string) map[string]string {
	vars := map[string]string{
		"$timestamp": fmt.Sprintf("%d", time.Now().Unix()),
		"$isoTimestamp": time.Now().UTC().Format(time.RFC3339),
	}

	wsVars, _ := e.store.GetWorkspaceVariables(ctx, db.PGUUID(workspaceID))
	mergeWorkspaceVars(vars, wsVars)
	for _, id := range e.collectionAncestorIDs(ctx, collectionID) {
		colVars, err := e.store.GetCollectionVariables(ctx, db.PGUUID(id))
		if err != nil {
			continue
		}
		mergeCollectionVars(vars, colVars)
	}

	if environmentID != uuid.Nil {
		rows, err := e.store.ListEnvironmentVariablesForExecutor(ctx, db.PGUUID(environmentID))
		if err == nil {
			for _, row := range rows {
				if !row.Enabled || row.Phase == "post_response" {
					continue
				}
				if row.IsSecret {
					if plain, err := e.crypto.Decrypt(row.ValueEncrypted); err == nil {
						vars[row.Key] = plain
					}
				} else {
					vars[row.Key] = row.ValueEncrypted
				}
			}
		}
	}
	for k, v := range local {
		vars[k] = v
	}
	return vars
}

func (e *Executor) BuildVariablesForRequest(ctx context.Context, workspaceID, collectionID, environmentID uuid.UUID, local map[string]string, req Model) map[string]string {
	vars := e.BuildVariables(ctx, workspaceID, collectionID, environmentID, local)
	if req.SourceSpecID != nil && *req.SourceSpecID != "" && environmentID != uuid.Nil {
		specID, err := uuid.Parse(*req.SourceSpecID)
		if err == nil {
			e.applySpecBaseURLForSpec(ctx, specID, environmentID, vars)
		}
	}
	return vars
}

func (e *Executor) applySpecBaseURLForSpec(ctx context.Context, specID, environmentID uuid.UUID, vars map[string]string) {
	row, err := e.store.GetSpecBaseURLForEnvironment(ctx, sqlc.GetSpecBaseURLForEnvironmentParams{
		EnvironmentID: db.PGUUID(environmentID),
		SpecID:        db.PGUUID(specID),
	})
	if err != nil || row.BaseUrlVar == "" {
		return
	}
	if row.BaseUrl != "" {
		vars[row.BaseUrlVar] = row.BaseUrl
	}
}

func mergeWorkspaceVars(vars map[string]string, data []byte) {
	if len(data) == 0 {
		return
	}
	var m map[string]any
	if json.Unmarshal(data, &m) == nil {
		for k, v := range m {
			vars[k] = fmt.Sprint(v)
		}
	}
}

func mergeCollectionVars(vars map[string]string, data []byte) {
	spec := domain.ParseVariablesSpec(data)
	for k, v := range spec.ToMap() {
		vars[k] = v
	}
}

func evalPostResponseExprs(exprs []domain.PostResponseVar, statusCode int, body string) map[string]string {
	out := map[string]string{}
	for _, row := range exprs {
		if !row.Enabled || row.Name == "" || row.Expr == "" {
			continue
		}
		out[row.Name] = evalExpr(row.Expr, statusCode, body)
	}
	return out
}

func evalExpr(expr string, statusCode int, body string) string {
	expr = strings.TrimSpace(expr)
	if expr == "res.status" || expr == "res.code" {
		return fmt.Sprintf("%d", statusCode)
	}
	if strings.HasPrefix(expr, "res.body.") {
		path := strings.TrimPrefix(expr, "res.body.")
		var parsed any
		if json.Unmarshal([]byte(body), &parsed) == nil {
			if v := jsonPathGet(parsed, path); v != nil {
				return fmt.Sprint(v)
			}
		}
	}
	return ""
}

func jsonPathGet(data any, path string) any {
	parts := strings.Split(path, ".")
	cur := data
	for _, p := range parts {
		m, ok := cur.(map[string]any)
		if !ok {
			return nil
		}
		cur, ok = m[p]
		if !ok {
			return nil
		}
	}
	return cur
}

func Interpolate(s string, vars map[string]string) string {
	return varPattern.ReplaceAllStringFunc(s, func(match string) string {
		key := strings.TrimSpace(match[2 : len(match)-2])
		if v, ok := vars[key]; ok {
			return v
		}
		return match
	})
}

func (e *Executor) Execute(ctx context.Context, req Model, vars map[string]string, inheritedPre, inheritedTest string) (*ExecuteResult, error) {
	result := &ExecuteResult{Headers: map[string]string{}}

	sb := e.script.NewContext(vars)
	preScript := inheritedPre + "\n" + req.PreRequestScript
	if preScript != "" {
		if err := sb.RunPreRequest(preScript); err != nil {
			result.Error = "pre-request script: " + err.Error()
		}
		vars = sb.Variables()
	}
	result.Console = append(result.Console, sb.Console()...)

	method := Interpolate(strings.ToUpper(req.Method), vars)
	rawURL := Interpolate(req.URL, vars)
	if rawURL == "" {
		return nil, fmt.Errorf("url is required")
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	for _, p := range req.Params {
		if p.Enabled && p.Key != "" {
			q.Set(p.Key, Interpolate(p.Value, vars))
		}
	}
	u.RawQuery = q.Encode()

	var body io.Reader
	var contentType string
	bodySpec := normalizeBodySpec(req.Body)
	switch bodySpec.Mode {
	case "raw":
		raw := Interpolate(bodySpec.Raw, vars)
		body = strings.NewReader(raw)
		contentType = contentTypeForRaw(bodySpec.RawLang)
	case "graphql":
		if req.Body.GraphQL != nil {
			payload := map[string]string{
				"query": Interpolate(req.Body.GraphQL.Query, vars),
				"variables": Interpolate(req.Body.GraphQL.Variables, vars),
			}
			b, _ := json.Marshal(payload)
			body = bytes.NewReader(b)
			contentType = "application/json"
		}
	case "urlencoded":
		form := url.Values{}
		for _, p := range req.Body.URLEncoded {
			if p.Enabled {
				form.Set(p.Key, Interpolate(p.Value, vars))
			}
		}
		body = strings.NewReader(form.Encode())
		contentType = "application/x-www-form-urlencoded"
	case "form-data":
		var buf bytes.Buffer
		writer := multipart.NewWriter(&buf)
		for _, p := range req.Body.FormData {
			if !p.Enabled || p.Key == "" {
				continue
			}
			fieldType := p.Type
			if fieldType == "" {
				fieldType = "text"
			}
			if fieldType == "file" && p.FileData != "" {
				data, err := base64.StdEncoding.DecodeString(p.FileData)
				if err != nil {
					return nil, fmt.Errorf("decode form file %q: %w", p.Key, err)
				}
				filename := p.FileName
				if filename == "" {
					filename = p.Key
				}
				part, err := writer.CreateFormFile(p.Key, filename)
				if err != nil {
					return nil, err
				}
				if _, err := part.Write(data); err != nil {
					return nil, err
				}
			} else {
				_ = writer.WriteField(p.Key, Interpolate(p.Value, vars))
			}
		}
		_ = writer.Close()
		body = &buf
		contentType = writer.FormDataContentType()
	}

	httpReq, err := http.NewRequestWithContext(ctx, method, u.String(), body)
	if err != nil {
		return nil, err
	}
	if contentType != "" {
		httpReq.Header.Set("Content-Type", contentType)
	}
	for _, h := range req.Headers {
		if h.Enabled && h.Key != "" {
			httpReq.Header.Set(h.Key, Interpolate(h.Value, vars))
		}
	}
	resolvedAuth := e.ResolveAuth(ctx, req.CollectionID, req.Auth)
	e.applyAuth(httpReq, resolvedAuth, vars)

	timeout := e.cfg.RequestTimeout
	if req.Settings.TimeoutMS > 0 {
		timeout = time.Duration(req.Settings.TimeoutMS) * time.Millisecond
	}
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: !req.Settings.VerifySSL},
	}
	client := &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}
	if !req.Settings.FollowRedirects {
		client.CheckRedirect = func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	var timing Timing
	trace := &httptrace.ClientTrace{
		DNSStart:             func(_ httptrace.DNSStartInfo) { timing.DNS = time.Now().UnixNano() },
		DNSDone:              func(_ httptrace.DNSDoneInfo) { timing.DNS = (time.Now().UnixNano() - timing.DNS) / 1e6 },
		ConnectStart:         func(_, _ string) { timing.Connect = time.Now().UnixNano() },
		ConnectDone:          func(_, _ string, _ error) { timing.Connect = (time.Now().UnixNano() - timing.Connect) / 1e6 },
		TLSHandshakeStart:    func() { timing.TLS = time.Now().UnixNano() },
		TLSHandshakeDone:     func(_ tls.ConnectionState, _ error) { timing.TLS = (time.Now().UnixNano() - timing.TLS) / 1e6 },
		GotFirstResponseByte: func() { timing.TTFB = time.Since(time.Now()).Milliseconds() },
	}
	start := time.Now()
	httpReq = httpReq.WithContext(httptrace.WithClientTrace(httpReq.Context(), trace))
	resp, err := client.Do(httpReq)
	timing.Total = time.Since(start).Milliseconds()
	if err != nil {
		result.Error = err.Error()
		return result, nil
	}
	defer resp.Body.Close()

	limited := io.LimitReader(resp.Body, e.cfg.MaxResponseBytes)
	respBody, _ := io.ReadAll(limited)
	result.StatusCode = resp.StatusCode
	result.Body = string(respBody)
	result.BodySize = int64(len(respBody))
	for k, v := range resp.Header {
		result.Headers[k] = strings.Join(v, ", ")
	}

	// Evaluate post-response variable expressions from collection chain
	postExprs := e.loadPostResponseExprs(ctx, req.CollectionID)
	for k, v := range evalPostResponseExprs(postExprs, result.StatusCode, result.Body) {
		if v != "" {
			vars[k] = v
		}
	}

	testScript := inheritedTest + "\n" + req.TestScript
	if testScript != "" {
		sb.SetResponse(result.StatusCode, result.Body, result.Headers)
		if err := sb.RunTests(testScript); err != nil {
			result.Error = "test script: " + err.Error()
		}
		result.TestResults = sb.TestResults()
		result.Console = append(result.Console, sb.Console()...)
	}
	return result, nil
}

func (e *Executor) loadPostResponseExprs(ctx context.Context, collectionID string) []domain.PostResponseVar {
	cid, err := uuid.Parse(collectionID)
	if err != nil {
		return nil
	}
	var out []domain.PostResponseVar
	for _, id := range e.collectionAncestorIDs(ctx, cid) {
		vars, err := e.store.GetCollectionVariables(ctx, db.PGUUID(id))
		if err != nil {
			continue
		}
		spec := domain.ParseVariablesSpec(vars)
		out = append(out, spec.PostResponse...)
	}
	return out
}

// collectionAncestorIDs returns collection IDs from root to leaf (request's collection).
func (e *Executor) collectionAncestorIDs(ctx context.Context, collectionID uuid.UUID) []uuid.UUID {
	var chain []uuid.UUID
	seen := make(map[uuid.UUID]bool)
	cur := collectionID
	for {
		if seen[cur] {
			break
		}
		seen[cur] = true
		parentPg, err := e.store.GetCollectionParentID(ctx, db.PGUUID(cur))
		if err != nil {
			break
		}
		chain = append([]uuid.UUID{cur}, chain...)
		if !parentPg.Valid {
			break
		}
		cur = db.FromPGUUID(parentPg)
	}
	return chain
}

func (e *Executor) applyAuth(req *http.Request, auth AuthSpec, vars map[string]string) {
	cfg := auth.Config
	if cfg == nil {
		cfg = map[string]any{}
	}
	switch auth.Type {
	case "basic":
		user := Interpolate(fmt.Sprint(cfg["username"]), vars)
		pass := Interpolate(fmt.Sprint(cfg["password"]), vars)
		req.SetBasicAuth(user, pass)
	case "bearer":
		token := Interpolate(fmt.Sprint(cfg["token"]), vars)
		req.Header.Set("Authorization", "Bearer "+token)
	case "apikey":
		key := Interpolate(fmt.Sprint(cfg["key"]), vars)
		val := Interpolate(fmt.Sprint(cfg["value"]), vars)
		if fmt.Sprint(cfg["in"]) == "query" {
			q := req.URL.Query()
			q.Set(key, val)
			req.URL.RawQuery = q.Encode()
		} else {
			req.Header.Set(key, val)
		}
	}
}

func normalizeBodySpec(body BodySpec) BodySpec {
	switch body.Mode {
	case "json":
		body.Mode = "raw"
		if body.RawLang == "" {
			body.RawLang = "json"
		}
	case "form":
		body.Mode = "urlencoded"
	}
	return body
}

func contentTypeForRaw(lang string) string {
	switch lang {
	case "json":
		return "application/json"
	case "xml":
		return "application/xml"
	case "html":
		return "text/html"
	default:
		return "text/plain"
	}
}

type requestParts struct {
	Method  string
	URL     string
	Headers map[string]string
	Body    string
}

func buildRequestParts(req Model, vars map[string]string) (requestParts, error) {
	method := Interpolate(strings.ToUpper(req.Method), vars)
	rawURL := Interpolate(req.URL, vars)
	if rawURL == "" {
		return requestParts{}, fmt.Errorf("url is required")
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return requestParts{}, err
	}
	q := u.Query()
	for _, p := range req.Params {
		if p.Enabled && p.Key != "" {
			q.Set(p.Key, Interpolate(p.Value, vars))
		}
	}
	u.RawQuery = q.Encode()

	headers := map[string]string{}
	var body string
	bodySpec := normalizeBodySpec(req.Body)
	switch bodySpec.Mode {
	case "raw":
		body = Interpolate(bodySpec.Raw, vars)
		if ct := contentTypeForRaw(bodySpec.RawLang); ct != "" {
			headers["Content-Type"] = ct
		}
	case "graphql":
		if req.Body.GraphQL != nil {
			payload := map[string]string{
				"query":     Interpolate(req.Body.GraphQL.Query, vars),
				"variables": Interpolate(req.Body.GraphQL.Variables, vars),
			}
			b, _ := json.Marshal(payload)
			body = string(b)
			headers["Content-Type"] = "application/json"
		}
	case "urlencoded":
		form := url.Values{}
		for _, p := range req.Body.URLEncoded {
			if p.Enabled {
				form.Set(p.Key, Interpolate(p.Value, vars))
			}
		}
		body = form.Encode()
		headers["Content-Type"] = "application/x-www-form-urlencoded"
	case "form-data":
		var buf bytes.Buffer
		writer := multipart.NewWriter(&buf)
		for _, p := range req.Body.FormData {
			if !p.Enabled || p.Key == "" {
				continue
			}
			fieldType := p.Type
			if fieldType == "" {
				fieldType = "text"
			}
			if fieldType == "file" && p.FileData != "" {
				data, err := base64.StdEncoding.DecodeString(p.FileData)
				if err != nil {
					return requestParts{}, fmt.Errorf("decode form file %q: %w", p.Key, err)
				}
				filename := p.FileName
				if filename == "" {
					filename = p.Key
				}
				part, err := writer.CreateFormFile(p.Key, filename)
				if err != nil {
					return requestParts{}, err
				}
				if _, err := part.Write(data); err != nil {
					return requestParts{}, err
				}
			} else {
				_ = writer.WriteField(p.Key, Interpolate(p.Value, vars))
			}
		}
		_ = writer.Close()
		body = buf.String()
		headers["Content-Type"] = writer.FormDataContentType()
	}

	for _, h := range req.Headers {
		if h.Enabled && h.Key != "" {
			headers[h.Key] = Interpolate(h.Value, vars)
		}
	}
	applyAuthToHeaders(u, req.Auth, vars, headers)

	return requestParts{
		Method:  method,
		URL:     u.String(),
		Headers: headers,
		Body:    body,
	}, nil
}

func applyAuthToHeaders(u *url.URL, auth AuthSpec, vars map[string]string, headers map[string]string) {
	cfg := auth.Config
	if cfg == nil {
		cfg = map[string]any{}
	}
	switch auth.Type {
	case "basic":
		user := Interpolate(fmt.Sprint(cfg["username"]), vars)
		pass := Interpolate(fmt.Sprint(cfg["password"]), vars)
		headers["Authorization"] = "Basic " + base64.StdEncoding.EncodeToString([]byte(user+":"+pass))
	case "bearer":
		token := Interpolate(fmt.Sprint(cfg["token"]), vars)
		headers["Authorization"] = "Bearer " + token
	case "apikey":
		key := Interpolate(fmt.Sprint(cfg["key"]), vars)
		val := Interpolate(fmt.Sprint(cfg["value"]), vars)
		if fmt.Sprint(cfg["in"]) == "query" {
			q := u.Query()
			q.Set(key, val)
			u.RawQuery = q.Encode()
		} else {
			headers[key] = val
		}
	}
}

func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

func GenerateSnippet(req Model, lang string, vars map[string]string) string {
	if vars == nil {
		vars = map[string]string{}
	}
	parts, err := buildRequestParts(req, vars)
	if err != nil {
		return "# " + err.Error()
	}
	switch lang {
	case "curl":
		return buildCurlSnippet(parts)
	case "httpie":
		return buildHttpieSnippet(parts)
	case "wget":
		return buildWgetSnippet(parts)
	case "javascript":
		return buildJavaScriptSnippet(parts)
	case "python":
		return buildPythonSnippet(parts)
	case "go":
		return buildGoSnippet(parts)
	default:
		return ""
	}
}

func buildCurlSnippet(p requestParts) string {
	lines := []string{fmt.Sprintf("curl --request %s \\", p.Method)}
	lines = append(lines, fmt.Sprintf("  --url %s \\", shellQuote(p.URL)))
	for k, v := range p.Headers {
		lines = append(lines, fmt.Sprintf("  --header %s \\", shellQuote(k+": "+v)))
	}
	if p.Body != "" {
		lines = append(lines, fmt.Sprintf("  --data %s", shellQuote(p.Body)))
	} else {
		lines[len(lines)-1] = strings.TrimSuffix(lines[len(lines)-1], " \\")
	}
	return strings.Join(lines, "\n")
}

func buildHttpieSnippet(p requestParts) string {
	lines := []string{fmt.Sprintf("http %s %s \\", strings.ToUpper(p.Method), shellQuote(p.URL))}
	for k, v := range p.Headers {
		lines = append(lines, fmt.Sprintf("  %s:%s \\", shellQuote(k), shellQuote(v)))
	}
	if p.Body != "" {
		lines = append(lines, fmt.Sprintf("  <<< %s", shellQuote(p.Body)))
	} else {
		lines[len(lines)-1] = strings.TrimSuffix(lines[len(lines)-1], " \\")
	}
	return strings.Join(lines, "\n")
}

func buildWgetSnippet(p requestParts) string {
	lines := []string{"wget \\"}
	lines = append(lines, fmt.Sprintf("  --method=%s \\", p.Method))
	for k, v := range p.Headers {
		lines = append(lines, fmt.Sprintf("  --header=%s \\", shellQuote(k+": "+v)))
	}
	if p.Body != "" {
		lines = append(lines, fmt.Sprintf("  --body-data=%s \\", shellQuote(p.Body)))
	}
	lines = append(lines, fmt.Sprintf("  %s", shellQuote(p.URL)))
	return strings.Join(lines, "\n")
}

func buildJavaScriptSnippet(p requestParts) string {
	var headerLines []string
	for k, v := range p.Headers {
		headerLines = append(headerLines, fmt.Sprintf("    %q: %q,", k, v))
	}
	headersBlock := "{\n" + strings.Join(headerLines, "\n") + "\n  }"
	if p.Body != "" {
		bodyEscaped := strings.ReplaceAll(p.Body, "`", "\\`")
		return fmt.Sprintf("fetch(%q, {\n  method: %q,\n  headers: %s,\n  body: `%s`,\n})", p.URL, p.Method, headersBlock, bodyEscaped)
	}
	return fmt.Sprintf("fetch(%q, {\n  method: %q,\n  headers: %s,\n})", p.URL, p.Method, headersBlock)
}

func buildPythonSnippet(p requestParts) string {
	var headerLines []string
	for k, v := range p.Headers {
		headerLines = append(headerLines, fmt.Sprintf("    %q: %q,", k, v))
	}
	headersBlock := "{\n" + strings.Join(headerLines, "\n") + "\n}"
	if p.Body != "" {
		return fmt.Sprintf("import requests\n\nresponse = requests.request(\n    method=%q,\n    url=%q,\n    headers=%s,\n    data=%q,\n)\nprint(response.text)",
			strings.ToUpper(p.Method), p.URL, headersBlock, p.Body)
	}
	return fmt.Sprintf("import requests\n\nresponse = requests.request(\n    method=%q,\n    url=%q,\n    headers=%s,\n)\nprint(response.text)",
		strings.ToUpper(p.Method), p.URL, headersBlock)
}

func buildGoSnippet(p requestParts) string {
	if p.Body != "" {
		return fmt.Sprintf(`package main

import (
	"fmt"
	"net/http"
	"strings"
)

func main() {
	body := strings.NewReader(%q)
	req, err := http.NewRequest(%q, %q, body)
	if err != nil {
		panic(err)
	}
%s
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	fmt.Println(resp.Status)
}`,
			p.Body, p.Method, p.URL, goHeaderLines(p.Headers))
	}
	return fmt.Sprintf(`package main

import (
	"fmt"
	"net/http"
)

func main() {
	req, err := http.NewRequest(%q, %q, nil)
	if err != nil {
		panic(err)
	}
%s
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	fmt.Println(resp.Status)
}`, p.Method, p.URL, goHeaderLines(p.Headers))
}

func goHeaderLines(headers map[string]string) string {
	var lines []string
	for k, v := range headers {
		lines = append(lines, fmt.Sprintf("\treq.Header.Set(%q, %q)", k, v))
	}
	return strings.Join(lines, "\n")
}
