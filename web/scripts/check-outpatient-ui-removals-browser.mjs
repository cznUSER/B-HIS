import assert from 'node:assert/strict'
import fs from 'node:fs/promises'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { chromium } from 'playwright'

const baseUrl = process.env.BHIS_UI_URL || 'http://127.0.0.1:8888'
const password = process.env.BHIS_ACCEPTANCE_PASSWORD
assert.ok(password, 'BHIS_ACCEPTANCE_PASSWORD must contain the authorized demo account password')

const here = path.dirname(fileURLToPath(import.meta.url))
const output = path.resolve(here, '../../deliverables/h01/outpatient-ui-removals-20261019')
await fs.mkdir(output, { recursive: true })

const browserPath = process.env.BHIS_BROWSER_EXECUTABLE || 'C:/Program Files (x86)/Microsoft/Edge/Application/msedge.exe'
const browser = await chromium.launch({ headless: true, executablePath: browserPath })
const page = await browser.newPage({ viewport: { width: 1440, height: 1000 } })
const pageErrors = []
const writeRequests = []
page.on('pageerror', (error) => pageErrors.push(error.message))

await page.route('**/api/v1/**', async (route) => {
  const request = route.request()
  const url = new URL(request.url())
  if (request.method() === 'POST' && url.pathname === '/api/v1/auth/login') {
    await route.continue()
    return
  }
  if (request.method() === 'GET' && url.pathname === '/api/v1/auth/me') {
    await route.continue()
    return
  }
  if (request.method() !== 'GET') {
    writeRequests.push(`${request.method()} ${url.pathname}`)
    await route.abort('blockedbyclient')
    return
  }
  await route.fulfill({
    status: 200,
    contentType: 'application/json',
    body: JSON.stringify({ items: [], total: 0, nextOffset: null }),
  })
})

try {
  await page.goto(`${baseUrl}/login`)
  await page.getByLabel('用户名').fill('system')
  await page.getByLabel('密码').fill(password)
  await page.getByRole('button', { name: '登录' }).click()
  await page.waitForURL('**/app')
  await page.locator('.sidebar').waitFor()
  await page.locator('.dashboard').waitFor()
  await page.waitForTimeout(400)
  await page.screenshot({ path: path.join(output, 'home-1440.png'), fullPage: true })

  const moduleKeys = ['register', 'diagnosis', 'order', 'charge', 'dispense', 'refund']
  for (const width of [1440, 1024, 390]) {
    await page.setViewportSize({ width, height: 1000 })
    for (const key of moduleKeys) {
      await page.goto(`${baseUrl}/app/outpatient/${key}`)
      await page.locator(`.outpatient-frame[data-module="${key}"]`).waitFor()
      await page.waitForTimeout(300)
      assert.equal(await page.locator('[data-testid="outpatient-context"]').count(), 1, `${key} has shared patient context at ${width}px`)
      assert.equal(await page.locator('[data-testid="outpatient-status-chain"]').count(), 1, `${key} has shared status chain at ${width}px`)
      assert.equal(await page.locator('.workflow-steps, .outpatient-step-nav').count(), 1, `${key} has top workflow strip at ${width}px`)
      assert.equal(await page.locator('.outpatient-left-rail').count(), 0, `${key} has no left patient/status rail at ${width}px`)
      assert.equal(await page.locator('.outpatient-frame-head, .outpatient-frame-actions').count(), 0, `${key} has no duplicate pre-workspace heading at ${width}px`)
      assert.equal(await page.locator('.outpatient-frame-link, .outpatient-frame-next').count(), 0, `${key} has no workflow shortcuts at ${width}px`)
      assert.doesNotMatch(await page.locator('body').innerText(), /今日接待/)
      assert.ok(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1), `${key} overflows at ${width}px`)
      if (key === 'register' && width === 1440) {
        await page.screenshot({ path: path.join(output, 'register-1440.png'), fullPage: true })
      }
    }
  }

  await page.goto(`${baseUrl}/app/outpatient/reception`)
  await page.waitForURL('**/app/outpatient')
  assert.equal(await page.locator('.module-grid .module-card').count(), 6, 'legacy reception URL lands on the six-module overview')
  await page.goto(`${baseUrl}/app/settings/master/reception`)
  await page.waitForURL('**/app/settings/master')
  assert.equal(await page.getByRole('link', { name: '门诊运营' }).count(), 0, 'master overview has no reception settings tab')
  assert.deepEqual(writeRequests, [], 'browser acceptance made no API write requests')
  assert.deepEqual(pageErrors, [], 'browser acceptance has no page errors')

  console.log(`OUTPATIENT_UI_REMOVALS_BROWSER_OK modules=${moduleKeys.length} widths=3 screenshots=${output}`)
} finally {
  await browser.close()
}
