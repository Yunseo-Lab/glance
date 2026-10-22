import { expect, test } from "@playwright/test";
import { readFileSync } from "node:fs";
import { parseDashboardFile } from "../src/transfer";
test("v1 import preserves legacy widget/source IDs and rejects malformed, future and oversized files", () => {
  const board = JSON.parse(
    readFileSync(
      new URL(
        "../../internal/dashboard/testdata/legacy-v1.json",
        import.meta.url,
      ),
      "utf8",
    ),
  );
  const parsed = parseDashboardFile(
    JSON.stringify({ format: "glance-dashboard", version: 1, board }),
  );
  expect(parsed).toEqual(board);
  for (const text of [
    "null",
    "[]",
    "{broken",
    JSON.stringify({ format: "glance-dashboard", version: 2, board }),
    JSON.stringify({ format: "other", version: 1, board }),
    JSON.stringify({ format: "glance-dashboard", version: 1, board: [] }),
  ])
    expect(() => parseDashboardFile(text)).toThrow();
  expect(() => parseDashboardFile("가".repeat(349526))).toThrow("1MB");
});
test("invalid source references and future widget config do not change the preview or persisted board", async ({
  page,
  request,
}) => {
  const baseline = await (await request.get("/api/v1/dashboard")).json();
  await page.goto("/");
  for (const widgets of [
    baseline.widgets.map((widget: { sourceId?: string }) =>
      widget.sourceId ? { ...widget, sourceId: "missing-source" } : widget,
    ),
    baseline.widgets.map((widget: unknown, index: number) =>
      index === 0 ? { ...(widget as object), configVersion: 999 } : widget,
    ),
  ]) {
    await page
      .getByRole("button", { name: "설정 가져오기", exact: true })
      .click();
    await page.locator("input[type=file]").setInputFiles({
      name: "invalid.json",
      mimeType: "application/json",
      buffer: Buffer.from(
        JSON.stringify({
          format: "glance-dashboard",
          version: 1,
          board: { ...baseline, widgets },
        }),
      ),
    });
    await expect(page.getByRole("alert")).toBeVisible();
    await expect(
      page.getByRole("button", { name: "편집", exact: true }),
    ).toBeVisible();
    expect(await (await request.get("/api/v1/dashboard")).json()).toEqual(
      baseline,
    );
  }
});
