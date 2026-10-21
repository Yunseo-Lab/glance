# 개인용 출시·운영 안내

이 배포는 소유자 한 명을 위한 서비스입니다. 현재 코드와 컨테이너는 검증한
출시 후보이며 실제 맥 미니와 공인 도메인에서는 아직 실행하지 않았습니다.

## 맥 미니에서 시작

1. Docker Desktop을 설치하고 실행합니다. Apple Silicon과 Intel 모두 기본
   아키텍처로 빌드합니다. 외장 디스크를 데이터 위치로 쓸 경우 먼저 연결합니다.
2. 저장소를 내려받고 검토한 커밋으로 이동합니다. 아래 명령은 저장소 루트에서
   실행합니다. Docker, Compose, OpenSSL, Python 3가 필요합니다.
3. `bash scripts/init-deployment.sh`를 실행합니다. `.secrets/owner_password`에
   무작위 비밀번호를 한 번만 만들며 기존 비밀번호는 덮어쓰지 않습니다.
   비밀번호 관리자에 보관하세요. 이 파일과 백업은 Git에 올리지 마세요.
4. `cp deploy/environment.example .env`를 실행하고 필요하면 로컬 포트를 바꿉니다.
5. `docker compose -f compose.dashboard.yaml up -d --build --wait`로 시작합니다.
6. 맥의 브라우저에서 `http://127.0.0.1:8090`을 열고 생성한 비밀번호로 로그인합니다.
   비밀번호 확인은 맥에서 직접 파일을 열어 하세요. 채팅이나 터미널 공유에
   비밀번호를 붙여 넣지 마세요.
7. `python3 scripts/verify-deployment.py --url http://127.0.0.1:8090`으로
   로그인, 준비 상태와 200회 읽기를 확인합니다. 테스트는 설정을 바꾸지 않습니다.

데이터는 Compose 프로젝트의 `dashboard_data` 볼륨에 저장됩니다. 컨테이너
재생성과 이미지 교체 후에도 유지됩니다. **운영 볼륨에 `down -v`를 실행하면
데이터가 삭제됩니다.** 프로젝트 이름을 바꾸면 다른 볼륨을 보게 됩니다.
Compose 명령에는 항상 같은 디렉터리와 프로젝트 이름을 사용하세요.

맥 로그인 시 Docker Desktop이 실행되게 설정하고, 서버로 사용할 때 자동
잠자기를 끕니다. 정전 후 재시동 및 FileVault 로그인 조건도 직접 확인해야
합니다. 맥을 끄거나 네트워크가 끊기면 서비스도 중단됩니다.

## 외부 HTTPS 연결

실제 도메인, DNS, 맥의 지속적인 가동 및 인터넷에서 접근 가능한 80/443 포트가
필요합니다. 라우터 포트 전달과 방화벽을 확인하세요. CGNAT 환경에서는 직접
포트 공개가 불가능할 수 있으므로 VPN/터널을 별도로 선택해야 합니다.

`.env`에 `GLANCE_DOMAIN=실제호스트명`과
`GLANCE_PUBLIC_URL=https://실제호스트명`을 설정한 뒤 실행합니다.

```sh
docker compose -f compose.dashboard.yaml -f compose.https.yaml up -d --build --wait
python3 scripts/verify-deployment.py --url https://실제호스트명
```

Caddy가 인증서를 발급·갱신하고 HTTPS로 전달합니다. Caddy 데이터 볼륨도
보존하세요. 비밀번호가 Secure 쿠키를 사용하므로 HTTPS 모드에서는 HTTPS
주소로 로그인합니다. 로컬 HTTP 포트는 맥의 loopback에만 열립니다. HTTP를
외부에 노출하거나 TLS 검증을 끄지 마세요. 인증서는 Caddy, 데이터는 앱이
담당하며, Caddy의 관리 포트는 외부로 공개하지 않습니다.

휴대전화의 다른 네트워크에서 로그인 → 위젯 편집 → 저장 → 새로고침 → 로그아웃을
확인하고, 맥 재시동 후 재로그인 및 데이터 유지도 확인해야 실제 출시가 완료됩니다.
클라우드 내부 테스트의 localhost 인증서는 공인 도메인 검증을 대신하지 않습니다.

## 백업과 복구

실행 중에도 일관된 SQLite 백업을 만듭니다.

```sh
bash scripts/backup-dashboard.sh
```

호스트의 `backups/`에 백업을 꺼내 SHA-256을 검증합니다. 이 폴더를 별도
디스크나 암호화된 백업 저장소에도 복사하세요. 같은 맥에만 있으면 디스크
고장을 대비할 수 없습니다. 보관 기간과 용량은 사용자가 관리합니다. 최소
매일 한 번 및 업데이트 직전에 실행하고 결과를 확인하세요.

자동화는 macOS의 `launchd`로 예약할 수 있습니다. `deploy/backup.launchd.plist`
예제의 **저장소 절대경로, Docker CLI 경로와 사용자 로그 경로**를 수정하여
`~/Library/LaunchAgents/lab.glance.backup.plist`에 저장합니다.
`launchctl bootstrap gui/$(id -u) ~/Library/LaunchAgents/lab.glance.backup.plist`로
등록하고 `launchctl kickstart gui/$(id -u)/lab.glance.backup`으로 시험합니다.
Docker Desktop과 사용자 로그인이 유지되어야 하며 잠자기 중 정시 실행은
보장되지 않습니다. 기록과 실제 백업 생성 시각을 주기적으로 확인하세요.

복구는 원본을 덮어쓰지 않고 **새 DB 경로**에 수행합니다. 서비스가 접근하는
경로를 바꾸기 전에 먼저 앱을 중단합니다. 아래 `BACKUP.db`와 `recovered.db`를
실제 파일명으로 바꾸세요. 복구 전에 현재 상태도 백업해 둡니다.

```sh
cp backups/BACKUP.db .secrets/restore-input.db
chmod 444 .secrets/restore-input.db
docker compose -f compose.dashboard.yaml stop dashboard
docker compose -f compose.dashboard.yaml run --rm --no-deps \
  -v "$PWD/.secrets/restore-input.db:/restore-input.db:ro" dashboard \
  dashboard verify-backup --input /restore-input.db
docker compose -f compose.dashboard.yaml run --rm --no-deps \
  -v "$PWD/.secrets/restore-input.db:/restore-input.db:ro" dashboard \
  dashboard restore --input /restore-input.db --db /app/data/recovered.db
rm .secrets/restore-input.db
```

`.env`에 `GLANCE_DB_PATH=/app/data/recovered.db`를 설정한 뒤 사용 중인 Compose
명령으로 `up -d --wait`를 실행합니다. 로그인하여 위젯과 최근 저장 내용을
확인하세요. 임시 입력 파일은 비루트 컨테이너가 읽을 수 있게 준비하며,
상위 `.secrets/` 디렉터리는 개인 계정에만 허용됩니다. 복구 명령은 이미 존재하는 출력 파일을 거부합니다.
온라인 DB의 본체 파일만 복사하면 WAL의 최신 쓰기를 잃을 수 있습니다.

## 업데이트와 롤백

1. 현재 커밋, `GLANCE_IMAGE`와 `GLANCE_DB_PATH`를 기록하고 백업합니다.
2. 새 커밋을 검토하고 `GLANCE_IMAGE=glance-dashboard:새버전`으로 빌드합니다.
   `docker compose -f compose.dashboard.yaml build`는 `.env`의 이미지 이름을
   사용합니다. 이전 이미지 태그와 백업을 삭제하지 마세요.
3. 같은 Compose 프로젝트로 `up -d --wait` 후 검증 스크립트와 브라우저에서
   실제 저장을 확인합니다. 새 이미지 배포 시 세션은 만료되어 재로그인합니다.
4. 실패하면 앱을 중단하고 `.env`를 이전 이미지 태그로 되돌립니다. DB 스키마가
   변경됐다면 이전 바이너리를 새 DB에 실행하지 말고, 업데이트 직전 백업을
   별도 경로로 복구하여 그 경로를 선택합니다. `up -d --no-build --wait`로
   이전 이미지를 시작하고 재검증합니다.

`docker compose ... ps`와 `logs --tail 100 dashboard caddy`로 상태를 봅니다.
준비 상태는 DB와 내장 UI를 확인하며 외부 RSS/날씨 성공까지 보장하지 않습니다.
외부 API 오류는 카드별로 격리됩니다. 초기 출시에는 가동률 보장이나 유료
모니터링이 없으며 맥의 디스크·백업·인증서 상태는 소유자가 점검합니다.

## 빌드와 검증 범위

`bash scripts/build-dashboard.sh`는 고정 lockfile로 UI와 네이티브 CGO 바이너리,
SHA-256 파일을 만듭니다. `Dockerfile.dashboard`는 의존 이미지 digest를 고정하고
비루트 UID 10001로 실행합니다. 기존 `Dockerfile`은 상위 프로젝트의 YAML 모드용
구성이므로 이 대시보드 배포에는 사용하지 마세요. Linux CGO 바이너리는 해당
런타임 C 라이브러리에 의존합니다. macOS는 macOS에서 네이티브로 빌드합니다.

GitHub Dashboard CI는 Go race/vet, 브라우저, 의존성 및 실제 컨테이너 검사를
수행합니다. 수동 release-candidate workflow는 Linux/macOS 바이너리와 체크섬을
아티팩트로 만듭니다. 원격 CI와 실제 맥 검증 결과를 확인한 후 운영에 적용하세요.
측정 결과와 외부 미완료 항목은 [출시 체크리스트](release-readiness.md)에 있습니다.
