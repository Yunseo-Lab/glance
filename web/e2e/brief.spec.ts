import { expect, test } from "@playwright/test";
test("daily brief displays dated counts, direct citations and timezone controls", async ({
  page,
}) => {
  const requests: string[] = [];
  await page.route("**/api/v1/brief?*", (route) => {
    requests.push(route.request().url());
    const params = new URL(route.request().url()).searchParams;
    return route.fulfill({
      json: {
        date: params.get("date"),
        timezone: params.get("timezone"),
        total: 1,
        unread: 1,
        saved: 0,
        truncated: false,
        fingerprint: "fixture",
        asOf: "2026-10-22T00:00:00Z",
        items: [
          {
            id: "brief-one",
            title: "근거가 있는 소식",
            url: "https://example.com/original",
            excerpt: "보관한 원문 발췌",
            sourceTitle: "원본 피드",
            sourceIds: [],
            collectedAt: "2026-10-22T00:00:00Z",
            isRead: false,
            isSaved: false,
            tags: [],
          },
        ],
      },
    });
  });
  await page.goto("/#/brief");
  await expect(
    page.getByRole("heading", { name: "하루 브리핑", exact: true }),
  ).toBeVisible();
  await expect(
    page.getByRole("link", { name: /근거가 있는 소식/ }),
  ).toHaveAttribute("href", "https://example.com/original");
  await expect(page.getByText("원본 피드", { exact: true })).toBeVisible();
  await page.getByLabel("브리핑 날짜", { exact: true }).fill("2026-11-01");
  await page
    .getByLabel("브리핑 시간대", { exact: true })
    .fill("America/New_York");
  await page.getByRole("button", { name: "브리핑 보기", exact: true }).click();
  await expect
    .poll(() =>
      requests.some(
        (url) =>
          new URL(url).searchParams.get("timezone") === "America/New_York" &&
          new URL(url).searchParams.get("date") === "2026-11-01",
      ),
    )
    .toBe(true);
  await page
    .getByRole("button", { name: "모아 읽기로 이동", exact: true })
    .click();
  await expect(
    page.getByRole("heading", { name: "모아 읽기", exact: true }),
  ).toBeVisible();
});
