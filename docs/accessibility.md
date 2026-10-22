# 접근성 검증

2026-10-22, axe-core/Playwright 4.14.0으로 밝음/어두움 보드, 위젯 편집, 빠른 기록, 보드 관리, 소스/개인 연결, 모아 읽기, 브리핑, 설정, 로그인, 360px 화면의 serious/critical 결과 0을 확인했다. 발견했던 낮은 보조 텍스트 대비, 어두운 달력 오늘 표시, 보이는 이름과 접근성 이름의 불일치를 고쳤다.

키보드로 skip link, 명령 검색 첫 초점, native dialog 열기/닫기/초점 복귀, Escape, reader 이동과 상태 변경을 점검했다. modal 공통 hook은 닫기 전에 native `close()`를 호출하고 원래 요소가 남아 있으면 초점을 돌린다. 입력/IME/다른 modal에서는 전역 단축키를 가로채지 않는다. reduced motion에서 부드러운 스크롤/불필요 애니메이션을 줄인다. Chrome 접근성 트리에서 dialog/combobox/textbox 이름과 필수 상태도 확인했다.

실제 Linux Orca 48.1 + AT-SPI2 2.56.2 + headed Chromium을 실행해 편집 버튼, 명령 팝업/복귀, 카탈로그 검색, 필수 제목 입력을 점검했다. 실제 브레일 모니터 출력에 `편집 button`, `명령 팔레트 열기 button`, `카탈로그 검색`, `제목 ... required`가 나타났다. 음성 합성은 껐다. 컨테이너의 이전 orphan zombie를 무시하는 local launcher 조정 외에 reader 엔진은 바꾸지 않았다. [검사 기록](verification/accessibility.json)에 범위와 원본 로그 해시를 남겼다.

재현: `PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH=/usr/bin/chromium npm --prefix web run test:e2e -- accessibility.spec.ts`. 실제 reader smoke는 X11/Orca/AT-SPI가 준비된 Linux session에서 Orca debug/braille monitor를 켜고 `node web/scripts/screen-reader-smoke.mjs`로 실행한다. 테스트는 임시 DB와 loopback 8098을 사용한다. 전체 음성 경험, Mac VoiceOver, 실제 휴대전화는 운영 기기 인수에서 별도로 확인해야 한다. 자동 검사나 이 제한된 reader 경로만으로 WCAG 준수를 선언하지 않는다.
