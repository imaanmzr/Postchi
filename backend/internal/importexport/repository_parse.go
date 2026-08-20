package importexport

import (
	"fmt"
	"strings"

	"github.com/imaanmzr/postchi/backend/internal/importexport/detect"
	"github.com/imaanmzr/postchi/backend/internal/importexport/gitsync"
	"github.com/imaanmzr/postchi/backend/internal/importexport/model"
	openapiimport "github.com/imaanmzr/postchi/backend/internal/importexport/openapi"
	ocimport "github.com/imaanmzr/postchi/backend/internal/importexport/opencollection"
)

type ParsedRepository struct {
	Collections []model.Collection
	Errors      []string
}

func parseRepositoryRoots(roots []gitsync.DiscoveredRoot, defaultName string) ParsedRepository {
	out := ParsedRepository{}
	for _, root := range roots {
		col, err := parseDiscoveredRoot(root, defaultName)
		if err != nil {
			path := root.FilePath
			if path == "" {
				path = root.RootPath
			}
			out.Errors = append(out.Errors, fmt.Sprintf("%s: %v", path, err))
			continue
		}
		if countRequestsInTree(col) == 0 {
			path := root.FilePath
			if path == "" {
				path = root.RootPath
			}
			out.Errors = append(out.Errors, fmt.Sprintf("%s: contains no requests", path))
			continue
		}
		out.Collections = append(out.Collections, col)
	}
	return out
}

func parseDiscoveredRoot(root gitsync.DiscoveredRoot, defaultName string) (model.Collection, error) {
	switch root.Format {
	case detect.FormatBruno:
		files := make([]brunoSourceFile, 0, len(root.Files))
		for _, file := range root.Files {
			files = append(files, brunoSourceFile{Path: file.Path, Content: file.Content})
		}
		name := defaultName
		if root.RootPath != "" {
			name = filepathBaseName(root.RootPath)
		}
		return parseBrunoFiles(files, brunoParseOptions{
			RootName:         name,
			RootPathPrefix:   root.RootPath,
			RequireRootMeta:  true,
			ValidateRequests: true,
		})
	case detect.FormatPostman:
		col, err := ParsePostman(root.Content)
		if err != nil {
			return model.Collection{}, err
		}
		col.SourcePath = root.FilePath
		tagSourcePaths(&col, root.FilePath)
		return col, nil
	case detect.FormatOpenCollection:
		col, err := ocimport.Parse(root.Content)
		if err != nil {
			return model.Collection{}, err
		}
		col.SourcePath = root.FilePath
		tagSourcePaths(&col, root.FilePath)
		return col, nil
	case detect.FormatOpenAPI:
		name := strings.TrimSuffix(filepathBaseName(root.FilePath), filepathExt(root.FilePath))
		col, err := openapiimport.Parse(root.Content, name)
		if err != nil {
			return model.Collection{}, err
		}
		col.SourcePath = root.FilePath
		for i := range col.Requests {
			opID := col.Requests[i].Name
			if opID == "" {
				opID = fmt.Sprintf("%s-%d", col.Requests[i].Method, i)
			}
			col.Requests[i].SourcePath = root.FilePath + "#" + opID
		}
		return col, nil
	default:
		return model.Collection{}, fmt.Errorf("unsupported format")
	}
}

func tagSourcePaths(col *model.Collection, prefix string) {
	if col.SourcePath == "" && prefix != "" {
		col.SourcePath = prefix
	}
	for i := range col.Requests {
		if col.Requests[i].SourcePath == "" {
			col.Requests[i].SourcePath = prefix + "#" + col.Requests[i].Name
		}
	}
	for i := range col.Children {
		childPrefix := prefix
		if col.Children[i].Name != "" {
			childPrefix = prefix + "/" + col.Children[i].Name
		}
		tagSourcePaths(&col.Children[i], childPrefix)
	}
}

func countRequestsInTree(col model.Collection) int {
	_, requests := countCollectionTree(col)
	return requests
}

func filepathBaseName(path string) string {
	path = strings.Trim(path, "/")
	if path == "" {
		return ""
	}
	parts := strings.Split(path, "/")
	return parts[len(parts)-1]
}

func filepathExt(path string) string {
	if idx := strings.LastIndex(path, "."); idx >= 0 {
		return path[idx:]
	}
	return ""
}

func collectAllSourcePaths(collections []model.Collection) (collectionPaths, requestPaths []string) {
	for _, col := range collections {
		cPaths, rPaths := collectBrunoSourcePaths(col)
		collectionPaths = append(collectionPaths, cPaths...)
		requestPaths = append(requestPaths, rPaths...)
	}
	return collectionPaths, requestPaths
}
