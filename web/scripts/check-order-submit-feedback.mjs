import assert from 'node:assert/strict'

import { createSubmitFeedback } from '../src/components/feedback/useSubmitFeedback.js'
import { saveOrder, commitOrder, abateOrder } from '../src/api/orderApi.js'
import { mockLogin } from '../src/api/mockAuth.js'
import { mockCreatePatient } from '../src/api/mockPatients.js'
import { createRegister } from '../src/api/registerApi.js'
import { resetMockRegisters } from '../src/api/mockRegisters.js'
import { resetMockOrders } from '../src/api/mockOrders.js'

const feedback = createSubmitFeedback()
assert.equal(feedback.state.status, 'idle')
feedback.beginAttempt('save')
assert.equal(feedback.state.status, 'submitting')
feedback.beginAttempt('save')
assert.equal(feedback.state.status, 'submitting')
feedback.succeed({ orderId: 'ord-1', orderNo: 'R202610310001' })
assert.equal(feedback.state.status, 'success')
assert.match(feedback.state.message, /R202610310001/)
assert.equal(feedback.state.orderNo, 'R202610310001')
assert.equal(feedback.state.orderId, 'ord-1')

for (const action of ['commit', 'abate']) {
  feedback.beginAttempt(action)
  feedback.succeed({ orderId: 'ord-x', orderNo: 'R-X' }, action + ' ok\uff0c\u6d41\u6c34\u53f7 R-X')
  assert.equal(feedback.state.status, 'success')
  assert.match(feedback.state.message, /R-X/)
}

// Register + diagnosis additive paths must remain intact
feedback.beginAttempt('register')
feedback.succeed({ registerId: 'reg-1', registerNo: 'R-1001' })
assert.equal(feedback.state.message, '\u6302\u53f7\u6210\u529f\uff0c\u6d41\u6c34\u53f7 R-1001')
assert.equal(feedback.state.registerNo, 'R-1001')
assert.equal(feedback.state.orderId, '')
assert.equal(feedback.state.diagnoseId, '')

feedback.beginAttempt('save')
feedback.succeed({ diagnoseId: 'diag-1' })
assert.match(feedback.state.message, /diag-1/)
assert.equal(feedback.state.diagnoseId, 'diag-1')
assert.equal(feedback.state.orderNo, '')

feedback.beginAttempt('commit')
feedback.fail('\u5f53\u524d\u72b6\u6001\u4e0d\u80fd\u63d0\u4ea4', 'ORDER_NOT_COMMITTABLE')
assert.equal(feedback.state.status, 'error')
assert.equal(feedback.state.code, 'ORDER_NOT_COMMITTABLE')

feedback.beginAttempt('save')
feedback.fail('\u670d\u52a1\u6682\u4e0d\u53ef\u7528', 'HTTP_ERROR')
assert.equal(feedback.state.status, 'error')
assert.equal(feedback.state.code, 'HTTP_ERROR')
feedback.beginAttempt('save')
assert.equal(feedback.state.status, 'submitting')
feedback.reset()
assert.equal(feedback.state.status, 'idle')

const originalMockMode = process.env.VITE_USE_MOCK
process.env.BHIS_WEB_TEST_MODE = '1'
process.env.VITE_USE_MOCK = 'true'
resetMockRegisters()
resetMockOrders()
const token = mockLogin({ username: 'system', password: '1' }).accessToken
const testPatient = mockCreatePatient(token, {
  name: 'Order Feedback',
  sex: 'M',
  birthday: '1990-01-01',
  idType: 'ID_CARD',
  fareIdentity: 'SELF',
  idNumber: '110101199001011288',
  phone: '13800000088',
  nationality: '\u4e2d\u56fd',
})
const register = await createRegister(token, {
  patientId: testPatient.patientId,
  departmentId: 'dept-1',
  doctorId: 'doc-1',
  visitDate: new Date().toISOString().slice(0, 10),
  registerFee: 0,
}, 'order-register-key')

const draft = {
  registerId: register.registerId,
  departmentId: 'dept-1',
  doctorId: 'doc-1',
  diagnoseName: '\u4e0a\u611f',
  patientId: testPatient.patientId,
  items: [{ itemNo: 'DRUG-A', amount: 1, usageText: '\u53e3\u670d', frequencyText: '\u4e00\u65e5\u4e09\u6b21', dosage: '1', dosageUnit: '\u76d2' }],
}

const saved = await saveOrder(token, draft, undefined, 'order-save-key')
assert.ok(saved.orderId)
assert.ok(saved.orderNo)
assert.equal(saved.status, 'SAVE')
const savedAgain = await saveOrder(token, draft, undefined, 'order-save-key')
assert.equal(savedAgain.orderId, saved.orderId)
assert.equal(savedAgain.orderNo, saved.orderNo)

const committed = await commitOrder(token, saved.orderId, 'order-commit-key')
assert.equal(committed.status, 'COMMIT')
const committedAgain = await commitOrder(token, saved.orderId, 'order-commit-key')
assert.equal(committedAgain.orderId, committed.orderId)
assert.equal(committedAgain.status, 'COMMIT')

const abated = await abateOrder(token, committed.orderId, 'order-abate-key')
assert.equal(abated.status, 'ABATE')
const abatedAgain = await abateOrder(token, committed.orderId, 'order-abate-key')
assert.equal(abatedAgain.orderId, abated.orderId)
assert.equal(abatedAgain.status, 'ABATE')

// Dual-key commit path: create with save-key, commit with commit-key (fresh order)
resetMockOrders()
const draft2 = {
  registerId: register.registerId,
  departmentId: 'dept-1',
  doctorId: 'doc-1',
  diagnoseName: '\u5934\u75db',
  patientId: testPatient.patientId,
  items: [{ itemNo: 'DRUG-A', amount: 2, usageText: '\u53e3\u670d', frequencyText: '\u4e00\u65e5\u4e24\u6b21', dosage: '1', dosageUnit: '\u76d2' }],
}
const saved2 = await saveOrder(token, draft2, undefined, 'dual-save-key')
const committed2 = await commitOrder(token, saved2.orderId, 'dual-commit-key')
assert.equal(committed2.status, 'COMMIT')
const saved2Again = await saveOrder(token, draft2, undefined, 'dual-save-key')
assert.equal(saved2Again.orderId, saved2.orderId)
const committed2Again = await commitOrder(token, saved2.orderId, 'dual-commit-key')
assert.equal(committed2Again.orderId, committed2.orderId)

process.env.VITE_USE_MOCK = 'false'
const originalFetch = globalThis.fetch
let sentKey = ''
let postCount = 0
globalThis.fetch = async (_url, init) => {
  sentKey = init.headers['Idempotency-Key']
  postCount += 1
  return new Response(JSON.stringify({
    orderId: 'ord-2',
    orderNo: 'R202610310002',
    registerId: 'reg-2',
    encounterId: 'reg-2',
    patientId: 'patient-2',
    departmentId: 'dept-1',
    doctorId: 'doc-1',
    diagnoseName: '',
    recipeSort: '',
    status: 'SAVE',
    totalFee: 10,
    charged: false,
    items: [{ orderItemId: 'oi-1', itemNo: 'DRUG-A', itemName: 'A', itemType: 'DRUG', amount: 1, units: '\u76d2', price: 10, totalFee: 10 }],
    submittedAt: null,
    abatedAt: null,
    abatedBy: null,
    operatedAt: new Date().toISOString(),
    operatorId: 'u1',
  }), { status: 201 })
}
const created = await saveOrder(token, {
  registerId: 'reg-2',
  departmentId: 'dept-1',
  doctorId: 'doc-1',
  diagnoseName: '',
  patientId: 'patient-2',
  items: [{ itemNo: 'DRUG-A', amount: 1 }],
}, [{ itemNo: 'DRUG-A', itemType: 'DRUG', itemName: 'A', units: '\u76d2', price: 10, status: 'ACTIVE' }], 'attempt-stable-order-key')
assert.equal(sentKey, 'attempt-stable-order-key')
assert.equal(created.orderNo, 'R202610310002')

globalThis.fetch = async () => new Response(JSON.stringify({ error: { code: 'ORDER_NOT_COMMITTABLE', message: '\u5f53\u524d\u72b6\u6001\u4e0d\u80fd\u63d0\u4ea4' } }), { status: 409 })
await assert.rejects(
  commitOrder(token, 'ord-2', 'attempt-illegal-key'),
  (error) => error.code === 'ORDER_NOT_COMMITTABLE' && error.status === 409,
)

globalThis.fetch = async () => new Response(JSON.stringify({ error: { code: 'HTTP_ERROR', message: 'server unavailable' } }), { status: 503 })
await assert.rejects(
  saveOrder(token, {
    registerId: 'reg-2',
    departmentId: 'dept-1',
    doctorId: 'doc-1',
    diagnoseName: '',
    patientId: 'patient-2',
    items: [{ itemNo: 'DRUG-A', amount: 1 }],
  }, [{ itemNo: 'DRUG-A', itemType: 'DRUG', itemName: 'A', units: '\u76d2', price: 10, status: 'ACTIVE' }], 'attempt-server-error-key'),
  (error) => error.status === 503,
)

globalThis.fetch = async () => { throw new Error('network offline') }
await assert.rejects(
  saveOrder(token, {
    registerId: 'reg-2',
    departmentId: 'dept-1',
    doctorId: 'doc-1',
    diagnoseName: '',
    patientId: 'patient-2',
    items: [{ itemNo: 'DRUG-A', amount: 1 }],
  }, [{ itemNo: 'DRUG-A', itemType: 'DRUG', itemName: 'A', units: '\u76d2', price: 10, status: 'ACTIVE' }], 'attempt-network-error-key'),
  (error) => error.status === 0 && error.code === 'HTTP_ERROR',
)

assert.equal(JSON.stringify(feedback.state).includes(token), false)
assert.ok(postCount >= 1)

globalThis.fetch = originalFetch
delete process.env.BHIS_WEB_TEST_MODE
if (originalMockMode === undefined) delete process.env.VITE_USE_MOCK
else process.env.VITE_USE_MOCK = originalMockMode

console.log('P0_D3_ORDER_FEEDBACK_OK')
