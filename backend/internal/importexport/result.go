package importexport

import "github.com/imaanmzr/postchi/backend/internal/importexport/model"

type ImportResult struct {
	CollectionID  string `json:"collection_id,omitempty"`
	RequestID     string `json:"request_id,omitempty"`
	Collections   int    `json:"collections"`
	Requests      int    `json:"requests"`
	Environments  int    `json:"environments"`
}

func (r ImportResult) Total() int {
	return r.Collections + r.Requests + r.Environments
}

func countCollectionTree(col model.Collection) (collections, requests int) {
	collections = 1
	requests = len(col.Requests)
	for _, child := range col.Children {
		c, req := countCollectionTree(child)
		collections += c
		requests += req
	}
	return collections, requests
}
