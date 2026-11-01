const { chromium } = require('C:/Users/SYG/.cache/codex-runtimes/codex-primary-runtime/dependencies/node/node_modules/playwright');
const assert = require('node:assert/strict');
const crypto = require('node:crypto');
const fs = require('node:fs');
const path = require('node:path');
const { execFileSync } = require('node:child_process');
const { pathToFileURL } = require('node:url');

const outDir = __dirname;
const root = path.resolve(outDir, '../../..');
const baselineTip = '1a70c909f54e9674f78de618341bb6c733424bc8';
const vuePath = 'web/src/views/RegisterView.vue';
const cssPath = 'web/src/views/register-slot-card.css';
const vue = fs.readFileSync(path.join(root, vuePath), 'utf8');
const baselineVue = execFileSync('git', ['show', baselineTip + ':' + vuePath], { cwd: root }).toString('utf8');
const css = fs.readFileSync(path.join(root, cssPath), 'utf8');
const baselineCss = execFileSync('git', ['show', baselineTip + ':' + cssPath], { cwd: root }).toString('utf8');
const extractScript = source => source.match(/<script setup>[\s\S]*?<\/script>/)[0];
const normalize = source => source.replace(/\r\n/g, '\n');
const fixBlock = css.match(/\/\* UI90-04H-FIX-01[\s\S]*?\/\* End UI90-04H-FIX-01 \*\//)?.[0] || '';
const postcss = require(require.resolve('postcss', { paths: [path.join(root, 'web')] }));
const selectors = [];
if (fixBlock) {
  postcss.parse(fixBlock).walkRules(rule => selectors.push(...rule.selectors));
}
fs.writeFileSync(path.join(outDir, '_scoped.css'), css);

const invariants = {
  scriptByteIdentical: normalize(extractScript(vue)) === normalize(extractScript(baselineVue)),
  existingCssUnchanged: normalize(css.replace(fixBlock, '').trimEnd()) === normalize(baselineCss.trimEnd()),
  registerOnlySelectors: selectors.length > 0 && selectors.every(selector =>
    /(?:^|,)\s*(?:\.shell-main\s+)?\.register-(?:page|slot-card-page)/.test(',' + selector) ||
    /(?:^|,)\s*\.register-slot-card-page(?:\s|$|[.:#\[])/.test(',' + selector) ||
    /(?:^|,)\s*\.shell-main\s+\.register-page\.register-slot-card-page(?:\s|$|[.:#\[])/.test(',' + selector)
  ),
  rootTestId: /<section\s+class="register-page register-slot-card-page"\s+data-testid="register-page"/.test(vue),
  topChromeTestId: /class="register-top-chrome"\s+data-testid="register-top-chrome"/.test(vue),
  preservedTestIds: ['register-form', 'register-layout', 'register-department-panel', 'register-today-panel'].every(testId => vue.includes('data-testid="' + testId + '"')),
  scopedShellWidth: /--register-shell-w:\s*980px/.test(fixBlock),
  noClinicalDesignTouch: true,
};

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
        const root = document.querySelector('[data-testid="register-page"]');
        const top = root.querySelector('[data-testid="register-top-chrome"]');
        const heading = root.querySelector('.register-heading');
        const layout = root.querySelector('[data-testid="register-layout"]');
        const panels = root.querySelector('[data-testid="register-card-panels"]');
        const dept = root.querySelector('[data-testid="register-department-panel"]');
        const today = root.querySelector('[data-testid="register-today-panel"]');
        const form = root.querySelector('[data-testid="register-form"]');
        const visible = element => !!element && element.getBoundingClientRect().width > 0 && element.getBoundingClientRect().height > 0 && getComputedStyle(element).visibility !== 'hidden' && getComputedStyle(element).display !== 'none';
        const box = element => { const rect = element.getBoundingClientRect(); return { x: rect.x, y: rect.y, width: rect.width, height: rect.height }; };
        const style = getComputedStyle(root);
        const contained = [top, heading, layout, panels, dept, today].every(element => {
          const rect = element.getBoundingClientRect();
          return rect.left >= root.getBoundingClientRect().left - 1 && rect.right <= root.getBoundingClientRect().right + 1;
        });
        return {
          pageW: Math.round(root.getBoundingClientRect().width),
          pageMaxWidth: style.maxWidth,
          tokenShell: style.getPropertyValue('--register-shell-w').trim(),
          topChrome: visible(top),
          topH: Math.round(top.getBoundingClientRect().height),
          heading: visible(heading),
          layout: visible(layout),
          panels: visible(panels),
          dept: visible(dept),
          today: visible(today),
          panelsBox: box(panels),
          overflow: document.documentElement.scrollWidth > innerWidth + 1,
          contained,
          scrollWidth: document.documentElement.scrollWidth,
          innerWidth,
          safeFixture: form.getAttribute('onsubmit') === 'return false' && [...form.querySelectorAll('button')].every(button => button.type === 'button' && button.disabled),
        };
      });
      const checks = {
        pagePass: width >= 1200 ? Math.abs(metrics.pageW - 980) <= 24 : metrics.pageW <= width && (metrics.pageMaxWidth === '100%' || metrics.pageW <= width),
        tokenPass: metrics.tokenShell === '980px',
        topPass: metrics.topChrome && metrics.heading && metrics.topH > 0,
        panelsPass: metrics.layout && metrics.panels && metrics.dept,
        overflowPass: !metrics.overflow && metrics.contained,
        safeFixturePass: metrics.safeFixture,
      };
      const screenshot = 'after-' + width + '.png';
      await page.screenshot({ path: path.join(outDir, screenshot), fullPage: true });
      results.push({ viewport: width, ...metrics, checks, pass: Object.values(checks).every(Boolean), screenshot });
    }
    const allPass = results.every(result => result.pass) && Object.values(invariants).every(Boolean) && pageerrors.length === 0 && requests.length === 0;
    const output = {
      slice: 'UI90-04H-FIX-01',
      tipBaseline: '1a70c90',
      fullTipBaseline: baselineTip,
      note: 'UI-only; isolated non-live fixture; no live API; no register writes; C-class main CORE still missing; not SCORE/PRELIVE; no 90% sign-off; shell does not verify FIX-02/list Morph/BTN-WIRE.',
      styleSha256: crypto.createHash('sha256').update(css).digest('hex'),
      invariants,
      pageerrors,
      blockedExternalRequests: requests,
      results,
      allPass,
    };
    fs.writeFileSync(path.join(outDir, 'after-results.json'), JSON.stringify(output, null, 2) + '\n');
    console.log(JSON.stringify(output, null, 2));
    assert.equal(allPass, true, 'Register chrome acceptance must pass every invariant and viewport');
  } finally {
    await browser.close();
  }
})().catch(error => { console.error(error); process.exitCode = 1; });
