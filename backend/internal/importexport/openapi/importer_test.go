package openapi

import (
	"encoding/json"
	"testing"
)

func TestParseWithHashStable(t *testing.T) {
	spec := []byte(`{
		"openapi": "3.0.0",
		"info": {"title": "Test", "version": "1.0"},
		"paths": {
			"/users": {
				"get": {"operationId": "listUsers", "responses": {"200": {"description": "ok"}}}
			}
		}
	}`)
	res, err := ParseWithHash(spec, "Test")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Operations) != 1 {
		t.Fatalf("expected 1 op, got %d", len(res.Operations))
	}
	if res.Operations[0].OperationID != "listUsers" {
		t.Fatalf("unexpected op id %s", res.Operations[0].OperationID)
	}
	if res.Operations[0].OpHash == "" {
		t.Fatal("expected hash")
	}
	if len(res.Operations[0].ApiDoc) == 0 {
		t.Fatal("expected api_doc")
	}
	res2, _ := ParseWithHash(spec, "Test")
	if res2.Operations[0].OpHash != res.Operations[0].OpHash {
		t.Fatal("hash should be stable")
	}
}

func TestParseWithHashResponseChangeAffectsHash(t *testing.T) {
	spec1 := []byte(`{
		"openapi": "3.0.0",
		"info": {"title": "Test", "version": "1.0"},
		"paths": {
			"/users": {
				"get": {"operationId": "listUsers", "responses": {"200": {"description": "ok", "content": {"application/json": {"schema": {"type": "object"}}}}}}
			}
		}
	}`)
	spec2 := []byte(`{
		"openapi": "3.0.0",
		"info": {"title": "Test", "version": "1.0"},
		"paths": {
			"/users": {
				"get": {"operationId": "listUsers", "responses": {"200": {"description": "ok", "content": {"application/json": {"schema": {"type": "array"}}}}}}
			}
		}
	}`)
	r1, _ := ParseWithHash(spec1, "Test")
	r2, _ := ParseWithHash(spec2, "Test")
	if r1.Operations[0].OpHash == r2.Operations[0].OpHash {
		t.Fatal("response schema change should affect op hash")
	}
}

func TestRequestBodyFromSchema(t *testing.T) {
	spec := []byte(`{
		"openapi": "3.0.0",
		"info": {"title": "Test", "version": "1.0"},
		"paths": {
			"/login": {
				"post": {
					"operationId": "login",
					"requestBody": {
						"required": true,
						"content": {
							"application/json": {
								"schema": {
									"type": "object",
									"required": ["username", "password"],
									"properties": {
										"username": {"type": "string", "example": "admin"},
										"password": {"type": "string", "format": "password"},
										"rememberMe": {"type": "boolean", "default": false}
									}
								}
							}
						}
					},
					"responses": {"200": {"description": "ok"}}
				}
			}
		}
	}`)
	res, err := ParseWithHash(spec, "Test")
	if err != nil {
		t.Fatal(err)
	}
	body := res.Operations[0].Request.Body
	if body.Mode != "raw" || body.RawLang != "json" {
		t.Fatalf("unexpected body mode: %+v", body)
	}
	var parsed map[string]any
	if err := json.Unmarshal([]byte(body.Raw), &parsed); err != nil {
		t.Fatalf("body not valid json: %s", body.Raw)
	}
	if parsed["username"] != "admin" {
		t.Fatalf("expected username example, got %v", parsed["username"])
	}
	if parsed["password"] != "password" {
		t.Fatalf("expected password placeholder, got %v", parsed["password"])
	}
	if parsed["rememberMe"] != false {
		t.Fatalf("expected rememberMe default false, got %v", parsed["rememberMe"])
	}
}

func TestExtractApiDocResponses(t *testing.T) {
	spec := []byte(`{
		"openapi": "3.0.0",
		"info": {"title": "Test", "version": "1.0"},
		"paths": {
			"/users/{id}": {
				"get": {
					"operationId": "getUser",
					"summary": "Get user",
					"tags": ["Users"],
					"parameters": [{"name": "id", "in": "path", "required": true, "schema": {"type": "string", "format": "uuid"}}],
					"responses": {
						"200": {"description": "OK", "content": {"application/json": {"schema": {"type": "object"}}}},
						"404": {"description": "Not found"}
					}
				}
			}
		}
	}`)
	res, err := ParseWithHash(spec, "Test")
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(res.Operations[0].ApiDoc, &doc); err != nil {
		t.Fatal(err)
	}
	if doc["summary"] != "Get user" {
		t.Fatalf("summary: %v", doc["summary"])
	}
	tags, ok := doc["tags"].([]any)
	if !ok || len(tags) != 1 {
		t.Fatalf("tags: %v", doc["tags"])
	}
	responses, ok := doc["responses"].(map[string]any)
	if !ok {
		t.Fatal("expected responses")
	}
	if _, ok := responses["200"]; !ok {
		t.Fatal("expected 200 response")
	}
	if _, ok := responses["404"]; !ok {
		t.Fatal("expected 404 response")
	}
}
