# Documentation linking

Postchi connects API requests, OpenAPI specs, Bruno collections, and team markdown docs in one place. When you open a request's **Docs** tab or browse the **API catalog**, you see a single merged view of everything relevant to that endpoint.

## What gets linked

A request's documentation bundle (`GET /api/requests/:id/docs-bundle`) combines:

| Source | How it appears | Editable in Postchi? |
|--------|----------------|----------------------|
| **OpenAPI `api_doc`** | Parameters, request body, responses, tags | No (synced from spec) |
| **Request notes** | Team markdown on the request | Yes |
| **Frontmatter-linked docs** | Workspace pages whose YAML lists matching operation IDs | Content synced from git; links are automatic |
| **Manual doc links** | Pages explicitly linked via the doc/request pickers | Link can be added or removed in the UI |
| **Suggested links** | Heuristic matches surfaced by **Analyze doc links** | Accept to create a manual link; reject to dismiss |

Linked workspace docs show an **auto** badge (frontmatter) and/or a **manual** badge. Manual links can be unlinked from the request panel without deleting the doc page.

## Operation IDs

Postchi matches docs to requests using **operation IDs**. Several formats are supported and normalized automatically:

| Source | `source_operation_id` format | Example |
|--------|------------------------------|---------|
| OpenAPI sync | Spec `operationId`, or fallback `METHOD /path` | `createUser` or `post /users` |
| Bruno import | Canonical method-path | `post-/users/{id}` |
| Doc frontmatter | Any of the above (aliases stored) | `post-/users`, `createUser` |

**Canonical method-path** (used for Bruno and recommended in frontmatter):

```
{lowercase-method}-/{normalized-path}
```

Examples: `get-/users`, `post-/users/{id}`

- Strips `{{baseUrl}}` and query strings
- Normalizes Bruno `:id` path params to `{id}`

### Bruno collections

New Bruno imports (ZIP or Git) assign `source_operation_id` automatically. For collections imported before this feature, run **Backfill API operation IDs** in **Settings → Documentation**.

## Documentation workspace

Open **Documentation** from the workspace toolbar (`/workspaces/:id/docs`).

### Views

- **Tree** — browse synced and local pages by folder structure
- **Edit / Preview / Split** — write markdown with autosave
- **Graph** — visual map of doc pages, operations, auto links, manual links, and pending suggestions
- **Suggestions** — review heuristic doc ↔ API matches (header button with pending count)

### Local pages

Editors can create pages directly in Postchi. Local pages are marked `is_local` and are not overwritten by git sync.

### Git sync (GitHub & GitLab)

In **Settings → Documentation**, add a git source:

1. Paste a repository URL (browser links like `/-/tree/main/docs` are supported)
2. Set branch (default: `main`) and optional path prefix
3. Provide a personal access token for private repos
4. Click **Sync now** or rely on periodic sync
5. Optionally enable **Analyze after sync** to refresh link suggestions

| Provider | Token scopes |
|----------|--------------|
| GitHub | `contents:read` |
| GitLab | `read_api`, `read_repository` (Reporter role or higher) |

Synced markdown files become workspace docs keyed by slug (path segments joined with `-`).

### Frontmatter operation links

Git-synced pages can declare which operations they document using YAML frontmatter:

```markdown
---
title: Create user
operations:
  - post-/users
  - get-/users/{id}
---

# Create user

Describe the endpoint for your team…
```

Inline arrays and quoted values are also supported:

```yaml
operations: [post-/users, "get-/users/{id}"]
```

During sync, Postchi parses `operations:`, normalizes aliases, and stores them on the workspace doc. Any request whose `source_operation_id` (or canonical method-path alias) matches automatically appears in that request's linked docs (badge: **auto**).

You can also reference OpenAPI `operationId` values directly when your spec uses custom IDs:

```yaml
operations: [createUser, listUsers]
```

## Smart suggestions (human review)

Suggestions **never auto-apply**. Run **Analyze doc links** from Settings or the Documentation **Suggestions** panel to find likely matches using:

| Heuristic | Confidence |
|-----------|------------|
| HTTP `METHOD /path` in doc body matches a request | high |
| Doc file path aligns with request URL segment | high |
| Doc title/slug similar to request name | medium |
| Doc folder mirrors collection structure | medium |

Review each suggestion in the **Suggestions** panel or the doc page **Linked requests** sidebar. **Accept** creates a manual link; **Reject** dismisses it. Use **Accept all high** for bulk review of high-confidence matches.

## Linking docs and requests in the UI

### From a request (Docs tab)

1. Open a request and switch to **Docs**
2. Edit **Documentation notes** for team-owned markdown (won't be overwritten by OpenAPI sync)
3. Click **Link doc page** to attach an existing workspace doc (**manual** link)
4. Use **Preview** for an in-panel modal, or **Open doc** to jump to the documentation workspace

### From a doc page

The **Linked requests** sidebar lists **auto**, **manual**, and **suggested** links. Click **Link** to search workspace requests (by name, URL, method, or `source_operation_id`). Only manual links can be removed with ×; suggestions offer Accept/Reject actions.

### From the API catalog

The workspace **Catalog** (`/workspaces/:id/catalog`) lists all requests with documentation coverage indicators (including frontmatter-linked docs). Select an endpoint to view the same docs bundle and edit notes or links without opening the full request builder.

## API endpoints

### Workspace docs

```http
GET    /api/workspaces/:id/workspace-docs?summary=1
POST   /api/workspaces/:id/workspace-docs
GET    /api/workspaces/:id/workspace-docs/:slug
PATCH  /api/workspaces/:id/workspace-docs/:slug
GET    /api/workspaces/:id/workspace-docs/:docId/request-links
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

### Doc link analysis

```http
POST   /api/workspaces/:id/doc-links/analyze
GET    /api/workspaces/:id/doc-links/suggestions?status=pending
POST   /api/workspaces/:id/doc-links/suggestions/:id/accept
POST   /api/workspaces/:id/doc-links/suggestions/:id/reject
POST   /api/workspaces/:id/doc-links/suggestions/accept-all?confidence=high
```

### Bruno operation ID backfill

```http
POST /api/workspaces/:id/requests/backfill-operation-ids
```

Updates requests with empty `source_operation_id` that were not synced from OpenAPI (Bruno/manual imports).

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
- After importing Bruno collections, run **Backfill API operation IDs** if auto frontmatter links don't appear immediately.
