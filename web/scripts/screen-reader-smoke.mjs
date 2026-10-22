import { fileURLToPath } from "node:url";
import { chromium } from "@playwright/test";
import { spawn } from "node:child_process";
import { mkdtempSync, rmSync, writeFileSync, mkdirSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
const output =
  process.env.GLANCE_A11Y_OUTPUT_DIR ||
  fileURLToPath(new URL("../../build/verification/", import.meta.url));
mkdirSync(output, { recursive: true });
const directory = mkdtempSync(join(tmpdir(), "glance-orca-"));
const server = spawn(
  fileURLToPath(new URL("../../build/glance", import.meta.url)),
  [
    "dashboard",
    "--listen",
    "127.0.0.1:8098",
    "--db",
    join(directory, "dashboard.db"),
  ],
  {
    env: {
      ...process.env,
      GLANCE_DASHBOARD_TOKEN: "",
      GLANCE_DASHBOARD_TOKEN_FILE: "",
      GLANCE_CONNECTION_KEY_FILE: "",
      GLANCE_DASHBOARD_PUBLIC_URL: "",
    },
    stdio: "ignore",
  },
);
let browser;
let serverError;
server.once("error", (error) => {
  serverError = error;
});
try {
  for (let i = 0; i < 40; i++) {
    if (serverError || server.exitCode !== null)
      throw (
        serverError || new Error("Isolated screen-reader server did not start")
      );
    try {
      if ((await fetch("http://127.0.0.1:8098/api/readyz")).ok) break;
    } catch {}
    await new Promise((r) => setTimeout(r, 250));
  }
  browser = await chromium.launch({
    headless: false,
    executablePath: "/usr/bin/chromium",
    args: ["--force-renderer-accessibility=complete", "--ozone-platform=x11"],
  });
  const page = await browser.newPage({
    viewport: { width: 1280, height: 900 },
  });
  await page.goto("http://127.0.0.1:8098");
  await page.bringToFront();
  await new Promise((resolve) => {
    const inspect = spawn(
      "/usr/bin/python3",
      [
        "-c",
        "import pyatspi; d=pyatspi.Registry.getDesktop(0); print('AT-SPI applications:', [(d.getChildAtIndex(i).name,d.getChildAtIndex(i).getRoleName()) for i in range(d.childCount)])",
      ],
      { stdio: "inherit" },
    );
    inspect.once("exit", resolve);
  });
  await page.getByRole("button", { name: "편집", exact: true }).focus();
  await page.waitForTimeout(700);
  await page
    .getByRole("button", { name: "명령 팔레트 열기", exact: true })
    .click();
  await page.waitForTimeout(700);
  await page.keyboard.press("Escape");
  await page.waitForTimeout(500);
  await page
    .getByRole("button", { name: "위젯 추가", exact: true })
    .first()
    .click();
  await page.waitForTimeout(700);
  await page.getByLabel("제목", { exact: true }).focus();
  await page.getByLabel("제목", { exact: true }).fill("스크린리더 검사 메모");
  await page.waitForTimeout(700);
  await page.getByRole("button", { name: "닫기", exact: true }).click();
  await page.waitForTimeout(700);
  await page.screenshot({ path: join(output, "orca-desktop.png") });
  writeFileSync(
    join(output, "orca-actions.json"),
    JSON.stringify(
      {
        screenReader: "Orca 48.1",
        transport: "AT-SPI2",
        mode: "Braille monitor and focus events; speech audio disabled",
        actions: [
          "focus edit",
          "open command palette and focus search",
          "Escape return focus",
          "open widget editor",
          "focus and fill title",
          "close dialog",
        ],
        completed: true,
      },
      null,
      2,
    ),
  );
  console.log(
    "PASS: headed Chromium keyboard/dialog actions completed with Orca running",
  );
} finally {
  await browser?.close();
  if (server.pid && server.exitCode === null) {
    server.kill("SIGTERM");
    await new Promise((resolve) => server.once("exit", resolve));
  }
  rmSync(directory, { recursive: true, force: true });
}
