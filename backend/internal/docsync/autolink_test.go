package docsync

import (
	"testing"

	"github.com/google/uuid"
	"github.com/imaanmzr/postchi/backend/internal/docsync/linkmatcher"
)

func TestMatchFrontmatterRequests(t *testing.T) {
	docs := []linkmatcher.Doc{{
		ID: "d1", LinkedRequestNames: []string{"get-user"},
	}}
	requests := []linkmatcher.Request{
		{ID: "r1", Name: "get-user"},
		{ID: "r2", Name: "other"},
	}
	candidates := matchFrontmatterRequests(docs, requests)
	if len(candidates) != 1 || candidates[0].RequestID != "r1" {
		t.Fatalf("candidates = %+v", candidates)
	}
}

func TestFilterRequestsByCollection(t *testing.T) {
	cid := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	requests := []linkmatcher.Request{
		{ID: "r1", CollectionID: cid.String()},
		{ID: "r2", CollectionID: "c2"},
	}
	filtered := filterRequestsByCollection(requests, &cid)
	if len(filtered) != 1 || filtered[0].ID != "r1" {
		t.Fatalf("filtered = %+v", filtered)
	}
}
