// 2026-10-12 P1-H05-CHARGE-UI 收费 Mock：挂号、提交、现金、库存、退费。不访问 /charges。
import { readFileSync } from 'node:fs'
import http from 'node:http'
import { ApiError } from '../src/api/errors.js'
import { mockLogin, resetMockAuth } from '../src/api/mockAuth.js'
import { lineStock } from '../src/api/chargeContract.js'
import {
  mockGetCharge,
  mockListCharges,
  mockLoadPending,
  mockLookup,
  mockPay,
  mockRefund,
  resetMockCharges,
} from '../src/api/mockCharges.js'

function assert(condition, message) {
  if (!condition) throw new Error(message)
}

async function expectError(fn, code, status) {
  try {
    await fn()
  } catch (error) {
    assert(error instanceof ApiError, 'expected ApiError')
    assert(error.code === code, `expected code ${code}, got ${error.code}`)
    if (status != null) assert(error.status === status, `expected status ${status}, got ${error.status}`)
    return error
  }
  throw new Error(`expected ${code}`)
}

process.env.VITE_USE_MOCK = 'true'
resetMockAuth()
resetMockCharges()

const token = mockLogin({ username: 'system', password: '1' }).accessToken

function subject(name) {
  const found = mockLookup(token, name).items.find((item) => item.patientName === name)
  assert(found, `missing subject ${name}`)
  return found
}

const none = mockLoadPending(token, subject('李无号').patientId)
assert(none.scene === 'NO_REGISTER' && none.preview == null, 'register required')
assert(none.register == null, 'no encounter')

const draft = mockLoadPending(token, subject('王未开').patientId)
assert(draft.scene === 'NO_COMMIT' && draft.preview == null, 'commit required')
assert(draft.register && draft.register.status === 'NORMAL', 'register present')

const ready = mockLoadPending(token, subject('赵待收').patientId)
assert(ready.scene === 'PENDING', 'pending scene')
assert(ready.preview.favourableFeeCents === 0, 'favourable locked')
assert(ready.preview.amountPaidCents === ready.preview.amountDueCents, 'paid equals due')
assert(ready.preview.amountDueCents === 4300, 'due cents')
assert(ready.preview.payType === 'CASH', 'cash default')
assert(ready.preview.items.every((item) => item.stockSufficient), 'stock ok')
assert(ready.order.status === 'COMMIT', 'commit order')

await expectError(
  () => mockPay(token, { orderId: ready.preview.orderId, payType: 'WECHAT', amountCents: 4300 }),
  'VALIDATION_ERROR',
  400,
)
await expectError(
  () => mockPay(token, { orderId: ready.preview.orderId, payType: 'ALIPAY', amountCents: 4300 }),
  'VALIDATION_ERROR',
  400,
)
await expectError(
  () => mockPay(token, { orderId: ready.preview.orderId, payType: 'CASH', amountCents: 9999 }),
  'CHARGE_AMOUNT_MISMATCH',
  400,
)

const short = mockLoadPending(token, subject('钱缺货').patientId)
assert(short.scene === 'STOCK' && short.preview.stockBlocked, 'stock scene')
assert(short.preview.items.some((item) => item.stockHint.includes('库存不足')), 'stock hint')
await expectError(
  () => mockPay(token, { orderId: short.preview.orderId, payType: 'CASH', amountCents: short.preview.amountDueCents }),
  'INSUFFICIENT_STOCK',
  409,
)
assert(!mockListCharges(token, { keyword: '钱缺货' }).items.length, 'stock pay writes nothing')

const paid = mockPay(token, { orderId: ready.preview.orderId, payType: 'CASH', amountCents: 4300 })
assert(paid.charge.status === 'NORMAL', 'paid normal')
assert(paid.charge.favourableFeeCents === 0 && paid.charge.squareFeeCents === 4300, 'square fee')
assert(paid.payment.direction === 'CHARGE' && paid.payment.payType === 'CASH' && paid.payment.status === 'SUCCEEDED', 'cash payment')
await expectError(
  () => mockPay(token, { orderId: ready.preview.orderId, payType: 'CASH', amountCents: 4300 }),
  'ALREADY_CHARGED',
  409,
)

const amox = paid.charge.items.find((item) => item.itemNo === 'AMX')
const partial = mockRefund(token, paid.charge.chargeId, {
  payType: 'CASH',
  reason: '患者要求',
  items: [{ chargeItemId: amox.chargeItemId, refundQty: 1 }],
})
assert(partial.abatedSquare.status === 'ABATE', 'partial abates')
assert(partial.newSquare && partial.newSquare.squareFeeCents === 3050, 'remainder slip')
assert(partial.newSquare.favourableFeeCents === 0 && partial.fullRefund === false, 'partial not zero')

const sun = mockListCharges(token, { keyword: '孙已收' }).items.find((item) => item.status === 'NORMAL')
const sunDetail = mockGetCharge(token, sun.chargeId)
const full = mockRefund(token, sun.chargeId, {
  payType: 'CASH',
  items: sunDetail.items.map((item) => ({ chargeItemId: item.chargeItemId, refundQty: item.refundableQty })),
})
assert(full.fullRefund === true && full.newSquare == null, 'full refund has no new slip')
const listed = mockListCharges(token, {})
assert(!listed.items.some((item) => item.status === 'NORMAL' && item.squareFeeCents === 0), 'no zero normal slip')
assert(listed.items.some((item) => item.chargeId === sun.chargeId && item.status === 'ABATE' && item.squareFeeCents === 3950), 'abated keeps amount')
await expectError(
  () => mockRefund(token, sun.chargeId, {
    payType: 'CASH',
    items: [{ chargeItemId: sunDetail.items[0].chargeItemId, refundQty: 1 }],
  }),
  'ALREADY_ABATED',
  409,
)

const zhou = mockListCharges(token, { keyword: '周已发' }).items[0]
const zhouDetail = mockGetCharge(token, zhou.chargeId)
const drug = zhouDetail.items.find((item) => item.dispenseBlocked)
const locked = zhouDetail.items.find((item) => item.canBack === false)
await expectError(
  () => mockRefund(token, zhou.chargeId, { payType: 'CASH', items: [{ chargeItemId: drug.chargeItemId, refundQty: 1 }] }),
  'REFUND_REQUIRES_RETURN',
  409,
)
await expectError(
  () => mockRefund(token, zhou.chargeId, { payType: 'CASH', items: [{ chargeItemId: locked.chargeItemId, refundQty: 1 }] }),
  'ITEM_NOT_CAN_BACK',
  409,
)
assert(mockGetCharge(token, zhou.chargeId).status === 'NORMAL', 'blocked refund keeps normal')

const remain = partial.newSquare.items.find((item) => item.itemNo === 'AMX')
await expectError(
  () => mockRefund(token, partial.newSquare.chargeId, { payType: 'CASH', items: [{ chargeItemId: remain.chargeItemId, refundQty: 9 }] }),
  'REFUND_EXCEEDS_REFUNDABLE',
  400,
)
await expectError(
  () => mockRefund(token, partial.newSquare.chargeId, { payType: 'CASH', items: [{ chargeItemId: remain.chargeItemId, refundQty: 0 }] }),
  'INVALID_AMOUNT',
  400,
)

const view = readFileSync(new URL('../src/views/ChargeView.vue', import.meta.url), 'utf8')
const template = view.split('<template>')[1]
for (const text of ['请先挂号', '请先提交处方', '确认收费', '预留', '已发药，请先退药再退费', '库存不足，无法收费', '未生成新收费单']) {
  assert(view.includes(text), `missing copy: ${text}`)
}
const contract = readFileSync(new URL('../src/api/chargeContract.js', import.meta.url), 'utf8')
assert(contract.includes("label: '微信'") && contract.includes("label: '支付宝'"), 'reserved pay labels')
assert(view.includes('RESERVED_PAY_TYPES') && view.includes('disabled'), 'reserved pay is disabled')
assert(!/>\s*作废\s*</.test(template), 'no void action')
assert(!view.includes('作废处方') && !view.includes('作废已收费'), 'no void charged order')
assert(!/weixin:|alipay\.com|open\.weixin/i.test(view), 'no electronic checkout')

const apiSrc = readFileSync(new URL('../src/api/chargeApi.js', import.meta.url), 'utf8')
assert(/fetch\(/.test(apiSrc), 'charge client calls HTTP when mock is off')
assert(apiSrc.includes('useMock'), 'mock still gates the client')
assert(apiSrc.includes('/charge-preview'), 'preview path')
assert(apiSrc.includes("payType: 'CASH'") || apiSrc.includes('payType: PAY_CASH'), 'cash pay')
assert(apiSrc.includes('amount: created.squareFee'), 'pay amount is server squareFee')
const moduleSrc = readFileSync(new URL('../src/views/OutpatientModuleView.vue', import.meta.url), 'utf8')
assert(moduleSrc.includes('ChargeView'), 'charge page is mounted')
assert(!moduleSrc.includes("useMock() && allowed('charge')"), 'real mode is not mock-only')

assert(lineStock('DRUG-A', 2).sufficient, 'catalog stock allows DRUG-A')
assert(!lineStock('DRUG-B', 2).sufficient, 'catalog stock blocks DRUG-B')
assert(lineStock('TREAT-1', 1).sufficient, 'treat does not use stock')

const PATIENT = '11111111-1111-4111-8111-111111111101'
const REGISTER = '11111111-1111-4111-8111-111111111102'
const calls = []
let settled = false

function chargeDto(overrides = {}) {
  return {
    chargeId: overrides.chargeId ?? 'ch-1',
    squareNo: overrides.squareNo ?? 'S20261012001',
    orderId: overrides.orderId ?? 'ord-1',
    registerId: REGISTER,
    encounterId: REGISTER,
    patientId: PATIENT,
    totalFee: 20,
    squareFee: 20,
    favourableFee: 0,
    status: overrides.status ?? 'NORMAL',
    successorChargeId: null,
    abatedFromId: null,
    paidAt: overrides.paidAt ?? null,
    items: [
      {
        chargeItemId: 'ci-1',
        orderItemId: 'oi-1',
        lineNo: 1,
        itemNo: 'DRUG-A',
        itemName: '演示阿莫西林胶囊',
        qty: 2,
        unitPrice: 10,
        totalFee: 20,
        canBack: true,
        refundedQty: 0,
      },
    ],
  }
}

function orderDto(status, charged) {
  return {
    orderId: 'ord-1',
    orderNo: 'O20261012001',
    registerId: REGISTER,
    patientId: PATIENT,
    departmentId: 'dept-1',
    doctorId: 'doc-1',
    status,
    totalFee: 20,
    charged,
    items: [
      {
        lineNo: 1,
        itemNo: 'DRUG-A',
        itemType: 'DRUG',
        itemName: '演示阿莫西林胶囊',
        amount: 2,
        price: 10,
        totalFee: 20,
      },
    ],
  }
}

const server = http.createServer(async (req, res) => {
  const url = new URL(req.url || '/', 'http://127.0.0.1')
  const chunks = []
  for await (const chunk of req) chunks.push(chunk)
  const raw = Buffer.concat(chunks).toString()
  calls.push({ method: req.method, path: url.pathname, search: url.search, body: raw })
  const json = (status, payload) => {
    res.writeHead(status, { 'Content-Type': 'application/json; charset=utf-8' })
    res.end(JSON.stringify(payload))
  }
  const fail = (status, code, message) => json(status, {
    success: false,
    data: null,
    error: { code, message, details: {} },
    requestId: 'req-charge',
  })
  if (!String(req.headers.authorization || '').startsWith('Bearer ')) {
    fail(401, 'UNAUTHORIZED', '未认证')
    return
  }
  if (req.method === 'GET' && url.pathname === '/api/v1/registers') {
    json(200, {
      items: [{
        registerId: REGISTER,
        registerNo: 'G202610120001',
        patientId: PATIENT,
        departmentId: 'dept-1',
        doctorId: 'doc-1',
        visitDate: '2026-10-12',
        status: 'NORMAL',
        registerFee: 0,
        registerCharged: false,
      }],
    })
    return
  }
  if (req.method === 'GET' && url.pathname === `/api/v1/registers/${REGISTER}`) {
    json(200, {
      registerId: REGISTER,
      registerNo: 'G202610120001',
      patientId: PATIENT,
      departmentId: 'dept-1',
      doctorId: 'doc-1',
      visitDate: '2026-10-12',
      status: 'NORMAL',
      registerFee: 0,
      registerCharged: false,
    })
    return
  }
  if (req.method === 'GET' && url.pathname === `/api/v1/patients/${PATIENT}`) {
    json(200, { patientId: PATIENT, name: '赵待收', patientNo: 'P2026101203', status: 'ACTIVE' })
    return
  }
  if (req.method === 'GET' && url.pathname === '/api/v1/orders') {
    json(200, { items: [orderDto('COMMIT', settled)] })
    return
  }
  if (req.method === 'GET' && url.pathname === '/api/v1/orders/ord-1') {
    json(200, orderDto('COMMIT', settled))
    return
  }
  if (req.method === 'GET' && url.pathname === '/api/v1/orders/ord-1/charge-preview') {
    json(200, chargeDto({ chargeId: '', squareNo: '' }))
    return
  }
  if (req.method === 'POST' && url.pathname === '/api/v1/charges') {
    const body = JSON.parse(raw || '{}')
    if (body.orderId === 'ord-short') {
      fail(409, 'INSUFFICIENT_STOCK', '库存不足')
      return
    }
    json(201, chargeDto())
    return
  }
  if (req.method === 'GET' && url.pathname === '/api/v1/charges') {
    json(200, { items: settled ? [chargeDto({ paidAt: '2026-10-12T01:20:00Z' })] : [] })
    return
  }
  if (req.method === 'GET' && url.pathname === '/api/v1/charges/ch-1') {
    json(200, chargeDto({ paidAt: settled ? '2026-10-12T01:20:00Z' : null }))
    return
  }
  if (req.method === 'POST' && url.pathname === '/api/v1/charges/ch-1/pay') {
    const body = JSON.parse(raw || '{}')
    if (body.payType !== 'CASH' || body.amount !== 20) {
      fail(409, 'CHARGE_AMOUNT_MISMATCH', '实收与应收不一致')
      return
    }
    settled = true
    json(200, {
      paymentId: 'pay-1',
      paymentNo: 'P20261012001',
      chargeId: 'ch-1',
      direction: 'CHARGE',
      payType: 'CASH',
      amount: 20,
      status: 'SUCCEEDED',
      paidAt: '2026-10-12T01:20:00Z',
      succeededAt: '2026-10-12T01:20:00Z',
    })
    return
  }
  if (req.method === 'POST' && url.pathname === '/api/v1/charges/ch-1/refund') {
    json(200, {
      abatedCharge: chargeDto({ status: 'ABATE', paidAt: '2026-10-12T01:20:00Z' }),
      newSquare: null,
      refundPayment: {
        paymentId: 'pay-2',
        paymentNo: 'R20261012001',
        chargeId: 'ch-1',
        direction: 'REFUND',
        payType: 'CASH',
        amount: 20,
        status: 'SUCCEEDED',
        paidAt: '2026-10-12T01:30:00Z',
      },
    })
    return
  }
  fail(404, 'NOT_FOUND', '未找到')
})

await new Promise((resolve) => server.listen(0, '127.0.0.1', resolve))
const { port } = server.address()
process.env.VITE_USE_MOCK = 'false'
process.env.VITE_API_BASE_URL = `http://127.0.0.1:${port}/api/v1`

try {
  const { listCharges, loadPendingCharge, lookupChargeSubjects, payCharge, refundCharge } = await import('../src/api/chargeApi.js')
  const looked = await lookupChargeSubjects(token, '')
  assert(looked.items.length === 1 && looked.items[0].scene === 'PENDING', 'real lookup pending')
  assert(looked.items[0].registerId === REGISTER, 'registerId primary')
  const pending = await loadPendingCharge(token, looked.items[0])
  assert(pending.scene === 'PENDING' && pending.preview.squareFee === 20, 'preview squareFee')
  assert(pending.preview.favourableFeeCents === 0, 'preview favourable')
  assert(pending.preview.amountPaidCents === pending.preview.amountDueCents, 'paid equals due')
  assert(!pending.preview.stockBlocked, 'preview stock')

  const paidResult = await payCharge(token, {
    orderId: 'ord-1',
    registerId: REGISTER,
    payType: 'CASH',
    amount: 1,
    amountCents: 100,
  })
  assert(paidResult.payment.payType === 'CASH' && paidResult.payment.amount === 20, 'payment uses squareFee')
  assert(paidResult.charge.paid === true && paidResult.charge.status === 'NORMAL', 'charge paid')
  const payCall = calls.find((item) => item.path.endsWith('/pay'))
  assert(payCall && JSON.parse(payCall.body).amount === 20 && JSON.parse(payCall.body).payType === 'CASH', 'wire amount is squareFee')
  assert(calls.some((item) => item.path.endsWith('/charge-preview')), 'preview was requested')
  assert(calls.some((item) => item.method === 'POST' && item.path === '/api/v1/charges'), 'create charge')

  await expectError(
    () => payCharge(token, { orderId: 'ord-short', registerId: REGISTER, payType: 'CASH', amount: 20 }),
    'INSUFFICIENT_STOCK',
    409,
  )

  const listed = await listCharges(token, { todayOnly: true })
  assert(listed.items.some((item) => item.squareNo === 'S20261012001' && item.patientName === '赵待收'), 'query list')

  const refunded = await refundCharge(token, 'ch-1', {
    payType: 'CASH',
    reason: '患者要求',
    items: [{ chargeItemId: 'ci-1', refundQty: 2 }],
  })
  assert(refunded.fullRefund === true && refunded.newSquare == null, 'full refund has no new slip')
} finally {
  await new Promise((resolve, reject) => server.close((error) => (error ? reject(error) : resolve())))
}

console.log('charge mock checks passed')
