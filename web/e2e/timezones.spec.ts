import { test, expect } from "@playwright/test";
for (const scenario of [
  {
    name: "Seoul midnight",
    zone: "Asia/Seoul",
    start: "2026-10-22T14:59:59Z",
    clock: "00:00",
    day: "23",
  },
  {
    name: "UTC midnight",
    zone: "UTC",
    start: "2026-10-22T23:59:59Z",
    clock: "00:00",
    day: "23",
  },
  {
    name: "New York spring DST gap",
    zone: "America/New_York",
    start: "2026-03-08T06:59:59Z",
    clock: "03:00",
    day: "8",
  },
  {
    name: "New York autumn DST fold",
    zone: "America/New_York",
    start: "2026-11-01T05:59:59Z",
    clock: "01:00",
    day: "1",
  },
]) {
  test.describe(scenario.name, () => {
    test.use({ timezoneId: scenario.zone, locale: "en-US" });
    test("clock and calendar follow the actual local date boundary", async ({
      page,
      request,
    }) => {
      await page.clock.install({ time: new Date(scenario.start) });
      const board = await (await request.get("/api/v1/dashboard")).json();
      board.sources = [];
      board.widgets = [
        {
          id: "zone-clock",
          type: "clock",
          title: "시간 경계",
          size: "small",
          config: { timezone: scenario.zone },
        },
        {
          id: "zone-calendar",
          type: "calendar",
          title: "날짜 경계",
          size: "small",
          config: {},
        },
      ];
      await page.route("**/api/v1/dashboard", (route) =>
        route.fulfill({ json: board }),
      );
      await page.goto("/");
      await expect(page.locator(".clock-time")).toBeVisible();
      await page.clock.fastForward(2000);
      await expect(page.locator(".clock-time")).toContainText(scenario.clock);
      await expect(
        page.locator('.calendar-grid [aria-current="date"]'),
      ).toHaveText(scenario.day);
      await expect(page.locator("html")).toHaveAttribute("lang", "ko");
    });
  });
}

test("long translated labels remain readable on a narrow touch viewport", async ({
  page,
}) => {
  await page.setViewportSize({ width: 360, height: 800 });
  await page.goto("/");
  await expect(page.locator("main h1")).toBeVisible();
  // Pseudo-translation tests expansion only; it does not claim a translation service integration.
  await page
    .locator("main h1")
    .evaluate(
      (node) =>
        (node.textContent =
          "All your important information together in one personal dashboard"),
    );
  await page
    .getByRole("button", { name: "설정 내보내기", exact: true })
    .evaluate((node) => {
      for (const child of node.childNodes)
        if (child.nodeType === Node.TEXT_NODE)
          child.textContent = " Export personal dashboard configuration ";
    });
  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= innerWidth,
    ),
  ).toBe(true);
  const heading = await page.locator("main h1").boundingBox();
  expect(heading!.width).toBeLessThanOrEqual(360);
  await expect(
    page.getByRole("button", { name: "빠른 기록", exact: true }),
  ).toBeVisible();
});
