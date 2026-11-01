const { chromium } = require('C:/Users/SYG/.cache/codex-runtimes/codex-primary-runtime/dependencies/node/node_modules/playwright');
const fs = require('node:fs');
const path = require('node:path');

const outDir = 'E:/B-HIS/deliverables/h05/ui90-04d-fix-03';
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
    await page.setViewportSize({ width, height: 1100 });
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
      const hint = document.querySelector('[data-testid="dept-extra-fields-hint"]');
      const phone = document.querySelector('[data-testid="dept-phone"]');
      const manager = document.querySelector('[data-testid="dept-manager"]');
      const inputCode = document.querySelector('[data-testid="dept-input-code"]');
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
        normalVisible: !panelNormal.hidden && getComputedStyle(panelNormal).display !== 'none',
        nursingVisible: !panelNursing.hidden && getComputedStyle(panelNursing).display !== 'none',
        fields,
        fieldCount: fields.length,
        hasPhone: !!phone,
        hasManager: !!manager,
        hasInputCode: !!inputCode,
        hintText: hint?.textContent?.trim() || '',
        saveW: Math.round(sr.width),
        saveH: Math.round(sr.height),
        cancelW: Math.round(cr.width),
        cancelH: Math.round(cr.height),
        tokenW: csForm.getPropertyValue('--dept-editor-w').trim(),
        overflow: document.documentElement.scrollWidth > innerWidth + 1,
        scrollWidth: document.documentElement.scrollWidth,
        innerWidth,
        savePayloadKeys: window.__savePayloadKeys(),
      };
    });
    base.tabNursingSelected = await page.evaluate(() => document.querySelector('[data-testid="dept-tab-nursing"]').getAttribute('aria-selected') === 'true');

    // fill extras then cancel discard
    await page.fill('[data-testid="dept-phone"]', '13800000000');
    await page.fill('[data-testid="dept-manager"]', '张三');
    await page.fill('[data-testid="dept-input-code"]', 'KS');
    const filled = await page.evaluate(() => window.__extraState());
    await page.click('[data-testid="dept-cancel"]');
    await page.waitForTimeout(40);
    const discarded = await page.evaluate(() => window.__extraState());

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
      const lr = list.getBoundingClientRect();
      const ar = add.getBoundingClientRect();
      const dr = del.getBoundingClientRect();
      return {
        tabNormalSelected: tabNormal.getAttribute('aria-selected') === 'true',
        tabNursingSelected: tabNursing.getAttribute('aria-selected') === 'true',
        normalVisible: !panelNormal.hidden && getComputedStyle(panelNormal).display !== 'none',
        nursingVisible: !panelNursing.hidden && getComputedStyle(panelNursing).display !== 'none',
        listRole: list.getAttribute('role'),
        listH: Math.round(lr.height),
        addText: add.textContent.trim(),
        delText: del.textContent.trim(),
        addW: Math.round(ar.width),
        addH: Math.round(ar.height),
        delW: Math.round(dr.width),
        delH: Math.round(dr.height),
        note: note?.textContent?.trim() || '',
      };
    });

    await page.click('[data-testid="dept-nursing-add"]');
    await page.waitForTimeout(30);
    const afterAdd = await page.evaluate(() => ({
      count: document.querySelectorAll('[data-testid^="dept-nursing-item-"]').length,
      state: window.__nursingState(),
    }));

    const shot = path.join(outDir, `after-${width}.png`);
    await page.click('[data-testid="dept-tab-normal"]');
    await page.waitForTimeout(40);
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
      near(base.tabBarH, 30, 2);
    const fieldsPass =
      base.fieldCount === 7 &&
      base.fields.includes('电话号码') &&
      base.fields.includes('负责人') &&
      base.fields.includes('助记码') &&
      base.hasPhone && base.hasManager && base.hasInputCode &&
      (base.hintText || '').includes('未持久化') &&
      (base.hintText || '').includes('不写入 saveDepartment') &&
      JSON.stringify(base.savePayloadKeys) === JSON.stringify(['deptCode','deptName','status','sortNo']);
    const discardPass =
      filled.phone === '13800000000' &&
      filled.manager === '张三' &&
      filled.inputCode === 'KS' &&
      discarded.phone === '' &&
      discarded.manager === '' &&
      discarded.inputCode === '';
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
      afterAdd.count === 1;
    const overflowPass = base.overflow === false;

    results.push({
      viewport: width,
      ...base,
      filled,
      discarded,
      nursing,
      afterAdd,
      checks: {
        widthPass,
        btnPass,
        tabPass,
        fieldsPass,
        discardPass,
        nursingPass,
        overflowPass,
        pass: widthPass && btnPass && tabPass && fieldsPass && discardPass && nursingPass && overflowPass,
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
    formH: r.formH,
    overflow: r.overflow,
    fieldCount: r.fieldCount,
    fields: r.fields,
    savePayloadKeys: r.savePayloadKeys,
    listH: r.nursing.listH,
    saveW: r.saveW,
    saveH: r.saveH,
    pass: r.checks.pass,
    checks: r.checks,
  })) }, null, 2));
  await browser.close();
  process.exit(allPass ? 0 : 2);
})().catch((err) => {
  console.error(err);
  process.exit(1);
});
