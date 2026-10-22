# 개인용 출시 후보 최종 검증 — G7

2026-10-22 UTC. F001–F008 조사·설계 문서화, F009–F078 구현·검증 완료: **78/80**. F079 실제 Mac mini 및 개인 연동, F080 공인 HTTPS·휴대전화 인수는 외부 조건으로 남긴다. 서비스가 실제 출시됐다는 뜻은 아니다.

실행 코드와 릴리스 패키지 기준은 clean commit `0d3b1a36441e67fc2343cd3fd7e48b63d8b687ff`다. 이후 커밋은 검증 증거와 진행 문서 갱신이다. 브랜치 `release/personal-dashboard`에 구현·증거를 보관한다. 실제 GitHub Actions 결과는 이 환경에서 조회하지 못했으므로 원격 CI 성공을 주장하지 않는다.

| 검사 | 실제 결과 | 증거 |
| --- | --- | --- |
| Go | 전체 race·vet·module 검증 통과, 계약 drift 없음 | CI 명령과 OpenAPI/생성 타입, `internal/dashboard` 테스트 |
| 브라우저 | Chromium 74개 통과, 고정 글꼴·6개 시각 비교 포함 | `web/e2e`, `web/e2e/visual.spec.ts-snapshots` |
| 접근성 | axe AA serious/critical 0; Linux Orca/AT-SPI 실제 브레일 출력과 키보드 흐름 확인 | [접근성 범위](accessibility.md), `verification/accessibility.json` |
| 모바일 성능 | 32카드, CPU4x·RTT150ms·1.6Mbps, cold 5회: LCP 중앙값2000ms, CLS0, 조작 피드백 p95150.7ms | `verification/frontend-performance.json`; 필드 INP 측정은 아님 |
| 지속 부하 | 실제60분, 8클라이언트·128소스·10000entry,143972요청,실패0·중복수집0 | `verification/soak-60m.json` |
| 동시 API 지연 | 검색 p9555.686ms, 보드9.725ms; 재시작·백업 복구 통과 | 같은 soak 결과; 정확한 부하 테스트 backend commit은40a1114 |
| 개인 정보·오프라인 | 공개 앱 파일만 SW cache; 개인 API는 cache하지 않음. 세션 만료 폼 복구·명시적 logout purge·대기 업데이트 확인 | [오프라인 범위](pwa-offline.md), auth/PWA/conflict E2E |
| DB 업그레이드 | 실제 v1 binary→schema1→현재 schema12→구버전+이전DB 복귀 | `verification/upgrade-rollback.json` |
| 백업·키 | 온라인 백업, 분리된 private DB/key export, 복구·잘못된 키 거부, 정상본 최소2개 보존 | `verification/backup-recovery.json`, backup rotation/key tests |
| 운영 경고 | 없음·정상·72시간 경과한 실제 SQLite 백업, 손상된 최신 백업 경고 확인 | `verification/operations.json` |
| HTTPS | 클라이언트가 로컬 CA를 검증한 Caddy 경유 로그인·Secure/HttpOnly/Strict 쿠키·저장·logout 통과 | `verification/https-rehearsal.json`; 공인 DNS/ACME 검증은 아님 |
| 컨테이너 | AMD64 비루트·읽기 전용·재시작·새 볼륨 복구, ARM64 실제 image rootfs를 QEMU로 실행해 FTS5·인증·준비 상태 확인 | `verification/arm64-image.json`, container smoke |
| 기존 YAML | 최신 바이너리의 설정 검증·HTML/calendar/clock·static assets 통과 | 독립 loopback fixture 리허설 |

## 보안 결과의 범위

npm의 개발·운영 전체 의존성 audit은0건이다. 앱 Go 검사는 공식 Go 취약점 저장소에서 생성한 오프라인 DB로0건이다. 프록시는 호출 가능한 취약점0건이며 요구 module에 있지만 사용 경로가 없는 항목1개를 구분해 기록한다. 실시간 `vuln.go.dev` 접근 차단을 성공으로 세지 않는다.

기존 Debian12 앱 이미지와 공식 Caddy 이미지에서 HIGH/CRITICAL이 확인돼 교체했다. 검증한 최종 AMD64/ARM64 앱·프록시 네 이미지 모두 **HIGH/CRITICAL 0건**이다. 앱 Distroless 기반에는 MEDIUM15·LOW8, 프록시에는 UNKNOWN1이 남는다. 취약점 전체0건이라고 표현하지 않는다. Trivy0.76.1과2026-10-22 DB로 검사했으며 SBOM·보고서·바이너리·소스 commit·image ID를 연결했다. [검사 요약](verification/security.json), [최종 이미지·패키지](verification/release-artifacts.json)을 참고한다.

## 패키지와 재현

현재 클라우드 작업 공간의 `release-artifacts/frontier-0d3b1a3/`에는 AMD64/ARM64 native CGO 바이너리, 패치 프록시, 각각의 대응 소스 archive와 전체 LICENSE/NOTICE, 빌드 정보, 네 이미지의 CycloneDX SBOM·취약점 보고서, `container-images.tar.gz`가 있다. 전체448개 파일의 SHA256SUMS를 실제 대조했다. 이미지에는 운영 DB·키·비밀번호를 포함하지 않는다. 레지스트리 업로드·GitHub Release 생성은 하지 않았다. 기록된 SHA-256 image ID는 로컬 image 구성 digest이며 registry manifest digest와 구분한다.

macOS native 빌드는 release workflow에 준비돼 있지만 실제 Mac 바이너리를 여기서 만든 것으로 표시하지 않는다. Linux 바이너리는 호환 C 라이브러리가 필요하며 Mac에서 실행하는 패키지가 아니다. Mac에서는 네이티브 빌드 또는 해당 아키텍처 Linux 컨테이너를 사용한다. 실제 CPU·재부팅·로그인·절전·VoiceOver 음성·개인 GitHub/calendar/JSON 자격 증명과 날씨 확인은 F079에 남긴다. 원격 백업은 분리된 mounted-directory 모의 검증까지이며 실제 제공자 복제 확인은 운영 설정에 포함한다.

재사용 클라우드 설정 초안은 저장됐고 `requires_publish=true`다. 현재 환경에서 설치 스크립트 재실행과8090 시작 helper를 두 번 실행했으며, DB/key를 보존하고 FTS5와 정확한 소스 commit 링크를 확인했다. 새 환경에 적용하려면 환경 설정에서 검토·저장 후 Publish가 필요하다. fresh restore나 Publish 완료를 주장하지 않는다.
