import { expect, test } from "@playwright/test";
import type { Board } from "../src/types";
test("JSON pointer widget displays an escaped scalar without executing markup", async ({
  page,
  request,
}) => {
  const headers = { Origin: "http://127.0.0.1:8091" };
  const board: Board = await (
    await request.post("/api/v1/boards", {
      headers,
      data: { title: "JSON 검증" },
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
    await page.getByLabel("위젯 종류", { exact: true }).selectOption("json");
    await page.getByLabel("제목", { exact: true }).fill("내 측정값");
    await page
      .getByLabel("공개 JSON 주소", { exact: true })
      .fill("https://example.com/metrics.json");
    await page
      .getByLabel("JSON Pointer", { exact: true })
      .fill("/current/value");
    await page.getByLabel("값의 단위", { exact: true }).fill("단위");
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
    expect(saved.sources[0].config.pointer).toBe("/current/value");
    await page.route(`**/api/v1/boards/${board.id}/snapshots`, (route) =>
      route.fulfill({
        json: {
          [saved.sources[0].id]: {
            state: "fresh",
            data: {
              kind: "string",
              value: '<img src=x onerror="window.jsonExecuted=true">',
            },
          },
        },
      }),
    );
    await page.reload();
    await expect(page.locator(".json-value strong")).toHaveText(
      '<img src=x onerror="window.jsonExecuted=true">',
    );
    expect(await page.locator(".json-value img").count()).toBe(0);
    expect(await page.evaluate(() => "jsonExecuted" in window)).toBe(false);
    await expect(page.locator(".json-value small")).toHaveText("단위");
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
