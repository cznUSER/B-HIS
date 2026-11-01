const fs = require("node:fs");
const path = require("node:path");
const { pathToFileURL } = require("node:url");
const { chromium } = require("C:/Users/SYG/.cache/codex-runtimes/codex-primary-runtime/dependencies/node/node_modules/playwright");
const root = path.resolve(__dirname, "../../..");
const sourcePath = path.join(root, "web/src/views/access/UserOperationsLogPanel.vue");
const phase = process.argv[2] || "after";

const SAMPLE_ITEM = {
  moduleId: "module-access",
  moduleName: "权限管理",
  logType: "查询",
  occurredAt: "2026-11-01T08:30:00+08:00",
  actorUserId: "u001",
  actorUsername: "测试用户",
  departmentId: "信息科",
  description: "查询用户操作日志用于 LEFT-04 列宽验收，内容略长以触发 ellipsis。",
  clientIp: "192.168.1.10",
  clientMac: "AA-BB-CC-DD-EE-FF",
  buildVersion: "2.0.1",
  requestId: "req-left04-001",
  outcome: "SUCCESS",
  errorCode: ""
};

const COL_EXPECT = [
  ["module", ".uol-col-module", 102],
  ["type", ".uol-col-type", 76],
  ["time", ".uol-col-time", 123],
  ["user", ".uol-col-user", 67],
  ["dept", ".uol-col-dept", 67],
  ["content", ".uol-col-content", 324],
  ["ip", ".uol-col-ip", 87],
  ["mac", ".uol-col-mac", 85],
  ["version", ".uol-col-version", 75],
];

(async () => {
  const { createServer } = await import(pathToFileURL(path.join(root, "web/node_modules/vite/dist/node/index.js")));
  const { default: vue } = await import(pathToFileURL(path.join(root, "web/node_modules/@vitejs/plugin-vue/dist/index.mjs")));
  const server = await createServer({
    configFile: false,
    root,
    plugins: [vue()],
    cacheDir: path.join(require("node:os").tmpdir(), "uol-left04-vite"),
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
    
    await page.route("**/api/v1/**", async route => {
      const url = route.request().url();
      if (url.includes("user-operation") || url.includes("/audit/")) {
        return route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify({ items: [SAMPLE_ITEM], nextCursor: "" }) });
      }
      if (url.includes("module")) {
        return route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify({ items: [{ id: 1, name: "权限管理" }] }) });
      }
      if (url.includes("users")) {
        return route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify({ items: [{ userId: "u001", username: "tester", displayName: "测试用户" }] }) });
      }
      return route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify({ items: [], nextCursor: "" }) });
    });
    const measurements = [];
    for (const width of [1440, 1280, 1024, 1008, 861, 860, 520, 390]) {
      await page.setViewportSize({ width, height: 900 });
      await page.goto("http://127.0.0.1:" + server.httpServer.address().port + "/deliverables/h05/ui90-04b-left-04/geometry-fixture.html");
      await page.locator(".user-operations-filters").waitFor();
      await page.locator(".user-operations-results, .result-empty, .shell-status, .banner").first().waitFor({ timeout: 15000 });
      await page.evaluate(() => document.fonts.ready);
      // Prefer table; if empty state, still measure shell
      const hasTable = await page.locator("table.user-operations-table, table.register-table").count();
      if (!hasTable && width >= 861) {
        // retry once after short wait for load
        await page.waitForTimeout(500);
      }
      const measurement = await page.evaluate((colExpect) => {
        const panel = document.querySelector(".user-operations-panel");
        const header = document.querySelector(".master-panel-header");
        const filters = document.querySelector(".user-operations-filters");
        const results = document.querySelector(".user-operations-results") || document.querySelector(".result-empty") || document.querySelector(".shell-status");
        const table = document.querySelector("table.user-operations-table") || document.querySelector("table.register-table");
        const panelBounds = panel.getBoundingClientRect();
        const headerBounds = header.getBoundingClientRect();
        const filtersBounds = filters.getBoundingClientRect();
        const resultsBounds = results.getBoundingClientRect();
        const panelStyle = getComputedStyle(panel);
        const headerStyle = getComputedStyle(header);
        const filtersStyle = getComputedStyle(filters);
        const returnEl = document.querySelector(".uol-hdr-return");
        const titleEl = document.querySelector(".master-panel-copy");
        const searchEl = document.querySelector(".uol-hdr-search");
        const exportEl = document.querySelector(".uol-hdr-export");
        const relHeader = (el) => {
          const b = el.getBoundingClientRect();
          return {
            left: b.left - headerBounds.left,
            top: b.top - headerBounds.top,
            right: headerBounds.right - b.right,
            width: b.width,
            height: b.height,
            position: getComputedStyle(el).position,
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
          };
        });
        const controls = [".user-operations-filters select", ".user-operations-user-button-edit", "input[aria-label='记录日期起']", "input[aria-label='记录日期止']"].map(selector => {
          const bounds = document.querySelector(selector).getBoundingClientRect();
          return { selector, width: bounds.width, height: bounds.height };
        });
        const cols = colExpect.map(([name, selector, expected]) => {
          const th = table ? table.querySelector("thead th" + selector) : null;
          const td = table ? table.querySelector("tbody td" + selector) : null;
          const thB = th ? th.getBoundingClientRect() : null;
          const tdB = td ? td.getBoundingClientRect() : null;
          return {
            name,
            expected,
            thWidth: thB ? thB.width : null,
            tdWidth: tdB ? tdB.width : null,
            thText: th ? th.textContent.trim() : null,
            present: Boolean(th),
          };
        });
        const ext = ["outcome", "request", "error"].map(name => {
          const selector = ".uol-col-ext-" + name;
          const th = table ? table.querySelector("thead th" + selector) : null;
          const td = table ? table.querySelector("tbody td" + selector) : null;
          return {
            name,
            thPresent: Boolean(th),
            tdPresent: Boolean(td),
            thText: th ? th.textContent.trim() : null,
            thWidth: th ? th.getBoundingClientRect().width : null,
          };
        });
        const resultsStyle = getComputedStyle(results);
        const pageOverflow = document.documentElement.scrollWidth > innerWidth + 1;
        const resultsScrollable = results.scrollWidth > results.clientWidth + 1;
        return {
          viewport: innerWidth,
          shell: {
            token: panelStyle.getPropertyValue("--uol-shell-w").trim(),
            colTokens: {
              module: panelStyle.getPropertyValue("--uol-col-w-module").trim(),
              type: panelStyle.getPropertyValue("--uol-col-w-type").trim(),
              time: panelStyle.getPropertyValue("--uol-col-w-time").trim(),
              user: panelStyle.getPropertyValue("--uol-col-w-user").trim(),
              dept: panelStyle.getPropertyValue("--uol-col-w-dept").trim(),
              content: panelStyle.getPropertyValue("--uol-col-w-content").trim(),
              ip: panelStyle.getPropertyValue("--uol-col-w-ip").trim(),
              mac: panelStyle.getPropertyValue("--uol-col-w-mac").trim(),
              version: panelStyle.getPropertyValue("--uol-col-w-version").trim(),
            },
            panelWidth: panelBounds.width,
            panelClientWidth: panel.clientWidth,
            panelBoxSizing: panelStyle.boxSizing,
            resultsSelector: results.className,
            resultsWidth: resultsBounds.width,
            resultsLeft: resultsBounds.left - panelBounds.left,
            resultsOverflowX: resultsStyle.overflowX,
            resultsScrollable,
            headerWidth: headerBounds.width,
            filtersWidth: filtersBounds.width,
            stackHeight: headerBounds.height + filtersBounds.height,
            headerHeight: headerBounds.height,
            filtersHeight: filtersBounds.height,
          },
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
            return: relHeader(returnEl),
            title: relHeader(titleEl),
            search: relHeader(searchEl),
            export: relHeader(exportEl),
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
          columns: cols,
          extensionColumns: ext,
          tablePresent: Boolean(table),
          overflow: pageOverflow,
        };
      }, COL_EXPECT);

      const desktopChrome = measurement.viewport > 860;
      const shellDesktop = measurement.viewport >= 1008;
      const hdr = measurement.header;
      const shell = measurement.shell;
      measurement.shellPass = shellDesktop
        ? shell.token === "1008px"
          && Math.abs(shell.panelWidth - 1008) <= 2
          && Math.abs(shell.resultsWidth - shell.panelClientWidth) <= 2
          && Math.abs(shell.resultsLeft) <= 2
          && Math.abs(shell.stackHeight - 126) <= 4
        : true;
      measurement.headerPass = desktopChrome
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
      measurement.headerFlowPass = !desktopChrome
        ? hdr.return.position !== "absolute" && hdr.search.position !== "absolute" && hdr.export.position !== "absolute"
        : true;
      measurement.left01Pass = desktopChrome
        ? measurement.filters.position === "relative"
          && measurement.filters.tokens.filtersHeight === "81px"
          && measurement.filters.tokens.controlWidth === "196px"
          && measurement.filters.tokens.row1 === "6px"
          && measurement.filters.tokens.row2 === "37px"
          && Math.abs(measurement.filters.height - 81) <= 2
          && measurement.filters.controls.every(c => Math.abs(c.width - 196) <= 2)
          && measurement.filters.targets.every(t => t.position === "absolute" && Math.abs(t.left - t.expectedLeft) <= 2 && Math.abs(t.top - t.expectedTop) <= 2)
        : measurement.filters.targets.every(t => t.position !== "absolute");
      measurement.colsPass = desktopChrome
        ? measurement.tablePresent
          && measurement.columns.every(c => c.present && c.thWidth != null && Math.abs(c.thWidth - c.expected) <= 2)
          && measurement.extensionColumns.every(c => c.thPresent && c.tdPresent)
        : true;
      measurements.push(measurement);
      await page.screenshot({ path: path.join(__dirname, phase + "-" + width + ".png"), fullPage: true });
    }
    const desktopPass = measurements.filter(m => m.viewport > 860).every(m => m.headerPass && m.left01Pass && m.colsPass && !m.overflow);
    const shellPass = measurements.filter(m => m.viewport >= 1008).every(m => m.shellPass && !m.overflow);
    const colsPass = measurements.filter(m => m.viewport > 860).every(m => m.colsPass);
    const responsivePass = measurements.every(m => !m.overflow && (m.viewport > 860 || (m.headerFlowPass && m.left01Pass)));
    const source = fs.readFileSync(sourcePath, "utf8");
    const result = {
      task: "UI90-04B-LEFT-04",
      tipBase: "3954ad3fa4a934122651876374d4652925a778c4",
      phase,
      mode: "isolated Vue component; synthetic API with one row; DFM nine col Width tokens; LEFT-01/02/03 regression; not live host or SCORE; no 90% sign-off",
      measureEdge: "th getBoundingClientRect().width = border-box; ColumnAutoWidth=True still accepted by declared Width ±2; extension cols remain auto; page overflow forbidden, results may scroll-x",
      sourceSha256: require("node:crypto").createHash("sha256").update(source).digest("hex"),
      measurements,
      desktopPass,
      shellPass,
      colsPass,
      responsivePass,
      errors,
    };
    fs.writeFileSync(path.join(__dirname, phase + "-results.json"), JSON.stringify(result, null, 2) + String.fromCharCode(10));
    console.log(JSON.stringify({
      phase,
      desktopPass,
      shellPass,
      colsPass,
      responsivePass,
      errors,
      summary: measurements.map(m => ({
        viewport: m.viewport,
        panelW: m.shell.panelWidth,
        resultsLeft: m.shell.resultsLeft,
        stackH: m.shell.stackHeight,
        cols: (m.columns || []).map(c => ({ name: c.name, th: c.thWidth, exp: c.expected })),
        ext: (m.extensionColumns || []).map(c => ({ name: c.name, th: c.thText, w: c.thWidth, present: c.thPresent })),
        tablePresent: m.tablePresent,
        shellPass: m.shellPass,
        headerPass: m.headerPass,
        left01Pass: m.left01Pass,
        colsPass: m.colsPass,
        overflow: m.overflow,
        resultsScrollable: m.shell.resultsScrollable,
      })),
    }, null, 2));
    if (!desktopPass || !shellPass || !colsPass || !responsivePass || errors.length) process.exitCode = 1;
  } finally {
    if (browser) await browser.close();
    await server.close();
  }
})().catch(error => { console.error(error); process.exitCode = 1; });
