const { chromium } = require('C:/Users/SYG/.cache/codex-runtimes/codex-primary-runtime/dependencies/node/node_modules/playwright');
const fs = require('node:fs');
const path = require('node:path');

const outDir = 'E:/B-HIS/deliverables/h05/ui90-04d-fix-01';
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
    await page.waitForSelector('[data-testid="dept-form"]');

    // default = 常规
    const base = await page.evaluate(() => {
      const form = document.querySelector('[data-testid="dept-form"]');
      const tabs = document.querySelector('[data-testid="dept-editor-tabs"]');
      const tabNormal = document.querySelector('[data-testid="dept-tab-normal"]');
      const tabNursing = document.querySelector('[data-testid="dept-tab-nursing"]');
      const panelNormal = document.querySelector('[data-testid="dept-tabpanel-normal"]');
      const panelNursing = document.querySelector('[data-testid="dept-tabpanel-nursing"]');
      const save = document.querySelector('[data-testid="dept-save"]');
      const cancel = document.querySelector('[data-testid="dept-cancel"]');
      const fields = [...panelNormal.querySelectorAll('.field')].map((el) => el.querySelector('span')?.textContent?.trim() || '');
      const fr = form.getBoundingClientRect();
      const tr = tabs.getBoundingClientRect();
      const sr = save.getBoundingClientRect();
      const cr = cancel.getBoundingClientRect();
      const csForm = getComputedStyle(form);
      return {
        formW: Math.round(fr.width),
        formH: Math.round(fr.height),
        tabBarH: Math.round(tr.height),
        tabNormalText: tabNormal.textContent.trim(),
        tabNursingText: tabNursing.textContent.trim(),
        tabNormalSelected: tabNormal.getAttribute('aria-selected') === 'true',
        tabNursingSelected: tabNursing.getAttribute('aria-selected') === 'true',
        normalVisible: !panelNormal.hidden && getComputedStyle(panelNormal).display !== 'none',
        nursingVisible: !panelNursing.hidden && getComputedStyle(panelNursing).display !== 'none',
        fields,
        fieldCount: fields.length,
        saveW: Math.round(sr.width),
        saveH: Math.round(sr.height),
        cancelW: Math.round(cr.width),
        cancelH: Math.round(cr.height),
        tokenW: csForm.getPropertyValue('--dept-editor-w').trim(),
        tokenTabH: csForm.getPropertyValue('--dept-tab-h').trim(),
        overflow: document.documentElement.scrollWidth > innerWidth + 1,
        scrollWidth: document.documentElement.scrollWidth,
        innerWidth,
        nursingCrudButtons: [...panelNursing.querySelectorAll('button, [data-action], .text-btn')].length,
        nursingPlaceholder: document.querySelector('[data-testid="dept-nursing-placeholder"]')?.textContent?.trim() || '',
      };
    });

    // switch to nursing
    await page.click('[data-testid="dept-tab-nursing"]');
    await page.waitForTimeout(40);
    const nursing = await page.evaluate(() => {
      const panelNormal = document.querySelector('[data-testid="dept-tabpanel-normal"]');
      const panelNursing = document.querySelector('[data-testid="dept-tabpanel-nursing"]');
      const tabNormal = document.querySelector('[data-testid="dept-tab-normal"]');
      const tabNursing = document.querySelector('[data-testid="dept-tab-nursing"]');
      return {
        tabNormalSelected: tabNormal.getAttribute('aria-selected') === 'true',
        tabNursingSelected: tabNursing.getAttribute('aria-selected') === 'true',
        normalVisible: !panelNormal.hidden && getComputedStyle(panelNormal).display !== 'none',
        nursingVisible: !panelNursing.hidden && getComputedStyle(panelNursing).display !== 'none',
        nursingCrudButtons: [...panelNursing.querySelectorAll('button, [data-action], .text-btn')].length,
        nursingPlaceholder: document.querySelector('[data-testid="dept-nursing-placeholder"]')?.textContent?.trim() || '',
        nursingHasTable: !!panelNursing.querySelector('table, .register-table, [data-testid*="nursing-list"]'),
      };
    });

    // back to normal
    await page.click('[data-testid="dept-tab-normal"]');
    await page.waitForTimeout(40);

    const shot = path.join(outDir, `after-${width}.png`);
    await page.screenshot({ path: shot, fullPage: true });

    const widthPass = width >= 861 ? near(base.formW, 408, 2) : (base.formW <= 408 && base.overflow === false);
    const btnPass = near(base.saveW, 90, 2) && near(base.saveH, 30, 2) && near(base.cancelW, 90, 2) && near(base.cancelH, 30, 2);
    const tabPass =
      base.tabNormalText === '常规' &&
      base.tabNursingText === '护理单元包含科室' &&
      base.tabNormalSelected === true &&
      base.tabNursingSelected === false &&
      base.normalVisible === true &&
      base.nursingVisible === false &&
      base.fieldCount === 4 &&
      near(base.tabBarH, 30, 2);
    const nursingPass =
      nursing.tabNursingSelected === true &&
      nursing.nursingVisible === true &&
      nursing.normalVisible === false &&
      nursing.nursingCrudButtons === 0 &&
      nursing.nursingHasTable === false &&
      (nursing.nursingPlaceholder || '').includes('未实现');
    const overflowPass = base.overflow === false;

    results.push({
      viewport: width,
      ...base,
      nursing,
      widthPass,
      btnPass,
      tabPass,
      nursingPass,
      overflowPass,
      shot: path.basename(shot),
    });
  }

  await browser.close();

  const summary = {
    tipBaseline: '7f9c4fd',
    task: 'UI90-04D-FIX-01',
    pageerrors,
    results,
    allPass: results.every((r) => r.widthPass && r.btnPass && r.tabPass && r.nursingPass && r.overflowPass) && pageerrors.length === 0,
    no90SignOff: true,
    noScore: true,
    noPrelive: true,
  };
  fs.writeFileSync(path.join(outDir, 'after-results.json'), JSON.stringify(summary, null, 2), 'utf8');
  console.log(JSON.stringify(summary, null, 2));
  process.exit(summary.allPass ? 0 : 2);
})().catch((err) => {
  console.error(err);
  process.exit(1);
});
