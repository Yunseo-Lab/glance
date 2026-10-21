# Editable dashboard

The new `glance dashboard` mode provides a Korean, responsive dashboard with a
browser editor. The existing YAML-based mode and its widgets remain available.

## Build and run

Requirements: Go 1.27.3 or newer, a C compiler (SQLite's bundled C engine), and
Node 22.12 or newer. Node is needed only to build the frontend. On macOS, install
the Xcode Command Line Tools for the C compiler. Both Intel and Apple Silicon
machines can build natively.

```sh
cd web
npm ci
npm run build
cd ..
CGO_ENABLED=1 go build -o build/glance .
./build/glance dashboard --listen 127.0.0.1:8090 --db data/dashboard.db
```

The binary embeds the built React assets. Building Go without first building the
frontend still supports the legacy mode and JSON API, but the new dashboard page
returns a helpful 503 until assets are built and the binary is rebuilt.

For frontend development, run the backend on its default port 8090 and `npm run
dev` in `web/`. Vite proxies `/api` to that backend. Use the production build when
testing production cookie authentication.

## Features

- Clock with a configurable IANA timezone, month calendar, text notes, bookmarks,
  RSS/Atom feeds, and current weather via Open-Meteo.
- Add, configure, resize, remove and reorder widgets in edit mode. Drag or use the
  accessible move buttons. Save commits the whole edit; Cancel restores it.
- Rename the dashboard, duplicate widgets, and see unsaved-change status. Cancel
  and home navigation ask before discarding local changes. New widgets are
  revealed through active filters and receive keyboard focus.
- Export committed configuration as versioned JSON. Import previews use server
  validation and do not persist or fetch anything until you save. Export is a
  settings backup; it does not include cached provider data or server credentials.
- Persistent light/dark themes, category filters and widget-title search.
- Stable IDs and optimistic revisions: an older browser tab cannot overwrite a
  newer saved dashboard. On conflict, back up the local draft or explicitly load
  the latest saved configuration. Reload never silently overwrites local edits.
- SQLite persistence, independent source refresh, shared source caches, bounded
  concurrency, retries with backoff and last-success data during brief failures.
- Per-widget render failure isolation, editor validation and connection recovery
  when the network or a hidden tab returns.

The calendar is a month/date view, not an external agenda integration. There is
one dashboard and one local owner in this release; accounts, sharing, YAML
import/export and SSE are future work. Existing YAML settings are not imported
or modified automatically.

## Configuration transfer

Use **설정 내보내기** for the last committed configuration. Unsaved changes are
excluded and the UI explains that when applicable. During a revision conflict,
**내 편집 백업** exports the local draft instead. The file envelope contains
`format: "glance-dashboard"`, `version: 1` and a `board` document.

**설정 가져오기** accepts a JSON file up to 1 MiB. Unknown formats, versions and
invalid widget/source settings are rejected. A successful import opens a draft
for review; **대시보드 저장** is the only commit step. Imported revision numbers
are rebased to the current local revision, and optimistic concurrency still
protects a newer save from another tab. This is JSON transfer for the editable
mode, not import/export of the legacy YAML renderer's configuration.

## Access and outbound requests

By default the server listens only on loopback. Listening on a non-loopback
address requires an owner password of at least 16 characters, supplied through
`GLANCE_DASHBOARD_TOKEN` or `GLANCE_DASHBOARD_TOKEN_FILE` (choose one). Never put
credentials in source. The browser login creates an HttpOnly, SameSite=Strict
session valid for 12 hours; logout revokes it and a server restart requires login
again. Static login assets are public; dashboard data and health APIs are private.
Automated checks can use Basic authentication. Login attempts are bounded to five
per minute per peer address (a reverse proxy shares this allowance). Remote access
requires HTTPS and `GLANCE_DASHBOARD_PUBLIC_URL=https://your-domain` for Secure
cookies. See [the deployment runbook](operations.md).

Public, unauthenticated HTTPS RSS feeds on port 443 are supported. Private
network destinations, URL credentials and unsafe links are rejected. Direct
connections validate DNS addresses again at connection time. Redirects are also
validated, and response sizes, timeouts and fetch concurrency are limited.

When an HTTPS egress proxy is configured, DNS resolution belongs to the proxy.
The built-in proxy destinations are `github.com`, `raw.githubusercontent.com`
and `api.open-meteo.com`. To approve additional public RSS destinations, set
`GLANCE_DASHBOARD_FEED_HOSTS` to a comma-separated list of exact hostnames before
starting the server. Administrators must approve only trusted public hosts and
use a proxy that enforces destination policy. Every redirect target needs its
own approval. This application setting does not grant access through a managed
cloud environment's separate network policy.

Weather requires access to `api.open-meteo.com`. The cloud development environment
may deny that destination until it is allowed in environment settings. The card
then displays an isolated connection error and retries; local widgets still work.
The initial dashboard uses a public GitHub Atom feed and needs no API credentials.

## Persistence and backups

The `--db` path is the persistent state. Keep its directory on a persistent volume.
SQLite uses WAL mode. Use `glance dashboard backup --db PATH --output NEW_FILE`
for a consistent online backup, `verify-backup --input FILE` to check it, and
`restore --input FILE --db NEW_PATH` for a validated restore. Restore deliberately
refuses to overwrite files. JSON export does not replace a full database backup.
See [backup, restore and rollback procedures](operations.md). Newer, unknown
schema versions fail closed rather than being modified.

## Architecture

The HTTP contract is documented in [OpenAPI](dashboard-api.openapi.yaml).

```text
web/src/                     React editor, presentation and API types
internal/dashboard/model.go  Board, widget and source contracts + validation
internal/dashboard/store.go  SQLite persistence, migration and revisions
internal/dashboard/runtime.go  Freshness, deduplication and fetch lifecycle
internal/dashboard/provider.go RSS/weather adapters + outbound HTTP policy
internal/dashboard/server.go HTTP API, authentication and embedded assets
web/src/useSnapshots.ts      Source polling, visibility and connection recovery
web/src/transfer.ts          Versioned JSON export and import envelope checks
web/src/WidgetBoundary.tsx   Widget rendering failure isolation
internal/glance/             Existing YAML-based renderer, unchanged behavior
```

The board is a versioned aggregate stored as one JSON document in SQLite.
Widgets and sources have separate stable identities inside that document, so
layout and source edits commit atomically. Cache data lives in a separate table.
This avoids introducing many tables before cross-dashboard relationships exist.

Providers return typed data and never render HTML. A widget owns its display
configuration and may reference a source. Source keys include the local owner,
provider type and canonical query configuration. New authenticated integrations
must add their access identity to that scope before any cache sharing is allowed.

Reading snapshots immediately returns available data and schedules expired
sources asynchronously. At most four fetches run concurrently; identical source
queries share one task. RSS is fresh for 10 minutes, weather for 15 minutes, and
last-success data is shown for at most 24 hours. Failed sources back off from
30 seconds to 8 minutes. The browser polls pending sources every 2 seconds and
otherwise every 30 seconds, pausing when its tab is hidden. It does not directly
call provider APIs. Connection failures retry after 5 seconds; returning online
or making the tab visible triggers an immediate check. A cold cache displays a loading state without blocking other
cards. This initial release collects on demand rather than continuously.

## Verification

```sh
go test ./...
go test -race ./internal/dashboard
go vet ./...
cd web
npm ci
npm run build
# From the repository root, rebuild build/glance after the frontend build.
# Install a Playwright Chromium browser once if a system browser is unavailable:
npx playwright install chromium
npm run test:e2e
```

The browser suite starts a separate backend on port 8091 with a temporary SQLite
database and removes it afterward. It never edits a development dashboard. Build
`build/glance` before running it. To use system Chromium instead of downloading
one, set `PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH=/usr/bin/chromium` (or the appropriate
path). Go tests use injected providers for deterministic tests of slow upstreams,
duplicate fetches, failure recovery, restart persistence, validation, CSRF and
authentication. Browser tests exercise editing, reordering, save/reload, cancel,
delete, themes, filtering, mobile layout and stale-tab conflicts.
The ten additional improvement scenarios and their individual verification
records are listed in [development cycles](development-cycles.md).
