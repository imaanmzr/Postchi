package gitrepo

import "testing"

func TestSanitizeBranchName(t *testing.T) {
	valid := []string{"main", "develop", "feature/PAY-482", "uat/release-1", "bugfix/foo_bar"}
	for _, name := range valid {
		if got, ok := SanitizeBranchName(name); !ok || got != name {
			t.Fatalf("expected %q to be valid, got %q ok=%v", name, got, ok)
		}
	}

	invalid := []string{"", "-bad", "bad.", "bad..name", "has space", "bad~name", "bad^name", "bad:name", "bad?name", "bad*name", "bad\\name", "bad[name"}
	for _, name := range invalid {
		if got, ok := SanitizeBranchName(name); ok {
			t.Fatalf("expected %q to be invalid, got %q", name, got)
		}
	}
}
