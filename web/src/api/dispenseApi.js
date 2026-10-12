// 2026-10-12 P1-H05-DISPENSE-UI
// 2026-10-12 P1-H05-DISPENSE 真接口：VITE_USE_MOCK=false 时走已支付收费单拼待发，
// POST/GET /dispenses、POST /dispenses/{id}/return、GET /inventory/items。
// 不调用 deprecated POST /orders/{orderId}/dispense。Mock 仍只走本地演示。

import { apiUrl, useMock } from './config.js'
import { assertChargeDto } from './chargeApi.js'
import { apiError, readErrorResponse } from './errors.js'
import {
  PHARMACY,
  formatQty,
  lineDispenseStatus,
  round2,
  stockHint,
} from './dispenseContract.js'
import {
  mockConfirmDispense,
  mockGetDispense,
  mockListDispenses,
  mockListTodayPending,
  mockReturnDispense,
  mockSearchPending,
  resetMockDispenses,
} from './mockDispenses.js'
import { findCatalogItem, assertOrderDto, assertOrderList } from './orderContract.js'
import { fetchPatient, searchPatients } from './patientApi.js'
import { assertPatientList, todayInShanghai } from './patientContract.js'

const JSON_ACCEPT = 'application/json'
const RANGE_FROM = '2020-01-01'
const UUID_TEXT = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i

// 主路径 POST/GET /dispenses。POST /orders/{orderId}/dispense 是 deprecated 别名，客户端不调用。
export const DISPENSE_ROUTES = {
  collection: '/dispenses',
  item: (dispenseId) => `/dispenses/${dispenseId}`,
  return: (dispenseId) => `/dispenses/${dispenseId}/return`,
  orderAlias: (orderId) => `/orders/${orderId}/dispense`,
  inventory: '/inventory/items',
}

const COPY = {
  NOT_CHARGED: '未收费，不能发药',
  INSUFFICIENT_STOCK: '库存不足，无法发药',
  DISPENSE_EXCEEDS_AVAILABLE: '发药数量超过可发数量',
  EMPTY_DISPENSE_ITEMS: '没有可发明细',
  ALREADY_DISPENSED: '已全部发完，不能再发',
  ALREADY_FULLY_DISPENSED: '已全部发完，不能再发',
  NOT_DISPENSED: '尚未发药，不能退药',
  RETURN_EXCEEDS_DISPENSE: '退药数量超过可退数量',
  RETURN_REASON_REQUIRED: '请填写退药原因',
  IDEMPOTENT_REJECT: '重复提交已处理，未再次扣减/回滚',
  IDEMPOTENCY_KEY_CONFLICT: '重复提交已处理，未再次扣减/回滚',
  REFUND_REQUIRES_RETURN: '已发药，请先退药再退费',
}

function delay(ms) {
  return new Promise((resolve) => {
    setTimeout(resolve, ms)
  })
}

function text(value) {
  return typeof value === 'string' ? value.trim() : ''
}

function numberOr(value, fallback) {
  const number = Number(value)
  return Number.isFinite(number) ? number : fallback
}

function isUuid(value) {
  return UUID_TEXT.test(value)
}

function isIdNumber(value) {
  return /^(\d{15}|\d{17}[\dX])$/i.test(value)
}

function isNotFound(error) {
  return Boolean(error && (error.status === 404 || error.code === 'NOT_FOUND'))
}

function bearer(accessToken) {
  return accessToken ? { Authorization: `Bearer ${accessToken}` } : {}
}

function requestInit(init) {
  return { credentials: 'omit', cache: 'no-store', ...init }
}

export function presentDispenseError(error) {
  const copy = COPY[error && error.code]
  if (!copy) return error
  return apiError(error.status || 409, error.code, copy, {
    details: error.details,
    requestId: error.requestId,
  })
}

async function parseJson(response) {
  const raw = await response.text()
  if (!raw) return null
  try {
    return JSON.parse(raw)
  } catch {
    throw apiError(response.status, 'HTTP_ERROR', '响应无法解析')
  }
}

async function callJson(path, init) {
  let response
  try {
    response = await fetch(apiUrl(path), requestInit(init))
  } catch {
    throw apiError(0, 'HTTP_ERROR', '无法连接发药服务')
  }
  return response
}

async function getJson(path, accessToken) {
  const response = await callJson(path, {
    method: 'GET',
    headers: { Accept: JSON_ACCEPT, ...bearer(accessToken) },
  })
  if (response.status === 404) return null
  if (!response.ok) throw presentDispenseError(await readErrorResponse(response))
  return parseJson(response)
}

async function postJson(path, accessToken, body, idempotencyKey, okStatus) {
  const headers = {
    Accept: JSON_ACCEPT,
    'Content-Type': 'application/json; charset=utf-8',
    ...bearer(accessToken),
  }
  if (idempotencyKey) headers['Idempotency-Key'] = idempotencyKey
  const response = await callJson(path, {
    method: 'POST',
    headers,
    body: JSON.stringify(body),
  })
  if (response.status !== okStatus) throw presentDispenseError(await readErrorResponse(response))
  return parseJson(response)
}

function remember(map, key, load) {
  if (!key) return Promise.resolve(null)
  if (map.has(key)) return map.get(key)
  const pending = Promise.resolve().then(load).catch((error) => {
    map.delete(key)
    throw error
  })
  map.set(key, pending)
  return pending
}

function newCache() {
  return {
    patient: new Map(),
    register: new Map(),
    order: new Map(),
    charge: new Map(),
  }
}

function dayOf(iso) {
  const date = new Date(iso)
  if (Number.isNaN(date.getTime())) return ''
  return todayInShanghai(date)
}

function dateRange() {
  return { from: RANGE_FROM, to: todayInShanghai() }
}

async function listCharges(accessToken, query) {
  const params = new URLSearchParams()
  for (const key of ['patientId', 'registerId', 'visitDate', 'from', 'to']) {
    if (query[key]) params.set(key, query[key])
  }
  const body = await getJson(`/charges?${params.toString()}`, accessToken)
  const items = body && Array.isArray(body.items) ? body.items : []
  return items.map((item) => assertChargeDto(item))
}

function dedupeBy(list, key) {
  const map = new Map()
  for (const item of list) {
    const id = item && item[key]
    if (id && !map.has(id)) map.set(id, item)
  }
  return [...map.values()]
}

async function findPatients(accessToken, keyword) {
  const found = new Map()
  const add = (items) => {
    for (const item of items || []) {
      if (item && item.patientId) found.set(item.patientId, item)
    }
  }
  if (isIdNumber(keyword)) {
    try {
      add((await searchPatients(accessToken, { idNumber: keyword })).items)
    } catch (error) {
      if (!isNotFound(error)) throw error
    }
  }
  try {
    add((await searchPatients(accessToken, { name: keyword })).items)
  } catch (error) {
    if (!isNotFound(error)) throw error
  }
  const byNo = await getJson(`/patients?hospPkNo=${encodeURIComponent(keyword)}`, accessToken)
  if (byNo) add(assertPatientList(byNo).items)
  return [...found.values()]
}

async function listRegisters(accessToken, registerNo) {
  const body = await getJson(`/registers?registerNo=${encodeURIComponent(registerNo)}`, accessToken)
  const items = body && Array.isArray(body.items) ? body.items : []
  return items
    .map((item) => ({
      registerId: text(item && item.registerId),
      registerNo: text(item && item.registerNo),
    }))
    .filter((item) => item.registerId)
}

async function listOrdersByNo(accessToken, orderNo) {
  const body = await getJson(`/orders?orderNo=${encodeURIComponent(orderNo)}`, accessToken)
  if (!body) return []
  try {
    return assertOrderList(body).items
  } catch {
    return []
  }
}

async function chargesForPatients(accessToken, patients) {
  const found = []
  const range = dateRange()
  for (const patient of patients) {
    found.push(...await listCharges(accessToken, { patientId: patient.patientId, ...range }))
  }
  return found
}

async function chargesForUuid(accessToken, id) {
  const found = []
  const range = dateRange()
  const direct = await getJson(`/charges/${encodeURIComponent(id)}`, accessToken)
  if (direct) found.push(assertChargeDto(direct))
  found.push(...await listCharges(accessToken, { patientId: id, ...range }))
  found.push(...await listCharges(accessToken, { registerId: id, ...range }))
  const orderBody = await getJson(`/orders/${encodeURIComponent(id)}`, accessToken)
  if (orderBody) {
    try {
      const order = assertOrderDto(orderBody)
      const rows = await listCharges(accessToken, {
        patientId: order.patientId,
        registerId: order.registerId,
        ...range,
      })
      found.push(...rows.filter((row) => row.orderId === order.orderId))
    } catch {
      // 不是处方号时忽略，待发仍可由收费单命中。
    }
  }
  return dedupeBy(found, 'chargeId')
}

async function chargesForKeyword(accessToken, keyword) {
  const found = []
  found.push(...await chargesForPatients(accessToken, await findPatients(accessToken, keyword)))
  const range = dateRange()
  for (const reg of await listRegisters(accessToken, keyword)) {
    found.push(...await listCharges(accessToken, { registerId: reg.registerId, ...range }))
  }
  for (const order of await listOrdersByNo(accessToken, keyword)) {
    const rows = await listCharges(accessToken, {
      patientId: order.patientId,
      registerId: order.registerId,
      ...range,
    })
    found.push(...rows.filter((row) => row.orderId === order.orderId))
  }
  let charges = dedupeBy(found, 'chargeId')
  if (charges.length === 0) {
    const needle = keyword.toLowerCase()
    const all = await listCharges(accessToken, range)
    charges = all.filter((row) => row.squareNo.toLowerCase().includes(needle))
  }
  return charges
}

function loadPatient(accessToken, patientId, cache) {
  return remember(cache.patient, patientId, async () => {
    try {
      return await fetchPatient(accessToken, patientId)
    } catch (error) {
      if (isNotFound(error)) return null
      throw error
    }
  })
}

function loadRegister(accessToken, registerId, cache) {
  return remember(cache.register, registerId, async () => {
    const body = await getJson(`/registers/${encodeURIComponent(registerId)}`, accessToken)
    if (!body || typeof body !== 'object') return null
    return {
      registerId: text(body.registerId) || registerId,
      registerNo: text(body.registerNo),
    }
  })
}

function loadOrder(accessToken, orderId, cache) {
  return remember(cache.order, orderId, async () => {
    const body = await getJson(`/orders/${encodeURIComponent(orderId)}`, accessToken)
    if (!body) return null
    try {
      return assertOrderDto(body)
    } catch {
      return null
    }
  })
}

function loadCharge(accessToken, chargeId, cache) {
  return remember(cache.charge, chargeId, async () => {
    const body = await getJson(`/charges/${encodeURIComponent(chargeId)}`, accessToken)
    if (!body) return null
    try {
      return assertChargeDto(body)
    } catch {
      return null
    }
  })
}

async function loadStock(accessToken) {
  const response = await callJson(DISPENSE_ROUTES.inventory, {
    method: 'GET',
    headers: { Accept: JSON_ACCEPT, ...bearer(accessToken) },
  })
  if (!response.ok) throw presentDispenseError(await readErrorResponse(response))
  const body = await parseJson(response)
  const map = new Map()
  const items = body && Array.isArray(body.items) ? body.items : []
  for (const item of items) {
    const row = item && typeof item === 'object' ? item : {}
    const code = text(row.itemNo) || text(row.skuCode)
    if (!code) continue
    map.set(code, { qty: numberOr(row.qtyOnHand, 0), units: text(row.units) })
  }
  return map
}

async function fetchDispenseList(accessToken, params) {
  const query = params.toString()
  const path = query ? `${DISPENSE_ROUTES.collection}?${query}` : DISPENSE_ROUTES.collection
  const body = await getJson(path, accessToken)
  const items = body && Array.isArray(body.items) ? body.items : []
  return items.map((item) => assertDispenseDto(item))
}

function dispensesForCharge(accessToken, chargeId, cache) {
  return remember(cache.slips, chargeId, () => {
    const params = new URLSearchParams({ chargeId })
    return fetchDispenseList(accessToken, params)
  })
}

function drugMeta(order, chargeItem) {
  const items = order && Array.isArray(order.items) ? order.items : []
  const match = items.find((item) => item.orderItemId && item.orderItemId === chargeItem.orderItemId)
    || items.find((item) => item.itemNo === chargeItem.itemNo)
  if (match) return match.itemType === 'DRUG' ? match : null
  const catalog = findCatalogItem(chargeItem.itemNo)
  if (catalog) return catalog.itemType === 'DRUG' ? catalog : null
  return { itemNo: chargeItem.itemNo, spec: '', units: '' }
}

function netForChargeItem(slips, chargeItemId) {
  let net = 0
  for (const slip of slips) {
    for (const item of slip.items) {
      if (item.chargeItemId !== chargeItemId) continue
      net = round2(net + round2(item.dispensedQty) - round2(item.returnedQty))
    }
  }
  return net
}

async function pendingFromCharges(accessToken, charges, options) {
  const today = Boolean(options && options.today)
  const paid = []
  let unpaid = 0
  for (const charge of charges) {
    if (!charge || charge.status === 'ABATE') continue
    if (charge.status === 'NORMAL' && charge.paidAt) paid.push(charge)
    else unpaid += 1
  }
  if (paid.length === 0) {
    if (today) return { items: [], empty: 'today' }
    if (unpaid > 0) throw apiError(409, 'NOT_CHARGED', COPY.NOT_CHARGED)
    return { items: [], empty: 'none' }
  }
  const cache = newCache()
  cache.slips = new Map()
  const stock = await loadStock(accessToken)
  const items = []
  let sawDrug = false
  for (const charge of paid) {
    const order = await loadOrder(accessToken, charge.orderId, cache)
    const slips = await dispensesForCharge(accessToken, charge.chargeId, cache)
    const patient = await loadPatient(accessToken, charge.patientId, cache)
    const register = await loadRegister(accessToken, charge.registerId, cache)
    for (const line of charge.items) {
      const meta = drugMeta(order, line)
      if (!meta) continue
      sawDrug = true
      const chargedQty = round2(line.qty)
      const net = netForChargeItem(slips, line.chargeItemId)
      const availableQty = round2(chargedQty - net)
      if (!(availableQty > 0)) continue
      const onHand = stock.get(line.itemNo)
      const stockQty = round2(onHand ? onHand.qty : 0)
      items.push({
        chargeItemId: line.chargeItemId,
        itemNo: line.itemNo,
        itemName: line.itemName,
        spec: text(meta.spec),
        units: text(meta.units) || (onHand ? onHand.units : ''),
        chargedQty,
        dispensedQty: net,
        returnedQty: 0,
        availableQty,
        stockQty,
        stockHint: stockHint(stockQty, availableQty),
        lineDispenseStatus: lineDispenseStatus(chargedQty, net),
        registerId: charge.registerId,
        encounterId: charge.registerId,
        patientId: charge.patientId,
        patientName: patient ? patient.name : '',
        patientNo: patient && patient.patientNo ? patient.patientNo : '',
        registerNo: register ? register.registerNo : '',
        orderId: charge.orderId,
        orderNo: order ? order.orderNo : '',
        chargeId: charge.chargeId,
        squareNo: charge.squareNo,
        orderItemId: line.orderItemId,
        deptId: PHARMACY.deptId,
        deptName: PHARMACY.deptName,
      })
    }
  }
  if (items.length > 0) return { items, empty: '' }
  if (today) return { items: [], empty: 'today' }
  if (sawDrug) return { items: [], empty: 'done' }
  return { items: [], empty: 'none' }
}

function assertDispenseItem(raw) {
  const row = raw && typeof raw === 'object' ? raw : {}
  const dispenseQty = numberOr(row.dispenseQty, numberOr(row.dispensedQty, 0))
  const returnedQty = numberOr(row.returnedQty, 0)
  const status = text(row.lineDispenseStatus)
  return {
    dispenseItemId: text(row.dispenseItemId),
    orderItemId: text(row.orderItemId),
    chargeItemId: text(row.chargeItemId),
    itemNo: text(row.itemNo),
    itemName: text(row.itemName),
    units: text(row.units),
    chargedQty: numberOr(row.chargedQty, 0),
    dispensedQty: numberOr(row.dispensedQty, dispenseQty),
    returnedQty,
    availableQty: numberOr(row.availableQty, 0),
    dispenseQty,
    stockHint: text(row.stockHint),
    stockQty: numberOr(row.stockQty, 0),
    lineDispenseStatus: status === 'PARTIAL' || status === 'FULL' ? status : 'NONE',
  }
}

export function assertDispenseDto(raw) {
  const body = raw && typeof raw === 'object' ? raw : null
  if (!body || !text(body.dispenseId) || !text(body.dispenseNo)) {
    throw apiError(500, 'HTTP_ERROR', '发药响应无法读取')
  }
  return {
    dispenseId: text(body.dispenseId),
    dispenseNo: text(body.dispenseNo),
    orderId: text(body.orderId),
    chargeId: text(body.chargeId),
    registerId: text(body.registerId),
    patientId: text(body.patientId),
    status: text(body.status),
    orderStatus: text(body.orderStatus),
    returnReason: text(body.returnReason),
    operatorId: text(body.operatorId),
    items: (Array.isArray(body.items) ? body.items : []).map((item) => assertDispenseItem(item)),
    dispensedAt: text(body.dispensedAt),
  }
}

function aggregateStatus(items) {
  const statuses = items.map((item) => item.lineDispenseStatus)
  if (statuses.length > 0 && statuses.every((status) => status === 'FULL')) return 'FULL'
  if (statuses.every((status) => status === 'NONE')) return 'NONE'
  return 'PARTIAL'
}

async function presentSlip(accessToken, dto, cache) {
  const patient = await loadPatient(accessToken, dto.patientId, cache)
  const register = await loadRegister(accessToken, dto.registerId, cache)
  const order = await loadOrder(accessToken, dto.orderId, cache)
  const charge = dto.chargeId ? await loadCharge(accessToken, dto.chargeId, cache) : null
  const items = dto.items.map((item) => {
    const orderLine = order && order.items.find((row) => row.orderItemId === item.orderItemId || row.itemNo === item.itemNo)
    const catalog = findCatalogItem(item.itemNo)
    return {
      ...item,
      spec: (orderLine && orderLine.spec) || (catalog && catalog.spec) || '',
      units: item.units || (orderLine && orderLine.units) || (catalog && catalog.units) || '',
      returnableQty: round2(item.dispenseQty - item.returnedQty),
    }
  })
  return {
    dispenseId: dto.dispenseId,
    dispenseNo: dto.dispenseNo,
    operateAt: dto.dispensedAt,
    operatorId: dto.operatorId,
    operatorName: '',
    deptId: PHARMACY.deptId,
    deptName: PHARMACY.deptName,
    patientId: dto.patientId,
    patientName: patient ? patient.name : '',
    patientNo: patient && patient.patientNo ? patient.patientNo : '',
    registerId: dto.registerId,
    registerNo: register ? register.registerNo : '',
    chargeId: dto.chargeId,
    squareNo: charge ? charge.squareNo : '',
    orderId: dto.orderId,
    orderNo: order ? order.orderNo : '',
    items,
    summary: items.map((item) => `${item.itemName} × ${formatQty(item.dispenseQty)}`).join('、'),
    lineDispenseStatus: aggregateStatus(items),
    status: dto.status,
  }
}

async function normalizeRealItems(accessToken, input) {
  const raw = input && Array.isArray(input.items) ? input.items : []
  const items = []
  for (const item of raw) {
    if (!item || item.selected === false) continue
    const dispenseQty = round2(item.dispenseQty)
    if (!(dispenseQty > 0)) continue
    let orderId = text(item.orderId)
    const chargeId = text(item.chargeId)
    const orderItemId = text(item.orderItemId)
    const chargeItemId = text(item.chargeItemId)
    if (!chargeItemId && !orderItemId) continue
    if (!orderId && chargeId) {
      const body = await getJson(`/charges/${encodeURIComponent(chargeId)}`, accessToken)
      if (body) orderId = text(assertChargeDto(body).orderId)
    }
    if (!orderId) throw apiError(400, 'EMPTY_DISPENSE_ITEMS', COPY.EMPTY_DISPENSE_ITEMS)
    items.push({
      orderId,
      chargeId,
      orderItemId,
      chargeItemId,
      dispenseQty,
      selected: true,
    })
  }
  return items
}

function groupByOrder(items) {
  const groups = new Map()
  for (const item of items) {
    const key = `${item.orderId}\0${item.chargeId}`
    if (!groups.has(key)) groups.set(key, { orderId: item.orderId, chargeId: item.chargeId, items: [] })
    groups.get(key).items.push(item)
  }
  return [...groups.values()]
}

function idempotencyKey(input, index, total) {
  const given = text(input && input.idempotencyKey)
  if (total === 1) return given || crypto.randomUUID()
  if (given) return `${given}-${index + 1}`
  return crypto.randomUUID()
}

export async function searchPending(accessToken, query) {
  if (useMock()) {
    await delay(120)
    return mockSearchPending(accessToken, query)
  }
  const keyword = text(query)
  if (!keyword) return { items: [], empty: 'prompt' }
  const charges = isUuid(keyword)
    ? await chargesForUuid(accessToken, keyword)
    : await chargesForKeyword(accessToken, keyword)
  return pendingFromCharges(accessToken, charges, { today: false })
}

export async function listTodayPending(accessToken) {
  if (useMock()) {
    await delay(120)
    return mockListTodayPending(accessToken)
  }
  const charges = await listCharges(accessToken, { visitDate: todayInShanghai() })
  return pendingFromCharges(accessToken, charges, { today: true })
}

export async function confirmDispense(accessToken, input) {
  if (useMock()) {
    await delay(160)
    return mockConfirmDispense(accessToken, input)
  }
  const requested = await normalizeRealItems(accessToken, input)
  if (requested.length === 0) throw apiError(400, 'EMPTY_DISPENSE_ITEMS', COPY.EMPTY_DISPENSE_ITEMS)
  const groups = groupByOrder(requested)
  const cache = newCache()
  const slips = []
  for (let index = 0; index < groups.length; index += 1) {
    const group = groups[index]
    const payload = {
      orderId: group.orderId,
      items: group.items.map((item) => ({
        orderItemId: item.orderItemId || undefined,
        chargeItemId: item.chargeItemId || undefined,
        dispenseQty: item.dispenseQty,
        selected: true,
      })),
    }
    if (group.chargeId) payload.chargeId = group.chargeId
    const body = await postJson(
      DISPENSE_ROUTES.collection,
      accessToken,
      payload,
      idempotencyKey(input, index, groups.length),
      201,
    )
    slips.push(await presentSlip(accessToken, assertDispenseDto(body), cache))
  }
  return { ...slips[0], dispenses: slips }
}

export async function listDispenses(accessToken, input = {}) {
  if (useMock()) {
    await delay(80)
    return mockListDispenses(accessToken, input)
  }
  const today = input.today !== false
  const patientText = text(input.patient)
  const dispenseNo = text(input.dispenseNo).toLowerCase()
  const params = new URLSearchParams()
  if (today) params.set('visitDate', todayInShanghai())
  else {
    params.set('from', RANGE_FROM)
    params.set('to', todayInShanghai())
  }
  let dtos = []
  if (patientText && isUuid(patientText)) {
    params.set('patientId', patientText)
    dtos = await fetchDispenseList(accessToken, params)
  } else if (patientText && !today) {
    const patients = await findPatients(accessToken, patientText)
    if (patients.length === 0) return { items: [] }
    for (const patient of patients) {
      const query = new URLSearchParams(params)
      query.set('patientId', patient.patientId)
      dtos.push(...await fetchDispenseList(accessToken, query))
    }
    dtos = dedupeBy(dtos, 'dispenseId')
  } else {
    dtos = await fetchDispenseList(accessToken, params)
  }
  const cache = newCache()
  const items = []
  for (const dto of dtos) items.push(await presentSlip(accessToken, dto, cache))
  const filtered = items.filter((item) => {
    if (today && dayOf(item.operateAt) !== todayInShanghai()) return false
    if (dispenseNo && !String(item.dispenseNo || '').toLowerCase().includes(dispenseNo)) return false
    if (patientText && !isUuid(patientText)) {
      const blob = `${item.patientName}\n${item.patientNo}\n${item.patientId}`.toLowerCase()
      if (!blob.includes(patientText.toLowerCase())) return false
    }
    return true
  })
  filtered.sort((left, right) => String(right.operateAt).localeCompare(String(left.operateAt)))
  return { items: filtered }
}

export async function fetchDispense(accessToken, dispenseId) {
  if (useMock()) {
    await delay(80)
    return mockGetDispense(accessToken, dispenseId)
  }
  const id = text(dispenseId)
  if (!id) throw apiError(404, 'NOT_FOUND', '未找到发药单')
  const response = await callJson(DISPENSE_ROUTES.item(id), {
    method: 'GET',
    headers: { Accept: JSON_ACCEPT, ...bearer(accessToken) },
  })
  if (response.status === 404) throw apiError(404, 'NOT_FOUND', '未找到发药单')
  if (!response.ok) throw presentDispenseError(await readErrorResponse(response))
  return presentSlip(accessToken, assertDispenseDto(await parseJson(response)), newCache())
}

function normalizeReturnItems(input) {
  const raw = input && Array.isArray(input.items) ? input.items : []
  const items = []
  for (const item of raw) {
    if (!item || item.selected === false) continue
    const returnQty = round2(item.returnQty)
    if (!(returnQty > 0)) continue
    const dispenseItemId = text(item.dispenseItemId)
    if (!dispenseItemId) throw apiError(409, 'NOT_DISPENSED', COPY.NOT_DISPENSED)
    items.push({ dispenseItemId, returnQty })
  }
  return items
}

export async function returnDispense(accessToken, input) {
  if (useMock()) {
    await delay(160)
    return mockReturnDispense(accessToken, input)
  }
  const dispenseId = text(input && input.dispenseId)
  if (!dispenseId) throw apiError(409, 'NOT_DISPENSED', COPY.NOT_DISPENSED)
  const reason = text(input && input.reason)
  if (!reason) throw apiError(400, 'RETURN_REASON_REQUIRED', COPY.RETURN_REASON_REQUIRED)
  const items = normalizeReturnItems(input)
  if (items.length === 0) throw apiError(400, 'VALIDATION_ERROR', '没有可退明细（未发或已退完）')
  const total = items.reduce((sum, item) => round2(sum + item.returnQty), 0)
  const body = await postJson(
    DISPENSE_ROUTES.return(dispenseId),
    accessToken,
    { reason, items },
    text(input.idempotencyKey) || crypto.randomUUID(),
    200,
  )
  const slip = await presentSlip(accessToken, assertDispenseDto(body), newCache())
  return {
    ...slip,
    reason,
    returnedQty: total,
    stockNote: `库存已回加 ${formatQty(total)}`,
  }
}

export async function resetDispenseDemo(accessToken) {
  if (!useMock()) return
  resetMockDispenses(accessToken)
}
