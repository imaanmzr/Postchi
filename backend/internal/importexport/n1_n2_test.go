package importexport

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/imaanmzr/postchi/backend/internal/importexport/model"
	ocimport "github.com/imaanmzr/postchi/backend/internal/importexport/opencollection"
)

func countTree(col model.Collection) (folders, reqs int) {
	folders = len(col.Children)
	for _, c := range col.Children {
		f, r := countTree(c)
		folders += f
		reqs += r
	}
	reqs += len(col.Requests)
	return
}

func TestN1PostmanBrunoExport(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "n1.json"))
	if err != nil {
		t.Fatal(err)
	}
	col, err := ParsePostman(data)
	if err != nil {
		t.Fatal(err)
	}
	f, r := countTree(col)
	t.Logf("parsed: folders=%d requests=%d name=%q", f, r, col.Name)
	if r == 0 {
		t.Fatal("expected requests")
	}
	if col.Name != "Acme API - Staging" {
		t.Fatalf("name=%q", col.Name)
	}
}

func TestN2OpenCollection(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "n2.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if !ocimport.IsOpenCollection(data) {
		t.Fatal("expected opencollection marker")
	}
	col, err := ocimport.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	f, r := countTree(col)
	t.Logf("parsed: folders=%d requests=%d name=%q vars=%d", f, r, col.Name, len(col.Variables.PreRequest))
	if r == 0 {
		t.Fatal("expected requests")
	}
	if col.Name != "Acme API - Staging" {
		t.Fatalf("name=%q", col.Name)
	}
	if len(col.Variables.PreRequest) < 5 {
		t.Fatalf("expected collection variables, got %+v", col.Variables.PreRequest)
	}
}
