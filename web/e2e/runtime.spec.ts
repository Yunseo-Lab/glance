import { expect, test } from "@playwright/test";
test("manual refresh shows an explicit cooldown after an accepted request", async ({
  page,
}) => {
  await page.route("**/api/v1/sources/*/refresh", (route) =>
    route.fulfill({
      status: 202,
      contentType: "application/json",
      body: JSON.stringify({
        accepted: true,
        nextAllowedAt: new Date(Date.now() + 30_000).toISOString(),
      }),
    }),
  );
  await page.goto("/");
  await page.getByRole("button", { name: "새로고침", exact: true }).click();
  await expect(
    page.getByRole("button", { name: /새로고침 \(\d+초 대기\)/ }),
  ).toBeDisabled();
});
test("provider rate limit is explained and cannot be bypassed by another click", async ({
  page,
}) => {
  await page.route("**/api/v1/sources/*/refresh", (route) =>
    route.fulfill({
      status: 429,
      headers: { "Retry-After": "60" },
      contentType: "application/json",
      body: JSON.stringify({
        error: "정보 제공자의 대기 시간을 적용합니다.",
        code: "too_many_requests",
      }),
    }),
  );
  await page.goto("/");
  await page.getByRole("button", { name: "새로고침", exact: true }).click();
  await expect(
    page.getByText("정보 제공자의 대기 시간을 적용합니다.", { exact: true }),
  ).toBeVisible();
  await expect(
    page.getByRole("button", { name: /새로고침 \(\d+초 대기\)/ }),
  ).toBeDisabled();
});

test("remote widgets consistently explain loading, empty, stale, error and paused states", async ({
  page,
  request,
}) => {
  const board = await (await request.get("/api/v1/dashboard")).json();
  const source = board.sources.find(
    (source: { type: string }) => source.type === "rss",
  );
  let snapshot: Record<string, unknown> = { state: "loading" };
  await page.route("**/api/v1/snapshots", (route) =>
    route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({ [source.id]: snapshot }),
    }),
  );
  const scenarios: [Record<string, unknown>, string][] = [
    [{ state: "loading" }, "소식을 가져오고 있어요"],
    [
      {
        state: "fresh",
        data: { title: "Fixture", items: [] },
        updatedAt: new Date().toISOString(),
      },
      "아직 게시물이 없습니다.",
    ],
    [
      {
        state: "stale",
        data: {
          title: "Fixture",
          items: [{ title: "Last success", url: "https://example.com/last" }],
        },
        error: "temporary",
        errorCode: "rate_limited",
        retryAt: new Date(Date.now() + 60000).toISOString(),
      },
      "갱신이 지연되어 마지막 정보를 표시합니다.",
    ],
    [
      { state: "error", errorCode: "authentication" },
      "연결 인증을 다시 확인하세요.",
    ],
    [{ state: "paused" }, "자동 갱신이 일시정지되었습니다."],
  ];
  for (const [fixture, message] of scenarios) {
    snapshot = fixture;
    await page.goto("/");
    await expect(page.getByText(message, { exact: true })).toBeVisible();
  }
  await expect(
    page.getByRole("button", { name: "새로고침 (일시정지)", exact: true }),
  ).toBeDisabled();
});
