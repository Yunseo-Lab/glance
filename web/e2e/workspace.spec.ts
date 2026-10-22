import { expect, test } from "@playwright/test";
import type { Board } from "../src/types";
let board: Board;
test.beforeAll(async ({ request }) => {
  const response = await request.post("/api/v1/boards", {
    headers: { Origin: "http://127.0.0.1:8091" },
    data: { title: "워크스페이스 검증" },
  });
  expect(response.status()).toBe(201);
  board = await response.json();
});
test.afterAll(async ({ request }) => {
  const latest = await (await request.get(`/api/v1/boards/${board.id}`)).json();
  await request.delete(`/api/v1/boards/${board.id}`, {
    headers: { Origin: "http://127.0.0.1:8091" },
    data: { revision: latest.revision },
  });
});
test("direct board URLs, source/settings navigation and browser history preserve context", async ({
  page,
}) => {
  await page.goto(`/#/boards/${board.id}`);
  await expect(
    page.getByRole("heading", { name: "오늘도, 한눈에." }),
  ).toBeVisible();
  await expect(page.locator("article.widget-card")).toHaveCount(0);
  await expect(page.getByLabel("보드 선택", { exact: true })).toHaveValue(
    board.id,
  );
  await page.getByRole("button", { name: "데이터 소스", exact: true }).click();
  await expect(
    page.getByRole("heading", { name: "데이터 소스", exact: true }),
  ).toBeVisible();
  await page.goBack();
  await expect(
    page.getByRole("heading", { name: "오늘도, 한눈에." }),
  ).toBeVisible();
  await page.getByRole("button", { name: "설정", exact: true }).click();
  await expect(
    page.getByRole("heading", { name: "설정", exact: true }),
  ).toBeVisible();
  await page.reload();
  await expect(page.getByLabel("보드 선택", { exact: true })).toHaveValue(
    board.id,
  );
  await expect(
    page.getByRole("heading", { name: "설정", exact: true }),
  ).toBeVisible();
});
test("mobile navigation remains usable and unknown pages can recover", async ({
  page,
}) => {
  await page.setViewportSize({ width: 360, height: 800 });
  await page.goto("/");
  await page
    .getByLabel("모바일 보드 선택", { exact: true })
    .selectOption(board.id);
  await expect(page.locator("article.widget-card")).toHaveCount(0);
  await page.getByRole("button", { name: "모아 읽기", exact: true }).click();
  await expect(
    page.getByRole("heading", { name: "모아 읽기", exact: true }),
  ).toBeVisible();
  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= innerWidth,
    ),
  ).toBe(true);
  await page.goto("/#/unknown");
  await expect(
    page.getByRole("heading", { name: "이 페이지를 찾을 수 없습니다" }),
  ).toBeVisible();
  await page.getByRole("button", { name: "내 대시보드로 이동" }).click();
  await expect(
    page.getByRole("heading", { name: "오늘도, 한눈에." }),
  ).toBeVisible();
});

test("starter template previews without saving and commits key-free daily widgets", async ({
  page,
  request,
}) => {
  const response = await request.post("/api/v1/boards", {
    headers: { Origin: "http://127.0.0.1:8091" },
    data: { title: "첫 사용 검증" },
  });
  const starter: Board = await response.json();
  try {
    await page.goto(`/#/boards/${starter.id}`);
    await page.getByRole("button", { name: /일상으로 시작/ }).click();
    await expect(page.locator("article.widget-card")).toHaveCount(4);
    let persisted: Board = await (
      await request.get(`/api/v1/boards/${starter.id}`)
    ).json();
    expect(persisted.widgets).toHaveLength(0);
    await page
      .getByRole("button", { name: "대시보드 저장", exact: true })
      .click();
    await expect(
      page.getByText("변경 사항을 저장했습니다.", { exact: true }),
    ).toBeVisible();
    await page.reload();
    await expect(page.locator("article.widget-card")).toHaveCount(4);
    persisted = await (
      await request.get(`/api/v1/boards/${starter.id}`)
    ).json();
    expect(persisted.sources).toHaveLength(0);
  } finally {
    const current: Board = await (
      await request.get(`/api/v1/boards/${starter.id}`)
    ).json();
    await request.delete(`/api/v1/boards/${starter.id}`, {
      headers: { Origin: "http://127.0.0.1:8091" },
      data: { revision: current.revision },
    });
  }
});
