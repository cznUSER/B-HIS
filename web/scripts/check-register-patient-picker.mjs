import assert from 'node:assert/strict'
import { createRequire } from 'node:module'
import net from 'node:net'
import { fileURLToPath } from 'node:url'
import vue from '@vitejs/plugin-vue'
import { createServer } from 'vite'

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
  assert.deepEqual(await layoutRects(), beforeSelection, 'selecting a patient fills data without changing the preparation layout')
  const pairedCardHeights = await page.evaluate(() => ({
    patient: Math.round(document.querySelector('[data-testid="register-patient-panel"]').getBoundingClientRect().height),
    department: Math.round(document.querySelector('[data-testid="register-department-panel"]').getBoundingClientRect().height),
    today: Math.round(document.querySelector('[data-testid="register-today-panel"]').getBoundingClientRect().height),
  }))
  assert.equal(pairedCardHeights.patient, pairedCardHeights.department, `patient and department cards have equal height: ${JSON.stringify(pairedCardHeights)}`)
  assert.equal(pairedCardHeights.patient, pairedCardHeights.today, `patient and today-work cards have equal height: ${JSON.stringify(pairedCardHeights)}`)
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
  assert.match(await page.getByTestId('register-patient-identity').innerText(), /患者patient-alpha[\s\S]*P-patient-alpha[\s\S]*已读取/, 'selected patient shows the archive identity summary')
  const archiveText = await page.getByTestId('register-patient-detail-table').innerText()
  assert.ok(!archiveText.includes('姓名'), 'patient detail table omits the name already shown in the identity summary')
  for (const text of ['性别', '出生日期', '身份证号', '费别', '手机号', '院内档案号', '联系地址', '自费', '山西省太原市小店区测试路99号']) {
    assert.ok(archiveText.includes(text), `patient archive misses ${text}`)
  }
  const archiveLayout = await page.evaluate(() => {
    const card = document.querySelector('[data-testid="register-patient-panel"]')
    const table = document.querySelector('[data-testid="register-patient-detail-table"]')
    const name = document.querySelector('[data-testid="register-patient-identity"] strong')
    const patientNo = document.querySelector('[data-testid="register-patient-identity"] div span')
    const state = document.querySelector('[data-testid="register-patient-identity"] .register-archive-state')
    const label = table?.querySelector('th')
    const value = table?.querySelector('td')
    const cardRect = card?.getBoundingClientRect()
    const tableRect = table?.getBoundingClientRect()
    return {
      bottomGap: cardRect && tableRect ? Math.round(cardRect.bottom - tableRect.bottom) : null,
      nameFont: name ? Number.parseFloat(getComputedStyle(name).fontSize) : null,
      patientNoFont: patientNo ? Number.parseFloat(getComputedStyle(patientNo).fontSize) : null,
      stateFont: state ? Number.parseFloat(getComputedStyle(state).fontSize) : null,
      labelFont: label ? Number.parseFloat(getComputedStyle(label).fontSize) : null,
      valueFont: value ? Number.parseFloat(getComputedStyle(value).fontSize) : null,
    }
  })
  assert.ok(archiveLayout.bottomGap <= 26, `patient archive should fill the card to its standard inner padding: ${JSON.stringify(archiveLayout)}`)
  assert.deepEqual(
    { nameFont: archiveLayout.nameFont, patientNoFont: archiveLayout.patientNoFont, stateFont: archiveLayout.stateFont, labelFont: archiveLayout.labelFont, valueFont: archiveLayout.valueFont },
    { nameFont: 18, patientNoFont: 12, stateFont: 11, labelFont: 11, valueFont: 12 },
    `patient archive typography should be easier to scan: ${JSON.stringify(archiveLayout)}`,
  )
  const viewportMetrics = await page.evaluate(() => ({
    viewportHeight: document.documentElement.clientHeight,
    pageHeight: document.documentElement.scrollHeight,
    shellViewportHeight: document.querySelector('.shell-main')?.clientHeight ?? 0,
    shellContentHeight: document.querySelector('.shell-main')?.scrollHeight ?? 0,
    shellWidth: document.querySelector('.shell-main')?.clientWidth ?? 0,
    shellAvailableWidth: (() => {
      const main = document.querySelector('.shell-main')
      if (!main) return 0
      const style = getComputedStyle(main)
      return main.clientWidth - Number.parseFloat(style.paddingLeft) - Number.parseFloat(style.paddingRight)
    })(),
    frameWidth: document.querySelector('.outpatient-frame')?.clientWidth ?? 0,
    sections: Array.from(document.querySelectorAll('.outpatient-frame > *, .outpatient-frame-content > *, .clinical-register-workspace > *')).map((element) => {
      const rect = element.getBoundingClientRect()
      return { testId: element.dataset.testid || '', className: String(element.className || ''), top: Math.round(rect.top), height: Math.round(rect.height), bottom: Math.round(rect.bottom) }
    }),
  }))
  await page.screenshot({ path: fileURLToPath(new URL('../dist/register-restored-1440.png', import.meta.url)) })
  console.log(`register viewport ${JSON.stringify(viewportMetrics)}`)
  assert.ok(viewportMetrics.shellContentHeight <= viewportMetrics.shellViewportHeight, `register content must fit the viewport: ${JSON.stringify(viewportMetrics)}`)
  assert.ok(Math.abs(viewportMetrics.frameWidth - Math.min(viewportMetrics.shellAvailableWidth, 1480)) <= 2, `desktop workbench must use the available width: ${JSON.stringify(viewportMetrics)}`)
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
  await page.getByTestId('register-form').getByLabel('挂号费（元）').fill('25.50')
  await page.getByTestId('register-form').getByLabel('接待方式').selectOption('true')
  await page.getByTestId('register-form').getByLabel('预约备注').fill('原有暂填')
  await page.getByTestId('register-form').locator('input[type="date"]').fill('2026-11-20')
  const beforeCancel = await page.getByTestId('register-form').locator('input').evaluateAll((inputs) => inputs.map((input) => input.value))
  await page.getByTestId('register-choose-patient').click()
  await page.getByRole('button', { name: '取消', exact: true }).click()
  assert.equal(new URL(page.url()).searchParams.get('patientId'), 'patient-alpha')
  assert.deepEqual(await page.getByTestId('register-form').locator('input').evaluateAll((inputs) => inputs.map((input) => input.value)), beforeCancel, 'cancel preserves existing draft')

  await page.getByTestId('register-choose-patient').click()
  await page.getByTestId('register-patient-keyword').fill('beta')
  await page.getByRole('button', { name: '查询', exact: true }).click()
  await page.getByTestId('register-patient-row-patient-beta').waitFor()
  await page.getByTestId('register-patient-row-patient-beta').getByRole('button', { name: '选择' }).click()
  await page.getByTestId('register-form').waitFor()
  assert.equal(new URL(page.url()).searchParams.get('patientId'), 'patient-beta', 'selection replaces patientId route')
  assert.equal(Number(await page.getByTestId('register-form').getByLabel('挂号费（元）').inputValue()), 25.5, 'selection preserves fee draft')
  assert.equal(await page.getByTestId('register-form').getByLabel('接待方式').inputValue(), 'true', 'selection preserves appointment switch')
  assert.equal(await page.getByTestId('register-form').locator('input[type="date"]').inputValue(), '2026-11-20', 'selection preserves visit date')
  assert.equal(registerPosts.length, 0, 'selecting a patient never posts a register')

  await page.getByTestId('register-choose-patient').click()
  failPickerOnce = true
  await page.getByTestId('register-patient-keyword').fill('fail')
  await page.getByRole('button', { name: '查询', exact: true }).click()
  await page.getByTestId('register-picker-error').waitFor()
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
