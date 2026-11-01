const { chromium } = require('C:/Users/SYG/.cache/codex-runtimes/codex-primary-runtime/dependencies/node/node_modules/playwright');
const fs = require('node:fs');
const path = require('node:path');

const outDir = 'E:/B-HIS/deliverables/h05/ui90-04f-fix-02';
const htmlPath = path.join(outDir, 'geometry-fixture.html');

function near(actual, expected, tol) {
  return Math.abs(actual - expected) <= tol;
}

(async () => {
  const browser = await chromium.launch({
    headless: true,
    executablePath: 'C:/Program Files (x86)/Microsoft/Edge/Application/msedge.exe',
  });
  const page = await browser.newPage();
  const pageerrors = [];
  page.on('pageerror', (e) => pageerrors.push(String(e.message || e)));

  const viewports = [1440, 1024, 390];
  const results = [];

  for (const width of viewports) {
    await page.setViewportSize({ width, height: 900 });
    await page.goto('file:///' + htmlPath.replace(/\\/g, '/'));
    await page.waitForSelector('[data-testid="order-page"]');
    await page.waitForSelector('[data-testid="order-line-scroll"]');
    await page.waitForSelector('[data-testid="clinical-order-history"]');
    await page.waitForSelector('[data-testid="clinical-order-actionbar"]');

    const m = await page.evaluate(() => {
      const pageEl = document.querySelector('[data-testid="order-page"]');
      const top = document.querySelector('[data-testid="order-top-chrome"]');
      const scroll = document.querySelector('[data-testid="order-line-scroll"]');
      const table = document.querySelector('[data-testid="order-line-table"]');
      const history = document.querySelector('[data-testid="clinical-order-history"]');
      const actionbar = document.querySelector('[data-testid="clinical-order-actionbar"]');
      const banner = document.querySelector('[data-testid="fixture-ui-only-banner"]');
      const layout = document.querySelector('[data-testid="order-form"]');
      const pr = pageEl.getBoundingClientRect();
      const tr = top.getBoundingClientRect();
      const sr = scroll.getBoundingClientRect();
      const hr = history.getBoundingClientRect();
      const ar = actionbar.getBoundingClientRect();
      const cs = getComputedStyle(pageEl);
      const tcs = getComputedStyle(table);
      const heads = [...table.querySelectorAll('thead th')].map((th) => th.textContent.trim()).filter(Boolean);
      return {
        pageW: Math.round(pr.width),
        topH: Math.round(tr.height),
        lineScrollH: Math.round(sr.height),
        lineScrollW: Math.round(sr.width),
        historyW: Math.round(hr.width),
        historyH: Math.round(hr.height),
        actionbarH: Math.round(ar.height),
        actionbarW: Math.round(ar.width),
        tableMinWidth: tcs.minWidth,
        tokenShell: cs.getPropertyValue('--order-shell-w').trim(),
        tokenTop: cs.getPropertyValue('--order-top-h').trim(),
        tokenLines: cs.getPropertyValue('--order-lines-h').trim(),
        tokenHistory: cs.getPropertyValue('--order-history-w').trim(),
        heads,
        bannerText: banner ? banner.textContent.trim() : '',
        layoutCols: getComputedStyle(layout).gridTemplateColumns,
        overflow: document.documentElement.scrollWidth > innerWidth + 1,
        scrollWidth: document.documentElement.scrollWidth,
        innerWidth,
      };
    });

    const shellTarget = width >= 1200 ? 1036 : null;
    const pagePass = shellTarget == null
      ? (m.pageW <= width && !m.overflow)
      : (near(m.pageW, 1036, 24) && !m.overflow);
    const topTol = width <= 400 ? 80 : 24;
    const topPass = width <= 400 ? (m.topH <= 105 + topTol) : near(m.topH, 105, topTol);
    const tokenPass = m.tokenShell === '1036px' && m.tokenTop === '105px' && m.tokenLines === '263px' && m.tokenHistory === '268px';
    const overflowPass = m.overflow === false;
    const linesTarget = width >= 1200 ? 263 : (width >= 1000 ? 220 : 180);
    const linesTol = width >= 1200 ? 48 : 80;
    const linesPass = m.lineScrollH >= linesTarget - linesTol;
    const historyPass = width >= 1280
      ? near(m.historyW, 268, 24)
      : (m.historyW > 0 && m.historyH >= 80);
    const actionPass = m.actionbarH >= 40 && m.actionbarW > 0;
    const minWidthPass = m.tableMinWidth === '1070px';
    const headsPass = m.heads.includes('项目') && m.heads.includes('数量') && m.heads.includes('用法');
    const bannerPass = /UI-only/.test(m.bannerText) && (/非活体/.test(m.bannerText) || /几何≠已真开单/.test(m.bannerText));
    const pass = pagePass && topPass && tokenPass && overflowPass && linesPass && historyPass && actionPass && minWidthPass && headsPass && bannerPass;

    const shot = path.join(outDir, `after-${width}.png`);
    await page.screenshot({ path: shot, fullPage: true });

    results.push({
      viewport: width,
      ...m,
      checks: {
        pagePass,
        topPass,
        tokenPass,
        overflowPass,
        linesPass,
        historyPass,
        actionPass,
        minWidthPass,
        headsPass,
        bannerPass,
      },
      pass,
      screenshot: shot,
    });
  }

  const allPass = results.every((r) => r.pass) && pageerrors.length === 0;
  const out = {
    tipBaseline: '1e3cf29',
    implementRef: '97462c0',
    slice: 'UI90-04F-FIX-02',
    note: 'isolated chrome fixture; non-live; UI-only; no save/commit/abate writes; not SCORE; not 90% sign-off; FIX-01 shell conserved; min-width:1070 kept; not BTN-WIRE',
    allPass,
    pageerrors,
    results,
  };
  fs.writeFileSync(path.join(outDir, 'after-results.json'), JSON.stringify(out, null, 2), 'utf8');
  console.log(JSON.stringify({
    allPass,
    pageerrors,
    summary: results.map((r) => ({
      viewport: r.viewport,
      pageW: r.pageW,
      topH: r.topH,
      lineScrollH: r.lineScrollH,
      historyW: r.historyW,
      actionbarH: r.actionbarH,
      tableMinWidth: r.tableMinWidth,
      overflow: r.overflow,
      pass: r.pass,
      checks: r.checks,
    })),
  }, null, 2));
  await browser.close();
  process.exit(allPass ? 0 : 1);
})().catch((err) => {
  console.error(err);
  process.exit(2);
});
