# 개인 대시보드 고도화: 80단계 구현 계획

작성일: **2026-10-21** · 기준 커밋: `6a4275b` · 범위: 소유자 한 명 / 맥 미니.

요청한 순서대로 **조사와 구현 계획을 먼저 작성**한다. 현재 제품은 개인용
출시 후보이며, 이 계획의 새 기능이 구현됐거나 실제 운영 서버에 출시됐다는
뜻은 아니다. 이전 출시 준비 43/45와 이 새로운 80단계의 숫자를 섞지 않는다.

목표는 정보가 많은 화면을 넘어 **빠르게 설정하고, 최신 상태와 출처를 믿고,
편집 실수에서 복구하고, 필요한 정보를 다시 찾을 수 있는 개인 정보 작업 공간**이다.
“프론티어급”은 비교 대상의 모든 기능 합집합이나 검증 없는 최상위 주장 대신,
아래 사용자 경험과 검증 기준으로 판단한다.

## 조사 결과와 핵심 선택

[관련 저장소 조사](research/related-repositories.md): 후보 41개 README 확인,
40개 참고/1개 제외. Homepage·Homarr·Dashy·Nimbus의 설정/연동,
Miniflux·FreshRSS·Folo의 수집/읽기, Linkwarden·Karakeep·Memos의 재발견,
Home Assistant의 카드 경계, dnd-kit/GridStack·TanStack Query의 UI 계약을
비교했다. 일부 소스·manifest도 추가로 읽었으며 실행 감사와는 구분했다.

[아키텍처 결정문](architecture/frontier-dashboard.md): Go·React·SQLite 유지,
명시적인 catalog와 connection, versioned migration, bounded scheduler,
편집 상태와 서버 상태 분리, 한국어 로컬 검색을 채택한다. 유료 서비스나
별도 검색/캐시 서버를 기본 요구하지 않는다. 모든 신규 라이브러리는 해당
버전에서 짧은 실험·호환성·번들·라이선스 검사 후 채택한다.

## 현재 기준선과 새로 필요한 것

| 영역 | 이미 있는 기준선 | 다음 목표 |
| --- | --- | --- |
| 데이터/배포 | SQLite, CAS revision, 온라인 backup/restore, Go+CGO 이미지 | 순차 migration, 복수 보드·공유 source, upgrade/rollback 실증 |
| 위젯/연동 | clock/calendar/note/bookmarks/rss/weather, RSS/날씨 fetch | catalog·안전한 연결·GitHub·일정·JSON·OPML |
| 편집 | 추가/설정/삭제/정렬/복제·JSON 이동·충돌 보존 | undo/redo·선택적 draft 복구·touch/키보드 품질·diff |
| 정보 신뢰성 | TTL·backoff·중복 수집 방지·최종 자료 유지 | 영속 scheduler·조건부 GET·quota·실패 분류·운영 상태 |
| 읽기/검색 | 위젯 제목 검색·category | 통합 인박스·읽음/저장·태그·한국어 검색·출처 있는 브리핑 |
| 인증/보안 | owner login·12h session·logout·SSRF·제한된 요청 | 연결 암호화·rotation·인증 모든 경로 제한·확장 fuzz/계약 검사 |
| 품질 | 기존 Go 16개/브라우저 16개 및 container/HTTPS/복구 검증 기록 | a11y·시각 회귀·mobile 성능·장시간 부하·실제 Mac 인수 |

기준선 검증은 [기존 기록](release-readiness.md)에 있다. 이 문서 작업에서 제품
회귀 검사를 다시 실행했다고 주장하지 않는다. 기능 변경에 맞는 검사를 이후
단계에서 다시 실행하고 증거를 갱신한다.

## 사용자 경험의 완료 기준

- 처음 로그인한 사용자가 템플릿에서 위젯을 추가하고 저장까지 진행할 수 있다.
  첫 작업의 목표는 **3분 이내**, 반복 추가는 **핵심 행동 3회 이내**다. 조작 수는
  열기→선택→저장 기준으로 세고 입력·연결 권한 부여는 별도 측정한다.
- 한 source가 멈춰도 나머지 화면과 편집·저장이 작동한다. 오래된 자료에는 시각과
  이유가 보이고, 원문으로 이동할 수 있으며 실패를 성공처럼 표현하지 않는다.
- 마우스 없이 위젯 추가·설정·이동·저장·취소·검색·읽기를 수행할 수 있다.
  360px 모바일과 한국어 IME·긴 텍스트에서 데이터나 필수 행동이 잘리지 않는다.
- 실수는 undo/redo로 복구하고, 다른 탭과의 충돌 및 재로그인이 로컬 편집을
  조용히 덮어쓰지 않는다. 저장된 데이터는 새 볼륨에 복원해 확인할 수 있다.
- 정보는 읽음/저장/태그와 로컬 검색으로 다시 찾는다. 브리핑은 모든 항목의
  출처·시간을 제공한다. LLM 없이 핵심 흐름을 사용할 수 있다.

## 측정 예산 — 현재 달성 수치가 아닌 목표

| 항목 | 목표와 측정 조건 |
| --- | --- |
| 첫 화면 | 고정 fixture 32개 카드, Chromium mobile viewport/CPU 4x slowdown/RTT 150ms/down 1.6Mbps에서 5회 중앙값 LCP ≤2.5s, CLS ≤0.1 |
| 조작 지연 | 위 fixture에서 주요 20회 조작의 입력→다음 paint p95 ≤200ms; 현장 INP와 실험 수치를 구분 |
| 초기 전송 | 초기 실행에 필요한 JS gzip ≤150KiB, CSS gzip ≤20KiB; 위젯·읽기 상세는 필요한 때 별도 로딩 |
| API·검색 | 고정 1만 항목/8동시 local clients에서 board read p95 ≤100ms, local search p95 ≤150ms; 느린 쿼리·오류율·payload 함께 기록 |
| 자원 | 앱 컨테이너 steady RSS ≤200MiB 후보, peak는 512MiB 제한 내; Caddy 별도 측정. 실제 맥 하드웨어에서 재검증 |
| 장시간 동작 | 128 sources·60분 혼합 부하에서 미처리 예외/계속 증가하는 작업·메모리·중복 수집 없음; 그 뒤 restart/restore 성공 |
| 접근성 | 자동 serious/critical 0 + 수동 키보드/초점/스크린리더 검사. 자동 검사만으로 WCAG 준수 선언하지 않음 |
| 비밀정보 | 응답·로그·export·browser persistence에서 credential 없음, 철회/rotation/backup key 복원 경로 검증 |

미달하면 측정 조건을 낮춰 통과로 만들지 않고 원인과 예산 변경 근거를 기록한다.
앞선 cloud-local 200회 읽기 결과는 이 새 성능·soak 목표의 증거가 아니다.

## 실행 순서

각 행은 한 검토 가능한 변경 단위다. 커밋 하나로 끝나기 어려우면 하위 작업을
추가하되 상위 ID와 수용 기준을 유지한다. 단계 시작 시 다음 고위험 항목을
선택 → 구현 → 해당 검사 → 통합 gate → 증거 기록 순서로 진행한다.

계획 작성 시점: **설계 기록 8/80, 신규 제품 구현 0/72**. 최신 진행은 [구현 기록](frontier-progress.md)을 본다. F001–F008은 이 문서와 조사물로
완료된 조사·설계다. F009부터는 미구현이며 F079–F080은 외부 조건이 필요하다.
실행 상태의 기계 판독 원장은 [frontier-backlog.json](frontier-backlog.json)이다.

### 0단계 묶음 — 조사·목표·설계 (F001–F008)

| ID | 구현 결과물 | 통과해야 할 확인 | 상태 |
| --- | --- | --- | --- |
| F001 | 현재 릴리스와 변경 지점 목록화 | 6a4275b의 단일 home·6종 위젯·수집·인증·복구와 파일 경계를 기록 | 조사·설계 기록 완료 |
| F002 | 관련 저장소 후보 확장 | topic·분류 색인·인접 제품을 통해 후보 41개 원본 README 확인 | 조사·설계 기록 완료 |
| F003 | 기능 비교와 범위 선별 | 40개 참고/1개 제외 및 각 채택·보류 이유를 출처와 연결 | 조사·설계 기록 완료 |
| F004 | 핵심 소스·의존성 확인 | registry·카드 경계·HTTP·query 정책 및 선택 manifest 조회 근거 보존 | 조사·설계 기록 완료 |
| F005 | 제품 목표와 품질 예산 정의 | 개인용·기본 무과금·정보 신뢰성·기기 제한과 측정 조건 문서화 | 조사·설계 기록 완료 |
| F006 | 아키텍처 결정문 작성 | catalog·source·connection·entry·scheduler·cache 경계와 대안 기록 | 조사·설계 기록 완료 |
| F007 | 호환성과 외부 조건 목록화 | DB v1/export v1/API·기존 YAML 보존, Mac·도메인·키 미확보를 분리 | 조사·설계 기록 완료 |
| F008 | 의존 순서와 단계별 완료 기준 작성 | 80개 고유 ID·선행 작업·수용 기준·검증 gate를 기계 판독 원장과 함께 저장 | 조사·설계 기록 완료 |

### 1단계 묶음 — 계약·모듈·저장 모델 (F009–F016)

| ID | 구현 결과물 | 통과해야 할 확인 | 상태 |
| --- | --- | --- | --- |
| F009 | 위젯·provider catalog 도입 | 기존 6종이 catalog로 등록되고 기존 설정·렌더·검증 동작 동일 | 예정 |
| F010 | config version과 capability 계약 | 알 수 없는 type/version 거부, 오래된 설정을 명시적으로 변환 | 예정 |
| F011 | API 계약과 TS 타입 생성 검사 | OpenAPI와 생성 타입 drift를 CI에서 검출하고 오류 응답 형태 통일 | 예정 |
| F012 | 응용 서비스와 HTTP 경계 분리 | handler가 비즈니스 저장·refresh를 직접 혼합하지 않고 기존 API 호환 | 예정 |
| F013 | 순차 DB migration 기반 | v1 fixture → 최신, 중간 실패 rollback, 미래 버전 거부, 사전 backup 경로 검증 | 예정 |
| F014 | 복수 보드 모델과 API | home 유지, owner-scoped CRUD·CAS revision·삭제 참조 제약 검증 | 예정 |
| F015 | 공유 source·connection·entry 저장 구조 | 동일 source 공유·독립 읽음 상태·원자적 이전 및 고아 참조 거부 | 예정 |
| F016 | 용량·보관·작업 예산 확정 | 10보드/32위젯/128소스/1만 항목 후보의 상한·삭제·보관 정책과 query plan 기록 | 예정 |

### 2단계 묶음 — 수집 엔진·신선도·장애 격리 (F017–F024)

| ID | 구현 결과물 | 통과해야 할 확인 | 상태 |
| --- | --- | --- | --- |
| F017 | 결과 envelope와 오류 분류 | fresh/stale/error·최종 성공·다음 재시도·정제된 원인 코드가 일관됨 | 예정 |
| F018 | 영속 due/lease scheduler | 브라우저 종료 후 갱신, 재시작 lease 회수, 중복 작업 방지 | 예정 |
| F019 | 조건부 HTTP 상태 저장 | ETag/Last-Modified와 304에서 본문 유지·freshness 갱신 검증 | 예정 |
| F020 | Retry-After·jitter·host별 제한 | 429/503/timeout과 긴 Retry-After 처리, 재시도 폭주가 없음을 검증 | 예정 |
| F021 | 설정 세대와 취소 경합 처리 | 수집 중 source 수정·삭제·connection 교체 후 옛 응답이 새 상태를 덮지 못함 | 예정 |
| F022 | 허용량 있는 수동 새로고침 | 중복 클릭·다중 탭이 quota를 넘지 않으며 버튼에 대기 이유 표시 | 예정 |
| F023 | 수집 상태·진단 API | 실패 소스·마지막 성공·작업 수를 credential/원문 유출 없이 조회 | 예정 |
| F024 | 가짜 시간·장애 주입 테스트 | 동시 4개·호스트별 2개, 304·429·취소·panic·느린 응답을 결정적으로 재현 | 예정 |

### 3단계 묶음 — 연결 보안·개인정보 (F025–F032)

| ID | 구현 결과물 | 통과해야 할 확인 | 상태 |
| --- | --- | --- | --- |
| F025 | connection secret 저장 경계 | credential이 board JSON·로그·응답·export에 나타나지 않는 계약 검사 | 예정 |
| F026 | 외부 key file 기반 암호화 | 버전 있는 AEAD ciphertext, 잘못된 key/손상 거부, key 권한 검증 | 예정 |
| F027 | 최소 권한 연결 생성·테스트·철회 | 서버만 credential에 접근, 테스트 응답은 capability·만료 상태만 반환 | 예정 |
| F028 | key·credential rotation과 복구 | 중간 실패 원자성, 예전 key 제거 전 재암호화 검증, key 별도 백업 절차 | 예정 |
| F029 | 브라우저·자동화 인증 방어 개선 | Basic/로그인 실패 제한·credential 길이·동시 세션·proxy 신뢰 경계 검증 | 예정 |
| F030 | origin·CSRF·헤더·redirect 경계 | 설정 HTTPS origin 일치, 신뢰하지 않는 forwarding header 거부, 외부 유출 방지 | 예정 |
| F031 | SSRF·악성 피드·달력 입력 검증 | IPv4/IPv6·DNS 변경·redirect·폭탄/초대형 입력·HTML/URL payload로 fuzz/regression | 예정 |
| F032 | 보관·삭제·내보내기 privacy 정책 | 삭제된 source/connection의 잔존 자료, secret 없는 export, 로그·진단 redaction 검사 | 예정 |

### 4단계 묶음 — 제품 흐름·탐색·서버 상태 (F033–F040)

| ID | 구현 결과물 | 통과해야 할 확인 | 상태 |
| --- | --- | --- | --- |
| F033 | 보드 중심 shell과 정보 구조 | 홈/보드/소스/읽기/설정의 뒤로 가기·직접 URL·빈 상태가 일관됨 | 예정 |
| F034 | 첫 사용 템플릿과 안내 | 뉴스·일상·개발 템플릿 → 첫 source/위젯 → 저장을 외부 키 없이 완료 | 예정 |
| F035 | 검색 가능한 위젯 catalog | 종류·필요 연결·미리보기·지원 범위와 추가 행동을 한 화면에서 확인 | 예정 |
| F036 | 일관된 설정 폼과 미리보기 | 필드별 오류·도움말·취소·키보드 focus, preview가 저장/실수집을 하지 않음 | 예정 |
| F037 | 전 위젯 공통 상태 표현 | loading/empty/stale/error/paused 및 최종 갱신·재시도·원문 이동 규칙 통일 | 예정 |
| F038 | 명령 팔레트와 단축키 | Ctrl/Meta+K·검색·보드 이동·위젯 추가; IME 조합·입력창 단축키 충돌 방지 | 예정 |
| F039 | 연결·source 관리 화면 | 어느 보드가 사용하는지·테스트·재연결·정지·삭제 영향을 확인 | 예정 |
| F040 | server query와 editor 상태 분리 | query key/abort/invalidation/retry를 통합하고 logout 시 private state 제거 | 예정 |

### 5단계 묶음 — 편집기·레이아웃·안전한 변경 (F041–F048)

| ID | 구현 결과물 | 통과해야 할 확인 | 상태 |
| --- | --- | --- | --- |
| F041 | 편집 명령/reducer로 상태 분리 | 추가·삭제·중복·설정·이동의 불변성과 save/cancel을 집중 검증 | 예정 |
| F042 | 배치 엔진 비교 실험 | dnd-kit/GridStack의 현재 React 호환·터치·키보드·번들·재현 fixture를 기록해 하나 선택 | 예정 |
| F043 | 키보드·터치 정렬 구현 | 시각 순서=DOM 순서=저장 순서, 스크린리더 알림과 이동 버튼 유지 | 예정 |
| F044 | undo/redo와 삭제 복구 | 명령 연속 실행·되돌리기·다시 적용·저장 경계·히스토리 상한 검사 | 예정 |
| F045 | 선택적 기기 draft 복구 | 기본 off, 허용한 draft만 복구·만료·logout 삭제; credential 저장 불가 | 예정 |
| F046 | 다중 탭 충돌과 변경 비교 | revision 충돌 시 로컬 편집 보존, 변경 비교/백업/명시적 재적용 | 예정 |
| F047 | 복수 보드 편집과 레이아웃 | 보드 전환의 dirty guard, 이름/복제/삭제, 360~1440px 배치와 ID 보존 | 예정 |
| F048 | versioned import/export와 템플릿 | 기존 v1 파일·잘못된 참조·새 버전·용량 제한·preview 후 원자적 적용 검사 | 예정 |

### 6단계 묶음 — 실용 연동과 위젯 깊이 (F049–F056)

| ID | 구현 결과물 | 통과해야 할 확인 | 상태 |
| --- | --- | --- | --- |
| F049 | RSS/Atom/JSON Feed 품질 강화 | 안정 item ID·안전한 원문 URL·정렬·빈/손상/중복 피드와 validator 재수집 검사 | 예정 |
| F050 | 날씨 도시·단위·시간대 설정 | 위치 검색·수동 좌표·섭씨/화씨·빈 관측·stale 표시; 실제 endpoint 검증은 별도 기록 | 예정 |
| F051 | GitHub 공개 정보 위젯 | 공개 repo release/issue/PR의 필요한 범위를 먼저 구현하고 rate-limit 상태 표시 | 예정 |
| F052 | GitHub 개인 연결 지원 | read-only 최소 권한·만료/철회·private 데이터 격리; 실제 키 없이는 live 완료 표시 금지 | 예정 |
| F053 | 읽기 전용 iCalendar 일정 | TZID·종일·DST·반복/예외·기간 상한·민감 calendar URL 저장 경계 검사 | 예정 |
| F054 | 제한된 JSON/API 위젯 | 허용 HTTPS + JSON Pointer + 정해진 표시 타입만 제공, 임의 JS/template/헤더 실행 거부 | 예정 |
| F055 | OPML과 RSS bridge 호환 | OPML 중복/용량/악성 XML 검사, RSSHub/Bridge의 표준 피드 입력 호환 | 예정 |
| F056 | 연동 공통 contract suite | 성공·빈 결과·오래됨·429·401·철회·손상 응답과 fixture/live 상태를 연동별 기록 | 예정 |

### 7단계 묶음 — 읽기·저장·검색·브리핑 (F057–F064)

| ID | 구현 결과물 | 통과해야 할 확인 | 상태 |
| --- | --- | --- | --- |
| F057 | 통합 항목 저장과 중복 식별 | source/item identity·원문·수집시각·발췌문 보존, 중복·서명 URL을 손상시키지 않음 | 예정 |
| F058 | 읽음·저장·나중에 읽기 | 낙관 UI의 실패 rollback·다중 탭 동기화·cache 만료와 사용자 저장 분리 | 예정 |
| F059 | 태그·필터·일괄 동작 | 페이지/선택 범위를 명시하고 대량 변경·취소·키보드 조작 검증 | 예정 |
| F060 | FTS5 빌드·인덱스·migration | sqlite_fts5 태그를 native/Docker/CI에 통일하고 capability probe·재색인 검사 | 예정 |
| F061 | 한국어 포함 로컬 검색 | 1만 항목·한글/영문/숫자/1~2글자·공백·특수문자 fixture, query plan과 p95 기록 | 예정 |
| F062 | 인박스·읽기 상세 화면 | 키보드 다음/이전, 원문 출처, 추적 이미지 기본 차단·HTML 안전 표시 | 예정 |
| F063 | 메모·북마크 빠른 수집 | 생성·태그·편집·검색의 일관된 UX와 제한된 Markdown, raw HTML 비활성 | 예정 |
| F064 | 출처 있는 오늘 브리핑 | 기간·중복·중요 표시를 결정적으로 묶고 모든 항목에 원문·수집시각·오래됨 표시 | 예정 |

### 8단계 묶음 — 접근성·오프라인·성능 검증 (F065–F072)

| ID | 구현 결과물 | 통과해야 할 확인 | 상태 |
| --- | --- | --- | --- |
| F065 | 접근성 자동·수동 gate | axe serious/critical 0 + 키보드/초점/스크린리더/reduced motion 수동 검증 | 예정 |
| F066 | 전송·렌더 성능 예산 적용 | lazy widget/검색 청크·불필요 render 정리, 정해진 mobile profile로 성능 기록 | 예정 |
| F067 | 설치 가능한 최소 PWA | manifest·아이콘·버전 갱신·app-shell만 캐시, 인증 API 무차별 캐시 금지 | 예정 |
| F068 | 오프라인과 세션 lifecycle | 네트워크 복귀·업데이트·logout cache purge·미전송 draft 충돌과 공유 기기 검사 | 예정 |
| F069 | 시각 회귀와 제품 흐름 감사 | 고정 fixture로 360/768/1440px·밝음/어두움·긴 텍스트·오류 상태 스크린샷 비교 | 예정 |
| F070 | 시간대·언어·느린 장치 검사 | 서울/UTC/DST·자정·한국어 IME·긴 번역·느린 네트워크·touch 경로 검사 | 예정 |
| F071 | 부하·장시간·자원 회귀 | 8동시 API·1만 항목 검색·source 128개·60분 soak의 지연/메모리/중복 작업 측정 | 예정 |
| F072 | 업그레이드 전 보안·호환 검사 | Go/npm/이미지 검사 결과 분리, contract·backup·export·YAML 및 기존 16 E2E 회귀 | 예정 |

### 9단계 묶음 — 운영·배포·실기기 인수 (F073–F080)

| ID | 구현 결과물 | 통과해야 할 확인 | 상태 |
| --- | --- | --- | --- |
| F073 | 재현 가능한 운영 패키지 | 새 의존성/FTS5/설정 포함 native·ARM64/AMD64 image·non-root·readiness 검증 | 예정 |
| F074 | 단계별 CI gate와 아티팩트 | 계약·race·browser·a11y·migration·container·의존성 검사와 실패 증거 보존 | 예정 |
| F075 | 버전·SBOM·체크섬 릴리스 | 검토한 태그/commit·라이선스·native 패키지·이미지 digest의 추적성 확보 | 예정 |
| F076 | v1 → 새 버전 upgrade/rollback | 실제 v1 fixture 사전 backup → migration → 동작 → 구버전+이전 DB 복귀 리허설 | 예정 |
| F077 | 백업 보관·오프사이트·복구 점검 | rotation이 마지막 정상본을 지우지 않고 key+DB를 분리 복구, 체크섬·복원 시각 기록 | 예정 |
| F078 | 운영 진단·자동 시작·runbook | 민감정보 없는 로그/상태/백업 경고, Docker Desktop·잠자기·디스크 조건 문서화 | 예정 |
| F079 | 실제 맥 미니 설치와 외부 연동 인수 | 실제 CPU 아키텍처·재부팅·절전 복귀·날씨·선택한 credential 연동 검증 | 외부 조건 필요 |
| F080 | 공인 HTTPS·휴대전화·최종 출시 | DNS/TLS/다른 네트워크·로그인·편집·검색·backup/restore 확인 후 실제 출시 표시 | 외부 조건 필요 |

## 의존성과 통합 gate

원장의 `depends_on`은 순차 진행을 기본으로 하고 추가 선행 조건을 명시한다.
예컨대 private GitHub F052는 secret 저장/rotation F025–F028을 건너뛰지 못하며,
PWA F067은 인증·캐시 경계 F030/F040 이후에 진행한다. 자유 그리드나 query
라이브러리 실험에서 채택하지 않은 대안은 구현 완료로 세지 않는다.

| Gate | 통과 범위 | 다음 단계로 가기 위한 증거 |
| --- | --- | --- |
| G1: F016 | 계약·저장 기반 | 기존 v1 fixture와 API/export/YAML 회귀, migration rollback, catalog/type 계약 |
| G2: F032 | 수집·보안 | scheduler 장애 주입·race, SSRF/redirect/credential redaction·rotation 복구 |
| G3: F048 | 핵심 제품 경험 | 첫 사용/편집/undo/충돌/복수 보드/키보드·모바일 E2E, 이전 설정 import |
| G4: F064 | 유용한 정보 작업 | 연동별 fixture/live 구분, 읽음/저장/검색/브리핑에서 출처와 접근 경계 |
| G5: F072 | 출시 품질 | 시각/a11y/성능/soak/security/compatibility 보고서와 실패 잔여 목록 |
| G6: F078 | 배포 가능한 후보 | 실제 image·archive·checksum·v1 upgrade/rollback·새 볼륨 복구 및 운영 안내 |
| G7: F080 | 실제 출시 | 실제 Mac 재부팅/네트워크/TLS/휴대전화/선택한 실연동과 운영자 acceptance |

매 단계에서 전체 비싼 검사를 반복하지 않는다. 실제 변경과 관련된 검사를 먼저
실행하고 gate에서 넓힌다. 실패 원인을 진단·수정한 뒤 영향을 받는 검사를 다시
실행한다. fixture만 성공한 연동은 `code-tested`, 실제 연결까지 확인한 연동은
`live-verified`로 구분한다. 미연결 기능은 연결 필요 상태를 표시하며 성공한
데이터처럼 데모 값을 보여주지 않는다.

## 위험과 대응

| 위험 | 먼저 하는 대응 |
| --- | --- |
| 기능 추가 전에 현재 동작이 깨짐 | F009부터 호환 fixture를 유지하고 단계별 변경; 대규모 일괄 재작성 금지 |
| 마이그레이션·삭제가 데이터 손실 유발 | 사전 backup·트랜잭션·참조 검사·v1 실데이터 fixture·새 파일 복구 |
| 동시 수집·재시작으로 API 폭주 | DB lease, host별 quota, Retry-After/jitter, 설정 generation 검사 |
| generic API/일정 URL이 비밀·사설망을 노출 | connection secret 경계, 기본 public-only egress, URL/로그 redaction |
| 편집 기능이 키보드·touch에서 퇴화 | 배치 엔진 실험과 move-button fallback, DOM/저장 순서 계약 |
| PWA가 개인 자료를 공유 기기에 남김 | app-shell 우선, private 응답 무차별 캐시 금지, 명시적 opt-in과 logout purge |
| 한국어 검색이 문서상으로만 동작 | trigram/짧은 검색어 fixture, FTS5 빌드 probe, 실제 query plan·지연 검사 |
| 새 라이브러리·모델이 기본 실행 비용 증가 | 번들·메모리 gate, 선택 feature 분리, Redis/외부 검색/LLM 기본 의존 금지 |
| 네트워크 정책/키 부재로 실연동 검증 불가 | fixture 개발은 진행하고 해당 live 검증만 외부 미완료로 유지 |
| Docker VFS 빌드가 cloud 디스크 소모 | 순차 image build·재사용 cache, 디스크 확인, 이 작업의 build cache만 정리 |

## 선택 확장: LLM 브리핑 (핵심 80단계와 별도)

기본 브리핑 F064가 유용하고 검증된 뒤 다음 여섯 작업을 독립 수행한다.
O1 사용자가 선택한 remote/local provider와 데이터 범위 계약,
O2 key·비용·timeout·일일 사용 상한,
O3 외부 콘텐츠를 명령으로 취급하지 않는 입력 경계 및 tool 실행 차단,
O4 항목별 인용·원문·불확실성 표시와 근거 없는 문장 거부,
O5 고정 평가 세트로 인용 정확성/누락/환각·fallback 검증,
O6 실제 맥 메모리/응답 시간 또는 실제 provider 비용 측정 후 opt-in 활성화.
이 영역은 아직 구현되지 않았고 사용자 API 키·모델·맥 상태 없이 완료라고 세지 않는다.

## Git와 외부 조건

기존 `release/personal-dashboard` 브랜치에서 검토 가능한 묶음으로 커밋하고
검증 후 push한다. main 자동 merge나 실제 서버 배포를 Git push와 혼동하지 않는다.
새 기능이 cloud에서 검증돼도 Mac 가동, DNS/TLS, 사설 연결·사용자 credential,
휴대전화 실기기 인수는 별도다. 지금은 운영 Mac과 도메인이 없으므로 이를
필요로 하는 단계는 외부 미완료로 남긴다.

**가장 먼저 구현할 범위는 F009–F016**이다. registry/계약·migration·보드/source
경계부터 잡아 이후 UI와 연동 추가가 같은 코드를 계속 복제하지 않게 한다.

## 이 계획 문서의 확인

2026-10-21에 문서/JSON 일치 여부를 검사했다. 80개 ID의 유일성, 각 선행 작업이
앞에 존재하는 DAG, 설계 8/예정 70/외부 2의 상태 합계, 41개 출처와 40개
포함 판단, 원본 문서 해시 및 로컬 문서 링크가 통과했다. 변경은 조사·설계
문서와 README 링크에 한정되며 앱 기능을 구현하거나 새 제품 회귀 검사를
실행한 것으로 표시하지 않는다.
