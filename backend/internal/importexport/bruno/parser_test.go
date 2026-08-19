package bruno

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseSampleRequest(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "testdata", "bruno", "sample.bru"))
	if err != nil {
		t.Fatal(err)
	}
	p := Parse(string(data))
	req := ToRequest(p)
	if req.URL != "{{baseUrl}}/users" {
		t.Fatalf("url=%q", req.URL)
	}
}

func TestParsePostWithJSONBody(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "testdata", "bruno", "post_with_body.bru"))
	if err != nil {
		t.Fatal(err)
	}
	p := Parse(string(data))
	req := ToRequest(p)
	if req.Method != "POST" {
		t.Fatalf("method=%q", req.Method)
	}
	if req.BodyType != "json" {
		t.Fatalf("body type=%q", req.BodyType)
	}
	if !strings.Contains(req.Body, "Example Group") {
		t.Fatalf("body=%q", req.Body)
	}
	exported := ExportRequest(req)
	reParsed := Parse(exported)
	reReq := ToRequest(reParsed)
	if reReq.BodyType != "json" || !strings.Contains(reReq.Body, "Example Group") {
		t.Fatalf("round-trip body lost: type=%q body=%q", reReq.BodyType, reReq.Body)
	}
}
