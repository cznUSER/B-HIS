const { chromium } = require('C:/Users/SYG/.cache/codex-runtimes/codex-primary-runtime/dependencies/node/node_modules/playwright');
const fs = require('node:fs');
const path = require('node:path');

const root = __dirname;
const fixture = path.join(root, 'geometry-fixture.html');
const edge = 'C:/Program Files (x86)/Microsoft/Edge/Application/msedge.exe';

(async () => {
  const browser = await chromium.launch({ headless: true, executablePath: edge });
  const page = await browser.newPage();
  const fileUrl = 'file:///' + fixture.replace(/\\/g, '/');
  const viewports = [1440, 1024, 390];
  const measurements = [];

  for (const width of viewports) {
    await page.setViewportSize({ width, height: 900 });
    await page.goto(fileUrl);
    await page.waitForTimeout(200);
    const m = await page.evaluate(() => {
      const header = document.getElementById('header');
      const filters = document.getElementById('filters');
      const inputs = [...filters.querySelectorAll('input, select')];
      const button = filters.querySelector('button');
      const hr = header.getBoundingClientRect();
      const fr = filters.getBoundingClientRect();
      return {
        viewport: innerWidth,
        header: { h: Math.round(hr.height * 1000) / 1000, bottom: hr.bottom },
        filters: { h: Math.round(fr.height * 1000) / 1000, top: fr.top, y: fr.y },
        gridStartApprox: Math.round(fr.bottom * 1000) / 1000,
        headerToFilterGap: Math.round((fr.top - hr.bottom) * 1000) / 1000,
        controls: {
          inputHeights: inputs.map((el) => Math.round(el.getBoundingClientRect().height * 1000) / 1000),
          buttonHeight: Math.round(button.getBoundingClientRect().height * 1000) / 1000,
        },
        overlap: fr.top < hr.bottom - 0.5,
        overflow: document.documentElement.scrollWidth > innerWidth + 1,
        tableFont: getComputedStyle(document.querySelector('.register-table')).font,
        labelFont: getComputedStyle(filters.querySelector('.field > span')).font,
      };
    });
    measurements.push(m);
    await page.screenshot({ path: path.join(root, `geometry-${width}.png`), fullPage: true });
  }

  const targets = { header: 45, filters: 81, input: 25, button: 30, gridTopFromHeaderPlusFilters: 126 };
  const desktop = measurements.filter((x) => x.viewport >= 1024);
  const desktopPass = desktop.every((x) =>
    Math.abs(x.header.h - targets.header) <= 2 &&
    Math.abs(x.filters.h - targets.filters) <= 2 &&
    x.controls.inputHeights.every((h) => Math.abs(h - targets.input) <= 2) &&
    Math.abs(x.controls.buttonHeight - targets.button) <= 2 &&
    !x.overflow &&
    !x.overlap
  );
  const mobile = measurements.find((x) => x.viewport === 390);
  const out = {
    mode: 'tip-CSS isolated fixture (not full AppShell); source tip 2b69533 UserOperationsLogPanel scoped CSS',
    tip: '2b69533',
    targets,
    measurements,
    desktopGeometryPassWithin2px: desktopPass,
    mobileNote: mobile ? {
      headerH: mobile.header.h,
      filtersH: mobile.filters.h,
      overflow: mobile.overflow,
      overlap: mobile.overlap,
      remark: 'DFM has no mobile layout; 520px media sets header height:auto / single-column filters',
    } : null,
    errors: [],
  };
  fs.writeFileSync(path.join(root, 'geometry-probe-results.json'), JSON.stringify(out, null, 2));
  console.log('GEOMETRY_PROBE', JSON.stringify({
    desktopPass,
    viewports: measurements.map((x) => [x.viewport, x.header.h, x.filters.h, x.controls.buttonHeight, x.overflow]),
  }));
  await browser.close();
  if (!desktopPass) process.exitCode = 1;
})().catch((e) => { console.error(e); process.exitCode = 1; });

