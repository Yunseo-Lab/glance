import { expect, test } from "@playwright/test";
import type { Board } from "../src/types";
const origin = "http://127.0.0.1:8091";
test("board creation, rename, clone and deletion preserve widget identities correctly", async ({
  page,
  request,
}) => {
  const created: string[] = [];
  page.on("dialog", (dialog) => dialog.accept());
  try {
    await page.goto("/");
    await page.getByRole("button", { name: "보드 관리", exact: true }).click();
    await page.getByLabel("새 보드 이름", { exact: true }).fill("생활 보드");
    await page.getByLabel("시작할 구성", { exact: true }).selectOption("daily");
    await page
      .getByRole("button", { name: "보드 만들기", exact: true })
      .click();
    await expect(page.locator("article.widget-card")).toHaveCount(4);
    const id = await page.getByLabel("보드 선택", { exact: true }).inputValue();
    created.push(id);
    const original: Board = await (
      await request.get(`/api/v1/boards/${id}`)
    ).json();
    await page.getByRole("button", { name: "보드 관리", exact: true }).click();
    await page
      .getByLabel("생활 보드 보드 이름", { exact: true })
      .fill("내 생활");
    await page
      .getByRole("button", { name: "생활 보드 이름 저장", exact: true })
      .click();
    await expect(
      page.getByLabel("내 생활 보드 이름", { exact: true }),
    ).toHaveValue("내 생활");
    await page
      .getByRole("button", { name: "내 생활 보드 복제", exact: true })
      .click();
    await expect(page.getByLabel("보드 선택", { exact: true })).not.toHaveValue(
      id,
    );
    const cloneID = await page
      .getByLabel("보드 선택", { exact: true })
      .inputValue();
    created.push(cloneID);
    const clone: Board = await (
      await request.get(`/api/v1/boards/${cloneID}`)
    ).json();
    expect(clone.widgets).toHaveLength(4);
    expect(
      clone.widgets.some((widget) =>
        original.widgets.some((other) => widget.id === other.id),
      ),
    ).toBe(false);
    await page.getByRole("button", { name: "보드 관리", exact: true }).click();
    await page
      .getByRole("button", { name: "내 생활 복사 보드 삭제", exact: true })
      .click();
    await expect(page.getByLabel("보드 선택", { exact: true })).toHaveValue(
      "home",
    );
    expect((await request.get(`/api/v1/boards/${cloneID}`)).status()).toBe(404);
  } finally {
    for (const id of created) {
      const response = await request.get(`/api/v1/boards/${id}`);
      if (response.ok()) {
        const board: Board = await response.json();
        await request.delete(`/api/v1/boards/${id}`, {
          headers: { Origin: origin },
          data: { revision: board.revision },
        });
      }
    }
  }
});
test("dirty navigation cancellation preserves back history and the local edit", async ({
  page,
  request,
}) => {
  const board: Board = await (
    await request.post("/api/v1/boards", {
      headers: { Origin: origin },
      data: { title: "이동 검증" },
    })
  ).json();
  try {
    await page.goto("/");
    await page.getByLabel("보드 선택", { exact: true }).selectOption(board.id);
    await page.getByRole("button", { name: "편집", exact: true }).click();
    await page.getByLabel("대시보드 이름").fill("잃지 않을 초안");
    page.once("dialog", (dialog) => dialog.dismiss());
    await page.goBack();
    await expect(page.getByLabel("보드 선택", { exact: true })).toHaveValue(
      board.id,
    );
    await expect(page.getByLabel("대시보드 이름")).toHaveValue(
      "잃지 않을 초안",
    );
    page.once("dialog", (dialog) => dialog.accept());
    await page.goBack();
    await expect(page.getByLabel("보드 선택", { exact: true })).toHaveValue(
      "home",
    );
    await page.goForward();
    await expect(page.getByLabel("보드 선택", { exact: true })).toHaveValue(
      board.id,
    );
    await expect(
      page.getByRole("button", { name: "편집", exact: true }),
    ).toBeVisible();
  } finally {
    await request.delete(`/api/v1/boards/${board.id}`, {
      headers: { Origin: origin },
      data: { revision: board.revision },
    });
  }
});
