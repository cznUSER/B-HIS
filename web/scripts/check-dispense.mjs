// 2026-10-12 P1-H05-DISPENSE-UI 发药 Mock。
// 2026-10-12 真接口形状用本脚本里的假 HTTP，不访问 Go API。
import { readFileSync } from 'node:fs'
import http from 'node:http'
import { ApiError, apiError, messageForCode } from '../src/api/errors.js'
import { mockLogin, resetMockAuth } from '../src/api/mockAuth.js'
import {
  DISPENSE_ROUTES,
  confirmDispense,
  fetchDispense,
  listDispenses,
  listTodayPending,
  presentDispenseError,
  returnDispense,
  searchPending,
} from '../src/api/dispenseApi.js'
import {
  DEMO_IDS,
  mockConfirmDispense,
  mockListDispenses,
  mockListTodayPending,
  mockReturnDispense,
  mockSearchPending,
  resetMockDispenses,
} from '../src/api/mockDispenses.js'

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

const apiSrc = readFileSync(new URL('../src/api/dispenseApi.js', import.meta.url), 'utf8')
const viewSrc = readFileSync(new URL('../src/views/DispenseView.vue', import.meta.url), 'utf8')
assert(/\bfetch\s*\(/.test(apiSrc), 'dispense client calls HTTP')
assert(apiSrc.includes('if (useMock())'), 'mock switch still wraps the demo path')
assert(!apiSrc.includes('DISPENSE_ROUTES.orderAlias('), 'client does not call deprecated alias')
assert(DISPENSE_ROUTES.collection === '/dispenses', 'create path')
assert(DISPENSE_ROUTES.item('dsp-1') === '/dispenses/dsp-1', 'detail path')
assert(DISPENSE_ROUTES.return('dsp-1') === '/dispenses/dsp-1/return', 'return path')
assert(DISPENSE_ROUTES.orderAlias('ord-1') === '/orders/ord-1/dispense', 'deprecated alias')
assert(apiSrc.includes('useMock'), 'mock switch')
assert(viewSrc.includes('演示账号 system / 1'), 'login hint')
assert(!viewSrc.includes('发药接口尚未接入'), 'real mode is not a dead end')

for (const text of ['请选择已收费患者或处方', '暂无待发药品', '今日无待发', '暂无发药记录', '没有可退明细', '确认发药', '全部发药', '去退费', 'charge_id']) {
  assert(viewSrc.includes(text), `missing copy: ${text}`)
}
for (const text of ['采购', '调拨', '盘点', '批次', '追溯', '换算', '审方', 'admin']) {
  assert(!viewSrc.includes(text), `forbidden copy: ${text}`)
}
assert(messageForCode('NOT_CHARGED') === '未收费，不能发药', 'not charged copy')
assert(messageForCode('ALREADY_FULLY_DISPENSED') === '已全部发完，不能再发', 'full copy')
assert(messageForCode('ALREADY_DISPENSED') === '已全部发完，不能再发', 'openapi alias copy')
assert(messageForCode('INSUFFICIENT_STOCK') === '库存不足，无法收费', 'charge stock copy stays')
assert(presentDispenseError(apiError(409, 'INSUFFICIENT_STOCK', '库存不足')).message === '库存不足，无法发药', 'dispense stock copy')
assert(presentDispenseError(apiError(409, 'REFUND_REQUIRES_RETURN', '已发药须先退药')).message === '已发药，请先退药再退费', 'refund gate copy')
assert(presentDispenseError(apiError(409, 'IDEMPOTENT_REJECT', 'x')).message === '重复提交已处理，未再次扣减/回滚', 'idempotent copy')
assert(presentDispenseError(apiError(409, 'IDEMPOTENCY_KEY_CONFLICT', '幂等键与已有请求不一致')).message === '重复提交已处理，未再次扣减/回滚', 'idempotency conflict copy')
assert(presentDispenseError(apiError(409, 'ALREADY_DISPENSED', 'server')).message === '已全部发完，不能再发', 'already dispensed copy')
assert(presentDispenseError(apiError(409, 'ALREADY_FULLY_DISPENSED', 'server')).message === '已全部发完，不能再发', 'fully dispensed copy')

process.env.VITE_USE_MOCK = 'true'
resetMockAuth()
resetMockDispenses()
const token = mockLogin({ username: 'system', password: '1' }).accessToken

const unpaid = await expectError(() => mockSearchPending(token, '陈未收'), 'NOT_CHARGED', 409)
assert(unpaid.message === '未收费，不能发药', 'unpaid search')
await expectError(() => mockConfirmDispense(token, {
  items: [{ chargeItemId: DEMO_IDS.unpaid, dispenseQty: 1 }],
}), 'NOT_CHARGED', 409)

const today = mockListTodayPending(token)
const names = today.items.map((item) => item.patientName)
assert(names.includes('王全发') && names.includes('李部分') && names.includes('赵缺货'), 'pending cases')
assert(!names.includes('陈未收') && !names.includes('孙已发'), 'unpaid and finished stay out')
assert(today.items.every((item) => item.itemName !== '诊查费'), 'non-drug filtered')
const fullLine = today.items.find((item) => item.chargeItemId === DEMO_IDS.full)
assert(fullLine.lineDispenseStatus === 'NONE' && fullLine.availableQty === 2 && fullLine.units === '盒', 'full line')
assert(mockSearchPending(token, '王全发').items.length === 1, 'one drug line')
assert(mockSearchPending(token, '孙已发').empty === 'done' && mockSearchPending(token, '孙已发').items.length === 0, 'finished search')

await expectError(() => mockConfirmDispense(token, { items: [] }), 'EMPTY_DISPENSE_ITEMS', 400)
const exceeds = await expectError(() => mockConfirmDispense(token, {
  items: [{ chargeItemId: DEMO_IDS.full, dispenseQty: 3 }],
}), 'DISPENSE_EXCEEDS_AVAILABLE', 409)
assert(exceeds.message === '发药数量超过可发数量', 'exceeds copy')
assert(mockSearchPending(token, '王全发').items[0].stockQty === 20, 'exceeds keeps stock')

const short = await expectError(() => mockConfirmDispense(token, {
  items: [
    { chargeItemId: DEMO_IDS.full, dispenseQty: 2 },
    { chargeItemId: DEMO_IDS.short, dispenseQty: 8 },
  ],
}), 'INSUFFICIENT_STOCK', 409)
assert(short.message === '库存不足，无法发药', 'stock copy')
assert(mockSearchPending(token, '王全发').items[0].stockQty === 20, 'short keeps other stock')
assert(mockSearchPending(token, '赵缺货').items[0].stockQty === 2, 'short keeps own stock')

await expectError(() => mockConfirmDispense(token, {
  items: [{ chargeItemId: DEMO_IDS.treat, dispenseQty: 1 }],
}), 'VALIDATION_ERROR', 400)

const dispensed = mockConfirmDispense(token, {
  orderId: 'ord-full',
  chargeId: 'chg-full',
  items: [{ chargeItemId: DEMO_IDS.full, orderItemId: fullLine.orderItemId, dispenseQty: 2, selected: true }],
  idempotencyKey: 'once-full',
})
assert(dispensed.dispenseNo && dispensed.lines[0].lineDispenseStatus === 'FULL', 'full status')
assert(dispensed.lines[0].stockQty === 18, 'stock deducted')
assert(mockSearchPending(token, '王全发').items.length === 0, 'full leaves pending')
const again = await expectError(() => mockConfirmDispense(token, {
  items: [{ chargeItemId: DEMO_IDS.full, dispenseQty: 1 }],
}), 'ALREADY_FULLY_DISPENSED', 409)
assert(again.message === '已全部发完，不能再发', 'already full copy')
const replay = await expectError(() => mockConfirmDispense(token, {
  items: [{ chargeItemId: DEMO_IDS.part, dispenseQty: 1 }],
  idempotencyKey: 'once-full',
}), 'IDEMPOTENT_REJECT', 409)
assert(replay.message === '重复提交已处理，未再次扣减/回滚', 'idempotent copy')
assert(mockSearchPending(token, '李部分').items[0].stockQty === 30, 'replay keeps stock')

const partial = mockConfirmDispense(token, {
  items: [{ chargeItemId: DEMO_IDS.part, dispenseQty: 2 }],
})
assert(partial.lines[0].lineDispenseStatus === 'PARTIAL' && partial.lines[0].availableQty === 4, 'partial status')
assert(partial.lines[0].stockQty === 28, 'partial stock')
const rest = mockConfirmDispense(token, {
  items: [{ chargeItemId: DEMO_IDS.part, dispenseQty: 4 }],
})
assert(rest.lines[0].lineDispenseStatus === 'FULL' && rest.lines[0].stockQty === 24, 'second dispense fills')

const issued = mockListDispenses(token, { today: true, patient: '孙已发' })
assert(issued.items.some((item) => item.dispenseNo === 'FY-DONE-001' && item.status === 'DONE'), 'issued seed')

await expectError(() => mockReturnDispense(token, {
  dispenseId: DEMO_IDS.doneDispense,
  items: [{ dispenseItemId: DEMO_IDS.doneItem, returnQty: 1 }],
}), 'RETURN_REASON_REQUIRED', 400)
await expectError(() => mockReturnDispense(token, {
  dispenseId: 'missing',
  reason: '患者要求',
  items: [{ chargeItemId: DEMO_IDS.full, returnQty: 1 }],
}), 'NOT_DISPENSED', 409)
const tooMuch = await expectError(() => mockReturnDispense(token, {
  dispenseId: DEMO_IDS.doneDispense,
  reason: '患者要求',
  items: [{ dispenseItemId: DEMO_IDS.doneItem, returnQty: 5 }],
}), 'RETURN_EXCEEDS_DISPENSE', 409)
assert(tooMuch.message === '退药数量超过可退数量', 'return exceeds copy')

const returned = mockReturnDispense(token, {
  dispenseId: DEMO_IDS.doneDispense,
  reason: '患者要求',
  items: [{ dispenseItemId: DEMO_IDS.doneItem, returnQty: 1 }],
})
assert(returned.chargeId === 'chg-done' && returned.stockNote === '库存已回加 1', 'return rolls stock')
assert(returned.lines[0].lineDispenseStatus === 'PARTIAL' && returned.lines[0].availableQty === 1, 'return reopens qty')
assert(returned.lines[0].stockQty === 13, 'stock restored')
const after = mockListDispenses(token, { today: true, dispenseNo: 'FY-DONE-001' })
assert(after.items[0].status === 'PARTIAL_RETURNED', 'slip header after partial return')
const pendingAgain = mockSearchPending(token, '孙已发')
assert(pendingAgain.items.length === 1 && pendingAgain.items[0].lineDispenseStatus === 'PARTIAL', 'returned line is pending')

const nativeFetch = globalThis.fetch
globalThis.fetch = async () => {
  throw new Error('mock dispense must not call fetch')
}
await expectError(() => searchPending(token, '陈未收'), 'NOT_CHARGED', 409)
globalThis.fetch = nativeFetch

const CH_ZHAO = '11111111-1111-4111-8111-111111111111'
const calls = []
const stock = { 'DRUG-A': 20, 'DRUG-SHORT': 1, 'DRUG-DONE': 12 }
const lines = {
  'ci-zhao': {
    chargeItemId: 'ci-zhao',
    orderItemId: 'oi-zhao',
    orderId: 'ord-zhao',
    chargeId: CH_ZHAO,
    registerId: 'reg-zhao',
    patientId: 'pat-zhao',
    itemNo: 'DRUG-A',
    itemName: '阿莫西林胶囊',
    units: '盒',
    charged: 2,
    net: 0,
  },
  'ci-short': {
    chargeItemId: 'ci-short',
    orderItemId: 'oi-short',
    orderId: 'ord-short',
    chargeId: 'ch-short',
    registerId: 'reg-short',
    patientId: 'pat-short',
    itemNo: 'DRUG-SHORT',
    itemName: '头孢克肟分散片',
    units: '盒',
    charged: 8,
    net: 0,
  },
  'ci-done': {
    chargeItemId: 'ci-done',
    orderItemId: 'oi-done',
    orderId: 'ord-done',
    chargeId: 'ch-done',
    registerId: 'reg-done',
    patientId: 'pat-done',
    itemNo: 'DRUG-DONE',
    itemName: '维生素C片',
    units: '瓶',
    charged: 4,
    net: 4,
  },
}
const slips = [{
  dispenseId: 'dsp-done',
  dispenseNo: 'FY-DONE-001',
  chargeItemId: 'ci-done',
  qty: 4,
  returned: 0,
  dispensedAt: new Date().toISOString(),
  returnReason: '',
}]
const idem = new Map()

function lineStatus(line) {
  if (!(line.net > 0)) return 'NONE'
  if (line.net + 1e-9 < line.charged) return 'PARTIAL'
  return 'FULL'
}

function headerStatus(slip) {
  if (slip.returned > 0 && slip.returned + 1e-9 >= slip.qty) return 'RETURNED'
  if (slip.returned > 0) return 'PARTIAL_RETURNED'
  return 'DONE'
}

function slipDto(slip) {
  const line = lines[slip.chargeItemId]
  const on = stock[line.itemNo]
  const avail = Math.max(0, line.charged - line.net)
  return {
    dispenseId: slip.dispenseId,
    dispenseNo: slip.dispenseNo,
    orderId: line.orderId,
    chargeId: line.chargeId,
    registerId: line.registerId,
    patientId: line.patientId,
    status: headerStatus(slip),
    orderStatus: 'PROC',
    returnReason: slip.returnReason || null,
    operatorId: 'u-system',
    dispensedAt: slip.dispensedAt,
    items: [{
      dispenseItemId: `${slip.dispenseId}-1`,
      orderItemId: line.orderItemId,
      chargeItemId: line.chargeItemId,
      itemNo: line.itemNo,
      itemName: line.itemName,
      units: line.units,
      chargedQty: line.charged,
      dispensedQty: slip.qty,
      returnedQty: slip.returned,
      availableQty: avail,
      dispenseQty: slip.qty,
      stockHint: on > 0 ? '库存充足' : '库存不足',
      stockQty: on,
      lineDispenseStatus: lineStatus(line),
    }],
  }
}

function orderDto(line) {
  const drug = line.itemNo === 'DRUG-A'
  return {
    orderId: line.orderId,
    orderNo: drug ? 'RX-ZHAO-001' : `RX-${line.orderId}`,
    registerId: line.registerId,
    patientId: line.patientId,
    departmentId: 'dept-1',
    doctorId: 'doc-1',
    status: line === lines['ci-done'] ? 'PROC' : 'COMMIT',
    totalFee: 20,
    charged: true,
    items: [
      {
        orderItemId: line.orderItemId,
        lineNo: 1,
        itemNo: line.itemNo,
        itemType: 'DRUG',
        itemName: line.itemName,
        spec: drug ? '0.25g×24' : '',
        units: line.units,
        amount: line.charged,
        price: 10,
        totalFee: 20,
      },
      ...(drug ? [{
        orderItemId: 'oi-treat',
        lineNo: 2,
        itemNo: 'TREAT-1',
        itemType: 'TREAT',
        itemName: '诊查费',
        units: '次',
        amount: 1,
        price: 15,
        totalFee: 15,
      }] : []),
    ],
  }
}

function chargeDto(line, paid) {
  return {
    chargeId: line.chargeId,
    squareNo: line === lines['ci-zhao'] ? 'SQ-ZHAO-001' : `SQ-${line.chargeId}`,
    orderId: line.orderId,
    registerId: line.registerId,
    patientId: line.patientId,
    totalFee: 20,
    squareFee: 20,
    favourableFee: 0,
    status: 'NORMAL',
    paidAt: paid ? '2026-10-12T01:20:00Z' : null,
    items: [
      {
        chargeItemId: line.chargeItemId,
        orderItemId: line.orderItemId,
        lineNo: 1,
        itemNo: line.itemNo,
        itemName: line.itemName,
        qty: line.charged,
        unitPrice: 10,
        totalFee: 20,
        canBack: true,
        refundedQty: 0,
      },
      ...(line === lines['ci-zhao'] ? [{
        chargeItemId: 'ci-treat',
        orderItemId: 'oi-treat',
        lineNo: 2,
        itemNo: 'TREAT-1',
        itemName: '诊查费',
        qty: 1,
        unitPrice: 15,
        totalFee: 15,
        canBack: true,
        refundedQty: 0,
      }] : []),
    ],
  }
}

const patients = {
  'pat-zhao': { patientId: 'pat-zhao', name: '赵待发', patientNo: 'P-ZHAO', hospPkNo: 'P-ZHAO' },
  'pat-unpaid': { patientId: 'pat-unpaid', name: '陈未收', patientNo: 'P-UNPAID', hospPkNo: 'P-UNPAID' },
  'pat-done': { patientId: 'pat-done', name: '孙已发', patientNo: 'P-DONE', hospPkNo: 'P-DONE' },
  'pat-short': { patientId: 'pat-short', name: '赵缺货', patientNo: 'P-SHORT', hospPkNo: 'P-SHORT' },
}
const registers = {
  'reg-zhao': { registerId: 'reg-zhao', registerNo: 'GH-ZHAO-001', patientId: 'pat-zhao', departmentId: 'dept-1', doctorId: 'doc-1', visitDate: '2026-10-12', status: 'DIAGNOSING' },
  'reg-short': { registerId: 'reg-short', registerNo: 'GH-SHORT-001', patientId: 'pat-short', departmentId: 'dept-1', doctorId: 'doc-1', visitDate: '2026-10-12', status: 'DIAGNOSING' },
  'reg-done': { registerId: 'reg-done', registerNo: 'GH-DONE-001', patientId: 'pat-done', departmentId: 'dept-1', doctorId: 'doc-1', visitDate: '2026-10-12', status: 'DIAGNOSING' },
  'reg-unpaid': { registerId: 'reg-unpaid', registerNo: 'GH-UNPAID-001', patientId: 'pat-unpaid', departmentId: 'dept-1', doctorId: 'doc-1', visitDate: '2026-10-12', status: 'DIAGNOSING' },
}
const charges = [
  chargeDto(lines['ci-zhao'], true),
  chargeDto(lines['ci-short'], true),
  chargeDto(lines['ci-done'], true),
  { ...chargeDto({ ...lines['ci-zhao'], chargeItemId: 'ci-unpaid', orderItemId: 'oi-unpaid', orderId: 'ord-unpaid', chargeId: 'ch-unpaid', registerId: 'reg-unpaid', patientId: 'pat-unpaid', itemNo: 'DRUG-A', itemName: '感冒灵颗粒', units: '盒', charged: 1, net: 0 }, false), squareNo: 'SQ-UNPAID-001' },
]

function chargeMatches(charge, url) {
  const patientId = url.searchParams.get('patientId') || ''
  const registerId = url.searchParams.get('registerId') || ''
  if (patientId && charge.patientId !== patientId) return false
  if (registerId && charge.registerId !== registerId) return false
  if (url.searchParams.get('visitDate') && !charge.paidAt && charge.patientId === 'pat-unpaid') return true
  return true
}

function send(res, status, body) {
  res.writeHead(status, { 'Content-Type': 'application/json; charset=utf-8' })
  res.end(JSON.stringify(body))
}

function fail(res, status, code, message) {
  send(res, status, { success: false, error: { code, message, details: {} }, requestId: 'req-dispense' })
}

const server = http.createServer(async (req, res) => {
  const url = new URL(req.url || '/', 'http://127.0.0.1')
  const chunks = []
  for await (const chunk of req) chunks.push(chunk)
  const raw = Buffer.concat(chunks).toString()
  calls.push({
    method: req.method,
    path: url.pathname,
    query: url.search,
    body: raw,
    idempotency: req.headers['idempotency-key'] || '',
    authorization: req.headers.authorization || '',
  })
  if (req.method === 'GET' && url.pathname === '/api/v1/patients') {
    const name = url.searchParams.get('name') || ''
    const hospPkNo = url.searchParams.get('hospPkNo') || ''
    const items = Object.values(patients).filter((item) => {
      if (name && !item.name.includes(name)) return false
      if (hospPkNo && item.hospPkNo !== hospPkNo) return false
      return true
    })
    send(res, 200, { items })
    return
  }
  const patientHit = url.pathname.match(/^\/api\/v1\/patients\/([^/]+)$/)
  if (req.method === 'GET' && patientHit) {
    const patient = patients[decodeURIComponent(patientHit[1])]
    if (!patient) {
      fail(res, 404, 'NOT_FOUND', '资源不存在')
      return
    }
    send(res, 200, patient)
    return
  }
  if (req.method === 'GET' && url.pathname === '/api/v1/registers') {
    const registerNo = url.searchParams.get('registerNo') || ''
    const items = Object.values(registers).filter((item) => !registerNo || item.registerNo === registerNo)
    send(res, 200, { items })
    return
  }
  const registerHit = url.pathname.match(/^\/api\/v1\/registers\/([^/]+)$/)
  if (req.method === 'GET' && registerHit) {
    const register = registers[decodeURIComponent(registerHit[1])]
    if (!register) {
      fail(res, 404, 'NOT_FOUND', '资源不存在')
      return
    }
    send(res, 200, register)
    return
  }
  if (req.method === 'GET' && url.pathname === '/api/v1/orders') {
    const orderNo = url.searchParams.get('orderNo') || ''
    const items = Object.values(lines).filter((line) => orderDto(line).orderNo === orderNo).map((line) => orderDto(line))
    send(res, 200, { items })
    return
  }
  const orderHit = url.pathname.match(/^\/api\/v1\/orders\/([^/]+)$/)
  if (req.method === 'GET' && orderHit) {
    const line = Object.values(lines).find((item) => item.orderId === decodeURIComponent(orderHit[1]))
    if (!line) {
      fail(res, 404, 'NOT_FOUND', '资源不存在')
      return
    }
    send(res, 200, orderDto(line))
    return
  }
  if (req.method === 'GET' && url.pathname === '/api/v1/charges') {
    send(res, 200, { items: charges.filter((item) => chargeMatches(item, url)) })
    return
  }
  const chargeHit = url.pathname.match(/^\/api\/v1\/charges\/([^/]+)$/)
  if (req.method === 'GET' && chargeHit) {
    const charge = charges.find((item) => item.chargeId === decodeURIComponent(chargeHit[1]))
    if (!charge) {
      fail(res, 404, 'NOT_FOUND', '资源不存在')
      return
    }
    send(res, 200, charge)
    return
  }
  if (req.method === 'GET' && url.pathname === '/api/v1/inventory/items') {
    send(res, 200, {
      items: Object.entries(stock).map(([itemNo, qty]) => ({
        skuId: `sku-${itemNo}`,
        skuCode: itemNo,
        itemNo,
        name: itemNo,
        qtyOnHand: qty,
        units: '盒',
        status: 'ACTIVE',
      })),
    })
    return
  }
  if (req.method === 'GET' && url.pathname === '/api/v1/dispenses') {
    const chargeId = url.searchParams.get('chargeId') || ''
    const patientId = url.searchParams.get('patientId') || ''
    const items = slips.filter((slip) => {
      const line = lines[slip.chargeItemId]
      if (chargeId && line.chargeId !== chargeId) return false
      if (patientId && line.patientId !== patientId) return false
      return true
    }).map((slip) => slipDto(slip))
    send(res, 200, { items })
    return
  }
  const dispenseHit = url.pathname.match(/^\/api\/v1\/dispenses\/([^/]+)$/)
  if (req.method === 'GET' && dispenseHit) {
    const slip = slips.find((item) => item.dispenseId === decodeURIComponent(dispenseHit[1]))
    if (!slip) {
      fail(res, 404, 'NOT_FOUND', '资源不存在')
      return
    }
    send(res, 200, slipDto(slip))
    return
  }
  if (req.method === 'POST' && url.pathname === '/api/v1/dispenses') {
    const body = raw ? JSON.parse(raw) : {}
    const key = req.headers['idempotency-key'] || ''
    const hash = JSON.stringify(body)
    if (key && idem.has(key)) {
      const prev = idem.get(key)
      if (prev.hash !== hash) {
        fail(res, 409, 'IDEMPOTENCY_KEY_CONFLICT', '幂等键与已有请求不一致')
        return
      }
      send(res, prev.status, prev.body)
      return
    }
    const requested = Array.isArray(body.items) ? body.items : []
    if (requested.length === 0) {
      fail(res, 409, 'EMPTY_DISPENSE_ITEMS', '没有可发明细')
      return
    }
    const plan = []
    for (const item of requested) {
      const line = Object.values(lines).find((row) => row.chargeItemId === item.chargeItemId || row.orderItemId === item.orderItemId)
      if (!line) {
        fail(res, 400, 'VALIDATION_ERROR', '字段校验失败')
        return
      }
      const avail = line.charged - line.net
      if (!(avail > 0)) {
        fail(res, 409, 'ALREADY_DISPENSED', '已全部发完，不能再发')
        return
      }
      if (Number(item.dispenseQty) > avail) {
        fail(res, 409, 'DISPENSE_EXCEEDS_AVAILABLE', '发药数量超过可发数量')
        return
      }
      if (Number(item.dispenseQty) > stock[line.itemNo]) {
        fail(res, 409, 'INSUFFICIENT_STOCK', '库存不足')
        return
      }
      plan.push({ line, qty: Number(item.dispenseQty) })
    }
    const created = []
    for (const step of plan) {
      step.line.net += step.qty
      stock[step.line.itemNo] -= step.qty
      const slip = {
        dispenseId: `dsp-${slips.length + 1}`,
        dispenseNo: `FY-20261012-${String(slips.length + 1).padStart(4, '0')}`,
        chargeItemId: step.line.chargeItemId,
        qty: step.qty,
        returned: 0,
        dispensedAt: new Date().toISOString(),
        returnReason: '',
      }
      slips.unshift(slip)
      created.push(slipDto(slip))
    }
    const dto = created[0]
    if (key) idem.set(key, { hash, status: 201, body: dto })
    send(res, 201, dto)
    return
  }
  const returnHit = url.pathname.match(/^\/api\/v1\/dispenses\/([^/]+)\/return$/)
  if (req.method === 'POST' && returnHit) {
    const body = raw ? JSON.parse(raw) : {}
    const slip = slips.find((item) => item.dispenseId === decodeURIComponent(returnHit[1]))
    if (!slip) {
      fail(res, 409, 'NOT_DISPENSED', '尚未发药，不能退药')
      return
    }
    if (!String(body.reason || '').trim()) {
      fail(res, 400, 'RETURN_REASON_REQUIRED', '请填写退药原因')
      return
    }
    const item = (body.items || [])[0]
    const returnable = slip.qty - slip.returned
    if (!item || Number(item.returnQty) > returnable) {
      fail(res, 409, 'RETURN_EXCEEDS_DISPENSE', '退药数量超过可退数量')
      return
    }
    const qty = Number(item.returnQty)
    slip.returned += qty
    slip.returnReason = body.reason
    const line = lines[slip.chargeItemId]
    line.net -= qty
    stock[line.itemNo] += qty
    send(res, 200, slipDto(slip))
    return
  }
  fail(res, 404, 'NOT_FOUND', '资源不存在')
})

await new Promise((resolve) => server.listen(0, '127.0.0.1', resolve))
const { port } = server.address()
process.env.VITE_USE_MOCK = 'false'
process.env.VITE_API_BASE_URL = `http://127.0.0.1:${port}/api/v1`

try {
  const before = calls.length
  await expectError(() => confirmDispense(token, { items: [] }), 'EMPTY_DISPENSE_ITEMS', 400)
  assert(calls.length === before, 'empty dispense does not post')
  await expectError(() => returnDispense(token, {
    dispenseId: 'dsp-done',
    items: [{ dispenseItemId: 'dsp-done-1', returnQty: 1 }],
  }), 'RETURN_REASON_REQUIRED', 400)

  const named = await searchPending(token, '赵待发')
  assert(named.items.length === 1 && named.items[0].itemName === '阿莫西林胶囊', 'pending drug only')
  assert(named.items[0].lineDispenseStatus === 'NONE' && named.items[0].availableQty === 2, 'pending qty')
  assert(named.items[0].stockQty === 20 && named.items[0].patientName === '赵待发', 'pending stock and name')
  assert(named.items[0].squareNo === 'SQ-ZHAO-001' && named.items[0].orderNo === 'RX-ZHAO-001', 'pending refs')
  assert(named.items[0].spec === '0.25g×24', 'pending spec')

  const byRegister = await searchPending(token, 'GH-ZHAO-001')
  assert(byRegister.items.length === 1 && byRegister.items[0].chargeItemId === 'ci-zhao', 'search register no')
  const byOrder = await searchPending(token, 'RX-ZHAO-001')
  assert(byOrder.items[0].orderId === 'ord-zhao', 'search order no')
  const bySquare = await searchPending(token, 'SQ-ZHAO-001')
  assert(bySquare.items[0].chargeId === CH_ZHAO, 'search square no')
  const byCharge = await searchPending(token, CH_ZHAO)
  assert(byCharge.items[0].chargeItemId === 'ci-zhao', 'search charge id')

  await expectError(() => searchPending(token, '陈未收'), 'NOT_CHARGED', 409)
  const done = await searchPending(token, '孙已发')
  assert(done.items.length === 0 && done.empty === 'done', 'fully dispensed stays out')

  const today = await listTodayPending(token)
  const todayNames = today.items.map((item) => item.patientName)
  assert(todayNames.includes('赵待发') && todayNames.includes('赵缺货'), 'today pending')
  assert(!todayNames.includes('陈未收') && !todayNames.includes('孙已发'), 'today skips unpaid and finished')

  const exceeds = await expectError(() => confirmDispense(token, {
    items: [{ orderId: 'ord-zhao', chargeId: CH_ZHAO, chargeItemId: 'ci-zhao', orderItemId: 'oi-zhao', dispenseQty: 3, selected: true }],
  }), 'DISPENSE_EXCEEDS_AVAILABLE', 409)
  assert(exceeds.message === '发药数量超过可发数量', 'exceeds copy')
  assert(stock['DRUG-A'] === 20, 'exceeds keeps stock')

  const short = await expectError(() => confirmDispense(token, {
    items: [{ orderId: 'ord-short', chargeId: 'ch-short', chargeItemId: 'ci-short', orderItemId: 'oi-short', dispenseQty: 8, selected: true }],
  }), 'INSUFFICIENT_STOCK', 409)
  assert(short.message === '库存不足，无法发药', 'real stock copy')
  assert(stock['DRUG-SHORT'] === 1, 'short keeps stock')

  const idemKey = '11111111-1111-4111-8111-111111111199'
  const partial = await confirmDispense(token, {
    idempotencyKey: idemKey,
    items: [{ orderId: 'ord-zhao', chargeId: CH_ZHAO, chargeItemId: 'ci-zhao', orderItemId: 'oi-zhao', dispenseQty: 1, selected: true }],
  })
  assert(partial.lines === undefined || partial.dispenseNo, 'partial returns a slip')
  assert(partial.lineDispenseStatus === 'PARTIAL' && stock['DRUG-A'] === 19, 'partial status and stock')
  const postCall = calls.find((item) => item.method === 'POST' && item.path === '/api/v1/dispenses' && item.idempotency === idemKey)
  assert(postCall && postCall.idempotency === idemKey, 'idempotency header')
  assert(postCall.authorization.startsWith('Bearer '), 'bearer')
  const posted = JSON.parse(postCall.body)
  assert(posted.orderId === 'ord-zhao' && posted.chargeId === CH_ZHAO && posted.items[0].dispenseQty === 1, 'create body')
  assert(!calls.some((item) => item.path.includes('/orders/') && item.path.endsWith('/dispense')), 'alias not called')

  const conflict = await expectError(() => confirmDispense(token, {
    idempotencyKey: idemKey,
    items: [{ orderId: 'ord-zhao', chargeId: CH_ZHAO, chargeItemId: 'ci-zhao', orderItemId: 'oi-zhao', dispenseQty: 1.5, selected: true }],
  }), 'IDEMPOTENCY_KEY_CONFLICT', 409)
  assert(conflict.message === '重复提交已处理，未再次扣减/回滚', 'conflict copy on the wire')
  assert(stock['DRUG-A'] === 19, 'conflict does not decrement again')

  const rest = await searchPending(token, '赵待发')
  assert(rest.items[0].lineDispenseStatus === 'PARTIAL' && rest.items[0].availableQty === 1, 'remaining pending')
  const full = await confirmDispense(token, {
    items: [{ orderId: 'ord-zhao', chargeId: CH_ZHAO, chargeItemId: 'ci-zhao', orderItemId: 'oi-zhao', dispenseQty: 1, selected: true }],
  })
  assert(full.lineDispenseStatus === 'FULL' && stock['DRUG-A'] === 18, 'second dispense fills')
  const afterFull = await searchPending(token, '赵待发')
  assert(afterFull.items.length === 0 && afterFull.empty === 'done', 'full leaves pending')
  const again = await expectError(() => confirmDispense(token, {
    items: [{ orderId: 'ord-zhao', chargeId: CH_ZHAO, chargeItemId: 'ci-zhao', dispenseQty: 1, selected: true }],
  }), 'ALREADY_DISPENSED', 409)
  assert(again.message === '已全部发完，不能再发', 'already full from API')

  const issued = await listDispenses(token, { today: true, patient: '赵待发' })
  assert(issued.items.length >= 1 && issued.items[0].patientName === '赵待发', 'issued query')
  assert(issued.items.every((item) => item.patientName === '赵待发'), 'issued patient filter')
  assert(calls.some((item) => item.method === 'GET' && item.path === '/api/v1/dispenses' && item.query.includes('visitDate=')), 'issued uses visitDate')
  const detail = await fetchDispense(token, partial.dispenseId)
  assert(detail.items[0].returnableQty === 1 && detail.chargeId === CH_ZHAO, 'detail returnable')

  await expectError(() => returnDispense(token, {
    dispenseId: partial.dispenseId,
    reason: '患者要求',
    items: [{ dispenseItemId: detail.items[0].dispenseItemId, returnQty: 5, selected: true }],
  }), 'RETURN_EXCEEDS_DISPENSE', 409)
  const returned = await returnDispense(token, {
    dispenseId: partial.dispenseId,
    reason: '患者要求',
    idempotencyKey: '22222222-2222-4222-8222-222222222222',
    items: [{ dispenseItemId: detail.items[0].dispenseItemId, returnQty: 1, selected: true }],
  })
  assert(returned.stockNote === '库存已回加 1' && returned.chargeId === CH_ZHAO, 'return stock note and charge')
  assert(stock['DRUG-A'] === 19, 'return restores stock')
  const reopened = await searchPending(token, '赵待发')
  assert(reopened.items.length === 1 && reopened.items[0].lineDispenseStatus === 'PARTIAL', 'return reopens pending')
  assert(calls.some((item) => item.method === 'POST' && item.path === `/api/v1/dispenses/${partial.dispenseId}/return`), 'return path')
} finally {
  await new Promise((resolve, reject) => server.close((error) => (error ? reject(error) : resolve())))
}

console.log('dispense checks passed')
