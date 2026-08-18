package request

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/imaanmzr/postchi/backend/internal/shared/config"
)

func TestNormalizeBodySpecJSONMode(t *testing.T) {
	got := normalizeBodySpec(BodySpec{Mode: "json", Raw: `{"otpCode":"053224"}`})
	if got.Mode != "raw" {
		t.Fatalf("mode=%q want raw", got.Mode)
	}
	if got.RawLang != "json" {
		t.Fatalf("raw_lang=%q want json", got.RawLang)
	}
}

func TestExecuteJSONBodyMode(t *testing.T) {
	var receivedBody string
	var contentType string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		receivedBody = string(b)
		contentType = r.Header.Get("Content-Type")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	cfg := &config.Config{RequestTimeout: 0, MaxResponseBytes: 1 << 20}
	exec := NewExecutor(cfg, nil, nil)
	result, err := exec.Execute(context.Background(), Model{
		Method: "POST",
		URL:    srv.URL,
		Body: BodySpec{
			Mode: "json",
			Raw:  `{"phoneNumber":"09912260563","otpCode":"767692"}`,
		},
	}, map[string]string{}, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if result.StatusCode != http.StatusOK {
		t.Fatalf("status=%d body=%s err=%s", result.StatusCode, result.Body, result.Error)
	}
	if receivedBody != `{"phoneNumber":"09912260563","otpCode":"767692"}` {
		t.Fatalf("body=%q", receivedBody)
	}
	if contentType != "application/json" {
		t.Fatalf("content-type=%q", contentType)
	}
}
