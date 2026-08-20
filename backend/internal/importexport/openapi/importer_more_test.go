package openapi

import (
	"testing"
)

func TestParseWrapper(t *testing.T) {
	spec := []byte(`{
		"openapi": "3.0.0",
		"info": {"title": "Pets", "version": "1.0"},
		"paths": {
			"/pets": {
				"get": {"operationId": "listPets", "responses": {"200": {"description": "ok"}}}
			}
		}
	}`)
	col, err := Parse(spec, "")
	if err != nil {
		t.Fatal(err)
	}
	if col.Name != "Pets" || len(col.Requests) != 1 {
		t.Fatalf("col=%+v", col)
	}
}

func TestFormURLEncodedRequestBody(t *testing.T) {
	spec := []byte(`{
		"openapi": "3.0.0",
		"info": {"title": "Auth", "version": "1.0"},
		"paths": {
			"/login": {
				"post": {
					"operationId": "login",
					"requestBody": {
						"content": {
							"application/x-www-form-urlencoded": {
								"schema": {
									"type": "object",
									"properties": {
										"username": {"type": "string", "example": "admin"},
										"password": {"type": "string", "format": "password"}
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
	res, err := ParseWithHash(spec, "Auth")
	if err != nil {
		t.Fatal(err)
	}
	body := res.Operations[0].Request.Body
	if body.Mode != "urlencoded" || body.Raw == "" {
		t.Fatalf("body=%+v", body)
	}
}

func TestMultipartAndSchemaSamples(t *testing.T) {
	spec := []byte(`{
		"openapi": "3.0.0",
		"info": {"title": "Upload", "version": "1.0"},
		"paths": {
			"/upload": {
				"post": {
					"operationId": "upload",
					"requestBody": {
						"content": {
							"multipart/form-data": {
								"schema": {
									"type": "object",
									"properties": {
										"file": {"type": "string", "format": "binary"}
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
	res, err := ParseWithHash(spec, "Upload")
	if err != nil {
		t.Fatal(err)
	}
	if res.Operations[0].Request.Body.Mode != "multipart" {
		t.Fatalf("body=%+v", res.Operations[0].Request.Body)
	}
}

func TestSampleFromSchemaVariants(t *testing.T) {
	spec := []byte(`{
		"openapi": "3.0.0",
		"info": {"title": "Rich", "version": "1.0"},
		"paths": {
			"/items": {
				"post": {
					"operationId": "createItem",
					"requestBody": {
						"content": {
							"application/json": {
								"schema": {
									"type": "object",
									"properties": {
										"email": {"type": "string", "format": "email"},
										"count": {"type": "integer"},
										"active": {"type": "boolean"},
										"tags": {"type": "array", "items": {"type": "string"}},
										"kind": {"type": "string", "enum": ["a","b"]}
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
	res, err := ParseWithHash(spec, "Rich")
	if err != nil {
		t.Fatal(err)
	}
	if res.Operations[0].Request.Body.Raw == "" {
		t.Fatal("expected generated sample body")
	}
}
