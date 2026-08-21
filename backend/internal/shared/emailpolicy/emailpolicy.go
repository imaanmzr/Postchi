package emailpolicy

import "strings"

// Domain returns the lowercase domain part of an email, or empty if invalid.
func Domain(email string) string {
	email = strings.TrimSpace(strings.ToLower(email))
	at := strings.LastIndex(email, "@")
	if at < 0 || at == len(email)-1 {
		return ""
	}
	return email[at+1:]
}

// Allowed reports whether email is permitted by the domain allowlist.
// An empty allowlist permits all domains.
func Allowed(email string, domains []string) bool {
	if len(domains) == 0 {
		return true
	}
	domain := Domain(email)
	if domain == "" {
		return false
	}
	for _, allowed := range domains {
		if domain == strings.ToLower(strings.TrimSpace(allowed)) {
			return true
		}
	}
	return false
}
