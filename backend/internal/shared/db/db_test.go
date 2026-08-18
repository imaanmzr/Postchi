package db

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveMigrationsPath_keepsAbsoluteFileURL(t *testing.T) {
	got := ResolveMigrationsPath("file:///migrations")
	if got != "file:///migrations" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveMigrationsPath_prefixesBareAbsolute(t *testing.T) {
	got := ResolveMigrationsPath("/migrations")
	if got != "file:///migrations" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveMigrationsPath_fallsBackWhenMissing(t *testing.T) {
	got := ResolveMigrationsPath("file://migrations-does-not-exist-xyz")
	if !strings.HasPrefix(got, "file://") {
		t.Fatalf("expected file URL, got %q", got)
	}
	if !strings.HasSuffix(filepath.ToSlash(got), "/migrations") && !strings.Contains(got, "../migrations") {
		t.Fatalf("expected fallback to ../migrations, got %q", got)
	}
}
