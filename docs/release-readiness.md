> 최신 기준: [80단계 진행 기록](frontier-progress.md), [G7 최종 검증](verification-g7.md). 조사·설계8단계와 구현·검증70단계를 처리했습니다. 실제 Mac·공인 HTTPS·휴대전화 인수2단계는 외부 조건으로 미완료입니다. 아래 표는 초기 개인용 후보의 역사적 기록입니다.

# Personal release readiness

Scope: a single owner using a private dashboard, deployed to the user's Mac mini.
No signup, shared accounts or public multi-tenant hosting is included. The user
has no running deployment server or domain yet; live cutover requires those.

Items are complete only after implementation or an applicable verification.
External requirements remain explicitly blocked, never counted as passed.

| # | Release task | Status |
| --- | --- | --- |
| 01 | Confirm owner-only release scope | Complete |
| 02 | Inventory repository changes and preserve existing dashboard | Complete |
| 03 | Check Git remote access and current branch | Complete |
| 04 | Check local Docker daemon and cloud network constraints | Complete |
| 05 | Browser login screen | Complete |
| 06 | Signed, expiring HttpOnly sessions | Complete |
| 07 | Logout and session expiry handling | Complete |
| 08 | Constant-time credential verification | Complete |
| 09 | Bounded login attempt throttling | Complete |
| 10 | Same-origin checks on authentication mutations | Complete |
| 11 | Secure cookie and HTTPS deployment settings | Complete |
| 12 | Authentication integration/browser tests | Complete |
| 13 | Request panic recovery and error responses | Complete |
| 14 | Request limits and useful health/readiness checks | Complete |
| 15 | Bounded in-flight provider work and graceful shutdown | Complete |
| 16 | Provider error log redaction | Complete |
| 17 | Deterministic weather success/failure tests | Complete |
| 18 | Live external-service connectivity diagnosis | Complete |
| 19 | Online, consistent SQLite backup command | Complete |
| 20 | Backup integrity verification | Complete |
| 21 | Safe restore into a new database path | Complete |
| 22 | Backup/restore round-trip tests | Complete |
| 23 | Reproducible frontend and binary build command | Complete |
| 24 | CGO-enabled dashboard container image | Complete |
| 25 | Non-root container and persistent data permissions | Complete |
| 26 | Container health check without credential output | Complete |
| 27 | Local Docker Compose deployment | Complete |
| 28 | HTTPS Caddy deployment configuration | Complete |
| 29 | Secure file-based deployment credential loading | Complete |
| 30 | Persistent volumes and restart policy | Complete |
| 31 | Resource limits and log rotation | Complete |
| 32 | Build and run the real container | Complete |
| 33 | Container authentication and persistence smoke test | Complete |
| 34 | Stop/restart and clean restore rehearsal | Complete |
| 35 | Concurrent request/load and error isolation checks | Complete |
| 36 | Automated backend/frontend/browser CI | Complete |
| 37 | Release build/checksum automation | Complete |
| 38 | Update/rollback and incident runbook | Complete |
| 39 | Mac mini setup and backup scheduling guide | Complete |
| 40 | Dependency/security checks and final full regression | Complete |
| 41 | Document measured release candidate limitations | Complete |
| 42 | Commit implementation and release preparation | Complete |
| 43 | Push reviewed changes to the repository | Complete |
| 44 | Mac mini availability and domain selection | External: unavailable |
| 45 | Live HTTPS deployment and real-device acceptance | External: blocked by 44 |

Git publication is source publication, not deployment of the running service.

## Verified release candidate — 2026-10-21

- Go 1.27.3, SQLite driver 1.14.52, x/net 0.60.0 and x/text 0.42.0.
- Frontend type check/production build, native CGO binary, module checksums,
  all 16 backend tests with race detection, and `go vet ./...` passed.
- The backend race suite also passed 30 consecutive runs after fixing a race
  that could schedule another fetch before observing a newly written backoff.
- All 16 Playwright browser scenarios passed, including owner login/logout,
  revoked-cookie replay and preserving an unsaved draft during reauthentication.
- npm audit (including development dependencies) reported zero vulnerabilities.
- govulncheck 1.9.0 reported no vulnerabilities. The live vulnerability endpoint
  returned proxy 403, so the scan used all 4,710 original OSV entries from the
  official golang/vulndb repository at commit
  `e0bad56276e98711a598adbab976de7cd914013a`. A conservative offline module index
  omitted the optional fixed-version optimization; original advisories were not
  altered. The configured CI uses the official live database. This is a point in
  time dependency scan, not an assurance about future vulnerabilities or OS images.
- The final dashboard image built and ran as UID 10001 with a read-only root
  filesystem. Image ID:
  `sha256:726fd273e05c47310e1ed5a3e22289c4712954bafbf0a013141647cb1e79bbc2`.
  Size: 92,010,586 bytes. Its smoke check verified authentication, embedded
  assets, readiness, restart persistence and exact saved-board recovery into a
  clean volume. No cloud proxy certificate was baked into the runtime image.
- Both services became healthy using the actual Compose + Caddy configuration.
  HTTPS acceptance verified hostname and certificate chain using Caddy's local
  CA, HSTS, Secure sessions, login, readiness and 200/200 concurrent reads with
  eight workers. Measured p95: 14.8 ms; after backup/restore: 19.0 ms. These are
  cloud-local measurements, not a production capacity or availability promise.
- The host backup export/checksum and the documented read-only input-file restore
  into a new DB path passed against the final Compose image. Existing files were
  never overwritten. Daily launchd scheduling was syntax checked, not run on macOS.
- Linux native archive/checksum creation and verification passed. Release and CI
  workflows passed actionlint; shell/Python/launchd files passed syntax checks.
- The reusable cloud install script was rerun; the development server restarted
  with its existing database. Its startup helper passed twice. Updated
  `install_script` and `start_skill` were saved as an environment draft; publication
  and validation in a newly restored task remain separate user actions.

## Remaining external acceptance

Tasks 44–45 require the user's Mac mini and networking/domain choices. The server
is currently unavailable. No public deployment, production certificate or physical
Mac/phone acceptance is claimed. Open-Meteo returned a proxy 403 in this cloud;
weather parsing/error isolation passed deterministic tests, while live weather on
the operating Mac still needs verification. Initial GitHub API access also returned
403, so remote Actions results and PR creation cannot be verified through that API.
Source push is checked separately through the provided Git HTTPS connection.

Follow [the operations runbook](operations.md) for Mac startup, HTTPS, daily backup,
restore, updates and rollback. Keep the owner password and backups private and
store another backup copy away from the operating Mac.

## Git publication

Implementation commit `0a376f9` was pushed successfully to
[`release/personal-dashboard`](https://github.com/Yunseo-Lab/glance/tree/release/personal-dashboard).
This final record updates tasks 42–43 after the successful push. The main branch
was preserved. No PR, live deployment, or remote CI success is claimed.

**Completed: 43 / 45.** The remaining two tasks require the operating Mac mini and
real domain/network acceptance; they cannot be completed on this cloud machine.
