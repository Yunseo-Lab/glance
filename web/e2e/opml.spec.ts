import { expect, test } from "@playwright/test";
import { appendFeeds } from "../src/editor/feedImport";
import type { Board } from "../src/types";
test("feed import is atomic, deduplicated and bounded before changing a draft", () => {
  const board: Board = {
    id: "home",
    revision: 1,
    title: "Base",
    theme: "light",
    sources: [],
    widgets: [],
  };
  const feed = { title: "Feed", url: "https://example.com/feed" };
  let id = 0;
  const next = appendFeeds(board, [feed, feed], () => `id-${id++}`);
  expect(next.widgets).toHaveLength(1);
  expect(next.sources).toHaveLength(1);
  expect(board.widgets).toHaveLength(0);
  expect(appendFeeds(next, [feed]).widgets).toHaveLength(1);
  expect(() =>
    appendFeeds(
      board,
      Array.from({ length: 33 }, (_, index) => ({
        ...feed,
        url: `https://example.com/feed/${index}`,
      })),
    ),
  ).toThrow("32개");
  expect(board.sources).toHaveLength(0);
});
test("OPML preview makes no board changes and selected feeds persist only after save", async ({
  page,
  request,
}) => {
  const headers = { Origin: "http://127.0.0.1:8091" };
  const board: Board = await (
    await request.post("/api/v1/boards", {
      headers,
      data: { title: "OPML 검증" },
    })
  ).json();
  try {
    await page.goto(`/#/sources?board=${board.id}`);
    await page
      .getByRole("button", { name: "피드 묶음 가져오기", exact: true })
      .click();
    const document =
      '<opml version="2.0"><body><outline text="피드 하나" xmlUrl="https://example.com/a.xml"/><outline text="피드 둘" xmlUrl="https://example.com/b.xml"/></body></opml>';
    await page
      .getByLabel("OPML 파일", { exact: true })
      .setInputFiles({
        name: "feeds.opml",
        mimeType: "application/xml",
        buffer: Buffer.from(document),
      });
    await expect(
      page.getByText("피드 2개 · 중복 0개 제외", { exact: true }),
    ).toBeVisible();
    expect(
      (await (await request.get(`/api/v1/boards/${board.id}`)).json()).widgets,
    ).toHaveLength(0);
    await page.getByRole("checkbox", { name: /피드 둘/ }).uncheck();
    await page
      .getByRole("button", { name: "선택한 1개 초안에 추가", exact: true })
      .click();
    await expect(page.locator("article.widget-card")).toHaveCount(1);
    expect(
      (await (await request.get(`/api/v1/boards/${board.id}`)).json()).sources,
    ).toHaveLength(0);
    await page
      .getByRole("button", { name: "대시보드 저장", exact: true })
      .click();
    await expect(
      page.getByText("변경 사항을 저장했습니다.", { exact: true }),
    ).toBeVisible();
    const saved: Board = await (
      await request.get(`/api/v1/boards/${board.id}`)
    ).json();
    expect(saved.sources[0].config.url).toBe("https://example.com/a.xml");
  } finally {
    const latest: Board = await (
      await request.get(`/api/v1/boards/${board.id}`)
    ).json();
    await request.delete(`/api/v1/boards/${board.id}`, {
      headers,
      data: { revision: latest.revision },
    });
  }
});
