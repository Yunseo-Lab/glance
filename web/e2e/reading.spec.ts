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

test("bulk selection saves and tags the page, then tag filtering preserves the chosen label", async ({
  page,
}) => {
  const items = [1, 2].map((index) => ({
    id: `tag-fixture-${index}`,
    title: `태그 기사 ${index}`,
    url: `https://example.com/${index}`,
    excerpt: "본문",
    sourceTitle: "검증 소스",
    sourceIds: [],
    collectedAt: "2026-10-22T00:00:00Z",
    isRead: false,
    isSaved: false,
    tags: [] as string[],
  }));
  await page.route("**/api/v1/entries?*", (route) => {
    const tag = new URL(route.request().url()).searchParams.get("tag");
    const filtered = items.filter((item) => !tag || item.tags.includes(tag));
    return route.fulfill({
      json: { items: filtered, total: filtered.length, capacityReached: false },
    });
  });
  await page.route("**/api/v1/tags", (route) =>
    route.fulfill({
      json: items.some((item) => item.tags.includes("중요"))
        ? [{ tag: "중요", count: 2 }]
        : [],
    }),
  );
  await page.route("**/api/v1/entries/bulk/state", (route) => {
    const input = route.request().postDataJSON();
    for (const item of items)
      if (input.ids.includes(item.id)) {
        if (input.saved !== undefined) item.isSaved = input.saved;
        if (input.read !== undefined) item.isRead = input.read;
        if (input.tags !== undefined) item.tags = input.tags;
      }
    return route.fulfill({ json: { updated: input.ids.length } });
  });
  await page.goto("/#/inbox");
  await page
    .getByRole("checkbox", { name: "이 페이지 모두 선택", exact: true })
    .check();
  await page
    .getByRole("button", { name: "선택 항목 저장", exact: true })
    .click();
  await expect(
    page.getByRole("button", { name: "태그 기사 1 저장 해제", exact: true }),
  ).toBeEnabled();
  await page
    .getByRole("button", { name: "선택 태그 지정", exact: true })
    .click();
  await page.getByLabel("태그", { exact: true }).fill("중요");
  await page.getByRole("button", { name: "태그 저장", exact: true }).click();
  await expect(page.getByRole("dialog")).toHaveCount(0);
  await expect(page.locator(".tag-chip")).toHaveCount(2);
  await page.getByLabel("태그별 보기", { exact: true }).selectOption("중요");
  await expect(page.locator("article.reading-entry")).toHaveCount(2);
  await expect(page.getByLabel("태그별 보기", { exact: true })).toHaveValue(
    "중요",
  );
});

test("Korean search waits for IME completion and short queries update the reading list", async ({
  page,
}) => {
  const entries = [
    {
      id: "search-one",
      title: "서울 프론티어 소식",
      url: "https://example.com/one",
      excerpt: "한국어 검색",
      sourceTitle: "소스",
      sourceIds: [],
      collectedAt: "2026-10-22T00:00:00Z",
      isRead: false,
      isSaved: false,
      tags: [],
    },
    {
      id: "search-two",
      title: "부산 다른 소식",
      url: "https://example.com/two",
      excerpt: "다른 내용",
      sourceTitle: "소스",
      sourceIds: [],
      collectedAt: "2026-10-22T00:00:00Z",
      isRead: false,
      isSaved: false,
      tags: [],
    },
  ];
  const queries: string[] = [];
  await page.route("**/api/v1/entries?*", (route) => {
    const query = new URL(route.request().url()).searchParams.get("q") || "";
    queries.push(query);
    const items = entries.filter(
      (item) => !query || (item.title + item.excerpt).includes(query),
    );
    return route.fulfill({
      json: { items, total: items.length, capacityReached: false },
    });
  });
  await page.goto("/#/inbox");
  await expect(page.locator("article.reading-entry")).toHaveCount(2);
  const search = page.getByLabel("수집한 소식 검색", { exact: true });
  await search.dispatchEvent("compositionstart");
  await search.fill("서");
  await page.waitForTimeout(450);
  expect(queries).not.toContain("서");
  await search.dispatchEvent("compositionend");
  await expect(page.locator("article.reading-entry")).toHaveCount(1);
  expect(queries).toContain("서");
  await search.fill("서울");
  await expect.poll(() => queries.includes("서울")).toBe(true);
  await search.fill("없는검색어");
  await expect(
    page.getByRole("heading", {
      name: "해당하는 소식이 없습니다",
      exact: true,
    }),
  ).toBeVisible();
});
