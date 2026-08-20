package docsync

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/imaanmzr/postchi/backend/internal/db"
	"github.com/imaanmzr/postchi/backend/internal/db/sqlc"
	"github.com/imaanmzr/postchi/backend/internal/docsync/linkmatcher"
	"github.com/imaanmzr/postchi/backend/internal/shared/operationid"
)

type AutoLinkInput struct {
	WorkspaceID  uuid.UUID
	CollectionID *uuid.UUID
	LinkTemplate string
	PathPrefix   string
	Docs         []linkmatcher.Doc
	Requests     []linkmatcher.Request
}

type AutoLinkResult struct {
	AutoLinked int
	Ambiguous  int
	Skipped    int
}

func matchFrontmatterRequests(docs []linkmatcher.Doc, requests []linkmatcher.Request) []linkmatcher.Candidate {
	var out []linkmatcher.Candidate
	for _, doc := range docs {
		if len(doc.LinkedRequestNames) == 0 {
			continue
		}
		nameSet := make(map[string]struct{}, len(doc.LinkedRequestNames))
		for _, n := range doc.LinkedRequestNames {
			nameSet[n] = struct{}{}
		}
		for _, req := range requests {
			slug := linkmatcher.RequestSlug(req)
			if slug == "" {
				continue
			}
			if _, ok := nameSet[slug]; !ok {
				continue
			}
			out = append(out, linkmatcher.Candidate{
				DocID: doc.ID, RequestID: req.ID,
				Reason: "frontmatter_request", Confidence: "exact",
				Evidence: map[string]string{"request_slug": slug},
			})
		}
	}
	return out
}

func filterRequestsByCollection(requests []linkmatcher.Request, collectionID *uuid.UUID) []linkmatcher.Request {
	if collectionID == nil {
		return requests
	}
	cid := collectionID.String()
	filtered := make([]linkmatcher.Request, 0, len(requests))
	for _, req := range requests {
		if req.CollectionID == cid {
			filtered = append(filtered, req)
		}
	}
	return filtered
}

func buildCollectionMap(requests []linkmatcher.Request) map[string]linkmatcher.CollectionInfo {
	out := make(map[string]linkmatcher.CollectionInfo)
	for _, req := range requests {
		if req.CollectionID == "" {
			continue
		}
		if _, ok := out[req.CollectionID]; ok {
			continue
		}
		out[req.CollectionID] = linkmatcher.CollectionInfo{
			ID:   req.CollectionID,
			Name: req.CollectionName,
		}
	}
	return out
}

func (h *Handler) buildAutoLinkSkip(
	ctx context.Context,
	wsID uuid.UUID,
	docs []linkmatcher.Doc,
	reqByID map[string]linkmatcher.Request,
	refreshRejected bool,
) (func(docID, requestID string) bool, error) {
	manualRows, err := h.store.ListManualDocLinksByWorkspace(ctx, db.PGUUID(wsID))
	if err != nil {
		return nil, err
	}
	manualPairs := make(map[string]bool, len(manualRows))
	for _, row := range manualRows {
		key := db.FromPGUUID(row.WorkspaceDocID).String() + ":" + db.FromPGUUID(row.RequestID).String()
		manualPairs[key] = true
	}

	existing, _ := h.store.ListDocLinkSuggestionsForAnalyze(ctx, db.PGUUID(wsID))
	rejected := make(map[string]bool)
	for _, row := range existing {
		if row.Status == "rejected" && !refreshRejected {
			key := db.FromPGUUID(row.WorkspaceDocID).String() + ":" + db.FromPGUUID(row.RequestID).String()
			rejected[key] = true
		}
	}

	docByID := make(map[string]linkmatcher.Doc, len(docs))
	for _, d := range docs {
		docByID[d.ID] = d
	}

	return func(docID, requestID string) bool {
		key := docID + ":" + requestID
		if manualPairs[key] {
			return true
		}
		if rejected[key] {
			return true
		}
		doc := docByID[docID]
		req := reqByID[requestID]
		if operationid.Matches(doc.LinkedOperationIDs, operationid.AliasesForRequest(req.Method, req.URL, req.SourceOperationID)) {
			return true
		}
		reqSlug := linkmatcher.RequestSlug(req)
		for _, n := range doc.LinkedRequestNames {
			if n == reqSlug {
				return true
			}
		}
		return false
	}, nil
}

func (h *Handler) runAutoLink(ctx context.Context, input AutoLinkInput) (AutoLinkResult, error) {
	result := AutoLinkResult{}
	if len(input.Docs) == 0 || len(input.Requests) == 0 {
		return result, nil
	}

	reqByID := make(map[string]linkmatcher.Request, len(input.Requests))
	for _, req := range input.Requests {
		reqByID[req.ID] = req
	}

	skip, err := h.buildAutoLinkSkip(ctx, input.WorkspaceID, input.Docs, reqByID, false)
	if err != nil {
		return result, err
	}

	linked := make(map[string]bool)

	autoLink := func(c linkmatcher.Candidate) error {
		key := c.DocID + ":" + c.RequestID
		if linked[key] || skip(c.DocID, c.RequestID) {
			result.Skipped++
			return nil
		}
		docUUID, err := uuid.Parse(c.DocID)
		if err != nil {
			return err
		}
		reqUUID, err := uuid.Parse(c.RequestID)
		if err != nil {
			return err
		}
		if _, err := h.store.CreateManualDocLink(ctx, sqlc.CreateManualDocLinkParams{
			WorkspaceDocID: db.PGUUID(docUUID),
			RequestID:      db.PGUUID(reqUUID),
		}); err != nil {
			return err
		}
		linked[key] = true
		result.AutoLinked++
		return nil
	}

	suggestAmbiguous := func(c linkmatcher.Candidate, reason string) error {
		key := c.DocID + ":" + c.RequestID
		if linked[key] || skip(c.DocID, c.RequestID) {
			result.Skipped++
			return nil
		}
		docUUID, err := uuid.Parse(c.DocID)
		if err != nil {
			return err
		}
		reqUUID, err := uuid.Parse(c.RequestID)
		if err != nil {
			return err
		}
		c.Reason = reason
		if _, err := h.store.UpsertDocLinkSuggestion(ctx, sqlc.UpsertDocLinkSuggestionParams{
			WorkspaceID:    db.PGUUID(input.WorkspaceID),
			WorkspaceDocID: db.PGUUID(docUUID),
			RequestID:      db.PGUUID(reqUUID),
			Reason:         c.Reason,
			Confidence:     c.Confidence,
			Evidence:       linkmatcher.EvidenceJSON(c.Evidence),
		}); err != nil {
			return err
		}
		result.Ambiguous++
		return nil
	}

	// 1. Frontmatter request names — always auto-link (explicit intent).
	for _, c := range matchFrontmatterRequests(input.Docs, input.Requests) {
		if err := autoLink(c); err != nil {
			return result, err
		}
	}

	// 2. Path template — scoped to collection when set.
	scopedRequests := filterRequestsByCollection(input.Requests, input.CollectionID)
	if input.LinkTemplate != "" {
		collections := buildCollectionMap(scopedRequests)
		templateCandidates := linkmatcher.MatchPathTemplate(input.LinkTemplate, input.PathPrefix, input.Docs, scopedRequests, collections)
		auto, ambiguous := linkmatcher.PartitionUnique(templateCandidates)
		for _, c := range auto {
			if err := autoLink(c); err != nil {
				return result, err
			}
		}
		for _, c := range ambiguous {
			if err := suggestAmbiguous(c, "ambiguous_path_template"); err != nil {
				return result, err
			}
		}
	}

	// 3. Exact name — scoped to collection when set.
	exactCandidates := linkmatcher.MatchExactName(input.Docs, scopedRequests)
	auto, ambiguous := linkmatcher.PartitionUnique(exactCandidates)
	for _, c := range auto {
		if err := autoLink(c); err != nil {
			return result, err
		}
	}
	for _, c := range ambiguous {
		if err := suggestAmbiguous(c, "ambiguous_exact_name"); err != nil {
			return result, err
		}
	}

	return result, nil
}

func docRowsToLinkmatcher(docs []sqlc.ListWorkspaceDocsRow) []linkmatcher.Doc {
	out := make([]linkmatcher.Doc, 0, len(docs))
	for _, row := range docs {
		out = append(out, linkmatcher.Doc{
			ID:                 db.FromPGUUID(row.ID).String(),
			Slug:               row.Slug,
			Title:              row.Title,
			SourcePath:         row.SourcePath,
			ContentMD:          row.ContentMd,
			LinkedOperationIDs: row.LinkedOperationIds,
			LinkedRequestNames: row.LinkedRequestNames,
		})
	}
	return out
}

func docSourceRowsToLinkmatcher(docs []sqlc.ListWorkspaceDocsByDocSourceRow) []linkmatcher.Doc {
	out := make([]linkmatcher.Doc, 0, len(docs))
	for _, row := range docs {
		out = append(out, linkmatcher.Doc{
			ID:                 db.FromPGUUID(row.ID).String(),
			Slug:               row.Slug,
			Title:              row.Title,
			SourcePath:         row.SourcePath,
			ContentMD:          row.ContentMd,
			LinkedOperationIDs: row.LinkedOperationIds,
			LinkedRequestNames: row.LinkedRequestNames,
		})
	}
	return out
}

func requestRowsToLinkmatcher(reqRows []sqlc.ListRequestsByWorkspaceRow, collectionNames map[string]string) []linkmatcher.Request {
	out := make([]linkmatcher.Request, 0, len(reqRows))
	for _, row := range reqRows {
		cid := db.FromPGUUID(row.CollectionID).String()
		out = append(out, linkmatcher.Request{
			ID:                db.FromPGUUID(row.ID).String(),
			Name:              row.Name,
			Method:            row.Method,
			URL:               row.Url,
			SourceOperationID: row.SourceOperationID,
			CollectionID:      cid,
			CollectionName:    collectionNames[cid],
		})
	}
	return out
}

func linkTemplateFromConfig(config []byte) (template, pathPrefix string) {
	var cfg map[string]any
	if err := json.Unmarshal(config, &cfg); err != nil {
		return "", ""
	}
	if v, ok := cfg["link_template"].(string); ok {
		template = v
	}
	if v, ok := cfg["path_prefix"].(string); ok {
		pathPrefix = v
	}
	return template, pathPrefix
}

func collectionIDFromPG(colID pgtype.UUID) *uuid.UUID {
	if !colID.Valid {
		return nil
	}
	id := db.FromPGUUID(colID)
	return &id
}

func (h *Handler) autoLinkAfterDocSourceSync(ctx context.Context, wsID, sourceID uuid.UUID, collectionID pgtype.UUID, config map[string]any) (AutoLinkResult, error) {
	docRows, err := h.store.ListWorkspaceDocsByDocSource(ctx, sqlc.ListWorkspaceDocsByDocSourceParams{
		WorkspaceID: db.PGUUID(wsID),
		DocSourceID: db.PGUUID(sourceID),
	})
	if err != nil {
		return AutoLinkResult{}, err
	}
	reqRows, err := h.store.ListRequestsByWorkspace(ctx, db.PGUUID(wsID))
	if err != nil {
		return AutoLinkResult{}, err
	}
	collectionNames := map[string]string{}
	for _, row := range reqRows {
		collectionNames[db.FromPGUUID(row.CollectionID).String()] = ""
	}
	colRows, _ := h.store.ListCatalogCollections(ctx, db.PGUUID(wsID))
	for _, c := range colRows {
		collectionNames[db.FromPGUUID(c.ID).String()] = c.Name
	}
	cfgBytes, _ := json.Marshal(config)
	template, pathPrefix := linkTemplateFromConfig(cfgBytes)
	return h.runAutoLink(ctx, AutoLinkInput{
		WorkspaceID:  wsID,
		CollectionID: collectionIDFromPG(collectionID),
		LinkTemplate: template,
		PathPrefix:   pathPrefix,
		Docs:         docSourceRowsToLinkmatcher(docRows),
		Requests:     requestRowsToLinkmatcher(reqRows, collectionNames),
	})
}

func (h *Handler) runWorkspaceAnalyzeAutoLink(ctx context.Context, wsID uuid.UUID) (AutoLinkResult, error) {
	docRows, err := h.store.ListWorkspaceDocs(ctx, db.PGUUID(wsID))
	if err != nil {
		return AutoLinkResult{}, err
	}
	reqRows, err := h.store.ListRequestsByWorkspace(ctx, db.PGUUID(wsID))
	if err != nil {
		return AutoLinkResult{}, err
	}
	collectionNames := map[string]string{}
	for _, row := range reqRows {
		collectionNames[db.FromPGUUID(row.CollectionID).String()] = ""
	}
	colRows, _ := h.store.ListCatalogCollections(ctx, db.PGUUID(wsID))
	for _, c := range colRows {
		collectionNames[db.FromPGUUID(c.ID).String()] = c.Name
	}
	requests := requestRowsToLinkmatcher(reqRows, collectionNames)
	docs := docRowsToLinkmatcher(docRows)

	total, err := h.runAutoLink(ctx, AutoLinkInput{
		WorkspaceID: wsID,
		Docs:        docs,
		Requests:    requests,
	})
	if err != nil {
		return total, err
	}

	sourceRows, err := h.store.ListDocSources(ctx, db.PGUUID(wsID))
	if err != nil {
		return total, err
	}
	for _, src := range sourceRows {
		template, pathPrefix := linkTemplateFromConfig(src.Config)
		if template == "" && !src.CollectionID.Valid {
			continue
		}
		sourceDocs, err := h.store.ListWorkspaceDocsByDocSource(ctx, sqlc.ListWorkspaceDocsByDocSourceParams{
			WorkspaceID: db.PGUUID(wsID),
			DocSourceID: src.ID,
		})
		if err != nil {
			return total, err
		}
		if len(sourceDocs) == 0 {
			continue
		}
		part, err := h.runAutoLink(ctx, AutoLinkInput{
			WorkspaceID:  wsID,
			CollectionID: collectionIDFromPG(src.CollectionID),
			LinkTemplate: template,
			PathPrefix:   pathPrefix,
			Docs:         docSourceRowsToLinkmatcher(sourceDocs),
			Requests:     requests,
		})
		if err != nil {
			return total, err
		}
		total.AutoLinked += part.AutoLinked
		total.Ambiguous += part.Ambiguous
		total.Skipped += part.Skipped
	}
	return total, nil
}
