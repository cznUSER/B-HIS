// 2026-10-12 P1-H05-CHARGE-UI 收费 Mock。
// 演示数据：已提交处方 + 库存提示。不请求 /charges。规则对齐拍板 #22。

import { apiError } from './errors.js'
import { mockMe } from './mockAuth.js'
import { PAY_CASH, formatOperate } from './chargeContract.js'
import { todayInShanghai } from './patientContract.js'

const STORAGE_KEY = 'bhis.charges.v1'

let memory = null

function id(suffix) {
  return `00000000-0000-4000-8000-${suffix}`
}

function requireSession(accessToken) {
  mockMe(accessToken)
}

function nowInShanghai() {
  const parts = new Intl.DateTimeFormat('en-GB', {
    timeZone: 'Asia/Shanghai',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hourCycle: 'h23',
  }).formatToParts(new Date())
  const pick = (type) => {
    const found = parts.find((part) => part.type === type)
    return found ? found.value : '00'
  }
  return `${todayInShanghai()}T${pick('hour')}:${pick('minute')}:${pick('second')}+08:00`
}

function storage() {
  try {
    return typeof sessionStorage === 'undefined' ? null : sessionStorage
  } catch {
    return null
  }
}

function line(itemNo, itemName, qty, unitPriceCents, stockAvailable, extra = {}) {
  return {
    itemNo,
    itemName,
    qty,
    unit: extra.unit || '盒',
    unitPriceCents,
    lineFeeCents: qty * unitPriceCents,
    stockAvailable,
    canBack: extra.canBack !== false,
    dispensedQty: extra.dispensedQty || 0,
    returnedQty: extra.returnedQty || 0,
  }
}

function seed() {
  const day = todayInShanghai()
  const at = (hour, minute) => `${day}T${String(hour).padStart(2, '0')}:${String(minute).padStart(2, '0')}:00+08:00`
  const subjects = [
    {
      patientId: id('000000000c01'),
      patientNo: 'P2026101201',
      patientName: '李无号',
      register: null,
      order: null,
    },
    {
      patientId: id('000000000c02'),
      patientNo: 'P2026101202',
      patientName: '王未开',
      register: {
        registerId: id('000000000r02'),
        encounterId: id('000000000r02'),
        registerNo: 'GH20261012002',
        status: 'NORMAL',
        summary: 'GH20261012002 · 内科 · 周医生',
      },
      order: {
        orderId: id('000000000o02'),
        recipeNo: 'CF20261012002',
        status: 'SAVE',
        charged: false,
        items: [line('AMX', '阿莫西林胶囊', 1, 1250, 20)],
      },
    },
    {
      patientId: id('000000000c03'),
      patientNo: 'P2026101203',
      patientName: '赵待收',
      register: {
        registerId: id('000000000r03'),
        encounterId: id('000000000r03'),
        registerNo: 'GH20261012003',
        status: 'NORMAL',
        summary: 'GH20261012003 · 内科 · 李医生',
      },
      order: {
        orderId: id('000000000o03'),
        recipeNo: 'CF20261012003',
        status: 'COMMIT',
        charged: false,
        items: [
          line('AMX', '阿莫西林胶囊', 2, 1250, 40),
          line('IBU', '布洛芬缓释胶囊', 1, 1800, 12),
        ],
      },
    },
    {
      patientId: id('000000000c04'),
      patientNo: 'P2026101204',
      patientName: '钱缺货',
      register: {
        registerId: id('000000000r04'),
        encounterId: id('000000000r04'),
        registerNo: 'GH20261012004',
        status: 'NORMAL',
        summary: 'GH20261012004 · 内科 · 李医生',
      },
      order: {
        orderId: id('000000000o04'),
        recipeNo: 'CF20261012004',
        status: 'COMMIT',
        charged: false,
        items: [
          line('CEF', '头孢克肟分散片', 3, 2800, 1),
          line('VITC', '维生素C片', 1, 650, 20, { unit: '瓶' }),
        ],
      },
    },
    {
      patientId: id('000000000c05'),
      patientNo: 'P2026101205',
      patientName: '孙已收',
      register: {
        registerId: id('000000000r05'),
        encounterId: id('000000000r05'),
        registerNo: 'GH20261012005',
        status: 'NORMAL',
        summary: 'GH20261012005 · 中医科 · 赵医生',
      },
      order: {
        orderId: id('000000000o05'),
        recipeNo: 'CF20261012005',
        status: 'COMMIT',
        charged: true,
        items: [
          line('GML', '感冒灵颗粒', 2, 1500, 30),
          line('BLG', '板蓝根颗粒', 1, 950, 16),
        ],
      },
    },
    {
      patientId: id('000000000c06'),
      patientNo: 'P2026101206',
      patientName: '周已发',
      register: {
        registerId: id('000000000r06'),
        encounterId: id('000000000r06'),
        registerNo: 'GH20261012006',
        status: 'NORMAL',
        summary: 'GH20261012006 · 内科 · 李医生',
      },
      order: {
        orderId: id('000000000o06'),
        recipeNo: 'CF20261012006',
        status: 'COMMIT',
        charged: true,
        items: [
          line('LH', '连花清瘟胶囊', 2, 2200, 8, { dispensedQty: 2 }),
          line('INJ', '注射费', 1, 500, 99, { unit: '次', canBack: false }),
        ],
      },
    },
  ]

  const sun = subjects[4]
  const zhou = subjects[5]
  const charges = [
    buildCharge(sun, at(9, 15), 'SF20261012005', id('00000000ch05'), ['00000000ci05a', '00000000ci05b']),
    buildCharge(zhou, at(9, 40), 'SF20261012006', id('00000000ch06'), ['00000000ci06a', '00000000ci06b']),
  ]
  const payments = charges.map((charge, index) => ({
    paymentId: id(`00000000pm0${index + 5}`),
    paymentNo: `PM2026101200${index + 5}`,
    chargeId: charge.chargeId,
    direction: 'CHARGE',
    payType: PAY_CASH,
    amountCents: charge.squareFeeCents,
    status: 'SUCCEEDED',
    succeededAt: charge.operateAt,
  }))
  return { seq: 7, subjects, charges, payments }
}

function buildCharge(subject, operateAt, squareNo, chargeId, itemSuffixes) {
  const items = subject.order.items.map((item, index) => ({
    chargeItemId: id(itemSuffixes[index]),
    orderItemId: id(`00000000oi${squareNo.slice(-2)}${index}`),
    ...copyItem(item),
    refundedQty: 0,
  }))
  const total = items.reduce((sum, item) => sum + item.lineFeeCents, 0)
  return {
    chargeId,
    squareNo,
    orderId: subject.order.orderId,
    encounterId: subject.register.encounterId,
    registerId: subject.register.registerId,
    registerSummary: subject.register.summary,
    patientId: subject.patientId,
    patientNo: subject.patientNo,
    patientName: subject.patientName,
    recipeNo: subject.order.recipeNo,
    totalFeeCents: total,
    squareFeeCents: total,
    favourableFeeCents: 0,
    status: 'NORMAL',
    successorSquareId: null,
    abatedFromId: null,
    operateAt,
    items,
  }
}

function copyItem(item) {
  return {
    itemNo: item.itemNo,
    itemName: item.itemName,
    qty: item.qty,
    unit: item.unit,
    unitPriceCents: item.unitPriceCents,
    lineFeeCents: item.qty * item.unitPriceCents,
    stockAvailable: item.stockAvailable,
    canBack: item.canBack !== false,
    dispensedQty: item.dispensedQty || 0,
    returnedQty: item.returnedQty || 0,
  }
}

function readState() {
  const store = storage()
  if (!store) {
    if (!memory) memory = seed()
    return memory
  }
  try {
    const raw = store.getItem(STORAGE_KEY)
    if (!raw) {
      const seeded = seed()
      store.setItem(STORAGE_KEY, JSON.stringify(seeded))
      return seeded
    }
    const parsed = JSON.parse(raw)
    if (!parsed || !Array.isArray(parsed.subjects) || !Array.isArray(parsed.charges)) return seed()
    return parsed
  } catch {
    return seed()
  }
}

function writeState(state) {
  const store = storage()
  if (!store) {
    memory = state
    return
  }
  store.setItem(STORAGE_KEY, JSON.stringify(state))
}

export function resetMockCharges() {
  memory = null
  const store = storage()
  if (store) store.removeItem(STORAGE_KEY)
}

function findSubject(state, patientId) {
  return state.subjects.find((item) => item.patientId === patientId) || null
}

function sceneOf(subject) {
  if (!subject.register || subject.register.status !== 'NORMAL') return 'NO_REGISTER'
  if (!subject.order || subject.order.status !== 'COMMIT') return 'NO_COMMIT'
  if (subject.order.charged) {
    const blocked = subject.order.items.some((item) => (item.dispensedQty || 0) > (item.returnedQty || 0))
    return blocked ? 'DISPENSED' : 'CHARGED'
  }
  const short = subject.order.items.some((item) => item.qty > item.stockAvailable)
  return short ? 'STOCK' : 'PENDING'
}

const SCENE_LABEL = {
  NO_REGISTER: '请先挂号',
  NO_COMMIT: '请先提交处方',
  PENDING: '待收',
  STOCK: '库存不足',
  CHARGED: '已收费',
  DISPENSED: '已发药',
}

function stockHint(item) {
  if (item.qty > item.stockAvailable) {
    return `库存不足（可用 ${item.stockAvailable}，需要 ${item.qty}）`
  }
  return `可用 ${item.stockAvailable}`
}

function presentPreviewItem(item, index) {
  return {
    lineNo: index + 1,
    itemNo: item.itemNo,
    itemName: item.itemName,
    qty: item.qty,
    unit: item.unit,
    unitPriceCents: item.unitPriceCents,
    lineFeeCents: item.qty * item.unitPriceCents,
    stockAvailable: item.stockAvailable,
    stockSufficient: item.qty <= item.stockAvailable,
    stockHint: stockHint(item),
  }
}

export function mockLookup(accessToken, keyword) {
  requireSession(accessToken)
  const state = readState()
  const query = typeof keyword === 'string' ? keyword.trim() : ''
  const items = state.subjects
    .filter((subject) => {
      if (!query) return true
      const scene = sceneOf(subject)
      const hay = [subject.patientName, subject.patientNo, subject.register && subject.register.registerNo, subject.order && subject.order.recipeNo, SCENE_LABEL[scene]]
        .filter(Boolean)
        .join(' ')
      return hay.includes(query)
    })
    .map((subject) => {
      const scene = sceneOf(subject)
      return {
        patientId: subject.patientId,
        patientNo: subject.patientNo,
        patientName: subject.patientName,
        registerNo: subject.register ? subject.register.registerNo : '',
        recipeNo: subject.order && subject.order.status === 'COMMIT' ? subject.order.recipeNo : '',
        scene,
        sceneLabel: SCENE_LABEL[scene],
      }
    })
  return { items }
}

export function mockLoadPending(accessToken, patientId) {
  requireSession(accessToken)
  const state = readState()
  const subject = findSubject(state, patientId)
  if (!subject) throw apiError(404, 'NOT_FOUND', '未找到患者')
  const scene = sceneOf(subject)
  const patient = {
    patientId: subject.patientId,
    patientNo: subject.patientNo,
    patientName: subject.patientName,
  }
  const register = subject.register
    ? {
        registerId: subject.register.registerId,
        encounterId: subject.register.encounterId,
        registerNo: subject.register.registerNo,
        summary: subject.register.summary,
        status: subject.register.status,
      }
    : null
  if (scene === 'NO_REGISTER' || scene === 'NO_COMMIT') {
    return { scene, patient, register, order: null, preview: null }
  }
  const order = {
    orderId: subject.order.orderId,
    recipeNo: subject.order.recipeNo,
    status: subject.order.status,
  }
  if (scene === 'CHARGED' || scene === 'DISPENSED') {
    const charge = state.charges.find((item) => item.orderId === subject.order.orderId && item.status === 'NORMAL')
    return {
      scene,
      patient,
      register,
      order,
      preview: null,
      chargeId: charge ? charge.chargeId : '',
      squareNo: charge ? charge.squareNo : '',
    }
  }
  const items = subject.order.items.map(presentPreviewItem)
  const total = items.reduce((sum, item) => sum + item.lineFeeCents, 0)
  return {
    scene,
    patient,
    register,
    order,
    preview: {
      orderId: subject.order.orderId,
      items,
      totalFeeCents: total,
      favourableFeeCents: 0,
      squareFeeCents: total,
      squareFee: total / 100,
      amountDueCents: total,
      amountPaidCents: total,
      payType: PAY_CASH,
      stockBlocked: items.some((item) => !item.stockSufficient),
    },
  }
}

function nextSquareNo(state) {
  const seq = state.seq
  state.seq += 1
  const day = todayInShanghai().replace(/-/g, '')
  return `SF${day}${String(seq).padStart(3, '0')}`
}

export function mockPay(accessToken, input) {
  requireSession(accessToken)
  const orderId = input && input.orderId
  const payType = input && input.payType
  const amountCents = input && input.amountCents
  if (payType !== PAY_CASH) {
    throw apiError(400, 'VALIDATION_ERROR', '电子支付为预留，不能收款')
  }
  const state = readState()
  const subject = state.subjects.find((item) => item.order && item.order.orderId === orderId)
  if (!subject) throw apiError(404, 'NOT_FOUND', '未找到待收处方')
  const scene = sceneOf(subject)
  if (scene === 'NO_REGISTER') throw apiError(409, 'ENCOUNTER_REQUIRED', '未挂号/无有效就诊，不能收费')
  if (scene === 'NO_COMMIT') throw apiError(409, 'VALIDATION_ERROR', '请先提交处方')
  if (scene === 'CHARGED' || scene === 'DISPENSED') {
    const existing = state.charges.find((item) => item.orderId === orderId && item.status === 'NORMAL')
    throw apiError(409, 'ALREADY_CHARGED', '该处方已收费', {
      details: { chargeId: existing ? existing.chargeId : '' },
    })
  }
  const items = subject.order.items.map(presentPreviewItem)
  if (items.some((item) => item.qty <= 0)) throw apiError(400, 'INVALID_AMOUNT', '数量或金额不合法')
  if (items.some((item) => !item.stockSufficient)) {
    throw apiError(409, 'INSUFFICIENT_STOCK', '库存不足，无法收费')
  }
  const total = items.reduce((sum, item) => sum + item.lineFeeCents, 0)
  if (!Number.isInteger(amountCents) || amountCents !== total) {
    throw apiError(400, 'CHARGE_AMOUNT_MISMATCH', '实收与应收不一致')
  }
  const operateAt = nowInShanghai()
  const squareNo = nextSquareNo(state)
  const charge = {
    chargeId: crypto.randomUUID(),
    squareNo,
    orderId: subject.order.orderId,
    encounterId: subject.register.encounterId,
    registerId: subject.register.registerId,
    registerSummary: subject.register.summary,
    patientId: subject.patientId,
    patientNo: subject.patientNo,
    patientName: subject.patientName,
    recipeNo: subject.order.recipeNo,
    totalFeeCents: total,
    squareFeeCents: total,
    favourableFeeCents: 0,
    status: 'NORMAL',
    successorSquareId: null,
    abatedFromId: null,
    operateAt,
    items: items.map((item) => ({
      chargeItemId: crypto.randomUUID(),
      orderItemId: crypto.randomUUID(),
      itemNo: item.itemNo,
      itemName: item.itemName,
      qty: item.qty,
      unit: item.unit,
      unitPriceCents: item.unitPriceCents,
      lineFeeCents: item.lineFeeCents,
      stockAvailable: item.stockAvailable,
      canBack: true,
      dispensedQty: 0,
      returnedQty: 0,
      refundedQty: 0,
    })),
  }
  const payment = {
    paymentId: crypto.randomUUID(),
    paymentNo: `PM${squareNo.slice(2)}`,
    chargeId: charge.chargeId,
    direction: 'CHARGE',
    payType: PAY_CASH,
    amountCents: total,
    status: 'SUCCEEDED',
    succeededAt: charge.operateAt,
  }
  subject.order.charged = true
  state.charges.push(charge)
  state.payments.push(payment)
  writeState(state)
  return { charge: presentCharge(charge), payment: presentPayment(payment) }
}

function presentPayment(payment) {
  return { ...payment }
}

function presentCharge(charge) {
  return {
    chargeId: charge.chargeId,
    squareNo: charge.squareNo,
    orderId: charge.orderId,
    encounterId: charge.encounterId,
    registerId: charge.registerId,
    registerSummary: charge.registerSummary,
    patientId: charge.patientId,
    patientNo: charge.patientNo,
    patientName: charge.patientName,
    recipeNo: charge.recipeNo,
    totalFeeCents: charge.totalFeeCents,
    squareFeeCents: charge.squareFeeCents,
    favourableFeeCents: charge.favourableFeeCents,
    status: charge.status,
    successorSquareId: charge.successorSquareId,
    abatedFromId: charge.abatedFromId,
    operateAt: charge.operateAt,
    operateText: formatOperate(charge.operateAt),
    items: charge.items.map(presentRefundItem),
  }
}

function presentRefundItem(item) {
  const refundableQty = item.qty - (item.refundedQty || 0)
  const outstanding = (item.dispensedQty || 0) - (item.returnedQty || 0)
  return {
    chargeItemId: item.chargeItemId,
    itemNo: item.itemNo,
    itemName: item.itemName,
    qty: item.qty,
    unit: item.unit,
    unitPriceCents: item.unitPriceCents,
    lineFeeCents: item.lineFeeCents,
    canBack: item.canBack !== false,
    refundedQty: item.refundedQty || 0,
    refundableQty,
    dispensedQty: item.dispensedQty || 0,
    returnedQty: item.returnedQty || 0,
    dispenseBlocked: outstanding > 0,
  }
}

export function mockListCharges(accessToken, input = {}) {
  requireSession(accessToken)
  const state = readState()
  const keyword = typeof input.keyword === 'string' ? input.keyword.trim() : ''
  const todayOnly = input.todayOnly !== false
  const day = todayInShanghai()
  const items = state.charges
    .filter((charge) => {
      if (charge.status === 'NORMAL' && charge.squareFeeCents === 0) return false
      if (todayOnly && !String(charge.operateAt).startsWith(day)) return false
      if (!keyword) return true
      return charge.patientName.includes(keyword) || charge.patientNo.includes(keyword)
    })
    .slice()
    .sort((a, b) => String(b.operateAt).localeCompare(String(a.operateAt)))
    .map((charge) => ({
      chargeId: charge.chargeId,
      squareNo: charge.squareNo,
      operateAt: charge.operateAt,
      operateText: formatOperate(charge.operateAt),
      patientName: charge.patientName,
      patientNo: charge.patientNo,
      registerSummary: charge.registerSummary,
      squareFeeCents: charge.squareFeeCents,
      favourableFeeCents: charge.favourableFeeCents,
      status: charge.status,
      canRefund: charge.status === 'NORMAL',
    }))
  return { items }
}

export function mockGetCharge(accessToken, chargeId) {
  requireSession(accessToken)
  const state = readState()
  const charge = state.charges.find((item) => item.chargeId === chargeId)
  if (!charge) throw apiError(404, 'NOT_FOUND', '未找到收费单')
  return presentCharge(charge)
}

export function mockRefund(accessToken, chargeId, input) {
  requireSession(accessToken)
  const state = readState()
  const charge = state.charges.find((item) => item.chargeId === chargeId)
  if (!charge) throw apiError(404, 'NOT_FOUND', '未找到收费单')
  if (charge.status === 'ABATE') throw apiError(409, 'ALREADY_ABATED', '该收费单已作废，不能再退')
  if (charge.status !== 'NORMAL') throw apiError(409, 'ALREADY_ABATED', '该收费单已作废，不能再退')
  const payType = input && input.payType
  if (payType !== PAY_CASH) throw apiError(400, 'VALIDATION_ERROR', '电子支付为预留，不能退款')
  const requested = Array.isArray(input && input.items) ? input.items : []
  if (requested.length === 0) throw apiError(400, 'INVALID_AMOUNT', '数量或金额不合法')
  const byId = new Map(charge.items.map((item) => [item.chargeItemId, item]))
  const plan = []
  const seen = new Set()
  for (const row of requested) {
    const item = byId.get(row && row.chargeItemId)
    if (!item) throw apiError(400, 'ITEM_NOT_CAN_BACK', '该明细不可退')
    if (seen.has(item.chargeItemId)) throw apiError(400, 'INVALID_AMOUNT', '数量或金额不合法')
    seen.add(item.chargeItemId)
    const qty = row.refundQty
    if (!Number.isInteger(qty) || qty <= 0) throw apiError(400, 'INVALID_AMOUNT', '数量或金额不合法')
    plan.push({ item, qty })
  }
  if (plan.some((row) => row.item.canBack === false)) {
    throw apiError(409, 'ITEM_NOT_CAN_BACK', '该明细不可退')
  }
  if (plan.some((row) => (row.item.dispensedQty || 0) - (row.item.returnedQty || 0) > 0)) {
    throw apiError(409, 'REFUND_REQUIRES_RETURN', '已发药，请先退药再退费')
  }
  let refundCents = 0
  const remainItems = []
  for (const row of plan) {
    const item = row.item
    const refundable = item.qty - (item.refundedQty || 0)
    if (row.qty > refundable) throw apiError(400, 'REFUND_EXCEEDS_REFUNDABLE', '退数量超过可退数量')
    if (item.unitPriceCents <= 0 || item.lineFeeCents !== item.qty * item.unitPriceCents) {
      throw apiError(400, 'INVALID_AMOUNT', '数量或金额不合法')
    }
    refundCents += row.qty * item.unitPriceCents
  }
  for (const row of plan) {
    const item = row.item
    item.refundedQty = (item.refundedQty || 0) + row.qty
    const remainQty = item.qty - item.refundedQty
    if (remainQty > 0) {
      remainItems.push({
        chargeItemId: crypto.randomUUID(),
        orderItemId: crypto.randomUUID(),
        itemNo: item.itemNo,
        itemName: item.itemName,
        qty: remainQty,
        unit: item.unit,
        unitPriceCents: item.unitPriceCents,
        lineFeeCents: remainQty * item.unitPriceCents,
        stockAvailable: item.stockAvailable,
        canBack: true,
        dispensedQty: 0,
        returnedQty: 0,
        refundedQty: 0,
      })
    }
  }
  for (const item of charge.items) {
    if (seen.has(item.chargeItemId)) continue
    if ((item.refundedQty || 0) >= item.qty) continue
    const remainQty = item.qty - (item.refundedQty || 0)
    remainItems.push({
      chargeItemId: crypto.randomUUID(),
      orderItemId: crypto.randomUUID(),
      itemNo: item.itemNo,
      itemName: item.itemName,
      qty: remainQty,
      unit: item.unit,
      unitPriceCents: item.unitPriceCents,
      lineFeeCents: remainQty * item.unitPriceCents,
      stockAvailable: item.stockAvailable,
      canBack: item.canBack !== false,
      dispensedQty: item.dispensedQty || 0,
      returnedQty: item.returnedQty || 0,
      refundedQty: 0,
    })
  }
  const remainFee = remainItems.reduce((sum, item) => sum + item.lineFeeCents, 0)
  charge.status = 'ABATE'
  let newSquare = null
  if (remainFee > 0) {
    const squareNo = nextSquareNo(state)
    newSquare = {
      chargeId: crypto.randomUUID(),
      squareNo,
      orderId: charge.orderId,
      encounterId: charge.encounterId,
      registerId: charge.registerId,
      registerSummary: charge.registerSummary,
      patientId: charge.patientId,
      patientNo: charge.patientNo,
      patientName: charge.patientName,
      recipeNo: charge.recipeNo,
      totalFeeCents: remainFee,
      squareFeeCents: remainFee,
      favourableFeeCents: 0,
      status: 'NORMAL',
      successorSquareId: null,
      abatedFromId: charge.chargeId,
      operateAt: charge.operateAt,
      items: remainItems,
    }
    charge.successorSquareId = newSquare.chargeId
    state.charges.push(newSquare)
  }
  const reason = typeof input.reason === 'string' && input.reason.trim() ? input.reason.trim() : '患者要求'
  const payment = {
    paymentId: crypto.randomUUID(),
    paymentNo: `RF${Date.now()}`,
    chargeId: charge.chargeId,
    direction: 'REFUND',
    payType: PAY_CASH,
    amountCents: refundCents,
    status: 'SUCCEEDED',
    succeededAt: charge.operateAt,
    reason,
  }
  state.payments.push(payment)
  writeState(state)
  return {
    abatedSquare: presentCharge(charge),
    newSquare: newSquare ? presentCharge(newSquare) : null,
    refundPayment: presentPayment(payment),
    fullRefund: newSquare == null,
  }
}
