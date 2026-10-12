// 2026-10-12 P1-H05-ORDER-UI
// 对齐 OpenAPI 0.3.7：主字段 registerId（encounterId 仅别名），状态 SAVE|COMMIT|PROC|ABATE。
// 目录与 004_order 种子一致。库存不足只提示，不写入拒绝。

import { apiError } from './errors.js'

export const CATALOG = [
  {
    itemNo: 'DRUG-A',
    itemName: '演示阿莫西林胶囊',
    itemType: 'DRUG',
    spec: '0.25g×24',
    units: '盒',
    price: 10,
    stockQty: 20,
  },
  {
    itemNo: 'DRUG-B',
    itemName: '演示布洛芬片',
    itemType: 'DRUG',
    spec: '0.2g×20',
    units: '盒',
    price: 8,
    stockQty: 1,
  },
  {
    itemNo: 'TREAT-1',
    itemName: '演示诊查费',
    itemType: 'TREAT',
    spec: '',
    units: '次',
    price: 15,
    stockQty: null,
  },
]

export const USAGE_TEXTS = ['口服', '外用']
export const FREQUENCY_TEXTS = ['一日三次', '一日两次']
export const RECIPE_SORT = 'NORMAL'

const ORDER_STATUS = new Set(['SAVE', 'COMMIT', 'PROC', 'ABATE'])
const ORDERABLE_STATUS = new Set(['NORMAL', 'USED', 'DIAGNOSING', 'DIAGNOSED', 'PAUSED'])

export function round2(value) {
  const number = Number(value)
  if (!Number.isFinite(number)) return 0
  return Math.round((number + Number.EPSILON) * 100) / 100
}

export function money(value) {
  return round2(value).toFixed(2)
}

export function findCatalogItem(itemNo) {
  const key = asText(itemNo)
  return CATALOG.find((item) => item.itemNo === key) || null
}

export function orderStatusLabel(status) {
  if (status === 'SAVE') return '暂存'
  if (status === 'COMMIT') return '已提交'
  if (status === 'ABATE') return '已作废'
  if (status === 'PROC') return '已执行'
  return '未保存'
}

export function registerIsOrderable(register) {
  return Boolean(register && ORDERABLE_STATUS.has(register.status))
}

export function stockWarnings(items) {
  const warnings = []
  for (const item of Array.isArray(items) ? items : []) {
    const catalog = findCatalogItem(item && item.itemNo)
    if (!catalog || catalog.stockQty == null) continue
    const amount = Number(item.amount)
    if (Number.isFinite(amount) && amount > catalog.stockQty) {
      warnings.push(`${catalog.itemName} 可用 ${catalog.stockQty}，本次 ${amount}。库存不足，开单不拦截。`)
    }
  }
  return warnings
}

function asText(value) {
  return typeof value === 'string' ? value.trim() : ''
}

function amountBlank(value) {
  return value == null || asText(String(value)) === ''
}

export function resolveRegisterId(source) {
  const body = source && typeof source === 'object' ? source : {}
  const registerId = asText(body.registerId)
  const encounterId = asText(body.encounterId)
  if (registerId && encounterId && registerId !== encounterId) {
    throw apiError(400, 'VALIDATION_ERROR', 'registerId 与 encounterId 不一致')
  }
  return registerId || encounterId
}

function normalizeItems(rows) {
  const items = []
  for (const row of Array.isArray(rows) ? rows : []) {
    const current = row && typeof row === 'object' ? row : {}
    const itemNo = asText(current.itemNo)
    const usageText = asText(current.usageText)
    const frequencyText = asText(current.frequencyText)
    const dosage = asText(current.dosage)
    const dosageUnit = asText(current.dosageUnit)
    const blank = !itemNo && amountBlank(current.amount) && !usageText && !frequencyText && !dosage && !dosageUnit
    if (blank) continue
    if (!itemNo) throw apiError(400, 'VALIDATION_ERROR', '请选择项目')
    const catalog = findCatalogItem(itemNo)
    if (!catalog) throw apiError(409, 'CATALOG_ITEM_NOT_FOUND', '项目不在目录中')
    const amount = typeof current.amount === 'number' ? current.amount : Number(asText(String(current.amount ?? '')))
    if (!Number.isFinite(amount) || amount <= 0) throw apiError(409, 'ORDER_INVALID_AMOUNT', '数量必须大于 0')
    if (amount > 9999) throw apiError(409, 'ORDER_INVALID_AMOUNT', '数量过大')
    if (usageText.length > 40 || frequencyText.length > 40 || dosage.length > 40 || dosageUnit.length > 20) {
      throw apiError(400, 'VALIDATION_ERROR', '用法、频次或剂量过长')
    }
    const item = {
      lineNo: items.length + 1,
      itemNo: catalog.itemNo,
      itemType: catalog.itemType,
      itemName: catalog.itemName,
      units: catalog.units,
      amount,
      price: catalog.price,
      totalFee: round2(amount * catalog.price),
    }
    if (catalog.spec) item.spec = catalog.spec
    if (usageText) item.usageText = usageText
    if (frequencyText) item.frequencyText = frequencyText
    if (dosage) item.dosage = dosage
    if (dosageUnit) item.dosageUnit = dosageUnit
    const procDepartmentId = asText(current.procDepartmentId)
    if (procDepartmentId) item.procDepartmentId = procDepartmentId
    items.push(item)
  }
  return items
}

function headerFields(source, request) {
  const patientId = asText(source.patientId)
  const departmentId = asText(source.departmentId)
  const doctorId = asText(source.doctorId)
  const diagnoseName = asText(source.diagnoseName)
  const recipeSort = asText(source.recipeSort)
  if (patientId) request.patientId = patientId
  if (departmentId) request.departmentId = departmentId
  if (doctorId) request.doctorId = doctorId
  if (diagnoseName) request.diagnoseName = diagnoseName
  if (diagnoseName.length > 80) throw apiError(400, 'VALIDATION_ERROR', '诊断简述过长')
  if (recipeSort) request.recipeSort = recipeSort
  return request
}

export function assertCreateOrderRequest(input) {
  const source = input && typeof input === 'object' ? input : {}
  const registerId = resolveRegisterId(source)
  if (!registerId) throw apiError(409, 'ENCOUNTER_REQUIRED', '请先挂号')
  const request = headerFields(source, {
    registerId,
    recipeSort: RECIPE_SORT,
    items: normalizeItems(source.items),
  })
  if (asText(source.recipeSort)) request.recipeSort = asText(source.recipeSort)
  return request
}

export function assertPatchOrderRequest(input) {
  const source = input && typeof input === 'object' ? input : {}
  const request = headerFields(source, {})
  if (Object.prototype.hasOwnProperty.call(source, 'diagnoseName')) {
    const diagnoseName = asText(source.diagnoseName)
    if (diagnoseName.length > 80) throw apiError(400, 'VALIDATION_ERROR', '诊断简述过长')
    request.diagnoseName = diagnoseName || null
  }
  if (Object.prototype.hasOwnProperty.call(source, 'items')) request.items = normalizeItems(source.items)
  const registerId = asText(source.registerId)
  const encounterId = asText(source.encounterId)
  if (registerId && encounterId && registerId !== encounterId) {
    throw apiError(400, 'VALIDATION_ERROR', 'registerId 与 encounterId 不一致')
  }
  if (registerId) request.registerId = registerId
  else if (encounterId) request.registerId = encounterId
  return request
}

function assertItemDto(raw, index) {
  const item = raw && typeof raw === 'object' ? raw : {}
  const itemNo = asText(item.itemNo)
  const itemType = asText(item.itemType)
  const itemName = asText(item.itemName)
  const amount = Number(item.amount)
  const price = Number(item.price)
  if (!itemNo || !itemName || (itemType !== 'DRUG' && itemType !== 'TREAT')) {
    throw apiError(500, 'HTTP_ERROR', '处方明细无法读取')
  }
  if (!Number.isFinite(amount) || amount <= 0 || !Number.isFinite(price) || price < 0) {
    throw apiError(500, 'HTTP_ERROR', '处方明细无法读取')
  }
  const dto = {
    orderItemId: asText(item.orderItemId),
    lineNo: Number.isInteger(item.lineNo) ? item.lineNo : index + 1,
    itemNo,
    itemType,
    itemName,
    units: asText(item.units),
    amount,
    price,
    totalFee: item.totalFee == null ? round2(amount * price) : round2(item.totalFee),
  }
  if (asText(item.spec)) dto.spec = asText(item.spec)
  if (asText(item.usageText)) dto.usageText = asText(item.usageText)
  if (asText(item.frequencyText)) dto.frequencyText = asText(item.frequencyText)
  if (asText(item.dosage)) dto.dosage = asText(item.dosage)
  if (asText(item.dosageUnit)) dto.dosageUnit = asText(item.dosageUnit)
  if (asText(item.procDepartmentId)) dto.procDepartmentId = asText(item.procDepartmentId)
  return dto
}

export function assertOrderDto(raw) {
  const order = raw && typeof raw === 'object' ? raw : null
  if (!order) throw apiError(500, 'HTTP_ERROR', '处方响应无法读取')
  const orderId = asText(order.orderId)
  const registerId = asText(order.registerId || order.encounterId)
  const status = asText(order.status)
  const patientId = asText(order.patientId)
  const departmentId = asText(order.departmentId)
  const doctorId = asText(order.doctorId)
  if (!orderId || !asText(order.orderNo) || !registerId || !patientId || !departmentId || !doctorId || !ORDER_STATUS.has(status)) {
    throw apiError(500, 'HTTP_ERROR', '处方响应无法读取')
  }
  if (typeof order.charged !== 'boolean' || typeof order.totalFee !== 'number') {
    throw apiError(500, 'HTTP_ERROR', '处方响应无法读取')
  }
  const items = (Array.isArray(order.items) ? order.items : []).map((item, index) => assertItemDto(item, index))
  return {
    orderId,
    orderNo: asText(order.orderNo),
    registerId,
    encounterId: registerId,
    patientId,
    departmentId,
    doctorId,
    diagnoseName: asText(order.diagnoseName) || null,
    recipeSort: asText(order.recipeSort) || RECIPE_SORT,
    status,
    totalFee: round2(order.totalFee),
    charged: order.charged,
    items,
    submittedAt: asText(order.submittedAt) || null,
    abatedAt: asText(order.abatedAt) || null,
    abatedBy: asText(order.abatedBy) || null,
  }
}
