# 개인용 실행 예산

기본 상한은 `internal/dashboard/limits.go`와 `/api/v1/limits`에 있다.
현재 서버는 소유자 보드 10개, 보드당 위젯/소스 32개, 모든 보드가 참조하는
서로 다른 source 128개를 검증한다. 추가·복제·저장은 한 트랜잭션에서
판정하므로 한도를 넘은 시도가 보드나 source 일부를 남기지 않는다.

사용하는 보드가 없는 source는 제거하지만, 그 source의 과거 entry와
사용자의 읽음/저장 상태는 지우지 않는다. source_id만 NULL로 바뀐다.
공유 source의 query를 바꾸려면 새 source ID를 만들고 보드의 참조를 바꾼다.
다른 보드의 revision을 무시하고 공유 query를 덮어쓰는 저장은 거부한다.

요청 본문은 1MiB, 공급자 응답은 2MiB, 동시 수집은 4개다. 호스트별 2개,
quota·예약 수집도 같은 예산을 적용한다. 캐시 자료의 최대 stale 시간은
24시간이다. 사용하지 않는 source cache는 모든 보드 참조를 확인하고 정리한다.

읽기 모델의 실행 예산은 항목 10,000개·저장하지 않은 항목 90일이다.
retention은 시간별로 실행하며 저장하지 않은 오래된 entry만 지운다.
수집은 1만 항목에서 새 항목만 보류하고 기존 항목의 정보 갱신을 계속한다.
사용자가 저장한 항목을 한도 때문에 지우지 않는다. 설정·진단에 항목 수와 보관 한도 도달을 표시한다.

보드 owner index, source owner index, source 참조 index, 항목 수집시각/소스
index, 저장 상태 index를 migration으로 유지한다. source 조회의 실제 SQLite
query plan에서 owner index 사용을 검사한다. 1만 항목 검색과 60분 soak는
아직 실행하지 않았으며 F061/F071에서 측정한다. 상한을 올릴 때 해당 데이터
크기의 지연·메모리·디스크와 복구 시간을 함께 다시 확인한다.
