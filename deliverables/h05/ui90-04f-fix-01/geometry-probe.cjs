const { chromium } = require('C:/Users/SYG/.cache/codex-runtimes/codex-primary-runtime/dependencies/node/node_modules/playwright');
const fs = require('node:fs');
const path = require('node:path');

const outDir = 'E:/B-HIS/deliverables/h05/ui90-04f-fix-01';
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

    const m = await page.evaluate(() => {
      const pageEl = document.querySelector('[data-testid="order-page"]');
      const top = document.querySelector('[data-testid="order-top-chrome"]');
      const editor = document.querySelector('[data-testid="clinical-order-editor"]');
      const fields = document.querySelector('[data-testid="order-fields"]');
      const pr = pageEl.getBoundingClientRect();
      const tr = top.getBoundingClientRect();
      const er = editor.getBoundingClientRect();
      const cs = getComputedStyle(pageEl);
      const ecs = getComputedStyle(editor);
      const fcs = getComputedStyle(fields);
      return {
        pageW: Math.round(pr.width),
        pageH: Math.round(pr.height),
        topH: Math.round(tr.height),
        editorW: Math.round(er.width),
        editorPadTop: parseFloat(ecs.paddingTop) || 0,
        editorPadInline: parseFloat(ecs.paddingLeft) || 0,
        fieldsGap: fcs.gap || fcs.rowGap || '',
        tokenShell: cs.getPropertyValue('--order-shell-w').trim(),
        tokenTop: cs.getPropertyValue('--order-top-h').trim(),
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
    const tokenPass = m.tokenShell === '1036px' && m.tokenTop === '105px';
    const overflowPass = m.overflow === false;
    const pass = pagePass && topPass && tokenPass && overflowPass;

    const shot = path.join(outDir, `after-${width}.png`);
    await page.screenshot({ path: shot, fullPage: true });

    results.push({
      viewport: width,
      ...m,
      checks: { pagePass, topPass, tokenPass, overflowPass },
      pass,
      screenshot: shot,
    });
  }

  const allPass = results.every((r) => r.pass) && pageerrors.length === 0;
  const out = {
    tipBaseline: 'ec0106b',
    slice: 'UI90-04F-FIX-01',
    note: 'isolated chrome fixture; non-live; no save/commit/abate writes; not SCORE; not 90% sign-off; not FIX-02/BTN-WIRE',
    allPass,
    pageerrors,
    results,
  };
  fs.writeFileSync(path.join(outDir, 'after-results.json'), JSON.stringify(out, null, 2));
  console.log(JSON.stringify({
    allPass,
    results: results.map((r) => ({
      viewport: r.viewport,
      pageW: r.pageW,
      topH: r.topH,
      overflow: r.overflow,
      tokenShell: r.tokenShell,
      tokenTop: r.tokenTop,
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
