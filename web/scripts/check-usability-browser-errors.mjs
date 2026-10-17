// 2026-10-14 Browser regression against a real isolated API, replacing only failed HTTP responses.
import assert from 'node:assert/strict'
import { createRequire } from 'node:module'
import { readFile, writeFile, mkdir } from 'node:fs/promises'
import { resolve } from 'node:path'
const require = createRequire(import.meta.url)
const { chromium } = require(process.env.BHIS_BROWSER_DRIVER || 'playwright')
const base = process.env.BHIS_UI_URL || 'http://127.0.0.1:15174'
const output = resolve(process.env.BHIS_UI_REPORT || 'browser-errors-report')
const previous = JSON.parse(await readFile(process.env.BHIS_UI_CONTEXT, 'utf8'))
const { patientId, registerId } = previous.context
await mkdir(output, { recursive: true })
const browser = await chromium.launch({ headless: true, executablePath: process.env.BHIS_BROWSER_EXECUTABLE })
const page = await browser.newPage({ viewport: { width: 390, height: 1000 } })
page.setDefaultTimeout(15000)
const checks = []
const pass = (name) => { checks.push({ name, result: 'PASS' }); console.log(`PASS ${name}`) }
const failure = (route, status = 503, code = 'SERVICE_UNAVAILABLE') => route.fulfill({ status, contentType: 'application/json', body: JSON.stringify({ success: false, data: null, error: { code, message: '测试读取失败' }, requestId: 'ui-smoke' }) })
async function login() {
  await page.locator('[name=username]').fill('system')
  await page.locator('[name=password]').fill('1')
  await page.getByRole('button', { name: '登录', exact: true }).click()
}
try {
  await page.goto(`${base}/login`)
  await login()
  await page.waitForURL('**/app')
  await page.route('**/api/v1/departments*', (route) => failure(route))
  await page.goto(`${base}/app/settings/master/departments`)
  await page.getByTestId('dept-load-error').waitFor()
  assert.equal(await page.getByTestId('dept-empty').count(), 0)
  await page.screenshot({ path: resolve(output, 'departments-error-390.png'), fullPage: true })
  pass('master read failure is an error, not empty')
  await page.unroute('**/api/v1/departments*')
  await page.getByTestId('dept-retry').click()
  await page.getByTestId('dept-list').waitFor()
  pass('master retry loads rows')

  await page.route('**/api/v1/departments*', (route) => failure(route))
  await page.route('**/api/v1/registers?*', (route) => failure(route))
  await page.goto(`${base}/app/outpatient/register?patientId=${patientId}`)
  await page.getByTestId('register-master-error').waitFor()
  assert.equal(await page.getByTestId('save-register').isDisabled(), true)
  assert.equal(await page.locator('body').getByText('当日暂无挂号。', { exact: true }).count(), 0)
  assert.equal(await page.getByTestId('dept-select').locator('option').count(), 1)
  await page.screenshot({ path: resolve(output, 'register-error-390.png'), fullPage: true })
  pass('register API failure blocks save with no mock options or false empty')
  await page.unroute('**/api/v1/departments*')
  await page.unroute('**/api/v1/registers?*')

  await page.route('**/api/v1/registers?*', (route) => failure(route))
  await page.goto(`${base}/app/outpatient/order`)
  await page.getByRole('button', { name: '重试当日挂号', exact: true }).waitFor()
  assert.equal(await page.getByTestId('order-empty').count(), 0)
  pass('order register error does not prompt duplicate registration')
  await page.unroute('**/api/v1/registers?*')

  await page.route('**/api/v1/catalog-items*', (route) => failure(route))
  await page.goto(`${base}/app/outpatient/order?registerId=${registerId}`)
  await page.getByTestId('order-master-error').waitFor()
  assert.equal(await page.getByTestId('order-save').isDisabled(), true)
  assert.equal(await page.getByTestId('order-commit').isDisabled(), true)
  await page.screenshot({ path: resolve(output, 'order-error-390.png'), fullPage: true })
  pass('real catalog error blocks unsafe prescription writes')
  await page.unroute('**/api/v1/catalog-items*')

  await page.route('**/api/v1/departments*', (route) => failure(route, 403, 'FORBIDDEN'))
  await page.goto(`${base}/app/settings/master/departments`)
  await page.getByTestId('dept-load-error').waitFor()
  assert.ok(page.url().includes('/app/settings/master/departments'))
  pass('403 permission failure keeps authenticated context')
  await page.unroute('**/api/v1/departments*')
  await page.route('**/api/v1/departments*', (route) => failure(route, 401, 'TOKEN_EXPIRED'))
  await page.getByTestId('dept-retry').click()
  await page.waitForURL('**/login?**')
  assert.equal(new URL(page.url()).searchParams.get('redirect'), '/app/settings/master/departments')
  assert.equal(new URL(page.url()).searchParams.get('expired'), '1')
  pass('authenticated 401 returns to login with original page')
  await page.unroute('**/api/v1/departments*')
  await login()
  await page.waitForURL('**/app/settings/master/departments')
  await page.getByTestId('dept-list').waitFor()
  pass('reauthentication returns to original page')
} catch (error) {
  checks.push({ name: 'browser errors', result: 'FAIL', error: error.message })
  await page.screenshot({ path: resolve(output, 'failure.png'), fullPage: true })
  await writeFile(resolve(output, 'failure-text.txt'), await page.locator('body').innerText())
  console.error(error.message)
  process.exitCode = 1
} finally {
  await writeFile(resolve(output, 'browser-evidence.json'), JSON.stringify({ base, finishedAt: new Date().toISOString(), checks }, null, 2))
  await browser.close()
}
