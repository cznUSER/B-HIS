const { chromium } = require('C:/Users/SYG/.cache/codex-runtimes/codex-primary-runtime/dependencies/node/node_modules/playwright');
const assert = require('node:assert/strict');
const crypto = require('node:crypto');
const fs = require('node:fs');
const path = require('node:path');
const { execFileSync } = require('node:child_process');
const { pathToFileURL } = require('node:url');

const outDir = __dirname;
const root = path.resolve(outDir, '../../..');
const baselineTip = '403e8b202842caedf3abc56a0b6e34ff83c03b07';
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
const fixBlock = css.match(/\/\* UI90-04G-FIX-01[\s\S]*?\/\* End UI90-04G-FIX-01 \*\//)?.[0] || '';
const postcss = require(require.resolve('postcss', { paths: [path.join(root, 'web')] }));
const selectors = [];
if (fixBlock) postcss.parse(fixBlock).walkRules(rule => selectors.push(...rule.selectors));
const invariants = {
  scriptByteIdentical: normalize(extractScript(patient)) === normalize(extractScript(baselinePatient)),
  existingCssUnchanged: normalize(css.replace(fixBlock, '').trimEnd()) === normalize(baselineCss.trimEnd()),
  patientOnlySelectors: selectors.length > 0 && selectors.every(selector => /^(?:\.shell-main\s+)?\.patient-page(?:\s|$|[.:#\[])/.test(selector)),
  rootTestId: /<section\s+class="patient-page"\s+data-testid="patient-page"/.test(patient),
  topChromeTestId: /class="patient-top-chrome"\s+data-testid="patient-top-chrome"/.test(patient),
  preservedTestIds: ['patient-summary', 'patient-form', 'patient-mode-switch'].every(testId => patient.includes('data-testid="' + testId + '"')),
  scopedShellWidth: scopedChrome.includes('max-width: var(--patient-shell-w, 868px)'),
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
        const panel = root.querySelector('[data-testid="patient-form"]');
        const mode = root.querySelector('[data-testid="patient-mode-switch"]');
        const visible = element => !!element && element.getBoundingClientRect().width > 0 && element.getBoundingClientRect().height > 0 && getComputedStyle(element).visibility !== 'hidden' && getComputedStyle(element).display !== 'none';
        const box = element => { const rect = element.getBoundingClientRect(); return { x: rect.x, y: rect.y, width: rect.width, height: rect.height }; };
        const style = getComputedStyle(root);
        const contained = [top, summary, panel, mode, ...panel.querySelectorAll('input, select, button')].every(element => { const rect = element.getBoundingClientRect(); return rect.left >= root.getBoundingClientRect().left - 1 && rect.right <= root.getBoundingClientRect().right + 1; });
        return {
          pageW: Math.round(root.getBoundingClientRect().width), pageMaxWidth: style.maxWidth,
          tokenShell: style.getPropertyValue('--patient-shell-w').trim(),
          topChrome: visible(top), topH: Math.round(top.getBoundingClientRect().height),
          summary: visible(summary), panel: visible(panel), modeSwitch: visible(mode),
          summaryBox: box(summary), panelBox: box(panel), summaryPadding: getComputedStyle(summary).padding,
          panelPadding: getComputedStyle(panel).padding, layoutColumns: getComputedStyle(root.querySelector('.patient-layout')).gridTemplateColumns,
          footerButtonHeight: panel.querySelector('.clinical-form-footer .button').getBoundingClientRect().height,
          overflow: document.documentElement.scrollWidth > innerWidth + 1, contained, scrollWidth: document.documentElement.scrollWidth, innerWidth,
          safeFixture: panel.getAttribute('onsubmit') === 'return false' && [...panel.querySelectorAll('button')].every(button => button.type === 'button' && button.disabled),
        };
      });
      const checks = {
        pagePass: width >= 1200 ? Math.abs(metrics.pageW - 868) <= 24 : metrics.pageW <= width && metrics.pageMaxWidth === '100%',
        tokenPass: metrics.tokenShell === '868px', topPass: metrics.topChrome && metrics.topH > 0,
        summaryPass: metrics.summary, panelPass: metrics.panel && metrics.modeSwitch,
        stackingPass: width > 1040 || metrics.panelBox.y >= metrics.summaryBox.y + metrics.summaryBox.height - 1,
        overflowPass: !metrics.overflow && metrics.contained, footerPass: metrics.footerButtonHeight === 30, safeFixturePass: metrics.safeFixture,
      };
      const screenshot = 'after-' + width + '.png';
      await page.screenshot({ path: path.join(outDir, screenshot), fullPage: true });
      results.push({ viewport: width, ...metrics, checks, pass: Object.values(checks).every(Boolean), screenshot });
    }
    const allPass = results.every(result => result.pass) && Object.values(invariants).every(Boolean) && pageerrors.length === 0 && requests.length === 0;
    const output = {
      slice: 'UI90-04G-FIX-01', tipBaseline: '403e8b2', fullTipBaseline: baselineTip,
      note: 'UI-only; isolated non-live fixture; no live API; no patient writes; not SCORE/PRELIVE; no 90% sign-off; shell does not verify list Morph or FIX-02.',
      styleSha256: crypto.createHash('sha256').update(style).digest('hex'), invariants, pageerrors, blockedExternalRequests: requests, results, allPass,
    };
    fs.writeFileSync(path.join(outDir, 'after-results.json'), JSON.stringify(output, null, 2) + '\n');
    console.log(JSON.stringify(output, null, 2));
    assert.equal(allPass, true, 'Patient chrome acceptance must pass every invariant and viewport');
  } finally { await browser.close(); }
})().catch(error => { console.error(error); process.exitCode = 1; });
