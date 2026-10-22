#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
go test -tags sqlite_fts5 -race ./internal/dashboard -run 'Test(ProviderContract|Feed|Weather|Location|GitHub|Private|ICalendar|Calendar|JSON|OPML)'
