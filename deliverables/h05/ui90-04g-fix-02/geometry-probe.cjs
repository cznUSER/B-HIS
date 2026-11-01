const { chromium } = require('C:/Users/SYG/.cache/codex-runtimes/codex-primary-runtime/dependencies/node/node_modules/playwright');
const assert = require('node:assert/strict');
const crypto = require('node:crypto');
const fs = require('node:fs');
const path = require('node:path');
const { execFileSync } = require('node:child_process');
const { pathToFileURL } = require('node:url');

const outDir = __dirname;
const root = path.resolve(outDir, '../../..');
const baselineTip = '9d23097fc2419add4a8054c1f4da0d0c69051d8b';
const patientPath = 'web/src/views/PatientView.vue';
const cssPath = 'web/src/views/clinical-design.css';
const patient = fs.readFileSync(path.join(root, patientPath), 'utf8');
const baselinePatient = execFileSync('git', ['show', baselineTip + ':' + patientPath], { cwd: root }).toString('utf8');
const css = fs.readFileSync(path.join(root, cssPath), 'utf8');
const baselineCss = execFileSync('git', ['show', baselineTip + ':' + cssPath], { cwd: root }).toString('utf8');
const extractScript = source => source.match(/<script setup>[\s\S]*?<\/script>/)[0];
const scopedChrome = patient.match(/<style scoped>([\s\S]*?)<\/style>/)[1];
const style = css + '\n/* PatientView.vue scoped summary/mode-switch/create-layout chrome; exact extraction. */\n' + scopedChrome;
const normalize = source => source.replace(/\r\n/g, '\n');
const fix01Block = css.match(/\/\* UI90-04G-FIX-01[\s\S]*?\/\* End UI90-04G-FIX-01 \*\//)?.[0] || '';
const fix02Block = css.match(/\/\* UI90-04G-FIX-02[\s\S]*?\/\* End UI90-04G-FIX-02 \*\//)?.[0] || '';
const postcss = require(require.resolve('postcss', { paths: [path.join(root, 'web')] }));
const selectors = [];
if (fix02Block) postcss.parse(fix02Block).walkRules(rule => selectors.push(...rule.selectors));
const stripFix02 = source => normalize(source).replace(/\/\* UI90-04G-FIX-02[\s\S]*?\/\* End UI90-04G-FIX-02 \*\//, '').trimEnd();
const invariants = {
  scriptByteIdentical: normalize(extractScript(patient)) === normalize(extractScript(baselinePatient)),
  fix01Kept: Boolean(fix01Block) && /--patient-shell-w:\s*868px/.test(fix01Block),
  cssWithoutFix02MatchesTip: stripFix02(css) === normalize(baselineCss).trimEnd(),
  patientOnlySelectors: selectors.length > 0 && selectors.every(selector => /^(?:\.shell-main\s+)?\.patient-page(?:\s|$|[.:#\[])/.test(selector)),
  rootTestId: /<section\s+class="patient-page"\s+data-testid="patient-page"/.test(patient),
  topChromeTestId: /class="patient-top-chrome"\s+data-testid="patient-top-chrome"/.test(patient),
  preservedTestIds: ['patient-summary', 'patient-form', 'patient-mode-switch', 'patient-simple-fields'].every(testId => patient.includes('data-testid="' + testId + '"')),
  newTestIds: ['patient-summary-grid', 'patient-form-footer', 'patient-panel-actions'].every(testId => patient.includes('data-testid="' + testId + '"')),
  scopedShellWidth: scopedChrome.includes('max-width: var(--patient-shell-w, 868px)'),
  noSharedFooterRewrite: !/^\s*:is\(\.patient-page,\s*\.register-page\)\s*\.clinical-form-footer/m.test(fix02Block) && !fix02Block.includes(':is(.patient-page, .register-page)'),
};
fs.writeFileSync(path.join(outDir, '_scoped.css'), style);

(async () => {
  const browser = await chromium.launch({ headless: true, executablePath: 'C:/Program Files (x86)/Microsoft/Edge/Application/msedge.exe' });
  const context = await browser.newContext({ serviceWorkers: 'block' });
  const requests = [];
  const pageerrors = [];
  await context.route('**/*', route => {
    const url = route.request().url();
    if (url.startsWith('file:')) return route.continue();
    requests.push({ url, method: route.request().method() });
    return route.abort();
  });
  const page = await context.newPage();
  page.on('pageerror', error => pageerrors.push(error.message));
  const results = [];
  try {
    for (const width of [1440, 1024, 390]) {
      await page.setViewportSize({ width, height: 980 });
      await page.goto(pathToFileURL(path.join(outDir, 'geometry-fixture.html')).href);
      await page.evaluate(() => document.fonts.ready);
      const metrics = await page.evaluate(() => {
        const root = document.querySelector('[data-testid="patient-page"]');
        const top = root.querySelector('[data-testid="patient-top-chrome"]');
        const summary = root.querySelector('[data-testid="patient-summary"]');
        const summaryGrid = root.querySelector('[data-testid="patient-summary-grid"]');
        const panel = root.querySelector('[data-testid="patient-form"]');
        const mode = root.querySelector('[data-testid="patient-mode-switch"]');
        const section = root.querySelector('[data-testid="patient-simple-fields"]');
        const fieldGrid = section && section.querySelector('.field-grid');
        const footer = root.querySelector('[data-testid="patient-form-footer"]');
        const actions = root.querySelector('[data-testid="patient-panel-actions"]');
        const primary = actions && actions.querySelector('.button');
        const visible = element => !!element && element.getBoundingClientRect().width > 0 && element.getBoundingClientRect().height > 0 && getComputedStyle(element).visibility !== 'hidden' && getComputedStyle(element).display !== 'none';
        const box = element => { const rect = element.getBoundingClientRect(); return { x: rect.x, y: rect.y, width: rect.width, height: rect.height }; };
        const style = getComputedStyle(root);
        const contained = [top, summary, summaryGrid, panel, mode, section, footer, actions, ...panel.querySelectorAll('input, select, button')].every(element => { const rect = element.getBoundingClientRect(); return rect.left >= root.getBoundingClientRect().left - 1 && rect.right <= root.getBoundingClientRect().right + 1; });
        return {
          pageW: Math.round(root.getBoundingClientRect().width), pageMaxWidth: style.maxWidth,
          tokenShell: style.getPropertyValue('--patient-shell-w').trim(),
          topChrome: visible(top), topH: Math.round(top.getBoundingClientRect().height),
          summary: visible(summary), summaryGrid: visible(summaryGrid), summaryGridBox: box(summaryGrid),
          summaryGridGap: getComputedStyle(summaryGrid).gap,
          panel: visible(panel), modeSwitch: visible(mode), section: visible(section),
          fieldGridGap: fieldGrid ? getComputedStyle(fieldGrid).gap : '',
          fieldWideColumn: fieldGrid && fieldGrid.querySelector('.field-wide') ? getComputedStyle(fieldGrid.querySelector('.field-wide')).gridColumn : '',
          footer: visible(footer), actions: visible(actions), footerBox: box(footer),
          footerButtonHeight: primary ? primary.getBoundingClientRect().height : 0,
          footerButtonMinWidth: primary ? getComputedStyle(primary).minWidth : '',
          footerButtonWidth: primary ? Math.round(primary.getBoundingClientRect().width) : 0,
          summaryBox: box(summary), panelBox: box(panel),
          overflow: document.documentElement.scrollWidth > innerWidth + 1, contained,
          scrollWidth: document.documentElement.scrollWidth, innerWidth,
          safeFixture: panel.getAttribute('onsubmit') === 'return false' && [...panel.querySelectorAll('button')].every(button => button.type === 'button' && button.disabled),
        };
      });
      const checks = {
        pagePass: width >= 1200 ? Math.abs(metrics.pageW - 868) <= 24 : metrics.pageW <= width && metrics.pageMaxWidth === '100%',
        tokenPass: metrics.tokenShell === '868px',
        topPass: metrics.topChrome && Math.abs(metrics.topH - 87) <= 8,
        summaryPass: metrics.summary && metrics.summaryGrid,
        fieldPass: metrics.panel && metrics.modeSwitch && metrics.section && !!metrics.fieldGridGap,
        footerPass: metrics.footer && metrics.actions && metrics.footerButtonHeight === 30 && metrics.footerButtonMinWidth === '90px',
        stackingPass: width > 1040 || metrics.panelBox.y >= metrics.summaryBox.y + metrics.summaryBox.height - 1,
        overflowPass: !metrics.overflow && metrics.contained,
        safeFixturePass: metrics.safeFixture,
      };
      const screenshot = 'after-' + width + '.png';
      await page.screenshot({ path: path.join(outDir, screenshot), fullPage: true });
      results.push({ viewport: width, ...metrics, checks, pass: Object.values(checks).every(Boolean), screenshot });
    }
    const allPass = results.every(result => result.pass) && Object.values(invariants).every(Boolean) && pageerrors.length === 0 && requests.length === 0;
    const output = {
      slice: 'UI90-04G-FIX-02',
      tipBaseline: '9d23097',
      fullTipBaseline: baselineTip,
      implementAnchor: 'f282e09',
      note: 'UI-only; isolated non-live fixture; no live API; no patient writes; not SCORE/PRELIVE; no 90% sign-off; conserve FIX-01 shell 868/topH~87; geometry does not verify list Morph / BTN-WIRE / live createPatient.',
      styleSha256: crypto.createHash('sha256').update(style).digest('hex'),
      invariants, pageerrors, blockedExternalRequests: requests, results, allPass,
    };
    fs.writeFileSync(path.join(outDir, 'after-results.json'), JSON.stringify(output, null, 2) + '\n');
    console.log(JSON.stringify({ allPass, invariants, viewports: results.map(r => ({ viewport: r.viewport, pageW: r.pageW, topH: r.topH, footerButtonHeight: r.footerButtonHeight, footerButtonMinWidth: r.footerButtonMinWidth, overflow: r.overflow, pass: r.pass, checks: r.checks })) }, null, 2));
    assert.equal(allPass, true, 'Patient FIX-02 acceptance must pass every invariant and viewport');
  } finally { await browser.close(); }
})().catch(error => { console.error(error); process.exitCode = 1; });
