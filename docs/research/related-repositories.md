# 관련 저장소 조사와 적용 판단

조사일: **2026-10-21**. 후보 **41개**의 원본 README를 읽었고, 프로젝트 관리
제품 Plane 1개를 제외한 **40개**를 참고 목록에 남겼다. 그중 하나는 발견용
색인 awesome-selfhosted이고, 나머지는 제품·라이브러리·생태계 저장소다.
웹 전체를 빠짐없이 찾았다는 의미가 아니다. 별 개수나 순위로 품질을 단정하지 않는다.

GitHub [startpage topic](https://github.com/topics/startpage)과
[Personal Dashboards 목록](https://github.com/awesome-selfhosted/awesome-selfhosted#personal-dashboards)에서
후보를 넓힌 뒤, 읽기·검색·수집·운영·UI 도구까지 확장했다. 일반 검색 도구가
없는 환경이라 GitHub 원본 문서를 직접 검색·열람했다. 제품을 모두 설치해
실행하거나 UX를 실제 화면에서 감사한 조사는 아니다. README의 주장은 제품
문서상 특성이며, 우리의 채택 판단 및 구현 완료와 구분한다.

## 비교와 적용

| 저장소 | 분류 | 원본에서 확인한 참고점 | 이 프로젝트에 적용할 판단 |
| --- | --- | --- | --- |
| [glanceapp/glance](https://github.com/glanceapp/glance) | 기반 | 정보 밀도, Go 배포, 위젯 구성 | 새 편집 모드와 기존 YAML 모드를 구분해 호환 유지 |
| [gethomepage/homepage](https://github.com/gethomepage/homepage) | 대시보드 | 서버 프록시, 서비스 연동, 위젯 registry | 비밀정보를 브라우저에 보내지 않는 adapter/catalog 경계 |
| [homarr-labs/homarr](https://github.com/homarr-labs/homarr) | 대시보드 | 웹 설정, 그리드, 통합 검색, 실시간 갱신 | 편집·연동 관리 UX 참고; Redis/tRPC 전체 스택 도입은 보류 |
| [Lissy93/dashy](https://github.com/Lissy93/dashy) | 대시보드 | 단축키, 다양한 보기, 설정 문서, PWA | 명령 팔레트·설정 검증·오프라인 범위를 명확히 설계 |
| [bastienwirtz/homer](https://github.com/bastienwirtz/homer) | 대시보드 | 간결한 시작 화면, 검색 단축키 | 첫 사용과 키보드 경로를 짧게 유지 |
| [pawelmalak/flame](https://github.com/pawelmalak/flame) | 대시보드 | 브라우저 북마크 관리, 검색, 날씨 | 소스 추가를 짧은 흐름으로 구성; Docker socket 기본 연결은 제외 |
| [linuxserver/Heimdall](https://github.com/linuxserver/Heimdall) | 대시보드 | 앱 타일, 검색 공급자, enhanced applications | 외부 앱으로 이동과 내부 정보 표시를 분리 |
| [makeplane/plane](https://github.com/makeplane/plane) | 제외 | 프로젝트 관리 제품 | 대시보드라는 이름만으로 범위를 넓히지 않음; 핵심 비교에서 제외 |
| [MauriceNino/dashdot](https://github.com/MauriceNino/dashdot) | 모니터링 | 호스트 상태와 시각화 | 자체 상태 표시 참고; 호스트 전체를 읽는 권한은 기본 제공하지 않음 |
| [miniflux/v2](https://github.com/miniflux/v2) | 읽기 | 읽음 상태, 검색, 조건부 HTTP, 서버 측 피드 수집 | ETag/Last-Modified, 출처·읽기 모델, 안전한 HTTP 경계 |
| [FreshRSS/FreshRSS](https://github.com/FreshRSS/FreshRSS) | 읽기 | 구독 관리, API, 확장, 국제화 | OPML·읽음/보관·키보드 탐색; 전체 리더 재구현은 범위 제한 |
| [RSS-Bridge/rss-bridge](https://github.com/RSS-Bridge/rss-bridge) | 수집 | 사이트를 RSS로 연결하는 bridge | 기존 피드와 호환; 스크레이퍼를 앱 안에 무제한 내장하지 않음 |
| [RSSNext/Folo](https://github.com/RSSNext/Folo) | 읽기 | 타임라인, 형식별 읽기, 요약·AI 기능 | 통합 인박스·출처 있는 브리핑 참고; AI는 선택 기능 |
| [RSSHub/RSSHub](https://github.com/RSSHub/RSSHub) | 수집 | 다양한 사이트의 RSS route | 공개 RSS 입력 호환, 외부 bridge 운영은 별도 |
| [linkwarden/linkwarden](https://github.com/linkwarden/linkwarden) | 지식 | 저장, 태그, 읽기와 검색 | 저장 항목과 읽음 상태 분리; 전체 웹 아카이브는 후순위 |
| [hoarder-app/hoarder](https://github.com/hoarder-app/hoarder) | 지식 | Karakeep 문서: 검색, 태그, 오프라인 읽기 | 태그와 수집 후 재발견 참고; Meilisearch/LLM 기본 의존성 제외 |
| [usememos/memos](https://github.com/usememos/memos) | 지식 | 개인 타임라인, 메모, 소유 데이터 | 빠른 메모 입력과 태그·개인 검색 |
| [louislam/uptime-kuma](https://github.com/louislam/uptime-kuma) | 모니터링 | 체크 주기, 상태 기록, 운영 화면 | 에러·최종 성공 시각·상태 기록을 사용자가 볼 수 있게 함 |
| [grafana/grafana](https://github.com/grafana/grafana) | 모니터링 | 데이터 소스와 패널, 시각화 | 소스와 표현 분리; 범용 BI/시계열 DB 구축은 제외 |
| [home-assistant/frontend](https://github.com/home-assistant/frontend) | UI 구조 | 카드 생성·갱신·오류 카드 경계 | 플러그인 경계와 오류 격리 참고; 임의 실행 코드 로딩은 제외 |
| [gridstack/gridstack.js](https://github.com/gridstack/gridstack.js) | UI 도구 | 그리드 배치, 리사이즈, 터치 지원 | 자유 그리드 실험 후보; 저장 모델과 키보드 동작을 먼저 검증 |
| [clauderic/dnd-kit](https://github.com/clauderic/dnd-kit) | UI 도구 | 센서, 키보드, ARIA, 드래그 확장 | 현재 정렬형 배치의 우선 실험 후보; 버전 호환성 spike 필요 |
| [TanStack/query](https://github.com/TanStack/query) | UI 도구 | 서버 상태의 신선도·재시도·무효화 | 서버 캐시 정책과 구분하고 로그아웃 시 private cache 제거 |
| [shadcn-ui/ui](https://github.com/shadcn-ui/ui) | UI 도구 | 재사용 UI 소스와 구성 방식 | 접근성 있는 dialog/command/field 구성 참고; 무조건 전체 교체하지 않음 |
| [awesome-selfhosted/awesome-selfhosted](https://github.com/awesome-selfhosted/awesome-selfhosted) | 발견 색인 | Personal Dashboards 및 주변 카테고리 목록 | 후보 발굴용; 기능 사실은 각 원본 문서로 재확인 |
| [hslr-s/sun-panel](https://github.com/hslr-s/sun-panel) | 대시보드 | 시각적 시작 화면, 제품의 공개/상용 경계 | README가 closed-source 전환을 알림; 공개 소스의 최신성 가정 금지 |
| [hywax/mafl](https://github.com/hywax/mafl) | 대시보드 | 최소 구성, 서비스와 테마 | 짧은 설정과 시각적 단순함 |
| [victrme/Bonjourr](https://github.com/victrme/Bonjourr) | 시작 화면 | 개인 브라우저 시작 화면 | 개인화 UX 참고; 브라우저 확장 배포는 현재 범위 밖 |
| [zombieFox/nightTab](https://github.com/zombieFox/nightTab) | 시작 화면 | 사용자 정의 시작 화면 | 북마크 정리·설정 이동성 참고 |
| [yusing/godoxy](https://github.com/yusing/godoxy) | 운영 인접 | 프록시·자동 발견·상태 표시 | 발견 기능과 운영 권한 비용 비교; 앱이 프록시 제어자가 되지 않음 |
| [searxng/searxng](https://github.com/searxng/searxng) | 검색 인접 | 프라이버시 중심 메타 검색 | 외부 검색 이동 후보; 자체 검색 엔진 운영은 기본 요구하지 않음 |
| [toboshii/hajimari](https://github.com/toboshii/hajimari) | 대시보드 | Kubernetes 기반 앱 발견 | 참고만 하고 개인용 배포에 Kubernetes를 요구하지 않음 |
| [glanceapp/community-widgets](https://github.com/glanceapp/community-widgets) | 확장 생태계 | custom-api/extension 위젯 예시 목록 | 실사용 연동 수요와 adapter 계약 참고; 전체 코드 자동 수입 금지 |
| [adinhodovic/compass](https://github.com/adinhodovic/compass) | 대시보드 | 발견 소스·상태·태그·운영 문맥 | 연결 상태·마지막 갱신·문제 진단 흐름 참고 |
| [Turbootzz/Nimbus](https://github.com/Turbootzz/Nimbus) | 대시보드 | 시각 편집, 검색, 상태·지표, Go 기반 서비스 | Go 개인 대시보드의 직접 비교 후보; 다중 사용자 기능 제외 |
| [arran4/gobookmarks](https://github.com/arran4/gobookmarks) | 대시보드 | 시각 편집·텍스트 편집·단축키 | 설정 내보내기와 키보드 기반 북마크 관리 |
| [knrdl/hubleys-dashboard](https://github.com/knrdl/hubleys-dashboard) | 대시보드 | 검색, 중앙 설정, forward auth | 외부 인증 경계 참고; 신뢰 헤더를 기본 인증으로 받지 않음 |
| [kodehat/portkey](https://github.com/kodehat/portkey) | 대시보드 | 검색 키워드·fuzzy matching, 단일 구성 | 명령 팔레트와 링크 탐색의 작은 상호작용 |
| [notclickable-jordan/starbase-80](https://github.com/notclickable-jordan/starbase-80) | 대시보드 | 앱 그리드와 단순 JSON 구성 | 초기 로딩과 단순 배치 비교 기준 |
| [tomershvueli/homepage](https://github.com/tomershvueli/homepage) | 대시보드 | 독립 실행 자산·간단한 구성 | 외부 CDN 없는 동작과 기본 화면의 비용 최소화 |
| [LinkStackOrg/LinkStack](https://github.com/LinkStackOrg/LinkStack) | 링크 인접 | 소유 데이터와 링크 편집 | 읽기 좋은 링크 묶음 참고; public link-in-bio 기능은 제외 |

## 소스까지 추가로 확인한 부분

- [Homepage registry](https://github.com/gethomepage/homepage/blob/HEAD/src/widgets/widgets.js):
  위젯 모듈을 모아 등록하는 구조를 읽었다. 지금의 `switch` 중심 등록을
  catalog/adapter로 분리하는 근거다. Homepage 서버 전체를 코드 감사한 것은 아니다.
- [Homarr manifest](https://github.com/homarr-labs/homarr/blob/HEAD/package.json):
  drag-and-drop, query, DB 및 Next 관련 의존성을 확인했다. Homarr의 요구를
  만족시키는 전체 스택과 이 개인용 앱의 최소 스택을 동일하게 취급하지 않는다.
- [Home Assistant card](https://github.com/home-assistant/frontend/blob/HEAD/src/panels/lovelace/cards/hui-card.ts):
  카드 생성·설정 갱신과 오류 카드로 전환하는 경계를 확인했다. 현재 ErrorBoundary를
  유지하면서 재설정·재시도를 넣을 때 참고한다.
- [Miniflux HTTP client](https://github.com/miniflux/v2/blob/HEAD/internal/http/client/client.go):
  연결 시점에 IP를 검사하고 실제 안전한 IP로 접속하는 방식을 읽었다. 우리
  egress proxy·CA 유지 정책과 DNS 재검사를 유지할 근거다. 이 파일에서
  조건부 GET 구현을 확인한 것은 아니며 ETag 관련 기능은 Miniflux README 근거다.
- [TanStack 기본 정책](https://github.com/TanStack/query/blob/HEAD/docs/framework/react/guides/important-defaults.md):
  staleTime, refetch, retry, gcTime의 기본 동작을 확인했다. 기본값을 그대로
  적용하면 현재 서버 backoff와 겹칠 수 있으므로 별도 정책을 정의한다.
- dnd-kit·GridStack의 README와 package manifest, Dashy의 설정 문서를 확인했다.
  둘 중 어느 배치 엔진도 아직 도입하지 않았다. React 현재 버전·터치·키보드·
  저장 후 동일 배치에 대한 짧은 실험을 먼저 수행한다.

## 공통적으로 가져올 것

1. **화면을 보기 전에 설정을 공부하지 않아도 되는 흐름**: 템플릿 → 연결 → 미리보기 → 저장.
2. **소스와 위젯 분리**: 동일 소스의 네트워크 요청·비밀정보·실패 상태를 한곳에서 관리.
3. **언제 수집한 정보인지 설명**: 최신·오래됨·실패·재시도·원문 출처를 일관되게 표시.
4. **실수에서 복구**: undo/redo, draft recovery, revision 충돌, 복원 가능한 백업.
5. **찾고 읽고 다시 발견하기**: 명령 팔레트, 읽음/저장/태그, 제한된 로컬 검색.
6. **맥에서 감당할 수 있는 운영 비용**: 단일 Go 프로세스와 SQLite, 독립된 Caddy.

## 지금 가져오지 않을 것

다중 사용자 SaaS, 결제, Kubernetes, Redis/외부 검색 DB 필수화, 임의 실행
플러그인, 호스트 Docker socket 접근, 인증 없는 사설망 탐색, 전체 웹 크롤러,
API 키 없는 AI 기능을 동작한 것처럼 보이는 데모는 기본 제품 범위에 넣지 않는다.
구조를 참고하며 코드를 복사하지 않았다. 향후 코드·아이콘을 실제 도입할 때는
해당 버전의 라이선스와 귀속을 확인한다. Sun-Panel의 README는 폐쇄 소스 전환을
명시하므로 공개 저장소가 최신 유료 제품 구현과 같다는 가정을 하지 않는다.

각 문서의 URL·조회 날짜·콘텐츠 SHA-256·일부 저장소의 조회 시점 HEAD와
실패한 파일 경로까지 [조사 원장](repository-sources.json)에 남겼다.
정확한 구현 순서와 수용 기준은 [80단계 계획](../frontier-implementation-plan.md),
구조 결정은 [아키텍처 결정문](../architecture/frontier-dashboard.md)에 있다.
