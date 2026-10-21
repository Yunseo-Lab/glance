# 구현 진행 기록

최근 기록: 2026-10-21T23:09:36+00:00

조사·설계 8/8 · 신규 구현 검증 16/72 · 진행 중 1 · 예정 53 · 차단 0 · 외부 조건 2

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
