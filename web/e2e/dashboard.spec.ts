import { expect, test } from "@playwright/test";
import type { Board } from "../src/types";

let baseline: Board;
test.beforeEach(async ({ request }) => {
  baseline = await (await request.get("/api/v1/dashboard")).json();
});
test.afterEach(async ({ request }) => {
  const latest: Board = await (await request.get("/api/v1/dashboard")).json();
  const result = await request.put("/api/v1/dashboard", {
    headers: { Origin: "http://127.0.0.1:8091" },
    data: { ...baseline, revision: latest.revision },
  });
  expect(result.ok()).toBe(true);
});

test("edit, reorder, save, reload, cancel, and delete a widget", async ({
  page,
}) => {
  const errors: string[] = [];
  page.on("pageerror", (error) => errors.push(error.message));
  await page.goto("/");
  await expect(
    page.getByRole("heading", { name: "오늘도, 한눈에." }),
  ).toBeVisible();
  await expect(page.locator("article.widget-card")).toHaveCount(5);
  await page.getByRole("button", { name: "위젯 추가", exact: true }).click();
  await page.getByLabel("위젯 종류").selectOption("note");
  await page.getByLabel("제목", { exact: true }).fill("나의 테스트 메모");
  await page
    .getByLabel("내용", { exact: true })
    .fill("새로고침 후에도 남는 메모");
  await page
    .getByRole("button", { name: "위젯 추가", exact: true })
    .last()
    .click();
  await expect(page.getByRole("dialog")).not.toBeVisible();
  await page
    .getByRole("button", { name: "나의 테스트 메모 앞으로 이동" })
    .click();
  await expect(page.locator("article.widget-card").nth(4)).toContainText(
    "나의 테스트 메모",
  );
  await page.getByRole("button", { name: "대시보드 저장" }).click();
  await expect(page.locator('.banner[role="status"]')).toContainText(
    "변경 사항을 저장했습니다.",
  );
  await page.reload();
  await expect(page.locator("article.widget-card").nth(4)).toContainText(
    "새로고침 후에도 남는 메모",
  );
  await page.getByRole("button", { name: "편집", exact: true }).click();
  await page.getByRole("button", { name: "나의 테스트 메모 설정" }).click();
  await expect(page.getByRole("dialog")).toBeVisible();
  await page.getByLabel("내용", { exact: true }).fill("취소할 변경");
  await page.getByRole("button", { name: "설정 적용" }).click();
  page.once("dialog", (dialog) => dialog.accept());
  await page.getByRole("button", { name: "취소", exact: true }).click();
  await expect(
    page.getByText("새로고침 후에도 남는 메모", { exact: true }),
  ).toBeVisible();
  await page.getByRole("button", { name: "편집", exact: true }).click();
  await page.getByRole("button", { name: "나의 테스트 메모 삭제" }).click();
  await page.getByRole("button", { name: "대시보드 저장" }).click();
  await expect(
    page.getByRole("heading", { name: "나의 테스트 메모" }),
  ).toHaveCount(0);
  await page.reload();
  await expect(page.locator("article.widget-card")).toHaveCount(5);
  expect(errors).toEqual([]);
});

test("theme persists and search and categories filter widgets", async ({
  page,
}) => {
  await page.goto("/");
  await page.getByRole("button", { name: "어두운 테마" }).click();
  await expect(page.locator("html")).toHaveAttribute("data-theme", "dark");
  await expect(page.locator('.banner[role="status"]')).toContainText(
    "변경 사항을 저장했습니다.",
  );
  await page.reload();
  await expect(page.locator("html")).toHaveAttribute("data-theme", "dark");
  await page.getByRole("button", { name: "밝은 테마" }).click();
  await expect(page.locator('.banner[role="status"]')).toContainText(
    "변경 사항을 저장했습니다.",
  );
  await page.getByLabel("위젯 검색").fill("서울");
  await expect(page.locator("article.widget-card")).toHaveCount(1);
  await page.getByRole("button", { name: "검색 지우기" }).click();
  await page
    .getByRole("button", { name: "관심 소식", exact: true })
    .first()
    .click();
  await expect(page.locator("article.widget-card")).toHaveCount(1);
  await expect(
    page.getByRole("heading", { name: "Glance 업데이트" }),
  ).toBeVisible();
});

test("mobile layout, dialog keyboard interaction, and independent failure state", async ({
  page,
}) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await page.route("**/api/v1/snapshots", (route) =>
    route.fulfill({ json: {} }),
  );
  await page.goto("/");
  await expect(page.getByRole("heading", { name: "지금, 서울" })).toBeVisible();
  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= innerWidth,
    ),
  ).toBe(true);
  await page.getByRole("button", { name: "위젯 추가", exact: true }).click();
  await expect(page.getByRole("dialog")).toBeVisible();
  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= innerWidth,
    ),
  ).toBe(true);
  await page.keyboard.press("Escape");
  await expect(page.getByRole("dialog")).not.toBeVisible();
  // A source still loading does not prevent local widgets or the editor working.
  await expect(page.getByText("소식을 가져오고 있어요")).toBeVisible();
  await expect(page.getByRole("heading", { name: "이번 달" })).toBeVisible();
});

test("stale browser revisions are rejected without losing local edits", async ({
  page,
  context,
}) => {
  await page.goto("/");
  await expect(page.locator("article.widget-card")).toHaveCount(5);
  await page.getByRole("button", { name: "편집", exact: true }).click();
  const other = await context.newPage();
  await other.goto("/");
  await other.getByRole("button", { name: "어두운 테마" }).click();
  await expect(other.locator('.banner[role="status"]')).toContainText(
    "변경 사항을 저장했습니다.",
  );
  await page.getByRole("button", { name: "대시보드 저장" }).click();
  await expect(page.getByRole("alert")).toContainText(
    "다른 창에서 대시보드가 변경되었습니다.",
  );
  await expect(
    page.getByRole("button", { name: "대시보드 저장" }),
  ).toBeVisible();
  await other.close();
});
