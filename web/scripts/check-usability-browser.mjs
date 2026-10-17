// 2026-10-14 Real browser smoke. Requires Playwright through BHIS_BROWSER_DRIVER.
import assert from 'node:assert/strict'
import { createRequire } from 'node:module'
import { mkdir, writeFile } from 'node:fs/promises'
import { resolve } from 'node:path'

const require = createRequire(import.meta.url)
const { chromium } = require(process.env.BHIS_BROWSER_DRIVER || 'playwright')
const base = process.env.BHIS_UI_URL || 'http://127.0.0.1:15173'
const output = resolve(process.env.BHIS_UI_REPORT || 'browser-report')
const executablePath = process.env.BHIS_BROWSER_EXECUTABLE
await mkdir(output, { recursive: true })
const browser = await chromium.launch({ headless: true, ...(executablePath ? { executablePath } : {}) })
const page = await browser.newPage({ viewport: { width: 1440, height: 1000 } })
const report = { base, startedAt: new Date().toISOString(), checks: [], layouts: [], consoleErrors: [] }
page.on('pageerror', (error) => report.consoleErrors.push(error.message))
page.setDefaultTimeout(15000)
const patientName = '网站验收' + Date.now().toString().slice(-7)
const pass = (name) => { report.checks.push({ name, result: 'PASS' }); console.log(`PASS ${name}`) }

async function signIn() {
  await page.locator('[name=username]').fill('system')
  await page.locator('[name=password]').fill('1')
  await page.getByRole('button', { name: '登录', exact: true }).click()
}
async function open(path, ready) {
  await page.goto(`${base}${path}`)
  if (ready) await page.locator(ready).waitFor()
}
async function layout(name) {
  for (const width of [390, 768, 1440]) {
    await page.setViewportSize({ width, height: 1000 })
    await page.waitForTimeout(350)
    const dimensions = await page.evaluate(() => ({ viewport: innerWidth, document: document.documentElement.scrollWidth }))
    report.layouts.push({ name, width, ...dimensions })
    assert.ok(dimensions.document <= width + 1, `${name} page overflows at ${width}: ${dimensions.document}`)
    await page.screenshot({ path: resolve(output, `${name}-${width}.png`), fullPage: true })
  }
}

try {
  await page.goto(`${base}/app/outpatient/order?registerId=missing-smoke`)
  await page.locator('[name=username]').waitFor()
  assert.ok(new URL(page.url()).searchParams.get('redirect')?.includes('registerId=missing-smoke'))
  await signIn()
  await page.waitForURL('**/app/outpatient/order?registerId=missing-smoke')
  await page.locator('#order-title').waitFor()
  pass('protected deep link returns after login')

  await open('/app/outpatient/patient', '[name=patient-name]')
  await page.locator('[name=patient-name]').fill(patientName)
  await page.locator('[name=patient-sex]').selectOption('M')
  await page.locator('[name=patient-birthday]').fill('1990-01-01')
  await page.locator('[name=patient-fare]').selectOption('SELF')
  await page.getByRole('button', { name: '保存建档', exact: true }).click()
  await page.getByTestId('patient-success').waitFor()
  pass('create patient through form')
  await layout('patient')
  await page.getByRole('link', { name: '去挂号', exact: true }).click()
  await page.getByTestId('dept-select').waitFor()
  await page.getByTestId('dept-select').selectOption({ index: 1 })
  await page.getByTestId('doctor-select').selectOption({ index: 1 })
  await page.getByTestId('save-register').click()
  await page.getByTestId('register-list').waitFor()
  const patientId = new URL(page.url()).searchParams.get('patientId')
  report.context = { patientId }
  assert.ok(patientId)
  pass('create registration with patient context')
  await layout('register')
  await page.locator('.workflow-steps a').filter({ hasText: '开单' }).click()
  await page.getByTestId('order-empty').waitFor()
  assert.equal(new URL(page.url()).searchParams.get('patientId'), patientId)
  const registerOption = await page.getByTestId('order-empty').locator('option').filter({ hasText: patientName }).last().getAttribute('value')
  await page.getByTestId('order-empty').locator('select').selectOption(registerOption)
  await page.getByRole('button', { name: '使用该挂号', exact: true }).click()
  report.context.registerId = registerOption
  await page.getByTestId('order-doctor').waitFor()
  await page.locator('select[aria-label="第 1 行项目"]').selectOption('DRUG-A')
  await page.locator('input[aria-label="第 1 行数量"]').fill('1')
  await page.getByTestId('order-save').click()
  await page.waitForFunction(() => document.querySelector('[data-testid=order-status]')?.value === '暂存')
  await page.getByTestId('order-commit').click()
  await page.waitForFunction(() => document.querySelector('[data-testid=order-status]')?.value === '已提交')
  pass('save and commit prescription')
  await layout('order')

  await page.locator('.workflow-steps a').filter({ hasText: '收费' }).click()
  await page.locator('#charge-title').waitFor()
  await page.locator('[name=charge-keyword]').fill(patientName)
  await page.getByTestId('charge-search').getByRole('button', { name: '检索' }).click()
  await page.locator('.demo-chips button').filter({ hasText: patientName }).click()
  await page.getByTestId('charge-confirm').waitFor()
  await page.getByTestId('charge-confirm').click()
  await page.getByTestId('charge-paid').waitFor()
  pass('cash charge succeeds')
  await layout('charge')
  await page.getByRole('link', { name: '去发药', exact: true }).click()
  await page.getByTestId('dispense-lines').waitFor()
  const chargeId = new URL(page.url()).searchParams.get('charge_id')
  report.context.chargeId = chargeId
  assert.ok(chargeId)
  await page.getByTestId('dispense-all').click()
  await page.getByTestId('dispense-ok').waitFor()
  pass('dispense paid prescription with charge context')
  await page.getByRole('button', { name: /已发查询/ }).click()
  await page.locator('.dispense-issued-panel tbody tr').filter({ hasText: patientName }).getByTestId('issued-return').click()
  await page.getByTestId('return-reason').fill('页面验收退药')
  await page.getByTestId('return-confirm').click()
  await page.getByTestId('return-ok').waitFor()
  await layout('return')
  await page.getByTestId('go-refund').click()
  await page.getByTestId('charge-refund-dialog').waitFor()
  assert.equal(new URL(page.url()).searchParams.get('charge_id'), chargeId)
  await page.getByTestId('charge-refund-dialog').locator('input[type=checkbox]').first().check()
  await page.locator('[name=refund-reason]').fill('页面验收退费')
  await page.getByTestId('charge-refund-confirm').click()
  await page.getByTestId('charge-refund-done').waitFor()
  pass('return then refund preserves original charge')

  // Same component query switches and history must update the active workspace.
  await page.evaluate(async () => document.querySelector('#app').__vue_app__.config.globalProperties.$router.push('/app/outpatient/charge?view=pending'))
  await page.waitForFunction(() => document.querySelector('.charge-view-switch button:first-child')?.getAttribute('aria-pressed') === 'true')
  await page.evaluate(async () => document.querySelector('#app').__vue_app__.config.globalProperties.$router.push('/app/outpatient/charge?view=records'))
  await page.waitForFunction(() => document.querySelector('.charge-view-switch button:nth-child(2)')?.getAttribute('aria-pressed') === 'true')
  await page.goBack()
  await page.waitForFunction(() => document.querySelector('.charge-view-switch button:first-child')?.getAttribute('aria-pressed') === 'true')
  pass('charge query reuse and browser back')
  await open('/app/settings/master/catalog', '.master-page')
  await layout('master')
  await open('/app/settings/master/inventory', '[data-testid=stock-list]')
  await page.getByTestId('stock-adjust-DRUG-A').click()
  await page.getByTestId('adjust-dialog').waitFor()
  await page.waitForFunction(() => document.activeElement?.getAttribute('data-testid') === 'adjust-delta')
  await layout('inventory-dialog')
  await page.keyboard.press('Escape')
  await page.getByTestId('adjust-dialog').waitFor({ state: 'hidden' })
  pass('inventory dialog focus and Escape')
  assert.deepEqual(report.consoleErrors, [], 'browser has no uncaught application errors')
  pass('three viewport widths and no uncaught page errors')
} catch (error) {
  report.checks.push({ name: 'browser smoke', result: 'FAIL', error: error.message })
  await page.screenshot({ path: resolve(output, 'failure.png'), fullPage: true })
  await writeFile(resolve(output, 'failure-text.txt'), await page.locator('body').innerText())
  await writeFile(resolve(output, 'overflow.json'), JSON.stringify(await page.evaluate(() => [...document.querySelectorAll('body *')].map((el) => {
    const rect = el.getBoundingClientRect(); const css = getComputedStyle(el)
    return { tag: el.tagName, class: el.className?.toString(), left: rect.left, right: rect.right, width: rect.width, scroll: el.scrollWidth, overflow: css.overflowX, minWidth: css.minWidth, position: css.position, transform: css.transform }
  }).filter((item) => item.right > innerWidth + 1 || item.left < -1)), null, 2))
  process.exitCode = 1
  console.error(error.message)
} finally {
  report.finishedAt = new Date().toISOString()
  await writeFile(resolve(output, 'browser-evidence.json'), JSON.stringify(report, null, 2))
  await browser.close()
}
