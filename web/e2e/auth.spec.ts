import { test, expect } from "@playwright/test";
import { spawn, type ChildProcess } from "node:child_process";
import { randomBytes } from "node:crypto";
import { mkdtempSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { fileURLToPath } from "node:url";

const password = randomBytes(32).toString("base64url");
const origin = "http://127.0.0.1:8094";
let server: ChildProcess, directory: string;
test.beforeEach(async () => {
  directory = mkdtempSync(join(tmpdir(), "glance-auth-"));
  server = spawn(
    fileURLToPath(new URL("../../build/glance", import.meta.url)),
    [
      "dashboard",
      "--listen",
      "127.0.0.1:8094",
      "--db",
      join(directory, "dashboard.db"),
    ],
    {
      env: {
        ...process.env,
        GLANCE_DASHBOARD_TOKEN: password,
        GLANCE_DASHBOARD_TOKEN_FILE: "",
        GLANCE_DASHBOARD_PUBLIC_URL: "",
        GLANCE_CONNECTION_KEY_FILE: "",
      },
      stdio: "ignore",
    },
  );
  await expect
    .poll(async () => {
      try {
        return (await fetch(origin + "/api/v1/auth/session")).status;
      } catch {
        return 0;
      }
    })
    .toBe(200);
});
test.afterEach(async () => {
  if (server && server.exitCode === null) {
    const exited = new Promise((resolve) => server.once("exit", resolve));
    server.kill("SIGTERM");
    await exited;
  }
  rmSync(directory, { recursive: true, force: true });
});

test("owner login protects data and logout revokes the session", async ({
  page,
}) => {
  await page.goto(origin);
  await expect(
    page.getByRole("heading", { name: "나의 공간에 로그인" }),
  ).toBeVisible();
  await page.getByLabel("비밀번호", { exact: true }).fill("wrong-password");
  await page.getByRole("button", { name: "로그인", exact: true }).click();
  await expect(page.getByRole("alert")).toContainText("비밀번호를 확인");
  await page.getByLabel("비밀번호", { exact: true }).fill(password);
  await page.getByRole("button", { name: "로그인", exact: true }).click();
  await expect(
    page.getByRole("heading", { name: "오늘도, 한눈에." }),
  ).toBeVisible();
  const cookie = (await page.context().cookies()).find(
    (cookie) => cookie.name === "glance_session",
  )!;
  expect(cookie.httpOnly).toBe(true);
  expect(cookie.sameSite).toBe("Strict");
  expect(await page.evaluate(() => document.cookie)).not.toContain(
    "glance_session",
  );
  await page.reload();
  await expect(page.getByRole("button", { name: "로그아웃" })).toBeVisible();
  await page.getByRole("button", { name: "로그아웃" }).click();
  await expect(
    page.getByRole("heading", { name: "나의 공간에 로그인" }),
  ).toBeVisible();
  expect(
    (
      await page.request.get(origin + "/api/v1/dashboard", {
        headers: { Cookie: `glance_session=${cookie.value}` },
      })
    ).status(),
  ).toBe(401);
});

test("session loss returns to login while preserving an unsaved draft", async ({
  page,
}) => {
  await page.goto(origin);
  await page.getByLabel("비밀번호", { exact: true }).fill(password);
  await page.getByRole("button", { name: "로그인", exact: true }).click();
  await page.getByRole("button", { name: "편집", exact: true }).click();
  await page.getByLabel("대시보드 이름").fill("세션 복구 후에도 유지");
  await page.context().clearCookies();
  await page.evaluate(() => window.dispatchEvent(new Event("online")));
  await expect(
    page.getByRole("heading", { name: "나의 공간에 로그인" }),
  ).toBeVisible();
  await page.getByLabel("비밀번호", { exact: true }).fill(password);
  await page.getByRole("button", { name: "로그인", exact: true }).click();
  await expect(page.getByLabel("대시보드 이름")).toHaveValue(
    "세션 복구 후에도 유지",
  );
});

test("logout removes private query responses and a new login fetches current connection metadata", async ({
  page,
  request,
}) => {
  await page.goto(origin);
  await page.getByLabel("비밀번호", { exact: true }).fill(password);
  await page.getByRole("button", { name: "로그인", exact: true }).click();
  await page.getByRole("button", { name: "데이터 소스", exact: true }).click();
  await expect(
    page.getByText("아직 개인 연결이 없습니다.", { exact: false }),
  ).toBeVisible();
  let reads = 0;
  page.on("request", (req) => {
    if (req.url() === origin + "/api/v1/connections" && req.method() === "GET")
      reads++;
  });
  await page.evaluate(() => {
    localStorage.setItem("glance:draft-opt-in", "true");
    localStorage.setItem("glance:draft:fixture", "private fixture");
  });
  await page.getByRole("button", { name: "로그아웃", exact: true }).click();
  await expect(
    page.getByRole("heading", { name: "나의 공간에 로그인" }),
  ).toBeVisible();
  expect(
    await page.evaluate(() =>
      Object.keys(localStorage).filter((key) => key.startsWith("glance:draft")),
    ),
  ).toEqual([]);
  const created = await request.post(origin + "/api/v1/connections", {
    headers: {
      Origin: origin,
      Authorization:
        "Basic " + Buffer.from("owner:" + password).toString("base64"),
    },
    data: {
      name: "새 세션 연결",
      kind: "github",
      token: "fixture_not_used_for_network",
    },
  });
  expect(created.status()).toBe(201);
  await page.getByLabel("비밀번호", { exact: true }).fill(password);
  await page.getByRole("button", { name: "로그인", exact: true }).click();
  await expect(
    page.getByRole("heading", { name: "새 세션 연결", exact: true }),
  ).toBeVisible();
  expect(reads).toBeGreaterThan(0);
});

test("session expiry hides private content and retains an unfinished widget form only in memory", async ({
  page,
}) => {
  await page.goto(origin);
  await page.getByLabel("비밀번호", { exact: true }).fill(password);
  await page.getByRole("button", { name: "로그인", exact: true }).click();
  await page.getByRole("button", { name: "위젯 추가", exact: true }).click();
  await page.getByLabel("위젯 종류", { exact: true }).selectOption("note");
  await page
    .getByLabel("제목", { exact: true })
    .fill("아직 저장하지 않은 제목");
  await page.context().clearCookies();
  await page.evaluate(() => window.dispatchEvent(new Event("online")));
  await expect(
    page.getByRole("heading", { name: "나의 공간에 로그인" }),
  ).toBeVisible();
  await expect(page.getByRole("dialog")).toHaveCount(0);
  await expect(
    page.getByRole("heading", { name: "오늘도, 한눈에." }),
  ).toBeHidden();
  expect(
    await page.evaluate(() => JSON.stringify({ ...localStorage })),
  ).not.toContain("아직 저장하지 않은 제목");
  await page.getByLabel("비밀번호", { exact: true }).fill(password);
  await page.getByRole("button", { name: "로그인", exact: true }).click();
  await expect(page.getByLabel("제목", { exact: true })).toHaveValue(
    "아직 저장하지 않은 제목",
  );
  await expect(page.getByRole("dialog")).toBeVisible();
});
