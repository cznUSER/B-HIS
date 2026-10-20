// 2026-10-14 P1-H05-WEB-USABILITY: execute compiled Vue setup and rendered states.
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { parse, compileScript, compileTemplate } from '@vue/compiler-sfc'
import * as Vue from 'vue'
import * as SSR from 'vue/server-renderer'
import { ApiError, apiError } from '../src/api/errors.js'
import * as registerMaster from '../src/register/master.js'
import * as orderContract from '../src/api/orderContract.js'
import { fareLabel, sexLabel, todayInShanghai } from '../src/api/patientContract.js'
import { saveOrder as saveOrderHttp } from '../src/api/orderApi.js'
import http from 'node:http'

const patient = { patientId: 'patient-live', name: '真实患者', sex: 'M', patientNo: 'P-live' }
const encounter = { registerId: 'reg-live', registerNo: 'G-live', patientId: patient.patientId, patientName: patient.name, departmentId: 'dept-1', doctorId: 'doc-1', status: 'NORMAL' }
const catalog = [{ itemNo: 'LIVE-DRUG', itemName: '真实药品', itemType: 'DRUG', spec: '5mg', units: '瓶', price: 17.75, status: 'ACTIVE' }]
const department = { departmentId: 'dept-1', deptName: '真实科室', status: 'ACTIVE' }
const doctor = { doctorId: 'doc-1', doctorName: '真实医生', departmentId: 'dept-1', status: 'ACTIVE' }
const savedOrder = (status = 'SAVE') => ({ orderId: 'order-live', orderNo: 'R-live', registerId: encounter.registerId, patientId: patient.patientId, departmentId: department.departmentId, doctorId: doctor.doctorId, status, charged: false, totalFee: 37, items: [{ orderItemId: 'line-live', ...catalog[0], itemName: '服务端药品名称', units: '支', price: 18.5, amount: 2, totalFee: 37 }] })
const failure = () => apiError(503, 'HTTP_ERROR', '服务暂不可用')
const componentStub = { props: ['title', 'description'], setup: (props, { slots }) => () => Vue.h('div', [props.title, props.description, slots.default?.(), slots.detail?.()]) }

function visitDateAfter(days) {
  const [year, month, day] = todayInShanghai().split('-').map(Number)
  const result = new Date(Date.UTC(year, month - 1, day + days))
  return [result.getUTCFullYear(), String(result.getUTCMonth() + 1).padStart(2, '0'), String(result.getUTCDate()).padStart(2, '0')].join('-')
}

function scriptWithoutImports(code) {
  return code.replace(/^import[\s\S]*?from\s+['"][^'"]+['"]\s*;?\r?\n/gm, '').replace(/^import\s+['"][^'"]+['"]\s*;?\r?\n/gm, '')
}

function renderFunction(code) {
  const bindings = {}
  code = code.replace(/^import\s+\{([^}]+)\}\s+from\s+"([^"]+)"\r?\n/gm, (_, list, source) => {
    for (const part of list.split(',')) {
      const [name, alias = name] = part.trim().split(/\s+as\s+/)
      bindings[alias] = (source === 'vue' ? Vue : SSR)[name]
    }
    return ''
  })
  return new Function(...Object.keys(bindings), `${code.replace('export function ssrRender', 'function ssrRender')}\nreturn ssrRender`)(...Object.values(bindings))
}

async function view(name, overrides = {}, query = {}) {
  const { descriptor } = parse(readFileSync(new URL(`../src/views/${name}.vue`, import.meta.url), 'utf8'))
  const script = compileScript(descriptor, { id: `check-${name}` })
  const mounted = []
  const route = Vue.reactive({ query })
  const scope = Vue.effectScope()
  const dependencies = {
    ...Vue, ...registerMaster, ...orderContract,
    ApiError, fareLabel, sexLabel, todayInShanghai,
    onMounted: (fn) => mounted.push(fn), useRoute: () => route,
    useRouter: () => ({ replace: async (target) => { route.query = target.query || {} } }),
    useMock: () => false, session: { accessToken: 'test-token' },
    BusinessPageHeader: componentStub, BusinessEmptyState: componentStub, RegisterPatientPicker: componentStub,
    fetchPatient: async () => patient,
    listRegisters: async () => ({ items: [] }), createRegister: async () => encounter,
    cancelRegister: async () => ({ ...encounter, status: 'BACK' }), checkInRegister: async () => ({ ...encounter, status: 'DIAGNOSING' }),
    fetchDepartments: async () => ({ items: [department] }), fetchDoctors: async () => ({ items: [doctor] }),
    fetchCatalog: async () => ({ items: catalog }), fetchInventory: async () => ({ items: [{ itemNo: 'LIVE-DRUG', qtyOnHand: 1 }] }),
    listOrderRegisters: async () => ({ items: [encounter] }), fetchOrderRegister: async () => encounter,
    listOrders: async () => ({ items: [] }), saveOrder: async () => savedOrder(), commitOrder: async () => savedOrder('COMMIT'), abateOrder: async () => savedOrder('ABATE'),
    outpatientContext: { patient: null, register: null, diagnosis: null, diagnoses: [], orders: [], charges: [], dispense: null, history: [], lastAction: '', updatedAt: '' },
    setPatient: () => {}, setRegister: () => {}, setDiagnosis: () => {}, setOrders: () => {}, setCharges: () => {}, setDispense: () => {}, recordAction: () => {},
    ...overrides,
  }
  const names = Object.keys(dependencies).filter((name) => /^[\w$]+$/.test(name))
  const definition = new Function(...names, scriptWithoutImports(script.content).replace('export default', 'return'))(...names.map((name) => dependencies[name]))
  const state = scope.run(() => definition.setup({}, { expose() {} }))
  await Promise.all(mounted.map((fn) => fn()))
  await new Promise((resolve) => setImmediate(resolve))
  const template = compileTemplate({ source: descriptor.template.content, filename: `${name}.vue`, id: `check-${name}`, ssr: true, ssrCssVars: [], compilerOptions: { bindingMetadata: script.bindings } })
  assert.equal(template.errors.length, 0)
  const ssrRender = renderFunction(template.code)
  return {
    state, route, close: () => scope.stop(),
    html: () => SSR.renderToString(Vue.createSSRApp({ setup: () => state, ssrRender }).component('RouterLink', { setup: (_, { slots }) => () => Vue.h('a', slots.default?.()) })),
  }
}

const checks = []
async function check(label, fn) {
  try { await fn(); checks.push({ label, passed: true }); console.log(`PASS ${label}`) }
  catch (error) { checks.push({ label, passed: false }); console.error(`FAIL ${label}: ${error.message}`) }
}

await check('register master failure never exposes mock options or saves', async () => {
  let saveCalls = 0
  const v = await view('RegisterView', { fetchDepartments: async () => { throw failure() }, createRegister: async () => { saveCalls++; return encounter } }, { patientId: patient.patientId })
  try {
    assert.deepEqual(v.state.departments.value, [], 'failed live master must not use mock departments')
    v.state.departmentId.value = 'dept-1'; v.state.doctorId.value = 'doc-1'
    await v.state.onSave()
    assert.equal(saveCalls, 0, 'save must be blocked until master succeeds')
    const html = await v.html()
    assert.match(html, /重新读取科室和医生/)
    assert.match(html, /data-testid="save-register" disabled/)
  } finally { v.close() }
})

await check('register list failure is retryable and does not render empty state', async () => {
  let listFails = true
  const v = await view('RegisterView', { listRegisters: async () => { if (listFails) throw failure(); return { items: [] } } }, { patientId: patient.patientId })
  try {
    assert.doesNotMatch(await v.html(), /当日暂无挂号/)
    assert.match(await v.html(), /重试挂号列表/)
    listFails = false; await v.state.refreshTickets()
    assert.match(await v.html(), /当日暂无挂号/)
  } finally { v.close() }
})

await check('order uses live doctor, catalog fields and inventory', async () => {
  const v = await view('OrderView', {}, { registerId: encounter.registerId })
  try {
    assert.equal(v.state.orderDoctors.value[0]?.name, '真实医生')
    assert.equal(v.state.drugOptions.value[0]?.itemName, '真实药品')
    const line = v.state.lines.value[0]; line.itemNo = 'LIVE-DRUG'; line.amount = '2'
    assert.equal(v.state.lineMoney(line), '35.50')
    assert.equal(v.state.totalFee.value, 35.5)
    assert.equal(v.state.lineShort(line), true)
    assert.match(v.state.warnings.value[0], /真实药品 可用 1/)
    const html = await v.html(); assert.match(html, /5mg/); assert.match(html, /瓶/); assert.match(html, /17.75/)
    assert.doesNotMatch(html, /演示阿莫西林|张医生/)
  } finally { v.close() }
})

await check('order inventory permission is optional for doctor', async () => {
  const v = await view('OrderView', { fetchInventory: async () => { throw apiError(403, 'FORBIDDEN', '没有库存查询权限') } }, { registerId: encounter.registerId })
  try {
    assert.equal(v.state.masterError.value, '')
    assert.equal(v.state.masterReady.value, true)
    assert.equal(v.state.drugOptions.value[0]?.itemNo, 'LIVE-DRUG')
    const line = v.state.lines.value[0]; line.itemNo = 'LIVE-DRUG'; line.amount = '2'
    assert.equal(v.state.lineShort(line), false, 'missing optional inventory must not block order')
  } finally { v.close() }
})

await check('order master failure blocks save without mock fallback then retry recovers', async () => {
  let fails = true, saveCalls = 0
  const v = await view('OrderView', { fetchInventory: async () => { if (fails) throw failure(); return { items: [{ itemNo: 'LIVE-DRUG', qtyOnHand: 1 }] } }, saveOrder: async () => { saveCalls++; return savedOrder() } }, { registerId: encounter.registerId })
  try {
    assert.deepEqual(v.state.drugOptions.value, [])
    await v.state.onSave(); assert.equal(saveCalls, 0)
    assert.match(await v.html(), /重新读取开单数据/)
    fails = false; await v.state.loadMaster()
    assert.equal(v.state.drugOptions.value[0]?.itemNo, 'LIVE-DRUG')
  } finally { v.close() }
})

await check('order register list failure offers retry without suggesting duplicate registration', async () => {
  let fails = true
  const v = await view('OrderView', { listOrderRegisters: async () => { if (fails) throw failure(); return { items: [encounter] } } })
  try {
    const html = await v.html(); assert.match(html, /重试当日挂号/); assert.doesNotMatch(html, /请先挂号|去挂号/)
    fails = false; await v.state.loadRegisters()
    assert.match(await v.html(), /选择已有挂号/)
  } finally { v.close() }
})

for (const action of ['onSave', 'onCommit', 'onAbate']) {
  await check(`${action} success survives failed history refresh and preserves server fields`, async () => {
    let refreshFails = false
    const v = await view('OrderView', { listOrders: async () => { if (refreshFails) throw failure(); return { items: [] } } }, { registerId: encounter.registerId })
    try {
      if (action === 'onAbate') v.state.applyOrder(savedOrder())
      v.state.lines.value[0].itemNo = 'LIVE-DRUG'; v.state.lines.value[0].amount = '2'
      refreshFails = true; await v.state[action]()
      assert.equal(v.state.status.value, action === 'onSave' ? 'SAVE' : action === 'onCommit' ? 'COMMIT' : 'ABATE')
      assert.equal(v.state.errorMessage.value, '', 'a successful write must not become an action failure')
      assert.match(v.state.infoMessage.value, /已暂存|已提交|已作废/)
      assert.equal(v.state.lineMoney(v.state.lines.value[0]), '37.00')
      const html = await v.html(); assert.match(html, /服务端药品名称/); assert.match(html, /18.50/); assert.match(html, /重试处方列表/); assert.doesNotMatch(html, /尚无处方/)
      refreshFails = false; await v.state.refreshOrders()
      assert.doesNotMatch(await v.html(), /重试处方列表/)
    } finally { v.close() }
  })
}

await check('order history failure keeps valid encounter editable', async () => {
  const v = await view('OrderView', { listOrders: async () => { throw failure() } }, { registerId: encounter.registerId })
  try {
    assert.equal(v.state.register.value?.registerId, encounter.registerId)
    assert.equal(v.state.invalidMessage.value, '')
    const html = await v.html(); assert.match(html, /data-testid="order-form"/); assert.match(html, /重试处方列表/); assert.doesNotMatch(html, /尚无处方/)
  } finally { v.close() }
})

await check('route change during order save cannot apply another encounter response', async () => {
  let finishSave
  const delayed = new Promise((resolve) => { finishSave = resolve })
  const v = await view('OrderView', { fetchOrderRegister: async (_, id) => ({ ...encounter, registerId: id }), saveOrder: () => delayed }, { registerId: encounter.registerId })
  try {
    const pending = v.state.onSave()
    v.route.query = { registerId: 'reg-other' }
    await new Promise((resolve) => setImmediate(resolve))
    finishSave(savedOrder()); await pending
    assert.equal(v.state.register.value?.registerId, 'reg-other')
    assert.equal(v.state.orderId.value, '', 'old encounter save must not overwrite new draft')
    assert.equal(v.state.infoMessage.value, '')
  } finally { v.close() }
})

await check('route change during register save cannot add another patient ticket', async () => {
  let finishSave
  const delayed = new Promise((resolve) => { finishSave = resolve })
  const v = await view('RegisterView', { fetchPatient: async (_, id) => ({ ...patient, patientId: id }), createRegister: () => delayed }, { patientId: patient.patientId })
  try {
    v.state.departmentId.value = 'dept-1'; v.state.doctorId.value = 'doc-1'
    const pending = v.state.onSave()
    v.route.query = { patientId: 'patient-other' }
    await new Promise((resolve) => setImmediate(resolve))
    finishSave(encounter); await pending
    assert.equal(v.state.patient.value?.patientId, 'patient-other')
    assert.deepEqual(v.state.tickets.value, [], 'old patient save must not appear on new patient')
    assert.equal(v.state.notice.value, '')
  } finally { v.close() }
})

await check('register date change reloads only for a selected patient and does not duplicate requests', async () => {
  const calls = []
  const nextDate = visitDateAfter(1)
  const v = await view('RegisterView', { listRegisters: async (_, query) => { calls.push(query.visitDate); return { items: [] } } }, { patientId: patient.patientId })
  try {
    assert.equal(calls.length, 1, 'initial patient load requests the current date once')
    v.state.setVisitDate(nextDate)
    await new Promise((resolve) => setImmediate(resolve))
    assert.deepEqual(calls, [todayInShanghai(), nextDate], 'changing the date triggers one matching list request')
  } finally { v.close() }
})

await check('register date change without patient keeps the single today-work query', async () => {
	const calls = []
	const v = await view('RegisterView', { listRegisters: async (_, query) => { calls.push(query.visitDate); return { items: [] } } })
  try {
		assert.deepEqual(calls, [todayInShanghai()])
    v.state.setVisitDate(visitDateAfter(1))
    await new Promise((resolve) => setImmediate(resolve))
		assert.deepEqual(calls, [todayInShanghai()])
  } finally { v.close() }
})

await check('late register response for an older date cannot replace current results', async () => {
  let finishOldDate
  const calls = []
  const oldDate = visitDateAfter(1)
  const currentDate = visitDateAfter(2)
  const v = await view('RegisterView', { listRegisters: async (_, query) => {
    calls.push(query.visitDate)
    if (query.visitDate === oldDate) return new Promise((resolve) => { finishOldDate = resolve })
    return { items: [{ ...encounter, registerId: `reg-${query.visitDate}`, visitDate: query.visitDate }] }
  } }, { patientId: patient.patientId })
  try {
    v.state.setVisitDate(oldDate)
    await new Promise((resolve) => setImmediate(resolve))
    v.state.setVisitDate(currentDate)
    await new Promise((resolve) => setImmediate(resolve))
    assert.deepEqual(v.state.tickets.value.map((item) => item.visitDate), [currentDate])
    finishOldDate({ items: [{ ...encounter, registerId: 'reg-old-date', visitDate: oldDate }] })
    await new Promise((resolve) => setImmediate(resolve))
    assert.deepEqual(v.state.tickets.value.map((item) => item.visitDate), [currentDate])
    assert.deepEqual(calls, [todayInShanghai(), oldDate, currentDate])
  } finally { v.close() }
})

await check('picking and changing encounter keeps patient route context', async () => {
  const v = await view('OrderView', {}, { patientId: patient.patientId })
  try {
    v.state.pickedId.value = encounter.registerId
    await v.state.usePicked()
    assert.equal(v.route.query.patientId, patient.patientId)
    await new Promise((resolve) => setImmediate(resolve))
    await v.state.changeRegister()
    assert.equal(v.route.query.patientId, patient.patientId)
    assert.equal(v.route.query.registerId, undefined)
  } finally { v.close() }
})

await check('existing SAVE line submits its displayed server snapshot in the actual HTTP payload', async () => {
  const writes = []
  const commits = []
  const server = http.createServer(async (req, res) => {
    const chunks = []; for await (const chunk of req) chunks.push(chunk)
    const body = JSON.parse(Buffer.concat(chunks).toString() || '{}')
    if (req.method === 'PATCH') writes.push(body)
    if (req.url.endsWith('/commit')) commits.push(req.url)
    const items = req.method === 'PATCH' ? body.items : savedOrder().items
    res.writeHead(200, { 'Content-Type': 'application/json' })
    res.end(JSON.stringify({ ...savedOrder(req.url.endsWith('/commit') ? 'COMMIT' : 'SAVE'), items, totalFee: items.reduce((sum, item) => sum + item.totalFee, 0) }))
  })
  await new Promise((resolve) => server.listen(0, '127.0.0.1', resolve))
  process.env.VITE_USE_MOCK = 'false'
  process.env.VITE_API_BASE_URL = `http://127.0.0.1:${server.address().port}/api/v1`
  const { commitOrder: commitOrderHttp } = await import('../src/api/orderApi.js')
  const v = await view('OrderView', { saveOrder: saveOrderHttp, commitOrder: commitOrderHttp }, { registerId: encounter.registerId })
  try {
    v.state.applyOrder(savedOrder())
    assert.equal(v.state.totalFee.value, 37)
    await v.state.onCommit()
    assert.equal(v.state.errorMessage.value, '')
    assert.equal(commits.length, 1)
    const { itemName, price, units, spec, totalFee } = writes[0].items[0]
    assert.deepEqual({ itemName, price, units, spec, totalFee }, { itemName: '服务端药品名称', price: 18.5, units: '支', spec: '5mg', totalFee: 37 })
    assert.equal(v.state.totalFee.value, 37)
  } finally { v.close(); await new Promise((resolve) => server.close(resolve)) }
})

if (checks.some((result) => !result.passed)) process.exitCode = 1
else console.log(`clinical usability checks passed (${checks.length})`)
