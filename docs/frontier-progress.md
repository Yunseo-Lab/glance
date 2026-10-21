# 구현 진행 기록

최근 기록: 2026-10-21T22:51:20+00:00

조사·설계 8/8 · 신규 구현 검증 4/72 · 진행 중 1 · 예정 65 · 차단 0 · 외부 조건 2

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
