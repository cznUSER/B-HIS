import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import {
  buildStatusChainSummary,
  classifyDomainError,
  contextFromStatusChainPayloads,
  STATUS_CHAIN_STATE_LABELS,
} from '../src/workbench/statusChainExplain.js'
import {
  outpatientContext,
  clearOutpatientContext,
  setPatient,
  setRegister,
  setDiagnosis,
  setOrders,
  setCharges,
  setDispense,
  deriveWorkflowStatus,
} from '../src/workbench/outpatientContext.js'

const patient = { patientId: 'p-status', name: '状态验收' }
const register = { registerId: 'r-status', patientId: patient.patientId, status: 'DIAGNOSED' }
const base = { patient, register, diagnosis: null, diagnoses: [], orders: [], charges: [], dispense: null }
const order = { orderId: 'o-status', registerId: register.registerId, patientId: patient.patientId, status: 'COMMIT' }
const charge = {
  chargeId: 'c-status',
  orderId: order.orderId,
  registerId: register.registerId,
  patientId: patient.patientId,
  status: 'NORMAL',
  paid: true,
}

const summary = buildStatusChainSummary(base)
assert.equal(summary.kind, 'normal')
assert.equal(summary.key, 'WAIT_ORDER')
assert.equal(summary.next, deriveWorkflowStatus(base).next)
assert.match(summary.explain, /开单/)
assert.equal(summary.registerId, register.registerId)
assert.equal(buildStatusChainSummary(null).kind, 'empty')
assert.equal(buildStatusChainSummary({}).label, '待选择患者')

const paidContext = { ...base, orders: [order], charges: [charge] }
const partial = buildStatusChainSummary(paidContext)
assert.equal(partial.kind, 'partial')
assert.ok(partial.partialReasons.length)
assert.match(partial.explain, /收费.*发药/)

const slip = {
  dispenseId: 'd-status',
  registerId: register.registerId,
  patientId: patient.patientId,
  lineDispenseStatus: 'PARTIAL',
  items: [{ lineDispenseStatus: 'PARTIAL' }],
}
const partiallyDispensed = buildStatusChainSummary({ ...paidContext, dispense: slip })
assert.equal(partiallyDispensed.kind, 'partial')
assert.equal(partiallyDispensed.steps.find((step) => step.key === 'dispense').complete, false)
assert.match(partiallyDispensed.explain, /部分发药/)
assert.equal(
  buildStatusChainSummary({
    ...paidContext,
    dispense: { ...slip, lineDispenseStatus: 'FULL', items: [{ lineDispenseStatus: 'FULL' }] },
  }).steps.at(-1).state,
  'done',
)
assert.equal(
  buildStatusChainSummary({
    ...paidContext,
    dispense: { ...slip, lineDispenseStatus: 'NONE', items: [{ lineDispenseStatus: 'NONE' }] },
  }).steps.at(-1).complete,
  false,
)

const blocked = buildStatusChainSummary({ ...base, orders: [order] })
assert.equal(blocked.kind, 'blocked')
assert.equal(blocked.steps.find((step) => step.key === 'charge').state, 'blocked')
assert.equal(blocked.steps.find((step) => step.key === 'charge').current, true)
assert.match(blocked.blockReason, /未收费|未支付/)

const back = buildStatusChainSummary({ ...paidContext, register: { ...register, status: 'BACK' } })
assert.equal(back.kind, 'void')
assert.equal(back.steps.find((step) => step.key === 'register').state, 'void')
assert.match(back.explain, /退号|作废/)
assert.equal(back.steps.find((step) => step.key === 'dispense').state, 'skipped')

const abated = buildStatusChainSummary({ ...base, orders: [{ ...order, status: 'ABATE' }] })
assert.equal(abated.kind, 'void')
assert.equal(abated.steps.find((step) => step.key === 'order').state, 'void')
assert.equal(buildStatusChainSummary({ ...base, register: { ...register, status: 'PAUSED' } }).kind, 'blocked')

for (const error of [{ status: 403 }, { response: { status: 403 } }, { code: 'FORBIDDEN' }, { error: { code: 'FORBIDDEN' } }]) {
  const access = classifyDomainError(error)
  assert.equal(access.forbidden, true)
  assert.match(access.message, /无权限/)
  assert.doesNotMatch(access.message, /无数据|暂无/)
}
assert.equal(classifyDomainError({ status: 500 }).forbidden, false)

const denied = buildStatusChainSummary(paidContext, {
  domainAccess: { charges: { ok: false, ...classifyDomainError({ status: 403 }) } },
})
assert.equal(denied.kind, 'blocked')
assert.equal(denied.domainAccess.charges.forbidden, true)
assert.notEqual(denied.domainAccess.charges.empty, true)
assert.equal(denied.steps.find((step) => step.key === 'charge').complete, false)
assert.match(denied.explain, /无权限/)

const payloads = {
  patient,
  registers: { items: [register] },
  orders: { items: [order, { ...order, orderId: 'foreign', registerId: 'another' }] },
  charges: { items: [charge] },
  diagnoses: { items: [] },
  dispenses: { items: [slip, { ...slip, dispenseId: 'foreign', registerId: 'another' }] },
}
const originalPayloads = JSON.stringify(payloads)
const refreshed = contextFromStatusChainPayloads(
  { patientId: patient.patientId, registerId: register.registerId },
  payloads,
)
assert.equal(JSON.stringify(payloads), originalPayloads)
assert.equal(refreshed.orders.length, 1)
assert.equal(refreshed.dispense.dispenseId, slip.dispenseId)

clearOutpatientContext()
setPatient(refreshed.patient)
setRegister(refreshed.register)
setOrders(refreshed.orders)
setCharges(refreshed.charges)
setDiagnosis(refreshed.diagnosis, refreshed.diagnoses)
setDispense(refreshed.dispense)
assert.deepEqual(buildStatusChainSummary(outpatientContext), buildStatusChainSummary(refreshed))

const emptyRefresh = contextFromStatusChainPayloads(
  { patientId: patient.patientId, registerId: register.registerId },
  { ...payloads, dispenses: { items: [] }, diagnoses: { items: [] } },
)
assert.equal(emptyRefresh.dispense, null)
assert.equal(emptyRefresh.diagnosis, null)
assert.equal(buildStatusChainSummary(emptyRefresh).key, 'WAIT_DISPENSE')
assert.throws(
  () => contextFromStatusChainPayloads({ patientId: patient.patientId, registerId: 'missing' }, payloads),
  /挂号/,
)
assert.throws(
  () => contextFromStatusChainPayloads({ patientId: 'foreign', registerId: register.registerId }, payloads),
  /患者|挂号/,
)

const refreshedAt = '2026-10-31T05:00:00.000Z'
assert.equal(buildStatusChainSummary(refreshed, { refreshedAt }).refreshedAt, refreshedAt)

for (const result of [summary, partial, partiallyDispensed, blocked, back, denied, abated]) {
  assert.ok(result.explain)
  assert.doesNotThrow(() => JSON.stringify(result))
  for (const step of result.steps) {
    assert.ok(STATUS_CHAIN_STATE_LABELS[step.state], `missing label for ${step.state}`)
    assert.ok(step.label && step.completeLabel && step.reason)
    assert.equal(typeof step.complete, 'boolean')
    assert.equal(typeof step.current, 'boolean')
  }
}

const component = await readFile(new URL('../src/components/outpatient/OutpatientStatusChain.vue', import.meta.url), 'utf8')
assert.ok(component.includes('aria-label="流程状态"'))
assert.ok(component.includes('data-testid="outpatient-status-chain"'))
assert.ok(component.includes('STATUS_CHAIN_STATE_LABELS[step.state]'))
assert.ok(component.includes('<style scoped>'))
assert.ok(!component.includes('workbench/summary'))
assert.ok(component.includes('classifyDomainError') || component.includes('domainAccess') || component.includes('accessBanners'))

const frame = await readFile(new URL('../src/components/OutpatientWorkbenchFrame.vue', import.meta.url), 'utf8')
assert.ok(frame.includes("import OutpatientStatusChain from './outpatient/OutpatientStatusChain.vue'"))
assert.ok(frame.includes('<OutpatientStatusChain'))
assert.ok(!frame.includes('v-for="step in workflowProgress"'))

clearOutpatientContext()
console.log('P0_C_STATUS_CHECK_OK')
