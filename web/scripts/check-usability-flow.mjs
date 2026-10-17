// 2026-10-14 P1-H05-WEB-USABILITY: execute the actual Vue setup and API behavior.
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { parse, compileScript, compileTemplate } from '@vue/compiler-sfc'
import { transform } from 'esbuild'
import * as vue from 'vue'
import { apiError, ApiError } from '../src/api/errors.js'
import * as chargeContract from '../src/api/chargeContract.js'
import * as dispenseContract from '../src/api/dispenseContract.js'
import { loadPendingCharge, payCharge, listCharges, fetchCharge } from '../src/api/chargeApi.js'
import { confirmDispense, returnDispense, listDispenses, listTodayPending } from '../src/api/dispenseApi.js'

const failures = []
globalThis.Document ||= class Document {}
globalThis.ShadowRoot ||= class ShadowRoot {}
async function check(name, run) {
  try { await run(); console.log(`PASS ${name}`) }
  catch (error) { failures.push(name); console.error(`FAIL ${name}: ${error.message}`) }
}
const readFailure = () => apiError(503, 'SERVICE_UNAVAILABLE', '服务暂不可用')
const subject = { key: 'ord-1', patientId: 'patient-1', patientName: '患者', registerId: 'reg-1', orderId: 'ord-1', scene: 'PENDING', orderItems: [{ itemNo: 'LIVE-DRUG', itemType: 'DRUG', units: '盒' }] }
const pending = { scene: 'PENDING', patient: { patientName: '患者', patientNo: 'P1' }, register: { registerId: 'reg-1', summary: '挂号' }, order: { recipeNo: 'R1', status: 'COMMIT' }, preview: { orderId: 'ord-1', squareFee: 20, amountDueCents: 2000, items: [], stockBlocked: false } }
const charge = { chargeId: 'charge-1', squareNo: 'SQ-1', status: 'NORMAL', paid: true, items: [{ chargeItemId: 'ci-1', canBack: true, refundableQty: 2, itemName: '药品', qty: 2, unit: '盒' }] }
const slip = { dispenseId: 'dispense-1', dispenseNo: 'FY-1', chargeId: 'charge-1', items: [{ dispenseItemId: 'di-1', chargeItemId: 'ci-1', returnableQty: 2, dispenseQty: 2, itemName: '药品' }] }

// A tiny Vue renderer keeps lifecycle/watch behavior real without requiring a browser.
const renderer = vue.createRenderer({
  createElement: (tag) => ({ tag, tagName: tag.toUpperCase(), children: [], props: {}, style: {}, getRootNode: () => ({ activeElement: null }), addEventListener() {}, removeEventListener() {}, setAttribute() {}, removeAttribute() {}, focus() {} }),
  createText: (text) => ({ text }), createComment: (text) => ({ comment: text }),
  setText: (node, text) => { node.text = text }, setElementText: (node, text) => { node.text = text },
  parentNode: (node) => node?.parent,
  nextSibling: (node) => { const siblings = node?.parent?.children; return siblings?.[siblings.indexOf(node) + 1] || null },
  patchProp: (node, key, _before, after) => { node.props[key] = after },
  insert: (node, parent, anchor) => { const before = node.parent?.children; if (before && before.includes(node)) before.splice(before.indexOf(node), 1); node.parent = parent; const index = parent.children.indexOf(anchor); parent.children.splice(index < 0 ? parent.children.length : index, 0, node) },
  remove: (node) => { const siblings = node?.parent?.children; if (siblings && siblings.includes(node)) siblings.splice(siblings.indexOf(node), 1); if (node) node.parent = null },
})
async function settle() { for (let n = 0; n < 10; n += 1) { await Promise.resolve(); await vue.nextTick() } }
async function mountView(name, api, initialQuery = {}) {
  const path = new URL(`../src/views/${name}.vue`, import.meta.url)
  const { descriptor } = parse(readFileSync(path, 'utf8'), { filename: name })
  const script = compileScript(descriptor, { id: name })
  const template = compileTemplate({ source: descriptor.template.content, filename: name, id: name, compilerOptions: { bindingMetadata: script.bindings } })
  assert.equal(template.errors.length, 0, 'Vue template must compile')
  const route = vue.reactive({ query: initialQuery })
  const modules = {
    vue, 'vue-router': { useRoute: () => route },
    '../api/errors.js': { ApiError },
    '../api/config.js': { useMock: () => false },
    '../api/chargeApi.js': api, '../api/dispenseApi.js': api,
    '../api/chargeContract.js': chargeContract, '../api/dispenseContract.js': dispenseContract,
    '../api/mockDispenses.js': { DEMO_CASES: [] },
    '../auth/session.js': { session: { accessToken: 'test-token', user: { displayName: '操作员' } } },
    '../workbench/outpatientContext.js': {
      outpatientContext: { patient: null, register: null, diagnosis: null, diagnoses: [], orders: [], charges: [], dispense: null, history: [], lastAction: '', updatedAt: '' },
      setPatient: () => {}, setRegister: () => {}, setDiagnosis: () => {}, setOrders: () => {}, setCharges: () => {}, setDispense: () => {}, recordAction: () => {},
    },
  }
  const empty = { render: () => vue.h('div') }
  function dependency(id) {
    if (id.endsWith('.vue')) return { __esModule: true, default: empty }
    assert.ok(modules[id], `unsupported view dependency ${id}`)
    return modules[id]
  }
  async function evaluate(code) {
    const output = await transform(code, { format: 'cjs', loader: 'js' })
    const module = { exports: {} }
    new Function('require', 'module', 'exports', output.code)(dependency, module, module.exports)
    return module.exports
  }
  const component = (await evaluate(script.content)).default
  component.render = (await evaluate(template.code)).render
  const app = renderer.createApp(component)
  app.component('RouterLink', { props: ['to'], setup: (props, context) => () => vue.h('a', { 'data-route': props.to }, context.slots.default?.()) })
  const container = { children: [] }
  app.mount(container)
  await settle()
  return { state: app._instance.setupState, route, container, close: () => app.unmount() }
}
const chargeApi = () => ({ lookupChargeSubjects: async () => ({ items: [subject] }), loadPendingCharge: async () => structuredClone(pending), listCharges: async () => ({ items: [] }), fetchCharge: async () => structuredClone(charge), payCharge: async () => ({ charge }), refundCharge: async () => ({ fullRefund: true, abatedSquare: { ...charge, status: 'ABATE' } }) })
const dispenseApi = () => ({ listDispenses: async () => ({ items: [slip] }), listTodayPending: async () => ({ items: [] }), searchPending: async () => ({ items: [] }), confirmDispense: async () => ({ ...slip, dispenses: [slip] }), fetchDispense: async () => structuredClone(slip), returnDispense: async () => ({ ...slip, stockNote: '库存已回加 1' }), resetDispenseDemo: async () => {} })

await check('charge responds to view query and browser-back query', async () => {
  const view = await mountView('ChargeView', chargeApi())
  try {
    view.route.query = { view: 'records' }; await settle(); assert.equal(view.state.viewMode, 'records')
    view.route.query = {}; await settle(); assert.equal(view.state.viewMode, 'pending')
  } finally { view.close() }
})
await check('charge opens a changed charge_id and closes when browser back removes it', async () => {
  const ids = []; const api = chargeApi(); api.fetchCharge = async (_token, id) => { ids.push(id); return { ...charge, chargeId: id } }
  const view = await mountView('ChargeView', api)
  try {
    view.route.query = { view: 'records', charge_id: 'charge-2' }; await settle()
    assert.equal(view.state.refundTarget?.chargeId, 'charge-2'); assert.equal(view.state.refundOpen, true)
    view.route.query = { view: 'records' }; await settle(); assert.equal(view.state.refundOpen, false)
    assert.deepEqual(ids, ['charge-2'])
  } finally { view.close() }
})
await check('dispense responds to query context and browser back', async () => {
  const view = await mountView('DispenseView', dispenseApi())
  try {
    view.route.query = { view: 'issued', charge_id: 'charge-1' }; await settle()
    assert.equal(view.state.viewMode, 'issued'); assert.equal(view.state.chargeContext, 'charge-1')
    view.route.query = {}; await settle(); assert.equal(view.state.viewMode, 'pending'); assert.equal(view.state.chargeContext, '')
  } finally { view.close() }
})
await check('leaving a pending charge query ignores its late read and clears loading', async () => {
  let finish; const api = dispenseApi(); api.searchPending = () => new Promise((resolve) => { finish = resolve })
  const view = await mountView('DispenseView', api)
  try {
    view.route.query = { charge_id: 'charge-1' }; await settle(); assert.equal(view.state.loading, true)
    view.route.query = { view: 'issued' }; await settle(); assert.equal(view.state.loading, false)
    finish({ items: [{ patientId: 'old-patient', availableQty: 1, stockQty: 1 }], empty: '' }); await settle()
    assert.deepEqual(view.state.rows, []); assert.equal(view.state.viewMode, 'issued')
  } finally { view.close() }
})
for (const action of ['onToday', 'onSearch', 'onRefreshStock', 'refreshAfterWrite']) await check(`${action} ignores an old manual response after navigation to another charge`, async () => {
  let finish; const api = dispenseApi()
  const oldRead = () => new Promise((resolve) => { finish = resolve })
  api.listTodayPending = oldRead
  api.searchPending = async (_token, value) => value === 'charge-B' ? { items: [{ chargeItemId: 'ci-B', chargeId: 'charge-B', patientId: 'patient-B', availableQty: 1, stockQty: 1 }] } : oldRead()
  const view = await mountView('DispenseView', api)
  try {
    view.state.query = 'manual-A'; view.state.searched = true; view.state.loadMode = 'search'
    const before = view.state[action](); await settle()
    view.route.query = { charge_id: 'charge-B' }; await settle()
    assert.equal(view.state.rows[0].chargeId, 'charge-B')
    finish({ items: [{ chargeItemId: 'ci-A', chargeId: 'charge-A', patientId: 'patient-A', availableQty: 2, stockQty: 2 }] })
    await before; await settle()
    assert.equal(view.state.rows[0].chargeId, 'charge-B', 'late manual read must not replace the route-selected charge')
  } finally { view.close() }
})
await check('paid charge is retained when following subject read fails', async () => {
  const api = chargeApi(); const view = await mountView('ChargeView', api)
  try {
    api.lookupChargeSubjects = async () => { throw readFailure() }
    await view.state.onPay(); await settle()
    assert.equal(view.state.paidSquareNo, 'SQ-1'); assert.equal(view.state.paidChargeId, 'charge-1')
    assert.match(view.state.payMessage, /收费成功/); assert.equal(view.state.errorMessage, '')
    assert.ok(view.state.refreshWarning); assert.equal(view.state.canPay, false)
  } finally { view.close() }
})
await check('a pending payment cannot be applied to a different selected patient', async () => {
  let finish; const api = chargeApi(); api.payCharge = () => new Promise((resolve) => { finish = resolve })
  api.loadPendingCharge = async (_token, current) => ({ ...structuredClone(pending), preview: { ...pending.preview, orderId: current.orderId }, patient: { patientName: current.patientName } })
  const view = await mountView('ChargeView', api)
  try {
    const paying = view.state.onPay(); await settle()
    await view.state.openSubject({ ...subject, key: 'ord-2', orderId: 'ord-2', patientId: 'patient-2', patientName: '患者B' })
    assert.equal(view.state.activeKey, 'ord-1', 'patient switching must remain locked while a write is pending')
    finish({ charge }); await paying; await settle()
    assert.equal(view.state.paidOrderId, 'ord-1'); assert.equal(view.state.activeKey, 'ord-1')
  } finally { view.close() }
})
await check('refund keeps success and original abated state when reads fail', async () => {
  const api = chargeApi(); const view = await mountView('ChargeView', api)
  try {
    await view.state.openRefund('charge-1'); view.state.refundPicks['ci-1'] = true
    api.listCharges = async () => { throw readFailure() }; api.loadPendingCharge = async () => { throw readFailure() }
    await view.state.onRefund(); await settle()
    assert.match(view.state.refundMessage, /已全额退费/); assert.equal(view.state.refundError, '')
    assert.equal(view.state.refundTarget.status, 'ABATE'); assert.ok(view.state.refreshWarning)
  } finally { view.close() }
})
await check('discounted partial refund stays blocked while full refund remains available', async () => {
  let refunds = 0
  const api = chargeApi()
  api.fetchCharge = async () => ({ ...structuredClone(charge), favourableFee: 4 })
  api.refundCharge = async () => { refunds += 1; return { fullRefund: true, abatedSquare: { ...charge, status: 'ABATE' } } }
  const view = await mountView('ChargeView', api)
  try {
    await view.state.openRefund('charge-1')
    view.state.refundPicks['ci-1'] = true
    view.state.refundQty['ci-1'] = 1
    await settle()
    assert.equal(view.state.discountedPartialRefund, true)
    await view.state.onRefund()
    assert.equal(refunds, 0)
    view.state.refundQty['ci-1'] = 2
    await settle()
    assert.equal(view.state.discountedPartialRefund, false)
    await view.state.onRefund()
    assert.equal(refunds, 1)
  } finally { view.close() }
})
await check('a refund response cannot overwrite a newly routed charge dialog', async () => {
  let finish; const api = chargeApi(); api.fetchCharge = async (_token, id) => ({ ...structuredClone(charge), chargeId: id })
  api.refundCharge = () => new Promise((resolve) => { finish = resolve })
  const view = await mountView('ChargeView', api)
  try {
    await view.state.openRefund('charge-1'); view.state.refundPicks['ci-1'] = true
    const writing = view.state.onRefund(); await settle()
    view.route.query = { view: 'records', charge_id: 'charge-2' }; await settle()
    finish({ fullRefund: true, abatedSquare: { ...charge, status: 'ABATE' } }); await writing; await settle()
    assert.equal(view.state.refundTarget.chargeId, 'charge-2'); assert.equal(view.state.refundOpen, true)
  } finally { view.close() }
})
await check('confirm dispense keeps successful slip when pending refresh fails', async () => {
  const api = dispenseApi(); const view = await mountView('DispenseView', api)
  try {
    view.state.loadMode = 'search'; view.state.searched = true
    api.searchPending = async () => { throw readFailure() }
    await view.state.submit([{ selected: true, dispenseQty: 1 }]); await settle()
    assert.match(view.state.infoMessage, /已发药 FY-1/); assert.equal(view.state.errorMessage, '')
    assert.ok(view.state.refreshWarning); assert.equal(view.state.pendingNeedsRefresh, true)
  } finally { view.close() }
})
await check('partial dispense receipts survive a later group failure and failed refresh locks retry', async () => {
  const api = dispenseApi(); let writes = 0
  const view = await mountView('DispenseView', api)
  try {
    view.state.loadMode = 'search'; view.state.searched = true
    api.confirmDispense = async () => { writes += 1; const error = readFailure(); error.partialDispenses = [slip]; throw error }
    api.searchPending = async () => { throw readFailure() }; api.listDispenses = async () => { throw readFailure() }
    await view.state.submit([{ selected: true, dispenseQty: 1 }]); await settle()
    assert.match(view.state.infoMessage, /FY-1/); assert.match(view.state.infoMessage, /部分/)
    assert.equal(view.state.pendingNeedsRefresh, true); assert.ok(view.state.refreshWarning)
    assert.ok(view.state.issued.some((item) => item.dispenseNo === 'FY-1'))
    await view.state.submit([{ selected: true, dispenseQty: 1 }]); assert.equal(writes, 1, 'refresh failure must lock resubmission')
  } finally { view.close() }
})
await check('return keeps successful result when pending refresh fails', async () => {
  const api = dispenseApi(); const view = await mountView('DispenseView', api)
  try {
    await view.state.openReturn(slip); view.state.loadMode = 'search'; view.state.searched = true
    api.searchPending = async () => { throw readFailure() }
    await view.state.onReturn(); await settle()
    assert.equal(view.state.returnResult.dispenseNo, 'FY-1'); assert.equal(view.state.returnError, '')
    assert.ok(view.state.refreshWarning)
  } finally { view.close() }
})
await check('a pending return locks selection of another return slip', async () => {
  let finish; const api = dispenseApi(); api.returnDispense = () => new Promise((resolve) => { finish = resolve })
  api.fetchDispense = async (_token, id) => ({ ...structuredClone(slip), dispenseId: id })
  const view = await mountView('DispenseView', api)
  try {
    await view.state.openReturn(slip)
    const writing = view.state.onReturn(); await settle()
    await view.state.openReturn({ ...slip, dispenseId: 'dispense-2' })
    assert.equal(view.state.returnSlip.dispenseId, 'dispense-1')
    finish({ ...slip, stockNote: '库存已回加 1' }); await writing; await settle()
  } finally { view.close() }
})
await check('refund links target implemented views and preserve charge context', async () => {
  const view = await mountView('ChargeView', chargeApi())
  try {
    view.state.refundOpen = true; view.state.refundTarget = { ...charge, items: [{ ...charge.items[0], dispenseBlocked: true }] }; await settle()
    const all = []; const walk = (node) => { if (node.props?.['data-route']) all.push(node.props['data-route']); for (const child of node.children || []) walk(child) }; walk(view.container)
    assert.ok(all.some((to) => to.params?.module === 'dispense' && to.query?.view === 'issued' && to.query?.charge_id === 'charge-1'))
    assert.ok(!all.some((to) => to.params?.module === 'refund'))
  } finally { view.close() }
})

process.env.VITE_USE_MOCK = 'false'
process.env.VITE_API_BASE_URL = 'http://behavior.test/api/v1'
const originalFetch = globalThis.fetch
const dto = { chargeId: 'charge-1', squareNo: 'SQ-1', orderId: 'ord-1', registerId: 'reg-1', patientId: 'patient-1', totalFee: 20, squareFee: 20, favourableFee: 0, status: 'NORMAL', items: [{ chargeItemId: 'ci-1', orderItemId: 'oi-1', itemNo: 'LIVE-DRUG', itemName: '真实药品', qty: 2, unitPrice: 10, totalFee: 20 }] }
const slipDto = { dispenseId: 'dispense-1', dispenseNo: 'FY-1', orderId: 'ord-1', chargeId: 'charge-1', registerId: 'reg-1', patientId: 'patient-1', items: [{ dispenseItemId: 'di-1', chargeItemId: 'ci-1', dispenseQty: 2, returnedQty: 0 }] }
const json = (body, status = 200) => new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } })
try {
  await check('real charge stock uses inventory for a drug absent from fixtures', async () => {
    const calls = []
    globalThis.fetch = async (url) => { const path = new URL(url).pathname; calls.push(path); if (path.endsWith('/inventory/items')) return json({ items: [{ itemNo: 'LIVE-DRUG', qtyOnHand: 1, units: '盒' }] }); if (path.endsWith('/charge-preview')) return json(dto); if (path.endsWith('/orders/ord-1')) return json({ orderId: 'ord-1', items: subject.orderItems }); throw new Error(path) }
    const result = await loadPendingCharge('test-token', subject)
    assert.equal(result.preview.stockBlocked, true); assert.equal(result.preview.items[0].stockAvailable, 1)
    assert.ok(calls.some((path) => path.endsWith('/inventory/items')))
  })
  await check('real inventory failure rejects pending load instead of enabling charge', async () => {
    globalThis.fetch = async (url) => new URL(url).pathname.endsWith('/inventory/items') ? json({ error: { code: 'SERVICE_UNAVAILABLE', message: '库存服务失败' } }, 503) : json(dto)
    await assert.rejects(() => loadPendingCharge('test-token', subject), /库存服务失败/)
  })
  await check('real payment retains successful DTO when detail GET fails', async () => {
    globalThis.fetch = async (url, init) => { const path = new URL(url).pathname; if (path.endsWith('/pay')) return json({ paymentId: 'payment-1', paymentNo: 'PAY-1', chargeId: 'charge-1', direction: 'CHARGE', payType: 'CASH', amount: 20, status: 'SUCCEEDED', paidAt: '2026-10-14T01:00:00Z' }); if (init.method === 'POST') return json(dto, 201); return json({ error: { code: 'SERVICE_UNAVAILABLE', message: '刷新失败' } }, 503) }
    const result = await payCharge('test-token', { orderId: 'ord-1', payType: 'CASH', amount: 20 })
    assert.equal(result.charge.squareNo, 'SQ-1'); assert.equal(result.charge.paid, true); assert.equal(result.payment.paymentNo, 'PAY-1'); assert.ok(result.refreshWarning)
  })
  await check('charge label read failure remains an error instead of an empty filtered list', async () => {
    globalThis.fetch = async (url) => new URL(url).pathname.endsWith('/charges') ? json({ items: [{ ...dto, paidAt: '2026-10-14T01:00:00Z' }] }) : json({ error: { code: 'SERVICE_UNAVAILABLE', message: '患者读取失败' } }, 503)
    await assert.rejects(() => listCharges('test-token', { keyword: '患者' }), /患者读取失败/)
  })
  await check('refund cannot open an invalid order detail as safe-to-refund', async () => {
    globalThis.fetch = async (url) => new URL(url).pathname.endsWith('/orders/ord-1') ? json({ status: 'PROC' }) : json(dto)
    await assert.rejects(() => fetchCharge('test-token', 'charge-1'))
  })
  await check('refund uses actual net dispensed quantities after a full or partial return', async () => {
    let returnedQty = 2; let orderStatus = 'PROC'; const calls = []
    globalThis.fetch = async (url) => {
      const parsed = new URL(url); const path = parsed.pathname; calls.push(parsed)
      if (path.endsWith('/charges/charge-1')) return json({ ...dto, paidAt: '2026-10-14T01:00:00Z' })
      if (path.endsWith('/orders/ord-1')) return json({ orderId: 'ord-1', orderNo: 'RX-1', registerId: 'reg-1', patientId: 'patient-1', departmentId: 'dept-1', doctorId: 'doctor-1', status: orderStatus, charged: true, totalFee: 20, items: [{ itemNo: 'LIVE-DRUG', itemName: '真实药品', itemType: 'DRUG', amount: 2, price: 10, totalFee: 20 }] })
      if (path.endsWith('/dispenses')) return json({ items: [{ ...slipDto, items: [{ ...slipDto.items[0], dispensedQty: 2, returnedQty }] }] })
      if (path.endsWith('/patients/patient-1')) return json({ patientId: 'patient-1', name: '患者', patientNo: 'P1', status: 'ACTIVE' })
      if (path.endsWith('/registers/reg-1')) return json({ registerId: 'reg-1', registerNo: 'GH-1' })
      throw new Error(path)
    }
    const returned = await fetchCharge('test-token', 'charge-1')
    assert.equal(returned.items[0].dispenseBlocked, false, 'PROC status must not block a fully returned drug')
    assert.equal(returned.items[0].netDispensedQty, 0)
    returnedQty = 1
    assert.equal((await fetchCharge('test-token', 'charge-1')).items[0].dispenseBlocked, true)
    orderStatus = 'COMMIT'
    assert.equal((await fetchCharge('test-token', 'charge-1')).items[0].dispenseBlocked, true, 'an outstanding dispense blocks regardless of order status')
    assert.ok(calls.some((url) => url.pathname.endsWith('/dispenses') && url.searchParams.get('chargeId') === 'charge-1'))
  })
  await check('refund stays unavailable if dispense history cannot be read', async () => {
    globalThis.fetch = async (url) => {
      const path = new URL(url).pathname
      if (path.endsWith('/charges/charge-1')) return json(dto)
      if (path.endsWith('/orders/ord-1')) return json({ orderId: 'ord-1', orderNo: 'RX-1', registerId: 'reg-1', patientId: 'patient-1', departmentId: 'dept-1', doctorId: 'doctor-1', status: 'PROC', charged: true, totalFee: 20, items: [{ itemNo: 'LIVE-DRUG', itemName: '真实药品', itemType: 'DRUG', amount: 2, price: 10, totalFee: 20 }] })
      if (path.endsWith('/dispenses')) return json({ error: { code: 'SERVICE_UNAVAILABLE', message: '发药记录读取失败' } }, 503)
      if (path.endsWith('/patients/patient-1')) return json({ patientId: 'patient-1', name: '患者', patientNo: 'P1', status: 'ACTIVE' })
      if (path.endsWith('/registers/reg-1')) return json({ registerId: 'reg-1', registerNo: 'GH-1' })
      throw new Error(path)
    }
    await assert.rejects(() => fetchCharge('test-token', 'charge-1'), /发药记录读取失败/)
  })
  await check('malformed charge and dispense lists reject instead of showing empty success', async () => {
    globalThis.fetch = async () => json({ unexpected: [] })
    await assert.rejects(() => listCharges('test-token'))
    await assert.rejects(() => listDispenses('test-token'))
  })
  await check('malformed real dispense order never falls back to fixture metadata', async () => {
    globalThis.fetch = async (url) => {
      const path = new URL(url).pathname
      if (path.endsWith('/charges')) return json({ items: [{ ...dto, paidAt: '2026-10-14T01:00:00Z' }] })
      if (path.endsWith('/inventory/items')) return json({ items: [{ itemNo: 'LIVE-DRUG', qtyOnHand: 2 }] })
      if (path.endsWith('/orders/ord-1')) return json({ status: 'COMMIT' })
      if (path.endsWith('/dispenses')) return json({ items: [] })
      if (path.endsWith('/patients/patient-1')) return json({ patientId: 'patient-1', name: '患者', patientNo: 'P1', status: 'ACTIVE' })
      if (path.endsWith('/registers/reg-1')) return json({ registerId: 'reg-1', registerNo: 'GH-1' })
      throw new Error(path)
    }
    await assert.rejects(() => listTodayPending('test-token'))
  })
  await check('real dispense refuses a charge line absent from its actual order', async () => {
    globalThis.fetch = async (url) => {
      const path = new URL(url).pathname
      if (path.endsWith('/charges')) return json({ items: [{ ...dto, paidAt: '2026-10-14T01:00:00Z' }] })
      if (path.endsWith('/inventory/items')) return json({ items: [{ itemNo: 'LIVE-DRUG', qtyOnHand: 2 }] })
      if (path.endsWith('/orders/ord-1')) return json({ orderId: 'ord-1', orderNo: 'RX-1', registerId: 'reg-1', patientId: 'patient-1', departmentId: 'dept-1', doctorId: 'doctor-1', status: 'COMMIT', charged: true, totalFee: 20, items: [] })
      if (path.endsWith('/dispenses')) return json({ items: [] })
      if (path.endsWith('/patients/patient-1')) return json({ patientId: 'patient-1', name: '患者', patientNo: 'P1', status: 'ACTIVE' })
      if (path.endsWith('/registers/reg-1')) return json({ registerId: 'reg-1', registerNo: 'GH-1' })
      throw new Error(path)
    }
    await assert.rejects(() => listTodayPending('test-token'))
  })
  for (const operation of ['confirm', 'return']) await check(`real ${operation} retains write response when label reads fail`, async () => {
    globalThis.fetch = async (_url, init) => init.method === 'POST' ? json(slipDto, operation === 'confirm' ? 201 : 200) : json({ error: { code: 'SERVICE_UNAVAILABLE', message: '刷新失败' } }, 503)
    const result = operation === 'confirm' ? await confirmDispense('test-token', { items: [{ orderId: 'ord-1', chargeId: 'charge-1', chargeItemId: 'ci-1', dispenseQty: 2 }] }) : await returnDispense('test-token', { dispenseId: 'dispense-1', reason: '患者要求', items: [{ dispenseItemId: 'di-1', returnQty: 1 }] })
    assert.equal(result.dispenseNo, 'FY-1'); assert.equal(result.chargeId, 'charge-1'); assert.ok(result.refreshWarning)
  })
  await check('real grouped dispense exposes receipts from prior successful groups', async () => {
    let writes = 0
    globalThis.fetch = async (_url, init) => {
      if (init.method === 'POST' && ++writes === 1) return json(slipDto, 201)
      return json({ error: { code: 'SERVICE_UNAVAILABLE', message: '第二单失败' } }, 503)
    }
    await assert.rejects(() => confirmDispense('test-token', { items: [{ orderId: 'ord-1', chargeId: 'charge-1', chargeItemId: 'ci-1', dispenseQty: 2 }, { orderId: 'ord-2', chargeId: 'charge-2', chargeItemId: 'ci-2', dispenseQty: 2 }] }), (error) => {
      assert.equal(error.partialDispenses?.[0].dispenseNo, 'FY-1'); assert.equal(writes, 2); return true
    })
  })
} finally { globalThis.fetch = originalFetch }
if (failures.length) { console.error(`${failures.length} usability flow checks failed`); process.exitCode = 1 }
else console.log('usability flow checks passed')
