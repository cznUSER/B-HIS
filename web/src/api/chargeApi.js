// 2026-10-12 P1-H05-CHARGE-UI
// 2026-10-12 真接口为主：charge-preview、POST /charges、pay、GET /charges、refund。
// VITE_USE_MOCK=true 时仍走本地演示，不发 HTTP。

import { apiUrl, useMock } from './config.js'
import { apiError, readErrorResponse } from './errors.js'
import {
  PAY_CASH,
  formatOperate,
  lineStock,
  moneyCents,
} from './chargeContract.js'
import {
  mockGetCharge,
  mockListCharges,
  mockLoadPending,
  mockLookup,
  mockPay,
  mockRefund,
} from './mockCharges.js'
import { fetchPatient } from './patientApi.js'
import { todayInShanghai } from './patientContract.js'
import { findCatalogItem, registerIsOrderable } from './orderContract.js'
import { listOrderRegisters, listOrders } from './orderApi.js'
import { deptName, doctorName } from '../register/master.js'

const JSON_ACCEPT = 'application/json'

export const CHARGE_ROUTES = {
  preview: (orderId) => `/orders/${orderId}/charge-preview`,
  collection: '/charges',
  item: (chargeId) => `/charges/${chargeId}`,
  pay: (chargeId) => `/charges/${chargeId}/pay`,
  refund: (chargeId) => `/charges/${chargeId}/refund`,
}

const COPY = {
  ENCOUNTER_REQUIRED: '未挂号/无有效就诊，不能收费',
  INSUFFICIENT_STOCK: '库存不足，无法收费',
  INVALID_AMOUNT: '数量或金额不合法',
  CHARGE_AMOUNT_MISMATCH: '实收与应收不一致',
  ITEM_NOT_CAN_BACK: '该明细不可退',
  REFUND_REQUIRES_RETURN: '已发药，请先退药再退费',
  REFUND_EXCEEDS_REFUNDABLE: '退数量超过可退数量',
  ALREADY_ABATED: '该收费单已作废，不能再退',
  ALREADY_CHARGED: '该处方已收费',
}

function delay(ms) {
  return new Promise((resolve) => {
    setTimeout(resolve, ms)
  })
}

function bearer(accessToken) {
  return accessToken ? { Authorization: `Bearer ${accessToken}` } : {}
}

function requestInit(init) {
  return { credentials: 'omit', cache: 'no-store', ...init }
}

async function parseJson(response) {
  const text = await response.text()
  if (!text) return null
  try {
    return JSON.parse(text)
  } catch {
    throw apiError(response.status, 'HTTP_ERROR', '响应无法解析')
  }
}

export function presentChargeError(error) {
  const copy = COPY[error && error.code]
  if (!copy) return error
  return apiError(error.status || 409, error.code, copy, {
    details: error.details,
    requestId: error.requestId,
  })
}

async function callJson(path, init) {
  let response
  try {
    response = await fetch(apiUrl(path), requestInit(init))
  } catch {
    throw apiError(0, 'HTTP_ERROR', '无法连接收费服务')
  }
  return response
}

function asText(value) {
  return typeof value === 'string' ? value.trim() : ''
}

function assertMoney(value, field) {
  const number = Number(value)
  if (!Number.isFinite(number) || number < 0) {
    throw apiError(500, 'HTTP_ERROR', `收费响应缺少${field}`)
  }
  return number
}

export function assertChargeDto(raw) {
  const charge = raw && typeof raw === 'object' ? raw : null
  if (!charge) throw apiError(500, 'HTTP_ERROR', '收费响应无法读取')
  const registerId = asText(charge.registerId || charge.encounterId)
  const patientId = asText(charge.patientId)
  if (!registerId || !patientId) throw apiError(500, 'HTTP_ERROR', '收费响应无法读取')
  const status = asText(charge.status)
  if (status && status !== 'NORMAL' && status !== 'ABATE') {
    throw apiError(500, 'HTTP_ERROR', '收费响应无法读取')
  }
  const items = (Array.isArray(charge.items) ? charge.items : []).map((item, index) => {
    const row = item && typeof item === 'object' ? item : {}
    return {
      chargeItemId: asText(row.chargeItemId),
      orderItemId: asText(row.orderItemId),
      lineNo: Number.isInteger(row.lineNo) ? row.lineNo : index + 1,
      itemNo: asText(row.itemNo),
      itemName: asText(row.itemName) || asText(row.itemNo),
      qty: assertMoney(row.qty, '数量'),
      unitPrice: assertMoney(row.unitPrice, '单价'),
      totalFee: assertMoney(row.totalFee, '金额'),
      canBack: row.canBack !== false,
      refundedQty: row.refundedQty == null ? 0 : assertMoney(row.refundedQty, '已退数量'),
    }
  })
  return {
    chargeId: asText(charge.chargeId),
    squareNo: asText(charge.squareNo),
    orderId: asText(charge.orderId),
    registerId,
    encounterId: registerId,
    patientId,
    totalFee: assertMoney(charge.totalFee, '合计'),
    squareFee: assertMoney(charge.squareFee, '应收'),
    favourableFee: assertMoney(charge.favourableFee, '优惠'),
    status: status || 'NORMAL',
    successorChargeId: asText(charge.successorChargeId),
    abatedFromId: asText(charge.abatedFromId),
    paidAt: asText(charge.paidAt),
    items,
  }
}

function assertPaymentDto(raw) {
  const payment = raw && typeof raw === 'object' ? raw : null
  if (!payment || !asText(payment.paymentId) || !asText(payment.chargeId)) {
    throw apiError(500, 'HTTP_ERROR', '支付响应无法读取')
  }
  return {
    paymentId: asText(payment.paymentId),
    paymentNo: asText(payment.paymentNo),
    chargeId: asText(payment.chargeId),
    direction: asText(payment.direction),
    payType: asText(payment.payType),
    amount: assertMoney(payment.amount, '支付金额'),
    status: asText(payment.status),
    paidAt: asText(payment.paidAt || payment.succeededAt),
  }
}

function viewItem(item, order) {
  const catalog = findCatalogItem(item.itemNo)
  const orderLine = order && Array.isArray(order.items)
    ? order.items.find((row) => row.itemNo === item.itemNo)
    : null
  const drug = (catalog && catalog.itemType === 'DRUG') || (orderLine && orderLine.itemType === 'DRUG')
  const stock = lineStock(item.itemNo, item.qty)
  const refundableQty = Math.max(0, item.qty - (item.refundedQty || 0))
  return {
    chargeItemId: item.chargeItemId,
    lineNo: item.lineNo,
    itemNo: item.itemNo,
    itemName: item.itemName,
    qty: item.qty,
    unit: catalog ? catalog.units : '',
    unitPriceCents: moneyCents(item.unitPrice),
    lineFeeCents: moneyCents(item.totalFee),
    stockAvailable: stock.available,
    stockSufficient: stock.sufficient,
    stockHint: stock.hint,
    canBack: item.canBack !== false,
    refundedQty: item.refundedQty || 0,
    refundableQty,
    dispenseBlocked: Boolean(order && order.status === 'PROC' && drug),
  }
}

function viewCharge(dto, extra = {}) {
  const items = (dto.items || []).map((item) => viewItem(item, extra.order))
  return {
    chargeId: dto.chargeId,
    squareNo: dto.squareNo,
    orderId: dto.orderId,
    registerId: dto.registerId,
    encounterId: dto.encounterId,
    patientId: dto.patientId,
    patientName: extra.patientName || '',
    patientNo: extra.patientNo || '',
    registerSummary: extra.registerSummary || dto.registerId,
    totalFeeCents: moneyCents(dto.totalFee),
    squareFeeCents: moneyCents(dto.squareFee),
    squareFee: dto.squareFee,
    favourableFeeCents: moneyCents(dto.favourableFee),
    status: dto.status,
    paid: Boolean(dto.paidAt),
    paidAt: dto.paidAt || '',
    operateAt: dto.paidAt || '',
    operateText: dto.paidAt ? formatOperate(dto.paidAt) : '待支付',
    canRefund: dto.status === 'NORMAL' && Boolean(dto.paidAt),
    items,
  }
}

function previewFrom(dto) {
  const items = (dto.items || []).map((item) => viewItem(item, null))
  return {
    orderId: dto.orderId,
    chargeId: dto.chargeId,
    squareNo: dto.squareNo,
    items,
    totalFeeCents: moneyCents(dto.totalFee),
    favourableFeeCents: moneyCents(dto.favourableFee),
    squareFeeCents: moneyCents(dto.squareFee),
    squareFee: dto.squareFee,
    amountDueCents: moneyCents(dto.squareFee),
    amountPaidCents: moneyCents(dto.squareFee),
    payType: PAY_CASH,
    stockBlocked: items.some((item) => !item.stockSufficient),
  }
}

async function readCharge(response, okStatus) {
  if (response.status !== okStatus) throw presentChargeError(await readErrorResponse(response))
  return assertChargeDto(await parseJson(response))
}

export async function previewCharge(accessToken, orderId) {
  const id = asText(orderId)
  if (!id) throw apiError(400, 'VALIDATION_ERROR', '请先提交处方')
  if (useMock()) {
    await delay(80)
    const pending = mockLoadPending(accessToken, id)
    if (!pending.preview) throw apiError(409, 'VALIDATION_ERROR', '请先提交处方')
    return pending
  }
  const response = await callJson(CHARGE_ROUTES.preview(id), {
    method: 'GET',
    headers: { Accept: JSON_ACCEPT, ...bearer(accessToken) },
  })
  return readCharge(response, 200)
}

async function createCharge(accessToken, orderId) {
  const response = await callJson(CHARGE_ROUTES.collection, {
    method: 'POST',
    headers: {
      Accept: JSON_ACCEPT,
      'Content-Type': 'application/json; charset=utf-8',
      'Idempotency-Key': crypto.randomUUID(),
      ...bearer(accessToken),
    },
    body: JSON.stringify({ orderId }),
  })
  return readCharge(response, 201)
}

async function getChargeDto(accessToken, chargeId) {
  const response = await callJson(CHARGE_ROUTES.item(chargeId), {
    method: 'GET',
    headers: { Accept: JSON_ACCEPT, ...bearer(accessToken) },
  })
  return readCharge(response, 200)
}

async function listChargeDtos(accessToken, query) {
  const params = new URLSearchParams()
  if (query.registerId) params.set('registerId', query.registerId)
  if (query.patientId) params.set('patientId', query.patientId)
  if (query.visitDate) params.set('visitDate', query.visitDate)
  if (query.from) params.set('from', query.from)
  if (query.to) params.set('to', query.to)
  const response = await callJson(`${CHARGE_ROUTES.collection}?${params.toString()}`, {
    method: 'GET',
    headers: { Accept: JSON_ACCEPT, ...bearer(accessToken) },
  })
  if (!response.ok) throw presentChargeError(await readErrorResponse(response))
  const body = await parseJson(response)
  const items = body && Array.isArray(body.items) ? body.items : []
  return items.map((item) => assertChargeDto(item))
}

async function findOpenCharge(accessToken, registerId, orderId) {
  const items = await listChargeDtos(accessToken, { registerId })
  return items.find((item) => item.orderId === orderId && item.status === 'NORMAL' && !item.paidAt) || null
}

function patientOf(subject) {
  return {
    patientId: subject.patientId || '',
    patientNo: subject.patientNo || '',
    patientName: subject.patientName || '',
  }
}

function registerOf(subject) {
  if (!subject.registerId) return null
  return {
    registerId: subject.registerId,
    encounterId: subject.registerId,
    registerNo: subject.registerNo || '',
    summary: subject.registerSummary || subject.registerNo || subject.registerId,
    status: 'NORMAL',
  }
}

export async function lookupChargeSubjects(accessToken, keyword) {
  if (useMock()) {
    await delay(60)
    const found = mockLookup(accessToken, keyword)
    return {
      items: found.items.map((item) => ({ ...item, key: item.patientId })),
    }
  }
  const query = asText(keyword)
  const listed = await listOrderRegisters(accessToken)
  const registers = (listed.items || []).filter((item) => registerIsOrderable(item))
  const items = []
  for (const reg of registers) {
    const summary = [reg.registerNo, reg.departmentName || deptName(reg.departmentId), reg.doctorName || doctorName(reg.doctorId)]
      .filter(Boolean)
      .join(' · ')
    const hay = [reg.patientName, reg.patientNo, reg.registerNo, summary].join(' ')
    if (query && !hay.includes(query)) continue
    const orders = await listOrders(accessToken, { registerId: reg.registerId })
    const bills = (orders.items || []).filter((item) => item.status === 'COMMIT' || item.status === 'PROC')
    if (bills.length === 0) {
      items.push({
        key: `reg:${reg.registerId}`,
        patientId: reg.patientId,
        patientNo: reg.patientNo || '',
        patientName: reg.patientName || '',
        registerId: reg.registerId,
        registerNo: reg.registerNo || '',
        registerSummary: summary,
        orderId: '',
        recipeNo: '',
        scene: 'NO_COMMIT',
        sceneLabel: '请先提交处方',
      })
      continue
    }
    for (const order of bills) {
      const short = (order.items || []).some((item) => !lineStock(item.itemNo, item.amount).sufficient)
      let scene = 'PENDING'
      let sceneLabel = '待收'
      if (order.charged) {
        scene = order.status === 'PROC' ? 'DISPENSED' : 'CHARGED'
        sceneLabel = order.status === 'PROC' ? '已发药' : '已收费'
      } else if (short) {
        scene = 'STOCK'
        sceneLabel = '库存不足'
      }
      items.push({
        key: order.orderId,
        patientId: reg.patientId,
        patientNo: reg.patientNo || '',
        patientName: reg.patientName || '',
        registerId: reg.registerId,
        registerNo: reg.registerNo || '',
        registerSummary: summary,
        orderId: order.orderId,
        recipeNo: order.orderNo,
        orderStatus: order.status,
        scene,
        sceneLabel,
      })
    }
  }
  return { items }
}

export async function loadPendingCharge(accessToken, subject) {
  if (useMock()) {
    await delay(80)
    const patientId = typeof subject === 'string' ? subject : subject && subject.patientId
    return mockLoadPending(accessToken, patientId)
  }
  const current = subject && typeof subject === 'object' ? subject : {}
  const patient = patientOf(current)
  const register = registerOf(current)
  if (!register) return { scene: 'NO_REGISTER', patient, register: null, order: null, preview: null }
  if (!current.orderId || current.scene === 'NO_COMMIT') {
    return { scene: 'NO_COMMIT', patient, register, order: null, preview: null }
  }
  const order = { orderId: current.orderId, recipeNo: current.recipeNo || '', status: current.orderStatus || 'COMMIT' }
  if (current.scene === 'CHARGED' || current.scene === 'DISPENSED') {
    const existing = await listChargeDtos(accessToken, { registerId: current.registerId })
    const paid = existing.find((item) => item.orderId === current.orderId && item.status === 'NORMAL' && item.paidAt)
    return {
      scene: current.scene,
      patient,
      register,
      order,
      preview: null,
      chargeId: paid ? paid.chargeId : '',
      squareNo: paid ? paid.squareNo : '',
    }
  }
  const dto = await previewCharge(accessToken, current.orderId)
  const preview = previewFrom(dto)
  return {
    scene: preview.stockBlocked ? 'STOCK' : 'PENDING',
    patient,
    register,
    order,
    preview,
  }
}

export async function payCharge(accessToken, input) {
  const payType = input && input.payType
  if (payType !== PAY_CASH) throw apiError(400, 'VALIDATION_ERROR', '电子支付为预留，不能收款')
  if (useMock()) {
    await delay(120)
    const amountCents = Number.isInteger(input.amountCents) ? input.amountCents : moneyCents(input.amount)
    return mockPay(accessToken, { orderId: input.orderId, payType, amountCents })
  }
  let created
  try {
    created = await createCharge(accessToken, input.orderId)
  } catch (error) {
    if (error.code === 'ILLEGAL_STATE_TRANSITION' && input.registerId) {
      created = await findOpenCharge(accessToken, input.registerId, input.orderId)
      if (!created) throw apiError(409, 'ALREADY_CHARGED', COPY.ALREADY_CHARGED)
    } else {
      throw presentChargeError(error)
    }
  }
  if (created.paidAt) throw apiError(409, 'ALREADY_CHARGED', COPY.ALREADY_CHARGED)
  if (moneyCents(created.favourableFee) !== 0) {
    throw apiError(409, 'CHARGE_AMOUNT_MISMATCH', COPY.CHARGE_AMOUNT_MISMATCH)
  }
  const response = await callJson(CHARGE_ROUTES.pay(created.chargeId), {
    method: 'POST',
    headers: {
      Accept: JSON_ACCEPT,
      'Content-Type': 'application/json; charset=utf-8',
      'Idempotency-Key': crypto.randomUUID(),
      ...bearer(accessToken),
    },
    body: JSON.stringify({ payType: PAY_CASH, amount: created.squareFee }),
  })
  if (!response.ok) throw presentChargeError(await readErrorResponse(response))
  const payment = assertPaymentDto(await parseJson(response))
  const paid = await getChargeDto(accessToken, created.chargeId)
  return { charge: viewCharge(paid), payment }
}

async function orderOf(accessToken, orderId) {
  if (!orderId) return null
  const response = await callJson(`/orders/${encodeURIComponent(orderId)}`, {
    method: 'GET',
    headers: { Accept: JSON_ACCEPT, ...bearer(accessToken) },
  })
  if (!response.ok) return null
  const body = await parseJson(response)
  return body && typeof body === 'object' ? body : null
}

async function labelCharge(accessToken, dto) {
  let patientName = ''
  let patientNo = ''
  let registerSummary = dto.registerId
  try {
    const patient = await fetchPatient(accessToken, dto.patientId)
    patientName = patient.name || ''
    patientNo = patient.patientNo || ''
  } catch {
    patientName = ''
  }
  try {
    const response = await callJson(`/registers/${encodeURIComponent(dto.registerId)}`, {
      method: 'GET',
      headers: { Accept: JSON_ACCEPT, ...bearer(accessToken) },
    })
    if (response.ok) {
      const reg = await parseJson(response)
      registerSummary = [reg.registerNo, deptName(reg.departmentId), doctorName(reg.doctorId)].filter(Boolean).join(' · ') || dto.registerId
    }
  } catch {
    registerSummary = dto.registerId
  }
  const view = viewCharge(dto, { patientName, patientNo, registerSummary })
  if (view.status === 'NORMAL' && view.squareFeeCents === 0) return null
  return view
}

export async function listCharges(accessToken, input = {}) {
  if (useMock()) {
    await delay(60)
    return mockListCharges(accessToken, input)
  }
  const todayOnly = input.todayOnly !== false
  const query = { registerId: asText(input.registerId) }
  if (todayOnly) query.visitDate = todayInShanghai()
  else {
    query.from = '2020-01-01'
    query.to = todayInShanghai()
  }
  const dtos = await listChargeDtos(accessToken, query)
  const items = []
  for (const dto of dtos) {
    const view = await labelCharge(accessToken, dto)
    if (view) items.push(view)
  }
  const keyword = asText(input.keyword)
  const filtered = keyword
    ? items.filter((item) => item.patientName.includes(keyword) || item.patientNo.includes(keyword))
    : items
  filtered.sort((a, b) => String(b.operateAt).localeCompare(String(a.operateAt)))
  return { items: filtered }
}

export async function fetchCharge(accessToken, chargeId) {
  if (useMock()) {
    await delay(40)
    return mockGetCharge(accessToken, chargeId)
  }
  const dto = await getChargeDto(accessToken, chargeId)
  const order = await orderOf(accessToken, dto.orderId)
  const view = await labelCharge(accessToken, dto)
  if (!view) throw apiError(404, 'NOT_FOUND', '未找到收费单')
  view.items = (dto.items || []).map((item) => viewItem(item, order))
  return view
}

export async function refundCharge(accessToken, chargeId, input) {
  const payType = input && input.payType
  if (payType !== PAY_CASH) throw apiError(400, 'VALIDATION_ERROR', '电子支付为预留，不能退款')
  if (useMock()) {
    await delay(120)
    return mockRefund(accessToken, chargeId, input)
  }
  const response = await callJson(CHARGE_ROUTES.refund(chargeId), {
    method: 'POST',
    headers: {
      Accept: JSON_ACCEPT,
      'Content-Type': 'application/json; charset=utf-8',
      'Idempotency-Key': crypto.randomUUID(),
      ...bearer(accessToken),
    },
    body: JSON.stringify({
      reason: asText(input.reason) || '患者要求',
      payType: PAY_CASH,
      favourableAlloc: 'PROPORTIONAL',
      items: (input.items || []).map((item) => ({
        chargeItemId: item.chargeItemId,
        refundQty: item.refundQty,
      })),
    }),
  })
  if (!response.ok) throw presentChargeError(await readErrorResponse(response))
  const body = await parseJson(response)
  const abated = assertChargeDto(body && (body.abatedCharge || body.abatedSquare))
  const created = body && body.newSquare ? assertChargeDto(body.newSquare) : null
  const next = created && moneyCents(created.squareFee) > 0 ? viewCharge(created) : null
  return {
    abatedSquare: viewCharge(abated),
    newSquare: next,
    refundPayment: body && body.refundPayment ? assertPaymentDto(body.refundPayment) : null,
    fullRefund: next == null,
  }
}
