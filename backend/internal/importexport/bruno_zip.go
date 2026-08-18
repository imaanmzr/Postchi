package importexport

import (
	"archive/zip"
	"bytes"
	"io"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/imaanmzr/postchi/backend/internal/importexport/bruno"
	"github.com/imaanmzr/postchi/backend/internal/importexport/model"
	"github.com/imaanmzr/postchi/backend/internal/shared/domain"
)

type bruDirNode struct {
	name      string
	sortOrder int
	variables domain.VariablesSpec
	children  map[string]*bruDirNode
	requests  []model.Request
}

func parseBrunoZip(data []byte) (model.Collection, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return model.Collection{}, err
	}
	root := &bruDirNode{name: "Imported Bruno", children: map[string]*bruDirNode{}}

	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		name := filepath.ToSlash(f.Name)
		if !strings.HasSuffix(strings.ToLower(name), ".bru") {
			continue
		}
		parts := strings.Split(name, "/")
		base := parts[len(parts)-1]
		dirParts := parts[:len(parts)-1]
		if len(dirParts) > 0 && strings.EqualFold(dirParts[0], "environments") {
			continue
		}

		rc, err := f.Open()
		if err != nil {
			continue
		}
		b, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			continue
		}

		parsed := bruno.Parse(string(b))
		node := bruGetOrCreatePath(root, dirParts)

		if strings.EqualFold(base, "collection.bru") || strings.EqualFold(base, "folder.bru") {
			if parsed.Name != "" {
				node.name = parsed.Name
			}
			node.variables = bruVarsToSpec(bruno.ToVars(parsed.Sections["vars:pre-request"], parsed.Sections["vars:post-response"]))
			if seq := bruMetaSeq(parsed); seq >= 0 {
				node.sortOrder = seq
			}
			continue
		}

		req := bruToNorm(bruno.ToRequest(parsed))
		if req.Name == "" {
			req.Name = strings.TrimSuffix(base, ".bru")
		}
		if seq := bruMetaSeq(parsed); seq >= 0 {
			req.SortOrder = seq
		} else {
			req.SortOrder = len(node.requests)
		}
		node.requests = append(node.requests, req)
	}

	return bruDirToModel(root), nil
}

func bruGetOrCreatePath(root *bruDirNode, dirParts []string) *bruDirNode {
	cur := root
	for _, part := range dirParts {
		if part == "" {
			continue
		}
		if cur.children[part] == nil {
			cur.children[part] = &bruDirNode{
				name:      part,
				children:  map[string]*bruDirNode{},
			}
		}
		cur = cur.children[part]
	}
	return cur
}

func bruMetaSeq(parsed bruno.ParsedBru) int {
	block, ok := parsed.Sections["meta"]
	if !ok {
		return -1
	}
	for _, line := range strings.Split(block, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "seq:") {
			val := strings.TrimSpace(strings.TrimPrefix(line, "seq:"))
			if n, err := strconv.Atoi(val); err == nil {
				return n
			}
		}
	}
	return -1
}

func bruDirToModel(node *bruDirNode) model.Collection {
	col := model.Collection{
		Name:      node.name,
		SortOrder: node.sortOrder,
		Variables: node.variables,
		Requests:  append([]model.Request{}, node.requests...),
	}
	type childEntry struct {
		key  string
		node *bruDirNode
	}
	entries := make([]childEntry, 0, len(node.children))
	for key, child := range node.children {
		entries = append(entries, childEntry{key: key, node: child})
	}
	sort.Slice(entries, func(i, j int) bool {
		a, b := entries[i], entries[j]
		if a.node.sortOrder != b.node.sortOrder {
			return a.node.sortOrder < b.node.sortOrder
		}
		return a.node.name < b.node.name
	})
	for i, entry := range entries {
		child := bruDirToModel(entry.node)
		if child.SortOrder == 0 && entry.node.sortOrder == 0 {
			child.SortOrder = i
		}
		col.Children = append(col.Children, child)
	}
	return col
}
