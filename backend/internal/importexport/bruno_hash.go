package importexport

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"github.com/imaanmzr/postchi/backend/internal/importexport/model"
	"github.com/imaanmzr/postchi/backend/internal/request"
)

func brunoRequestHash(req model.Request) string {
	payload := struct {
		Name             string             `json:"name"`
		Method           string             `json:"method"`
		URL              string             `json:"url"`
		Description      string             `json:"description"`
		Headers          []request.KVPair   `json:"headers"`
		Params           []request.KVPair   `json:"params"`
		PathVars         []request.KVPair   `json:"path_vars"`
		Body             request.BodySpec   `json:"body"`
		Auth             request.AuthSpec   `json:"auth"`
		PreRequestScript string             `json:"pre_request_script"`
		TestScript       string             `json:"test_script"`
		SortOrder        int                `json:"sort_order"`
	}{
		Name:             req.Name,
		Method:           req.Method,
		URL:              req.URL,
		Description:      req.Description,
		Headers:          req.Headers,
		Params:           req.Params,
		PathVars:         req.PathVars,
		Body:             req.Body,
		Auth:             req.Auth,
		PreRequestScript: req.PreRequestScript,
		TestScript:       req.TestScript,
		SortOrder:        req.SortOrder,
	}
	b, _ := json.Marshal(payload)
	sum := sha256.Sum256(b)
	return fmt.Sprintf("%x", sum)
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
