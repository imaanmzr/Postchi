package importexport

import (
	"archive/zip"
	"bytes"
	"testing"

	"github.com/imaanmzr/postchi/backend/internal/importexport/bruno"
)

func TestValidateBrunoMetadataEdgeCases(t *testing.T) {
	parsed := bruno.Parse("meta {\n  name: Folder\n}\n")
	if err := validateBrunoMetadata("orders/folder.bru", parsed, true); err != nil {
		t.Fatal(err)
	}
	empty := bruno.Parse("")
	if err := validateBrunoMetadata("broken.bru", empty, false); err == nil {
		t.Fatal("expected metadata error")
	}
}

func TestParseBrunoZipInvalidArchive(t *testing.T) {
	_, err := parseBrunoZip([]byte("not a zip"))
	if err == nil {
		t.Fatal("expected zip error")
	}
}

func TestParseBrunoZipValidArchive(t *testing.T) {
	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)
	f, _ := zw.Create("collection.bru")
	_, _ = f.Write([]byte("meta {\n  name: Zip\n  type: collection\n}\n"))
	f2, _ := zw.Create("ping.bru")
	_, _ = f2.Write([]byte("meta {\n  name: Ping\n  type: http\n}\nget {\n  url: https://example.com\n}\n"))
	_ = zw.Close()
	col, err := parseBrunoZip(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if len(col.Requests) != 1 {
		t.Fatalf("requests=%d", len(col.Requests))
	}
}

func TestBruMetaSeqMissingOrInvalid(t *testing.T) {
	parsed := bruno.Parse("meta {\n  name: Ping\n  seq: not-a-number\n}\n")
	if seq := bruMetaSeq(parsed); seq != -1 {
		t.Fatalf("seq=%d", seq)
	}
}

func TestParseBrunoSingleFileCollectionMetaOnly(t *testing.T) {
	col, err := parseBrunoSingleFile([]byte("meta {\n  name: Only Meta\n  type: collection\n}\n"), "collection.bru")
	if err != nil {
		t.Fatal(err)
	}
	if col.Name != "Only Meta" || len(col.Requests) != 0 {
		t.Fatalf("col=%+v", col)
	}
}

func TestParseBrunoSingleFileUnrecognized(t *testing.T) {
	_, err := parseBrunoSingleFile([]byte("not bru"), "x.bru")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestValidateBrunoRequestMissingURL(t *testing.T) {
	parsed := bruno.Parse("meta {\n  name: Broken\n  type: http\n}\nget {\n}\n")
	if err := validateBrunoRequest(parsed); err == nil {
		t.Fatal("expected missing URL error")
	}
}
