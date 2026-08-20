package hash

import (
	"testing"

	"github.com/imaanmzr/postchi/backend/internal/importexport/model"
)

func TestRequestHashStable(t *testing.T) {
	req := model.Request{Name: "Ping", Method: "GET", URL: "https://example.com"}
	a := Request(req)
	b := Request(req)
	if a != b {
		t.Fatalf("hash mismatch: %q vs %q", a, b)
	}
	req.Method = "POST"
	if Request(req) == a {
		t.Fatal("expected hash to change when method changes")
	}
}
