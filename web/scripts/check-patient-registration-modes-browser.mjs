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
  permissions: ['patient:create'],
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
const creates = []

try {
  await server.listen()
  const base = `http://127.0.0.1:${port}`
  browser = await chromium.launch({ headless: true, ...(process.env.BHIS_BROWSER_EXECUTABLE ? { executablePath: process.env.BHIS_BROWSER_EXECUTABLE } : {}) })
  const context = await browser.newContext({ viewport: { width: 1440, height: 1600 } })
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
  await page.route('**/api/v1/auth/me', (route) => route.fulfill({
    status: 200,
    contentType: 'application/json',
    body: JSON.stringify(system),
  }))
  await page.route('**/api/v1/patients', async (route) => {
    if (route.request().method() !== 'POST') return route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ items: [] }) })
    const payload = route.request().postDataJSON()
    creates.push(payload)
    return route.fulfill({
      status: 201,
      contentType: 'application/json',
      body: JSON.stringify({
        ...payload,
        patientId: `patient-${creates.length}`,
        hospPkNo: `H-000${creates.length}`,
        patientNo: `P-000${creates.length}`,
        cardStatus: payload.cardNo ? 'NORMAL' : 'NOT_SENT',
        status: 'ACTIVE',
        createdAt: '2026-10-17T00:00:00Z',
      }),
    })
  })

  await page.goto(`${base}/app/archive/patients/create`)
  const form = page.getByTestId('patient-form')
  await form.locator('[name="patient-name"]').waitFor()
  assert.equal(await form.locator('[name="patient-address"]').count(), 0, 'simple mode hides address')
  assert.equal(await form.locator('[name="patient-card-no"]').count(), 0, 'simple mode hides detailed archive fields')

  await form.locator('[name="patient-name"]').fill('简易档案')
  await form.locator('[name="patient-sex"]').selectOption('F')
  await form.locator('[name="patient-birthday"]').fill('1990-01-15')
  await form.locator('[name="patient-fare"]').selectOption('SELF')
  await form.locator('[name="patient-id"]').fill('110101199001151234')
  await form.locator('[name="patient-phone"]').fill('13800000000')
  await form.getByRole('button', { name: '保存建档', exact: true }).click()
  await page.getByTestId('patient-success').waitFor()
  assert.equal(creates.length, 1)
  assert.equal(creates[0].idType, 'ID_CARD')
  assert.equal(creates[0].phone, '13800000000')
  assert.equal(Object.hasOwn(creates[0], 'address'), false, 'simple request omits address')

  await form.getByRole('button', { name: '继续建档', exact: true }).click()
  await page.getByTestId('patient-mode-switch').getByRole('button', { name: '详细建档' }).click()
  assert.equal(await form.locator('[name="patient-address"]').count(), 1, 'detailed mode shows address')
  await form.locator('[name="patient-name"]').fill('详细档案')
  await form.locator('[name="patient-sex"]').selectOption('M')
  await form.locator('[name="patient-birthday"]').fill('1988-08-08')
  await form.locator('[name="patient-fare"]').selectOption('SELF')
  await form.locator('[name="patient-member-card-no"]').fill('M-1001')
  await form.locator('[name="patient-household-address"]').fill('山西省太原市小店区')
  await form.locator('[name="patient-household-phone"]').fill('03517654321')
  await form.locator('[name="patient-household-postcode"]').fill('030012')
  await form.locator('[name="patient-contact-name"]').fill('李四')
  await form.locator('[name="patient-contact-phone"]').fill('13900000000')
  await form.locator('[name="patient-address"]').fill('太原市示范路')
  await page.evaluate(() => window.scrollTo(0, 0))
  await page.waitForTimeout(100)
  await page.screenshot({ path: fileURLToPath(new URL('../dist/patient-registration-detail-preview.png', import.meta.url)) })
  await form.getByRole('button', { name: '保存建档', exact: true }).click()
  await page.getByTestId('patient-success').waitFor()
  assert.equal(creates.length, 2)
  assert.equal(creates[1].memberCardNo, 'M-1001')
  assert.equal(creates[1].householdAddress, '山西省太原市小店区')
  assert.equal(creates[1].householdPhone, '03517654321')
  assert.equal(creates[1].householdPostcode, '030012')
  assert.equal(creates[1].contactName, '李四')
  assert.equal(creates[1].address, '太原市示范路')
  assert.equal(Object.hasOwn(creates[1], 'patientSource'), false, 'CRM data is excluded')
  console.log('patient registration mode browser check passed')
} finally {
  await browser?.close()
  await server.close()
}
