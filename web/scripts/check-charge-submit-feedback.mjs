import assert from 'node:assert/strict'

import { createSubmitFeedback } from '../src/components/feedback/useSubmitFeedback.js'
import { payCharge, refundCharge } from '../src/api/chargeApi.js'
import { mockLogin, resetMockAuth } from '../src/api/mockAuth.js'
import {
  mockGetCharge,
  mockLoadPending,
  mockLookup,
  mockPay,
  mockRefund,
  resetMockCharges,
} from '../src/api/mockCharges.js'

const feedback = createSubmitFeedback()
assert.equal(feedback.state.status, 'idle')
feedback.beginAttempt('pay')
assert.equal(feedback.state.status, 'submitting')
feedback.beginAttempt('pay')
assert.equal(feedback.state.status, 'submitting')
feedback.succeed({ chargeId: 'chg-1', squareNo: 'SF202610310001' })
assert.equal(feedback.state.status, 'success')
assert.match(feedback.state.message, /SF202610310001/)
assert.equal(feedback.state.squareNo, 'SF202610310001')
assert.equal(feedback.state.chargeId, 'chg-1')

feedback.beginAttempt('refund')
feedback.succeed({ chargeId: 'chg-2', squareNo: 'SF-2' }, '已全额退费。收费单号 SF-2')
assert.equal(feedback.state.status, 'success')
assert.match(feedback.state.message, /SF-2/)

// Additive paths must remain intact
feedback.beginAttempt('register')
feedback.succeed({ registerId: 'reg-1', registerNo: 'R-1001' })
assert.equal(feedback.state.message, '挂号成功，流水号 R-1001')
assert.equal(feedback.state.registerNo, 'R-1001')
assert.equal(feedback.state.chargeId, '')
assert.equal(feedback.state.orderId, '')

feedback.beginAttempt('save')
feedback.succeed({ diagnoseId: 'diag-1' })
assert.match(feedback.state.message, /diag-1/)
assert.equal(feedback.state.diagnoseId, 'diag-1')
assert.equal(feedback.state.squareNo, '')

feedback.beginAttempt('commit')
feedback.succeed({ orderId: 'ord-1', orderNo: 'R-ORD' })
assert.match(feedback.state.message, /R-ORD/)
assert.equal(feedback.state.orderNo, 'R-ORD')
assert.equal(feedback.state.chargeId, '')

feedback.beginAttempt('pay')
feedback.fail('该处方已收费', 'ALREADY_CHARGED')
assert.equal(feedback.state.status, 'error')
assert.equal(feedback.state.code, 'ALREADY_CHARGED')

feedback.beginAttempt('pay')
feedback.fail('服务暂不可用', 'HTTP_ERROR')
assert.equal(feedback.state.status, 'error')
assert.equal(feedback.state.code, 'HTTP_ERROR')
feedback.beginAttempt('pay')
assert.equal(feedback.state.status, 'submitting')
feedback.reset()
assert.equal(feedback.state.status, 'idle')

const originalMockMode = process.env.VITE_USE_MOCK
process.env.BHIS_WEB_TEST_MODE = '1'
process.env.VITE_USE_MOCK = 'true'
resetMockAuth()
resetMockCharges()
const token = mockLogin({ username: 'system', password: '1' }).accessToken

function subject(name) {
  const found = mockLookup(token, name).items.find((item) => item.patientName === name)
  assert.ok(found, `missing subject ${name}`)
  return found
}

const ready = mockLoadPending(token, subject('赵待收').patientId)
assert.equal(ready.scene, 'PENDING')
assert.equal(ready.preview.amountDueCents, 4300)

const paid = await payCharge(token, {
  orderId: ready.preview.orderId,
  payType: 'CASH',
  amountCents: 4300,
  createIdempotencyKey: 'charge-create-key',
  idempotencyKey: 'charge-pay-key',
})
assert.ok(paid.charge.chargeId)
assert.ok(paid.charge.squareNo)
assert.equal(paid.charge.squareFeeCents, 4300)

const paidAgain = await payCharge(token, {
  orderId: ready.preview.orderId,
  payType: 'CASH',
  amountCents: 4300,
  createIdempotencyKey: 'charge-create-key',
  idempotencyKey: 'charge-pay-key',
})
assert.equal(paidAgain.charge.chargeId, paid.charge.chargeId)
assert.equal(paidAgain.charge.squareNo, paid.charge.squareNo)

const charge = mockGetCharge(token, paid.charge.chargeId)
const amox = charge.items.find((item) => item.itemNo === 'AMX')
assert.ok(amox)

const refundBody = {
  payType: 'CASH',
  reason: '患者要求',
  items: [{ chargeItemId: amox.chargeItemId, refundQty: 1 }],
  idempotencyKey: 'charge-refund-key',
}
const refunded = await refundCharge(token, paid.charge.chargeId, refundBody)
assert.equal(refunded.fullRefund, false)
assert.ok(refunded.abatedSquare)
assert.ok(refunded.newSquare)
const refundedAgain = await refundCharge(token, paid.charge.chargeId, refundBody)
assert.equal(refundedAgain.abatedSquare.chargeId, refunded.abatedSquare.chargeId)
assert.equal(refundedAgain.newSquare.chargeId, refunded.newSquare.chargeId)
assert.equal(refundedAgain.fullRefund, refunded.fullRefund)

// Direct mockPay same-key on a fresh fixture after reset for isolated pay idempotency
resetMockCharges()
const ready2 = mockLoadPending(token, subject('赵待收').patientId)
const direct = mockPay(token, { orderId: ready2.preview.orderId, payType: 'CASH', amountCents: 4300, idempotencyKey: 'direct-pay-key' })
const directAgain = mockPay(token, { orderId: ready2.preview.orderId, payType: 'CASH', amountCents: 4300, idempotencyKey: 'direct-pay-key' })
assert.equal(directAgain.charge.chargeId, direct.charge.chargeId)

process.env.VITE_USE_MOCK = 'false'
const originalFetch = globalThis.fetch
let sentKeys = []
globalThis.fetch = async (url, init) => {
  sentKeys.push(init.headers['Idempotency-Key'])
  const path = String(url)
  if (path.includes('/charges') && init.method === 'POST' && !path.includes('/pay') && !path.includes('/refund')) {
    return new Response(JSON.stringify({
      chargeId: 'chg-live-1',
      squareNo: 'SF-LIVE-1',
      orderId: 'ord-live',
      registerId: 'reg-live',
      encounterId: 'reg-live',
      patientId: 'patient-live',
      status: 'NORMAL',
      totalFee: 43,
      squareFee: 43,
      favourableFee: 0,
      items: [],
    }), { status: 201 })
  }
  if (path.includes('/pay')) {
    return new Response(JSON.stringify({
      paymentId: 'pay-1',
      paymentNo: 'PM1',
      chargeId: 'chg-live-1',
      direction: 'CHARGE',
      payType: 'CASH',
      amount: 43,
      status: 'SUCCEEDED',
      paidAt: new Date().toISOString(),
    }), { status: 200 })
  }
  if (path.includes('/charges/chg-live-1') && (!init || init.method === 'GET' || !init.method)) {
    return new Response(JSON.stringify({
      chargeId: 'chg-live-1',
      squareNo: 'SF-LIVE-1',
      orderId: 'ord-live',
      registerId: 'reg-live',
      encounterId: 'reg-live',
      patientId: 'patient-live',
      status: 'NORMAL',
      totalFee: 43,
      squareFee: 43,
      favourableFee: 0,
      paidAt: new Date().toISOString(),
      items: [],
    }), { status: 200 })
  }
  return new Response(JSON.stringify({ error: { code: 'HTTP_ERROR', message: 'unexpected ' + path } }), { status: 500 })
}

const live = await payCharge(token, {
  orderId: 'ord-live',
  registerId: 'reg-live',
  payType: 'CASH',
  amount: 43,
  createIdempotencyKey: 'live-create-key',
  idempotencyKey: 'live-pay-key',
})
assert.ok(sentKeys.includes('live-create-key'))
assert.ok(sentKeys.includes('live-pay-key'))
assert.ok(live.charge.chargeId)

globalThis.fetch = async () => new Response(JSON.stringify({ error: { code: 'ALREADY_CHARGED', message: '该处方已收费' } }), { status: 409 })
await assert.rejects(
  payCharge(token, {
    orderId: 'ord-x',
    registerId: 'reg-x',
    payType: 'CASH',
    amount: 10,
    createIdempotencyKey: 'err-create',
    idempotencyKey: 'err-pay',
  }),
  (error) => error.code === 'ALREADY_CHARGED' && error.status === 409,
)

globalThis.fetch = async () => new Response(JSON.stringify({ error: { code: 'HTTP_ERROR', message: 'server unavailable' } }), { status: 503 })
await assert.rejects(
  payCharge(token, {
    orderId: 'ord-x',
    registerId: 'reg-x',
    payType: 'CASH',
    amount: 10,
    createIdempotencyKey: 'err-create-503',
    idempotencyKey: 'err-pay-503',
  }),
  (error) => error.status === 503,
)

globalThis.fetch = async () => { throw new Error('network offline') }
await assert.rejects(
  payCharge(token, {
    orderId: 'ord-x',
    registerId: 'reg-x',
    payType: 'CASH',
    amount: 10,
    createIdempotencyKey: 'err-create-net',
    idempotencyKey: 'err-pay-net',
  }),
  (error) => error.status === 0 && error.code === 'HTTP_ERROR',
)

assert.equal(JSON.stringify(feedback.state).includes(token), false)

globalThis.fetch = originalFetch
delete process.env.BHIS_WEB_TEST_MODE
if (originalMockMode === undefined) delete process.env.VITE_USE_MOCK
else process.env.VITE_USE_MOCK = originalMockMode

console.log('P0_D4_CHARGE_FEEDBACK_OK')