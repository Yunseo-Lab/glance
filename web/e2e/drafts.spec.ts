import { expect, test } from "@playwright/test";
import {
  encodeDraft,
  decodeDraft,
  draftTTL,
} from "../src/editor/draftRecovery";
import type { Board } from "../src/types";
test("device draft has bounded lifetime and contains source IDs without provider addresses", () => {
  const saved: Board = {
    id: "home",
    revision: 3,
    title: "원본",
    theme: "light",
    widgets: [
      {
        id: "feed",
        type: "rss",
        size: "wide",
        title: "소식",
        sourceId: "source",
        config: { limit: 5 },
      },
    ],
    sources: [
      {
        id: "source",
        type: "rss",
        config: { url: "https://example.com/private?token=never-store" },
      },
    ],
  };
  const draft = { ...saved, title: "복구" };
  const text = encodeDraft(saved, draft, 1000)!;
  expect(text).not.toContain("example.com");
  expect(text).not.toContain("never-store");
  expect(decodeDraft(text, saved, 1001)).toEqual(draft);
  expect(decodeDraft(text, saved, 1000 + draftTTL)).toBeUndefined();
  expect(decodeDraft(text, { ...saved, revision: 4 }, 1001)).toBeUndefined();
  expect(decodeDraft("{broken", saved)).toBeUndefined();
  expect(
    encodeDraft(saved, {
      ...draft,
      sources: [
        ...draft.sources,
        { id: "new", type: "rss", config: { url: "https://new.example.com" } },
      ],
    }),
  ).toBeUndefined();
});
test("device recovery is opt in, previews validated drafts after reload, and can erase them", async ({
  page,
}) => {
  page.on("dialog", (dialog) => dialog.accept());
  await page.goto("/");
  await page.getByRole("button", { name: "편집", exact: true }).click();
  const optIn = page.getByRole("checkbox", {
    name: /이 기기에 초안 복구 허용/,
  });
  await expect(optIn).not.toBeChecked();
  await page.getByLabel("대시보드 이름").fill("복구할 개인 초안");
  expect(
    await page.evaluate(() => localStorage.getItem("glance:draft:home")),
  ).toBeNull();
  await optIn.check();
  await expect
    .poll(() => page.evaluate(() => localStorage.getItem("glance:draft:home")))
    .toContain("복구할 개인 초안");
  expect(
    await page.evaluate(
      () =>
        JSON.parse(localStorage.getItem("glance:draft:home")!).board.sources,
    ),
  ).toEqual([]);
  await page.reload();
  await expect(
    page.getByRole("button", { name: "기기 초안 복구", exact: true }),
  ).toBeVisible();
  await page
    .getByRole("button", { name: "기기 초안 복구", exact: true })
    .click();
  await expect(page.getByLabel("대시보드 이름")).toHaveValue(
    "복구할 개인 초안",
  );
  await optIn.uncheck();
  expect(
    await page.evaluate(() =>
      Object.keys(localStorage).filter((key) => key.startsWith("glance:draft")),
    ),
  ).toEqual([]);
});
