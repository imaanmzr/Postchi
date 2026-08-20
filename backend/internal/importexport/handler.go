package importexport

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"

	"github.com/google/uuid"

	"github.com/imaanmzr/postchi/backend/internal/auth"
	"github.com/imaanmzr/postchi/backend/internal/db"
	"github.com/imaanmzr/postchi/backend/internal/db/sqlc"
	"github.com/imaanmzr/postchi/backend/internal/importexport/bruno"
	"github.com/imaanmzr/postchi/backend/internal/importexport/model"
	openapiimport "github.com/imaanmzr/postchi/backend/internal/importexport/openapi"
	ocimport "github.com/imaanmzr/postchi/backend/internal/importexport/opencollection"
	"github.com/imaanmzr/postchi/backend/internal/request"
	"github.com/imaanmzr/postchi/backend/internal/shared/crypto"
	"github.com/imaanmzr/postchi/backend/internal/shared/domain"
	"github.com/imaanmzr/postchi/backend/internal/shared/respond"
)

type Handler struct {
	store        *db.Store
	crypto       *crypto.Service
	sourceSyncMu sync.Map
}

func NewHandler(store *db.Store, cryptoSvc *crypto.Service) *Handler {
	return &Handler{store: store, crypto: cryptoSvc}
}

func parseWorkspaceID(r *http.Request) (uuid.UUID, error) {
	wsID, err := uuid.Parse(r.URL.Query().Get("workspace_id"))
	if err != nil {
		return uuid.Nil, err
	}
	return wsID, nil
}

func (h *Handler) writeImportResult(w http.ResponseWriter, result ImportResult) {
	if result.Requests == 0 {
		respond.Error(w, http.StatusUnprocessableEntity, "import produced no requests")
		return
	}
	respond.JSON(w, http.StatusCreated, result)
}

func (h *Handler) ImportPostman(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.UserIDFromContext(r.Context())
	if err != nil {
		respond.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	wsID, err := parseWorkspaceID(r)
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "workspace_id required")
		return
	}
	if err := h.validateWorkspaceEditor(r.Context(), wsID, userID); err != nil {
		respond.Error(w, http.StatusForbidden, err.Error())
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid body")
		return
	}
	parsed, err := ParsePostmanWithWarnings(body)
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid postman collection: "+err.Error())
		return
	}
	_, result, err := h.persistCollection(r.Context(), wsID, userID, parsed.Collection, nil)
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "import failed: "+err.Error())
		return
	}
	result.Warnings = parsed.Warnings
	h.writeImportResult(w, result)
}

func (h *Handler) ImportOpenAPI(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.UserIDFromContext(r.Context())
	if err != nil {
		respond.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	wsID, err := parseWorkspaceID(r)
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "workspace_id required")
		return
	}
	if err := h.validateWorkspaceEditor(r.Context(), wsID, userID); err != nil {
		respond.Error(w, http.StatusForbidden, err.Error())
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid body")
		return
	}
	var meta struct {
		Name string `json:"name"`
	}
	_ = json.Unmarshal(body, &meta)
	col, err := openapiimport.Parse(body, meta.Name)
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid openapi spec: "+err.Error())
		return
	}
	_, result, err := h.persistCollection(r.Context(), wsID, userID, col, nil)
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "import failed: "+err.Error())
		return
	}
	h.writeImportResult(w, result)
}

func (h *Handler) ImportOpenCollection(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.UserIDFromContext(r.Context())
	if err != nil {
		respond.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	wsID, err := parseWorkspaceID(r)
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "workspace_id required")
		return
	}
	if err := h.validateWorkspaceEditor(r.Context(), wsID, userID); err != nil {
		respond.Error(w, http.StatusForbidden, err.Error())
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid body")
		return
	}
	col, err := ocimport.Parse(body)
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid opencollection file: "+err.Error())
		return
	}
	_, result, err := h.persistCollection(r.Context(), wsID, userID, col, nil)
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "import failed: "+err.Error())
		return
	}
	h.writeImportResult(w, result)
}

func (h *Handler) ImportBruno(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.UserIDFromContext(r.Context())
	if err != nil {
		respond.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	wsID, err := parseWorkspaceID(r)
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "workspace_id required")
		return
	}
	if err := h.validateWorkspaceEditor(r.Context(), wsID, userID); err != nil {
		respond.Error(w, http.StatusForbidden, err.Error())
		return
	}
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		respond.Error(w, http.StatusBadRequest, "expected multipart upload")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "file required")
		return
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "failed to read file")
		return
	}

	var result ImportResult
	if strings.HasSuffix(strings.ToLower(header.Filename), ".zip") {
		_, result, err = h.importBrunoZip(r.Context(), wsID, userID, data)
	} else {
		col, parseErr := parseBrunoSingleFile(data, header.Filename)
		if parseErr != nil {
			respond.Error(w, http.StatusBadRequest, parseErr.Error())
			return
		}
		_, result, err = h.persistCollection(r.Context(), wsID, userID, col, nil)
	}
	if err != nil {
		if strings.Contains(err.Error(), "collection.bru not found") {
			respond.Error(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
		respond.Error(w, http.StatusInternalServerError, "bruno import failed: "+err.Error())
		return
	}
	h.writeImportResult(w, result)
}

func (h *Handler) importBrunoZip(ctx context.Context, wsID, userID uuid.UUID, data []byte) (uuid.UUID, ImportResult, error) {
	rootCol, err := parseBrunoZip(data)
	if err != nil {
		return uuid.Nil, ImportResult{}, err
	}
	return h.persistCollection(ctx, wsID, userID, rootCol, nil)
}

func (h *Handler) ExportPostman(w http.ResponseWriter, r *http.Request) {
	colID, err := uuid.Parse(r.URL.Query().Get("collection_id"))
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "collection_id required")
		return
	}
	col, err := h.loadNormalized(r.Context(), colID)
	if err != nil {
		respond.Error(w, http.StatusNotFound, "collection not found")
		return
	}
	respond.JSON(w, http.StatusOK, ExportPostmanCollection(col))
}

func (h *Handler) ExportBruno(w http.ResponseWriter, r *http.Request) {
	colID, err := uuid.Parse(r.URL.Query().Get("collection_id"))
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "collection_id required")
		return
	}
	col, err := h.loadNormalized(r.Context(), colID)
	if err != nil {
		respond.Error(w, http.StatusNotFound, "collection not found")
		return
	}
	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)
	meta, _ := zw.Create("collection.bru")
	_, _ = meta.Write([]byte(bruno.ExportCollectionMeta(col.Name, specToBruVars(col.Variables))))
	for _, req := range col.Requests {
		f, _ := zw.Create(req.Name + ".bru")
		_, _ = f.Write([]byte(bruno.ExportRequest(bruFromNorm(req))))
	}
	_ = zw.Close()
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="`+col.Name+`.zip"`)
	w.Write(buf.Bytes())
}

func (h *Handler) ImportCurl(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.UserIDFromContext(r.Context())
	if err != nil {
		respond.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req struct {
		Command      string `json:"command"`
		CollectionID string `json:"collection_id"`
		Name         string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid body")
		return
	}
	method, urlStr, headers, body := parseCurl(req.Command)
	if urlStr == "" {
		respond.Error(w, http.StatusBadRequest, "could not parse curl command")
		return
	}
	colID, err := uuid.Parse(req.CollectionID)
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "collection_id required")
		return
	}
	if err := h.validateCollectionInWorkspace(r.Context(), colID, userID); err != nil {
		respond.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.Name == "" {
		req.Name = "Imported curl"
	}
	headersB, _ := json.Marshal(headers)
	bodyB, _ := json.Marshal(body)
	requestID, err := h.store.ImportCurlRequest(r.Context(), sqlc.ImportCurlRequestParams{
		CollectionID: db.PGUUID(colID),
		Name:         req.Name,
		Method:       method,
		Url:          urlStr,
		Headers:      headersB,
		Body:         bodyB,
		CreatedBy:    db.PGUUID(userID),
	})
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "import failed: "+err.Error())
		return
	}
	h.writeImportResult(w, ImportResult{
		RequestID:   db.FromPGUUID(requestID).String(),
		Collections: 0,
		Requests:    1,
	})
}

var curlMethodRe = regexp.MustCompile(`-X\s+(\w+)`)
var curlURLRe = regexp.MustCompile(`'(https?://[^']+)'|"(https?://[^"]+)"|(https?://\S+)`)
var curlHeaderRe = regexp.MustCompile(`-H\s+'([^']+)'|-H\s+"([^"]+)"`)

func parseCurl(cmd string) (method, url string, headers []request.KVPair, body request.BodySpec) {
	method = "GET"
	body = request.BodySpec{Mode: "none"}
	if m := curlMethodRe.FindStringSubmatch(cmd); len(m) > 1 {
		method = strings.ToUpper(m[1])
	}
	if m := curlURLRe.FindStringSubmatch(cmd); len(m) > 0 {
		for _, g := range m[1:] {
			if g != "" {
				url = strings.TrimSuffix(strings.TrimSuffix(g, "'"), `"`)
				break
			}
		}
	}
	for _, m := range curlHeaderRe.FindAllStringSubmatch(cmd, -1) {
		h := m[1]
		if h == "" {
			h = m[2]
		}
		parts := strings.SplitN(h, ":", 2)
		if len(parts) == 2 {
			headers = append(headers, request.KVPair{Key: strings.TrimSpace(parts[0]), Value: strings.TrimSpace(parts[1]), Enabled: true})
		}
	}
	if idx := strings.Index(cmd, "-d "); idx >= 0 {
		body = request.BodySpec{Mode: "raw", Raw: strings.TrimSpace(cmd[idx+3:]), RawLang: "json"}
		method = "POST"
	}
	return method, url, headers, body
}

func (h *Handler) loadNormalized(ctx context.Context, colID uuid.UUID) (model.Collection, error) {
	var col model.Collection
	row, err := h.store.GetCollectionForExport(ctx, db.PGUUID(colID))
	if err != nil {
		return col, err
	}
	col.Name = row.Name
	col.Description = row.Description
	col.PreRequestScript = row.PreRequestScript
	col.TestScript = row.TestScript
	col.Variables = domain.ParseVariablesSpec(row.Variables)
	_ = json.Unmarshal(row.Headers, &col.Headers)
	_ = json.Unmarshal(row.Auth, &col.Auth)

	reqRows, err := h.store.ListRequestsForExport(ctx, db.PGUUID(colID))
	if err != nil {
		return col, err
	}
	for _, reqRow := range reqRows {
		var req model.Request
		req.Name = reqRow.Name
		req.Method = reqRow.Method
		req.URL = reqRow.Url
		req.PreRequestScript = reqRow.PreRequestScript
		req.TestScript = reqRow.TestScript
		req.SortOrder = int(reqRow.SortOrder)
		req.Description = reqRow.Description
		_ = json.Unmarshal(reqRow.Headers, &req.Headers)
		_ = json.Unmarshal(reqRow.Body, &req.Body)
		col.Requests = append(col.Requests, req)
	}
	childIDs, err := h.store.ListChildCollectionIDs(ctx, db.PGUUID(colID))
	if err != nil {
		return col, err
	}
	for _, cid := range childIDs {
		child, _ := h.loadNormalized(ctx, db.FromPGUUID(cid))
		col.Children = append(col.Children, child)
	}
	return col, nil
}
