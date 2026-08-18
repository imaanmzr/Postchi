package bruno

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseSampleRequest(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "testdata", "bruno", "sample.bru"))
	if err != nil {
		t.Fatal(err)
	}
	p := Parse(string(data))
	for k, v := range p.Sections {
		t.Logf("section %q: %q", k, v)
	}
	req := ToRequest(p)
	if req.URL == "" {
		t.Fatalf("empty url, sections=%v", p.Sections)
	}
}
