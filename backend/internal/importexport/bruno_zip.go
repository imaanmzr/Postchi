package importexport

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/imaanmzr/postchi/backend/internal/importexport/bruno"
	"github.com/imaanmzr/postchi/backend/internal/importexport/model"
	"github.com/imaanmzr/postchi/backend/internal/shared/domain"
)

type brunoSourceFile struct {
	Path    string
	Content []byte
}

type brunoParseOptions struct {
	RootName         string
	RequireRootMeta  bool
	ValidateRequests bool
}

type bruDirNode struct {
	name       string
	sortOrder  int
	sourcePath string
	variables  domain.VariablesSpec
	children   map[string]*bruDirNode
	requests   []model.Request
}

func parseBrunoZip(data []byte) (model.Collection, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return model.Collection{}, err
	}
	files := make([]brunoSourceFile, 0, len(zr.File))
	for _, file := range zr.File {
		if file.FileInfo().IsDir() {
			continue
		}
		name := filepath.ToSlash(file.Name)
		if !strings.HasSuffix(strings.ToLower(name), ".bru") {
			continue
		}
		reader, err := file.Open()
		if err != nil {
			return model.Collection{}, fmt.Errorf("open %q: %w", name, err)
		}
		content, readErr := io.ReadAll(reader)
		closeErr := reader.Close()
		if readErr != nil {
			return model.Collection{}, fmt.Errorf("read %q: %w", name, readErr)
		}
		if closeErr != nil {
			return model.Collection{}, fmt.Errorf("close %q: %w", name, closeErr)
		}
		files = append(files, brunoSourceFile{Path: name, Content: content})
	}
	return parseBrunoFiles(files, brunoParseOptions{})
}

func parseBrunoFiles(files []brunoSourceFile, options brunoParseOptions) (model.Collection, error) {
	root := &bruDirNode{name: "Imported Bruno", children: map[string]*bruDirNode{}}
	foundRootMeta := false

	for _, file := range files {
		name := strings.Trim(filepath.ToSlash(file.Path), "/")
		if !strings.HasSuffix(strings.ToLower(name), ".bru") {
			continue
		}
		parts := strings.Split(name, "/")
		if containsPathSegment(parts[:len(parts)-1], "environments") {
			continue
		}
		base := parts[len(parts)-1]
		dirParts := parts[:len(parts)-1]
		isCollection := strings.EqualFold(base, "collection.bru")
		isFolder := strings.EqualFold(base, "folder.bru")

		if len(bytes.TrimSpace(file.Content)) == 0 {
			return model.Collection{}, fmt.Errorf("%s is empty", name)
		}
		parsed := bruno.Parse(string(file.Content))
		node := bruGetOrCreatePath(root, dirParts)

		if isCollection || isFolder {
			if isCollection && len(dirParts) == 0 {
				foundRootMeta = true
			}
			if err := validateBrunoMetadata(name, parsed, isFolder); err != nil {
				return model.Collection{}, fmt.Errorf("%s: %w", name, err)
			}
			if parsed.Name != "" {
				node.name = parsed.Name
			}
			node.sourcePath = name
			node.variables = bruVarsToSpec(bruno.ToVars(parsed.Sections["vars:pre-request"], parsed.Sections["vars:post-response"]))
			if seq := bruMetaSeq(parsed); seq >= 0 {
				node.sortOrder = seq
			}
			continue
		}

		if options.ValidateRequests {
			if err := validateBrunoRequest(parsed); err != nil {
				return model.Collection{}, fmt.Errorf("%s: %w", name, err)
			}
		}
		req := bruToNorm(bruno.ToRequest(parsed))
		req.SourcePath = name
		if req.Name == "" {
			req.Name = strings.TrimSuffix(base, filepath.Ext(base))
		}
		if seq := bruMetaSeq(parsed); seq >= 0 {
			req.SortOrder = seq
		} else {
			req.SortOrder = len(node.requests)
		}
		node.requests = append(node.requests, req)
	}

	if options.RequireRootMeta && !foundRootMeta {
		return model.Collection{}, fmt.Errorf("collection.bru not found at repository import root")
	}
	if strings.TrimSpace(options.RootName) != "" {
		root.name = strings.TrimSpace(options.RootName)
	}
	return bruDirToModel(root), nil
}

func validateBrunoMetadata(filePath string, parsed bruno.ParsedBru, isFolder bool) error {
	if _, ok := parsed.Sections["meta"]; ok {
		return nil
	}
	if isFolder {
		return nil
	}
	if strings.EqualFold(filepath.Base(filePath), "collection.bru") {
		return nil
	}
	if len(parsed.Sections) == 0 && parsed.Name == "" {
		return fmt.Errorf("malformed or missing meta block")
	}
	return nil
}

func validateBrunoRequest(parsed bruno.ParsedBru) error {
	if _, ok := parsed.Sections["meta"]; !ok {
		return fmt.Errorf("malformed or missing meta block")
	}
	for _, method := range []string{"get", "post", "put", "patch", "delete"} {
		if block, ok := parsed.Sections[method]; ok {
			if strings.TrimSpace(bruno.ToRequest(parsed).URL) == "" || strings.TrimSpace(block) == "" {
				return fmt.Errorf("request URL is missing")
			}
			return nil
		}
	}
	return fmt.Errorf("HTTP method block is missing")
}

func containsPathSegment(parts []string, target string) bool {
	for _, part := range parts {
		if strings.EqualFold(part, target) {
			return true
		}
	}
	return false
}

func bruGetOrCreatePath(root *bruDirNode, dirParts []string) *bruDirNode {
	current := root
	pathParts := make([]string, 0, len(dirParts))
	for _, part := range dirParts {
		if part == "" {
			continue
		}
		pathParts = append(pathParts, part)
		if current.children[part] == nil {
			current.children[part] = &bruDirNode{
				name:       part,
				sourcePath: strings.Join(pathParts, "/") + "/",
				children:   map[string]*bruDirNode{},
			}
		}
		current = current.children[part]
	}
	return current
}

func bruMetaSeq(parsed bruno.ParsedBru) int {
	block, ok := parsed.Sections["meta"]
	if !ok {
		return -1
	}
	for _, line := range strings.Split(block, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "seq:") {
			value := strings.TrimSpace(strings.TrimPrefix(line, "seq:"))
			if sequence, err := strconv.Atoi(value); err == nil {
				return sequence
			}
		}
	}
	return -1
}

func bruDirToModel(node *bruDirNode) model.Collection {
	collection := model.Collection{
		Name:             node.name,
		SortOrder:        node.sortOrder,
		SourcePath:       node.sourcePath,
		Variables:        node.variables,
		Requests:         append([]model.Request{}, node.requests...),
	}
	sort.SliceStable(collection.Requests, func(i, j int) bool {
		return collection.Requests[i].SortOrder < collection.Requests[j].SortOrder
	})
	type childEntry struct {
		key  string
		node *bruDirNode
	}
	entries := make([]childEntry, 0, len(node.children))
	for key, child := range node.children {
		entries = append(entries, childEntry{key: key, node: child})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].node.sortOrder != entries[j].node.sortOrder {
			return entries[i].node.sortOrder < entries[j].node.sortOrder
		}
		return entries[i].node.name < entries[j].node.name
	})
	for index, entry := range entries {
		child := bruDirToModel(entry.node)
		if child.SortOrder == 0 && entry.node.sortOrder == 0 {
			child.SortOrder = index
		}
		collection.Children = append(collection.Children, child)
	}
	return collection
}
