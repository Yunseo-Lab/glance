import { expect, test } from "@playwright/test";
import type { Board } from "../src/types";
test("GitHub widget saves a bounded public repository query and renders a safe release link", async ({
  page,
  request,
}) => {
  const headers = { Origin: "http://127.0.0.1:8091" };
  const board: Board = await (
    await request.post("/api/v1/boards", {
      headers,
      data: { title: "GitHub 검증" },
    })
  ).json();
  try {
    await page.route(`**/api/v1/boards/${board.id}/snapshots`, (route) =>
      route.fulfill({ json: {} }),
    );
    await page.goto(`/#/boards/${board.id}`);
    await page
      .getByRole("button", { name: "위젯 추가", exact: true })
      .first()
      .click();
    await page.getByLabel("위젯 종류", { exact: true }).selectOption("github");
    await page.getByLabel("제목", { exact: true }).fill("개발 소식");
    await page.getByLabel("저장소", { exact: true }).fill("owner/repo");
    await page
      .getByLabel("가져올 정보", { exact: true })
      .selectOption("releases");
    await page
      .getByRole("button", { name: "위젯 추가", exact: true })
      .last()
      .click();
    await page
      .getByRole("button", { name: "대시보드 저장", exact: true })
      .click();
    await expect(
      page.getByText("변경 사항을 저장했습니다.", { exact: true }),
    ).toBeVisible();
    const saved: Board = await (
      await request.get(`/api/v1/boards/${board.id}`)
    ).json();
    expect(saved.sources[0].type).toBe("github");
    expect(saved.sources[0].config.repository).toBe("owner/repo");
    await page.route(`**/api/v1/boards/${board.id}/snapshots`, (route) =>
      route.fulfill({
        json: {
          [saved.sources[0].id]: {
            state: "fresh",
            data: {
              title: "owner/repo",
              items: [
                {
                  id: "release-1",
                  title: "v1 release",
                  url: "https://github.com/owner/repo/releases/tag/v1",
                },
              ],
            },
          },
        },
      }),
    );
    await page.reload();
    await expect(
      page.getByRole("link", { name: /v1 release/ }),
    ).toHaveAttribute("href", "https://github.com/owner/repo/releases/tag/v1");
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

test("private GitHub selection keeps only a connection reference in the unsaved editor", async ({
  page,
}) => {
  await page.route("**/api/v1/connections", (route) =>
    route.fulfill({
      json: [
        {
          id: "fixture-connection",
          kind: "github",
          name: "나의 읽기 연결",
          endpoint: "https://api.github.com",
          hasSecret: true,
          generation: 1,
          status: "unverified",
          createdAt: new Date().toISOString(),
        },
      ],
    }),
  );
  await page.goto("/");
  await page
    .getByRole("button", { name: "위젯 추가", exact: true })
    .first()
    .click();
  await page.getByLabel("위젯 종류", { exact: true }).selectOption("github");
  await page.getByLabel("제목", { exact: true }).fill("비공개 편집 검증");
  await page
    .getByLabel("GitHub 연결", { exact: true })
    .selectOption("fixture-connection");
  expect(await page.locator("input[type=password]").count()).toBe(0);
  await page
    .getByRole("button", { name: "위젯 추가", exact: true })
    .last()
    .click();
  await page
    .getByRole("button", { name: "비공개 편집 검증 설정", exact: true })
    .click();
  await expect(page.getByLabel("GitHub 연결", { exact: true })).toHaveValue(
    "fixture-connection",
  );
  await page.getByRole("button", { name: "닫기", exact: true }).click();
});
