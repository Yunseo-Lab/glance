import { expect, test } from "@playwright/test";

test("installed app shell remains offline without caching private API responses", async ({
  page,
  context,
}) => {
  await page.goto("/");
  await page.evaluate(() => navigator.serviceWorker.ready);
  await expect
    .poll(() => page.evaluate(() => !!navigator.serviceWorker.controller))
    .toBe(true);
  const manifest = await (
    await page.request.get("/manifest.webmanifest")
  ).json();
  expect(manifest.display).toBe("standalone");
  expect(manifest.icons).toHaveLength(3);
  await page.evaluate(async () => {
    await fetch("/api/v1/dashboard");
    await fetch("/api/v1/entries");
  });
  const cached = await page.evaluate(async () => {
    const result: string[] = [];
    for (const key of await caches.keys())
      for (const request of await (await caches.open(key)).keys())
        result.push(new URL(request.url).pathname);
    return result;
  });
  expect(cached).toContain("/");
  expect(cached.some((path) => path.startsWith("/assets/"))).toBe(true);
  expect(cached.some((path) => path.startsWith("/api/"))).toBe(false);
  await context.setOffline(true);
  await expect(
    page.getByText("오프라인입니다.", { exact: false }),
  ).toBeVisible();
  // A new navigation can load the public shell; private data must still fail.
  await page.reload();
  await expect(page.getByRole("button", {name:"다시 연결",exact:true})).toBeVisible();
  expect(
    await page.evaluate(async () => {
      try {
        await fetch("/api/v1/dashboard");
        return true;
      } catch {
        return false;
      }
    }),
  ).toBe(false);
  await context.setOffline(false);
  await page.reload();
  await expect(
    page.getByRole("heading", { name: "오늘도, 한눈에." }),
  ).toBeVisible();
});

test("a waiting service worker requires explicit update and protects an open form", async ({
  page,
}) => {
  await page.goto("/");
  await page.evaluate(() => navigator.serviceWorker.ready);
  await expect
    .poll(() => page.evaluate(() => !!navigator.serviceWorker.controller))
    .toBe(true);
  await page.getByRole("button", { name: "위젯 추가", exact: true }).click();
  await page
    .getByLabel("제목", { exact: true })
    .fill("업데이트 전에 남겨둔 입력");
  // Different script URL exercises the browser's actual waiting-worker lifecycle.
  await page.evaluate(() =>
    navigator.serviceWorker.register("/sw.js?release-rehearsal=2", {
      scope: "/",
    }),
  );
  await expect(
    page.getByRole("button", { name: "앱 업데이트", exact: true }),
  ).toBeDisabled();
  await expect(page.getByLabel("제목", { exact: true })).toHaveValue(
    "업데이트 전에 남겨둔 입력",
  );
  await page.getByRole("button", { name: "닫기", exact: true }).click();
  await expect(
    page.getByRole("button", { name: "앱 업데이트", exact: true }),
  ).toBeEnabled();
  await page.getByRole("button", { name: "앱 업데이트", exact: true }).click();
  await expect(
    page.getByRole("heading", { name: "오늘도, 한눈에." }),
  ).toBeVisible();
  await expect(
    page.getByRole("button", { name: "앱 업데이트", exact: true }),
  ).toHaveCount(0);
});
