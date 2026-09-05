package gitrepo

import "testing"

func TestParseGitLabTreeRefSlashBranch(t *testing.T) {
	branch, path := ParseGitLabTreeRef("fix/BO-1287-remove-merchant-domain-check/bruno-collection")
	if branch != "fix/BO-1287-remove-merchant-domain-check" || path != "bruno-collection" {
		t.Fatalf("got branch=%q path=%q", branch, path)
	}
}

func TestParseGitLabTreeRefEncodedBranch(t *testing.T) {
	branch, path := ParseGitLabTreeRef("fix%2FBO-1287-remove-merchant-domain-check/bruno-collection")
	if branch != "fix/BO-1287-remove-merchant-domain-check" || path != "bruno-collection" {
		t.Fatalf("got branch=%q path=%q", branch, path)
	}
}

func TestParseGitLabTreeRefNestedPath(t *testing.T) {
	branch, path := ParseGitLabTreeRef("develop/collections/api")
	if branch != "develop" || path != "collections/api" {
		t.Fatalf("got branch=%q path=%q", branch, path)
	}
}

func TestNormalizePathPrefixStripsDuplicateTicketFolder(t *testing.T) {
	got := NormalizePathPrefix(
		"fix/BO-1287-remove-merchant-domain-check",
		"BO-1287-remove-merchant-domain-check/bruno-collection",
	)
	if got != "bruno-collection" {
		t.Fatalf("got %q", got)
	}
}
