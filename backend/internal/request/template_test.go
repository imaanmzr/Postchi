package request

import "testing"

func TestMergeWithTemplate(t *testing.T) {
	template := Model{
		URL:    "https://api.example.com",
		Method: "GET",
		Headers: []KVPair{{Key: "X-Template", Value: "yes", Enabled: true}},
	}
	child := Model{
		TemplateID:       strPtr("tpl"),
		OverriddenFields: []string{"body"},
		Body:             BodySpec{Mode: "raw", Raw: "custom"},
	}
	merged := mergeWithTemplate(child, template)
	if merged.URL != template.URL {
		t.Errorf("url should inherit: %s", merged.URL)
	}
	if merged.Body.Raw != "custom" {
		t.Errorf("body should stay overridden")
	}
	if len(merged.Headers) != 1 || merged.Headers[0].Key != "X-Template" {
		t.Errorf("headers should inherit from template")
	}
}

func TestDiffOverriddenFields(t *testing.T) {
	template := Model{URL: "https://a.com", Method: "GET"}
	child := Model{TemplateID: strPtr("t"), OverriddenFields: []string{}}
	incoming := Model{URL: "https://b.com", Method: "GET"}
	fields := diffOverriddenFields(child, incoming, template)
	if len(fields) != 1 || fields[0] != "url" {
		t.Fatalf("expected url overridden, got %v", fields)
	}
}

func strPtr(s string) *string { return &s }
