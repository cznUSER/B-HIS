// Browser regression: charge-only users may enter the patient workspace but cannot create patients.
import assert from 'node:assert/strict'
import { createRequire } from 'node:module'
import net from 'node:net'
import { resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import vue from '@vitejs/plugin-vue'
import { createServer } from 'vite'

const require = createRequire(import.meta.url)
const { chromium } = require(process.env.BHIS_BROWSER_DRIVER || 'playwright')
const webRoot = fileURLToPath(new URL('../', import.meta.url))
const portProbe = net.createServer()
await new Promise((resolveListen, reject) => {
  portProbe.once('error', reject)
  portProbe.listen(0, '127.0.0.1', resolveListen)
})
const port = portProbe.address().port
await new Promise((resolveClose, reject) => portProbe.close((error) => error ? reject(error) : resolveClose()))
const cashier = {
  id: 'cashier-1',
  username: 'cashier',
  displayName: '收费员',
  roles: ['CASHIER'],
  permissions: ['charge:pay', 'charge:refund'],
  scopes: ['outpatient:'],
  deniedPermissions: [],
  accessControlLoaded: true,
  super: false,
  developer: false,
  mustChangePassword: false,
  cannotModifyPassword: false,
}

const server = await createServer({
  configFile: false,
  root: webRoot,
  plugins: [vue()],
  logLevel: 'error',
  server: { host: '127.0.0.1', port, strictPort: true },
})
let browser
let createRequests = 0

try {
  await server.listen()
  const base = `http://127.0.0.1:${port}`
  browser = await chromium.launch({ headless: true, ...(process.env.BHIS_BROWSER_EXECUTABLE ? { executablePath: process.env.BHIS_BROWSER_EXECUTABLE } : {}) })
  const context = await browser.newContext()
  await context.addInitScript((user) => {
    sessionStorage.setItem('bhis.session.v1', JSON.stringify({
      accessToken: 'cashier-token',
      tokenType: 'Bearer',
      expiresIn: 3600,
      expiresAt: Date.now() + 3600000,
      user,
    }))
  }, cashier)
  const page = await context.newPage()
  await page.route('**/api/v1/auth/me', (route) => route.fulfill({
    status: 200,
    contentType: 'application/json',
    body: JSON.stringify(cashier),
  }))
  await page.route('**/api/v1/patients', (route) => {
    if (route.request().method() === 'POST') createRequests += 1
    return route.fulfill({
      status: 403,
      contentType: 'application/json',
      body: JSON.stringify({ success: false, error: { code: 'FORBIDDEN', message: 'forbidden' } }),
    })
  })

  await page.goto(`${base}/app/archive/patients/create`)
  await page.locator('[name="patient-name"]').waitFor()
  assert.equal(await page.getByRole('button', { name: '保存建档', exact: true }).count(), 0, 'charge-only user cannot see a create action')

  await page.getByTestId('patient-form').evaluate((form) => {
    form.dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }))
  })
  await page.getByTestId('patient-error').getByText('当前账号没有患者建档权限').waitFor()
  assert.equal(createRequests, 0, 'unauthorized form submission never reaches the patient API')
  console.log('patient create permission browser check passed')
} finally {
  await browser?.close()
  await server.close()
}
