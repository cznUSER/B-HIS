const { chromium } = require('C:/Users/SYG/.cache/codex-runtimes/codex-primary-runtime/dependencies/node/node_modules/playwright');
const fs = require('node:fs');
const path = require('node:path');

const outDir = 'E:/B-HIS/deliverables/h05/ui90-04d-fix-02';
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
        tabNursingSelected: tabNursing.getAttribute('aria-selected') === 'false' ? false : tabNormal.getAttribute('aria-selected') === 'true',
        tabNursingSelectedRaw: tabNursing.getAttribute('aria-selected'),
        normalVisible: !panelNormal.hidden && getComputedStyle(panelNormal).display !== 'none',
        nursingVisible: !panelNursing.hidden && getComputedStyle(panelNursing).display !== 'none',
        fields,
        fieldCount: fields.length,
        saveW: Math.round(sr.width),
        saveH: Math.round(sr.height),
        cancelW: Math.round(cr.width),
        cancelH: Math.round(cr.height),
        tokenW: csForm.getPropertyValue('--dept-editor-w').trim(),
        overflow: document.documentElement.scrollWidth > innerWidth + 1,
        scrollWidth: document.documentElement.scrollWidth,
        innerWidth,
      };
    });
    // fix tabNursingSelected properly
    base.tabNursingSelected = await page.evaluate(() => document.querySelector('[data-testid="dept-tab-nursing"]').getAttribute('aria-selected') === 'true');

    await page.click('[data-testid="dept-tab-nursing"]');
    await page.waitForTimeout(40);

    const nursing = await page.evaluate(() => {
      const panelNormal = document.querySelector('[data-testid="dept-tabpanel-normal"]');
      const panelNursing = document.querySelector('[data-testid="dept-tabpanel-nursing"]');
      const tabNormal = document.querySelector('[data-testid="dept-tab-normal"]');
      const tabNursing = document.querySelector('[data-testid="dept-tab-nursing"]');
      const list = document.querySelector('[data-testid="dept-nursing-list"]');
      const add = document.querySelector('[data-testid="dept-nursing-add"]');
      const del = document.querySelector('[data-testid="dept-nursing-del"]');
      const note = document.querySelector('[data-testid="dept-nursing-note"]');
      const empty = document.querySelector('[data-testid="dept-nursing-empty"]');
      const lr = list.getBoundingClientRect();
      const ar = add.getBoundingClientRect();
      const dr = del.getBoundingClientRect();
      const cs = getComputedStyle(panelNursing);
      return {
        tabNormalSelected: tabNormal.getAttribute('aria-selected') === 'true',
        tabNursingSelected: tabNursing.getAttribute('aria-selected') === 'true',
        normalVisible: !panelNormal.hidden && getComputedStyle(panelNormal).display !== 'none',
        nursingVisible: !panelNursing.hidden && getComputedStyle(panelNursing).display !== 'none',
        listRole: list.getAttribute('role'),
        listH: Math.round(lr.height),
        listW: Math.round(lr.width),
        addText: add.textContent.trim(),
        delText: del.textContent.trim(),
        addW: Math.round(ar.width),
        addH: Math.round(ar.height),
        delW: Math.round(dr.width),
        delH: Math.round(dr.height),
        note: note?.textContent?.trim() || '',
        emptyText: empty?.textContent?.trim() || '',
        tokenListH: cs.getPropertyValue('--dept-nursing-list-h').trim(),
        hasTable: !!panelNursing.querySelector('table, .register-table'),
        placeholderGone: !document.querySelector('[data-testid="dept-nursing-placeholder"]'),
      };
    });

    // local thin add/del
    await page.click('[data-testid="dept-nursing-add"]');
    await page.waitForTimeout(30);
    await page.click('[data-testid="dept-nursing-add"]');
    await page.waitForTimeout(30);
    const afterAdd = await page.evaluate(() => ({
      count: document.querySelectorAll('[data-testid^="dept-nursing-item-"]').length,
      labels: [...document.querySelectorAll('[data-testid^="dept-nursing-item-"]')].map((el) => el.textContent.trim()),
      emptyGone: !document.querySelector('[data-testid="dept-nursing-empty"]'),
      state: window.__nursingState(),
    }));
    await page.click('[data-testid="dept-nursing-del"]');
    await page.waitForTimeout(30);
    const afterDel = await page.evaluate(() => ({
      count: document.querySelectorAll('[data-testid^="dept-nursing-item-"]').length,
      state: window.__nursingState(),
    }));

    await page.click('[data-testid="dept-tab-normal"]');
    await page.waitForTimeout(40);

    const shot = path.join(outDir, `after-${width}.png`);
    // switch nursing for screenshot of chrome
    await page.click('[data-testid="dept-tab-nursing"]');
    await page.waitForTimeout(40);
    await page.screenshot({ path: shot, fullPage: true });
    await page.click('[data-testid="dept-tab-normal"]');

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
      nursing.listRole === 'listbox' &&
      nursing.addText === '增加' &&
      nursing.delText === '删除' &&
      near(nursing.addW, 90, 2) &&
      near(nursing.addH, 30, 2) &&
      near(nursing.delW, 90, 2) &&
      near(nursing.delH, 30, 2) &&
      near(nursing.listH, 257, 4) &&
      (nursing.note || '').includes('未持久化') &&
      (nursing.emptyText || '').includes('未持久化') &&
      nursing.hasTable === false &&
      nursing.placeholderGone === true;
    const localPass =
      afterAdd.count === 2 &&
      afterAdd.labels.every((t) => t.includes('未持久化') || t.includes('mock')) &&
      afterDel.count === 1;
    const overflowPass = base.overflow === false;

    results.push({
      viewport: width,
      ...base,
      nursing,
      afterAdd,
      afterDel,
      checks: {
        widthPass,
        btnPass,
        tabPass,
        nursingPass,
        localPass,
        overflowPass,
        pass: widthPass && btnPass && tabPass && nursingPass && localPass && overflowPass,
      },
      shot,
    });
  }

  const allPass = results.every((r) => r.checks.pass) && pageerrors.length === 0;
  const summary = { allPass, pageerrors, results };
  fs.writeFileSync(path.join(outDir, 'after-results.json'), JSON.stringify(summary, null, 2), 'utf8');
  console.log(JSON.stringify({ allPass, pageerrors, viewports: results.map((r) => ({
    viewport: r.viewport,
    formW: r.formW,
    overflow: r.overflow,
    listH: r.nursing.listH,
    addW: r.nursing.addW,
    addH: r.nursing.addH,
    pass: r.checks.pass,
    checks: r.checks,
  })) }, null, 2));
  await browser.close();
  process.exit(allPass ? 0 : 2);
})().catch((err) => {
  console.error(err);
  process.exit(1);
});
