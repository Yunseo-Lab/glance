import { preview } from "vite";
import { chromium } from "@playwright/test";
import { readFile, writeFile } from "node:fs/promises";
import { gzipSync } from "node:zlib";
const server = await preview({
  preview: { host: "127.0.0.1", port: 8097, strictPort: true },
});
const browser = await chromium.launch({
  executablePath:
    process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH || "/usr/bin/chromium",
});
const results = {
  versions: {
    react: "19.3.1",
    dnd: "6.3.1",
    sortable: "10.0.0",
    gridstack: "14.0.0",
  },
  engines: {},
};
try {
  const manifest = JSON.parse(
    await readFile("dist/.vite/manifest.json", "utf8"),
  );
  for (const engine of ["dnd", "gridstack"]) {
    const page = await browser.newPage({
      viewport: { width: 360, height: 800 },
      hasTouch: true,
    });
    const errors = [];
    page.on("pageerror", (err) => errors.push(err.message));
    await page.goto(`http://127.0.0.1:8097/${engine}.html`);
    await page.locator("article").first().waitFor();
    await page.waitForTimeout(250);
    const initial = await page
      .locator("article")
      .evaluateAll((nodes) => nodes.map((node) => node.dataset.id));
    let keyboard = false,
      touch = false;
    if (engine === "dnd") {
      const handle = page.getByRole("button", { name: "alpha move" });
      await handle.focus();
      await page.keyboard.press("Space");
      await page.waitForTimeout(150);
      await page.keyboard.press("ArrowDown");
      await page.waitForTimeout(150);
      await page.keyboard.press("Space");
      await page.waitForTimeout(300);
      keyboard =
        (
          await page
            .locator("article")
            .evaluateAll((nodes) => nodes.map((node) => node.dataset.id))
        ).join(",") === "beta,alpha,gamma";
      const source = await handle.boundingBox(),
        target = await page
          .getByRole("button", { name: "gamma move" })
          .boundingBox();
      const cdp = await page.context().newCDPSession(page);
      await cdp.send("Input.dispatchTouchEvent", {
        type: "touchStart",
        touchPoints: [{ x: source.x + 10, y: source.y + 10 }],
      });
      await page.waitForTimeout(200);
      for (let i = 1; i <= 8; i++) {
        await cdp.send("Input.dispatchTouchEvent", {
          type: "touchMove",
          touchPoints: [
            {
              x: source.x + 10,
              y: source.y + 10 + ((target.y - source.y) * i) / 8,
            },
          ],
        });
        await page.waitForTimeout(25);
      }
      await cdp.send("Input.dispatchTouchEvent", {
        type: "touchEnd",
        touchPoints: [],
      });
      await page.waitForTimeout(300);
      touch =
        (await page.locator("output").textContent()) ===
        '["beta","gamma","alpha"]';
    } else {
      const cards = page.locator(".grid-stack-item-content");
      const source = await cards.nth(0).boundingBox(),
        target = await cards.nth(2).boundingBox();
      const cdp = await page.context().newCDPSession(page);
      await cdp.send("Input.dispatchTouchEvent", {
        type: "touchStart",
        touchPoints: [{ x: source.x + 30, y: source.y + 30 }],
      });
      await page.waitForTimeout(250);
      for (let i = 1; i <= 10; i++) {
        await cdp.send("Input.dispatchTouchEvent", {
          type: "touchMove",
          touchPoints: [
            {
              x: source.x + 30,
              y: source.y + 30 + ((target.y - source.y) * i) / 10,
            },
          ],
        });
        await page.waitForTimeout(50);
      }
      await cdp.send("Input.dispatchTouchEvent", {
        type: "touchEnd",
        touchPoints: [],
      });
      await page.waitForTimeout(300);
      touch =
        (await page.locator("article").first().getAttribute("gs-y")) !== "0";
    }
    const seen = new Set();
    function dependencies(key) {
      if (seen.has(key)) return;
      seen.add(key);
      for (const dep of manifest[key].imports || []) dependencies(dep);
    }
    dependencies(`${engine}.html`);
    let jsGzip = 0,
      cssGzip = 0;
    for (const key of seen) {
      jsGzip += gzipSync(await readFile("dist/" + manifest[key].file)).length;
      for (const css of manifest[key].css || [])
        cssGzip += gzipSync(await readFile("dist/" + css)).length;
    }
    results.engines[engine] = {
      initial,
      keyboard,
      touch,
      domOrder: await page
        .locator("article")
        .evaluateAll((nodes) => nodes.map((node) => node.dataset.id)),
      positions: await page
        .locator("article")
        .evaluateAll((nodes) => nodes.map((node) => node.getAttribute("gs-y"))),
      pageErrors: errors,
      jsGzip,
      cssGzip,
    };
    await page.close();
  }
  await writeFile("results.json", JSON.stringify(results, null, 2));
  console.log(JSON.stringify(results, null, 2));
  if (
    !results.engines.dnd.keyboard ||
    !results.engines.dnd.touch ||
    results.engines.dnd.pageErrors.length
  )
    process.exitCode = 1;
} finally {
  await browser.close();
  await new Promise((resolve) => server.httpServer.close(resolve));
}
