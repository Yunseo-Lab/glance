import { expect, test } from "@playwright/test";
test("reading state updates optimistically, survives reload and rolls back a failed save", async ({
  page,
}) => {
  const item = {
    id: "reading-fixture",
    title: "수집한 기사",
    url: "https://example.com/article",
    excerpt: "실제 원문을 읽기 전에 살펴볼 내용",
    sourceTitle: "검증 소스",
    sourceIds: [],
    collectedAt: "2026-10-22T00:00:00Z",
    isRead: false,
    isSaved: false,
    tags: [],
  };
  let fail = false;
  await page.route("**/api/v1/entries?*", (route) =>
    route.fulfill({
      json: { items: [item], total: 1, capacityReached: false },
    }),
  );
  await page.route("**/api/v1/entries/*/state", async (route) => {
    await new Promise((resolve) => setTimeout(resolve, 350));
    if (fail)
      return route.fulfill({
        status: 503,
        json: {
          error: "저장하지 못했습니다. 다시 시도해 주세요.",
          code: "unavailable",
        },
      });
    const changes = route.request().postDataJSON();
    if (changes.read !== undefined) item.isRead = changes.read;
    if (changes.saved !== undefined) item.isSaved = changes.saved;
    return route.fulfill({
      json: {
        isRead: item.isRead,
        isSaved: item.isSaved,
        tags: [],
        updatedAt: new Date().toISOString(),
      },
    });
  });
  await page.goto("/#/inbox");
  await page
    .getByRole("button", { name: "수집한 기사 읽음으로", exact: true })
    .click();
  await expect(
    page.getByRole("button", { name: "수집한 기사 안 읽음으로", exact: true }),
  ).toHaveAttribute("aria-pressed", "true");
  await expect(
    page.getByRole("button", { name: "수집한 기사 안 읽음으로", exact: true }),
  ).toBeEnabled();
  await page.reload();
  await expect(
    page.getByRole("button", { name: "수집한 기사 안 읽음으로", exact: true }),
  ).toHaveAttribute("aria-pressed", "true");
  fail = true;
  await page
    .getByRole("button", { name: "수집한 기사 저장", exact: true })
    .click();
  await expect(
    page.getByRole("button", { name: "수집한 기사 저장 해제", exact: true }),
  ).toHaveAttribute("aria-pressed", "true");
  await expect(page.getByRole("alert")).toContainText("저장하지 못했습니다");
  await expect(
    page.getByRole("button", { name: "수집한 기사 저장", exact: true }),
  ).toHaveAttribute("aria-pressed", "false");
  fail = false;
  await page
    .getByRole("button", { name: "수집한 기사 저장", exact: true })
    .click();
  await expect(
    page.getByRole("button", { name: "수집한 기사 저장 해제", exact: true }),
  ).toBeEnabled();
  await page.getByRole("button", { name: "저장함", exact: true }).click();
  await expect(page.getByRole("link", { name: /수집한 기사/ })).toHaveAttribute(
    "href",
    "https://example.com/article",
  );
});
