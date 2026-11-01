const { chromium } = require('C:/Users/SYG/.cache/codex-runtimes/codex-primary-runtime/dependencies/node/node_modules/playwright');
const fs = require('node:fs');
const path = require('node:path');

const outDir = 'E:/B-HIS/deliverables/h05/ui90-04d-fix-03c';
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
    await page.setViewportSize({ width, height: 1400 });
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
      const kindGroup = document.querySelector('[data-testid="dept-kind-group"]');
      const kindNormal = document.querySelector('[data-testid="dept-kind-normal"]');
      const kindEk = document.querySelector('[data-testid="dept-kind-ek"]');
      const kindJz = document.querySelector('[data-testid="dept-kind-jz"]');
      const kindZy = document.querySelector('[data-testid="dept-kind-zy"]');
      const iface1 = document.querySelector('[data-testid="dept-iface-1"]');
      const iface2 = document.querySelector('[data-testid="dept-iface-2"]');
      const guide = document.querySelector('[data-testid="dept-guide-info"]');
      const activeMark = document.querySelector('[data-testid="dept-active-mark"]');
      const status = document.querySelector('[data-testid="dept-status"]');
      const fields = [...panelNormal.querySelectorAll('.field')].map((el) => {
        const span = el.querySelector(':scope > span, .dept-kind-label');
        return span?.textContent?.trim() || '';
      });
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
        hasAutoCode: !!autoCode,
        autoCodeDisabled: autoCode?.disabled === true,
        hasFunctions: !!functions,
        hasFunctionsBtn: !!functionsBtn,
        hasCategory: !!category,
        hasKindGroup: !!kindGroup,
        hasKindNormal: !!kindNormal,
        hasKindEk: !!kindEk,
        hasKindJz: !!kindJz,
        hasKindZy: !!kindZy,
        kindDefaultNormal: kindNormal?.checked === true,
        hasIface1: !!iface1,
        hasIface2: !!iface2,
        hasGuide: !!guide,
        hasActiveMark: !!activeMark,
        activeMarkDefaultOff: activeMark?.checked === false,
        statusValue: status?.value || '',
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

    // mutual exclusion: select EK then check only EK
    await page.click('[data-testid="dept-kind-ek"]');
    const afterEk = await page.evaluate(() => ({
      normal: document.querySelector('[data-testid="dept-kind-normal"]').checked,
      ek: document.querySelector('[data-testid="dept-kind-ek"]').checked,
      jz: document.querySelector('[data-testid="dept-kind-jz"]').checked,
      zy: document.querySelector('[data-testid="dept-kind-zy"]').checked,
      deptCode: document.querySelector('[data-testid="dept-code"]').value,
    }));

    // fill 03c + extras; toggle active; ensure status unchanged
    const statusBefore = await page.inputValue('[data-testid="dept-status"]');
    await page.fill('[data-testid="dept-phone"]', '13800000000');
    await page.fill('[data-testid="dept-manager"]', '张三');
    await page.fill('[data-testid="dept-input-code"]', 'KS');
    await page.fill('[data-testid="dept-functions"]', '门诊');
    await page.selectOption('[data-testid="dept-category"]', 'ui-placeholder-clinical');
    await page.fill('[data-testid="dept-iface-1"]', 'IF1');
    await page.fill('[data-testid="dept-iface-2"]', 'IF2');
    await page.fill('[data-testid="dept-guide-info"]', '请到一楼');
    await page.check('[data-testid="dept-active-mark"]');
    const statusAfterActive = await page.inputValue('[data-testid="dept-status"]');
    await page.evaluate(() => window.__setExtraFromDom());
    const filled = await page.evaluate(() => window.__extraState());
    await page.click('[data-testid="dept-functions-btn"]');
    const dialogAfterClick = await page.evaluate(() => window.__dialogOpened());
    await page.click('[data-testid="dept-save"]');
    const saveKeys = await page.evaluate(() => window.__savePayloadKeys());
    await page.click('[data-testid="dept-cancel"]');
    const afterCancel = await page.evaluate(() => window.__extraState());
    const radiosAfterCancel = await page.evaluate(() => ({
      normal: document.querySelector('[data-testid="dept-kind-normal"]').checked,
      ek: document.querySelector('[data-testid="dept-kind-ek"]').checked,
      active: document.querySelector('[data-testid="dept-active-mark"]').checked,
    }));

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
      };
    });

    await page.screenshot({ path: path.join(outDir, `after-${width}.png`), fullPage: true });

    const expectedFormW = width === 390 ? null : 408;
    const formWPass = expectedFormW == null ? true : near(base.formW, expectedFormW, 2);
    const btnPass = near(base.saveW, 90, 2) && near(base.saveH, 30, 2) && near(base.cancelW, 90, 2) && near(base.cancelH, 30, 2);
    const controlsPass =
      base.hasKindGroup && base.hasKindNormal && base.hasKindEk && base.hasKindJz && base.hasKindZy &&
      base.kindDefaultNormal && base.hasIface1 && base.hasIface2 && base.hasGuide && base.hasActiveMark && base.activeMarkDefaultOff;
    const conservePass =
      base.hasPhone && base.hasManager && base.hasInputCode && base.hasAutoCode && base.autoCodeDisabled &&
      base.hasFunctions && base.hasFunctionsBtn && base.hasCategory &&
      base.tabNormalText === '常规' && base.tabNursingText === '护理单元包含科室';
    const mutexPass = afterEk.ek === true && afterEk.normal === false && afterEk.jz === false && afterEk.zy === false && afterEk.deptCode === 'D001';
    const activeNeStatusPass = statusBefore === statusAfterActive && statusBefore === 'ACTIVE' && filled.isActiveMark === true;
    const discardPass =
      afterCancel.deptKind === 'normal' && afterCancel.expandInterface1 === '' && afterCancel.expandInterface2 === '' &&
      afterCancel.guidInfo === '' && afterCancel.isActiveMark === false && afterCancel.phone === '' &&
      radiosAfterCancel.normal === true && radiosAfterCancel.active === false;
    const savePass = JSON.stringify(saveKeys) === JSON.stringify(['deptCode', 'deptName', 'status', 'sortNo']);
    const nursingPass = nursing.nursingVisible && near(nursing.listH, 257, 2) && near(nursing.addW, 90, 2) && near(nursing.addH, 30, 2);
    const overflowPass = base.overflow === false;
    const hintPass = /激活标记≠启停 status/.test(base.hintText) && /未接主数据/.test(base.hintText) && /未持久化/.test(base.hintText);
    const noopPass = dialogAfterClick === false;

    const pass = formWPass && btnPass && controlsPass && conservePass && mutexPass && activeNeStatusPass && discardPass && savePass && nursingPass && overflowPass && hintPass && noopPass && pageerrors.length === 0;

    results.push({
      width,
      ...base,
      afterEk,
      statusBefore,
      statusAfterActive,
      filled,
      dialogAfterClick,
      saveKeys,
      afterCancel,
      radiosAfterCancel,
      nursing,
      checks: { formWPass, btnPass, controlsPass, conservePass, mutexPass, activeNeStatusPass, discardPass, savePass, nursingPass, overflowPass, hintPass, noopPass },
      pass,
    });
  }

  const allPass = results.every((r) => r.pass);
  const summary = { allPass, pageerrors, results };
  fs.writeFileSync(path.join(outDir, 'after-results.json'), JSON.stringify(summary, null, 2), 'utf8');
  console.log(JSON.stringify({ allPass, pageerrors, passes: results.map((r) => ({ width: r.width, formW: r.formW, formH: r.formH, fieldCount: r.fieldCount, overflow: r.overflow, pass: r.pass, checks: r.checks })) }, null, 2));
  await browser.close();
  process.exit(allPass ? 0 : 1);
})().catch((err) => {
  console.error(err);
  process.exit(2);
});
