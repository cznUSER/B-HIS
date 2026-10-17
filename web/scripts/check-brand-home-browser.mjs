// Browser regression: the B-HIS brand is the only sidebar home entry.
import assert from 'node:assert/strict'
import { createRequire } from 'node:module'
import net from 'node:net'
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

const system = {
  id: 'system-1',
  username: 'system',
  displayName: '系统管理员',
  roles: ['SYSTEM'],
  permissions: [],
  scopes: [],
  deniedPermissions: [],
  accessControlLoaded: true,
  super: true,
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

try {
  await server.listen()
  browser = await chromium.launch({ headless: true, ...(process.env.BHIS_BROWSER_EXECUTABLE ? { executablePath: process.env.BHIS_BROWSER_EXECUTABLE } : {}) })
  const context = await browser.newContext({ viewport: { width: 1440, height: 900 } })
  await context.addInitScript((user) => {
    sessionStorage.setItem('bhis.session.v1', JSON.stringify({
      accessToken: 'system-token',
      tokenType: 'Bearer',
      expiresIn: 3600,
      expiresAt: Date.now() + 3600000,
      user,
    }))
  }, system)
  const page = await context.newPage()
  await page.route('**/api/v1/**', (route) => route.fulfill({
    status: 200,
    contentType: 'application/json',
    body: JSON.stringify(route.request().url().endsWith('/auth/me') ? system : {}),
  }))

  await page.goto(`http://127.0.0.1:${port}/app/archive/patients/create`)
  await page.locator('[name="patient-name"]').waitFor()
  const homeLinks = page.locator('.sidenav').getByRole('link', { name: /^首页(?:\s|$)/ })
  assert.equal(await homeLinks.count(), 0, 'the sidebar has no standalone home link')

  const brandLink = page.getByRole('link', { name: /B-HIS/ })
  let brandReachedByTab = false
  for (let index = 0; index < 12 && !brandReachedByTab; index += 1) {
    await page.keyboard.press('Tab')
    brandReachedByTab = await brandLink.evaluate((link) => document.activeElement === link)
  }
  assert.equal(brandReachedByTab, true, 'Tab navigation reaches the B-HIS brand link')
  await page.keyboard.press('Enter')
  await page.waitForURL(`http://127.0.0.1:${port}/app`)
  await page.getByRole('heading', { name: '经营概况' }).waitFor()
  assert.equal(await brandLink.getAttribute('aria-current'), 'page', 'brand link marks the active home page')

  await page.goto(`http://127.0.0.1:${port}/app/archive/patients/create`)
  await page.locator('[name="patient-name"]').waitFor()
  await brandLink.click()
  await page.waitForURL(`http://127.0.0.1:${port}/app`)
  await page.getByRole('heading', { name: '经营概况' }).waitFor()
  console.log('brand home browser check passed')
} finally {
  await browser?.close()
  await server.close()
}
