const { chromium } = require('C:/Users/SYG/.cache/codex-runtimes/codex-primary-runtime/dependencies/node/node_modules/playwright');
const fs = require('node:fs');
const path = require('node:path');
const outDir = 'E:/B-HIS/deliverables/h05/ui90-04a-fix-01';
const htmlPath = path.join(outDir, 'token-probe.html');

function styleOf(el) {
  const cs = getComputedStyle(el);
  return {
    borderTopWidth: cs.borderTopWidth,
    borderRightWidth: cs.borderRightWidth,
    borderBottomWidth: cs.borderBottomWidth,
    borderLeftWidth: cs.borderLeftWidth,
    borderWidth: cs.borderWidth,
    transform: cs.transform,
    transition: cs.transition,
    transitionProperty: cs.transitionProperty,
  };
}

(async () => {
  const browser = await chromium.launch({
    headless: true,
    executablePath: 'C:/Program Files (x86)/Microsoft/Edge/Application/msedge.exe',
  });
  const page = await browser.newPage();
  const errors = [];
  page.on('pageerror', (e) => errors.push(String(e.message || e)));
  const viewports = [1440, 1024, 390];
  const results = [];

  for (const width of viewports) {
    await page.setViewportSize({ width, height: 900 });
    await page.goto('file:///' + htmlPath.replace(/\\/g, '/'));
    await page.waitForSelector('[data-testid="password-change-form"]');

    const base = await page.evaluate(() => {
      const dlg = document.querySelector('.ui2-password-dialog');
      const form = document.querySelector('.password-change-form');
      const btn = document.querySelector('#ok');
      const pick = (el) => {
        const cs = getComputedStyle(el);
        return {
          borderTopWidth: cs.borderTopWidth,
          borderRightWidth: cs.borderRightWidth,
          borderBottomWidth: cs.borderBottomWidth,
          borderLeftWidth: cs.borderLeftWidth,
          transform: cs.transform,
          transitionProperty: cs.transitionProperty,
        };
      };
      return {
        dialog: pick(dlg),
        form: pick(form),
        button: pick(btn),
        overflow: document.documentElement.scrollWidth > innerWidth,
        scrollWidth: document.documentElement.scrollWidth,
        innerWidth,
      };
    });

    // hover
    await page.hover('#ok');
    await page.waitForTimeout(50);
    const hover = await page.evaluate(() => {
      const cs = getComputedStyle(document.querySelector('#ok'));
      return { transform: cs.transform, borderTopWidth: cs.borderTopWidth };
    });

    // active
    await page.dispatchEvent('#ok', 'mousedown', { button: 0 });
    await page.waitForTimeout(50);
    const active = await page.evaluate(() => {
      const cs = getComputedStyle(document.querySelector('#ok'));
      return { transform: cs.transform, borderTopWidth: cs.borderTopWidth };
    });
    await page.dispatchEvent('#ok', 'mouseup', { button: 0 });

    const shot = path.join(outDir, `token-probe-${width}.png`);
    await page.screenshot({ path: shot, fullPage: true });

    results.push({ viewport: width, base, hover, active, screenshot: shot });
  }

  const settled = {
    states: {
      normal: { borderWidth: results[0].base.button.borderTopWidth, transform: results[0].base.button.transform },
      hover: { borderWidth: results[0].hover.borderTopWidth, transform: results[0].hover.transform },
      active: { borderWidth: results[0].active.borderTopWidth, transform: results[0].active.transform },
      dialogBorder: results[0].base.dialog.borderTopWidth,
      formBorderTop: results[0].base.form.borderTopWidth, formBorderLeft: results[0].base.form.borderLeftWidth,
    },
    viewports: results.map((r) => ({
      viewport: r.viewport,
      overflow: r.base.overflow,
      dialogBorder: r.base.dialog.borderTopWidth,
      formBorder: r.base.form.borderTopWidth,
      buttonBorder: r.base.button.borderTopWidth,
      hoverTransform: r.hover.transform,
      activeTransform: r.active.transform,
    })),
    errors,
    note: 'Isolated HTML fixture with design-system .button motion leak + PasswordChangeView scoped CSS copy; not full Vue mount.',
  };

  fs.writeFileSync(path.join(outDir, 'settled-states.json'), JSON.stringify(settled, null, 2));
  fs.writeFileSync(path.join(outDir, 'token-probe-results.json'), JSON.stringify(results, null, 2));

  const fail = [];
  for (const r of results) {
    if (r.base.dialog.borderTopWidth !== '2px' || r.base.dialog.borderLeftWidth !== '2px') fail.push(`dialog border ${r.viewport}=${JSON.stringify(r.base.dialog)}`);
    // form keeps border-top:0 under header; sides/bottom must be 2px (DFM FrameWidth=2)
    if (r.base.form.borderLeftWidth !== '2px' || r.base.form.borderRightWidth !== '2px' || r.base.form.borderBottomWidth !== '2px') fail.push(`form side/bottom border ${r.viewport}=${JSON.stringify(r.base.form)}`);
    if (r.base.form.borderTopWidth !== '0px') fail.push(`form top unexpectedly ${r.base.form.borderTopWidth}`);
    if (r.base.button.borderTopWidth !== '2px' || r.base.button.borderLeftWidth !== '2px') fail.push(`button border ${r.viewport}=${JSON.stringify(r.base.button)}`);
    if (r.hover.transform !== 'none') fail.push(`hover transform ${r.viewport}=${r.hover.transform}`);
    if (r.active.transform !== 'none') fail.push(`active transform ${r.viewport}=${r.active.transform}`);
    if (r.base.overflow) fail.push(`overflow ${r.viewport}`);
  }
  if (errors.length) fail.push(...errors.map((e) => 'pageerror:' + e));

  console.log(JSON.stringify({ ok: fail.length === 0, fail, settled: settled.states, overflows: results.map((r) => [r.viewport, r.base.overflow]) }, null, 2));
  await browser.close();
  if (fail.length) process.exitCode = 1;
})().catch((e) => {
  console.error(e);
  process.exitCode = 1;
});