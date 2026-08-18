package request

import (
	"encoding/json"
)

var overridableFields = []string{
	"url", "method", "headers", "params", "path_vars", "body", "auth", "settings",
	"pre_request_script", "test_script",
}

func mergeWithTemplate(child, template Model) Model {
	out := child
	if child.TemplateID == nil || *child.TemplateID == "" {
		return out
	}
	overridden := map[string]bool{}
	for _, f := range child.OverriddenFields {
		overridden[f] = true
	}
	if !overridden["url"] {
		out.URL = template.URL
	}
	if !overridden["method"] {
		out.Method = template.Method
	}
	if !overridden["headers"] {
		out.Headers = template.Headers
	}
	if !overridden["params"] {
		out.Params = template.Params
	}
	if !overridden["path_vars"] {
		out.PathVars = template.PathVars
	}
	if !overridden["body"] {
		out.Body = template.Body
	}
	if !overridden["auth"] {
		out.Auth = template.Auth
	}
	if !overridden["settings"] {
		out.Settings = template.Settings
	}
	if !overridden["pre_request_script"] {
		out.PreRequestScript = template.PreRequestScript
	}
	if !overridden["test_script"] {
		out.TestScript = template.TestScript
	}
	return out
}

func diffOverriddenFields(child, incoming, template Model) []string {
	overridden := map[string]bool{}
	for _, f := range child.OverriddenFields {
		overridden[f] = true
	}
	merged := mergeWithTemplate(child, template)
	checks := map[string]func() bool{
		"url":              func() bool { return incoming.URL != merged.URL },
		"method":           func() bool { return incoming.Method != merged.Method },
		"headers":          func() bool { return !jsonEqual(incoming.Headers, merged.Headers) },
		"params":           func() bool { return !jsonEqual(incoming.Params, merged.Params) },
		"path_vars":        func() bool { return !jsonEqual(incoming.PathVars, merged.PathVars) },
		"body":             func() bool { return !jsonEqual(incoming.Body, merged.Body) },
		"auth":             func() bool { return !jsonEqual(incoming.Auth, merged.Auth) },
		"settings":         func() bool { return !jsonEqual(incoming.Settings, merged.Settings) },
		"pre_request_script": func() bool { return incoming.PreRequestScript != merged.PreRequestScript },
		"test_script":      func() bool { return incoming.TestScript != merged.TestScript },
	}
	for field, differs := range checks {
		if differs() {
			overridden[field] = true
		}
	}
	out := make([]string, 0, len(overridden))
	for _, f := range overridableFields {
		if overridden[f] {
			out = append(out, f)
		}
	}
	return out
}

func jsonEqual(a, b any) bool {
	ab, _ := json.Marshal(a)
	bb, _ := json.Marshal(b)
	return string(ab) == string(bb)
}

func applyOverrides(base Model, overrides map[string]json.RawMessage) (Model, []string) {
	out := base
	var overridden []string
	for field, raw := range overrides {
		switch field {
		case "url":
			_ = json.Unmarshal(raw, &out.URL)
			overridden = append(overridden, "url")
		case "method":
			_ = json.Unmarshal(raw, &out.Method)
			overridden = append(overridden, "method")
		case "headers":
			_ = json.Unmarshal(raw, &out.Headers)
			overridden = append(overridden, "headers")
		case "params":
			_ = json.Unmarshal(raw, &out.Params)
			overridden = append(overridden, "params")
		case "path_vars":
			_ = json.Unmarshal(raw, &out.PathVars)
			overridden = append(overridden, "path_vars")
		case "body":
			_ = json.Unmarshal(raw, &out.Body)
			overridden = append(overridden, "body")
		case "auth":
			_ = json.Unmarshal(raw, &out.Auth)
			overridden = append(overridden, "auth")
		case "settings":
			_ = json.Unmarshal(raw, &out.Settings)
			overridden = append(overridden, "settings")
		case "pre_request_script":
			_ = json.Unmarshal(raw, &out.PreRequestScript)
			overridden = append(overridden, "pre_request_script")
		case "test_script":
			_ = json.Unmarshal(raw, &out.TestScript)
			overridden = append(overridden, "test_script")
		case "name":
			_ = json.Unmarshal(raw, &out.Name)
		}
	}
	return out, overridden
}

func modelToMap(m Model) map[string]any {
	b, _ := json.Marshal(m)
	var out map[string]any
	_ = json.Unmarshal(b, &out)
	return out
}

func fieldValueEqual(field string, a, b Model) bool {
	switch field {
	case "url":
		return a.URL == b.URL
	case "method":
		return a.Method == b.Method
	case "headers":
		return jsonEqual(a.Headers, b.Headers)
	case "params":
		return jsonEqual(a.Params, b.Params)
	case "path_vars":
		return jsonEqual(a.PathVars, b.PathVars)
	case "body":
		return jsonEqual(a.Body, b.Body)
	case "auth":
		return jsonEqual(a.Auth, b.Auth)
	case "settings":
		return jsonEqual(a.Settings, b.Settings)
	case "pre_request_script":
		return a.PreRequestScript == b.PreRequestScript
	case "test_script":
		return a.TestScript == b.TestScript
	default:
		return true
	}
}

func removeFieldFromList(fields []string, field string) []string {
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		if f != field {
			out = append(out, f)
		}
	}
	return out
}

func containsField(fields []string, field string) bool {
	for _, f := range fields {
		if f == field {
			return true
		}
	}
	return false
}

func snapshotMergedChild(child, template Model) Model {
	merged := mergeWithTemplate(child, template)
	merged.TemplateID = nil
	merged.OverriddenFields = nil
	return merged
}
