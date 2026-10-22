import { test, expect } from "@playwright/test";
import { readFileSync } from "node:fs";
const fixtureFont = readFileSync(
  new URL("./fonts/NanumGothic-Regular.ttf", import.meta.url),
);
for (const width of [360, 768, 1440])
  for (const theme of ["light", "dark"]) {
    test(`fixed ${width}px ${theme} board covers long Korean text and provider errors`, async ({
      page,
      request,
    }) => {
      await page.setViewportSize({ width, height: 900 });
      await page.clock.install({ time: new Date("2026-10-22T03:45:00Z") });
      const board = {
        id: "home",
        title: "나의 대시보드",
        revision: 1,
        theme,
        widgets: [
          {
            id: "clock",
            type: "clock",
            title: "지금, 서울",
            size: "small",
            config: { timezone: "Asia/Seoul" },
          },
          {
            id: "rss",
            type: "rss",
            title: "소식",
            size: "wide",
            sourceId: "feed",
            config: { limit: 5 },
          },
          {
            id: "calendar",
            type: "calendar",
            title: "이번 달",
            size: "small",
            config: {},
          },
          {
            id: "bookmarks",
            type: "bookmarks",
            title: "자주 가는 곳",
            size: "small",
            config: {
              links: [
                { title: "GitHub", url: "https://github.com" },
                {
                  title: "Glance 문서",
                  url: "https://github.com/glanceapp/glance",
                },
              ],
            },
          },
          {
            id: "note",
            type: "note",
            title: "오늘의 메모",
            size: "small",
            config: { body: "" },
          },
        ],
        sources: [
          {
            id: "feed",
            type: "rss",
            config: {
              url: "https://github.com/glanceapp/glance/releases.atom",
            },
          },
        ],
      };
      await page.route("**/api/v1/boards", (route) =>
        route.fulfill({ json: [board] }),
      );
      board.theme = theme;
      board.widgets.find((widget: any) => widget.type === "note").config.body =
        "## 오늘 집중할 일\n- 읽을거리와 일정을 한눈에 확인하기\n- [x] 중요한 메모 저장하기\n\n길이가 긴 한국어 문장과 **강조된 내용**이 작은 화면에서도 잘 읽히는지 확인합니다.\n\nhttps://example.com/averylongunbrokentokenwithoutspaces12345678901234567890";
      board.widgets.find((widget: any) => widget.type === "rss").title =
        "아주 긴 제목의 읽을거리 모음: 프로젝트 소식과 오늘의 관심사";
      const snapshots = Object.fromEntries(
        board.sources.map((source: any) => [
          source.id,
          {
            state: "error",
            errorCode: "fetch_failed",
            error: "정보 제공자에 연결하지 못했습니다.",
          },
        ]),
      );
      await page.route("**/api/v1/dashboard", (route) =>
        route.fulfill({ json: board }),
      );
      await page.route("**/api/v1/snapshots", (route) =>
        route.fulfill({ json: snapshots }),
      );
      await page.route("**/visual-fixture.ttf", (route) =>
        route.fulfill({ body: fixtureFont, contentType: "font/ttf" }),
      );
      await page.goto("/");
      await page.route("**/visual-fixture.css", (route) =>
        route.fulfill({
          contentType: "text/css",
          body: "@font-face {font-family:VisualFixture;src:url(/visual-fixture.ttf)} :root, body, button, input, select, textarea {font-family:VisualFixture,sans-serif !important}",
        }),
      );
      await page.addStyleTag({ url: "/visual-fixture.css" });
      await page.evaluate(() => document.fonts.load("16px VisualFixture"));
      await expect(page.locator(".widget-card").first()).toBeVisible();
      await expect(
        page.getByText("연결이 지연되고 있습니다.", { exact: true }).first(),
      ).toBeVisible();
      await page.evaluate(() => document.fonts.ready);
      expect(
        await page.evaluate(
          () => document.documentElement.scrollWidth <= innerWidth,
        ),
      ).toBe(true);
      await expect(page).toHaveScreenshot(`board-${width}-${theme}.png`, {
        fullPage: true,
        animations: "disabled",
        caret: "hide",
        maxDiffPixelRatio: 0.001,
      });
    });
  }
