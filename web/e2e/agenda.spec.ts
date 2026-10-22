import { expect, test } from "@playwright/test";
import type { Board } from "../src/types";
test("read-only agenda stores its query and renders timezone and all-day events safely", async ({
  page,
  request,
}) => {
  const headers = { Origin: "http://127.0.0.1:8091" };
  const board: Board = await (
    await request.post("/api/v1/boards", {
      headers,
      data: { title: "일정 검증" },
    })
  ).json();
  try {
    await page.route(`**/api/v1/boards/${board.id}/snapshots`, (route) =>
      route.fulfill({ json: {} }),
    );
    await page.goto(`/#/boards/${board.id}`);
    await page
      .getByRole("button", { name: "위젯 추가", exact: true })
      .first()
      .click();
    await page.getByLabel("위젯 종류", { exact: true }).selectOption("agenda");
    await page.getByLabel("제목", { exact: true }).fill("내 일정");
    await page
      .getByLabel("공개 일정 주소", { exact: true })
      .fill("https://example.com/calendar.ics");
    await page
      .getByLabel("일정 시간대", { exact: true })
      .fill("America/New_York");
    await page
      .getByRole("button", { name: "위젯 추가", exact: true })
      .last()
      .click();
    await page
      .getByRole("button", { name: "대시보드 저장", exact: true })
      .click();
    await expect(
      page.getByText("변경 사항을 저장했습니다.", { exact: true }),
    ).toBeVisible();
    const saved: Board = await (
      await request.get(`/api/v1/boards/${board.id}`)
    ).json();
    expect(saved.sources[0].type).toBe("ical");
    await page.route(`**/api/v1/boards/${board.id}/snapshots`, (route) =>
      route.fulfill({
        json: {
          [saved.sources[0].id]: {
            state: "fresh",
            data: {
              title: "Calendar",
              warnings: [],
              events: [
                {
                  id: "one",
                  title: "미팅",
                  start: "2026-11-01T14:00:00Z",
                  end: "2026-11-01T15:00:00Z",
                  allDay: false,
                  timezone: "America/New_York",
                },
                {
                  id: "two",
                  title: "휴가",
                  start: "2026-11-02T00:00:00-05:00",
                  end: "2026-11-03T00:00:00-05:00",
                  allDay: true,
                  timezone: "America/New_York",
                },
              ],
            },
          },
        },
      }),
    );
    await page.reload();
    await expect(page.getByText("미팅", { exact: true })).toBeVisible();
    await expect(page.locator(".agenda-event").first()).toContainText(
      "America/New_York",
    );
    await expect(page.locator(".agenda-event").nth(1)).toContainText("종일");
    expect(
      await page.getByRole("button", { name: /일정 (수정|삭제|생성)/ }).count(),
    ).toBe(0);
  } finally {
    const latest: Board = await (
      await request.get(`/api/v1/boards/${board.id}`)
    ).json();
    await request.delete(`/api/v1/boards/${board.id}`, {
      headers,
      data: { revision: latest.revision },
    });
  }
});
