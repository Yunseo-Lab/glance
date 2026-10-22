#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
command -v node >/dev/null
command -v cc >/dev/null
go version
npm --prefix web ci --no-audit --no-fund
revision=$(git rev-parse HEAD)
VITE_SOURCE_REF="$revision" npm --prefix web run build
CGO_ENABLED=1 go build -tags sqlite_fts5 -mod=readonly -trimpath -ldflags="-X github.com/glanceapp/glance/internal/glance.buildVersion=$revision" -o build/glance .
if command -v sha256sum >/dev/null; then sha256sum build/glance > build/glance.sha256; else shasum -a 256 build/glance > build/glance.sha256; fi
