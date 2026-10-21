#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
command -v node >/dev/null
command -v cc >/dev/null
go version
npm --prefix web ci --no-audit --no-fund
npm --prefix web run build
CGO_ENABLED=1 go build -mod=readonly -trimpath -o build/glance .
if command -v sha256sum >/dev/null; then sha256sum build/glance > build/glance.sha256; else shasum -a 256 build/glance > build/glance.sha256; fi
