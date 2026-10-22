import { expect, test } from "@playwright/test";
import type { Board } from "../src/types";
test("city search fills coordinates, timezone and Fahrenheit without saving before confirmation", async ({
  page,
  request,
}) => {
  const origin = "http://127.0.0.1:8091",
    headers = { Origin: origin };
  const board: Board = await (
    await request.post("/api/v1/boards", {
      headers,
      data: { title: "날씨 검증" },
    })
  ).json();
  try {
    await page.route("**/api/v1/locations?q=*", (route) =>
      route.fulfill({
        json: [
          {
            name: "New York",
            region: "New York",
            country: "USA",
            latitude: 40.71,
            longitude: -74.01,
            timezone: "America/New_York",
          },
        ],
      }),
    );
    await page.route(`**/api/v1/boards/${board.id}/snapshots`, (route) =>
      route.fulfill({ json: {} }),
    );
    await page.goto(`/#/boards/${board.id}`);
    await page
      .getByRole("button", { name: "위젯 추가", exact: true })
      .first()
      .click();
    await page.getByLabel("위젯 종류", { exact: true }).selectOption("weather");
    await page.getByLabel("제목", { exact: true }).fill("뉴욕 날씨");
    await page.getByLabel("도시 찾기", { exact: true }).fill("New York");
    await page.getByRole("button", { name: "지역 검색", exact: true }).click();
    await page
      .getByRole("button", { name: "New York · New York · USA", exact: true })
      .click();
    await expect(page.getByLabel("위도", { exact: true })).toHaveValue("40.71");
    await expect(page.getByLabel("날씨 시간대", { exact: true })).toHaveValue(
      "America/New_York",
    );
    await page
      .getByLabel("온도 단위", { exact: true })
      .selectOption("fahrenheit");
    await page
      .getByRole("button", { name: "위젯 추가", exact: true })
      .last()
      .click();
    expect(
      (await (await request.get(`/api/v1/boards/${board.id}`)).json()).widgets,
    ).toHaveLength(0);
    await page
      .getByRole("button", { name: "대시보드 저장", exact: true })
      .click();
    await expect(
      page.getByText("변경 사항을 저장했습니다.", { exact: true }),
    ).toBeVisible();
    const saved: Board = await (
      await request.get(`/api/v1/boards/${board.id}`)
    ).json();
    expect(saved.sources[0].config.units).toBe("fahrenheit");
    expect(saved.sources[0].config.timezone).toBe("America/New_York");
    await page.route(`**/api/v1/boards/${board.id}/snapshots`, (route) =>
      route.fulfill({
        json: {
          [saved.sources[0].id]: {
            state: "fresh",
            data: {
              location: "New York",
              temperature: 32,
              unit: "fahrenheit",
              code: null,
              humidity: null,
              feelsLike: null,
            },
          },
        },
      }),
    );
    await page.reload();
    await expect(
      page.getByText("습도 정보 없음", { exact: true }),
    ).toBeVisible();
    await expect(
      page.getByText("체감 정보 없음", { exact: true }),
    ).toBeVisible();
    await expect(page.locator(".weather-view strong")).toHaveText("32°F");
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
