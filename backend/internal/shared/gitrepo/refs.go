package gitrepo

import (
	"strings"
	"unicode"
)

// SanitizeBranchName validates a branch/ref name using a conservative git ref-name subset.
func SanitizeBranchName(name string) (string, bool) {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 255 {
		return "", false
	}
	if strings.HasPrefix(name, "-") || strings.HasSuffix(name, ".") || strings.Contains(name, "..") {
		return "", false
	}
	for _, r := range name {
		if r <= 0x1f || r == 0x7f || unicode.IsSpace(r) {
			return "", false
		}
	}
	if strings.ContainsAny(name, "~^:?*\\[]") {
		return "", false
	}
	return name, true
}

func ValidateBranchName(name string) error {
	if _, ok := SanitizeBranchName(name); !ok {
		return invalidError("invalid branch name")
	}
	return nil
}
