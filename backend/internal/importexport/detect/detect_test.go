package detect

import "testing"

func TestDetectFormat(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		content  string
		want     Format
	}{
		{name: "bruno extension", filename: "req.bru", content: "", want: FormatBruno},
		{name: "postman json", filename: "c.json", content: `{"info":{"schema":"https://schema.getpostman.com/json/collection/v2.1.0/collection.json"}}`, want: FormatPostman},
		{name: "opencollection json", filename: "c.json", content: `{"opencollection":"1.0.0","info":{"name":"x"}}`, want: FormatOpenCollection},
		{name: "openapi json", filename: "c.json", content: `{"openapi":"3.0.0","info":{"title":"x"}}`, want: FormatOpenAPI},
		{name: "openapi yaml content", filename: "spec.yml", content: "openapi: 3.0.0\ninfo:\n  title: x\n", want: FormatOpenAPI},
		{name: "opencollection yaml content", filename: "spec.yml", content: "opencollection: 1.0.0\ninfo:\n  name: x\n", want: FormatOpenCollection},
		{name: "yaml extension fallback", filename: "unknown.yml", content: "foo: bar\n", want: FormatOpenCollection},
		{name: "json extension fallback", filename: "unknown.json", content: "{}", want: FormatPostman},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := DetectFormat(tc.filename, []byte(tc.content)); got != tc.want {
				t.Fatalf("DetectFormat() = %q, want %q", got, tc.want)
			}
		})
	}
}
