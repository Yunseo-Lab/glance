# 항목 검색

SQLite FTS5 trigram은 3자 이상 검색어에 사용한다. 1~2자는 저장 시 계산한 NFC/Unicode case-fold 문자열에서 찾는다. 여러 단어는 AND로 결합하고 검색어를 FTS 연산자로 실행하지 않는다. 한글 자모 조합과 완성형, 영문 대소문자, 숫자, 공백과 인용 부호를 검증한다. 검색어는 200자/8단어 이내이며 결과는 owner 범위의 날짜순 keyset 페이지다.

기본 빌드: `CGO_ENABLED=1 go build -tags sqlite_fts5 -o build/glance .`. CGO/FTS5가 없는 dashboard는 DB migration 전에 명확한 오류로 중단한다. 기존 YAML 모드 빌드는 가능하다. `/api/v1/capabilities`에서 기능을 확인한다. 검색 index와 정규화 문자열은 migration이 기존 항목으로 재구성하며 insert/update/delete trigger로 동기화한다.

1만 항목, Linux cloud, Go 1.27.3/SQLite 3.53.4, 쿼리당 warm-up 1회 뒤 30회 측정했다. 초기 1~2자 p95 335~357ms를 발견해 정규화 문자열을 영속화했고 재측정 p95는 13.8~15.0ms였다. 3자 이상 공통 검색은 p95 13.5ms, 없는 검색어는 1.4ms였다. [이전 측정](verification/search-10000-before.json)과 [이후 측정](verification/search-10000-after.json)을 보관한다. 이는 단일 클라이언트 수치다. 8동시 클라이언트·60분 부하와 실제 Mac 수치는 별도 인수 대상이다.

재현: `GLANCE_SEARCH_BENCH=1 go test -tags sqlite_fts5 ./internal/dashboard -run TestSearch10000Performance -count=1 -v`. 일반 race 검사에는 길이가 긴 성능 반복을 포함하지 않는다.
