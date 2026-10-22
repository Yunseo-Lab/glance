import { expect, test } from "@playwright/test";
import type { Board } from "../src/types";
let board: Board;
const origin = "http://127.0.0.1:8091";
test.beforeEach(async ({ request }) => {
  board = await (
    await request.post("/api/v1/boards", {
      headers: { Origin: origin },
      data: { title: "순서 검증" },
    })
  ).json();
  const response = await request.put(`/api/v1/boards/${board.id}`, {
    headers: { Origin: origin },
    data: {
      ...board,
      widgets: ["하나", "둘", "셋"].map((title, index) => ({
        id: `layout-${index}`,
        type: "note",
        title,
        size: "small",
        config: { body: title },
      })),
    },
  });
  expect(response.status(), await response.text()).toBe(200);
  board = await response.json();
});
test.afterEach(async ({ request }) => {
  const latest = await (await request.get(`/api/v1/boards/${board.id}`)).json();
  await request.delete(`/api/v1/boards/${board.id}`, {
    headers: { Origin: origin },
    data: { revision: latest.revision },
  });
});
test("keyboard sorting preserves DOM order, persisted IDs and cancellation", async ({
  page,
  request,
}) => {
  await page.setViewportSize({ width: 360, height: 1000 });
  await page.goto(`/#/boards/${board.id}`);
  await page.getByRole("button", { name: "편집", exact: true }).click();
  const handle = page.getByRole("button", { name: "하나 순서 바꾸기" });
  await handle.focus();
  await page.keyboard.press("Space");
  await page.waitForTimeout(150);
  await page.keyboard.press("ArrowDown");
  await page.waitForTimeout(150);
  await page.keyboard.press("Space");
  await expect(page.locator("article.widget-card").first()).toContainText("둘");
  expect(
    await page
      .locator("article.widget-card")
      .evaluateAll((nodes) =>
        nodes.map((node) => node.getAttribute("data-widget-id")),
      ),
  ).toEqual(["layout-1", "layout-0", "layout-2"]);
  await handle.focus();
  await page.keyboard.press("Space");
  await page.waitForTimeout(100);
  await page.keyboard.press("ArrowDown");
  await page.keyboard.press("Escape");
  await expect(page.locator("article.widget-card").first()).toContainText("둘");
  await page
    .getByRole("button", { name: "대시보드 저장", exact: true })
    .click();
  await expect(
    page.getByText("변경 사항을 저장했습니다.", { exact: true }),
  ).toBeVisible();
  await page.reload();
  expect(
    (
      await (await request.get(`/api/v1/boards/${board.id}`)).json()
    ).widgets.map((widget: { id: string }) => widget.id),
  ).toEqual(["layout-1", "layout-0", "layout-2"]);
  await expect(page.locator("article.widget-card").first()).toContainText("둘");
});
test("mobile touch handle reorders without making the entire card unscrollable", async ({
  browser,
  request,
}) => {
  const context = await browser.newContext({
    viewport: { width: 360, height: 1100 },
    hasTouch: true,
  });
  const page = await context.newPage();
  try {
    await page.goto(origin + `/#/boards/${board.id}`);
    await page.getByRole("button", { name: "편집", exact: true }).click();
    const source = await page
        .getByRole("button", { name: "하나 순서 바꾸기" })
        .boundingBox(),
      target = await page
        .getByRole("button", { name: "둘 순서 바꾸기" })
        .boundingBox();
    expect(source).toBeTruthy();
    expect(target).toBeTruthy();
    const cdp = await context.newCDPSession(page);
    await cdp.send("Input.dispatchTouchEvent", {
      type: "touchStart",
      touchPoints: [{ x: source!.x + 8, y: source!.y + 8 }],
    });
    await page.waitForTimeout(220);
    for (let index = 1; index <= 10; index++) {
      await cdp.send("Input.dispatchTouchEvent", {
        type: "touchMove",
        touchPoints: [
          {
            x: source!.x + 8,
            y: source!.y + 8 + ((target!.y - source!.y) * index) / 10,
          },
        ],
      });
      await page.waitForTimeout(25);
    }
    await cdp.send("Input.dispatchTouchEvent", {
      type: "touchEnd",
      touchPoints: [],
    });
    await expect(page.locator("article.widget-card").first()).toContainText(
      "둘",
    );
    await page
      .getByRole("button", { name: "대시보드 저장", exact: true })
      .click();
    await expect(
      page.getByText("변경 사항을 저장했습니다.", { exact: true }),
    ).toBeVisible();
    expect(
      (await (await request.get(`/api/v1/boards/${board.id}`)).json())
        .widgets[0].id,
    ).toBe("layout-1");
    expect(
      await page
        .locator("article.widget-card")
        .first()
        .evaluate((node) => getComputedStyle(node).touchAction),
    ).not.toBe("none");
  } finally {
    await context.close();
  }
});
test("delete undo and redo preserve IDs and save starts a new history", async ({
  page,
}) => {
  await page.goto(`/#/boards/${board.id}`);
  await page.getByRole("button", { name: "편집", exact: true }).click();
  await expect(
    page.getByRole("button", { name: "편집 되돌리기" }),
  ).toBeDisabled();
  await page.getByRole("button", { name: "하나 삭제" }).click();
  await expect(page.locator("article.widget-card")).toHaveCount(2);
  await page.getByRole("button", { name: "편집 되돌리기" }).click();
  await expect(page.locator("article.widget-card")).toHaveCount(3);
  await expect(page.locator("article.widget-card").first()).toHaveAttribute(
    "data-widget-id",
    "layout-0",
  );
  await page.getByRole("button", { name: "편집 다시 적용" }).click();
  await expect(page.locator("article.widget-card")).toHaveCount(2);
  await page.getByRole("button", { name: "편집 되돌리기" }).focus();
  await page.keyboard.press("Control+z");
  await expect(page.locator("article.widget-card")).toHaveCount(3);
  await page
    .getByRole("button", { name: "대시보드 저장", exact: true })
    .click();
  await page.getByRole("button", { name: "편집", exact: true }).click();
  await expect(
    page.getByRole("button", { name: "편집 되돌리기" }),
  ).toBeDisabled();
});
