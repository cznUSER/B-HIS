const { chromium } = require('C:/Users/SYG/.cache/codex-runtimes/codex-primary-runtime/dependencies/node/node_modules/playwright');
const fs = require('node:fs');
const path = require('node:path');

const outDir = 'E:/B-HIS/deliverables/h05/ui90-04e-fix-02';
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
    await page.waitForSelector('[data-testid="charge-preview"]');
    await page.waitForSelector('[data-testid="charge-pay"]');

    const m = await page.evaluate(() => {
      const pageEl = document.querySelector('[data-testid="charge-page"]');
      const workbar = document.querySelector('[data-testid="charge-workbar"]');
      const search = document.querySelector('[data-testid="charge-search"]');
      const preview = document.querySelector('[data-testid="charge-preview"]');
      const wrap = document.querySelector('[data-testid="charge-pending-scroll-wrap"]');
      const pay = document.querySelector('[data-testid="charge-pay"]');
      const workspace = document.querySelector('[data-testid="charge-workspace"]');
      const table = document.querySelector('[data-testid="charge-pending"]');
      const banner = document.querySelector('[data-testid="fixture-ui-only-banner"]');
      const pr = pageEl.getBoundingClientRect();
      const wr = workbar.getBoundingClientRect();
      const sr = search.getBoundingClientRect();
      const pvr = preview.getBoundingClientRect();
      const wrapR = wrap.getBoundingClientRect();
      const payR = pay.getBoundingClientRect();
      const wsR = workspace.getBoundingClientRect();
      const cs = getComputedStyle(pageEl);
      const heads = [...table.querySelectorAll('thead th')].map((th) => th.textContent.trim());
      return {
        pageW: Math.round(pr.width),
        workbarH: Math.round(wr.height),
        searchH: Math.round(sr.height),
        previewH: Math.round(pvr.height),
        wrapH: Math.round(wrapR.height),
        settleW: Math.round(payR.width),
        settleH: Math.round(payR.height),
        workspaceW: Math.round(wsR.width),
        tokenShell: cs.getPropertyValue('--charge-shell-w').trim(),
        tokenTop: cs.getPropertyValue('--charge-top-h').trim(),
        tokenPreview: cs.getPropertyValue('--charge-preview-h').trim(),
        tokenSettle: cs.getPropertyValue('--charge-settle-w').trim(),
        heads,
        bannerText: banner ? banner.textContent.trim() : '',
        overflow: document.documentElement.scrollWidth > innerWidth + 1,
        scrollWidth: document.documentElement.scrollWidth,
        innerWidth,
      };
    });

    const shellTarget = width >= 1200 ? 1008 : null;
    const pagePass = shellTarget == null
      ? (m.pageW <= width && !m.overflow)
      : (near(m.pageW, 1008, 24) && !m.overflow);
    const topTol = width <= 400 ? 80 : 8;
    const workbarPass = m.workbarH <= 47 + topTol;
    const searchPass = width <= 400 ? (m.searchH <= 120) : near(m.searchH, 47, topTol);
    const tokenPass = m.tokenShell === '1008px' && m.tokenTop === '47px' && m.tokenPreview === '329px';
    const overflowPass = m.overflow === false;
    // Desktop preview wrap height ~329 (tol 48); narrower allows shrink
    const previewTarget = width >= 1200 ? 329 : (width >= 1000 ? 240 : 200);
    const previewTol = width >= 1200 ? 48 : 80;
    const previewPass = m.wrapH >= previewTarget - previewTol;
    const settlePass = width >= 1200
      ? (m.settleW >= 280 && m.settleW <= 380 && m.settleH >= 220)
      : (m.settleW > 0 && m.settleH >= 180);
    const headsPass = JSON.stringify(m.heads) === JSON.stringify(['项目','数量','单价','金额','库存']);
    const bannerPass = /UI-only/.test(m.bannerText) && /非活体|非已支付/.test(m.bannerText);
    const pass = pagePass && workbarPass && searchPass && tokenPass && overflowPass && previewPass && settlePass && headsPass && bannerPass;

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
        previewPass,
        settlePass,
        headsPass,
        bannerPass,
      },
      pass,
      screenshot: shot,
    });
  }

  const allPass = results.every((r) => r.pass) && pageerrors.length === 0;
  const out = {
    tipBaseline: 'af35e66',
    implementRef: '6d44910',
    slice: 'UI90-04E-FIX-02',
    note: 'isolated chrome fixture; non-live; UI-only skeleton; no pay/refund writes; not SCORE; not 90% sign-off; FIX-01 shell conserved',
    allPass,
    pageerrors,
    results,
  };
  fs.writeFileSync(path.join(outDir, 'after-results.json'), JSON.stringify(out, null, 2), 'utf8');
  console.log(JSON.stringify({ allPass, pageerrors, summary: results.map((r) => ({
    viewport: r.viewport,
    pageW: r.pageW,
    workbarH: r.workbarH,
    wrapH: r.wrapH,
    settleW: r.settleW,
    settleH: r.settleH,
    overflow: r.overflow,
    pass: r.pass,
    checks: r.checks,
  })) }, null, 2));
  await browser.close();
  process.exit(allPass ? 0 : 1);
})().catch((err) => {
  console.error(err);
  process.exit(2);
});