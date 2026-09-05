package gitbranches

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/imaanmzr/postchi/backend/internal/shared/gitrepo"
)

func TestParseListOptions(t *testing.T) {
	req := &http.Request{URL: &url.URL{RawQuery: "search=dev&limit=50&refresh=true"}}
	opts := ParseListOptions(req)
	if opts.Search != "dev" || opts.Limit != 50 || !opts.Refresh {
		t.Fatalf("unexpected options: %+v", opts)
	}
}

func TestHTTPStatus(t *testing.T) {
	err := &gitrepo.Error{Kind: gitrepo.ErrorInvalid, Message: "bad"}
	if HTTPStatus(err) != 400 {
		t.Fatal("expected 400 for invalid")
	}
}
