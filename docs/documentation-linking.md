# Documentation linking

Postchi connects API requests, OpenAPI specs, and team markdown docs in one place. When you open a request's **Docs** tab or browse the **API catalog**, you see a single merged view of everything relevant to that endpoint.

## What gets linked

A request's documentation bundle (`GET /api/requests/:id/docs-bundle`) combines:

| Source | How it appears | Editable in Postchi? |
|--------|----------------|----------------------|
| **OpenAPI `api_doc`** | Parameters, request body, responses, tags | No (synced from spec) |
| **Request notes** | Team markdown on the request | Yes |
| **Frontmatter-linked docs** | Workspace pages whose YAML lists the request's `source_operation_id` | Content synced from git; links are automatic |
| **Manual doc links** | Pages explicitly linked via the doc/request pickers | Link can be added or removed in the UI |

Linked workspace docs show an **auto** badge (frontmatter) and/or a **manual** badge. Manual links can be unlinked from the request panel without deleting the doc page.

## Documentation workspace

Open **Documentation** from the workspace toolbar (`/workspaces/:id/docs`).

### Views

- **Tree** — browse synced and local pages by folder structure
- **Edit / Preview / Split** — write markdown with autosave
- **Graph** — visual map of doc pages, OpenAPI operations, and manual links

### Local pages

Editors can create pages directly in Postchi. Local pages are marked `is_local` and are not overwritten by git sync.

### Git sync (GitHub & GitLab)

In **Settings → Documentation**, add a git source:

1. Paste a repository URL (browser links like `/-/tree/main/docs` are supported)
2. Set branch (default: `main`) and optional path prefix
3. Provide a personal access token for private repos
4. Click **Sync now** or rely on periodic sync

| Provider | Token scopes |
|----------|--------------|
| GitHub | `contents:read` |
| GitLab | `read_api`, `read_repository` (Reporter role or higher) |

Synced markdown files become workspace docs keyed by slug (path segments joined with `-`).

### Frontmatter operation links

Git-synced pages can declare which OpenAPI operations they document using YAML frontmatter:

```markdown
---
title: Create user
operations: [post-/users, get-/users/{id}]
---

# Create user

Describe the endpoint for your team…
```

During sync, Postchi parses `operations:` and stores them on the workspace doc. Any request whose `source_operation_id` matches one of those values automatically appears in that request's linked docs (badge: **auto**).

Operation IDs follow the same format Postchi assigns when syncing OpenAPI specs into collections.

## Linking docs and requests in the UI

### From a request (Docs tab)

1. Open a request and switch to **Docs**
2. Edit **Documentation notes** for team-owned markdown (won't be overwritten by OpenAPI sync)
3. Click **Link doc page** to attach an existing workspace doc (**manual** link)
4. Use **Preview** for an in-panel modal, or **Open doc** to jump to the documentation workspace

### From a doc page

The **Linked requests** sidebar lists requests manually linked to the current page. Click **Link** to search workspace requests (by name, URL, method, or `source_operation_id`).

### From the API catalog

The workspace **Catalog** (`/workspaces/:id/catalog`) lists all requests with documentation coverage indicators. Select an endpoint to view the same docs bundle and edit notes or links without opening the full request builder.

## API endpoints

### Workspace docs

```http
GET    /api/workspaces/:id/workspace-docs?summary=1
POST   /api/workspaces/:id/workspace-docs
GET    /api/workspaces/:id/workspace-docs/:slug
PATCH  /api/workspaces/:id/workspace-docs/:slug
GET    /api/workspaces/:id/workspace-docs/:docId/links
POST   /api/workspaces/:id/workspace-docs/:docId/links
DELETE /api/workspaces/:id/workspace-docs/:docId/links/:linkId
GET    /api/workspaces/:id/doc-graph
```

Create link body (one of):

```json
{ "request_id": "<uuid>" }
```

```json
{ "operation_id": "get-/users/{id}" }
```

### Doc sources & sync

```http
GET    /api/workspaces/:id/doc-sources
POST   /api/workspaces/:id/doc-sources
PATCH  /api/workspaces/:id/doc-sources/:sourceId
DELETE /api/workspaces/:id/doc-sources/:sourceId
POST   /api/doc-sources/:id/sync
```

### Per-request bundle

```http
GET /api/requests/:id/docs-bundle
```

Response shape:

```json
{
  "api_doc": { },
  "description": "Team notes markdown…",
  "linked_workspace_docs": [
    {
      "id": "…",
      "slug": "guides-auth",
      "title": "Authentication guide",
      "content_md": "…",
      "link_sources": ["frontmatter", "manual"],
      "link_id": "…"
    }
  ]
}
```

### Catalog

```http
GET /api/workspaces/:id/catalog
GET /api/collections/:id/catalog
```

Query filters: `tag`, `method`, `documented` (`true` / `false`).

## CI: push OpenAPI from pipelines

Generate a workspace token in **Settings → Documentation → CI automation tokens**, then push specs from CI:

```http
POST /api/workspaces/:id/api-specs/push
Authorization: Bearer <workspace-token>
```

Scope required: `spec:push`. This keeps collections and `api_doc` payloads up to date alongside git-synced markdown.

## Tips

- Manual request notes on a synced request mark documentation as team-owned and prevent OpenAPI sync from overwriting the description field.
- Use the doc graph to spot orphaned pages or endpoints missing guides.
- Share read-only catalog snapshots via share links (`kind: catalog`) for external reviewers.
