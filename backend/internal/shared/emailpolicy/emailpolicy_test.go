package emailpolicy

import "testing"

func TestDomain(t *testing.T) {
	tests := []struct {
		email string
		want  string
	}{
		{"user@Company.com", "company.com"},
		{"  user@example.com  ", "example.com"},
		{"invalid", ""},
		{"user@", ""},
	}
	for _, tc := range tests {
		if got := Domain(tc.email); got != tc.want {
			t.Errorf("Domain(%q) = %q, want %q", tc.email, got, tc.want)
		}
	}
}

func TestAllowed(t *testing.T) {
	allowed := []string{"company.com", "subsidiary.com"}
	if !Allowed("a@company.com", allowed) {
		t.Fatal("expected company.com allowed")
	}
	if Allowed("a@other.com", allowed) {
		t.Fatal("expected other.com rejected")
	}
	if !Allowed("a@any.com", nil) {
		t.Fatal("empty allowlist should permit all")
	}
	if !Allowed("a@any.com", []string{}) {
		t.Fatal("empty allowlist should permit all")
	}
}
