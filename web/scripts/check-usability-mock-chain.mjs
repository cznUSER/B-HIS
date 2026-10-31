// 2026-10-14 P1-H05-WEB-USABILITY: a newly created patient's actual Mock API chain.
// Catches disconnected fixtures, lost IDs, repeated stock writes, and refund/return drift.
import assert from 'node:assert/strict'
import { mockLogin, resetMockAuth } from '../src/api/mockAuth.js'
import { mockCreatePatient, resetMockPatients } from '../src/api/mockPatients.js'
import { mockCreateRegister, mockCheckInRegister, resetMockRegisters } from '../src/api/mockRegisters.js'
import { mockCreateOrder, mockCommitOrder, mockListOrders, mockAbateOrder, resetMockOrders } from '../src/api/mockOrders.js'
import { mockLookup, mockLoadPending, mockPay, mockGetCharge, mockRefund, resetMockCharges } from '../src/api/mockCharges.js'
import { mockSearchPending, mockConfirmDispense, mockGetDispense, mockReturnDispense, mockListTodayPending, resetMockDispenses } from '../src/api/mockDispenses.js'
import { listInventoryItems, listLedger, resetMasterStore } from '../src/master/store.js'
import { todayInShanghai } from '../src/api/patientContract.js'

process.env.VITE_USE_MOCK = 'true'

function expectError(run, code) {
  assert.throws(run, (error) => error.code === code, `expected ${code}`)
}

function runChain(storageMode) {
  resetMasterStore()
  resetMockAuth()
  resetMockPatients()
  resetMockRegisters()
  resetMockOrders()
  resetMockCharges()
  resetMockDispenses()
  const token = mockLogin({ username: 'system', password: '1' }).accessToken
  const patient = mockCreatePatient(token, {
    name: `链路患者${storageMode}`, sex: 'F', birthday: '1990-01-01', fareIdentity: 'SELF',
    idType: 'ID_CARD', idNumber: '110101199001011234', phone: '13800000000', nationality: '中国',
  })
  const register = mockCreateRegister(token, { patientId: patient.patientId, departmentId: 'dept-1', doctorId: 'doc-1', visitDate: todayInShanghai() })
  mockCheckInRegister(token, register.registerId)
  const order = mockCreateOrder(token, { registerId: register.registerId, items: [{ itemNo: 'DRUG-A', amount: 2 }, { itemNo: 'TREAT-1', amount: 1 }] })
  const committed = mockCommitOrder(token, order.orderId)

  const subjects = mockLookup(token, patient.name).items
  assert.equal(subjects.length, 1, 'new patient must be discoverable from actual stored order')
  assert.equal(subjects[0].patientId, patient.patientId)
  const pending = mockLoadPending(token, patient.patientId)
  assert.equal(pending.scene, 'PENDING', 'checked-in register must remain chargeable')
  assert.equal(pending.register.registerId, register.registerId)
  assert.equal(pending.order.orderId, committed.orderId)
  assert.equal(pending.preview.amountDueCents, 3500)

  const sku = listInventoryItems({ itemNo: 'DRUG-A' }).items[0]
  const beforeQty = sku.qtyOnHand
  const beforeLedger = listLedger(sku.skuId).items.length
  const paid = mockPay(token, { orderId: order.orderId, payType: 'CASH', amountCents: 3500 })
  assert.equal(paid.charge.orderId, order.orderId)
  assert.equal(paid.charge.registerId, register.registerId)
  assert.equal(paid.charge.patientId, patient.patientId)
  assert.equal(listInventoryItems({ itemNo: 'DRUG-A' }).items[0].qtyOnHand, beforeQty, 'cash payment must not deduct stock')
  assert.equal(mockListOrders(token, { registerId: register.registerId }).items[0].charged, true, 'payment must mark actual order charged')
  expectError(() => mockAbateOrder(token, order.orderId), 'ORDER_CHARGED_NOT_ABATABLE')
  expectError(() => mockPay(token, { orderId: order.orderId, payType: 'CASH', amountCents: 3500 }), 'ALREADY_CHARGED')

  const charge = mockGetCharge(token, paid.charge.chargeId)
  const drugCharge = charge.items.find((item) => item.itemNo === 'DRUG-A')
  assert.equal(drugCharge.orderItemId, committed.items[0].orderItemId, 'charge must preserve order line identity')
  const lines = mockSearchPending(token, paid.charge.chargeId).items
  assert.equal(lines.length, 1, 'only paid drug lines can be dispensed')
  assert.equal(lines[0].chargeItemId, drugCharge.chargeItemId)
  assert.equal(lines[0].orderItemId, committed.items[0].orderItemId)
  assert.equal(lines[0].registerId, register.registerId)
  assert.equal(lines[0].patientId, patient.patientId)
  assert.equal(lines[0].stockQty, beforeQty)
  assert.equal(mockListTodayPending(token).items.filter((item) => item.chargeId === charge.chargeId).length, 1)
  const input = { idempotencyKey: `dispense-${storageMode}`, items: [{ chargeItemId: drugCharge.chargeItemId, dispenseQty: 2 }] }
  const issued = mockConfirmDispense(token, input)
  assert.equal(issued.chargeId, charge.chargeId)
  assert.equal(issued.orderId, order.orderId)
  assert.equal(issued.lines[0].stockQty, beforeQty - 2)
  assert.equal(listInventoryItems({ itemNo: 'DRUG-A' }).items[0].qtyOnHand, beforeQty - 2)
  assert.equal(listLedger(sku.skuId).items.length, beforeLedger + 1)
  const issuedAgain = mockConfirmDispense(token, input)
  assert.equal(issuedAgain.dispenseId, issued.dispenseId, 'same-key same-fingerprint dispense must replay prior result')
  assert.equal(issuedAgain.dispenseNo, issued.dispenseNo)
  expectError(() => mockConfirmDispense(token, { ...input, items: [{ chargeItemId: drugCharge.chargeItemId, dispenseQty: 1 }] }), 'IDEMPOTENCY_KEY_CONFLICT')
  assert.equal(listInventoryItems({ itemNo: 'DRUG-A' }).items[0].qtyOnHand, beforeQty - 2, 'dispense replay must not deduct twice')
  assert.equal(mockSearchPending(token, charge.chargeId).empty, 'done', 'sync must not reset dispensed counters')
  assert.equal(mockLoadPending(token, patient.patientId).scene, 'DISPENSED')
  expectError(() => mockRefund(token, charge.chargeId, { payType: 'CASH', items: [{ chargeItemId: drugCharge.chargeItemId, refundQty: 2 }] }), 'REFUND_REQUIRES_RETURN')

  const detail = mockGetDispense(token, issued.dispenseId)
  const returnInput = { dispenseId: detail.dispenseId, idempotencyKey: `return-${storageMode}`, reason: '患者要求', items: [{ dispenseItemId: detail.items[0].dispenseItemId, returnQty: 2 }] }
  const returned = mockReturnDispense(token, returnInput)
  assert.equal(returned.chargeId, charge.chargeId)
  assert.equal(returned.items[0].stockQty, beforeQty)
  assert.equal(listInventoryItems({ itemNo: 'DRUG-A' }).items[0].qtyOnHand, beforeQty)
  assert.equal(listLedger(sku.skuId).items.length, beforeLedger + 2)
  const returnedAgain = mockReturnDispense(token, returnInput)
  assert.equal(returnedAgain.returnId, returned.returnId, 'same-key same-fingerprint return must replay prior result')
  assert.equal(returnedAgain.dispenseId, returned.dispenseId)
  assert.equal(returnedAgain.returnedQty, returned.returnedQty)
  expectError(() => mockReturnDispense(token, { ...returnInput, reason: '不同原因' }), 'IDEMPOTENCY_KEY_CONFLICT')
  assert.equal(listInventoryItems({ itemNo: 'DRUG-A' }).items[0].qtyOnHand, beforeQty, 'return replay must not restore twice')
  assert.equal(mockGetCharge(token, charge.chargeId).items.find((item) => item.itemNo === 'DRUG-A').dispenseBlocked, false)
  assert.equal(mockSearchPending(token, charge.chargeId).items[0].availableQty, 2)

  const refund = mockRefund(token, charge.chargeId, { payType: 'CASH', items: charge.items.map((item) => ({ chargeItemId: item.chargeItemId, refundQty: item.qty })) })
  assert.equal(refund.fullRefund, true)
  assert.equal(refund.refundPayment.amountCents, 3500)
  assert.equal(refund.abatedSquare.chargeId, charge.chargeId)
  assert.equal(mockGetCharge(token, charge.chargeId).status, 'ABATE')
  assert.equal(mockListTodayPending(token).items.filter((item) => item.chargeId === charge.chargeId).length, 0, 'refunded charge must leave dispensing queue')
  expectError(() => mockConfirmDispense(token, { items: [{ chargeItemId: drugCharge.chargeItemId, dispenseQty: 1 }] }), 'NOT_CHARGED')
  assert.equal(listInventoryItems({ itemNo: 'DRUG-A' }).items[0].qtyOnHand, beforeQty, 'refund must not alter stock after return')
  assert.equal(listLedger(sku.skuId).items.length, beforeLedger + 2)
  assert.deepEqual(listLedger(sku.skuId).items.slice(0, 2).map((item) => item.bizType), ['RETURN', 'DISPENSE'], 'business inventory types stay explicit')
  console.log(`PASS new patient -> checked-in register -> commit -> charge -> dispense -> return -> refund (${storageMode})`)
}

runChain('memory')
const values = new Map()
globalThis.sessionStorage = { getItem: (key) => values.get(key) ?? null, setItem: (key, value) => values.set(key, String(value)), removeItem: (key) => values.delete(key) }
runChain('sessionStorage')
delete globalThis.sessionStorage
