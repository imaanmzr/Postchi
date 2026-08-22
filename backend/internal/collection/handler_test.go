package collection

import (
	"strings"
	"testing"

	"github.com/imaanmzr/postchi/backend/internal/request"
	"github.com/imaanmzr/postchi/backend/internal/shared/domain"
)

func sampleCollection() Collection {
	parent := "parent-id"
	return Collection{
		ID:               "col-id",
		WorkspaceID:      "ws-id",
		ParentID:         &parent,
		Name:             "AldyPay API Collection",
		Description:      "Gateway APIs",
		SortOrder:        3,
		Variables:        domain.EmptyVariablesSpec(),
		Headers:          []request.KVPair{{Key: "X-App", Value: "postchi", Enabled: true}},
		Auth:             request.AuthSpec{Type: "inherit"},
		PreRequestScript: "console.log('pre')",
		TestScript:       "console.log('test')",
	}
}

func TestApplyCollectionPatchKeepsNameWhenSavingVars(t *testing.T) {
	existing := sampleCollection()
	body := strings.NewReader(`{"variables":{"pre_request":[{"enabled":true,"name":"uatBaseUrl","value":"https://gateway.uat.example.com","type":"string"}],"post_response":[]}}`)

	got, err := applyCollectionPatch(existing, body)
	if err != nil {
		t.Fatalf("patch: %v", err)
	}
	if got.Name != existing.Name {
		t.Fatalf("name = %q, want %q", got.Name, existing.Name)
	}
	if got.Description != existing.Description {
		t.Fatalf("description = %q, want %q", got.Description, existing.Description)
	}
	if got.ParentID == nil || *got.ParentID != *existing.ParentID {
		t.Fatalf("parent_id = %v, want %v", got.ParentID, existing.ParentID)
	}
	if got.SortOrder != existing.SortOrder {
		t.Fatalf("sort_order = %d, want %d", got.SortOrder, existing.SortOrder)
	}
	if len(got.Headers) != 1 || got.Headers[0].Key != "X-App" {
		t.Fatalf("headers = %+v, want existing headers", got.Headers)
	}
	if got.Auth.Type != "inherit" {
		t.Fatalf("auth type = %q, want inherit", got.Auth.Type)
	}
	if len(got.Variables.PreRequest) != 1 || got.Variables.PreRequest[0].Name != "uatBaseUrl" {
		t.Fatalf("variables = %+v", got.Variables)
	}
	if got.ID != existing.ID || got.WorkspaceID != existing.WorkspaceID {
		t.Fatalf("id/workspace changed: %+v", got)
	}
}

func TestApplyCollectionPatchUpdatesName(t *testing.T) {
	existing := sampleCollection()
	got, err := applyCollectionPatch(existing, strings.NewReader(`{"name":"Renamed","description":"Updated docs"}`))
	if err != nil {
		t.Fatalf("patch: %v", err)
	}
	if got.Name != "Renamed" {
		t.Fatalf("name = %q, want Renamed", got.Name)
	}
	if got.Description != "Updated docs" {
		t.Fatalf("description = %q", got.Description)
	}
	if len(got.Headers) != 1 {
		t.Fatalf("headers were wiped: %+v", got.Headers)
	}
}

func TestApplyCollectionPatchIgnoresBlankName(t *testing.T) {
	existing := sampleCollection()
	got, err := applyCollectionPatch(existing, strings.NewReader(`{"name":""}`))
	if err != nil {
		t.Fatalf("patch: %v", err)
	}
	if got.Name != existing.Name {
		t.Fatalf("blank name overwrote existing: %q", got.Name)
	}
}

func TestParentIDsEqual(t *testing.T) {
	a := "a"
	b := "b"
	if !parentIDsEqual(nil, nil) {
		t.Fatal("nil parents should be equal")
	}
	if parentIDsEqual(&a, nil) || parentIDsEqual(nil, &a) {
		t.Fatal("nil vs value should differ")
	}
	if !parentIDsEqual(&a, &a) {
		t.Fatal("same parent should be equal")
	}
	if parentIDsEqual(&a, &b) {
		t.Fatal("different parents should differ")
	}
}
