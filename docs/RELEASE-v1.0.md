# Postchi v1.0: Self-Hosted API Collaboration Platform

The first release of **Postchi**, a self-hosted API collaboration platform for teams. Run requests, share collections, sync OpenAPI specs and Bruno collections from Git, link markdown documentation to your API surface, and collaborate in real time. All on your own infrastructure.

---

## Highlights

### Workspaces & collaboration

- Multi-user workspaces with **viewer**, **editor**, and **owner** roles
- **Real-time collaboration** over WebSocket when teammates edit the same workspace
- **Activity feed** per workspace
- **Team onboarding**: add registered users immediately; invite others via copyable link or optional email (SMTP)
- **Registration domain allowlist**: optional `REGISTRATION_ALLOWED_EMAIL_DOMAINS` for internal-only signup
- **Change password**: authenticated users can rotate credentials; other sessions are revoked
- **Share links** for read-only access or one-click import into another workspace
- **Catalog share links**: read-only snapshots of the API catalog and linked documentation

### Request builder & execution

- Full HTTP method support with tabs for **Params**, **Headers**, **Body**, **Auth**, **Scripts**, and **Settings**
- Body modes: raw (JSON/XML/text), **GraphQL**, **form-data** (with file uploads), and URL-encoded
- Auth types: none, Basic, Bearer token, API key (header or query), and inherit from parent
- Per-request settings: timeout, follow redirects, SSL verification
- **Template / variant model**: define a template request and spawn variants with field-level inheritance, reset, and push-to-children
- Drag-and-drop reordering in the request tree
- Variable autocomplete with `{{variable}}` interpolation; built-in `$timestamp` and `$isoTimestamp`
- Send requests with detailed **timing breakdown** (DNS, connect, TLS, TTFB, download)
- Response viewer with JSON tree, raw body, and headers
- **Pre-request scripts** and **post-response test scripts** (JavaScript via goja)
- **Collection runner** with pass/fail reporting across all requests in a collection
- Save response examples; generate **cURL** (and other) code snippets from saved requests
- Default timeout: 30s · Max response size: 10 MB

### Environments & variables

- Workspace- and collection-scoped environments
- **Encrypted secret storage** (AES-256 via server-side encryption key)
- Variable resolution across workspace, collection, environment, and runtime scopes
- Bulk variable import and environment URL mapping for OpenAPI sync

### OpenAPI & spec sync

- Connect OpenAPI 3 specs by URL
- Sync operations into collections with diff view (added, updated, removed)
- Per-environment base URL mapping; track source operation IDs for incremental updates
- **CI spec push**: workspace API tokens with `spec:push` scope for pipelines (`POST /api/workspaces/:id/api-specs/push`)

### Bruno & Git collection sync

- **Persistent Bruno Git sources**: sync `.bru` files from GitHub or GitLab on demand
- **One-shot Git import** for Bruno and OpenCollection trees (no persistent source required)
- **Incremental sync** with added/updated/removed request reporting
- Configurable branch, path prefix, and private-repo access tokens
- **Operation ID backfill** for legacy Bruno imports (canonical `method-/path` IDs for doc linking)
- **Import into parent collection**: choose a target folder when importing collections

### Documentation linking

- **Markdown documentation workspace** with tree navigation, live preview, split edit, and link graph
- **GitHub & GitLab sync** for markdown repos (public or private with PAT)
- **Unified docs bundle** per request: OpenAPI `api_doc`, team notes, frontmatter-linked pages, and manual doc links
- **Bidirectional linking**: attach doc pages from a request, or link requests from a doc page
- **In-request preview**: compact linked-doc cards with preview modal and **Open doc** shortcut
- **API catalog**: browse all endpoints with documentation coverage; edit docs without opening the full builder
- **Deterministic auto-linking** on git doc sync:
  - Exact name match (`get-user` request ↔ `get-user.md`) when unique
  - Frontmatter `request` / `requests`: declare request names in doc YAML for explicit auto-links
  - Frontmatter `operations`: auto-link pages to matching OpenAPI/Bruno operations
  - Configurable **path template** per doc source (e.g. `docs/{collection_slug}/{request_slug}.md`)
  - Optional **API collection** scope on doc sources to avoid cross-collection collisions
- **Smart suggestions**: tightened fuzzy heuristics limited to suggestions only; accept/reject in panel with bulk accept for high-confidence matches

### Collections & catalog

- **Nested collections** with folder hierarchy and drag-and-drop reordering
- Collection duplication; per-collection docs and catalog views

### Import & export

| Format | Import | Export |
|--------|:------:|:------:|
| Postman Collection v2.1 | ✓ | ✓ |
| Bruno (ZIP) | ✓ | ✓ |
| Bruno (Git sync) | ✓ | - |
| OpenAPI 3 | ✓ | - |
| OpenCollection | ✓ | - |
| OpenCollection / Bruno (Git one-shot) | ✓ | - |
| cURL | ✓ | - |

### UI & theming

- Modern Nuxt 4 + Vue 3 interface with shadcn-style components
- CodeMirror 6 editors for JSON, JavaScript, XML, and markdown
- Resizable panes, tabbed request workspace, method-colored badges
- **10 themes** with dark and light variants (Tokyo Night, Nord, GitHub, Rosé Pine, Kanagawa, One Dark)

### Deployment & operations

- Docker-based deployment with **automatic database migrations** on API startup
- Optional standalone migrate job for fleet rollouts
- **Health** (`GET /health`) and **readiness** (`GET /ready`) endpoints
- Single-container option: backend can serve built Nuxt static files via `STATIC_FILES_PATH`
- Mailpit included in docker-compose for local invite-email testing

---

## Docker images

```bash
docker pull ghcr.io/imaanmzr/postchi-api:v1.0
docker pull ghcr.io/imaanmzr/postchi-web:v1.0
```

---

## Requirements

- Docker and Docker Compose
- PostgreSQL 16 or compatible PostgreSQL version
- Go 1.26+ and Node.js 24.19.0 for local development

---

## Production configuration

Set these before deploying:

| Variable | Description |
|----------|-------------|
| `JWT_SECRET` | Secret for signing access tokens (min 32 chars) |
| `ENCRYPTION_KEY` | Exactly 32 bytes for encrypting secrets at rest |
| `DATABASE_URL` | PostgreSQL connection string |

Optional but recommended:

- `PUBLIC_APP_URL` / `APP_PUBLIC_URL`: base URL for invite and share links
- `CORS_ORIGINS`: browser origins allowed to call the API
- `REGISTRATION_ALLOWED_EMAIL_DOMAINS`: restrict self-registration to specific email domains
- SMTP settings for sending invite emails (copyable invite links work without SMTP)

See the [README](../README.md) for installation, configuration, and deployment instructions.

---

## Known limitations (v1 scope)

- Concurrent edits use last-write-wins (no OT/CRDT)
- OAuth2, AWS SigV4, Digest auth: partial or stub implementations
- WebSocket, SSE, gRPC protocol testing: planned for v2
- Mock servers: planned for v2
- SSO/OIDC: deferred (auth interface ready for extension)
- Bruno post-response Expr runtime: JSON-path subset only (`res.body.foo`, `res.status`)

---

## Documentation

- [README](../README.md), installation, configuration, and deployment
- [Documentation linking](documentation-linking.md)
- [Team members & invites](team-members.md)
