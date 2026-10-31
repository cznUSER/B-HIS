const fs = require("node:fs");
const path = require("node:path");
const { pathToFileURL } = require("node:url");
const assert = require("node:assert/strict");
const { chromium } = require("C:/Users/SYG/.cache/codex-runtimes/codex-primary-runtime/dependencies/node/node_modules/playwright");
const root = path.resolve(__dirname, "../../..");
const sourcePath = path.join(root, "web/src/views/access/UserOperationsLogPanel.vue");
const phase = process.argv[2] || "after";

(async () => {
  const { createServer } = await import(pathToFileURL(path.join(root, "web/node_modules/vite/dist/node/index.js")));
  const { default: vue } = await import(pathToFileURL(path.join(root, "web/node_modules/@vitejs/plugin-vue/dist/index.mjs")));
  const server = await createServer({ configFile: false, root, plugins: [vue()], cacheDir: path.join(require("node:os").tmpdir(), "uol-fix04-vite"), resolve: { dedupe: ["vue"], alias: { vue: path.join(root, "web/node_modules/vue/dist/vue.esm-bundler.js") } }, define: { "import.meta.env.VITE_USE_MOCK": "false", "import.meta.env.VITE_API_BASE_URL": JSON.stringify("/api/v1") }, server: { host: "127.0.0.1", port: 0 } });
  let browser;
  try {
    await server.listen();
    browser = await chromium.launch({ headless: true, executablePath: "C:/Program Files (x86)/Microsoft/Edge/Application/msedge.exe" });
    const page = await browser.newPage();
    const errors = [];
    page.on("pageerror", error => { errors.push(error.message); console.error("PAGE_ERROR", error.message); });
    page.on("response", response => { if (response.status() >= 400) console.error("HTTP_ERROR", response.status(), response.url()); });
    page.on("console", message => { if (message.type() === "error") console.error("CONSOLE_ERROR", message.text()); });
    await page.route("http://127.0.0.1:*/api/v1/**", route => route.fulfill({ contentType: "application/json", body: JSON.stringify({ items: [], nextCursor: "" }) }));
    const measurements = [];
    for (const width of [1440, 1024, 860, 520, 390]) {
      await page.setViewportSize({ width, height: 900 });
      await page.goto(`http://127.0.0.1:${server.httpServer.address().port}/deliverables/h05/ui90-04c-fix-04/geometry-fixture.html`);
      await page.locator(".user-operations-filters").waitFor();
      await page.evaluate(() => document.fonts.ready);
      const measurement = await page.evaluate(() => {
        const filters = document.querySelector(".user-operations-filters");
        const selectors = [".user-operations-filters select", ".user-operations-user-button-edit", "input[aria-label=记录日期起]", "input[aria-label=记录日期止]"];
        const controls = selectors.map(selector => {
          const bounds = document.querySelector(selector).getBoundingClientRect();
          return { selector, x: bounds.x, y: bounds.y, width: bounds.width, height: bounds.height };
        });
        const bounds = filters.getBoundingClientRect();
        const style = getComputedStyle(filters);
        return { viewport: innerWidth, filtersHeight: bounds.height, controls, twoRows: Math.abs(controls[0].y - controls[1].y) <= 1 && Math.abs(controls[2].y - controls[3].y) <= 1 && controls[2].y >= controls[0].y + controls[0].height, overflow: document.documentElement.scrollWidth > innerWidth + 1, filtersOverflow: filters.scrollWidth > filters.clientWidth + 1, tokens: { filtersHeight: style.getPropertyValue("--uol-filters-h").trim(), controlWidth: style.getPropertyValue("--uol-ctrl-w").trim() } };
      });
      measurements.push(measurement);
      await page.screenshot({ path: path.join(__dirname, `${phase}-${width}.png`), fullPage: true });
    }
    let behavior = null;
    if (phase === "after") {
      const requests = [];
      await page.unroute("http://127.0.0.1:*/api/v1/**");
      await page.route("http://127.0.0.1:*/api/v1/**", route => {
        const url = new URL(route.request().url());
        requests.push({ method: route.request().method(), pathname: url.pathname, query: Object.fromEntries(url.searchParams) });
        const items = url.pathname.endsWith("/users") ? [{ userId: "uol-fixture", username: "uol-fixture", displayName: "测宽用户" }] : url.pathname.endsWith("/access/modules") ? [{ id: 9, name: "探针模块" }] : [];
        return route.fulfill({ contentType: "application/json", body: JSON.stringify({ items, nextCursor: "" }) });
      });
      await page.setViewportSize({ width: 1024, height: 900 });
      await page.reload();
      await page.getByRole("option", { name: "探针模块（9）" }).waitFor({ state: "attached" });
      const userInput = page.locator(".user-operations-user-button-edit input");
      assert.equal(await userInput.getAttribute("readonly"), "");
      await page.getByRole("button", { name: "选择用户", exact: true }).click();
      await page.getByRole("button", { name: "测宽用户(uol-fixture)", exact: true }).click();
      assert.equal(await userInput.inputValue(), "测宽用户(uol-fixture)");
      await page.locator(".user-operations-filters select").selectOption("9");
      await page.getByLabel("记录日期起", { exact: true }).fill("2026-10-01");
      await page.getByLabel("记录日期止", { exact: true }).fill("2026-11-01");
      const selectedResponse = page.waitForResponse(response => response.url().includes("/audit/user-operations?") && new URL(response.url()).searchParams.get("userId") === "uol-fixture");
      await page.getByRole("button", { name: "检索(S)", exact: true }).click();
      await selectedResponse;
      const selectedQuery = requests.filter(item => item.pathname.endsWith("/audit/user-operations")).at(-1).query;
      assert.deepEqual(selectedQuery, { moduleId: "9", userId: "uol-fixture", begin: "2026-10-01", end: "2026-11-01", limit: "50" });
      await page.getByRole("button", { name: "选择用户", exact: true }).click();
      await page.getByRole("button", { name: "清除", exact: true }).click();
      assert.equal(await userInput.inputValue(), "");
      assert.equal(await page.getByRole("dialog", { name: "选择用户" }).count(), 0);
      const clearedResponse = page.waitForResponse(response => response.url().includes("/audit/user-operations?") && !new URL(response.url()).searchParams.has("userId"));
      await page.getByRole("button", { name: "检索(S)", exact: true }).click();
      await clearedResponse;
      const clearedQuery = requests.filter(item => item.pathname.endsWith("/audit/user-operations")).at(-1).query;
      assert.deepEqual(clearedQuery, { moduleId: "9", begin: "2026-10-01", end: "2026-11-01", limit: "50" });
      assert(requests.every(item => item.method === "GET" && ["/api/v1/users", "/api/v1/access/modules", "/api/v1/audit/user-operations"].includes(item.pathname)));
      await page.getByRole("button", { name: "选择用户", exact: true }).click();
      await page.getByRole("button", { name: "测宽用户(uol-fixture)", exact: true }).waitFor();
      await page.getByLabel("筛选用户", { exact: true }).fill("fixture");
      const pickerResponse = page.waitForResponse(response => response.url().includes("/users?q=fixture"));
      await page.getByRole("button", { name: "筛选", exact: true }).click();
      await pickerResponse;
      await page.screenshot({ path: path.join(__dirname, "after-picker-1024.png"), fullPage: true });
      behavior = { mode: "synthetic GET-only responses, no live host or login", readonly: true, ellipsisSelect: true, clear: true, selectedQuery, clearedQuery, pickerQuery: "fixture", requests };
      fs.writeFileSync(path.join(__dirname, "behavior-results.json"), JSON.stringify(behavior, null, 2) + String.fromCharCode(10));
    }
    const source = fs.readFileSync(sourcePath, "utf8");
    const baseline = require("node:child_process").execFileSync("git", ["show", "9443941:web/src/views/access/UserOperationsLogPanel.vue"], { cwd: root, encoding: "utf8" });
    const scriptAndTemplateUnchanged = source.split("<style scoped>")[0].replaceAll(String.fromCharCode(13), "") === baseline.split("<style scoped>")[0].replaceAll(String.fromCharCode(13), "");
    assert(scriptAndTemplateUnchanged);
    const desktopPass = measurements.filter(item => item.viewport >= 1024).every(item => item.twoRows && Math.abs(item.filtersHeight - 81) <= 2 && item.controls.every(control => Math.abs(control.width - 196) <= 2) && item.tokens.filtersHeight === "81px" && item.tokens.controlWidth === "196px" && !item.overflow && !item.filtersOverflow);
    const responsivePass = measurements.every(item => !item.overflow && !item.filtersOverflow);
    const result = { task: "UI90-04C-FIX-04", tipBase: "9443941", phase, mode: "isolated actual Vue component with existing global CSS; geometry uses empty synthetic API responses; not live host or SCORE", sourceSha256: require("node:crypto").createHash("sha256").update(source).digest("hex"), scriptAndTemplateUnchanged, measurements, desktopPass, responsivePass, behaviorPass: behavior ? true : null, errors };
    fs.writeFileSync(path.join(__dirname, `${phase}-results.json`), JSON.stringify(result, null, 2) + String.fromCharCode(10));
    console.log(JSON.stringify({ phase, desktopPass, responsivePass, errors, measurements: measurements.map(item => ({ viewport: item.viewport, filtersHeight: item.filtersHeight, widths: item.controls.map(control => control.width), twoRows: item.twoRows, overflow: item.overflow, filtersOverflow: item.filtersOverflow })) }));
    if (!desktopPass || !responsivePass || errors.length) process.exitCode = 1;
  } finally {
    if (browser) await browser.close();
    await server.close();
  }
})().catch(error => { console.error(error); process.exitCode = 1; });
