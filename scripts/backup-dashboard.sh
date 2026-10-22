#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
umask 077
mkdir -p backups
chmod 700 backups
filename="dashboard-$(date -u +%Y%m%dT%H%M%SZ)-$$.db"
compose=(docker compose -f compose.dashboard.yaml)
result=$("${compose[@]}" exec -T dashboard /app/glance dashboard backup --output "/app/backups/$filename")
expected=${result##* }
temp=$(mktemp -d ./backups/.export-XXXXXX)
trap 'rm -rf "$temp"' EXIT
"${compose[@]}" cp "dashboard:/app/backups/$filename" "$temp/backup.db"
chmod 600 "$temp/backup.db"
if command -v sha256sum >/dev/null; then actual=$(sha256sum "$temp/backup.db"); else actual=$(shasum -a 256 "$temp/backup.db"); fi
actual=${actual%% *}
test "$actual" = "$expected"
ln "$temp/backup.db" "backups/$filename"
"${compose[@]}" exec -T dashboard /app/glance dashboard prune-backups --directory /app/backups --keep 7 --apply
printf 'Verified backup exported: backups/%s\n' "$filename"
