package invite

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/imaanmzr/postchi/backend/internal/auth"
	appdb "github.com/imaanmzr/postchi/backend/internal/db"
	"github.com/imaanmzr/postchi/backend/internal/db/sqlc"
	"github.com/imaanmzr/postchi/backend/internal/shared/config"
	"github.com/imaanmzr/postchi/backend/internal/shared/db"
	"github.com/imaanmzr/postchi/backend/internal/testutil"
)

func TestInviteHandlerIntegration(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL not set")
	}

	ctx := context.Background()
	root := filepath.Join("..", "..", "..", "migrations")
	if err := db.RunMigrations(databaseURL, "file://"+root); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	defer pool.Close()

	ownerID, wsID := testutil.SeedWorkspace(t, ctx, pool)
	store := appdb.NewStore(pool)
	tokens := auth.NewService("test-secret", "postchi", 15*time.Minute, 7*24*time.Hour)
	cfg := &config.Config{
		AppPublicURL: "http://localhost:3000",
		InviteTTL:    24 * time.Hour,
	}
	h := NewHandler(store, cfg, tokens)

	registeredEmail := "member-" + uuid.New().String()[:8] + "@example.com"
	_, err = store.CreateUser(ctx, sqlc.CreateUserParams{
		Email:        registeredEmail,
		PasswordHash: "hash",
		DisplayName:  "Member",
	})
	if err != nil {
		t.Fatalf("create registered user: %v", err)
	}

	t.Run("create invite for registered user adds directly", func(t *testing.T) {
		rr := httptest.NewRecorder()
		body := `{"email":"` + registeredEmail + `","role":"editor"}`
		r := inviteRequest(http.MethodPost, wsID, ownerID, body)
		h.Create(rr, r)
		if rr.Code != http.StatusCreated {
			t.Fatalf("status %d: %s", rr.Code, rr.Body.String())
		}
		var resp map[string]any
		if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if resp["outcome"] != "added" {
			t.Fatalf("expected outcome added, got %#v", resp["outcome"])
		}
		role, err := store.GetWorkspaceMemberRole(ctx, sqlc.GetWorkspaceMemberRoleParams{
			WorkspaceID: appdb.PGUUID(wsID),
			UserID:      appdb.PGUUID(mustUserIDByEmail(ctx, store, registeredEmail)),
		})
		if err != nil || role != "editor" {
			t.Fatalf("member role: %v %s", err, role)
		}
	})

	t.Run("create invite for unknown user without SMTP", func(t *testing.T) {
		unknownEmail := "unknown-" + uuid.New().String()[:8] + "@example.com"
		rr := httptest.NewRecorder()
		body := `{"email":"` + unknownEmail + `","role":"viewer"}`
		r := inviteRequest(http.MethodPost, wsID, ownerID, body)
		h.Create(rr, r)
		if rr.Code != http.StatusCreated {
			t.Fatalf("status %d: %s", rr.Code, rr.Body.String())
		}
		var resp map[string]any
		if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if resp["outcome"] != "invited" {
			t.Fatalf("expected outcome invited, got %#v", resp["outcome"])
		}
		if resp["email_sent"] != false {
			t.Fatalf("expected email_sent false, got %#v", resp["email_sent"])
		}
		url, ok := resp["invite_url"].(string)
		if !ok || !strings.HasPrefix(url, "http://localhost:3000/invite/") {
			t.Fatalf("unexpected invite_url: %#v", resp["invite_url"])
		}
	})

	t.Run("list pending invites includes invite_url", func(t *testing.T) {
		rr := httptest.NewRecorder()
		r := inviteRequest(http.MethodGet, wsID, ownerID, "")
		h.List(rr, r)
		if rr.Code != http.StatusOK {
			t.Fatalf("status %d: %s", rr.Code, rr.Body.String())
		}
		var list []Invite
		if err := json.Unmarshal(rr.Body.Bytes(), &list); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(list) == 0 {
			t.Fatal("expected pending invites")
		}
		for _, inv := range list {
			if inv.InviteURL == "" {
				t.Fatalf("missing invite_url on %s", inv.Email)
			}
		}
	})
}

func inviteRequest(method string, wsID, ownerID uuid.UUID, body string) *http.Request {
	var r *http.Request
	if body != "" {
		r = httptest.NewRequest(method, "/api/workspaces/"+wsID.String()+"/invites", strings.NewReader(body))
	} else {
		r = httptest.NewRequest(method, "/api/workspaces/"+wsID.String()+"/invites", nil)
	}
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", wsID.String())
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	r = r.WithContext(context.WithValue(r.Context(), auth.UserIDKey, ownerID.String()))
	return r
}

func mustUserIDByEmail(ctx context.Context, store *appdb.Store, email string) uuid.UUID {
	id, err := store.GetUserIDByEmail(ctx, email)
	if err != nil {
		panic(err)
	}
	return appdb.FromPGUUID(id)
}
