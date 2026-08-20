package importexport

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/imaanmzr/postchi/backend/internal/auth"
)

func TestImportHandlersUnauthorized(t *testing.T) {
	h := NewHandler(nil, nil)
	wsID := uuid.New().String()
	cases := []struct {
		name string
		call func(http.ResponseWriter, *http.Request)
	}{
		{"postman", func(w http.ResponseWriter, r *http.Request) { h.ImportPostman(w, r) }},
		{"openapi", func(w http.ResponseWriter, r *http.Request) { h.ImportOpenAPI(w, r) }},
		{"opencollection", func(w http.ResponseWriter, r *http.Request) { h.ImportOpenCollection(w, r) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/api/import/"+tc.name+"?workspace_id="+wsID, bytes.NewReader([]byte("{}")))
			tc.call(rr, req)
			if rr.Code != http.StatusUnauthorized {
				t.Fatalf("status %d", rr.Code)
			}
		})
	}
}

func TestImportPostmanInvalidWorkspaceID(t *testing.T) {
	h := NewHandler(nil, nil)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/import/postman?workspace_id=not-a-uuid", bytes.NewReader([]byte("{}")))
	req = req.WithContext(context.WithValue(context.Background(), auth.UserIDKey, uuid.New().String()))
	h.ImportPostman(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status %d: %s", rr.Code, rr.Body.String())
	}
}

func TestImportBrunoUnauthorized(t *testing.T) {
	h := NewHandler(nil, nil)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/import/bruno?workspace_id="+uuid.New().String(), nil)
	h.ImportBruno(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status %d", rr.Code)
	}
}

func TestExportHandlersRequireCollectionID(t *testing.T) {
	h := NewHandler(nil, nil)
	rr := httptest.NewRecorder()
	h.ExportPostman(rr, httptest.NewRequest(http.MethodGet, "/api/export/postman", nil))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("postman export status %d", rr.Code)
	}
	rr = httptest.NewRecorder()
	h.ExportBruno(rr, httptest.NewRequest(http.MethodGet, "/api/export/bruno", nil))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("bruno export status %d", rr.Code)
	}
}

func TestWriteImportResultRejectsEmpty(t *testing.T) {
	h := NewHandler(nil, nil)
	rr := httptest.NewRecorder()
	h.writeImportResult(rr, ImportResult{})
	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status %d", rr.Code)
	}
}
