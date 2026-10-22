import { expect, test } from "@playwright/test";
test("leaving a server panel aborts its in-flight private read", async ({
  page,
}) => {
  let intercepted = false,
    aborted = false;
  page.on("requestfailed", (request) => {
    if (request.url().endsWith("/api/v1/connections")) aborted = true;
  });
  await page.route("**/api/v1/connections", async (route) => {
    intercepted = true;
    await new Promise((resolve) => setTimeout(resolve, 1500));
    await route
      .fulfill({ contentType: "application/json", body: "[]" })
      .catch(() => {});
  });
  await page.goto("/");
  await page.getByRole("button", { name: "데이터 소스", exact: true }).click();
  await expect.poll(() => intercepted).toBe(true);
  await page.getByRole("button", { name: "설정", exact: true }).click();
  await expect.poll(() => aborted).toBe(true);
  await expect(
    page.getByRole("heading", { name: "설정", exact: true }),
  ).toBeVisible();
});
