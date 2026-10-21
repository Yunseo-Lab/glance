#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
umask 077
mkdir -p .secrets
chmod 700 .secrets
if [ ! -e .secrets/owner_password ]; then
  (set -o noclobber; openssl rand -base64 32 > .secrets/owner_password)
fi
# The private parent directory protects the file on the host. Read permission
# lets Docker mount this file for the non-root application as a read-only secret.
chmod 444 .secrets/owner_password
printf '%s\n' 'Owner password is stored in .secrets/owner_password (not printed).'
printf '%s\n' 'Keep this file private. Back it up securely and use it on the login screen.'
