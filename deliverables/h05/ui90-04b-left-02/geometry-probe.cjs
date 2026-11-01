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
  const server = await createServer({
    configFile: false,
    root,
    plugins: [vue()],
    cacheDir: path.join(require("node:os").tmpdir(), "uol-left02-vite"),
    resolve: { dedupe: ["vue"], alias: { vue: path.join(root, "web/node_modules/vue/dist/vue.esm-bundler.js") } },
    define: { "import.meta.env.VITE_USE_MOCK": "false", "import.meta.env.VITE_API_BASE_URL": JSON.stringify("/api/v1") },
    server: { host: "127.0.0.1", port: 0 },
  });
  let browser;
  try {
    await server.listen();
    browser = await chromium.launch({ headless: true, executablePath: "C:/Program Files (x86)/Microsoft/Edge/Application/msedge.exe" });
    const page = await browser.newPage();
    const errors = [];
    page.on("pageerror", error => { errors.push(error.message); console.error("PAGE_ERROR", error.message); });
    await page.route("http://127.0.0.1:*/api/v1/**", route => route.fulfill({ contentType: "application/json", body: JSON.stringify({ items: [], nextCursor: "" }) }));
    const measurements = [];
    for (const width of [1440, 1024, 861, 860, 520, 390]) {
      await page.setViewportSize({ width, height: 900 });
      await page.goto(`http://127.0.0.1:${server.httpServer.address().port}/deliverables/h05/ui90-04b-left-02/geometry-fixture.html`);
      await page.locator(".user-operations-filters").waitFor();
      await page.evaluate(() => document.fonts.ready);
      const measurement = await page.evaluate(() => {
        const header = document.querySelector(".master-panel-header");
        const filters = document.querySelector(".user-operations-filters");
        const headerBounds = header.getBoundingClientRect();
        const filtersBounds = filters.getBoundingClientRect();
        const headerStyle = getComputedStyle(header);
        const filtersStyle = getComputedStyle(filters);
        const returnEl = document.querySelector(".uol-hdr-return");
        const titleEl = document.querySelector(".master-panel-copy");
        const searchEl = document.querySelector(".uol-hdr-search");
        const exportEl = document.querySelector(".uol-hdr-export");
        const rel = (el) => {
          const b = el.getBoundingClientRect();
          return {
            left: b.left - headerBounds.left,
            top: b.top - headerBounds.top,
            right: headerBounds.right - b.right,
            width: b.width,
            height: b.height,
            position: getComputedStyle(el).position,
            offsetParentIsHeader: el.offsetParent === header,
          };
        };
        const filterDefs = [
          ["label-module", ".user-operations-filters > label > span", 108, 6],
          ["ctrl-module", ".user-operations-filters select", 183, 6],
          ["label-user", "#user-operations-user-label", 385, 6],
          ["ctrl-user", ".user-operations-user-button-edit", 431, 6],
          ["label-begin", ".user-operations-record-date > span", 108, 37],
          ["ctrl-begin", "input[aria-label='记录日期起']", 183, 37],
          ["label-sep", ".user-operations-date-sep", 385, 37],
          ["ctrl-end", "input[aria-label='记录日期止']", 431, 37],
        ];
        const filterTargets = filterDefs.map(([name, selector, expectedLeft, expectedTop]) => {
          const element = document.querySelector(selector);
          const bounds = element.getBoundingClientRect();
          return {
            name,
            left: bounds.left - filtersBounds.left,
            top: bounds.top - filtersBounds.top,
            expectedLeft,
            expectedTop,
            position: getComputedStyle(element).position,
            width: bounds.width,
            token: filtersStyle.getPropertyValue("--uol-left-" + name).trim(),
          };
        });
        const controls = [".user-operations-filters select", ".user-operations-user-button-edit", "input[aria-label='记录日期起']", "input[aria-label='记录日期止']"].map(selector => {
          const bounds = document.querySelector(selector).getBoundingClientRect();
          return { selector, width: bounds.width, height: bounds.height };
        });
        return {
          viewport: innerWidth,
          header: {
            width: headerBounds.width,
            height: headerBounds.height,
            position: headerStyle.position,
            tokens: {
              leftReturn: headerStyle.getPropertyValue("--uol-hdr-left-return").trim(),
              leftTitle: headerStyle.getPropertyValue("--uol-hdr-left-title").trim(),
              leftSearch: headerStyle.getPropertyValue("--uol-hdr-left-search").trim(),
              leftExport: headerStyle.getPropertyValue("--uol-hdr-left-export").trim(),
              rightSearch: headerStyle.getPropertyValue("--uol-hdr-right-search").trim(),
              rightExport: headerStyle.getPropertyValue("--uol-hdr-right-export").trim(),
            },
            return: rel(returnEl),
            title: rel(titleEl),
            search: rel(searchEl),
            export: rel(exportEl),
          },
          filters: {
            height: filtersBounds.height,
            position: filtersStyle.position,
            tokens: {
              filtersHeight: filtersStyle.getPropertyValue("--uol-filters-h").trim(),
              controlWidth: filtersStyle.getPropertyValue("--uol-ctrl-w").trim(),
              row1: filtersStyle.getPropertyValue("--uol-top-r1").trim(),
              row2: filtersStyle.getPropertyValue("--uol-top-r2").trim(),
            },
            targets: filterTargets,
            controls,
          },
          overflow: document.documentElement.scrollWidth > innerWidth + 1,
        };
      });
      const desktop = measurement.viewport > 860;
      const hdr = measurement.header;
      measurement.headerPass = desktop
        ? hdr.position === "relative"
          && hdr.tokens.leftReturn === "15px"
          && hdr.tokens.leftTitle === "110px"
          && hdr.tokens.leftSearch === "813px"
          && hdr.tokens.leftExport === "909px"
          && hdr.tokens.rightSearch === "105px"
          && hdr.tokens.rightExport === "9px"
          && Math.abs(hdr.height - 45) <= 2
          && Math.abs(hdr.return.left - 15) <= 2
          && Math.abs(hdr.title.left - 110) <= 2
          && Math.abs(hdr.search.right - 105) <= 2
          && Math.abs(hdr.export.right - 9) <= 2
          && Math.abs(hdr.return.height - 30) <= 2
          && Math.abs(hdr.search.height - 30) <= 2
          && Math.abs(hdr.export.height - 30) <= 2
          && hdr.return.position === "absolute"
          && hdr.title.position === "absolute"
          && hdr.search.position === "absolute"
          && hdr.export.position === "absolute"
        : true;
      measurement.headerFlowPass = !desktop
        ? hdr.return.position !== "absolute" && hdr.search.position !== "absolute" && hdr.export.position !== "absolute"
        : true;
      measurement.left01Pass = desktop
        ? measurement.filters.position === "relative"
          && measurement.filters.tokens.filtersHeight === "81px"
          && measurement.filters.tokens.controlWidth === "196px"
          && measurement.filters.tokens.row1 === "6px"
          && measurement.filters.tokens.row2 === "37px"
          && Math.abs(measurement.filters.height - 81) <= 2
          && measurement.filters.controls.every(c => Math.abs(c.width - 196) <= 2)
          && measurement.filters.targets.every(t => t.position === "absolute" && Math.abs(t.left - t.expectedLeft) <= 2 && Math.abs(t.top - t.expectedTop) <= 2)
        : measurement.filters.targets.every(t => t.position !== "absolute");
      measurements.push(measurement);
      await page.screenshot({ path: path.join(__dirname, `${phase}-${width}.png`), fullPage: true });
    }
    const desktopPass = measurements.filter(m => m.viewport > 860).every(m => m.headerPass && m.left01Pass && !m.overflow);
    const responsivePass = measurements.every(m => !m.overflow && (m.viewport > 860 || (m.headerFlowPass && m.left01Pass)));
    const source = fs.readFileSync(sourcePath, "utf8");
    const result = {
      task: "UI90-04B-LEFT-02",
      tipBase: "e6730551f8d04fd2cafd54b29019940237ea4375",
      phase,
      mode: "isolated Vue component; synthetic API; relative-to-header Left/right; LEFT-01 regression; not live host or SCORE; no 90% sign-off",
      sourceSha256: require("node:crypto").createHash("sha256").update(source).digest("hex"),
      measurements,
      desktopPass,
      responsivePass,
      errors,
    };
    fs.writeFileSync(path.join(__dirname, `${phase}-results.json`), JSON.stringify(result, null, 2) + "\n");
    console.log(JSON.stringify({
      phase,
      desktopPass,
      responsivePass,
      errors,
      summary: measurements.map(m => ({
        viewport: m.viewport,
        headerH: m.header.height,
        returnLeft: m.header.return.left,
        titleLeft: m.header.title.left,
        searchRight: m.header.search.right,
        exportRight: m.header.export.right,
        btnH: [m.header.return.height, m.header.search.height, m.header.export.height],
        filtersH: m.filters.height,
        ctrlW: m.filters.controls.map(c => c.width),
        headerPass: m.headerPass,
        left01Pass: m.left01Pass,
        overflow: m.overflow,
      })),
    }, null, 2));
    if (!desktopPass || !responsivePass || errors.length) process.exitCode = 1;
  } finally {
    if (browser) await browser.close();
    await server.close();
  }
})().catch(error => { console.error(error); process.exitCode = 1; });
