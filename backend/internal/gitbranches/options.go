package gitbranches

import (
	"net/http"
	"strconv"
	"strings"
)

func ParseListOptions(r *http.Request) ListOptions {
	q := r.URL.Query()
	limit := 100
	if raw := strings.TrimSpace(q.Get("limit")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil {
			limit = n
		}
	}
	return ListOptions{
		Search:  strings.TrimSpace(q.Get("search")),
		Limit:   limit,
		Refresh: strings.EqualFold(q.Get("refresh"), "true") || q.Get("refresh") == "1",
	}
}
