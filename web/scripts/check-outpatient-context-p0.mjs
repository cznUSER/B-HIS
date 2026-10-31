import assert from 'node:assert/strict'
import * as context from '../src/workbench/outpatientContext.js'
assert.equal(context.patientDisplayMeta({ sex: 'F', birthday: '2000-11-01' }, '2026-10-31'), '女 · 25岁')

assert.equal(context.patientDisplayMeta({ sex: 'U' }, '2026-10-31'), '未知')
assert.equal(context.patientDisplayMeta({ age: 0 }, '2026-10-31'), '0岁')
assert.equal(context.patientDisplayMeta({ birthday: '2000-02-30' }, '2026-10-31'), '')
assert.equal(context.patientDisplayMeta({ birthday: '2027-01-01' }, '2026-10-31'), '')
assert.equal(context.patientDisplayMeta({ birthday: '2000-01-01' }), '')
assert.equal(context.patientFareDisplay({}), '费别：未登记')
assert.deepEqual(context.patientRiskDisplay({ allergyHistory: ' ' }), { label: '过敏史：未记录', highRisk: false })
assert.deepEqual(context.patientRiskDisplay({ allergyHistory: '青霉素' }), { label: '过敏史：青霉素', highRisk: true })

const saved = new Map()
globalThis.sessionStorage = {
  getItem: (key) => saved.get(key) ?? null,
  setItem: (key, value) => saved.set(key, value),
  removeItem: (key) => saved.delete(key),
}
context.clearOutpatientContext()
context.setPatient({ patientId: 'p1', name: '甲', allergyHistory: '旧过敏史' })
context.setRegister({ registerId: 'r1', patientId: 'p1', doctorName: '旧医生' })
context.outpatientContext.orders = [{ orderId: 'old' }]
context.outpatientContext.diagnoses = [{ diagnoseId: 'old' }]
context.outpatientContext.charges = [{ chargeId: 'old' }]
context.outpatientContext.dispense = { dispenseId: 'old' }
const original = JSON.stringify(context.outpatientContext)
const originalSaved = JSON.stringify([...saved])
const messages = []
context.setOutpatientSwitchHandler(async (message) => { messages.push(message); return false })
assert.equal(await context.selectOutpatientContext({ patient: { patientId: 'p2', name: '乙', patientNo: 'P002' } }), false)
assert.equal(JSON.stringify(context.outpatientContext), original)
assert.equal(JSON.stringify([...saved]), originalSaved)
assert.equal(messages[0].title, '切换患者？')
assert.equal(messages[0].confirmLabel, '确认切换')
assert.equal(messages[0].cancelLabel, '取消')
assert.match(messages[0].body, /已保存到服务器的数据不会删除/)
context.setOutpatientSwitchHandler(async () => true)
assert.equal(await context.selectOutpatientContext({ patient: { patientId: 'p2', name: '乙' } }), true)
assert.equal(context.outpatientContext.patient.allergyHistory, undefined)
assert.equal(context.outpatientContext.register, null)
assert.deepEqual(context.outpatientContext.orders, [])
assert.deepEqual(context.outpatientContext.diagnoses, [])
assert.deepEqual(context.outpatientContext.charges, [])
assert.equal(context.outpatientContext.dispense, null)
assert.equal(context.outpatientContext.lastAction, '')
assert.equal(context.outpatientContext.history.some((entry) => entry.detail === '甲'), false)
context.setRegister({ registerId: 'r2', patientId: 'p2' })
assert.deepEqual(JSON.parse(saved.get(context.OUTPATIENT_CONTEXT_STORAGE_KEY)), { patientId: 'p2', registerId: 'r2' })
assert.deepEqual(context.readOutpatientSelection({ patientId: 'p9', registerId: 'r9', name: '不能持久化' }), { patientId: 'p9', registerId: 'r9' })
assert.deepEqual(context.readOutpatientSelection({ patientId: 'p9' }), { patientId: 'p9' })
assert.deepEqual(context.readOutpatientSelection({}), { patientId: 'p2', registerId: 'r2' })
assert.throws(() => context.readOutpatientSelection({ patientId: ['p1', 'p2'] }), /参数/)
context.outpatientContext.orders = [{ orderId: 'old-visit' }]
await context.selectOutpatientContext({ patient: { patientId: 'p2' }, register: { patientId: 'p2', registerId: 'r3' } })
assert.deepEqual(context.outpatientContext.orders, [])
assert.equal(context.outpatientContext.register.doctorName, undefined)
context.setCharges([{ chargeId: 'foreign', patientId: 'p1' }])
assert.equal(context.outpatientContext.patient.patientId, 'p2')
assert.deepEqual(context.outpatientContext.charges, [])
context.setOrders([{ orderId: 'stale', patientId: 'p2', registerId: 'r2' }])
assert.deepEqual(context.outpatientContext.orders, [])

let reads = 0
await assert.rejects(context.resolveOutpatientSelection({ patientId: 'p2' }, {
  authenticated: false,
  loadPatient: async () => { reads++; return { patientId: 'p2' } },
}), /登录/)
assert.equal(reads, 0)
await assert.rejects(context.resolveOutpatientSelection({ patientId: 'p2', registerId: 'wrong' }, {
  authenticated: true,
  loadPatient: async () => ({ patientId: 'p2' }),
  loadRegisters: async () => ({ items: [{ patientId: 'p1', registerId: 'wrong' }] }),
}), /就诊/)
const restored = await context.resolveOutpatientSelection({ registerId: 'late' }, {
  authenticated: true,
  loadRegisters: async ({ offset }) => offset === 0
    ? { items: [{ patientId: 'p1', registerId: 'early' }], hasMore: true, nextOffset: 1 }
    : { items: [{ patientId: 'p8', registerId: 'late' }], hasMore: false },
  loadPatient: async (patientId) => ({ patientId, name: '服务端最新值' }),
})
assert.equal(restored.patient.name, '服务端最新值')
assert.equal(restored.patient.patientId, restored.register.patientId)
context.clearOutpatientContext()
assert.equal(saved.has(context.OUTPATIENT_CONTEXT_STORAGE_KEY), false)
context.setOutpatientSwitchHandler(null)
saved.set(context.OUTPATIENT_CONTEXT_STORAGE_KEY, '{broken')
assert.deepEqual(context.readOutpatientSelection({}), {})
assert.equal(saved.has(context.OUTPATIENT_CONTEXT_STORAGE_KEY), false)
context.setPatient({ patientId: 'p1', name: '甲' })
let decide
context.setOutpatientSwitchHandler(() => new Promise((resolve) => { decide = resolve }))
const waiting = context.selectOutpatientContext({ patient: { patientId: 'p2', name: '乙' } })
assert.equal(context.outpatientSelection.pending, true)
context.setCharges([{ chargeId: 'late', patientId: 'p1' }])
assert.deepEqual(context.outpatientContext.charges, [])
context.clearOutpatientContext()
decide(true)
assert.equal(await waiting, false)
assert.equal(context.outpatientContext.patient, null)
assert.equal(context.outpatientSelection.pending, false)
context.setOutpatientSwitchHandler(null)
const realStorage = globalThis.sessionStorage
Object.defineProperty(globalThis, 'sessionStorage', { configurable: true, get() { throw new Error('blocked') } })
assert.deepEqual(context.readOutpatientSelection({}), {})
assert.equal(context.setPatient({ patientId: 'p3', name: '丙' }), true)
assert.equal(context.outpatientContext.patient.patientId, 'p3')
Object.defineProperty(globalThis, 'sessionStorage', { configurable: true, value: realStorage, writable: true })
context.clearOutpatientContext()
console.log('P0_A_CONTEXT_CHECK_OK')
