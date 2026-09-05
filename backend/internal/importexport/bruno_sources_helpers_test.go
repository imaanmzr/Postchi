package importexport

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/imaanmzr/postchi/backend/internal/db"
	"github.com/imaanmzr/postchi/backend/internal/db/sqlc"
	"github.com/imaanmzr/postchi/backend/internal/shared/crypto"
)

func TestNormalizeBrunoRepoConfig(t *testing.T) {
	cfg, err := normalizeBrunoRepoConfig(map[string]any{
		"repo_url": "https://github.com/acme/repo",
		"access_token": "secret",
	})
	if err != nil {
		t.Fatal(err)
	}
	if cfg["provider"] != "github" || cfg["branch"] != "main" {
		t.Fatalf("cfg = %+v", cfg)
	}
	if _, ok := cfg["access_token"]; ok {
		t.Fatal("token should be stripped")
	}
}

func TestNormalizeBrunoRepoConfigMissingURL(t *testing.T) {
	_, err := normalizeBrunoRepoConfig(map[string]any{"branch": "main"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestNormalizeBrunoRepoConfigInvalidBranch(t *testing.T) {
	_, err := normalizeBrunoRepoConfig(map[string]any{
		"repo_url": "https://github.com/acme/repo",
		"branch":   "bad branch",
	})
	if err == nil {
		t.Fatal("expected invalid branch error")
	}
}

func TestStringFromConfig(t *testing.T) {
	if got := stringFromConfig(map[string]any{"branch": " develop "}, "branch"); got != "develop" {
		t.Fatalf("got %q", got)
	}
}

func TestMapBrunoSourceRows(t *testing.T) {
	now := pgtype.Timestamptz{Valid: true, Time: time.Now().UTC()}
	row := mapBrunoSourceListRow(sqlc.ListBrunoSourcesRow{
		ID:           db.PGUUID(uuid.New()),
		WorkspaceID:  db.PGUUID(uuid.New()),
		CollectionID: db.PGUUID(uuid.New()),
		Name:         "API",
		Config:       []byte(`{"repo_url":"https://github.com/acme/repo","branch":"main"}`),
		LastSyncedAt: now,
		CreatedAt:    now,
	})
	if row.Name != "API" || row.CollectionID == nil {
		t.Fatalf("row = %+v", row)
	}
	got := mapBrunoSourceGetRow(sqlc.GetBrunoSourceRow{
		ID:          db.PGUUID(uuid.New()),
		WorkspaceID: db.PGUUID(uuid.New()),
		Name:        "API",
		Config:      []byte(`{"repo_url":"https://github.com/acme/repo"}`),
		CreatedAt:   now,
	})
	if got.Name != "API" {
		t.Fatalf("get row = %+v", got)
	}
}

func TestEncryptTokenWithoutCrypto(t *testing.T) {
	h := NewHandler(nil, nil)
	if _, err := h.encryptToken("secret"); err == nil {
		t.Fatal("expected error without crypto")
	}
}

func TestEncryptTokenWithCrypto(t *testing.T) {
	svc, err := crypto.NewService("0123456789abcdef0123456789abcdef")
	if err != nil {
		t.Fatal(err)
	}
	h := NewHandler(nil, svc)
	enc, err := h.encryptToken("secret")
	if err != nil || enc == nil {
		t.Fatalf("encrypt: %v", err)
	}
}

func TestDecodeBrunoConfig(t *testing.T) {
	cfg := decodeBrunoConfig([]byte(`{"repo_url":"https://github.com/acme/repo"}`))
	if cfg["repo_url"] == nil {
		t.Fatal("expected config")
	}
}

func TestPgHelpers(t *testing.T) {
	id := uuid.New()
	if s := pgUUIDToStringPtr(db.PGUUID(id)); s == nil || *s != id.String() {
		t.Fatal("uuid ptr")
	}
	now := pgtype.Timestamptz{Valid: true, Time: time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)}
	if s := pgTimeToStringPtr(now); s == nil || *s == "" {
		t.Fatal("time ptr")
	}
	if formatBrunoTime(pgtype.Timestamptz{}) != "" {
		t.Fatal("invalid time should be empty")
	}
}
