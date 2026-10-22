import { expect, test } from "@playwright/test";
import { captureToBoard } from "../src/editor/quickCapture";
import type { Board } from "../src/types";
test("capture preserves board IDs, validates bookmarks and never partially mutates an invalid draft", () => {
  const base: Board = {
    id: "home",
    title: "Base",
    revision: 1,
    theme: "light",
    widgets: [],
    sources: [],
  };
  const note = captureToBoard(
    base,
    { kind: "note", title: "메모", body: "**강조**", url: "" },
    () => "note-id",
  );
  expect(base.widgets).toHaveLength(0);
  expect(note.widgets[0].id).toBe("note-id");
  expect(note.id).toBe(base.id);
  const links = captureToBoard(
    note,
    {
      kind: "bookmark",
      title: "Example",
      body: "",
      url: "https://example.com",
    },
    () => "bookmark-id",
  );
  expect(links.widgets[1].config.links?.[0].url).toBe("https://example.com");
  expect(() =>
    captureToBoard(links, {
      kind: "bookmark",
      title: "Duplicate",
      body: "",
      url: "https://example.com/",
    }),
  ).toThrow("이미");
  expect(() =>
    captureToBoard(base, {
      kind: "bookmark",
      title: "Unsafe",
      body: "",
      url: "javascript:alert(1)",
    }),
  ).toThrow();
  expect(() =>
    captureToBoard(base, { kind: "note", title: "", body: " ", url: "" }),
  ).toThrow();
});
test("quick notes save safe Markdown and bookmarks while editing captures remain an unsaved draft", async ({
  page,
  request,
}) => {
  const headers = { Origin: "http://127.0.0.1:8091" };
  const board: Board = await (
    await request.post("/api/v1/boards", {
      headers,
      data: { title: "빠른 기록 검증" },
    })
  ).json();
  try {
    await page.goto(`/#/boards/${board.id}`);
    await page.getByRole("button", { name: "빠른 기록", exact: true }).click();
    await page.getByLabel("기록 제목", { exact: true }).fill("안전한 메모");
    await page
      .getByLabel("메모 내용", { exact: true })
      .fill(
        '# 오늘 할 일\n- [x] 완료\n**강조**와 `코드`\n<img src="https://tracker.example.com/pixel">\n[잘못된 링크](javascript:alert(1))',
      );
    await page.getByRole("button", { name: "기록 저장", exact: true }).click();
    await expect(
      page.getByRole("heading", { name: "오늘 할 일", exact: true }),
    ).toBeVisible();
    const note = page
      .locator("article.widget-card")
      .filter({
        has: page.getByRole("heading", { name: "안전한 메모", exact: true }),
      });
    await expect(note.locator("strong")).toHaveText("강조");
    await expect(note.locator("code")).toHaveText("코드");
    await expect(note.locator("img")).toHaveCount(0);
    await expect(note.locator('a[href^="javascript:"]')).toHaveCount(0);
    await page.reload();
    await expect(note).toBeVisible();
    await page.getByRole("button", { name: "빠른 기록", exact: true }).click();
    await page.getByRole("button", { name: "즐겨찾기", exact: true }).click();
    await page.getByLabel("기록 제목", { exact: true }).fill("Example");
    await page
      .getByLabel("즐겨찾기 주소", { exact: true })
      .fill("https://example.com");
    await page.getByRole("button", { name: "기록 저장", exact: true }).click();
    await expect(page.getByRole("link", { name: /Example/ })).toHaveAttribute(
      "href",
      "https://example.com/",
    );
    await page.getByRole("button", { name: "편집", exact: true }).click();
    await page.getByRole("button", { name: "빠른 기록", exact: true }).click();
    await page
      .getByLabel("기록 제목", { exact: true })
      .fill("미저장 빠른 메모");
    await page
      .getByLabel("메모 내용", { exact: true })
      .fill("초안에만 있어야 합니다.");
    await page
      .getByRole("button", { name: "초안에 추가", exact: true })
      .click();
    await expect(
      page.getByRole("heading", { name: "미저장 빠른 메모", exact: true }),
    ).toBeVisible();
    const saved: Board = await (
      await request.get(`/api/v1/boards/${board.id}`)
    ).json();
    expect(
      saved.widgets.some((widget) => widget.title === "미저장 빠른 메모"),
    ).toBe(false);
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

test('limited Markdown keeps raw markup inert and caps inline rendering work',async()=>{
 const{markdownInline}=await import('../src/workspace/SafeMarkdown');expect(markdownInline('<img src=x onerror=alert(1)>')).toEqual(['<img src=x onerror=alert(1)>']);expect(markdownInline('['.repeat(12000))).toEqual(['['.repeat(12000)]);expect(markdownInline('*a*'.repeat(4000)).length).toBeLessThanOrEqual(257);const unsafe=markdownInline('[x](javascript:alert(1))');expect(unsafe.every(value=>typeof value==='string')).toBe(true);
});
