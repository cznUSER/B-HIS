const { chromium } = require('C:/Users/SYG/.cache/codex-runtimes/codex-primary-runtime/dependencies/node/node_modules/playwright');
const fs = require('node:fs');
const path = require('node:path');

const outDir = 'E:/B-HIS/deliverables/h05/ui90-04d-fix-03b';
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
    await page.setViewportSize({ width, height: 1200 });
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
      const autoCode = document.querySelector('[data-testid="dept-auto-code"]');
      const functions = document.querySelector('[data-testid="dept-functions"]');
      const functionsBtn = document.querySelector('[data-testid="dept-functions-btn"]');
      const category = document.querySelector('[data-testid="dept-category"]');
      const fields = [...panelNormal.querySelectorAll('.field')].map((el) => el.querySelector('span')?.textContent?.trim() || '');
      const fr = form.getBoundingClientRect();
      const tr = tabs.getBoundingClientRect();
      const sr = save.getBoundingClientRect();
      const cr = cancel.getBoundingClientRect();
      const br = functionsBtn.getBoundingClientRect();
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
        hasAutoCode: !!autoCode,
        autoCodeDisabled: autoCode?.disabled === true,
        autoCodeReadOnly: autoCode?.readOnly === true,
        hasFunctions: !!functions,
        hasFunctionsBtn: !!functionsBtn,
        functionsBtnW: Math.round(br.width),
        functionsBtnH: Math.round(br.height),
        hasCategory: !!category,
        categoryOptions: [...category.options].map((o) => ({ value: o.value, text: o.textContent.trim() })),
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
        dialogOpenedBefore: window.__dialogOpened(),
      };
    });

    // fill extras then cancel discard
    await page.fill('[data-testid="dept-phone"]', '13800000000');
    await page.fill('[data-testid="dept-manager"]', '张三');
    await page.fill('[data-testid="dept-input-code"]', 'KS');
    await page.fill('[data-testid="dept-functions"]', '门诊');
    await page.selectOption('[data-testid="dept-category"]', 'ui-placeholder-clinical');
    await page.evaluate(() => window.__setExtraFromDom());
    const filled = await page.evaluate(() => window.__extraState());
    await page.click('[data-testid="dept-functions-btn"]');
    const dialogAfterClick = await page.evaluate(() => window.__dialogOpened());
    await page.click('[data-testid="dept-cancel"]');
    const afterCancel = await page.evaluate(() => window.__extraState());

    // nursing shell check
    await page.click('[data-testid="dept-tab-nursing"]');
    const nursing = await page.evaluate(() => {
      const list = document.querySelector('[data-testid="dept-nursing-list"]');
      const add = document.querySelector('[data-testid="dept-nursing-add"]');
      const del = document.querySelector('[data-testid="dept-nursing-del"]');
      const note = document.querySelector('[data-testid="dept-nursing-note"]');
      const lr = list.getBoundingClientRect();
      const ar = add.getBoundingClientRect();
      const dr = del.getBoundingClientRect();
      return {
        nursingVisible: !document.querySelector('[data-testid="dept-tabpanel-nursing"]').hidden,
        listH: Math.round(lr.height),
        addW: Math.round(ar.width),
        addH: Math.round(ar.height),
        delW: Math.round(dr.width),
        delH: Math.round(dr.height),
        noteText: note?.textContent?.trim() || '',
        tabNursingSelected: document.querySelector('[data-testid="dept-tab-nursing"]').getAttribute('aria-selected') === 'true',
      };
    });
    await page.click('[data-testid="dept-tab-normal"]');

    const expectedDesktopW = width === 390 ? null : 408;
    const formWPass = expectedDesktopW == null ? base.formW <= width : near(base.formW, expectedDesktopW, 2);
    const btnPass = near(base.saveW, 90, 2) && near(base.saveH, 30, 2) && near(base.cancelW, 90, 2) && near(base.cancelH, 30, 2);
    const shellsPass =
      base.hasAutoCode && base.autoCodeDisabled && base.autoCodeReadOnly &&
      base.hasFunctions && base.hasFunctionsBtn &&
      base.hasCategory &&
      base.hasPhone && base.hasManager && base.hasInputCode &&
      base.fieldCount === 10 &&
      base.hintText.includes('自动编码') &&
      base.hintText.includes('未持久化') &&
      base.hintText.includes('未接主数据');
    const savePass = JSON.stringify(base.savePayloadKeys) === JSON.stringify(['deptCode','deptName','status','sortNo']);
    const discardPass =
      filled.functions === '门诊' && filled.departmentAttr === 'ui-placeholder-clinical' &&
      afterCancel.phone === '' && afterCancel.manager === '' && afterCancel.inputCode === '' &&
      afterCancel.functions === '' && afterCancel.departmentAttr === '';
    const noopPass = dialogAfterClick === false;
    const nursingPass = near(nursing.listH, 257, 2) && near(nursing.addW, 90, 2) && near(nursing.addH, 30, 2);
    const overflowPass = !base.overflow;
    const pass = formWPass && btnPass && shellsPass && savePass && discardPass && noopPass && nursingPass && overflowPass &&
      base.tabNormalText === '常规' && base.tabNursingText === '护理单元包含科室';

    const shot = path.join(outDir, `after-${width}.png`);
    await page.screenshot({ path: shot, fullPage: true });

    results.push({
      width,
      ...base,
      filled,
      afterCancel,
      dialogAfterClick,
      nursing,
      checks: { formWPass, btnPass, shellsPass, savePass, discardPass, noopPass, nursingPass, overflowPass, pass },
      screenshot: shot,
    });
  }

  const allPass = results.every((r) => r.checks.pass) && pageerrors.length === 0;
  const summary = { allPass, pageerrors, results };
  fs.writeFileSync(path.join(outDir, 'after-results.json'), JSON.stringify(summary, null, 2), 'utf8');
  console.log(JSON.stringify({ allPass, pageerrors, table: results.map((r) => ({
    width: r.width,
    formW: r.formW,
    formH: r.formH,
    save: `${r.saveW}x${r.saveH}`,
    fieldCount: r.fieldCount,
    overflow: r.overflow,
    pass: r.checks.pass,
  })) }, null, 2));
  await browser.close();
  process.exit(allPass ? 0 : 1);
})().catch((e) => { console.error(e); process.exit(1); });
