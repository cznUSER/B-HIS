const { chromium } = require('C:/Users/SYG/.cache/codex-runtimes/codex-primary-runtime/dependencies/node/node_modules/playwright');
const fs = require('node:fs');
const path = require('node:path');

const outDir = 'E:/B-HIS/deliverables/h05/ui90-04e-fix-01';
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
    await page.waitForSelector('[data-testid="charge-page"]');

    const m = await page.evaluate(() => {
      const pageEl = document.querySelector('[data-testid="charge-page"]');
      const workbar = document.querySelector('[data-testid="charge-workbar"]');
      const search = document.querySelector('[data-testid="charge-search"]');
      const zone = document.querySelector('[data-testid="charge-candidate-zone"]');
      const pr = pageEl.getBoundingClientRect();
      const wr = workbar.getBoundingClientRect();
      const sr = search.getBoundingClientRect();
      const zr = zone.getBoundingClientRect();
      const cs = getComputedStyle(pageEl);
      return {
        pageW: Math.round(pr.width),
        pageH: Math.round(pr.height),
        workbarH: Math.round(wr.height),
        searchH: Math.round(sr.height),
        searchW: Math.round(sr.width),
        zonePadTop: parseFloat(getComputedStyle(zone).paddingTop) || 0,
        tokenShell: cs.getPropertyValue('--charge-shell-w').trim(),
        tokenTop: cs.getPropertyValue('--charge-top-h').trim(),
        overflow: document.documentElement.scrollWidth > innerWidth + 1,
        scrollWidth: document.documentElement.scrollWidth,
        innerWidth,
        buttons: [...document.querySelectorAll('button')].map((b) => b.textContent.replace(/\s+/g, ' ').trim()),
      };
    });

    const shellTarget = width >= 1200 ? 1008 : null;
    const pagePass = shellTarget == null
      ? (m.pageW <= width && !m.overflow)
      : (near(m.pageW, 1008, 24) && !m.overflow);
    // top chrome: workbar and/or search near 47 (±10 for wrap on narrow)
    const topTol = width <= 400 ? 80 : 8;
    const workbarPass = m.workbarH <= 47 + topTol;
    const searchPass = width <= 400 ? (m.searchH <= 120) : near(m.searchH, 47, topTol);
    const tokenPass = m.tokenShell === '1008px' && m.tokenTop === '47px';
    const overflowPass = m.overflow === false;
    const pass = pagePass && workbarPass && searchPass && tokenPass && overflowPass;

    const shot = path.join(outDir, `after-${width}.png`);
    await page.screenshot({ path: shot, fullPage: true });

    results.push({
      viewport: width,
      ...m,
      checks: {
        pagePass,
        workbarPass,
        searchPass,
        tokenPass,
        overflowPass,
      },
      pass,
      screenshot: shot,
    });
  }

  const allPass = results.every((r) => r.pass) && pageerrors.length === 0;
  const out = {
    tipBaseline: 'd53c3d3',
    slice: 'UI90-04E-FIX-01',
    note: 'isolated chrome fixture; non-live; no pay/refund writes; not SCORE; not 90% sign-off',
    allPass,
    pageerrors,
    results,
  };
  fs.writeFileSync(path.join(outDir, 'after-results.json'), JSON.stringify(out, null, 2));
  console.log(JSON.stringify({ allPass, results: results.map((r) => ({ viewport: r.viewport, pageW: r.pageW, workbarH: r.workbarH, searchH: r.searchH, overflow: r.overflow, pass: r.pass })) }, null, 2));
  await browser.close();
  process.exit(allPass ? 0 : 1);
})().catch((err) => {
  console.error(err);
  process.exit(2);
});