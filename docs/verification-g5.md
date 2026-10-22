# G5 어댑터 검증

2026-10-22: 전체 Go race/vet, OpenAPI/생성 타입 검사와 브라우저 50개 통과. npm audit 개발 의존성 포함 0건. 다섯 provider의 공통 contract matrix가 성공·malformed·권한·404·429/Retry-After·5xx·응답 초과·timeout을 검사한다. 가짜 비밀정보는 mock transport 밖으로 보내지 않는다.

Node가 순수 편집기 테스트에서 JSON catalog를 읽을 때 import attribute가 없어 검사를 시작하지 못한 문제를 수정하고 전체 50개를 다시 실행했다. Vite 렌더뿐 아니라 테스트 런타임에서도 같은 catalog를 사용한다.

실제 운영 호스트와 개인 자격증명 인수는 [provider ledger](verification/provider-ledger.json)에서 미검증으로 남긴다. 이 기록은 이미지 OS 취약점 검사나 실제 맥 미니 인수를 대신하지 않는다.
