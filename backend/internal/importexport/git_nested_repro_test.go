package importexport

import (
	"context"
	"testing"

	"github.com/imaanmzr/postchi/backend/internal/importexport/gitsync"
	"github.com/imaanmzr/postchi/backend/internal/shared/gitrepo"
)

func TestGitNestedHubWallet(t *testing.T) {
	gitServer := newGitLabFileServer(t, map[string]string{
		"collection.bru": `meta {
  name: Hub Wallet API
  type: collection
}
`,
		"Hub-Wallet/folder.bru": `meta {
  name: Hub-Wallet
  seq: 1
}
`,
		"Hub-Wallet/login.bru": `meta {
  name: login
  type: http
}

post {
  url: {{paymentBaseUrl}}/api/v1/login
  body: json
}

body:json {
  {
    "username": "test",
    "password": "123456"
  }
}
`,
		"Hub-Wallet/create-transaction.bru": `meta {
  name: create-transaction
  type: http
}

post {
  url: {{paymentBaseUrl}}/api/v1/tx
}
`,
		"Services.Identity.Api/folder.bru": `meta {
  name: Services.Identity.Api
  seq: 2
}
`,
		"Services.Identity.Api/get-user.bru": `meta {
  name: get-user
  type: http
}

get {
  url: https://example.com/user
}
`,
		"health.bru": `meta {
  name: Health
  type: http
}

get {
  url: https://example.com/health
}
`,
	})
	defer gitServer.Close()

	client, err := gitrepo.New(gitrepo.Config{
		RepoURL:      gitServer.URL + "/group/repository",
		Branch:       "main",
		Token:        "gitlab-token",
		HTTPClient:   gitServer.Client(),
		APIBaseURL:   gitServer.URL,
		MaxTreeFiles: 10000,
		MaxFileBytes: maxGitBrunoFileBytes,
	})
	if err != nil {
		t.Fatal(err)
	}
	files, err := gitsync.FetchRepository(context.Background(), client)
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	roots := gitsync.Discover(files)
	parsed := parseRepositoryRoots(roots, "Imported Bruno")
	if len(parsed.Collections) == 0 {
		t.Fatalf("no collections, errors=%v", parsed.Errors)
	}
	col := parsed.Collections[0]
	_, reqs := countTree(col)
	if reqs < 4 {
		t.Fatalf("expected at least 4 requests, got %d", reqs)
	}
	hub := findChild(col, "Hub-Wallet")
	if hub == nil || len(hub.Requests) != 2 {
		t.Fatalf("hub wallet requests=%+v", hub)
	}
}

func TestGitNestedThreeLevels(t *testing.T) {
	gitServer := newGitLabFileServer(t, map[string]string{
		"collection.bru": "meta {\n  name: API\n  type: collection\n}\n",
		"Hub-Wallet/folder.bru": "meta {\n  name: Hub-Wallet\n}\n",
		"Hub-Wallet/Requests/folder.bru": "meta {\n  name: Requests\n}\n",
		"Hub-Wallet/Requests/login.bru": "meta {\n  name: login\n  type: http\n}\npost {\n  url: https://example.com/login\n}\n",
		"health.bru": "meta {\n  name: Health\n  type: http\n}\nget {\n  url: https://example.com/health\n}\n",
	})
	defer gitServer.Close()
	client, _ := gitrepo.New(gitrepo.Config{
		RepoURL: gitServer.URL + "/group/repository", Branch: "main", Token: "gitlab-token",
		HTTPClient: gitServer.Client(), APIBaseURL: gitServer.URL, MaxTreeFiles: 10000, MaxFileBytes: maxGitBrunoFileBytes,
	})
	files, _ := gitsync.FetchRepository(context.Background(), client)
	roots := gitsync.Discover(files)
	parsed := parseRepositoryRoots(roots, "Imported")
	col := parsed.Collections[0]
	_, reqs := countTree(col)
	if reqs < 2 {
		t.Fatalf("expected 2 requests, got %d tree=%+v errors=%v", reqs, col, parsed.Errors)
	}
	hub := findChild(col, "Hub-Wallet")
	if hub == nil {
		t.Fatal("missing Hub-Wallet")
	}
	reqsFolder := findChild(*hub, "Requests")
	if reqsFolder == nil || len(reqsFolder.Requests) != 1 {
		t.Fatalf("missing nested Requests folder: %+v", reqsFolder)
	}
}

func TestGitNestedCollectionBruInFolder(t *testing.T) {
	gitServer := newGitLabFileServer(t, map[string]string{
		"collection.bru": "meta {\n  name: Root API\n  type: collection\n}\n",
		"health.bru": "meta {\n  name: Health\n  type: http\n}\nget {\n  url: https://example.com/health\n}\n",
		"Hub-Wallet/collection.bru": "meta {\n  name: Hub-Wallet\n  type: collection\n}\n",
		"Hub-Wallet/login.bru": "meta {\n  name: login\n  type: http\n}\npost {\n  url: https://example.com/login\n}\n",
	})
	defer gitServer.Close()
	client, _ := gitrepo.New(gitrepo.Config{
		RepoURL: gitServer.URL + "/group/repository", Branch: "main", Token: "gitlab-token",
		HTTPClient: gitServer.Client(), APIBaseURL: gitServer.URL, MaxTreeFiles: 10000, MaxFileBytes: maxGitBrunoFileBytes,
	})
	files, _ := gitsync.FetchRepository(context.Background(), client)
	roots := gitsync.Discover(files)
	if len(roots) != 2 {
		t.Fatalf("expected 2 roots, got %d", len(roots))
	}
	parsed := parseRepositoryRoots(roots, "Imported")
	if len(parsed.Collections) != 2 {
		t.Fatalf("collections=%d errors=%v", len(parsed.Collections), parsed.Errors)
	}
	var rootReqs, hubReqs int
	for _, col := range parsed.Collections {
		_, reqs := countTree(col)
		if col.Name == "Imported" || col.Name == "Root API" {
			rootReqs = reqs
		}
		if col.Name == "Hub-Wallet" {
			hubReqs = reqs
		}
	}
	if rootReqs != 1 {
		t.Fatalf("root reqs=%d", rootReqs)
	}
	if hubReqs != 1 {
		t.Fatalf("hub reqs=%d", hubReqs)
	}
}
