import assert from 'node:assert/strict'
import { createHash } from 'node:crypto'
import { fileURLToPath } from 'node:url'
import { createServer } from 'vite'
import vue from '@vitejs/plugin-vue'
import { chromium } from 'playwright'

const root = fileURLToPath(new URL('..', import.meta.url))
const harness = `
import { createApp, defineComponent, h, ref } from '/node_modules/vue/dist/vue.runtime.esm-bundler.js'
import { createRouter, createMemoryHistory, RouterView, useRoute } from '/node_modules/vue-router/dist/vue-router.mjs'
import Frame from '/src/components/OutpatientWorkbenchFrame.vue'
import { applyLogin, clearSession } from '/src/auth/session.js'
import * as context from '/src/workbench/outpatientContext.js'
applyLogin({ accessToken: 'synthetic-p0-only', tokenType: 'Bearer', expiresIn: 3600, user: { roles: ['SYSTEM'], permissions: ['admin:*'] } })
const Draft = defineComponent({ setup() {
  const draft = ref('')
  return () => h('input', { 'data-testid': 'draft', value: draft.value, onInput: (event) => { draft.value = event.target.value } })
} })
const Page = defineComponent({ setup() {
  const route = useRoute()
  return () => h(Frame, { moduleKey: route.params.module }, { default: () => h(Draft) })
} })
const router = createRouter({ history: createMemoryHistory(), routes: [
  { path: '/app/outpatient/:module', name: 'outpatient-module', component: Page },
  { path: '/patients', name: 'patient-search', component: { render: () => h('p', 'search') } },
  { path: '/workbench', name: 'workbench', component: { render: () => h('p', 'workbench') } },
] })
await router.push('/app/outpatient/register' + location.search)
await router.isReady()
createApp({ render: () => h(RouterView) }).use(router).mount('#app')
window.__p0 = { router, context, clearSession }
`
const server = await createServer({
  root, configFile: false, plugins: [vue(), {
    name: 'p0-context-harness',
    configureServer(server) {
      server.middlewares.use((request, response, next) => {
        if (request.url?.split('?')[0] !== '/__p0') return next()
        response.setHeader('Content-Type', 'text/html; charset=utf-8')
        response.end('<!doctype html><meta name="viewport" content="width=device-width,initial-scale=1"><div id="app"></div><script type="module">' + harness + '</script>')
      })
    },
  }], logLevel: 'error',
  optimizeDeps: { noDiscovery: true, include: [] },
  server: { host: '127.0.0.1', port: 0 },
})
let browser
const errors = []
const methods = []
let patientReads = 0
const patients = {
  p1: { patientId: 'p1', patientNo: 'P001', name: '合成患者甲', sex: 'F', birthday: '2000-11-01', fareIdentity: 'SELF', allergyHistory: '青霉素（合成测试）' },
  p2: { patientId: 'p2', patientNo: 'P002', name: '合成患者乙', sex: 'M' },
}
const registers = [
  { registerId: 'r1', registerNo: 'R001', patientId: 'p1', departmentId: 'dep1', doctorId: 'doc1', visitDate: '2026-10-31', status: 'NORMAL' },
  { registerId: 'r2', registerNo: 'R002', patientId: 'p2', departmentId: 'missing', doctorId: 'missing', visitDate: '2026-10-31', status: 'NORMAL' },
]
try {
  await server.listen()
  const base = 'http://127.0.0.1:' + server.httpServer.address().port
  browser = await chromium.launch({ headless: true, channel: 'chrome' })
  const page = await browser.newPage({ viewport: { width: 1280, height: 900 } })
  page.on('pageerror', (error) => { errors.push(error.message); console.error('BROWSER_ERROR', error.message) })
  page.on('console', (message) => { if (message.type() === 'error') console.error('BROWSER_CONSOLE', message.text()) })
  await page.route('**/api/v1/**', async (route) => {
    const request = route.request()
    methods.push(request.method())
    assert.equal(request.method(), 'GET')
    const url = new URL(request.url())
    let body
    if (url.pathname.startsWith('/api/v1/patients/')) {
      patientReads++
      body = patients[url.pathname.split('/').at(-1)]
      if (!body) return route.fulfill({ status: 404, json: { code: 'NOT_FOUND', message: 'missing' } })
    } else if (url.pathname === '/api/v1/registers') {
      body = { items: registers.filter((item) => !url.searchParams.get('patientId') || item.patientId === url.searchParams.get('patientId')), hasMore: false }
    } else if (url.pathname === '/api/v1/departments') body = { items: [{ departmentId: 'dep1', deptName: '合成科室' }] }
    else if (url.pathname === '/api/v1/doctors') body = { items: [{ doctorId: 'doc1', doctorName: '合成医生' }] }
    // P0-C StatusChain mounts in Frame and GETs domain lists; empty items keep context-only harness green.
    else if (url.pathname === '/api/v1/patients') body = { items: [], total: 0, nextOffset: null }
    else if (url.pathname === '/api/v1/orders') body = { items: [] }
    else if (url.pathname === '/api/v1/charges') body = { items: [] }
    else if (url.pathname === '/api/v1/dispenses') body = { items: [] }
    else if (/^\/api\/v1\/registers\/[^/]+\/diagnoses$/.test(url.pathname)) body = { items: [] }
    else throw new Error('Unexpected endpoint ' + url.pathname)
    await route.fulfill({ json: body })
  })
  await page.route('**/health', (route) => route.fulfill({ json: { status: 'ok' }, headers: { Date: 'Sat, 31 Oct 2026 04:00:00 GMT' } }))
  await page.goto(base + '/__p0')
  await page.getByTestId('draft').waitFor({ timeout: 10000 }).catch(async (error) => { console.error(await page.locator('body').innerText()); throw error })
  assert.match(await page.getByTestId('outpatient-preconditions').innerText(), /尚未选择患者/)
  await page.evaluate(() => window.__p0.router.push({ name: 'outpatient-module', params: { module: 'register' }, query: { patientId: 'p1', registerId: 'r1' } }))
  await page.waitForFunction(() => document.querySelector('[data-testid="outpatient-patient-name"]')?.textContent === '合成患者甲')
  assert.match(await page.getByTestId('outpatient-fare').innerText(), /自费/)
  assert.match(await page.getByTestId('outpatient-context').innerText(), /女 · 25岁/)
  assert.match(await page.getByTestId('outpatient-department').innerText(), /合成科室/)
  assert.match(await page.getByTestId('outpatient-doctor').innerText(), /合成医生/)
  assert.match(await page.getByTestId('outpatient-visit-date').innerText(), /2026-10-31/)
  assert.equal(await page.getByTestId('outpatient-allergy').getAttribute('data-risk'), 'high')
  await page.getByTestId('outpatient-allergy').locator('summary').click()
  assert.match(await page.getByTestId('outpatient-allergy').innerText(), /青霉素/)
  await page.getByTestId('draft').fill('上一患者未保存草稿')
  await page.evaluate(() => { void window.__p0.router.push({ name: 'outpatient-module', params: { module: 'register' }, query: { patientId: 'p2', registerId: 'r2' } }) })
  await page.getByTestId('outpatient-switch-cancel').waitFor()
  assert.match(await page.locator('#outpatient-switch-title').innerText(), /切换患者？/)
  assert.match(await page.locator('#outpatient-switch-body').innerText(), /已保存到服务器的数据不会删除/)
  await page.getByTestId('outpatient-switch-cancel').click()
  assert.equal(await page.getByTestId('draft').inputValue(), '上一患者未保存草稿')
  assert.equal(await page.evaluate(() => window.__p0.router.currentRoute.value.query.patientId), 'p1')
  assert.equal(await page.getByTestId('outpatient-patient-name').innerText(), '合成患者甲')
  await page.evaluate(() => { void window.__p0.router.push({ name: 'outpatient-module', params: { module: 'register' }, query: { patientId: 'p2', registerId: 'r2' } }) })
  await page.getByTestId('outpatient-switch-confirm').click()
  await page.waitForFunction(() => document.querySelector('[data-testid="outpatient-patient-name"]')?.textContent === '合成患者乙')
  assert.equal(await page.getByTestId('draft').inputValue(), '')
  assert.equal(await page.getByTestId('outpatient-allergy').innerText(), '过敏史：未记录')
  assert.equal(await page.getByTestId('outpatient-fare').innerText(), '费别：未登记')
  assert.match(await page.getByTestId('outpatient-department').innerText(), /未知/)
  assert.match(await page.getByTestId('outpatient-doctor').innerText(), /未知/)
  const persisted = await page.evaluate(() => JSON.parse(sessionStorage.getItem('bhis.outpatient.ids.v1')))
  assert.deepEqual(persisted, { patientId: 'p2', registerId: 'r2' })
  for (const module of ['register', 'diagnosis', 'order', 'charge', 'dispense', 'refund']) {
    await page.evaluate((module) => window.__p0.router.push({ name: 'outpatient-module', params: { module }, query: {} }), module)
    assert.equal(await page.getByTestId('outpatient-patient-name').innerText(), '合成患者乙')
    assert.match(await page.getByTestId('outpatient-context').innerText(), /R002/)
  }
  const beforeReload = patientReads
  await page.reload()
  await page.getByTestId('draft').waitFor()
  assert.equal(await page.getByTestId('outpatient-patient-name').innerText(), '合成患者乙')
  assert.ok(patientReads > beforeReload, 'Refresh must refetch patient details')
  const screenshot = createHash('sha256').update(await page.screenshot()).digest('hex')
  await page.setViewportSize({ width: 390, height: 844 })
  assert.ok(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1), 'Context must not overflow mobile viewport')
  await page.evaluate(() => window.__p0.clearSession())
  assert.equal(await page.getByTestId('outpatient-patient-name').innerText(), '未选择患者')
  assert.equal(await page.evaluate(() => sessionStorage.getItem('bhis.outpatient.ids.v1')), null)
  assert.deepEqual(errors, [])
  console.log('P0_A_CONTEXT_BROWSER_OK ' + JSON.stringify({ sixModules: 6, patientReads, writes: methods.filter((method) => method !== 'GET').length, cancelPreservesDraft: true, confirmClearsDraft: true, refreshRefetches: true, screenshotSha256: screenshot, errors }))
} finally {
  await browser?.close()
  await server.close()
}
