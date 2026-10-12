// 2026-10-12 P1-H05-DISPENSE-UI 门诊发药 Mock。
// 仅发药与退药。简易数量账，无批次、采购、调拨、盘点、单位换算。毒麻门禁关闭。

import { apiError } from './errors.js'
import { mockMe } from './mockAuth.js'
import { todayInShanghai } from './patientContract.js'
import {
  PHARMACY,
  formatQty,
  lineDispenseStatus,
  round2,
  stockHint,
} from './dispenseContract.js'

const STORAGE_KEY = 'bhis.dispenses.v1'

export const DEMO_CASES = [
  { query: '王全发', label: '王全发 · 可全发' },
  { query: '李部分', label: '李部分 · 可部分发' },
  { query: '赵缺货', label: '赵缺货 · 库存不足' },
  { query: '陈未收', label: '陈未收 · 未收费' },
  { query: '孙已发', label: '孙已发 · 已发可退' },
]

export const DEMO_IDS = {
  full: 'ci-full',
  part: 'ci-part',
  short: 'ci-short',
  unpaid: 'ci-unpaid',
  done: 'ci-done',
  treat: 'ci-treat',
  doneDispense: 'dsp-done',
  doneItem: 'di-done',
}

const SEED_STOCK = {
  'DRUG-AMOX': 20,
  'DRUG-IBU': 30,
  'DRUG-CEFIX': 2,
  'DRUG-COLD': 10,
  'DRUG-VITC': 12,
}

function seedLines() {
  return [
    line({
      chargeItemId: DEMO_IDS.full,
      itemNo: 'DRUG-AMOX',
      itemName: '阿莫西林胶囊',
      spec: '0.25g×24',
      units: '盒',
      chargedQty: 2,
      patientId: 'p-full',
      patientName: '王全发',
      patientNo: 'P-DISP-001',
      registerId: 'r-full',
      registerNo: 'GH-FULL-001',
      orderId: 'ord-full',
      orderNo: 'RX-FULL-001',
      chargeId: 'chg-full',
      squareNo: 'SQ-FULL-001',
    }),
    line({
      chargeItemId: DEMO_IDS.treat,
      itemNo: 'TREAT-1',
      itemName: '诊查费',
      spec: '',
      units: '次',
      itemType: 'TREAT',
      chargedQty: 1,
      patientId: 'p-full',
      patientName: '王全发',
      patientNo: 'P-DISP-001',
      registerId: 'r-full',
      registerNo: 'GH-FULL-001',
      orderId: 'ord-full',
      orderNo: 'RX-FULL-001',
      chargeId: 'chg-full',
      squareNo: 'SQ-FULL-001',
    }),
    line({
      chargeItemId: DEMO_IDS.part,
      itemNo: 'DRUG-IBU',
      itemName: '布洛芬缓释胶囊',
      spec: '0.3g×20',
      units: '盒',
      chargedQty: 6,
      patientId: 'p-part',
      patientName: '李部分',
      patientNo: 'P-DISP-002',
      registerId: 'r-part',
      registerNo: 'GH-PART-001',
      orderId: 'ord-part',
      orderNo: 'RX-PART-001',
      chargeId: 'chg-part',
      squareNo: 'SQ-PART-001',
    }),
    line({
      chargeItemId: DEMO_IDS.short,
      itemNo: 'DRUG-CEFIX',
      itemName: '头孢克肟分散片',
      spec: '0.1g×6',
      units: '盒',
      chargedQty: 8,
      patientId: 'p-short',
      patientName: '赵缺货',
      patientNo: 'P-DISP-003',
      registerId: 'r-short',
      registerNo: 'GH-SHORT-001',
      orderId: 'ord-short',
      orderNo: 'RX-SHORT-001',
      chargeId: 'chg-short',
      squareNo: 'SQ-SHORT-001',
    }),
    line({
      chargeItemId: DEMO_IDS.unpaid,
      itemNo: 'DRUG-COLD',
      itemName: '感冒灵颗粒',
      spec: '10g×9',
      units: '盒',
      chargedQty: 1,
      chargeStatus: 'UNPAID',
      patientId: 'p-unpaid',
      patientName: '陈未收',
      patientNo: 'P-DISP-004',
      registerId: 'r-unpaid',
      registerNo: 'GH-UNPAID-001',
      orderId: 'ord-unpaid',
      orderNo: 'RX-UNPAID-001',
      chargeId: 'chg-unpaid',
      squareNo: 'SQ-UNPAID-001',
    }),
    line({
      chargeItemId: DEMO_IDS.done,
      itemNo: 'DRUG-VITC',
      itemName: '维生素C片',
      spec: '0.1g×100',
      units: '瓶',
      chargedQty: 4,
      dispensedNet: 4,
      patientId: 'p-done',
      patientName: '孙已发',
      patientNo: 'P-DISP-005',
      registerId: 'r-done',
      registerNo: 'GH-DONE-001',
      orderId: 'ord-done',
      orderNo: 'RX-DONE-001',
      chargeId: 'chg-done',
      squareNo: 'SQ-DONE-001',
    }),
  ]
}

function line(raw) {
  return {
    itemType: 'DRUG',
    chargeStatus: 'NORMAL',
    dispensedNet: 0,
    returnedQty: 0,
    orderItemId: raw.orderItemId || `oi-${raw.chargeItemId}`,
    encounterId: raw.registerId,
    ...raw,
  }
}

function seedDispense() {
  const done = seedLines().find((item) => item.chargeItemId === DEMO_IDS.done)
  return {
    dispenseId: DEMO_IDS.doneDispense,
    dispenseNo: 'FY-DONE-001',
    operateAt: new Date().toISOString(),
    operatorId: 'u-system',
    operatorName: '系统管理员',
    deptId: PHARMACY.deptId,
    deptName: PHARMACY.deptName,
    patientId: done.patientId,
    patientName: done.patientName,
    patientNo: done.patientNo,
    registerId: done.registerId,
    registerNo: done.registerNo,
    chargeId: done.chargeId,
    squareNo: done.squareNo,
    orderId: done.orderId,
    orderNo: done.orderNo,
    items: [
      {
        dispenseItemId: DEMO_IDS.doneItem,
        chargeItemId: done.chargeItemId,
        orderItemId: done.orderItemId,
        itemNo: done.itemNo,
        itemName: done.itemName,
        spec: done.spec,
        units: done.units,
        dispenseQty: 4,
        returnedQty: 0,
      },
    ],
  }
}

function blank() {
  return {
    seq: 20,
    lines: seedLines(),
    stocks: { ...SEED_STOCK },
    dispenses: [seedDispense()],
    idempotency: {},
  }
}

const memory = { state: null }

function storage() {
  try {
    return typeof sessionStorage === 'undefined' ? null : sessionStorage
  } catch {
    return null
  }
}

function readState() {
  const store = storage()
  if (!store) {
    if (!memory.state) memory.state = blank()
    return memory.state
  }
  try {
    const raw = store.getItem(STORAGE_KEY)
    if (!raw) return blank()
    const parsed = JSON.parse(raw)
    if (!parsed || !Array.isArray(parsed.lines) || !Array.isArray(parsed.dispenses)) return blank()
    return {
      seq: typeof parsed.seq === 'number' ? parsed.seq : 20,
      lines: parsed.lines,
      stocks: parsed.stocks && typeof parsed.stocks === 'object' ? parsed.stocks : { ...SEED_STOCK },
      dispenses: parsed.dispenses,
      idempotency: parsed.idempotency && typeof parsed.idempotency === 'object' ? parsed.idempotency : {},
    }
  } catch {
    return blank()
  }
}

function writeState(state) {
  const store = storage()
  if (!store) {
    memory.state = state
    return
  }
  store.setItem(STORAGE_KEY, JSON.stringify(state))
}

function requestId() {
  return `mock-${crypto.randomUUID()}`
}

function text(value) {
  return typeof value === 'string' ? value.trim() : ''
}

function fail(status, code, message) {
  throw apiError(status, code, message, { requestId: requestId() })
}

function haystack(item) {
  return [
    item.patientName,
    item.patientNo,
    item.patientId,
    item.registerNo,
    item.registerId,
    item.chargeId,
    item.squareNo,
    item.orderNo,
    item.orderId,
    item.itemName,
  ].join('\n').toLowerCase()
}

function presentLine(state, item) {
  const stockQty = round2(state.stocks[item.itemNo] ?? 0)
  const availableQty = round2(item.chargedQty - item.dispensedNet)
  const status = lineDispenseStatus(item.chargedQty, item.dispensedNet)
  return {
    chargeItemId: item.chargeItemId,
    itemNo: item.itemNo,
    itemName: item.itemName,
    spec: item.spec || '',
    units: item.units || '',
    chargedQty: item.chargedQty,
    dispensedQty: item.dispensedNet,
    returnedQty: item.returnedQty,
    availableQty,
    stockQty,
    stockHint: stockHint(stockQty, availableQty),
    lineDispenseStatus: status,
    registerId: item.registerId,
    encounterId: item.encounterId || item.registerId,
    patientId: item.patientId,
    patientName: item.patientName,
    patientNo: item.patientNo,
    registerNo: item.registerNo,
    orderId: item.orderId,
    orderNo: item.orderNo,
    chargeId: item.chargeId,
    squareNo: item.squareNo,
    orderItemId: item.orderItemId,
    deptId: PHARMACY.deptId,
    deptName: PHARMACY.deptName,
  }
}

function aggregateStatus(state, slip) {
  const statuses = slip.items.map((item) => {
    const line = state.lines.find((row) => row.chargeItemId === item.chargeItemId)
    if (!line) return 'NONE'
    return lineDispenseStatus(line.chargedQty, line.dispensedNet)
  })
  if (statuses.length > 0 && statuses.every((status) => status === 'FULL')) return 'FULL'
  if (statuses.every((status) => status === 'NONE')) return 'NONE'
  return 'PARTIAL'
}

function presentDispense(state, slip) {
  const items = slip.items.map((item) => ({
    dispenseItemId: item.dispenseItemId,
    chargeItemId: item.chargeItemId,
    orderItemId: item.orderItemId || '',
    itemNo: item.itemNo,
    itemName: item.itemName,
    spec: item.spec || '',
    units: item.units || '',
    dispenseQty: item.dispenseQty,
    returnedQty: item.returnedQty,
    returnableQty: round2(item.dispenseQty - item.returnedQty),
  }))
  return {
    dispenseId: slip.dispenseId,
    dispenseNo: slip.dispenseNo,
    operateAt: slip.operateAt,
    operatorId: slip.operatorId,
    operatorName: slip.operatorName,
    deptId: slip.deptId,
    deptName: slip.deptName,
    patientId: slip.patientId,
    patientName: slip.patientName,
    patientNo: slip.patientNo,
    registerId: slip.registerId,
    registerNo: slip.registerNo,
    chargeId: slip.chargeId,
    squareNo: slip.squareNo,
    orderId: slip.orderId,
    orderNo: slip.orderNo,
    items,
    summary: items.map((item) => `${item.itemName} × ${formatQty(item.dispenseQty)}`).join('、'),
    lineDispenseStatus: aggregateStatus(state, slip),
    status: slipHeaderStatus(slip),
  }
}

function slipHeaderStatus(slip) {
  const items = slip.items || []
  const returned = items.reduce((sum, item) => sum + round2(item.returnedQty), 0)
  const dispensed = items.reduce((sum, item) => sum + round2(item.dispenseQty), 0)
  if (returned > 0 && returned + 1e-9 >= dispensed) return 'RETURNED'
  if (returned > 0) return 'PARTIAL_RETURNED'
  return 'DONE'
}

function guardIdempotency(state, key) {
  if (key && state.idempotency[key]) fail(409, 'IDEMPOTENT_REJECT')
}

function rememberKey(state, key, kind) {
  if (key) state.idempotency[key] = { kind }
}

export function resetMockDispenses(accessToken) {
  if (accessToken) mockMe(accessToken)
  memory.state = null
  const store = storage()
  if (store) store.removeItem(STORAGE_KEY)
}

export function mockSearchPending(accessToken, query) {
  mockMe(accessToken)
  const keyword = text(query).toLowerCase()
  if (!keyword) return { items: [], empty: 'prompt' }
  const state = readState()
  const matched = state.lines.filter((item) => haystack(item).includes(keyword))
  if (matched.length === 0) return { items: [], empty: 'none' }
  const drugs = matched.filter((item) => item.itemType === 'DRUG')
  if (drugs.length === 0) return { items: [], empty: 'none' }
  const pending = drugs.filter((item) => item.chargeStatus === 'NORMAL' && round2(item.chargedQty - item.dispensedNet) > 0)
  if (pending.length > 0) return { items: pending.map((item) => presentLine(state, item)), empty: '' }
  const normal = drugs.some((item) => item.chargeStatus === 'NORMAL')
  if (!normal) fail(409, 'NOT_CHARGED')
  return { items: [], empty: 'done' }
}

export function mockListTodayPending(accessToken) {
  mockMe(accessToken)
  const state = readState()
  const pending = state.lines.filter((item) => item.itemType === 'DRUG' && item.chargeStatus === 'NORMAL' && round2(item.chargedQty - item.dispensedNet) > 0)
  return {
    items: pending.map((item) => presentLine(state, item)),
    empty: pending.length > 0 ? '' : 'today',
  }
}

function normalizeDispenseItems(input) {
  const raw = input && Array.isArray(input.items) ? input.items : []
  const items = []
  for (const item of raw) {
    if (!item || item.selected === false) continue
    const chargeItemId = text(item.chargeItemId)
    const dispenseQty = round2(item.dispenseQty)
    if (!chargeItemId || !(dispenseQty > 0)) continue
    items.push({ chargeItemId, dispenseQty })
  }
  return items
}

export function mockConfirmDispense(accessToken, input) {
  const user = mockMe(accessToken)
  const state = readState()
  const key = text(input && input.idempotencyKey)
  guardIdempotency(state, key)
  const requested = normalizeDispenseItems(input)
  if (requested.length === 0) fail(400, 'EMPTY_DISPENSE_ITEMS')

  const plan = []
  for (const item of requested) {
    const current = state.lines.find((row) => row.chargeItemId === item.chargeItemId)
    if (!current) fail(404, 'NOT_FOUND', '未找到待发明细')
    if (current.itemType !== 'DRUG') fail(400, 'VALIDATION_ERROR', '非药品不能发药')
    if (current.chargeStatus !== 'NORMAL') fail(409, 'NOT_CHARGED')
    const available = round2(current.chargedQty - current.dispensedNet)
    // OpenAPI 0.3.9 码为 ALREADY_DISPENSED，与清单 ALREADY_FULLY_DISPENSED 同义。
    if (!(available > 0)) fail(409, 'ALREADY_FULLY_DISPENSED')
    if (item.dispenseQty > available) fail(409, 'DISPENSE_EXCEEDS_AVAILABLE')
    plan.push({ current, qty: item.dispenseQty })
  }

  const need = new Map()
  for (const step of plan) {
    need.set(step.current.itemNo, round2((need.get(step.current.itemNo) || 0) + step.qty))
  }
  for (const [itemNo, qty] of need) {
    if (qty > round2(state.stocks[itemNo] ?? 0)) fail(409, 'INSUFFICIENT_STOCK', '库存不足，无法发药')
  }

  const groups = new Map()
  for (const step of plan) {
    const chargeId = step.current.chargeId
    if (!groups.has(chargeId)) groups.set(chargeId, [])
    groups.get(chargeId).push(step)
  }
  const slips = []
  for (const steps of groups.values()) {
    const head = steps[0].current
    const seq = state.seq + 1
    state.seq = seq
    const slip = {
      dispenseId: `dsp-${seq}`,
      dispenseNo: `FY${todayInShanghai().replace(/-/g, '')}-${String(seq).padStart(4, '0')}`,
      operateAt: new Date().toISOString(),
      operatorId: user.id,
      operatorName: user.displayName,
      deptId: PHARMACY.deptId,
      deptName: PHARMACY.deptName,
      patientId: head.patientId,
      patientName: head.patientName,
      patientNo: head.patientNo,
      registerId: head.registerId,
      registerNo: head.registerNo,
      chargeId: head.chargeId,
      squareNo: head.squareNo,
      orderId: head.orderId,
      orderNo: head.orderNo,
      items: steps.map((step, index) => {
        step.current.dispensedNet = round2(step.current.dispensedNet + step.qty)
        state.stocks[step.current.itemNo] = round2((state.stocks[step.current.itemNo] ?? 0) - step.qty)
        return {
          dispenseItemId: `di-${seq}-${index + 1}`,
          chargeItemId: step.current.chargeItemId,
          orderItemId: step.current.orderItemId,
          itemNo: step.current.itemNo,
          itemName: step.current.itemName,
          spec: step.current.spec,
          units: step.current.units,
          dispenseQty: step.qty,
          returnedQty: 0,
        }
      }),
    }
    slips.push(slip)
  }
  state.dispenses.unshift(...slips.slice().reverse())
  rememberKey(state, key, 'dispense')
  writeState(state)
  const presented = slips.map((slip) => presentDispense(state, slip))
  return {
    ...presented[0],
    dispenses: presented,
    lines: plan.map((step) => presentLine(state, step.current)),
  }
}

export function mockListDispenses(accessToken, input = {}) {
  mockMe(accessToken)
  const state = readState()
  const patient = text(input.patient).toLowerCase()
  const dispenseNo = text(input.dispenseNo).toLowerCase()
  const todayOnly = input.today !== false
  const day = todayInShanghai()
  const items = state.dispenses.filter((slip) => {
    if (todayOnly && formatDay(slip.operateAt) !== day) return false
    if (patient) {
      const blob = `${slip.patientName}\n${slip.patientNo}\n${slip.patientId}`.toLowerCase()
      if (!blob.includes(patient)) return false
    }
    if (dispenseNo && !String(slip.dispenseNo || '').toLowerCase().includes(dispenseNo)) return false
    return true
  })
  return { items: items.map((slip) => presentDispense(state, slip)) }
}

function formatDay(iso) {
  const date = new Date(iso)
  if (Number.isNaN(date.getTime())) return ''
  return todayInShanghai(date)
}

export function mockGetDispense(accessToken, dispenseId) {
  mockMe(accessToken)
  const state = readState()
  const slip = state.dispenses.find((item) => item.dispenseId === text(dispenseId))
  if (!slip) fail(404, 'NOT_FOUND', '未找到发药单')
  return presentDispense(state, slip)
}

function normalizeReturnItems(input) {
  const raw = input && Array.isArray(input.items) ? input.items : []
  const items = []
  for (const item of raw) {
    if (!item || item.selected === false) continue
    const returnQty = round2(item.returnQty != null ? item.returnQty : item.qty)
    if (!(returnQty > 0)) continue
    items.push({
      dispenseItemId: text(item.dispenseItemId),
      chargeItemId: text(item.chargeItemId),
      returnQty,
    })
  }
  return items
}

function findSlipItem(state, dispenseId, item) {
  const slips = dispenseId ? state.dispenses.filter((slip) => slip.dispenseId === dispenseId) : state.dispenses
  for (const slip of slips) {
    const row = slip.items.find((candidate) => {
      if (item.dispenseItemId) return candidate.dispenseItemId === item.dispenseItemId
      return item.chargeItemId && candidate.chargeItemId === item.chargeItemId
    })
    if (row) return { slip, row }
  }
  return null
}

export function mockReturnDispense(accessToken, input) {
  mockMe(accessToken)
  const state = readState()
  const key = text(input && input.idempotencyKey)
  guardIdempotency(state, key)
  const reason = text(input && input.reason)
  if (!reason) fail(400, 'RETURN_REASON_REQUIRED')
  const requested = normalizeReturnItems(input)
  if (requested.length === 0) fail(400, 'VALIDATION_ERROR', '没有可退明细（未发或已退完）')
  const dispenseId = text(input && input.dispenseId)

  const plan = []
  for (const item of requested) {
    const found = findSlipItem(state, dispenseId, item)
    if (!found) fail(409, 'NOT_DISPENSED')
    const charge = state.lines.find((row) => row.chargeItemId === found.row.chargeItemId)
    if (!charge || !(charge.dispensedNet > 0)) fail(409, 'NOT_DISPENSED')
    const returnable = round2(found.row.dispenseQty - found.row.returnedQty)
    if (item.returnQty > returnable || item.returnQty > round2(charge.dispensedNet)) {
      fail(409, 'RETURN_EXCEEDS_DISPENSE')
    }
    plan.push({ ...found, charge, qty: item.returnQty })
  }

  let total = 0
  for (const step of plan) {
    step.row.returnedQty = round2(step.row.returnedQty + step.qty)
    step.charge.dispensedNet = round2(step.charge.dispensedNet - step.qty)
    step.charge.returnedQty = round2(step.charge.returnedQty + step.qty)
    state.stocks[step.charge.itemNo] = round2((state.stocks[step.charge.itemNo] ?? 0) + step.qty)
    total = round2(total + step.qty)
  }
  const seq = state.seq + 1
  state.seq = seq
  const chargeId = plan[0].slip.chargeId
  rememberKey(state, key, 'return')
  writeState(state)
  return {
    returnId: `rt-${seq}`,
    dispenseId: plan[0].slip.dispenseId,
    chargeId,
    reason,
    returnedQty: total,
    stockNote: `库存已回加 ${formatQty(total)}`,
    items: plan.map((step) => ({
      dispenseItemId: step.row.dispenseItemId,
      chargeItemId: step.charge.chargeItemId,
      returnQty: step.qty,
      stockQty: round2(state.stocks[step.charge.itemNo] ?? 0),
    })),
    lines: plan.map((step) => presentLine(state, step.charge)),
  }
}
