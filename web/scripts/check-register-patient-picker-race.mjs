import assert from 'node:assert/strict'
import { createRequire } from 'node:module'
import { rm, writeFile } from 'node:fs/promises'
import net from 'node:net'
import { fileURLToPath } from 'node:url'
import vue from '@vitejs/plugin-vue'
import { createServer } from 'vite'

const require = createRequire(import.meta.url)
const { chromium } = require(process.env.BHIS_BROWSER_DRIVER || 'playwright')
const webRoot = fileURLToPath(new URL('../', import.meta.url))
const harnessPath = fileURLToPath(new URL('../.tmp-register-patient-picker-race.html', import.meta.url))
const harness = `<!doctype html>
<html lang="zh-CN">
  <head><meta charset="UTF-8"><title>Register patient picker race check</title></head>
  <body>
    <div id="app"></div>
    <script type="module">
      import { createApp, h, reactive } from 'vue'
      import RegisterPatientPicker from '/src/components/RegisterPatientPicker.vue'

      const state = reactive({ open: false })
      createApp({
        setup() {
          return () => h('main', [
            h('button', { 'data-testid': 'race-open', type: 'button', onClick: () => { state.open = true } }, 'Open picker'),
            h('output', { 'data-testid': 'race-state' }, state.open ? 'open' : 'closed'),
            h(RegisterPatientPicker, {
              open: state.open,
              accessToken: 'race-token',
              onClose: () => { state.open = false },
            }),
          ])
        },
      }).mount('#app')
    </script>
  </body>
</html>`
await writeFile(harnessPath, harness, 'utf8')

function patientRow(patientId, name) {
  return {
    patientId,
    patientNo: `P-${patientId}`,
    name,
    sex: 'M',
    idType: 'ID_CARD',
    idNumberMasked: '11**************34',
    phoneMasked: '138****5678',
    status: 'ACTIVE',
    selectable: true,
  }
}

const probe = net.createServer()
await new Promise((resolve, reject) => { probe.once('error', reject); probe.listen(0, '127.0.0.1', resolve) })
const port = probe.address().port
await new Promise((resolve, reject) => probe.close((error) => error ? reject(error) : resolve()))

let browser
const server = await createServer({
  configFile: false,
  root: webRoot,
  plugins: [vue()],
  logLevel: 'error',
  server: { host: '127.0.0.1', port, strictPort: true },
})

let releaseFirstRequest
let requestCount = 0
try {
  await server.listen()
  browser = await chromium.launch({ headless: true, ...(process.env.BHIS_BROWSER_EXECUTABLE ? { executablePath: process.env.BHIS_BROWSER_EXECUTABLE } : {}) })
  const context = await browser.newContext({ viewport: { width: 390, height: 844 } })
  const page = await context.newPage()
  await page.route('**/api/v1/patients/registration-search', async (route) => {
    requestCount += 1
    if (requestCount === 1) {
      await new Promise((resolve) => { releaseFirstRequest = async () => { await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ items: [patientRow('stale-patient', 'stale response')], total: 1, nextOffset: null }) }); resolve() } })
      return
    }
    await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ items: [patientRow('fresh-patient', 'fresh response')], total: 1, nextOffset: null }) })
  })

  await page.goto(`http://127.0.0.1:${port}/.tmp-register-patient-picker-race.html`)
  await page.getByTestId('race-open').click()
  await page.getByTestId('register-patient-picker').waitFor()
  await page.keyboard.press('Escape')
  await page.getByTestId('race-state').waitFor({ state: 'visible' })
  assert.equal(await page.getByTestId('race-state').textContent(), 'closed', 'Escape closes the picker')

  await page.getByTestId('race-open').click()
  await page.getByTestId('register-patient-row-fresh-patient').waitFor()
  assert.equal(await page.getByTestId('register-patient-row-fresh-patient').isVisible(), true, 'fresh response renders after reopening')

  await releaseFirstRequest()
  await page.waitForTimeout(25)
  assert.equal(await page.getByTestId('register-patient-row-fresh-patient').isVisible(), true, 'stale response cannot overwrite the reopened picker')
  assert.equal(await page.getByTestId('register-patient-row-stale-patient').count(), 0, 'stale row stays discarded')
  console.log(`register picker race check passed at 390px after ${requestCount} requests`)
} finally {
  await browser?.close()
  await server.close()
  await rm(harnessPath, { force: true })
}