import { expect, test } from "@playwright/test";
import type { Board } from "../src/types";

let baseline: Board;
test.beforeEach(async ({ request }) => {
  baseline = await (await request.get("/api/v1/dashboard")).json();
});
test.afterEach(async ({ request }) => {
  const latest: Board = await (await request.get("/api/v1/dashboard")).json();
  const response = await request.put("/api/v1/dashboard", {
    headers: { Origin: "http://127.0.0.1:8091" },
    data: { ...baseline, revision: latest.revision },
  });
  expect(response.ok()).toBe(true);
});

test("cycle 01: dashboard rename persists across reload", async ({ page }) => {
  await page.goto("/");
  await page.getByRole("button", { name: "편집", exact: true }).click();
  await page.getByLabel("대시보드 이름").fill("나의 아침 공간");
  await page.getByRole("button", { name: "대시보드 저장" }).click();
  await expect(page.locator('.banner[role="status"]')).toContainText(
    "저장했습니다",
  );
  await page.reload();
  await expect(page.locator(".breadcrumbs")).toContainText("나의 아침 공간");
});

test("cycle 02: cancel and home navigation preserve or explicitly discard edits", async ({
  page,
}) => {
  await page.goto("/");
  await page.getByRole("button", { name: "편집", exact: true }).click();
  await page.getByLabel("대시보드 이름").fill("버리지 않을 이름");
  await expect(
    page.getByText("저장하지 않은 변경 사항", { exact: true }),
  ).toBeVisible();
  page.once("dialog", (dialog) => dialog.dismiss());
  await page.getByRole("button", { name: "취소", exact: true }).click();
  await expect(page.getByLabel("대시보드 이름")).toHaveValue(
    "버리지 않을 이름",
  );
  page.once("dialog", (dialog) => dialog.dismiss());
  await page.getByRole("link", { name: "Glance 홈" }).click();
  await expect(page.getByLabel("대시보드 이름")).toHaveValue(
    "버리지 않을 이름",
  );
  page.once("dialog", (dialog) => dialog.accept());
  await page.getByRole("button", { name: "취소", exact: true }).click();
  await expect(page.getByLabel("대시보드 이름")).toHaveCount(0);
  await expect(page.locator(".breadcrumbs")).toContainText(baseline.title);
});

test("cycle 03: adding under filters reveals and focuses the new widget", async ({
  page,
}) => {
  await page.goto("/");
  await page
    .getByRole("button", { name: "나의 일상", exact: true })
    .first()
    .click();
  await page.getByLabel("위젯 검색").fill("서울");
  await expect(page.locator("article.widget-card")).toHaveCount(1);
  await page.getByRole("button", { name: "위젯 추가", exact: true }).click();
  await page.getByLabel("위젯 종류").selectOption("note");
  await page.getByLabel("제목", { exact: true }).fill("방금 만든 메모");
  await page
    .getByRole("dialog")
    .getByRole("button", { name: "위젯 추가", exact: true })
    .click();
  await expect(page.getByLabel("위젯 검색")).toHaveValue("");
  await expect(page.locator("article.widget-card")).toHaveCount(6);
  await expect(
    page
      .locator("article")
      .filter({ has: page.getByRole("heading", { name: "방금 만든 메모" }) }),
  ).toBeFocused();
});

test("cycle 04: invalid widget settings stay in the editor until corrected", async ({
  page,
}) => {
  await page.goto("/");
  await page.getByRole("button", { name: "위젯 추가", exact: true }).click();
  await page.getByLabel("제목", { exact: true }).fill("   ");
  await page.getByLabel("RSS / Atom 주소").fill("https://example.com/feed");
  await page
    .getByRole("dialog")
    .getByRole("button", { name: "위젯 추가", exact: true })
    .click();
  await expect(page.getByRole("alert")).toHaveText("위젯 제목을 입력하세요.");
  await page.getByLabel("제목", { exact: true }).fill("검증한 피드");
  await page
    .getByLabel("RSS / Atom 주소")
    .fill("https://user:password@example.com/feed");
  await page
    .getByRole("dialog")
    .getByRole("button", { name: "위젯 추가", exact: true })
    .click();
  await expect(page.getByRole("alert")).toContainText("인증 정보가 없는 HTTPS");
  await expect(page.locator("article.widget-card")).toHaveCount(5);
  await page
    .getByLabel("RSS / Atom 주소")
    .fill("https://github.com/glanceapp/glance/releases.atom");
  await page
    .getByRole("dialog")
    .getByRole("button", { name: "위젯 추가", exact: true })
    .click();
  await expect(page.getByRole("dialog")).toHaveCount(0);
  await expect(
    page.getByRole("heading", { name: "검증한 피드" }),
  ).toBeVisible();
});

test("cycle 05: a broken widget is isolated and can recover through editing", async ({
  page,
}) => {
  await page.route("**/api/v1/dashboard", async (route) => {
    if (route.request().method() !== "GET") return route.continue();
    const response = await route.fetch();
    const board: Board = await response.json();
    board.widgets.find((widget) => widget.type === "clock")!.config.timezone =
      "Invalid/Timezone";
    await route.fulfill({ json: board });
  });
  await page.goto("/");
  await expect(page.getByRole("alert")).toContainText(
    "이 위젯을 표시할 수 없습니다.",
  );
  await expect(page.getByRole("heading", { name: "이번 달" })).toBeVisible();
  await expect(
    page.getByRole("heading", { name: "오늘의 메모" }),
  ).toBeVisible();
  await page.getByRole("button", { name: "편집", exact: true }).click();
  await page.getByRole("button", { name: "지금, 서울 설정" }).click();
  await page.getByLabel("시간대").fill("Asia/Seoul");
  await page.getByRole("button", { name: "설정 적용" }).click();
  await expect(page.getByRole("alert")).toHaveCount(0);
  await expect(page.locator(".clock-time")).toBeVisible();
});

test("cycle 06: polling recovers on reconnect and resumes when a tab returns", async ({
  page,
}) => {
  let healthy = false,
    calls = 0;
  await page.route("**/api/v1/snapshots", (route) => {
    calls++;
    return healthy
      ? route.fulfill({ json: {} })
      : route.fulfill({ status: 503, json: { error: "offline" } });
  });
  await page.goto("/");
  await expect(page.locator(".connection-banner")).toBeVisible();
  await expect(page.locator(".clock-time")).toBeVisible();
  healthy = true;
  await page.evaluate(() => window.dispatchEvent(new Event("online")));
  await expect(page.locator(".connection-banner")).toHaveCount(0);
  const beforeResume = calls;
  await page.evaluate(() => {
    Object.defineProperty(document, "hidden", {
      configurable: true,
      value: true,
    });
    document.dispatchEvent(new Event("visibilitychange"));
  });
  await page.evaluate(() => {
    Object.defineProperty(document, "hidden", {
      configurable: true,
      value: false,
    });
    document.dispatchEvent(new Event("visibilitychange"));
  });
  await expect.poll(() => calls).toBeGreaterThan(beforeResume);
});

test("cycle 07: exported JSON contains the complete saved configuration", async ({
  page,
}) => {
  await page.goto("/");
  await page.getByRole("button", { name: "편집", exact: true }).click();
  await page.getByLabel("대시보드 이름").fill("아직 저장하지 않은 이름");
  const pending = page.waitForEvent("download");
  await page.getByRole("button", { name: "설정 내보내기" }).click();
  const download = await pending;
  expect(download.suggestedFilename()).toBe("glance-dashboard.json");
  const stream = await download.createReadStream();
  const chunks: Buffer[] = [];
  for await (const chunk of stream!) chunks.push(Buffer.from(chunk));
  const file = JSON.parse(Buffer.concat(chunks).toString());
  expect(file.format).toBe("glance-dashboard");
  expect(file.version).toBe(1);
  expect(file.board).toEqual(baseline);
  await expect(page.locator('.banner[role="status"]')).toContainText(
    "현재 편집은 파일에 포함되지 않습니다",
  );
});

test("cycle 08: import validates before preview and commits only on save", async ({
  page,
  request,
}) => {
  await page.goto("/");
  const imported = structuredClone(baseline);
  imported.title = "가져온 공간";
  imported.widgets.find((widget) => widget.type === "note")!.config.body =
    "백업에서 가져온 메모";
  await page
    .getByLabel("설정 JSON 파일")
    .setInputFiles({
      name: "backup.json",
      mimeType: "application/json",
      buffer: Buffer.from(
        JSON.stringify({
          format: "glance-dashboard",
          version: 1,
          board: imported,
        }),
      ),
    });
  await expect(page.getByLabel("대시보드 이름")).toHaveValue("가져온 공간");
  await expect(
    page.getByText("백업에서 가져온 메모", { exact: true }),
  ).toBeVisible();
  expect(await (await request.get("/api/v1/dashboard")).json()).toEqual(
    baseline,
  );
  await page.getByRole("button", { name: "대시보드 저장" }).click();
  await expect(page.locator('.banner[role="status"]')).toContainText(
    "저장했습니다",
  );
  await page.reload();
  await expect(page.locator(".breadcrumbs")).toContainText("가져온 공간");
  const bad = structuredClone(imported) as any;
  bad.widgets[0].type = "unknown-widget";
  await page
    .getByLabel("설정 JSON 파일")
    .setInputFiles({
      name: "invalid.json",
      mimeType: "application/json",
      buffer: Buffer.from(
        JSON.stringify({ format: "glance-dashboard", version: 1, board: bad }),
      ),
    });
  await expect(page.getByRole("alert")).toContainText("지원하지 않는 위젯");
  expect((await (await request.get("/api/v1/dashboard")).json()).title).toBe(
    "가져온 공간",
  );
  await page
    .getByLabel("설정 JSON 파일")
    .setInputFiles({
      name: "broken.json",
      mimeType: "application/json",
      buffer: Buffer.from("{"),
    });
  await expect(page.getByRole("alert")).toContainText("올바른 JSON 설정 파일");
});

test("cycle 09: conflict keeps local edits, supports backup and confirmed reload", async ({
  page,
  request,
}) => {
  await page.goto("/");
  await page.getByRole("button", { name: "편집", exact: true }).click();
  await page.getByLabel("대시보드 이름").fill("내가 편집한 이름");
  const other = await request.put("/api/v1/dashboard", {
    headers: { Origin: "http://127.0.0.1:8091" },
    data: { ...baseline, title: "다른 창에서 저장한 이름" },
  });
  expect(other.ok()).toBe(true);
  await page.getByRole("button", { name: "대시보드 저장" }).click();
  await expect(page.getByRole("alert")).toContainText(
    "다른 창에서 대시보드가 변경되었습니다",
  );
  await expect(page.getByLabel("대시보드 이름")).toHaveValue(
    "내가 편집한 이름",
  );
  const downloading = page.waitForEvent("download");
  await page.getByRole("button", { name: "내 편집 백업" }).click();
  const stream = await (await downloading).createReadStream();
  const chunks: Buffer[] = [];
  for await (const chunk of stream!) chunks.push(Buffer.from(chunk));
  expect(JSON.parse(Buffer.concat(chunks).toString()).board.title).toBe(
    "내가 편집한 이름",
  );
  page.once("dialog", (dialog) => dialog.dismiss());
  await page.getByRole("button", { name: "최신 상태 불러오기" }).click();
  await expect(page.getByLabel("대시보드 이름")).toHaveValue(
    "내가 편집한 이름",
  );
  page.once("dialog", (dialog) => dialog.accept());
  await page.getByRole("button", { name: "최신 상태 불러오기" }).click();
  await expect(page.locator(".breadcrumbs")).toContainText(
    "다른 창에서 저장한 이름",
  );
  await expect(page.getByLabel("대시보드 이름")).toHaveCount(0);
});

test("cycle 10: cloned widgets persist with new IDs and independently editable sources", async ({
  page,
  request,
}) => {
  const original = baseline.widgets.find((widget) => widget.type === "rss")!;
  const copyTitle = `${original.title} 복사`;
  await page.goto("/");
  await page.getByRole("button", { name: "편집", exact: true }).click();
  await page
    .getByRole("button", { name: `${original.title} 복제`, exact: true })
    .click();
  await expect(
    page.getByRole("heading", { name: copyTitle, exact: true }),
  ).toBeVisible();
  await page.getByRole("button", { name: "대시보드 저장" }).click();
  await expect(page.locator('.banner[role="status"]')).toContainText(
    "저장했습니다",
  );
  const duplicated: Board = await (
    await request.get("/api/v1/dashboard")
  ).json();
  const clone = duplicated.widgets.find(
    (widget) => widget.title === copyTitle,
  )!;
  expect(clone.id).not.toBe(original.id);
  expect(clone.sourceId).toBe(original.sourceId);
  expect(duplicated.sources).toEqual(baseline.sources);
  await page.reload();
  await expect(
    page.getByRole("heading", { name: copyTitle, exact: true }),
  ).toBeVisible();
  await page.getByRole("button", { name: "편집", exact: true }).click();
  await page.getByRole("button", { name: `${copyTitle} 설정` }).click();
  await page
    .getByLabel("RSS / Atom 주소")
    .fill("https://github.com/gethomepage/homepage/releases.atom");
  await page.getByRole("button", { name: "설정 적용" }).click();
  await page.getByRole("button", { name: "대시보드 저장" }).click();
  await expect(page.locator('.banner[role="status"]')).toContainText(
    "저장했습니다",
  );
  const changed: Board = await (await request.get("/api/v1/dashboard")).json();
  expect(
    changed.widgets.find((widget) => widget.id === original.id)!.sourceId,
  ).toBe(original.sourceId);
  const changedClone = changed.widgets.find(
    (widget) => widget.id === clone.id,
  )!;
  expect(changedClone.sourceId).not.toBe(original.sourceId);
  expect(
    changed.sources.find((source) => source.id === original.sourceId),
  ).toEqual(baseline.sources.find((source) => source.id === original.sourceId));
  expect(
    changed.sources.find((source) => source.id === changedClone.sourceId)!
      .config.url,
  ).toBe("https://github.com/gethomepage/homepage/releases.atom");
});
