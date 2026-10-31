import assert from 'node:assert/strict'
import { createRequire } from 'node:module'
import net from 'node:net'
import { fileURLToPath } from 'node:url'
import vue from '@vitejs/plugin-vue'
import { createServer } from 'vite'
import { DEFAULT_THEME_TOKENS } from '../src/auth/theme.js'

const require = createRequire(import.meta.url)
const { chromium } = require(process.env.BHIS_BROWSER_DRIVER || 'playwright')
const webRoot = fileURLToPath(new URL('../', import.meta.url))
const probe = net.createServer()
await new Promise((resolve, reject) => { probe.once('error', reject); probe.listen(0, '127.0.0.1', resolve) })
const port = probe.address().port
await new Promise((resolve, reject) => probe.close((error) => error ? reject(error) : resolve()))

const user = {
  id: 'picker-user', username: 'registrar', displayName: '挂号员', roles: ['REGISTRAR'],
  permissions: ['encounter:create'], scopes: ['outpatient:'], deniedPermissions: [],
  accessControlLoaded: true, super: false, developer: false, mustChangePassword: false, cannotModifyPassword: false,
}
const patient = (patientId) => ({
  patientId,
  patientNo: `P-${patientId}`,
  hospPkNo: `H-${patientId}`,
  name: `患者${patientId}`,
  sex: 'M',
  birthday: '1990-01-01',
  idType: 'ID_CARD',
  idNumber: '110101199001011234',
  fareIdentity: 'SELF',
  phone: '13800000000',
  address: '山西省太原市小店区测试路99号',
  status: 'ACTIVE',
})
const rows = Array.from({ length: 51 }, (_, index) => ({
  patientId: index === 0 ? 'patient-beta' : `patient-${index + 1}`,
  patientNo: `P-${String(index + 1).padStart(4, '0')}`,
  name: index === 0 ? '患者beta' : `患者${index + 1}`,
  sex: index % 2 ? 'F' : 'M', idType: 'ID_CARD', idNumberMasked: '11**************34', phoneMasked: '138****5678',
  status: index === 1 ? 'INACTIVE' : 'ACTIVE', selectable: index !== 1,
}))
let browser
const server = await createServer({ configFile: false, root: webRoot, plugins: [vue()], logLevel: 'error', server: { host: '127.0.0.1', port, strictPort: true } })
const pickerCalls = []
const registerPosts = []
let failPickerOnce = false

try {
  await server.listen()
  browser = await chromium.launch({ headless: true, ...(process.env.BHIS_BROWSER_EXECUTABLE ? { executablePath: process.env.BHIS_BROWSER_EXECUTABLE } : {}) })
  const context = await browser.newContext({ viewport: { width: 1440, height: 900 } })
  await context.addInitScript((currentUser) => sessionStorage.setItem('bhis.session.v1', JSON.stringify({ accessToken: 'picker-token', tokenType: 'Bearer', expiresIn: 3600, expiresAt: Date.now() + 3600000, user: currentUser })), user)
  const page = await context.newPage()
  page.on('dialog', (dialog) => dialog.accept().catch(() => {}))
  await page.route('**/api/v1/auth/me', (route) => route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(user) }))
  await page.route('**/api/v1/departments**', (route) => route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ items: [{ departmentId: 'dept-1', deptName: '内科', status: 'ACTIVE' }] }) }))
  await page.route('**/api/v1/doctors**', (route) => route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ items: [{ doctorId: 'doctor-1', doctorName: '张医生', departmentId: 'dept-1', status: 'ACTIVE' }] }) }))
  await page.route('**/api/v1/patients/registration-search', async (route) => {
    const body = route.request().postDataJSON()
    pickerCalls.push(body)
    if (failPickerOnce) {
      failPickerOnce = false
      return route.fulfill({ status: 503, contentType: 'application/json', body: JSON.stringify({ success: false, data: null, error: { code: 'SERVICE_UNAVAILABLE', message: '患者查询暂不可用，请稍后重试', details: {} }, requestId: 'picker-down' }) })
    }
    const matching = body.keyword && body.keyword !== 'fail' ? rows.filter((item) => item.name.toLowerCase().includes(body.keyword.toLowerCase()) || body.keyword.includes('138')) : rows
    const items = matching.slice(body.offset, body.offset + body.limit)
    return route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ items, total: matching.length, nextOffset: body.offset + items.length < matching.length ? body.offset + items.length : null }) })
  })
  await page.route(/\/api\/v1\/patients\/patient-[^/?]+$/, (route) => {
    const id = new URL(route.request().url()).pathname.split('/').at(-1)
    return route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(patient(id)) })
  })
  const shanghaiToday = () => new Date().toLocaleDateString('en-CA', { timeZone: 'Asia/Shanghai' })
  const slotBoardPayload = (visitDate = shanghaiToday()) => ({
    visitDate,
    departments: [{ departmentId: 'dept-1', departmentName: '内科', remainTotal: 1 }],
    doctors: [{
      doctorId: 'doctor-1', departmentId: 'dept-1', doctorName: '张医生', scheduleConfigured: true, remain: 1,
      slots: [{ start: `${visitDate}T09:00:00+08:00`, end: `${visitDate}T09:15:00+08:00`, capacity: 1, used: 0, remain: 1, period: 'AM' }],
    }],
  })
  await page.route('**/api/v1/appearance/effective**', (route) => route.fulfill({
    status: 200,
    contentType: 'application/json',
    body: JSON.stringify({ forceHospitalTheme: false, source: 'hospital', canEditPersonal: true, tokens: DEFAULT_THEME_TOKENS }),
  }))
  await page.route('**/api/v1/auth/me/preferences**', (route) => route.fulfill({
    status: 200,
    contentType: 'application/json',
    body: JSON.stringify({ density: 'standard', sidebarCollapsed: false, defaultPage: 'home', pageSize: 20 }),
  }))
  await page.route('**/api/v1/registers/slot-board**', (route) => {
    const visitDate = new URL(route.request().url()).searchParams.get('visitDate') || shanghaiToday()
    return route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(slotBoardPayload(visitDate)) })
  })
    await page.route(/\/api\/v1\/registers(?:\?.*)?$/, async (route) => {
    if (route.request().method() === 'POST') registerPosts.push(route.request().postDataJSON())
    const url = new URL(route.request().url())
    const selectedPatientId = url.searchParams.get('patientId')
    const selectedVisitDate = url.searchParams.get('visitDate')
    const items = !selectedPatientId || selectedPatientId === 'patient-beta'
      ? [{ registerId: 'register-today-1', registerNo: 'R-TODAY-1', patientId: 'patient-beta', patientName: '患者patient-beta', status: 'NORMAL', visitDate: selectedVisitDate, registerFee: 18.5, registeredAt: `${selectedVisitDate}T06:15:00Z` }]
      : []
    return route.fulfill({ status: route.request().method() === 'POST' ? 201 : 200, contentType: 'application/json', body: JSON.stringify(route.request().method() === 'POST' ? {} : { items }) })
  })

  await page.goto(`http://127.0.0.1:${port}/app/outpatient/register`)
  await page.getByTestId('register-form').waitFor()
  await page.getByTestId('register-today-table').getByText('患者patient-beta', { exact: true }).waitFor()
  const dismissSiteError = page.getByTestId('site-error-dialog').locator('button')
  if (await page.getByTestId('site-error-dialog').isVisible().catch(() => false)) {
    await dismissSiteError.first().click({ force: true }).catch(() => {})
    await page.getByTestId('site-error-dialog').waitFor({ state: 'hidden' }).catch(() => {})
  }
  const layoutRects = async () => page.evaluate(() => Object.fromEntries(['register-patient-panel', 'register-department-panel', 'register-today-panel', 'register-fee-panel', 'register-history-panel'].map((testId) => {
    const rect = document.querySelector(`[data-testid="${testId}"]`).getBoundingClientRect()
    return [testId, { x: Math.round(rect.x), y: Math.round(rect.y), width: Math.round(rect.width) }]
  })))
  const beforeSelection = await layoutRects()
  await page.screenshot({ path: fileURLToPath(new URL('../dist/register-persistent-before-1440.png', import.meta.url)) })
  await page.getByTestId('register-choose-patient').click()
  await page.getByTestId('register-patient-row-patient-beta').waitFor()
  await page.getByTestId('register-patient-row-patient-beta').getByRole('button', { name: '选择' }).click()
  await page.getByTestId('register-patient-identity').getByText('患者patient-beta').waitFor()
  const afterSelection = await layoutRects()
  const relative = (rects) => {
    const originY = rects['register-patient-panel'].y
    const originX = rects['register-patient-panel'].x
    return Object.fromEntries(Object.entries(rects).map(([key, value]) => [key, { x: value.x - originX, y: value.y - originY, width: value.width }]))
  }
  const beforeRel = relative(beforeSelection)
  const afterRel = relative(afterSelection)
  for (const key of Object.keys(beforeRel)) {
    assert.equal(afterRel[key].x, beforeRel[key].x, `${key}.x should stay`)
    assert.equal(afterRel[key].width, beforeRel[key].width, `${key}.width should stay`)
    assert.ok(Math.abs(afterRel[key].y - beforeRel[key].y) <= 2, `${key}.y drift ${afterRel[key].y - beforeRel[key].y} exceeds 2px`)
  }
  const pairedCardHeights = await page.evaluate(() => ({
    patient: Math.round(document.querySelector('[data-testid="register-patient-panel"]').getBoundingClientRect().height),
    department: Math.round(document.querySelector('[data-testid="register-department-panel"]').getBoundingClientRect().height),
    today: Math.round(document.querySelector('[data-testid="register-today-panel"]').getBoundingClientRect().height),
  }))
  assert.ok(pairedCardHeights.patient > 0 && pairedCardHeights.department > 0 && pairedCardHeights.today > 0, `preparation cards must render with positive height: ${JSON.stringify(pairedCardHeights)}`)
  assert.doesNotMatch(await page.getByTestId('register-department-panel').innerText(), /今天.*现场接待|未来.*预约挂号/, 'department card omits the date and reception hint')
  const todayTable = page.getByTestId('register-today-table')
  await todayTable.getByText('患者patient-beta', { exact: true }).waitFor()
  assert.deepEqual(await todayTable.locator('thead th').allTextContents(), ['患者姓名', '挂号费用', '挂号时间'])
  const todayHeadingsFit = await todayTable.locator('thead th').evaluateAll((headings) => headings.map((heading) => heading.scrollWidth <= heading.clientWidth))
  assert.deepEqual(todayHeadingsFit, [true, true, true], `today-work headings must remain fully visible: ${JSON.stringify(todayHeadingsFit)}`)
  assert.deepEqual(await todayTable.locator('tbody tr').first().locator('td').allTextContents(), ['患者patient-beta', '￥18.50', '14:15'])
  await page.screenshot({ path: fileURLToPath(new URL('../dist/register-persistent-after-1440.png', import.meta.url)) })

  await page.goto(`http://127.0.0.1:${port}/app/outpatient/register?patientId=patient-alpha`)
  await page.getByTestId('register-form').waitFor()
  assert.equal(await page.locator('.register-empty-workspace').count(), 1, 'selected patient keeps the preparation workspace')
  assert.equal(await page.locator('.clinical-register-workspace').count(), 0, 'selected patient does not mount a second layout')
  assert.match(await page.getByTestId('register-patient-identity').innerText(), /患者patient-alpha[\s\S]*P-patient-alpha/, 'selected patient shows the archive identity summary')
  const archiveText = await page.getByTestId('register-patient-fields').innerText()
  for (const text of ['身份证号', '手机号', '费别 / 医保', '过敏史', '出生日期', '院内档案号', '联系地址', '自费', '山西省太原市小店区测试路99号']) {
    assert.ok(archiveText.includes(text), `patient archive misses ${text}`)
  }
  await page.screenshot({ path: fileURLToPath(new URL('../dist/register-restored-1440.png', import.meta.url)) })
  await page.getByTestId('register-choose-patient').click()
  await page.getByTestId('register-patient-picker').waitFor()
  await page.getByTestId('register-patient-row-patient-beta').waitFor()
  assert.deepEqual(pickerCalls[0], { keyword: '', offset: 0, limit: 50 }, 'initial request loads the first 50 with no keyword')
  assert.equal(await page.locator('[data-testid^="register-patient-row-"]').count(), 50, `rows missing; picker=${await page.getByTestId('register-patient-picker').innerText()}; requests=${JSON.stringify(pickerCalls)}`)
  assert.equal(await page.getByTestId('register-patient-row-patient-2').getByRole('button', { name: '不可选' }).isDisabled(), true, 'inactive patient is visible and disabled')
  await page.getByRole('button', { name: '下一页' }).click()
  await page.getByTestId('register-patient-row-patient-51').waitFor()
  assert.deepEqual(pickerCalls.at(-1), { keyword: '', offset: 50, limit: 50 }, 'next page keeps the empty keyword and uses offset 50')
  await page.getByTestId('register-patient-keyword').fill('13800000000')
  await page.getByRole('button', { name: '查询', exact: true }).click()
  await page.getByTestId('register-patient-row-patient-beta').waitFor()
  assert.deepEqual(pickerCalls.at(-1), { keyword: '13800000000', offset: 0, limit: 50 }, 'search resets offset and sends the single keyword')

  await page.getByRole('button', { name: '取消', exact: true }).click()
  assert.equal(new URL(page.url()).searchParams.get('patientId'), 'patient-alpha', 'cancel preserves route')
  await page.getByTestId('pending-register-fee').fill('25.50')
  await page.getByTestId('payment-WECHAT').click()
  const beforeCancelFee = await page.getByTestId('pending-register-fee').inputValue()
  const beforeCancelPay = await page.getByTestId('payment-WECHAT').getAttribute('aria-pressed')
  await page.getByTestId('register-choose-patient').click()
  await page.getByRole('button', { name: '取消', exact: true }).click()
  assert.equal(new URL(page.url()).searchParams.get('patientId'), 'patient-alpha')
  assert.equal(await page.getByTestId('pending-register-fee').inputValue(), beforeCancelFee, 'cancel preserves fee draft')
  assert.equal(await page.getByTestId('payment-WECHAT').getAttribute('aria-pressed'), beforeCancelPay, 'cancel preserves payment preference')

  await page.getByTestId('register-choose-patient').click()
  await page.getByTestId('register-patient-keyword').fill('beta')
  await page.getByRole('button', { name: '查询', exact: true }).click()
  await page.getByTestId('register-patient-row-patient-beta').waitFor()
  await page.getByTestId('register-patient-row-patient-beta').getByRole('button', { name: '选择' }).click()
  const switchConfirm = page.getByTestId('outpatient-switch-confirm')
  try {
    await switchConfirm.waitFor({ state: 'visible', timeout: 2000 })
    await switchConfirm.click()
  } catch {}
  await page.getByTestId('register-patient-identity').getByText('患者patient-beta').waitFor()
  assert.equal(new URL(page.url()).searchParams.get('patientId'), 'patient-beta', 'selection replaces patientId route')
  assert.equal(Number(await page.getByTestId('pending-register-fee').inputValue()), 0, 'new patient load resets fee draft')
  assert.equal(await page.getByTestId('payment-CASH').getAttribute('aria-pressed'), 'true', 'new patient load resets payment preference to CASH')
  assert.equal(registerPosts.length, 0, 'selecting a patient never posts a register')

  await page.getByTestId('register-choose-patient').click()
  failPickerOnce = true
  await page.getByTestId('register-patient-keyword').fill('fail')
  await page.getByRole('button', { name: '查询', exact: true }).click()
  await page.getByTestId('register-picker-error').waitFor()
  const siteError = page.getByTestId('site-error-dialog')
  if (await siteError.isVisible().catch(() => false)) {
    await siteError.getByRole('button', { name: '确定' }).click()
    await siteError.waitFor({ state: 'hidden' }).catch(() => {})
  }
  await page.getByTestId('register-picker-error').getByRole('button', { name: '重试' }).click()
  await page.getByTestId('register-patient-row-patient-beta').waitFor()
  assert.equal(pickerCalls.at(-1).keyword, 'fail', 'retry preserves the active keyword')
  assert.equal(registerPosts.length, 0, 'picker retry still has no register side effect')

  await page.screenshot({ path: fileURLToPath(new URL('../dist/register-patient-picker-evidence.png', import.meta.url)), fullPage: true })
  console.log('register patient picker browser check passed')
} finally {
  await browser?.close()
  await server.close()
}
