package importexport

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/imaanmzr/postchi/backend/internal/importexport/model"
	"github.com/imaanmzr/postchi/backend/internal/shared/gitrepo"
)

func TestParseCurl(t *testing.T) {
	method, url, headers, body := parseCurl(`curl -X POST 'https://example.com/api' -H 'Accept: application/json' -d '{"ok":true}'`)
	if method != "POST" || url != "https://example.com/api" {
		t.Fatalf("method=%s url=%s", method, url)
	}
	if len(headers) != 1 || body.Mode != "raw" {
		t.Fatalf("headers=%+v body=%+v", headers, body)
	}
}

func TestWriteGitImportErrorMapping(t *testing.T) {
	cases := []struct {
		err    error
		status int
	}{
		{err: &gitrepo.Error{Kind: gitrepo.ErrorInvalid}, status: http.StatusBadRequest},
		{err: &gitrepo.Error{Kind: gitrepo.ErrorAuthentication}, status: http.StatusUnauthorized},
		{err: &gitrepo.Error{Kind: gitrepo.ErrorRateLimited}, status: http.StatusTooManyRequests},
		{err: &gitrepo.Error{Kind: gitrepo.ErrorTimeout}, status: http.StatusGatewayTimeout},
		{err: &gitrepo.Error{Kind: gitrepo.ErrorLimit}, status: http.StatusRequestEntityTooLarge},
	}
	for _, tc := range cases {
		rr := httptest.NewRecorder()
		writeGitImportError(rr, tc.err)
		if rr.Code != tc.status {
			t.Fatalf("%v => status %d body %s", tc.err, rr.Code, rr.Body.String())
		}
	}
}

func TestImportResultTotal(t *testing.T) {
	result := ImportResult{Collections: 2, Requests: 3, Environments: 1}
	if result.Total() != 6 {
		t.Fatalf("total = %d", result.Total())
	}
}

func TestSourceSyncMutex(t *testing.T) {
	h := NewHandler(nil, nil)
	id := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	a := h.sourceSyncMutex(id)
	b := h.sourceSyncMutex(id)
	if a != b {
		t.Fatal("expected same mutex for source id")
	}
}

func TestCollectAllSourcePaths(t *testing.T) {
	cols := []model.Collection{{
		SourcePath: "api.postman_collection.json",
		Requests:   []model.Request{{SourcePath: "api.postman_collection.json#Health"}},
	}}
	cPaths, rPaths := collectAllSourcePaths(cols)
	if len(cPaths) != 1 || len(rPaths) != 1 {
		t.Fatalf("paths = %v %v", cPaths, rPaths)
	}
}

func TestFilepathHelpers(t *testing.T) {
	if got := filepathBaseName("collections/api"); got != "api" {
		t.Fatalf("base = %q", got)
	}
	if got := filepathExt("spec.yaml"); got != ".yaml" {
		t.Fatalf("ext = %q", got)
	}
}
