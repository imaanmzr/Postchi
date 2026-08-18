package docsync

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/imaanmzr/postchi/backend/internal/db"
	"github.com/imaanmzr/postchi/backend/internal/shared/respond"
)

type GraphNode struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Type  string `json:"type"` // "doc" | "operation" | "request"
}

type GraphEdge struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Type   string `json:"type"` // "link" | "operation" | "manual"
}

type DocGraph struct {
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
}

type ManualGraphLink struct {
	DocSlug     string
	RequestID   string
	RequestName string
}

func buildDocGraph(docs []WorkspaceDoc, manual []ManualGraphLink) DocGraph {
	idx := buildDocIndex(docs)
	nodeSeen := make(map[string]struct{})
	edgeSeen := make(map[string]struct{})
	var nodes []GraphNode
	var edges []GraphEdge

	addNode := func(id, label, typ string) {
		if id == "" {
			return
		}
		key := typ + ":" + id
		if _, ok := nodeSeen[key]; ok {
			return
		}
		nodeSeen[key] = struct{}{}
		nodes = append(nodes, GraphNode{ID: id, Label: label, Type: typ})
	}
	addEdge := func(source, target, typ string) {
		if source == "" || target == "" {
			return
		}
		key := source + "->" + target + ":" + typ
		if _, ok := edgeSeen[key]; ok {
			return
		}
		edgeSeen[key] = struct{}{}
		edges = append(edges, GraphEdge{Source: source, Target: target, Type: typ})
	}

	for _, doc := range docs {
		addNode(doc.Slug, doc.Title, "doc")
		for _, targetSlug := range extractDocLinks(doc.ContentMD, idx) {
			addNode(targetSlug, targetSlug, "doc")
			addEdge(doc.Slug, targetSlug, "link")
		}
		for _, opID := range doc.LinkedOperationIDs {
			addNode(opID, opID, "operation")
			addEdge(doc.Slug, opID, "operation")
		}
	}

	for _, m := range manual {
		addNode(m.DocSlug, m.DocSlug, "doc")
		addNode(m.RequestID, m.RequestName, "request")
		addEdge(m.DocSlug, m.RequestID, "manual")
	}

	if nodes == nil {
		nodes = []GraphNode{}
	}
	if edges == nil {
		edges = []GraphEdge{}
	}
	return DocGraph{Nodes: nodes, Edges: edges}
}

func (h *Handler) GetDocGraph(w http.ResponseWriter, r *http.Request) {
	wsID, _ := uuid.Parse(chi.URLParam(r, "id"))
	ctx := r.Context()
	rows, err := h.store.ListWorkspaceDocs(ctx, db.PGUUID(wsID))
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "query failed")
		return
	}
	docs := make([]WorkspaceDoc, 0, len(rows))
	for _, row := range rows {
		docs = append(docs, mapWorkspaceDoc(row))
	}
	manualRows, err := h.store.ListManualDocLinksByWorkspace(ctx, db.PGUUID(wsID))
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "query failed")
		return
	}
	manual := make([]ManualGraphLink, 0, len(manualRows))
	for _, row := range manualRows {
		manual = append(manual, ManualGraphLink{
			DocSlug:     row.DocSlug,
			RequestID:   db.FromPGUUID(row.RequestID).String(),
			RequestName: row.RequestName,
		})
	}
	respond.JSON(w, http.StatusOK, buildDocGraph(docs, manual))
}
