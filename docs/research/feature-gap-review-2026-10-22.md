# 개인용 대시보드 기능 재검토

검토일: **2026-10-22**. 결론은 **현재 기능은 개인용 첫 버전을 쓰기 시작할 수 있는 범위이며, 매일 쓰는 종합 정보 작업 공간으로는 보강할 핵심 흐름이 있다**는 것이다. 다음 개발의 중심은 위젯 개수보다 **수집 → 보관 → 검색 → 오늘 할 일 → 필요한 변화 알림**이어야 한다.

기존 80단계는 합의한 개인용 출시 후보의 구현·운영 검증 계획이다. 78단계 완료가 다른 제품의 모든 기능이나 실제 Mac 출시 완료를 뜻하지 않는다. 이 검토의 R01–R12는 **새 후속 제안이며 미구현**이다. 기존 F079/F080의 실제 Mac·외부 접속 인수도 그대로 남는다.

## 이번에 더 확인한 근거

기존 조사41개에 이어 원본 README **18개를 다시 또는 추가로 확인**했다. 그중 새 비교 대상은 Fenrus, Linkding, wallabag, Vikunja, Super Productivity, Planka, Trilium, Glances의 **8개**다. Karakeep은 기존 Hoarder의 이름/소유 저장소 변경으로 분류하여 신규 프로젝트 수에 중복 계산하지 않았다.

18개 모두 조회 시점의 commit을 먼저 확인하고 해당 commit URL로 문서를 다시 받아 SHA-256을 기록했다. 추가 경로404도 원장에 남겼다. 각 제품의 설치·실행 감사는 하지 않았다. 원본 문서의 기능 설명과 우리 코드에서 확인한 현재 구현을 비교한 조사다. [원본 URL·commit·해시·로컬 코드 확인 원장](feature-gap-sources-2026-10-22.json)에 재현 가능한 출처가 있다.

| 비교 대상 | 원본에서 확인한 강점 | 우리에게 중요한 차이 |
| --- | --- | --- |
| [Glance](https://github.com/glanceapp/glance#features) | YouTube·커뮤니티·주가·컨테이너 상태·서버 지표 | 기존 YAML 모드에 있는 상당수 정보가 새 웹 편집 모드에는 아직 없음 |
| [Homepage](https://github.com/gethomepage/homepage#features), [Homarr](https://github.com/homarr-labs/homarr/blob/HEAD/docs/README.md) | 서비스별 정보, 앱 실행, 통합 검색, 폭넓은 연동 | 일반 JSON 값 하나만으로 다양한 서비스 정보를 보기 좋게 표현하기 어려움 |
| [Dashy](https://github.com/Lissy93/dashy#status-indicators-), [Fenrus](https://github.com/revenz/Fenrus#configuration) | 서비스 상태·그룹·Smart Apps·검색 엔진 단축키 | 서비스 카드와 본인 사설망의 NAS/홈서버 연결 흐름이 부족 |
| [Linkding](https://github.com/sissbruecker/linkding#readme) | URL 메타데이터 자동 수집, 나중에 읽기, bookmarklet/확장, HTML 이동 | 현재 빠른 북마크는 보드 설정의 링크 배열이며 독립된 자료 보관함이 아님 |
| [Karakeep](https://github.com/karakeep-app/karakeep#features), [Linkwarden](https://github.com/linkwarden/linkwarden#readme), [wallabag](https://github.com/wallabag/wallabag#readme) | 링크 수집·본문 추출·검색·확장/모바일 입력·보존 | URL 붙여넣기로 자료를 모으는 흐름과 본문 읽기/재발견이 제한됨 |
| [Super Productivity](https://github.com/johannesjo/super-productivity#heavy_check_mark-features), [Planka](https://github.com/plankanban/planka#key-features), [Vikunja](https://github.com/go-vikunja/vikunja#readme) | 할 일 중심 작업 관리. Super Productivity는 일정 옆 하루 계획·외부 이슈 연동 명시 | 현재 Markdown 체크리스트는 표시만 하고 완료·기한·반복 작업을 관리하지 않음 |
| [Miniflux](https://github.com/miniflux/v2#features), [FreshRSS](https://github.com/FreshRSS/FreshRSS#readme), [Folo](https://github.com/RSSNext/Folo#-features) | 읽기 전문화, Miniflux 원문 추출/필터, Folo 규칙·액션·여러 형식 | 읽기 상태는 있지만 관심사 규칙·보관된 검색·조건 알림·다중 형식 읽기는 부족 |
| [Trilium](https://github.com/TriliumNext/Trilium#-features), [Memos](https://github.com/usememos/memos#readme) | 메모를 축적·탐색하는 독립 제품 흐름; Trilium은 본문 검색·버전·계층 명시 | 메모가 화면 배치와 묶여 있고 모아 읽기 검색에서 찾을 수 없음 |
| [Glances](https://github.com/nicolargo/glances#readme) | 호스트 CPU·메모리·디스크·네트워크와 웹/API 경로 | Mac 지표는 별도 수집기/권한/실기기 확인이 필요한 선택 연동; glanceapp과 다른 프로젝트 |

기능 설명을 섞지 않는다. FreshRSS README의 push는 출처에서 리더로 보내는 갱신 수신을 뜻하며 사용자에게 보내는 푸시 알림의 증거로 쓰지 않았다. Vikunja README는 자세한 기능을 외부 사이트로 안내하여 그 문서만으로 CalDAV·알림 세부 기능을 확정하지 않았다. Planka의 calendar/timeline·반복 카드 등은 해당 README에서 **유료 Pro**로 나뉘므로 무료 기본 기능으로 소개하지 않는다. AI 요약·OCR·웹 보존도 별도 비용과 자원이 드는 제품 기능이며 우리에게 이미 있다고 계산하지 않는다.

## 현재 코드에서 확인한 실제 빈틈

1. **메모·북마크의 보관 모델**: `web/src/editor/quickCapture.ts`는 메모마다 위젯을 만들고 32개 위젯 제한을 적용한다. 북마크는 위젯당20개 배열이다. `entry_ingestion.go`는 RSS/GitHub만 읽기함에 저장한다. 따라서 수백 개의 개인 메모/링크를 같은 읽기·태그·검색 흐름에 쌓을 수 없다.
2. **검색의 분리**: `App.tsx`는 위젯 제목을 필터링하고, 명령 팔레트는 화면/보드/추가 동작을 찾는다. FTS는 수집 entry의 제목·excerpt·출처를 대상으로 한다. 북마크 URL·메모 본문·일정을 하나의 결과 목록에서 찾는 기능은 없다.
3. **할 일의 부재**: `SafeMarkdown.tsx`의 체크박스는 disabled다. 완료 상태, 기한, 우선순위, 반복을 저장하는 task 모델이 없다. iCal은 읽기 전용이며 Google Calendar가 iCal URL을 제공하면 읽는 경로는 있으나 OAuth·일정 수정·양방향 CalDAV 동기화는 없다.
4. **오늘 화면의 제한**: `brief.go`는 그날의 RSS/GitHub entry 수와 최대20개 최근 항목을 반환한다. 일정·할 일·중요한 변경을 합친 우선순위 화면이나 의미 기반 요약은 아니다.
5. **원본 위젯과 새 모드의 차이**: `internal/glance/widget.go`에는 videos, markets/stocks, hacker-news, reddit, monitor, docker-containers, server-stats 등이 남아 있다. `internal/dashboard/catalog.go`의 웹 편집 모드는9종이다. YAML에서 사용 가능한 것을 웹에서 추가할 수 있다고 안내하면 안 된다. RSS로 일부 정보를 구독할 수 있는 것과 전용 영상/주가/서비스 카드도 구분한다.
6. **홈서버 연결 범위**: 서버 수집은 공개 HTTPS/443과 공개 IP 경계를 유지하며 사설·loopback 주소를 거부한다. 북마크 URL도 HTTPS/443 검증을 공유한다. `http://192.168.x.x:5000` NAS 같은 주소를 현재 기본 연동으로 지원한다고 말할 수 없다. private connection은 비밀 자격 증명 저장을 뜻하며 사설망 접근 허용과는 다르다.
7. **본문·오프라인**: 읽기 화면은 저장된 plain excerpt를 표시하고 원문 전체를 자동 추출하지 않는다. PWA는 공개 앱 파일만 캐시한다. 설치 가능하다는 이유로 개인 기사 본문의 오프라인 읽기가 된다고 설명하면 안 된다.
8. **알림과 규칙**: 갱신 오류·마지막 성공·재시도는 화면에 표시하지만 사용자 관심 키워드/할 일/서비스 장애를 모아 보내는 알림함·외부 알림 채널은 없다. JSON 표시는 단일 scalar 값이며 표·목록·추이 데이터 모델은 없다.

이미 구현된 다중 보드·편집 복구·읽음/저장/태그·OPML·iCal·안전한 연결·출처 표시·백업과 현재 검증 결과는 유효하다. 위 차이를 기능 완료로 새로 세거나 기존 기능을 삭제하는 조사는 아니다.

## 후속 구현 제안과 수용 기준

우선순위는 개인 한 명의 매일 사용하는 정보 대시보드를 기준으로 정했다. 모두 **제안/미구현**이며 추정 구현 시간을 약속하지 않는다.

| ID / 순서 | 제안 | 완료를 판단할 사용자 결과 |
| --- | --- | --- |
| R01 / 우선 | 독립된 개인 자료 보관함 | 메모·링크·읽을거리를 저장해도 위젯 수가 늘지 않는다. 보드 배치 삭제가 자료를 지우지 않으며 명시적 자료 삭제·태그·내보내기가 가능하다. 기존 보드 메모/북마크는 사전 백업과 미리보기로 이동한다. |
| R02 / 우선 | URL 수집·RSS 발견·외부에서 저장 | 사이트 주소로 제목·설명·아이콘/피드를 확인하고 저장한다. 같은 URL은 중복을 안내한다. bookmarklet 또는 공유용 진입점으로 대시보드를 찾아 열지 않고 수집한다. 먼저 메타데이터, 원문 추출은 명시적 별도 단계로 한다. |
| R03 / 우선 | 한 곳에서 찾는 통합 검색 | 메모 본문·북마크·수집 기사·일정을 구분된 결과로 찾고 바로 연다. 현재 한국어 검색·비공개 연결 철회·삭제·출처 권한을 유지한다. |
| R04 / 우선 | 실제 할 일과 오늘 화면 | 완료/미완료·기한·우선순위가 저장된다. 오늘 일정·기한 있는 할 일·핵심 읽을거리를 한 화면에서 처리한다. 자정·시간대·재시작 후 결과가 일관된다. 복잡한 프로젝트 관리와 양방향 일정 수정은 초기 범위에 넣지 않는다. |
| R05 / 다음 | 필요한 원본 정보 위젯의 웹 모드 이식 | 영상·개발/커뮤니티 소식부터 원본 출처·시각·오류 상태를 가진 카드로 추가한다. 주가/환율은 실제 필요와 무료 API 조건을 확인해 선택한다. 기존 YAML 구현을 인증·수집 제한 없이 그대로 우회 연결하지 않는다. |
| R06 / 다음 | 서비스 카드와 승인한 사설망 연결 | 본인 NAS/서버의 명시적으로 승인한 주소·포트만 접근한다. 링크 열기와 서버 수집을 분리하고 연결별로 경계를 둔다. 임의 사설망 스캔이나 Docker socket 전권을 기본 요구하지 않는다. 관리 클라우드의 네트워크 정책을 우회하지 않는다. |
| R07 / 다음 | 관심사 규칙·보관된 검색·알림함 | 키워드/출처 필터를 보관하고 같은 변경을 반복 알리지 않는다. 먼저 앱 내부 알림, 이후 ntfy/Gotify 등 선택 채널을 붙인다. 발송 실패·재시도·해제·조용한 시간을 다룬다. Mac 절전 중 수집/발송을 보장하지 않는다. |
| R08 / 다음 | 여러 형태의 데이터 카드 | 숫자·상태·진행률·목록·표를 선언형 설정으로 표현한다. 추이 그래프는 별도의 제한된 이력 저장이 생긴 뒤 제공한다. 현재 값 하나만 저장하면서 과거 추이가 있는 것처럼 그리지 않는다. |
| R09 / 다음 | 실제 연결까지 안내하는 첫 설정 | 기존 일상/개발/소식 템플릿을 활용해 사용자 소스 추가·필요 권한·연결 확인·저장까지 안내한다. 템플릿 선택만 끝낸 상태와 본인 데이터 연결 완료를 구분한다. |
| R10 / 선택 | 선택한 원문의 보존·오프라인 읽기 | 사용자가 고른 항목만 원문 추출/보존한다. 저장 용량·보관 기간·로그아웃 삭제를 다루며 모든 개인 API를 조용히 캐시하지 않는다. |
| R11 / 선택 | AI 요약·번역·의미 검색 | 출처로 돌아갈 수 있고 실패해도 원문을 읽는다. 모델/API 비용과 Mac 메모리를 확인해 opt-in으로 제공한다. 기본 제품은 유료 키나 상시 LLM 없이 동작한다. |
| R12 / 선택 | 테마·아이콘·레이아웃 개인화 및 추가 연동 | 의미 있는 정보 그룹·밀도·아이콘을 먼저 개선한다. Google OAuth/CalDAV 쓰기·특정 할 일 서비스는 실제 사용하는 도구가 정해진 뒤 추가한다. arbitrary JS 플러그인이나 다중 사용자 SaaS는 개인용 목표에 필수로 넣지 않는다. |

## 추천하는 다음 목표

**R01 → R02/R03 → R04**를 다음 묶음으로 삼는다. “기사·메모·링크를 모으고, 검색해서 찾고, 오늘 해야 할 일을 처리하는” 하루 흐름을 실제로 완성하는 것이 가장 큰 개선이다. 그 뒤 R05–R09에서 본인이 매일 보는 소스와 서비스 카드를 채운다. 홈서버 관리가 주 목적이라면 R06을 이 묶음과 함께 우선하고, 뉴스/학습이 주 목적이면 R07/R10을 앞당긴다.

Go·React·SQLite와 현재 catalog/connection/scheduler 경계는 유지할 수 있다. 자료/할 일은 보드 레이아웃과 수명을 분리하고, 검색 index에는 자료 유형과 삭제·권한 변경을 반영한다. 알림에는 dedupe와 영속 전송 상태, 과거 그래프에는 제한된 이력 테이블이 필요하다. 이 때문에 위젯 UI만 급히 늘리는 작업으로 취급하지 않는다. Redis·외부 검색 서버·호스트 전체 권한을 새 기본 요구로 만들 필요도 없다.
