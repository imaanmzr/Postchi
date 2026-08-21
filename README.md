<p align="center">
  <img src="docs/brand/postchilogo.png" alt="Postchi" width="280">
</p>

<p align="center">
  <strong>Self-hosted API client for teams.</strong> Run requests, share collections, sync OpenAPI specs, and collaborate in real time. All on your own infrastructure.
</p>

<p align="center">
  Postchi is a multi-user, web-based alternative to Postman and Bruno built for internal teams who want full control over their API tooling, data, and secrets.
</p>

![Postchi request builder with JSON tree response viewer](screenshots/scrsht-01.png)

---

## Table of Contents

- [Features](#features)
- [Screenshots](#screenshots)
- [Branding](#branding)
- [Architecture](#architecture)
- [Quick Start](#quick-start)
- [Configuration](#configuration)
- [Local Development](#local-development)
- [Deployment](#deployment)
- [Import & Export](#import--export)
- [Documentation linking](#documentation-linking)
- [Variable Precedence](#variable-precedence)
- [Roles & Permissions](#roles--permissions)
- [API Reference](#api-reference)
- [Known Limitations](#known-limitations)
- [Tech Stack](#tech-stack)

---

## Features

### Workspaces & Collaboration

- **Multi-user workspaces** with role-based access control (viewer, editor, owner)
- **Real-time collaboration** over WebSocket when teammates edit the same workspace
- **Email invites** with configurable TTL (Mailpit included for local dev)
- **Share links** for read-only access or one-click import into another workspace
- **Catalog share links** — publish read-only snapshots of the API catalog and linked documentation
- **Activity feed** per workspace
- **Nested collections** — folder hierarchy with drag-and-drop reordering

![Workspace settings with Git markdown docs sync and CI automation tokens](screenshots/scrsht-02.png)

### Request Builder

- Full HTTP method support (GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS)
- Tabs for **Params**, **Headers**, **Body**, **Auth**, **Scripts**, and **Settings**
- Body modes: raw (JSON/XML/text), GraphQL, form-data (with file uploads), and URL-encoded
- Auth types: none, Basic, Bearer token, API key (header or query), and inherit from parent
- Per-request settings: timeout, follow redirects, SSL verification
- **Template / variant model**: define a template request and spawn variants that inherit fields
- Field-level inheritance with reset and push-to-children
- Drag-and-drop reordering in the request tree
- Variable autocomplete with `{{variable}}` interpolation
- Built-in placeholders: `$timestamp`, `$isoTimestamp`

### Environments & Variables

- Workspace and collection scoped environments
- Encrypted secret storage (AES-256 via server-side encryption key)
- Variable resolution across workspace, collection, environment, and runtime scopes
- Bulk variable import and environment URL mapping for OpenAPI sync

### Execution & Testing

- Send requests from the browser with detailed timing breakdown (DNS, connect, TLS, TTFB, download)
- Response viewer with JSON tree, raw body, and headers
- **Pre-request scripts** and **post-response test scripts** (JavaScript via goja)
- **Collection runner** with pass/fail report across all requests in a collection
- Save response examples back to a request
- Generate code snippets (cURL and more) from saved requests
- Default timeout: 30s | Max response size: 10 MB

### OpenAPI & Spec Sync

- Connect OpenAPI 3 specs by URL
- Sync operations into collections with diff view (added, updated, removed)
- Per-environment base URL mapping
- Track source operation IDs for incremental updates
- **CI spec push** — workspace API tokens with `spec:push` scope for pipelines (`POST /api/workspaces/:id/api-specs/push`)

### Bruno & Git Collection Sync

Keep Bruno collections in sync with your repository instead of one-off ZIP imports.

- **Bruno Git sources** — connect GitHub or GitLab repos; sync `.bru` files into workspace collections on demand
- **One-time Git import** — import a Bruno or OpenCollection tree from a repo URL without creating a persistent source
- **Incremental sync** — added, updated, and removed requests tracked via content hashes
- **Operation ID backfill** — assign canonical `method-/path` IDs to legacy Bruno imports for doc linking
- Configurable branch, path prefix, and private-repo access tokens (same PAT scopes as doc sync)

Configure sources under **Settings → API Sync → Sync Bruno collection from Git**.

### Documentation linking

Keep API knowledge next to the requests your team actually runs.

- **Markdown documentation workspace** with tree navigation, live preview, split edit, and a link graph
- **GitHub & GitLab sync** for markdown repos (public or private with PAT)
- **Unified docs bundle** per request: OpenAPI `api_doc`, team notes, frontmatter-linked pages, and manual doc links
- **Bidirectional linking** — attach doc pages from a request, or link requests from a doc page
- **In-request preview** — compact linked-doc cards with a centered preview modal and **Open doc** shortcut
- **API catalog** — browse all endpoints with documentation coverage and edit docs without opening the full builder
- **Frontmatter operation links** — `operations:` in synced markdown auto-links pages to matching OpenAPI/Bruno operations
- **Frontmatter request links** — `request:` or `requests:` in markdown links pages to collection requests by name
- **Deterministic auto-linking** on git doc sync:
  - Exact name match (`get-user` request ↔ `get-user.md`)
  - Configurable **path template** per doc source (e.g. `docs/{collection_slug}/{request_slug}.md`)
  - Optional **API collection** scope on doc sources to avoid cross-collection collisions
- **Smart suggestions** — tightened heuristics for ambiguous cases; accept/reject in the Suggestions panel
- **Catalog share links** for read-only documentation snapshots

See [docs/documentation-linking.md](docs/documentation-linking.md) for setup, frontmatter format, path templates, and API details.

![Per-request documentation notes with linked doc pages](screenshots/scrsht-03.png)

![In-app doc preview modal with Open in docs shortcut](screenshots/scrsht-04.png)

![Markdown documentation workspace with edit/preview split and linked requests](screenshots/scrsht-05.jpg)

### Import & Export

| Format | Import | Export |
|--------|:------:|:------:|
| Postman Collection v2.1 | Yes | Yes |
| Bruno (ZIP) | Yes | Yes |
| Bruno (Git sync) | Yes | - |
| OpenAPI 3 | Yes | - |
| OpenCollection | Yes | - |
| OpenCollection / Bruno (Git one-shot) | Yes | - |
| cURL | Yes | - |

### UI & Theming

- Modern Nuxt 4 + Vue 3 interface with shadcn-style components
- CodeMirror 6 editors for JSON, JavaScript, XML, and markdown
- Resizable panes, tabbed request workspace, method-colored badges
- **10 themes** with dark and light variants:
  - Tokyo Night / Tokyo Night Day
  - Nord / Nord Light
  - GitHub Dark / GitHub Light
  - Rosé Pine / Rosé Pine Dawn
  - Kanagawa / One Dark

---

## Screenshots

| | |
|---|---|
| Request builder with JSON tree response | ![Request builder](screenshots/scrsht-01.png) |
| Workspace settings — Git docs sync & CI tokens | ![Workspace settings](screenshots/scrsht-02.png) |
| Per-request docs & linked pages | ![Request docs](screenshots/scrsht-03.png) |
| In-app doc preview modal | ![Doc preview](screenshots/scrsht-04.png) |
| Markdown docs workspace (edit / preview) | ![Docs workspace](screenshots/scrsht-05.jpg) |

---

## Branding

Postchi brand assets live under [`frontend/public/brand/`](frontend/public/brand/) and [`docs/brand/`](docs/brand/).

| Asset | Path | Use |
|-------|------|-----|
| Wordmark | [`docs/brand/postchilogo.png`](docs/brand/postchilogo.png) | README, docs, presentations |
| Logo mark (SVG) | [`frontend/app/assets/brand/logo-mark.svg`](frontend/app/assets/brand/logo-mark.svg) | In-app header, compact placements |
| Wordmark (SVG) | [`frontend/app/assets/brand/logo-wordmark.svg`](frontend/app/assets/brand/logo-wordmark.svg) | In-app branding, scalable layouts |
| App icon (SVG) | [`frontend/public/brand/icon.svg`](frontend/public/brand/icon.svg) | Favicon source, general icon |
| App icons (PNG) | `icon-192.png`, `icon-512.png`, `icon-maskable-512.png` | PWA, home-screen, maskable tiles |
| Favicons | `favicon.ico`, `favicon-16.png`, `favicon-32.png`, `favicon-48.png` | Browser tabs |
| Apple touch icon | [`apple-touch-icon.png`](frontend/public/brand/apple-touch-icon.png) | iOS home screen |
| Open Graph image | [`og-image.png`](frontend/public/brand/og-image.png) | Social / link previews |
| Web manifest | [`site.webmanifest`](frontend/public/brand/site.webmanifest) | PWA metadata (`theme_color`: `#1a1b26`) |

<p align="center">
  <img src="frontend/public/brand/icon-512.png" alt="Postchi icon" width="96">
</p>

---

## Architecture

```mermaid
flowchart LR
  subgraph client [Browser]
    UI[Nuxt Frontend]
  end

  subgraph server [Postchi Server]
    API[Go API / chi]
    WS[WebSocket Hub]
    EX[Request Executor]
    SCR[Script Runtime]
  end

  subgraph data [Data Layer]
    PG[(PostgreSQL 16)]
  end

  subgraph external [External]
    SMTP[SMTP / Mailpit]
    Target[Target APIs]
  end

  UI -->|REST /api| API
  UI -->|/api/ws| WS
  API --> PG
  WS --> PG
  API --> EX
  EX --> SCR
  EX --> Target
  API --> SMTP
```

| Service | Port | Description |
|---------|------|-------------|
| Frontend | 3000 | Nuxt web UI |
| Backend | 8080 | Go REST API + WebSocket |
| PostgreSQL | 5432 | Primary data store |
| Mailpit | 8025 (UI), 1025 (SMTP) | Local email capture |

---

## Quick Start

**Requirements:** Docker and Docker Compose

```bash
git clone https://github.com/imaanmzr/postchi.git
cd postchi

cp .env.example .env
# Set JWT_SECRET and ENCRYPTION_KEY (exactly 32 bytes) in .env

docker compose up --build
```

Or with Make:

```bash
make setup          # create .env from .env.example
make tools          # install golang-migrate CLI (once)
make docker-up      # full stack in Docker
```

| Endpoint | URL |
|----------|-----|
| Frontend | http://localhost:3000 |
| Backend API | http://localhost:8080 |
| Health check | http://localhost:8080/health |
| Mailpit (local dev) | http://localhost:8025 |

Register the first user at the login page. That account becomes the workspace owner.

---

## Configuration

Copy `.env.example` to `.env` and adjust values for your environment.

### Required for Production

| Variable | Description |
|----------|-------------|
| `JWT_SECRET` | Secret for signing access tokens (min 32 chars) |
| `ENCRYPTION_KEY` | Exactly 32 bytes for encrypting secrets at rest |
| `DATABASE_URL` | PostgreSQL connection string |

### Server

| Variable | Default | Description |
|----------|---------|-------------|
| `HTTP_PORT` | `8080` | API listen port |
| `ENVIRONMENT` | `development` | Runtime environment label |
| `CORS_ORIGINS` | `http://localhost:3000` | Comma-separated allowed browser origins |
| `REQUEST_TIMEOUT_SECONDS` | `30` | Default outbound request timeout |
| `MAX_RESPONSE_BYTES` | `10485760` | Max response body size (10 MB) |

### URLs & Frontend

| Variable | Default | Description |
|----------|---------|-------------|
| `PUBLIC_APP_URL` | `http://localhost:3000` | URL teammates use in the browser |
| `NUXT_PUBLIC_API_URL` | *(empty)* | Browser-reachable API URL. Leave empty for same-origin `/api` behind a reverse proxy |
| `APP_PUBLIC_URL` | `http://localhost:3000` | Base URL for invite and share links |
| `NUXT_DEV_API_PROXY` | `http://localhost:8080` | Dev proxy target when `NUXT_PUBLIC_API_URL` is empty |

### Email (Invites)

| Variable | Default | Description |
|----------|---------|-------------|
| `SMTP_HOST` | *(empty)* | SMTP server (defaults to Mailpit in docker-compose) |
| `SMTP_PORT` | `587` | SMTP port (1025 for Mailpit) |
| `SMTP_USER` | *(empty)* | SMTP username |
| `SMTP_PASS` | *(empty)* | SMTP password |
| `SMTP_FROM` | `postchi@localhost` | From address |
| `INVITE_TTL_HOURS` | `168` | Invite link expiry (7 days) |
| `REGISTRATION_ALLOWED_EMAIL_DOMAINS` | *(empty)* | Comma-separated domains allowed for self-registration (e.g. `yourcompany.com`). Invites are not restricted. Empty = open registration. |

### Migrations

| Variable | Default | Description |
|----------|---------|-------------|
| `MIGRATIONS_PATH` | `file://migrations` | golang-migrate file source |
| `AUTO_MIGRATE` | `true` | Run pending migrations on API startup |
| `DB_READY_TIMEOUT` | `60s` | Wait for Postgres before migrate/connect |

The API image embeds SQL under `/migrations`. With `AUTO_MIGRATE=true` (default), pending migrations run on startup. Migration failures abort the process so a bad schema never serves traffic.

---

## Local Development

### Prerequisites

- Go 1.26+
- Node.js 24.19.0 (LTS)
- Docker (for Postgres)
- golang-migrate CLI (`make tools`)

### Make Targets

```bash
make help           # list all targets
make setup          # copy .env.example to .env
make install        # go mod download + npm install
make tools          # install golang-migrate CLI

make db-up          # start Postgres only (Docker)
make migrate-up     # apply SQL migrations
make migrate-down   # roll back one migration
make migrate-status # show current schema version
make migrate-create NAME=add_something  # scaffold a new migration

make backend        # run API on :8080 (migrates on startup)
make frontend       # run Nuxt dev on :3000
make dev            # db + migrate + backend & frontend together

make test           # go test + npm run test + frontend build
make build          # compile API binary + frontend bundle
```

### Manual Backend

```bash
cd backend
export DATABASE_URL=postgres://postchi:postchi@localhost:5432/postchi?sslmode=disable
export JWT_SECRET=dev-secret-change-in-production
export ENCRYPTION_KEY=dev-encryption-key-32-bytes!!
export MIGRATIONS_PATH=file://../migrations
go run ./cmd/api
```

### Manual Frontend

```bash
cd frontend
npm install
NUXT_PUBLIC_API_URL=http://localhost:8080 npm run dev
```

### Remote / LAN Access

When teammates access Postchi from other machines:

1. Set `CORS_ORIGINS` to every browser origin (e.g. `http://192.168.1.10:3000`)
2. Choose one approach:
   - **Same-origin (recommended):** put a reverse proxy in front of UI + API; leave `NUXT_PUBLIC_API_URL` empty
   - **Split ports:** set `NUXT_PUBLIC_API_URL=http://<server-ip>:8080` at frontend build time (Docker `ARG`) and runtime
3. Saved request URLs using `localhost` resolve inside the Docker container. Use the host LAN IP or `host.docker.internal` for local services.

See [docs/network-test-matrix.md](docs/network-test-matrix.md) for the acceptance test checklist.

Additional guides: [docs/README.md](docs/README.md) · [Documentation linking](docs/documentation-linking.md)

---

## Deployment

### Recommended: Auto-Migrate on Startup

Deploy the API container. It waits for the database, applies migrations, then serves traffic.

```bash
docker compose up --build -d
```

### Optional: Separate Migrate Job

If you prefer migrate-once before rolling the API fleet:

```bash
docker run --rm \
  -e DATABASE_URL="$DATABASE_URL" \
  -e MIGRATIONS_PATH=file:///migrations \
  -e DB_READY_TIMEOUT=60s \
  postchi-api /app/postchi-migrate
```

Then start the API with `AUTO_MIGRATE=false` so replicas do not race. golang-migrate uses advisory locks, so concurrent auto-migrate on multiple replicas is still safe if you keep the default.

### Health Probes

| Endpoint | Purpose |
|----------|---------|
| `GET /health` | Liveness (always 200 if process is up) |
| `GET /ready` | Readiness (checks database connectivity) |

### Single Binary + Static Frontend

The backend can serve the built Nuxt static files when `STATIC_FILES_PATH` is set. This enables a single-container deployment behind one reverse proxy.

---

## Import & Export

Import endpoints accept multipart uploads, JSON payloads, or Git repository URLs depending on format. All imports target a workspace and optionally a parent collection.

```http
POST /api/import/postman
POST /api/import/bruno
POST /api/import/openapi
POST /api/import/opencollection
POST /api/import/curl
POST /api/workspaces/:id/imports/bruno/git
POST /api/workspaces/:id/imports/git

GET  /api/export/postman?collection_id=<uuid>
GET  /api/export/bruno?collection_id=<uuid>
```

OpenAPI import creates both a synced collection and an API spec record for ongoing sync.

---

## Variable Precedence

When resolving `{{variables}}`, higher entries override lower ones:

| Priority | Source |
|:--------:|--------|
| 1 (highest) | Runtime (`pm.variables.set` in pre-request script) |
| 2 | Data file (collection runner iteration) |
| 3 | Environment variables |
| 4 | Collection variables |
| 5 | Workspace variables |
| 6 (lowest) | Built-in (`$timestamp`, `$isoTimestamp`) |

---

## Roles & Permissions

| Role | Read | Edit | Execute | Delete | Manage members |
|------|:----:|:----:|:-------:|:------:|:--------------:|
| **viewer** | Yes | No | Yes | No | No |
| **editor** | Yes | Yes | Yes | Own | No |
| **owner** | Yes | Yes | Yes | Yes | Yes |

Owners can add teammates (registered users are added immediately; others get an invite link to share manually or by email), revoke invites, and manage workspace settings. Editors can create and modify collections, requests, and environments within the workspace.

---

## API Reference

All authenticated routes require a Bearer token from `POST /api/auth/login` or `POST /api/auth/register`.

### Auth

```
GET  /api/config/public
POST /api/auth/register
POST /api/auth/login
POST /api/auth/refresh
POST /api/auth/logout
GET  /api/auth/me
```

`GET /api/config/public` returns `smtp_configured` and `registration_allowed_domains` (no authentication required).

### Workspaces

```
GET    /api/workspaces
POST   /api/workspaces
GET    /api/workspaces/:id
PATCH  /api/workspaces/:id
DELETE /api/workspaces/:id
GET    /api/workspaces/:id/members
POST   /api/workspaces/:id/members
PATCH  /api/workspaces/:id/members/:userId
DELETE /api/workspaces/:id/members/:userId
GET    /api/workspaces/:id/activity
GET    /api/workspaces/:id/collections
GET    /api/workspaces/:id/requests
GET    /api/workspaces/:id/shares
GET    /api/workspaces/:id/catalog
GET    /api/workspaces/:id/workspace-docs
POST   /api/workspaces/:id/workspace-docs
GET    /api/workspaces/:id/workspace-docs/{docId}/request-links
GET    /api/workspaces/:id/workspace-docs/{docId}/links
POST   /api/workspaces/:id/workspace-docs/{docId}/links
DELETE /api/workspaces/:id/workspace-docs/{docId}/links/{linkId}
GET    /api/workspaces/:id/workspace-docs/{slug}
PATCH  /api/workspaces/:id/workspace-docs/{slug}
GET    /api/workspaces/:id/doc-graph
POST   /api/workspaces/:id/doc-links/analyze
GET    /api/workspaces/:id/doc-links/suggestions
POST   /api/workspaces/:id/doc-links/suggestions/accept-all
POST   /api/workspaces/:id/doc-links/suggestions/{suggestionId}/accept
POST   /api/workspaces/:id/doc-links/suggestions/{suggestionId}/reject
POST   /api/workspaces/:id/requests/backfill-operation-ids
GET    /api/workspaces/:id/doc-sources
POST   /api/workspaces/:id/doc-sources
PATCH  /api/workspaces/:id/doc-sources/{sourceId}
DELETE /api/workspaces/:id/doc-sources/{sourceId}
GET    /api/workspaces/:id/bruno-sources
POST   /api/workspaces/:id/bruno-sources
PATCH  /api/workspaces/:id/bruno-sources/{sourceId}
DELETE /api/workspaces/:id/bruno-sources/{sourceId}
POST   /api/workspaces/:id/imports/bruno/git
POST   /api/workspaces/:id/imports/git
GET    /api/workspaces/:id/api-tokens
POST   /api/workspaces/:id/api-tokens
DELETE /api/workspaces/:id/api-tokens/{tokenId}
GET    /api/workspaces/:id/api-specs
POST   /api/workspaces/:id/api-specs
POST   /api/workspaces/:id/api-specs/upload
POST   /api/workspaces/:id/api-specs/push
GET    /api/workspaces/:id/invites
POST   /api/workspaces/:id/invites
DELETE /api/workspaces/:id/invites/:inviteId
```

### Collections & Requests

```
POST   /api/collections
PATCH  /api/collections/reorder
GET    /api/collections/:id
PATCH  /api/collections/:id
DELETE /api/collections/:id
POST   /api/collections/:id/duplicate
GET    /api/collections/:id/docs
GET    /api/collections/:id/catalog
POST   /api/collections/:id/run

GET    /api/requests
POST   /api/requests
PATCH  /api/requests/reorder
GET    /api/requests/:id
GET    /api/requests/:id/docs-bundle
PATCH  /api/requests/:id
DELETE /api/requests/:id
PATCH  /api/requests/:id/move
POST   /api/requests/:id/execute
POST   /api/requests/:id/duplicate
GET    /api/requests/:id/snippet
POST   /api/requests/:id/examples
POST   /api/requests/:id/children
GET    /api/requests/:id/children
POST   /api/requests/:id/reset-field
POST   /api/requests/:id/promote-to-template
POST   /api/requests/:id/push-to-children
```

### Environments, History, Specs, Shares

```
GET    /api/environments
POST   /api/environments
GET    /api/environments/:id
PATCH  /api/environments/:id
DELETE /api/environments/:id
POST   /api/environments/:id/resolve-variables
POST   /api/environments/:id/variables/bulk

GET    /api/history

GET    /api/api-specs/:id
PATCH  /api/api-specs/:id
DELETE /api/api-specs/:id
PUT    /api/api-specs/:id/environment-urls
POST   /api/api-specs/:id/sync
POST   /api/api-specs/:id/reupload

POST   /api/doc-sources/:id/sync
POST   /api/bruno-sources/:id/sync

POST   /api/shares
DELETE /api/shares/:id
GET    /api/shares/:token
POST   /api/shares/:token/import

GET    /api/invites/:token
POST   /api/invites/:token/accept
```

### Real-Time

```
GET /api/ws?workspace_id=<uuid>&access_token=<token>
```

---

## Known Limitations

These are intentional v1 scope boundaries:

- **Concurrent edits** use last-write-wins (no OT/CRDT)
- **OAuth2, AWS SigV4, Digest auth**: partial or stub implementations
- **WebSocket, SSE, gRPC protocol testing**: planned for v2
- **Mock servers**: planned for v2
- **SSO/OIDC**: deferred (auth interface is ready for extension)
- **Bruno post-response Expr runtime**: JSON-path subset only (`res.body.foo`, `res.status`)
- **Invite emails** are optional when SMTP is configured; without SMTP, owners can copy invite links from workspace settings. Use Mailpit in docker-compose for local email testing.

---

## Tech Stack

| Layer | Technology | Version |
|-------|------------|---------|
| Backend | Go | 1.26.3 |
| Router | chi | v5.2.1 |
| Database driver | pgx | v5.7.4 |
| Migrations | golang-migrate | v4.18.3 |
| WebSocket | gorilla/websocket | v1.5.3 |
| Script runtime | goja | latest |
| Database | PostgreSQL | 16 |
| Frontend | Nuxt | ^4.4.8 |
| UI framework | Vue | ^3.5.38 |
| State | Pinia | ^3.0.3 |
| Components | shadcn-nuxt | ^2.7.4 |
| Editor | CodeMirror 6 | ^6.x |

---

## Project Structure

```
postchi/
├── backend/           # Go API server
│   ├── cmd/api/       # Main entrypoint
│   ├── cmd/migrate/   # Standalone migration CLI
│   └── internal/      # Handlers, executor, import/export, etc.
├── frontend/          # Nuxt 4 web application
│   ├── app/           # Pages, components, stores, composables
│   └── public/brand/  # Icons, favicons, OG image, web manifest
├── migrations/        # SQL migrations (embedded in Docker image)
├── docs/              # Guides (documentation linking, network test matrix)
│   └── brand/         # Wordmark and shared brand assets
├── screenshots/       # README and marketing screenshots
├── docker-compose.yml
├── Makefile
└── .env.example
```

---

## License

[MIT](LICENSE)
