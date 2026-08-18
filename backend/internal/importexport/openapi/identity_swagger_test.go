package openapi

import (
	"os"
	"testing"
)

func TestParseIdentityUATSwagger(t *testing.T) {
	data, err := os.ReadFile("/tmp/swagger.json")
	if err != nil {
		t.Skip("download swagger to /tmp/swagger.json first")
	}
	res, err := ParseWithHash(data, "Identity")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(res.Operations) == 0 {
		t.Fatal("expected operations from identity swagger")
	}
	t.Logf("parsed %d operations", len(res.Operations))
}
