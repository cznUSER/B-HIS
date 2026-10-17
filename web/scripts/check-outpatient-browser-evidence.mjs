import assert from 'node:assert/strict'
import { createRequire } from 'node:module'
import { mkdir, writeFile } from 'node:fs/promises'
import { resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import net from 'node:net'
import vue from '@vitejs/plugin-vue'
import { createServer } from 'vite'

const require = createRequire(import.meta.url)
const { chromium } = require(process.env.BHIS_BROWSER_DRIVER || 'playwright')
const webRoot = fileURLToPath(new URL('../', import.meta.url))
const output = resolve(process.env.BHIS_UI_REPORT || 'deliverables/h01/outpatient-closure-browser-20261018')
await mkdir(output, { recursive: true })

const probe = net.createServer()
await new Promise((resolveListen, reject) => {
  probe.once('error', reject)
  probe.listen(0, '127.0.0.1', resolveListen)
})
const port = probe.address().port
await new Promise((resolveClose, reject) => probe.close((error) => error ? reject(error) : resolveClose()))

const system = {
  id: 'system-1', username: 'system', displayName: '系统管理员', roles: ['SYSTEM'],
  permissions: ['admin:*'], scopes: ['outpatient:'], deniedPermissions: [], deniedScopes: [],
  accessControlLoaded: true, super: true, developer: false, mustChangePassword: false, cannotModifyPassword: false,
}
const routes = [
  ['overview', '/app'], ['patient', '/app/outpatient/patient'], ['register', '/app/outpatient/register'],
  ['diagnosis', '/app/outpatient/diagnosis'], ['order', '/app/outpatient/order'],
  ['charge', '/app/outpatient/charge'], ['dispense', '/app/outpatient/dispense'], ['refund', '/app/outpatient/refund'],
]
const server = await createServer({ configFile: false, root: webRoot, plugins: [vue()], logLevel: 'error', server: { host: '127.0.0.1', port, strictPort: true } })
const browser = await chromium.launch({ headless: true, ...(process.env.BHIS_BROWSER_EXECUTABLE ? { executablePath: process.env.BHIS_BROWSER_EXECUTABLE } : {}) })
const report = { base: `http://127.0.0.1:${port}`, startedAt: new Date().toISOString(), routes: [], layouts: [], consoleErrors: [], requestFailures: [] }

try {
  await server.listen()
  const context = await browser.newContext({ viewport: { width: 1440, height: 1000 } })
  await context.addInitScript((user) => {
    sessionStorage.setItem('bhis.session.v1', JSON.stringify({ accessToken: 'browser-evidence', tokenType: 'Bearer', expiresIn: 3600, expiresAt: Date.now() + 3600000, user }))
  }, system)
  const page = await context.newPage()
  page.on('pageerror', (error) => report.consoleErrors.push(error.message))
  page.on('requestfailed', (request) => report.requestFailures.push(`${request.method()} ${request.url()} ${request.failure()?.errorText || ''}`))
  await page.route('**/api/v1/**', async (route) => {
    const url = new URL(route.request().url())
    if (url.pathname.endsWith('/auth/me')) return route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(system) })
    if (route.request().method() === 'GET') return route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ items: [], total: 0, nextCursor: '' }) })
    return route.fulfill({ status: 204, body: '' })
  })
  for (const [name, path] of routes) {
    await page.goto(`${report.base}${path}`)
    const ready = name === 'overview'
      ? page.locator('#dash-title')
      : name === 'patient'
        ? page.getByTestId('patient-form')
        : page.getByTestId('outpatient-step-nav')
    await ready.waitFor({ timeout: 10000 })
    const heading = await page.locator('h1').first().innerText().catch(() => name)
    report.routes.push({ name, path, heading })
    for (const width of [390, 768, 1440]) {
      await page.setViewportSize({ width, height: 1000 })
      await page.waitForTimeout(150)
      const dimensions = await page.evaluate(() => ({ viewport: innerWidth, document: document.documentElement.scrollWidth }))
      assert.ok(dimensions.document <= width + 1, `${name} overflows at ${width}: ${dimensions.document}`)
      report.layouts.push({ name, ...dimensions })
      if (width === 1440) await page.screenshot({ path: resolve(output, `${name}-1440.png`), fullPage: true })
    }
  }
  assert.deepEqual(report.consoleErrors, [])
  assert.deepEqual(report.requestFailures, [])
} finally {
  report.finishedAt = new Date().toISOString()
  await writeFile(resolve(output, 'browser-evidence.json'), JSON.stringify(report, null, 2))
  await browser.close()
  await server.close()
}
console.log(`OUTPATIENT_BROWSER_EVIDENCE_OK routes=${report.routes.length} layouts=${report.layouts.length} consoleErrors=${report.consoleErrors.length} requestFailures=${report.requestFailures.length}`)
