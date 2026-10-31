// P0-E-INTEGRATION-01 light write-chain smoke (mock): register -> diagnose -> order -> charge/pay -> dispense.
// Current D1-D5 same-key semantics. Evidence harness only.
import assert from 'node:assert/strict'
import { mockLogin, resetMockAuth } from '../src/api/mockAuth.js'
import { resetMockPatients, mockCreatePatient } from '../src/api/mockPatients.js'
import { resetMockRegisters } from '../src/api/mockRegisters.js'
import { resetMockDiagnoses } from '../src/api/mockDiagnoses.js'
import { resetMockOrders } from '../src/api/mockOrders.js'
import { resetMockCharges } from '../src/api/mockCharges.js'
import { resetMockDispenses, mockSearchPending } from '../src/api/mockDispenses.js'
import { resetMasterStore } from '../src/master/store.js'
import { createRegister, checkInRegister } from '../src/api/registerApi.js'
import { saveDiagnose, commitDiagnose } from '../src/api/diagnosisApi.js'
import { saveOrder, commitOrder } from '../src/api/orderApi.js'
import { payCharge } from '../src/api/chargeApi.js'
import { confirmDispense } from '../src/api/dispenseApi.js'
import { todayInShanghai } from '../src/api/patientContract.js'

process.env.BHIS_WEB_TEST_MODE = '1'
process.env.VITE_USE_MOCK = 'true'

resetMasterStore()
resetMockAuth()
resetMockPatients()
resetMockRegisters()
resetMockDiagnoses()
resetMockOrders()
resetMockCharges()
resetMockDispenses()

const token = mockLogin({ username: 'system', password: '1' }).accessToken
const patient = mockCreatePatient(token, {
  name: 'P0EWriteChain', sex: 'F', birthday: '1990-01-01', fareIdentity: 'SELF',
  idType: 'ID_CARD', idNumber: '110101199001011288', phone: '13800000088', nationality: 'CN',
})

const registerBody = {
  patientId: patient.patientId,
  departmentId: 'dept-1',
  doctorId: 'doc-1',
  visitDate: todayInShanghai(),
  registerFee: 0,
}
const register = await createRegister(token, registerBody, 'p0e-register-key')
const registerAgain = await createRegister(token, registerBody, 'p0e-register-key')
assert.equal(registerAgain.registerId, register.registerId)
assert.ok(register.registerNo)
await checkInRegister(token, register.registerId)

const diagnosePayload = {
  content: 'chief: fever\\nassessment: URI',
  structured: {
    chiefComplaint: 'fever',
    presentHistory: '',
    pastHistory: '',
    examination: '',
    assessment: 'URI',
    plan: '',
  },
  templateCode: 'GENERAL',
}
const diagnose = await saveDiagnose(token, register.registerId, diagnosePayload, 'p0e-diagnose-key')
const diagnoseAgain = await saveDiagnose(token, register.registerId, diagnosePayload, 'p0e-diagnose-key')
assert.equal(diagnoseAgain.diagnoseId, diagnose.diagnoseId)
const diagnoseCommit = await commitDiagnose(token, diagnose.diagnoseId, 'p0e-diagnose-commit-key')
assert.equal(diagnoseCommit.status, 'COMMIT')

const orderBody = {
  registerId: register.registerId,
  items: [{ itemNo: 'DRUG-A', amount: 1 }, { itemNo: 'TREAT-1', amount: 1 }],
}
const order = await saveOrder(token, orderBody, undefined, 'p0e-order-key')
const orderAgain = await saveOrder(token, orderBody, undefined, 'p0e-order-key')
assert.equal(orderAgain.orderId, order.orderId)
assert.ok(order.orderNo)
const orderCommit = await commitOrder(token, order.orderId, 'p0e-order-commit-key')
assert.equal(orderCommit.orderId, order.orderId)

const paid = await payCharge(token, {
  orderId: order.orderId,
  payType: 'CASH',
  amountCents: orderCommit.amountDueCents || order.amountDueCents || 2500,
  createIdempotencyKey: 'p0e-create-key',
  idempotencyKey: 'p0e-pay-key',
})
const paidAgain = await payCharge(token, {
  orderId: order.orderId,
  payType: 'CASH',
  amountCents: orderCommit.amountDueCents || order.amountDueCents || 2500,
  createIdempotencyKey: 'p0e-create-key',
  idempotencyKey: 'p0e-pay-key',
})
assert.equal(paidAgain.charge.chargeId, paid.charge.chargeId)
assert.ok(paid.charge.squareNo)

const pending = mockSearchPending(token, patient.name)
assert.ok(pending.items.length > 0, 'paid drug line should be pending dispense')
const line = pending.items[0]
const dispenseBody = {
  items: [{ chargeItemId: line.chargeItemId, dispenseQty: line.availableQty || 1, selected: true }],
  idempotencyKey: 'p0e-dispense-key',
}
const dispensed = await confirmDispense(token, dispenseBody)
const dispensedAgain = await confirmDispense(token, dispenseBody)
assert.equal(dispensedAgain.dispenseId, dispensed.dispenseId)
assert.ok(dispensed.dispenseNo)

assert.equal(register.patientId, patient.patientId)
assert.equal(order.registerId, register.registerId)
assert.equal(paid.charge.registerId, register.registerId)
assert.equal(paid.charge.patientId, patient.patientId)

console.log('P0_E_WRITE_CHAIN_SMOKE_OK ' + JSON.stringify({
  registerNo: register.registerNo,
  diagnoseId: diagnose.diagnoseId,
  orderNo: order.orderNo,
  chargeId: paid.charge.chargeId,
  squareNo: paid.charge.squareNo,
  dispenseNo: dispensed.dispenseNo,
  sameKeyNoDual: true,
}))
