#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
exec python3 scripts/container-smoke.py "${1:-glance-dashboard:local}"
