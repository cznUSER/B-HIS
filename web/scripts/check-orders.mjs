// 2026-10-12 P1-H05-ORDER-UI 开单对齐 OpenAPI 0.3.7。
// 2026-10-12 Mock 仍走本地；VITE_USE_MOCK=false 时打本脚本内的 HTTP，不访问 Go API。
import { readFileSync } from 'node:fs'
import http from 'node:http'
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
import { ORDER_ROUTES, abateOrder, commitOrder, fetchOrderRegister, listOrderRegisters, listOrders, presentOrderError, saveOrder } from '../src/api/orderApi.js'
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
assert(/\bfetch\s*\(/.test(apiSrc), 'order client calls HTTP when mock is off')
assert(!apiSrc.includes('/check-in'), 'order client does not check in')
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

const encounterCopy = presentOrderError({
  code: 'ENCOUNTER_REQUIRED',
  status: 409,
  message: '需要有效挂号',
  details: null,
  requestId: 'r-enc',
})
assert(encounterCopy.message === '请先挂号', 'encounter copy')

const REGISTER_ID_HTTP = '11111111-1111-4111-8111-111111111111'
const PATIENT_ID_HTTP = '22222222-2222-4222-8222-222222222221'

function orderDto(status, extra = {}) {
  return {
    orderId: 'ord-1',
    orderNo: 'R202610120001',
    registerId: REGISTER_ID_HTTP,
    encounterId: REGISTER_ID_HTTP,
    patientId: PATIENT_ID_HTTP,
    departmentId: 'dept-1',
    doctorId: 'doc-2',
    diagnoseName: '上感',
    recipeSort: 'NORMAL',
    status,
    totalFee: 20,
    charged: false,
    items: [
      {
        orderItemId: 'item-1',
        lineNo: 1,
        itemNo: 'DRUG-A',
        itemType: 'DRUG',
        itemName: '演示阿莫西林胶囊',
        spec: '0.25g×24',
        units: '盒',
        amount: 2,
        price: 10,
        totalFee: 20,
        usageText: '口服',
      },
    ],
    submittedAt: status === 'COMMIT' ? '2026-10-12T02:00:00Z' : null,
    abatedAt: status === 'ABATE' ? '2026-10-12T03:00:00Z' : null,
    abatedBy: status === 'ABATE' ? 'system' : null,
    ...extra,
  }
}

const calls = []
const server = http.createServer(async (req, res) => {
  const url = new URL(req.url || '/', 'http://127.0.0.1')
  const chunks = []
  for await (const chunk of req) chunks.push(chunk)
  const raw = Buffer.concat(chunks).toString()
  calls.push({ method: req.method, path: url.pathname, search: url.search, body: raw, headers: req.headers })
  const json = (status, payload) => {
    res.writeHead(status, { 'Content-Type': 'application/json; charset=utf-8' })
    res.end(JSON.stringify(payload))
  }
  if (!String(req.headers.authorization || '').startsWith('Bearer ')) {
    json(401, {
      success: false,
      data: null,
      error: { code: 'UNAUTHORIZED', message: '未认证', details: {} },
      requestId: 'req-auth',
    })
    return
  }
  if (req.method === 'GET' && url.pathname === `/api/v1/patients/${PATIENT_ID_HTTP}`) {
    json(200, {
      patientId: PATIENT_ID_HTTP,
      name: '李演示',
      patientNo: 'P202610120001',
      sex: 'M',
      birthday: '1990-01-15',
      fareIdentity: 'SELF',
      status: 'ACTIVE',
      cardStatus: 'NOT_SENT',
    })
    return
  }
  if (req.method === 'GET' && url.pathname === `/api/v1/registers/${REGISTER_ID_HTTP}`) {
    json(200, {
      registerId: REGISTER_ID_HTTP,
      registerNo: 'G202610120001',
      patientId: PATIENT_ID_HTTP,
      departmentId: 'dept-1',
      doctorId: 'doc-1',
      visitDate: '2026-10-12',
      status: 'NORMAL',
      registerFee: 0,
      registerCharged: false,
      registeredAt: '2026-10-12T01:00:00Z',
    })
    return
  }
  if (req.method === 'GET' && url.pathname === '/api/v1/registers') {
    json(200, {
      items: [
        {
          registerId: REGISTER_ID_HTTP,
          registerNo: 'G202610120001',
          patientId: PATIENT_ID_HTTP,
          departmentId: 'dept-1',
          doctorId: 'doc-1',
          visitDate: '2026-10-12',
          status: 'NORMAL',
          registerFee: 0,
          registerCharged: false,
          registeredAt: '2026-10-12T01:00:00Z',
        },
      ],
    })
    return
  }
  if (req.method === 'POST' && url.pathname === '/api/v1/orders') {
    const body = JSON.parse(raw || '{}')
    if (!body.registerId) {
      json(409, {
        success: false,
        data: null,
        error: { code: 'ENCOUNTER_REQUIRED', message: '需要有效挂号', details: {} },
        requestId: 'req-enc',
      })
      return
    }
    json(201, orderDto('SAVE', { doctorId: body.doctorId || 'doc-1', diagnoseName: body.diagnoseName || null }))
    return
  }
  if (req.method === 'PATCH' && url.pathname === '/api/v1/orders/ord-1') {
    json(200, orderDto('SAVE', { totalFee: 20 }))
    return
  }
  if (req.method === 'GET' && url.pathname === '/api/v1/orders') {
    json(200, { items: [orderDto('SAVE')] })
    return
  }
  if (req.method === 'POST' && url.pathname === '/api/v1/orders/ord-1/commit') {
    json(200, orderDto('COMMIT'))
    return
  }
  if (req.method === 'POST' && url.pathname === '/api/v1/orders/ord-1/abate') {
    json(200, orderDto('ABATE'))
    return
  }
  json(404, {
    success: false,
    data: null,
    error: { code: 'NOT_FOUND', message: '资源不存在', details: {} },
    requestId: 'req-miss',
  })
})

await new Promise((resolve) => {
  server.listen(0, '127.0.0.1', resolve)
})
const { port } = server.address()
process.env.VITE_USE_MOCK = 'false'
process.env.VITE_API_BASE_URL = `http://127.0.0.1:${port}/api/v1`

try {
  const found = await fetchOrderRegister('header.payload.sig', REGISTER_ID_HTTP)
  assert(found.status === 'NORMAL' && found.patientName === '李演示' && found.doctorName === '张医生', 'real register')
  assert(found.departmentName === '内科', 'dept name')
  const day = await listOrderRegisters('header.payload.sig')
  assert(day.items.length === 1 && day.items[0].registerId === REGISTER_ID_HTTP, 'today registers')
  const listCall = calls.find((item) => item.method === 'GET' && item.path === '/api/v1/registers')
  assert(listCall.search.includes('visitDate='), 'list by visit date')
  assert(!calls.some((item) => item.path.includes('/check-in')), 'no check-in')

  const created = await saveOrder('header.payload.sig', draft({
    registerId: REGISTER_ID_HTTP,
    doctorId: 'doc-2',
    diagnoseName: '上感',
  }))
  assert(created.status === 'SAVE' && created.registerId === REGISTER_ID_HTTP && created.totalFee === 20, 'real save')
  const post = calls.find((item) => item.method === 'POST' && item.path === '/api/v1/orders')
  const posted = JSON.parse(post.body)
  assert(posted.registerId === REGISTER_ID_HTTP, 'post registerId')
  assert(!Object.prototype.hasOwnProperty.call(posted, 'encounterId'), 'post keeps registerId')
  assert(posted.doctorId === 'doc-2' && posted.diagnoseName === '上感', 'doctor and diagnose')
  assert(posted.recipeSort === 'NORMAL', 'ordinary recipe')
  assert(posted.items[0].itemNo === 'DRUG-A' && posted.items[0].amount === 2 && posted.items[0].price === 10, 'catalog line')
  assert(typeof post.headers['idempotency-key'] === 'string', 'create idempotency')

  const edited = await saveOrder('header.payload.sig', draft({
    orderId: created.orderId,
    registerId: REGISTER_ID_HTTP,
    doctorId: 'doc-2',
    diagnoseName: '',
    items: [{ itemNo: 'DRUG-A', amount: 2 }],
  }))
  assert(edited.status === 'SAVE' && edited.orderId === 'ord-1', 'real patch')
  const patch = calls.find((item) => item.method === 'PATCH')
  const patched = JSON.parse(patch.body)
  assert(patched.doctorId === 'doc-2' && patched.diagnoseName === '', 'patch header')
  assert(!Object.prototype.hasOwnProperty.call(patched, 'registerId'), 'patch does not move register')
  assert(patched.items.length === 1 && patched.items[0].itemNo === 'DRUG-A', 'patch replaces items')

  const listed = await listOrders('header.payload.sig', REGISTER_ID_HTTP)
  assert(listed.items.length === 1 && listed.items[0].status === 'SAVE', 'real list')
  const search = calls.find((item) => item.method === 'GET' && item.path === '/api/v1/orders')
  assert(search.search.includes(`registerId=${REGISTER_ID_HTTP}`), 'list by registerId')

  const committed = await commitOrder('header.payload.sig', created.orderId)
  assert(committed.status === 'COMMIT' && committed.totalFee === 20, 'real commit')
  assert(calls.some((item) => item.method === 'POST' && item.path === '/api/v1/orders/ord-1/commit'), 'commit path used')
  assert(!calls.some((item) => item.path.endsWith('/submit')), 'no submit alias')
  const abated = await abateOrder('header.payload.sig', created.orderId)
  assert(abated.status === 'ABATE', 'real abate')

  await expectError(() => saveOrder('header.payload.sig', { items: [] }), 'ENCOUNTER_REQUIRED', 409)
  const refused = await expectError(
    () => saveOrder('', draft({ registerId: REGISTER_ID_HTTP })),
    'UNAUTHORIZED',
    401,
  )
  assert(refused.code === 'UNAUTHORIZED', 'missing token')
} finally {
  await new Promise((resolve) => server.close(resolve))
}

console.log('order checks passed')
