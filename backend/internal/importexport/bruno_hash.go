package importexport

import (
	"github.com/imaanmzr/postchi/backend/internal/importexport/hash"
	"github.com/imaanmzr/postchi/backend/internal/importexport/model"
)

func brunoRequestHash(req model.Request) string {
	return hash.Request(req)
}

func collectBrunoSourcePaths(col model.Collection) (collectionPaths, requestPaths []string) {
	var walk func(model.Collection)
	walk = func(node model.Collection) {
		if node.SourcePath != "" {
			collectionPaths = append(collectionPaths, node.SourcePath)
		}
		for _, req := range node.Requests {
			if req.SourcePath != "" {
				requestPaths = append(requestPaths, req.SourcePath)
			}
		}
		for _, child := range node.Children {
			walk(child)
		}
	}
	walk(col)
	return collectionPaths, requestPaths
}
