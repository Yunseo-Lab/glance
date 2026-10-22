import { expect, test } from "@playwright/test";
import AxeBuilder from "@axe-core/playwright";
import { writeFileSync } from "node:fs";
async function audit(page: import("@playwright/test").Page, name: string) {
  const results = await new AxeBuilder({ page })
    .withTags(["wcag2a", "wcag2aa", "wcag21a", "wcag21aa"])
    .analyze();
  writeFileSync(
    test.info().outputPath(`axe-${name}.json`),
    JSON.stringify(results, null, 2),
  );
  const issues = results.violations
    .filter(
      (issue) => issue.impact === "serious" || issue.impact === "critical",
    )
    .map((issue) => ({
      id: issue.id,
      impact: issue.impact,
      description: issue.description,
      nodes: issue.nodes.map((node) => ({
        target: node.target,
        summary: node.failureSummary,
      })),
    }));
  expect(issues, JSON.stringify(issues, null, 2)).toEqual([]);
}
let baseline: unknown;
test.beforeEach(async ({ request }) => {
  baseline = await (await request.get("/api/v1/dashboard")).json();
});
test.afterEach(async ({ request }) => {
  const current = await (await request.get("/api/v1/dashboard")).json();
  await request.put("/api/v1/dashboard", {
    headers: { Origin: "http://127.0.0.1:8091" },
    data: { ...(baseline as object), revision: current.revision },
  });
});
test("workspace, dialogs and login have no serious or critical axe findings", async ({
  page,
}) => {
  await page.goto("/");
  await audit(page, "board");
  await page
    .getByRole("button", { name: "위젯 추가", exact: true })
    .first()
    .click();
  await audit(page, "widget-editor");
  await page.getByRole("button", { name: "닫기", exact: true }).click();
  await page.getByRole("button", { name: "빠른 기록", exact: true }).click();
  await audit(page, "capture");
  await page
    .getByRole("button", { name: "빠른 기록 닫기", exact: true })
    .click();
  await page.getByRole("button", { name: "보드 관리", exact: true }).click();
  await audit(page, "boards");
  await page
    .getByRole("button", { name: "보드 관리 닫기", exact: true })
    .click();
  for (const route of ["sources", "inbox", "brief", "settings"]) {
    await page.goto("/#/" + route);
    await expect(
      page.locator("main").getByRole("heading", { level: 1 }),
    ).toBeVisible();
    await audit(page, route);
    if (route === "sources") {
      await page
        .getByRole("button", { name: "연결 추가", exact: true })
        .click();
      await audit(page, "connection-dialog");
      await page
        .getByRole("button", { name: "연결 창 닫기", exact: true })
        .click();
    }
  }
  await page.goto("/");
  await page.getByRole("button", { name: "어두운 테마", exact: true }).click();
  await audit(page, "dark-board");
  await page.getByRole("button", { name: "밝은 테마", exact: true }).click();
  await page.route("**/api/v1/auth/session", (route) =>
    route.fulfill({ json: { enabled: true, authenticated: false } }),
  );
  await page.reload();
  await expect(
    page.getByRole("heading", { name: "나의 공간에 로그인" }),
  ).toBeVisible();
  await audit(page, "login");
});
test("keyboard focus, accessibility tree and reduced motion preserve screen-reader controls", async ({
  page,
}) => {
  await page.emulateMedia({ reducedMotion: "reduce" });
  await page.goto("/");
  await expect(
    page.getByRole("heading", { name: "오늘도, 한눈에." }),
  ).toBeVisible();
  await page.keyboard.press("Tab");
  await expect(
    page.getByRole("link", { name: "본문으로 이동", exact: true }),
  ).toBeFocused();
  await page.keyboard.press("Enter");
  await expect(page.locator("#main")).toBeFocused();
  await page
    .getByRole("button", { name: "명령 팔레트 열기", exact: true })
    .click();
  await expect(
    page.getByRole("combobox", { name: "명령 검색", exact: true }),
  ).toBeFocused();
  await page.keyboard.press("Escape");
  await expect(
    page.getByRole("button", { name: "명령 팔레트 열기", exact: true }),
  ).toBeFocused();
  await page
    .getByRole("button", { name: "위젯 추가", exact: true })
    .first()
    .click();
  const dialog = page.getByRole("dialog");
  await expect(dialog).toBeVisible();
  const session = await page.context().newCDPSession(page);
  const tree = await session.send("Accessibility.getFullAXTree");
  const named = tree.nodes
    .filter((node) => !node.ignored && node.name?.value)
    .map((node) => ({ role: node.role?.value, name: node.name?.value }));
  expect(
    named.some(
      (node) => node.role === "dialog" && node.name === "새 위젯 추가",
    ),
  ).toBe(true);
  expect(
    named.some((node) => node.role === "combobox" && node.name === "위젯 종류"),
  ).toBe(true);
  expect(
    named.some((node) => node.role === "textbox" && node.name === "제목"),
  ).toBe(true);
  writeFileSync(
    test.info().outputPath("screen-reader-accessibility-tree.json"),
    JSON.stringify(named, null, 2),
  );
  await page.keyboard.press("Escape");
  await expect(dialog).toHaveCount(0);
  expect(
    await page.evaluate(
      () => getComputedStyle(document.documentElement).scrollBehavior,
    ),
  ).not.toBe("smooth");
  await page.setViewportSize({ width: 360, height: 800 });
  await audit(page, "mobile-board");
  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= innerWidth,
    ),
  ).toBe(true);
});
