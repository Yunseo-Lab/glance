import { expect, test } from "@playwright/test";
import { mergeBoards } from "../src/editor/merge";
import type { Board } from "../src/types";
const base: Board = {
  id: "home",
  revision: 1,
  title: "원본",
  theme: "light",
  widgets: [
    {
      id: "note",
      type: "note",
      size: "small",
      title: "메모",
      config: { body: "원본" },
    },
  ],
  sources: [],
};
test("three-way merge preserves disjoint remote changes and reports competing edits", () => {
  const local = { ...base, title: "내 이름" },
    remote = {
      ...base,
      revision: 2,
      widgets: [{ ...base.widgets[0], config: { body: "다른 탭" } }],
    };
  let result = mergeBoards(base, local, remote);
  expect(result.conflicts).toEqual([]);
  expect(result.board.title).toBe("내 이름");
  expect(result.board.widgets[0].config.body).toBe("다른 탭");
  expect(result.board.revision).toBe(2);
  result = mergeBoards(base, local, { ...remote, title: "원격 이름" });
  expect(result.conflicts).toContain("title");
  expect(result.board.title).toBe("내 이름");
  result = mergeBoards(base, { ...base, widgets: [] }, remote);
  expect(result.conflicts).toContain("widgets.note");
  expect(result.board.widgets).toEqual([]);
  expect(base.widgets).toHaveLength(1);
});
test("two tabs compare and reapply a local draft while preserving a remote note", async ({
  page,
  context,
  request,
}) => {
  const origin = "http://127.0.0.1:8091",
    headers = { Origin: origin };
  let board: Board = await (
    await request.post("/api/v1/boards", {
      headers,
      data: { title: "충돌 검증" },
    })
  ).json();
  const seeded = await request.put(`/api/v1/boards/${board.id}`, {
    headers,
    data: { ...base, id: board.id, revision: board.revision },
  });
  expect(seeded.status()).toBe(200);
  board = await seeded.json();
  const other = await context.newPage();
  try {
    await page.goto(`/#/boards/${board.id}`);
    await other.goto(`/#/boards/${board.id}`);
    await page.getByRole("button", { name: "편집", exact: true }).click();
    await page.getByLabel("대시보드 이름").fill("내 변경 보존");
    await other.getByRole("button", { name: "편집", exact: true }).click();
    await other.getByRole("button", { name: "메모 설정", exact: true }).click();
    await other.getByLabel("내용", { exact: true }).fill("다른 탭의 메모 보존");
    await other.getByRole("button", { name: "설정 적용", exact: true }).click();
    await other
      .getByRole("button", { name: "대시보드 저장", exact: true })
      .click();
    await expect(
      other.getByText("변경 사항을 저장했습니다.", { exact: true }),
    ).toBeVisible();
    await page
      .getByRole("button", { name: "대시보드 저장", exact: true })
      .click();
    await expect(page.getByLabel("대시보드 이름")).toHaveValue("내 변경 보존");
    await page.getByRole("button", { name: "변경 비교", exact: true }).click();
    await expect(
      page.getByRole("heading", { name: "다른 탭의 변경", exact: true }),
    ).toBeVisible();
    await page
      .getByRole("button", { name: "내 변경으로 재적용", exact: true })
      .click();
    await expect(
      page.getByText("다른 탭의 메모 보존", { exact: true }),
    ).toBeVisible();
    await expect(page.getByLabel("대시보드 이름")).toHaveValue("내 변경 보존");
    const unsaved: Board = await (
      await request.get(`/api/v1/boards/${board.id}`)
    ).json();
    expect(unsaved.title).toBe("원본");
    await page
      .getByRole("button", { name: "대시보드 저장", exact: true })
      .click();
    await expect(
      page.getByText("변경 사항을 저장했습니다.", { exact: true }),
    ).toBeVisible();
    const saved: Board = await (
      await request.get(`/api/v1/boards/${board.id}`)
    ).json();
    expect(saved.title).toBe("내 변경 보존");
    expect(saved.widgets[0].config.body).toBe("다른 탭의 메모 보존");
  } finally {
    await other.close();
    const latest: Board = await (
      await request.get(`/api/v1/boards/${board.id}`)
    ).json();
    await request.delete(`/api/v1/boards/${board.id}`, {
      headers,
      data: { revision: latest.revision },
    });
  }
});
