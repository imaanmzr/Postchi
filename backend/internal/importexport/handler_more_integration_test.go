package importexport

import (
	"archive/zip"
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/imaanmzr/postchi/backend/internal/auth"
	appdb "github.com/imaanmzr/postchi/backend/internal/db"
	"github.com/imaanmzr/postchi/backend/internal/shared/crypto"
	"github.com/imaanmzr/postchi/backend/internal/testutil"
)

func TestAdditionalImportHandlers(t *testing.T) {
	ctx := context.Background()
	pool := requireIntegrationDB(t)
	userID, wsID := testutil.SeedWorkspace(t, ctx, pool)
	store := appdb.NewStore(pool)
	cryptoSvc, _ := crypto.NewService("0123456789abcdef0123456789abcdef")
	h := NewHandler(store, cryptoSvc)

	t.Run("import opencollection yaml", func(t *testing.T) {
		data, _ := os.ReadFile(filepath.Join("testdata", "n2.yml"))
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/import/opencollection?workspace_id="+wsID.String(), bytes.NewReader(data))
		req = req.WithContext(context.WithValue(req.Context(), auth.UserIDKey, userID.String()))
		h.ImportOpenCollection(rr, req)
		if rr.Code != http.StatusCreated {
			t.Fatalf("status %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("import openapi yaml", func(t *testing.T) {
		data, _ := os.ReadFile(filepath.Join("testdata", "openapi", "minimal.yaml"))
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/import/openapi?workspace_id="+wsID.String(), bytes.NewReader(data))
		req = req.WithContext(context.WithValue(req.Context(), auth.UserIDKey, userID.String()))
		h.ImportOpenAPI(rr, req)
		if rr.Code != http.StatusCreated {
			t.Fatalf("status %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("import bruno zip", func(t *testing.T) {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("file", "collection.zip")
		_, _ = part.Write(buildZipArchive(map[string]string{
			"collection.bru": "meta {\n  name: Zip API\n  type: collection\n}\n",
			"ping.bru":       "meta {\n  name: Ping\n  type: http\n}\nget {\n  url: https://example.com/ping\n}\n",
		}))
		_ = writer.Close()
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/import/bruno?workspace_id="+wsID.String(), body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req = req.WithContext(context.WithValue(req.Context(), auth.UserIDKey, userID.String()))
		h.ImportBruno(rr, req)
		if rr.Code != http.StatusCreated {
			t.Fatalf("status %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("list bruno sources", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/workspaces/"+wsID.String()+"/bruno-sources", nil)
		routeContext := chi.NewRouteContext()
		routeContext.URLParams.Add("id", wsID.String())
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeContext))
		h.ListBrunoSources(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status %d: %s", rr.Code, rr.Body.String())
		}
	})
}

func buildZipArchive(files map[string]string) []byte {
	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)
	for name, content := range files {
		f, _ := zw.Create(name)
		_, _ = f.Write([]byte(content))
	}
	_ = zw.Close()
	return buf.Bytes()
}
