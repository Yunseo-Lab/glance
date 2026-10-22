# 배치 엔진 실험 (F042)

2026-10-21, Chromium Linux, React 19.3.1 StrictMode, 360×800 터치 컨텍스트. 재현: `npm ci --prefix experiments/layout-engines`, `npm --prefix experiments/layout-engines run build`, `npm --prefix experiments/layout-engines run verify` (Chromium 경로는 환경변수로 지정 가능).

| 측정 | dnd-kit core 6.3.1 + sortable 10.0.0 | GridStack 14.0.0 |
|---|---:|---:|
| React 포함 JS gzip | 82,760 B | 93,446 B |
| 추가 CSS gzip | 0 B | 1,426 B |
| StrictMode 페이지 예외 | 0 | 0 |
| 키보드 Space → Down → Space 순서 변경 | 통과 | fixture에 기본 키보드 조작 없음 |
| CDP 터치 이동 | 통과 | 이 기본 fixture에서는 변경 관찰 못함 |
| 변경 뒤 React/DOM 순서 | beta,gamma,alpha | 변경 없음 |

**dnd-kit을 선택한다.** 현재 문서 모델은 순서와 카드 크기만 저장한다. React 소유 DOM과 순서 배열을 유지하면서 키보드 센서, 전용 터치 핸들, 기존 이동 버튼을 함께 사용할 수 있다. GridStack의 좌표/리사이즈 엔진은 현재 요구에 추가 저장 모델과 DOM 동기화가 필요하다. 이 결과는 GridStack이 터치를 지원하지 않는다는 주장이 아니다. 고정한 기본 fixture에서만 터치 성공을 관찰하지 못했고 별도 키보드 구현도 필요했다.

초기 자동화에서 두 센서 이벤트를 연속 발행해 dnd-kit도 실패했다. React 센서 활성화 후 다음 이벤트까지 기다리도록 fixture를 수정해 통과했다. 실제 앱에서는 터치·키보드·저장 후 재열기를 별도로 검증한다. 비교 수치는 작은 재현 앱 전체 번들이며 운영 앱 증가량이나 실기기 성능 측정이 아니다.
