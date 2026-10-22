import { chromium } from "@playwright/test";
import { spawn } from "node:child_process";
import { mkdtempSync, rmSync, readFileSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { gzipSync } from "node:zlib";
const root = new URL("../../", import.meta.url);
const directory = mkdtempSync(join(tmpdir(), "glance-performance-"));
const base = "http://127.0.0.1:8097";
const server = spawn(
  new URL("build/glance", root).pathname,
  [
    "dashboard",
    "--listen",
    "127.0.0.1:8097",
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
try {
  for (let i = 0; i < 100; i++) {
    try {
      if ((await fetch(base + "/api/readyz")).ok) break;
    } catch {}
    await new Promise((resolve) => setTimeout(resolve, 100));
  }
  const board = await (await fetch(base + "/api/v1/dashboard")).json();
  board.sources = [];
  board.widgets = Array.from({ length: 32 }, (_, i) => {
    const type = ["clock", "calendar", "note", "bookmarks"][i % 4];
    return {
      id: "perf-" + i,
      type,
      title: `성능 검사 ${i + 1}`,
      size: "small",
      config:
        type === "clock"
          ? { timezone: "Asia/Seoul" }
          : type === "note"
            ? {
                body: "## 오늘의 기록\n개인 대시보드의 일정과 읽을거리를 모읍니다. **중요한 정보**를 편하게 확인하세요.",
              }
            : type === "bookmarks"
              ? {
                  links: [
                    {
                      title: "프로젝트",
                      url: "https://github.com/Yunseo-Lab/glance",
                    },
                  ],
                }
              : {},
    };
  });
  const response = await fetch(base + "/api/v1/dashboard", {
    method: "PUT",
    headers: { "Content-Type": "application/json", Origin: base },
    body: JSON.stringify(board),
  });
  if (!response.ok) throw Error(await response.text());
  browser = await chromium.launch({
    executablePath: process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH,
  });
  const runs = [];
  for (let run = 0; run < 5; run++) {
    const context = await browser.newContext({
      viewport: { width: 360, height: 800 },
      isMobile: true,
      hasTouch: true,
      timezoneId: "Asia/Seoul",
    });
    const page = await context.newPage();
    const cdp = await context.newCDPSession(page);
    await cdp.send("Network.enable");
    await cdp.send("Network.setCacheDisabled", { cacheDisabled: true });
    await cdp.send("Network.emulateNetworkConditions", {
      offline: false,
      latency: 150,
      downloadThroughput: 200000,
      uploadThroughput: 93750,
      connectionType: "cellular3g",
    });
    await cdp.send("Emulation.setCPUThrottlingRate", { rate: 4 });
    await page.addInitScript(() => {
      window.__perf = { lcp: 0, cls: 0, actions: [] };
      new PerformanceObserver((list) => {
        for (const entry of list.getEntries())
          window.__perf.lcp = entry.startTime;
      }).observe({ type: "largest-contentful-paint", buffered: true });
      new PerformanceObserver((list) => {
        for (const entry of list.getEntries())
          if (!entry.hadRecentInput) window.__perf.cls += entry.value;
      }).observe({ type: "layout-shift", buffered: true });
      for (const type of ["click", "input", "keydown"])
        document.addEventListener(
          type,
          () => {
            if (!window.__measure) return;
            window.__measure = false;
            const start = performance.now();
            requestAnimationFrame(() =>
              requestAnimationFrame(() =>
                window.__perf.actions.push(performance.now() - start),
              ),
            );
          },
          true,
        );
    });
    await page.goto(base);
    await page.locator(".widget-card").last().waitFor();
    await page.waitForTimeout(1300);
    const loading = await page.evaluate(() => ({
      ...window.__perf,
      resources: performance
        .getEntriesByType("resource")
        .filter((r) => r.name.endsWith(".js") || r.name.endsWith(".css"))
        .map((r) => ({
          url: r.name,
          transfer: r.transferSize,
          encoded: r.encodedBodySize,
        })),
    }));
    const search = page.getByLabel("위젯 검색", { exact: true });
    const actions = [
      () => search.fill("성능"),
      () => search.fill(""),
      () =>
        page.getByRole("button", { name: "어두운 테마", exact: true }).click(),
      () =>
        page.getByRole("button", { name: "밝은 테마", exact: true }).click(),
      () => page.getByRole("button", { name: "편집", exact: true }).click(),
      () =>
        page
          .getByRole("button", { name: "성능 검사 1 뒤로 이동", exact: true })
          .click(),
      () =>
        page
          .getByRole("button", { name: "편집 되돌리기", exact: true })
          .click(),
      () =>
        page
          .getByRole("button", { name: "편집 다시 적용", exact: true })
          .click(),
      () =>
        page
          .getByRole("button", { name: "편집 되돌리기", exact: true })
          .click(),
      () =>
        page
          .getByRole("button", { name: "성능 검사 1 설정", exact: true })
          .click(),
      () => page.getByRole("button", { name: "닫기", exact: true }).click(),
      () =>
        page.getByRole("button", { name: "빠른 기록", exact: true }).click(),
      () =>
        page
          .getByRole("button", { name: "빠른 기록 닫기", exact: true })
          .click(),
      () => page.getByRole("button", { name: "취소", exact: true }).click(),
      () => page.keyboard.press("Control+k"),
      () => page.keyboard.press("Escape"),
      () =>
        page.getByRole("button", { name: "보드 관리", exact: true }).click(),
      () =>
        page
          .getByRole("button", { name: "보드 관리 닫기", exact: true })
          .click(),
      () => search.fill("없는 정보"),
      () => search.fill(""),
    ];
    for (const action of actions) {
      await page.evaluate(() => (window.__measure = true));
      await action();
      await page.waitForTimeout(300);
    }
    const measured = await page.evaluate(() => window.__perf.actions);
    runs.push({
      run: run + 1,
      lcp_ms: loading.lcp,
      cls: loading.cls,
      actions_ms: measured,
      resources: loading.resources,
    });
    await context.close();
  }
  const median = (values) =>
    [...values].sort((a, b) => a - b)[Math.floor(values.length / 2)];
  const actions = runs.flatMap((r) => r.actions_ms).sort((a, b) => a - b);
  const manifest = JSON.parse(
    readFileSync(
      new URL("internal/dashboard/ui/dist/.vite/manifest.json", root),
    ),
  );
  const entry = manifest["index.html"];
  const size = (file) =>
    gzipSync(readFileSync(new URL("internal/dashboard/ui/dist/" + file, root)))
      .length;
  const result = {
    profile: {
      cards: 32,
      distribution:
        "8 clocks, 8 calendars, 8 notes, 8 bookmark groups; no external upstream",
      viewport: "360x800",
      cpu_slowdown: 4,
      rtt_ms: 150,
      download_bps: 1600000,
      upload_bps: 750000,
      cache: "cold each run",
      runs: 5,
      interaction_metric:
        "trusted event capture to second animation frame; lab feedback latency, not field INP",
      background: process.env.GLANCE_PERF_BACKGROUND || "none",
    },
    median_lcp_ms: median(runs.map((r) => r.lcp_ms)),
    max_cls: Math.max(...runs.map((r) => r.cls)),
    action_p95_ms: actions[Math.ceil(actions.length * 0.95) - 1],
    initial_js_gzip_bytes: size(entry.file),
    initial_css_gzip_bytes: entry.css.reduce(
      (sum, file) => sum + size(file),
      0,
    ),
    runs,
  };
  writeFileSync(
    new URL("docs/verification/frontend-performance.json", root),
    JSON.stringify(result, null, 2) + "\n",
  );
  console.log(JSON.stringify({ ...result, runs: undefined }, null, 2));
  if (
    result.median_lcp_ms > 2500 ||
    result.max_cls > 0.1 ||
    result.action_p95_ms > 200 ||
    result.initial_js_gzip_bytes > 150 * 1024 ||
    result.initial_css_gzip_bytes > 20 * 1024 ||
    actions.length !== 100
  )
    throw Error("Performance budget failed");
} finally {
  await browser?.close();
  server.kill("SIGTERM");
  await new Promise((resolve) => server.once("exit", resolve));
  rmSync(directory, { recursive: true, force: true });
}
