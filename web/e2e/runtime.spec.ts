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
