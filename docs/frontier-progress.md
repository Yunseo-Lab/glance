# 구현 진행 기록

최근 기록: 2026-10-22T00:07:39+00:00

조사·설계 8/8 · 신규 구현 검증 41/72 · 진행 중 1 · 예정 28 · 차단 0 · 외부 조건 2

전체 계획: [80단계](frontier-implementation-plan.md). 완료는 구현 후 검증 증거가 있는 경우만 센다. 실제 Mac·도메인·개인 키 검증은 fixture 성공과 구분한다.

| 시각 (UTC) | 단계 | 상태 | 결과·다음 작업 |
| --- | --- | --- | --- |
| 2026-10-21T22:44:58+00:00 | F009 | in_progress | 기존 6종 위젯과 2종 provider를 versioned catalog로 옮기고 기존 설정 호환을 검증한다. |
| 2026-10-21T22:47:16+00:00 | F009 | verified | 6종 위젯·2종 provider catalog와 renderer registry 전환, 기존 설정과 인증 호환 확인. |
| 2026-10-21T22:47:16+00:00 | F010 | in_progress | 기존 version 생략 설정을 v1으로 유지하고 미래 configVersion을 거부하는 계약을 추가한다. |
| 2026-10-21T22:47:42+00:00 | F010 | verified | version 생략/명시 v1 호환, 음수·미래 버전과 trailing JSON 거부, catalog capability 계약 검증. |
| 2026-10-21T22:47:42+00:00 | F011 | in_progress | OpenAPI 기반 TS 타입·Go catalog JSON 생성과 drift 검사, UI의 수동 타입/label 중복을 제거한다. |
| 2026-10-21T22:49:52+00:00 | F011 | verified | OpenAPI生成 TS 타입·Go catalog JSON을 UI가 사용하며 CI drift 검사와 Go DTO/enum 계약 검사가 통과했다. |
| 2026-10-21T22:49:52+00:00 | F012 | in_progress | 보드 저장·검증·snapshot 읽기 use case를 HTTP handler에서 분리한다. |
| 2026-10-21T22:51:20+00:00 | F012 | verified | 저장·캐시 정리·snapshot use case를 application service로 분리. 전체 Go race와 기존 브라우저 16개 통과. |
| 2026-10-21T22:51:20+00:00 | F013 | in_progress | 기존 DB의 자동 사전 백업·순차 migration·checksum·실패 rollback과 미래 버전 거부를 구현한다. |
| 2026-10-21T22:52:28+00:00 | F013 | verified | v1 설정·캐시 보존, 업그레이드 전 검증된 backup, migration 반복/rollback/checksum/future-version 검증 통과. |
| 2026-10-21T22:52:28+00:00 | F014 | in_progress | 소유자 범위 복수 보드 CRUD·CAS revision·home 호환 API와 보드 간 캐시 보존을 구현한다. |
| 2026-10-21T22:54:21+00:00 | F014 | verified | 복수 보드 CRUD·복제·owner 격리·CAS 저장/삭제·동시 10보드 제한·다른 보드 cache 보존과 home API 호환 검사 통과. |
| 2026-10-21T22:54:21+00:00 | F015 | in_progress | 공유 source·connection·entry 저장 구조와 보드 참조를 원자적으로 정규화한다. |
| 2026-10-21T22:55:52+00:00 | F015 | verified | source 공유·참조 제약·충돌 저장 rollback·읽음/저장 상태 분리·모호한 기존 데이터 migration rollback 검사 통과. |
| 2026-10-21T22:55:52+00:00 | F016 | in_progress | 소스 수·응답 크기·보관 기간·entry 상한과 indexed query plan을 고정하고 경계 검사를 추가한다. |
| 2026-10-21T22:58:18+00:00 | F016 | verified | G1 통과: 보드/소스 원자적 한도·owner index query plan·v1 migration·전체 보드 backup/restore·계약·race/vet·YAML·기존 E2E 검증. |
| 2026-10-21T22:58:18+00:00 | F017 | in_progress | snapshot 오류 분류·최종 성공·다음 재시도 정보를 일관된 결과 envelope로 확장한다. |
| 2026-10-21T23:00:13+00:00 | F017 | verified | 안전한 오류 분류와 마지막 시도·성공·재시도 metadata를 추가하고 민감한 원인 비노출 검증을 통과했다. |
| 2026-10-21T23:00:13+00:00 | F018 | in_progress | DB lease·due time을 공유하는 예약 갱신과 crash 후 lease 회수를 검증한다. |
| 2026-10-21T23:02:03+00:00 | F018 | verified | 브라우저 없는 갱신·due 영속화·중복 lease 거부·crash 만료 회수·옛 worker의 lease 덮어쓰기 방지 검사 통과. |
| 2026-10-21T23:02:03+00:00 | F019 | in_progress | ETag/Last-Modified를 snapshot과 원자적으로 저장하고 304에서 원문 데이터를 보존한다. |
| 2026-10-21T23:02:03+00:00 | F019 | verified | 조건부 요청 headers·304 freshness 갱신·기존 데이터 유지·본문 없는 304 거부·validators 저장 검사 통과. |
| 2026-10-21T23:02:03+00:00 | F020 | in_progress | Retry-After·jitter·동시 전역 4개/호스트 2개와 재시도 폭주 방지를 구현한다. |
| 2026-10-21T23:02:53+00:00 | F020 | verified | 長い Retry-After·HTTP date·상한 있는 jitter/backoff·호스트 2개/전역 4개 동시성 검사를 race 10회 반복 통과. |
| 2026-10-21T23:02:53+00:00 | F021 | in_progress | 삭제·설정 교체·lease 만료 뒤 옛 수집 결과가 snapshot/entry를 다시 만들지 못하도록 commit fence를 추가한다. |
| 2026-10-21T23:04:45+00:00 | F021 | verified | 삭제 즉시 취소, source generation·lease 만료·reference 검사를 통과한 결과만 원자적으로 저장. 미등록 source의 운영 fetch 거부 검사 통과. |
| 2026-10-21T23:04:45+00:00 | F022 | in_progress | 수동 새로고침을 소스별 30초/소유자 10회 분당으로 제한하고 upstream Retry-After를 보존한다. |
| 2026-10-21T23:07:07+00:00 | F022 | verified | 수동 갱신의 소스/소유자 quota·동시 탭 원자성·upstream 대기 보존·화면의 비활성/대기 설명까지 검증. |
| 2026-10-21T23:07:07+00:00 | F023 | in_progress | 소스 상태·사용 보드·예약/실행 작업과 민감정보 없는 진단 API, 일시정지·재개를 구현한다. |
| 2026-10-21T23:08:27+00:00 | F023 | verified | 소스별 상태/참조 보드/작업 metadata·본문 없는 diagnostics·pause/resume·동일 query alias의 정상 동작 검사 통과. |
| 2026-10-21T23:08:27+00:00 | F024 | in_progress | 수집 장애·가짜 시계·재시작·취소·동시성의 통합 regression gate를 실행하고 고장 시나리오를 보강한다. |
| 2026-10-21T23:09:36+00:00 | F024 | verified | 가짜 시계의 backoff/복구·oversize·panic·304·lease/crash·삭제취소·호스트/global 제한을 race 10회 반복. 전체 브라우저 18개 통과. |
| 2026-10-21T23:09:36+00:00 | F025 | in_progress | credential이 board·source JSON·export·browser·로그에 노출되지 않는 connection 계약을 구현한다. |
| 2026-10-21T23:11:27+00:00 | F025 | verified | connection 응답 DTO와 비밀 저장형을 분리하고 credential marker가 인증된 API·board export에 없음을 검증했다. |
| 2026-10-21T23:11:27+00:00 | F026 | in_progress | DB 밖 key file과 버전 있는 AES-GCM envelope·row binding·잘못된 key/손상·권한 검사를 구현한다. |
| 2026-10-21T23:13:04+00:00 | F026 | verified | DB 밖 key file·AES-GCM·record/provider binding·잘못된 키/손상 거부·0600 생성·기존 ciphertext에 대한 키 재생성 거부를 검증했다. |
| 2026-10-21T23:13:04+00:00 | F027 | in_progress | 연결 생성·안전한 metadata 조회·실제 endpoint 테스트·철회와 최소 권한 capability 응답을 구현한다. |
| 2026-10-21T23:15:05+00:00 | F027 | verified | 연결 생성·ciphertext 저장·공개 origin만 반환·권한 범위를 과장하지 않는 endpoint 테스트·quota·철회 검사 통과. 실제 개인 키 live 연결은 미검증. |
| 2026-10-21T23:15:05+00:00 | F028 | in_progress | 이전 DB/key를 보존하며 새 key file로 전 connection을 원자적으로 재암호화하고 실패 복구를 검증한다. |
| 2026-10-21T23:16:39+00:00 | F028 | verified | 새 key로 원자적 재암호화·옛 key/DB rollback·손상 중간 실패 rollback·신규 key 정리·출력 overwrite 거부 검증 통과. |
| 2026-10-21T23:16:39+00:00 | F029 | in_progress | 자동화 Basic 인증도 실패 횟수 제한을 적용하고 session·header·credential 길이와 proxy header 위조를 검증한다. |
| 2026-10-21T23:17:18+00:00 | F029 | verified | Basic 실패 10회/분·로그인 별도 quota·정상 200회 읽기 비소모·X-Forwarded-For 위조 무효·길이/MIME/header 상한 검사 통과. |
| 2026-10-21T23:17:18+00:00 | F030 | in_progress | 설정 HTTPS origin을 엄격히 비교하고 신뢰하지 않는 forwarding header와 cross-origin credential redirect를 거부한다. |
| 2026-10-21T23:18:45+00:00 | F030 | verified | HTTPS scheme/host/port origin·userinfo/path/query 거부·위조 forwarding header 무시·운영 Host의 DNS rebinding 방어와 교차 origin auth header 제거 검사 통과. |
| 2026-10-21T23:18:45+00:00 | F031 | in_progress | SSRF 특수 주소·리다이렉트·악성 피드·요청 입력과 키 envelope의 fuzz/regression을 보강한다. |
| 2026-10-21T23:24:20+00:00 | F031 | verified | 재시작 후 DNS rebinding/redirect/주소대역 회귀, DTD·ENTITY 명시 거부, origin/envelope fuzz를 재실행해 통과. 검증 로그를 workspace에 보존. |
| 2026-10-21T23:24:20+00:00 | F032 | in_progress | 삭제·철회·source orphan·90일 보관·export/log redaction과 G2 보안 통합 검증을 마무리한다. |
| 2026-10-21T23:28:22+00:00 | F032 | verified | G2 통과: saved entry를 보존하는 90일 retention·삭제/철회·key/DB rollback·SSRF/XXE/fuzz·API 전수 계약·race/vet·브라우저 18개. Go/npm 알려진 취약점 발견 0. |
| 2026-10-21T23:28:23+00:00 | F033 | in_progress | 복수 보드·소스·읽기·설정의 navigation shell과 직접 URL·뒤로 가기·빈 상태를 구현한다. |
| 2026-10-21T23:33:19+00:00 | F033 | verified | 복수 보드 direct URL·모바일 탐색·소스/읽기/설정 화면·뒤로 가기·새로고침 context·404 복구와 기존 시나리오 검증. |
| 2026-10-21T23:33:20+00:00 | F034 | in_progress | 외부 API 키 없이 시작하는 일상·개발·소식 템플릿과 첫 위젯 안내를 구현한다. |
| 2026-10-21T23:35:22+00:00 | F034 | verified | 키 없는 3개 템플릿·off-line 일상 설정·preview 비저장·명시적 저장/재시작 유지와 navigation 검증 통과. |
| 2026-10-21T23:35:22+00:00 | F035 | in_progress | 위젯 catalog 검색·필요 연결·미리보기·지원 범위와 추가 흐름을 editor에 통합한다. |
| 2026-10-21T23:38:10+00:00 | F035 | verified | catalog 검색·로컬/외부 분류·필요 연결 설명·native 선택·저장 전 비수집 preview와 빈 검색 상태 검증 통과. |
| 2026-10-21T23:38:10+00:00 | F036 | in_progress | 필드별 오류·focus·일관된 도움말과 preview/cancel 비저장 동작을 검증한다. |
| 2026-10-21T23:39:50+00:00 | F036 | verified | 설정 builder를 preview/submit이 공유하고 필드 오류 aria-invalid/describedby/focus·취소 비저장·기존 편집 10개 회귀를 검증했다. |
| 2026-10-21T23:39:50+00:00 | F037 | in_progress | 모든 remote 위젯의 loading/empty/stale/error/paused·최종 성공·다음 시도·원문 규칙을 공통 상태 표현으로 통합한다. |
| 2026-10-21T23:42:25+00:00 | F037 | verified | 공통 SourcePresentation으로 loading/empty/stale/error/paused·원인/재시도·최종 성공·persistent manual cooldown을 통합하고 상태 fixture를 검증했다. |
| 2026-10-21T23:42:25+00:00 | F038 | in_progress | Ctrl/Meta+K 명령 팔레트·보드/화면 이동·위젯 추가와 IME/입력 중 단축키 안전성을 구현한다. |
| 2026-10-21T23:45:29+00:00 | F038 | verified | Ctrl/Meta+K·검색/Enter/화면 이동·위젯 추가·slash 검색과 IME/기존 modal 입력 충돌 방지를 브라우저에서 검증. |
| 2026-10-21T23:45:29+00:00 | F039 | in_progress | 연결 추가·권한 테스트·철회·삭제·소스 일시정지/갱신/참조 보드 이동을 관리 화면에 연결한다. |
| 2026-10-21T23:51:51+00:00 | F039 | verified | 개인 연결 생성·명시적 검사·대체 연결 생성·철회·삭제, 소스 정지/계속/갱신과 참조 보드 이동을 구현했다. 가짜 토큰은 외부 전송 없이 검사 실패를 모킹했다. |
| 2026-10-21T23:51:51+00:00 | F040 | in_progress | TanStack Query 5.104.5로 서버 조회 키·취소·재시도·무효화와 private cache 제거를 통합하고 editor draft는 별도 유지한다. |
| 2026-10-21T23:53:57+00:00 | F040 | verified | 서버 조회에 private query key, abort, 재시도, mutation 무효화, online/visible 재조회와 logout/401 캐시 제거를 통합했다. 서버 상태와 미저장 editor draft를 분리했다. |
| 2026-10-21T23:53:57+00:00 | F041 | in_progress | 저장본/초안/편집 여부와 추가·수정·삭제·이동·복제·취소 명령을 순수 reducer로 분리하고 ID/참조/한도 불변성을 검사한다. |
| 2026-10-21T23:56:06+00:00 | F041 | verified | 명령 reducer로 저장본·초안·편집 상태를 분리했다. 참조 보존/정리, 유효한 이동, 중복 ID·한도·다른 보드 치환 거부와 취소/저장 경계를 검증했다. |
| 2026-10-21T23:56:06+00:00 | F042 | in_progress | React 19.3.1 StrictMode에서 dnd-kit 6.3.1/sortable 10.0.0과 GridStack 14.0.0의 실제 번들·키보드·모바일 터치·DOM 순서 fixture를 비교한다. |
| 2026-10-21T23:56:58+00:00 | F042 | verified | 동일 React StrictMode fixture로 두 엔진의 번들/터치/키보드를 실측했다. dnd-kit은 키보드와 터치 후 DOM/배열 순서 일치. GridStack 기본 fixture 한계와 결과 범위를 기록하고 dnd-kit을 선택했다. |
| 2026-10-21T23:56:58+00:00 | F043 | in_progress | dnd-kit 전용 핸들과 Pointer/Touch/Keyboard 센서, 한국어 안내, 기존 이동 버튼으로 DOM 순서와 저장 순서를 함께 변경한다. |
| 2026-10-21T23:59:17+00:00 | F043 | verified | 전용 터치 핸들, 키보드 센서, 한국어 스크린리더 안내와 이동 버튼을 구현했다. 360px에서 두 입력 방식의 DOM/저장 ID·순서와 Escape 취소를 검증했다. |
| 2026-10-21T23:59:17+00:00 | F044 | in_progress | 최대 50개 명령의 undo/redo, 삭제된 source 복구, 분기 시 redo 삭제와 저장/취소 경계, 입력란과 충돌하지 않는 Ctrl/Cmd+Z를 검증한다. |
| 2026-10-22T00:00:37+00:00 | F044 | verified | 최대 50개 undo/redo, 삭제된 source 복구, 새 편집의 redo 폐기와 저장/취소 경계를 구현했다. 버튼 및 Ctrl/Cmd+Z, ID 복원과 히스토리 상한 검증 통과. |
| 2026-10-22T00:00:37+00:00 | F045 | in_progress | 기본 off인 기기 초안 복구를 구현한다. 24시간 만료, 같은 revision에 한해 서버 검증 후 복구, source 설정/비밀정보 제외와 logout 삭제를 검증한다. |
| 2026-10-22T00:01:52+00:00 | F045 | verified | 기기 초안 복구는 기본 off이고 24시간 만료/동일 revision/서버 검증을 거친다. source 설정은 저장하지 않고 ID로 재구성한다. 새 source는 서버 저장을 요구하며 logout 시 opt-in과 초안을 지운다. |
| 2026-10-22T00:01:52+00:00 | F046 | in_progress | CAS 충돌에서 로컬 초안을 보존하고 원본/내 변경/원격 변경을 비교한다. 비충돌 변경은 보존하며 충돌 항목의 로컬 우선 재적용은 명시적으로 선택하고 다시 서버 검증/저장한다. |
| 2026-10-22T00:03:22+00:00 | F046 | verified | 타 탭 저장 알림은 ID/revision만 교환하고 CAS로 최종 확인한다. 로컬 초안을 보존해 세 버전을 비교하고 비충돌 원격 변경을 합친 뒤 검증된 초안으로 재적용한다. 재적용 자체는 저장하지 않는다. |
| 2026-10-22T00:03:22+00:00 | F047 | in_progress | 보드 생성/템플릿/이름/복제/삭제 UI와 dirty guard를 연결하고 browser back 취소 시 history 위치를 복원한다. |
| 2026-10-22T00:05:18+00:00 | F047 | verified | 보드 생성/템플릿/이름/복제/삭제 UI를 구현했다. 복제 ID 분리, 삭제 제한, dirty guard와 back 취소 후 재이동/forward 보존, 기존 360px 탐색을 검증했다. |
| 2026-10-22T00:05:18+00:00 | F048 | verified | v1 import/export 호환과 위젯/소스 참조, 미래 config version, UTF-8 1MiB 한도를 검증했다. 검증 전 렌더/DB 변경 없이 초안 미리보기 후 CAS 저장하며 기존 템플릿 흐름도 통과했다. |
| 2026-10-22T00:05:18+00:00 | F049 | in_progress | RSS/Atom/JSON Feed를 안정 ID·정규 URL·날짜 정렬·중복 제거로 통합하고 형식 오류/보안 fixture를 검증한다. |
| 2026-10-22T00:07:39+00:00 | F049 | verified | RSS/Atom/JSON Feed 항목의 안정 ID, 상대 URL/추적 파라미터 정규화, 날짜 내림차순, 중복 제거와 50개 상한, HTML 없는 발췌를 구현했다. malformed/DTD/unsafe 링크 fixture와 race 검사 통과. |
| 2026-10-22T00:07:39+00:00 | F050 | in_progress | 날씨 위치 검색과 단위/시간대 설정, 없는 응답값의 null 처리 및 안전한 표시를 구현하고 mock HTTP로 검증한다. |
