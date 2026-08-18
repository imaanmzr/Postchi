package request

import "testing"

func TestInterpolate(t *testing.T) {
	vars := map[string]string{"host": "api.example.com", "token": "abc"}
	got := Interpolate("https://{{host}}/v1?key={{token}}", vars)
	want := "https://api.example.com/v1?key=abc"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestInterpolateMissing(t *testing.T) {
	got := Interpolate("{{missing}}", map[string]string{})
	if got != "{{missing}}" {
		t.Errorf("got %q", got)
	}
}
