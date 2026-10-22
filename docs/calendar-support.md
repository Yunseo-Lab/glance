# 읽기 전용 일정

`agenda` 위젯은 공개 HTTPS iCalendar 또는 암호화된 `calendar` 개인 연결을 사용한다. 주소·토큰은 서버에서만 해결하며 CalDAV 쓰기나 일정 수정/삭제 API는 없다. 첨부·알람·설명 안의 URL을 추가로 요청하지 않는다.

IANA TZID와 선택한 기본 시간대의 floating time, UTC, 종일 DATE(끝 날짜 제외), DTEND/기본 길이/DURATION을 읽는다. YEARLY/MONTHLY/WEEKLY/DAILY RRULE, RDATE, EXDATE, 단일 RECURRENCE-ID 이동/취소를 확장한다. DST 전후 벽시계와 종일 날짜 경계를 테스트한다. 존재하지 않는 시작 시각을 조용히 다른 시각으로 바꾸지 않는다.

입력 2MiB·VEVENT 1000개·중첩 16단계, 향후 31일과 직전 하루의 창, 결과 200개로 제한한다. 반복별 최대 500개, 오래된 recurrence 탐색 예산을 두고 SECONDLY/MINUTELY/HOURLY, 다중 BYHOUR/BYMINUTE/BYSECOND와 120년 넘는 시작을 제한한다. 커스텀 VTIMEZONE 이름, RDATE PERIOD, RANGE=THISANDFUTURE 전체 변경 등 RFC의 모든 형태를 지원한다고 주장하지 않는다. 일부 일정을 제외하거나 한도에 닿으면 경고를 표시한다. 이 기능은 원본 캘린더의 대체 저장소가 아니다.

반복 파일이 수정되지 않아도 날짜 창이 이동하므로 iCalendar는 15분 캐시 뒤 본문을 다시 받아 확장한다. 다른 공급자의 304 본문 재사용 정책을 그대로 적용하지 않는다. 실제 개인 주소/계정 인수는 사용자 연결 뒤 별도로 수행한다.
