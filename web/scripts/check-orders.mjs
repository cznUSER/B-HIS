// 2026-10-12 P1-H05-ORDER-UI 开单 Mock 对齐 OpenAPI 0.3.7。不访问 /orders。
import { readFileSync } from 'node:fs'
import { ApiError } from '../src/api/errors.js'
import { mockLogin, resetMockAuth } from '../src/api/mockAuth.js'
import {
  mockAbateOrder,
  mockCommitOrder,
  mockCreateOrder,
  mockGetOrderRegister,
  mockListOrderRegisters,
  mockMarkOrderCharged,
  mockPatchOrder,
  resetMockOrders,
} from '../src/api/mockOrders.js'
import { ORDER_ROUTES, abateOrder, commitOrder, saveOrder } from '../src/api/orderApi.js'
import { assertCreateOrderRequest, stockWarnings } from '../src/api/orderContract.js'

function assert(condition, message) {
  if (!condition) throw new Error(message)
}

async function expectError(fn, code, status) {
  try {
    await fn()
  } catch (error) {
    assert(error instanceof ApiError, 'expected ApiError')
    assert(error.code === code, `expected code ${code}, got ${error.code}: ${error.message}`)
    if (status != null) assert(error.status === status, `expected status ${status}, got ${error.status}`)
    return error
  }
  throw new Error(`expected ${code}`)
}

const REGISTER_ID = '11111111-1111-4111-8111-111111111111'
const BACK_ID = '11111111-1111-4111-8111-111111111113'

function draft(overrides = {}, items = [{ itemNo: 'DRUG-A', amount: 2 }]) {
  return {
    registerId: REGISTER_ID,
    departmentId: 'dept-1',
    doctorId: 'doc-1',
    items,
    ...overrides,
  }
}

process.env.VITE_USE_MOCK = 'true'
resetMockAuth()
resetMockOrders()

const apiSrc = readFileSync(new URL('../src/api/orderApi.js', import.meta.url), 'utf8')
assert(!/\bfetch\s*\(/.test(apiSrc), 'order client does not call HTTP')
assert(ORDER_ROUTES.commit('ord-1') === '/orders/ord-1/commit', 'commit path')
assert(ORDER_ROUTES.abate('ord-1') === '/orders/ord-1/abate', 'abate path')
assert(ORDER_ROUTES.item('ord-1') === '/orders/ord-1', 'draft path')
assert(!apiSrc.includes('/submit'), 'commit is not a submit alias client')

const token = mockLogin({ username: 'system', password: '1' }).accessToken

const missing = await expectError(() => assertCreateOrderRequest({ items: [] }), 'ENCOUNTER_REQUIRED', 409)
assert(missing.message === '请先挂号', 'missing register')
const mismatch = await expectError(
  () => assertCreateOrderRequest({ registerId: REGISTER_ID, encounterId: 'other' }),
  'VALIDATION_ERROR',
  400,
)
assert(mismatch.message === 'registerId 与 encounterId 不一致', 'alias must match')
await expectError(() => assertCreateOrderRequest(draft({}, [{ itemNo: 'OTHER', amount: 1 }])), 'CATALOG_ITEM_NOT_FOUND', 409)
const zero = await expectError(() => assertCreateOrderRequest(draft({}, [{ itemNo: 'DRUG-A', amount: 0 }])), 'ORDER_INVALID_AMOUNT', 409)
assert(zero.message === '数量必须大于 0', 'amount')

const normalized = assertCreateOrderRequest({
  encounterId: REGISTER_ID,
  doctorId: 'doc-2',
  items: [{ itemNo: 'DRUG-A', amount: '2', price: 1, usageText: '口服', frequencyText: '一日三次' }],
})
assert(normalized.registerId === REGISTER_ID, 'encounterId alias becomes registerId')
assert(!Object.prototype.hasOwnProperty.call(normalized, 'encounterId'), 'request keeps registerId')
assert(normalized.doctorId === 'doc-2', 'doctor can change')
assert(normalized.recipeSort === 'NORMAL', 'ordinary recipe')
assert(!Object.prototype.hasOwnProperty.call(normalized, 'diagnoseName'), 'blank diagnose omitted')
assert(normalized.items[0].price === 10 && normalized.items[0].totalFee === 20, 'catalog price')
assert(normalized.items[0].itemType === 'DRUG' && normalized.items[0].units === '盒', 'drug units')
assert(normalized.items[0].usageText === '口服', 'usage text')
assert(!Object.prototype.hasOwnProperty.call(normalized.items[0], 'unit'), 'no legacy unit')

const both = assertCreateOrderRequest(draft({}, [
  { itemNo: 'DRUG-A', amount: 1 },
  { itemNo: 'DRUG-B', amount: 1 },
  { itemNo: '', amount: '' },
]))
assert(both.items.length === 2 && both.items[0].lineNo === 1 && both.items[1].lineNo === 2, 'line numbers')
assert(both.items.reduce((sum, item) => sum + item.totalFee, 0) === 18, 'A+B total')
const treat = assertCreateOrderRequest(draft({}, [{ itemNo: 'TREAT-1', amount: 1 }]))
assert(treat.items[0].itemType === 'TREAT' && treat.items[0].totalFee === 15, 'treat line')
assert(stockWarnings([{ itemNo: 'DRUG-B', amount: 2 }]).length === 1, 'stock tip')
assert(stockWarnings(treat.items).length === 0, 'treat has no stock gate')

const registers = mockListOrderRegisters(token)
assert(registers.items.some((item) => item.registerId === REGISTER_ID && item.status === 'NORMAL'), 'demo register')
const back = mockGetOrderRegister(token, BACK_ID)
assert(back.status === 'BACK', 'back register readable')
const refused = await expectError(() => mockCreateOrder(token, draft({ registerId: BACK_ID })), 'ENCOUNTER_REQUIRED', 409)
assert(refused.message === '请先挂号', 'back register blocked')
await expectError(() => mockCreateOrder(token, { registerId: 'missing', items: [] }), 'ENCOUNTER_REQUIRED', 409)

const empty = mockCreateOrder(token, { registerId: REGISTER_ID })
assert(empty.status === 'SAVE' && empty.items.length === 0 && empty.totalFee === 0, 'header draft')
assert(empty.registerId === REGISTER_ID && empty.encounterId === REGISTER_ID, 'alias echoed')
assert(empty.doctorId === 'doc-1' && empty.departmentId === 'dept-1', 'defaults from register')
await expectError(() => mockCommitOrder(token, empty.orderId), 'ORDER_EMPTY_ITEMS', 409)

const saved = mockCreateOrder(token, draft({
  diagnoseName: '上感',
  doctorId: 'doc-2',
  items: [
    { itemNo: 'DRUG-A', amount: 2, price: 1, usageText: '口服' },
    { itemNo: 'DRUG-B', amount: 2 },
  ],
}))
assert(saved.status === 'SAVE' && saved.totalFee === 36, 'draft total keeps shortage')
assert(saved.items[0].price === 10 && saved.items[0].totalFee === 20, 'price not client-supplied')
assert(saved.charged === false && saved.doctorId === 'doc-2' && saved.diagnoseName === '上感', 'header')
assert(saved.orderNo.startsWith('R'), 'system recipe no')
assert(stockWarnings(saved.items).length === 1, 'shortage does not block save')

const edited = mockPatchOrder(token, saved.orderId, draft({ items: [{ itemNo: 'DRUG-A', amount: 2 }] }))
assert(edited.orderId === saved.orderId && edited.orderNo === saved.orderNo && edited.totalFee === 20, 'patch draft')
const committed = mockCommitOrder(token, edited.orderId)
assert(committed.status === 'COMMIT' && committed.submittedAt && committed.totalFee === 20 && committed.charged === false, 'chargeable')
await expectError(() => mockPatchOrder(token, committed.orderId, draft()), 'ORDER_NOT_EDITABLE', 409)
const abated = mockAbateOrder(token, committed.orderId)
assert(abated.status === 'ABATE' && abated.abatedAt, 'unpaid abate')

const paid = mockCommitOrder(token, mockCreateOrder(token, draft()).orderId)
mockMarkOrderCharged(paid.orderId)
const charged = await expectError(() => mockAbateOrder(token, paid.orderId), 'ORDER_CHARGED_NOT_ABATABLE', 409)
assert(charged.message === '已收费，不能作废', 'charged abate')

const viaApi = await saveOrder(token, draft({ diagnoseName: '上感' }))
const viaCommit = await commitOrder(token, viaApi.orderId)
assert(viaCommit.status === 'COMMIT' && viaCommit.registerId === REGISTER_ID && viaCommit.totalFee === 20, 'facade commit')
assert((await abateOrder(token, viaCommit.orderId)).status === 'ABATE', 'facade abate')

process.env.VITE_USE_MOCK = 'false'
await expectError(() => saveOrder(token, draft()), 'NOT_IMPLEMENTED', 501)

console.log('order checks passed')
