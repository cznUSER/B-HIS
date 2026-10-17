// 2026-10-13 P1-H05-MASTER-UI 基础数据客户端。
// 2026-10-13 真接口：VITE_USE_MOCK=false 走 OpenAPI 0.3.10-draft / 已合 Go handlers。
// 路径：/departments、/doctors、/catalog-items、/inventory/items + adjust/ledger/batches、/dicts、/users。

import { apiUrl, useMock } from '../api/config.js'
import { apiError, readErrorResponse } from '../api/errors.js'
import {
  adjustInventory as adjustInventoryMock,
  createCatalogItem,
  createDepartment,
  createDictItem,
  createDoctor,
  createUser,
  listCatalogItems,
  listDepartments,
  listDictItems,
  listDictOverview,
  listDoctors,
  listInventoryItems,
  listLedger,
  listUsers,
  masterSummary,
  patchCatalogItem,
  patchDepartment,
  patchDictItem,
  patchDoctor,
  patchUser,
  resetUserPassword as resetUserPasswordMock,
} from './store.js'

// The access catalogue is deliberately kept in this client module for demo mode.
// The live branch uses the same response envelopes as the Go access-control API.
const MOCK_ACCESS = {
  modules: [
    { id: 'module-outpatient', code: 'outpatient', name: '门诊业务', description: '', status: 'ACTIVE', hidden: false },
    { id: 'module-master', code: 'master', name: '系统设置', description: '', status: 'ACTIVE', hidden: false },
  ],
  categories: [
    { id: 'patient', code: 'patient', name: '患者与档案', description: '', status: 'ACTIVE' },
    { id: 'order', code: 'order', name: '医嘱与开单', description: '', status: 'ACTIVE' },
    { id: 'charge', code: 'charge', name: '收费与退费', description: '', status: 'ACTIVE' },
    { id: 'admin', code: 'admin', name: '系统管理', description: '', status: 'ACTIVE' },
  ],
  modes: [
    { code: 'password', name: '密码', status: 'ACTIVE' },
    { code: 'card', name: '工卡（预留）', status: 'INACTIVE' },
  ],
  permissions: [
    { code: 'patient:create', name: '患者建档', moduleCode: 'outpatient', categoryId: 'patient', modeCode: 'password', secondPermissionCode: '', actions: ['N', 'S'], status: 'ACTIVE', hidden: false },
    { code: 'encounter:create', name: '创建就诊', moduleCode: 'outpatient', categoryId: 'patient', modeCode: 'password', secondPermissionCode: '', actions: ['N', 'S'], status: 'ACTIVE', hidden: false },
    { code: 'diagnose:write', name: '诊断录入', description: '保存、提交和作废门诊诊断', moduleCode: 'outpatient', categoryId: 'patient', modeCode: 'password', secondPermissionCode: '', actions: ['N', 'M', 'D', 'S'], status: 'ACTIVE', hidden: false },
    { code: 'order:submit', name: '提交医嘱', moduleCode: 'outpatient', categoryId: 'order', modeCode: 'password', secondPermissionCode: '', actions: ['N', 'S'], status: 'ACTIVE', hidden: false },
    { code: 'charge:pay', name: '现金收费', moduleCode: 'outpatient', categoryId: 'charge', modeCode: 'password', secondPermissionCode: '', actions: ['N', 'S'], status: 'ACTIVE', hidden: false },
    { code: 'charge:refund', name: '退费', moduleCode: 'outpatient', categoryId: 'charge', modeCode: 'password', secondPermissionCode: '', actions: ['N', 'S'], status: 'ACTIVE', hidden: false },
    { code: 'dispense:create', name: '发药', moduleCode: 'outpatient', categoryId: 'order', modeCode: 'password', secondPermissionCode: '', actions: ['N', 'S'], status: 'ACTIVE', hidden: false },
    { code: 'admin:user', name: '用户管理', moduleCode: 'master', categoryId: 'admin', modeCode: 'password', secondPermissionCode: '', actions: ['N', 'M', 'D', 'S'], status: 'ACTIVE', hidden: false },
    { code: 'admin:role', name: '角色管理', moduleCode: 'master', categoryId: 'admin', modeCode: 'password', secondPermissionCode: '', actions: ['N', 'M', 'D', 'S'], status: 'ACTIVE', hidden: false },
    { code: 'admin:permission', name: '权限目录', moduleCode: 'master', categoryId: 'admin', modeCode: 'password', secondPermissionCode: '', actions: ['N', 'M', 'D', 'S'], status: 'ACTIVE', hidden: false },
  ],
  roles: [
    { code: 'REGISTRAR', name: '建档挂号员', description: '患者建档与挂号', moduleCode: 'outpatient', departmentId: '', permissionCodes: ['patient:create', 'encounter:create'] , status: 'ACTIVE', hidden: false, system: false },
    { code: 'DOCTOR', name: '门诊医生', description: '门诊接诊与开单', moduleCode: 'outpatient', departmentId: '', permissionCodes: ['patient:create', 'encounter:create', 'diagnose:write', 'order:submit'], status: 'ACTIVE', hidden: false, system: false },
    { code: 'CASHIER', name: '收费员', description: '现金收费与退费', moduleCode: 'outpatient', departmentId: '', permissionCodes: ['charge:pay', 'charge:refund'], status: 'ACTIVE', hidden: false, system: false },
    { code: 'PHARMACIST', name: '药师', description: '药房发药', moduleCode: 'outpatient', departmentId: '', permissionCodes: ['dispense:create'], status: 'ACTIVE', hidden: false, system: false },
    { code: 'SYSTEM', name: '系统管理员', description: '系统设置与权限', moduleCode: 'master', departmentId: '', permissionCodes: ['admin:user', 'admin:role', 'admin:permission'], status: 'ACTIVE', hidden: false, system: true },
  ],
}
const MOCK_USER_OVERRIDES = new Map()
const MOCK_USER_ROLE_SCOPES = new Map()

// Resolve the same effective permission shape as the live access runtime so a
// Mock relogin reflects changes made in the users-and-permissions workspace.
export function mockAccessForUser(user = {}) {
  const assignedRoles = new Set(Array.isArray(user.roles) ? user.roles.map((role) => text(role).toUpperCase()) : [])
  const configuredScopes = Array.isArray(MOCK_USER_ROLE_SCOPES.get(user.id))
    ? cloneAccess(MOCK_USER_ROLE_SCOPES.get(user.id))
    : []
  const explicitRoles = new Set(configuredScopes.map((scope) => text(scope && scope.roleCode).toUpperCase()))
  const activeDepartmentIds = new Set(listDepartments({ status: 'ACTIVE' }).items.map((item) => item.departmentId))
  const grants = []
  const addGrant = (permissionCode, moduleCode, departmentId) => {
    if (!permissionCode) return
    if (!grants.some((grant) => grant.permissionCode === permissionCode && grant.moduleCode === moduleCode && grant.departmentId === departmentId)) {
      grants.push({ permissionCode, moduleCode, departmentId })
    }
  }
  for (const role of MOCK_ACCESS.roles) {
    if (!assignedRoles.has(role.code) || role.status !== 'ACTIVE') continue
    const roleScopes = configuredScopes.filter((scope) => text(scope && scope.roleCode).toUpperCase() === role.code)
    const departments = explicitRoles.has(role.code)
      ? roleScopes.filter((scope) => text(scope && scope.status).toUpperCase() === 'ACTIVE' && activeDepartmentIds.has(text(scope.departmentId))).map((scope) => text(scope.departmentId))
      : (!role.departmentId || activeDepartmentIds.has(role.departmentId) ? [role.departmentId || ''] : [])
    const permissionCodes = role.code === 'SYSTEM' ? [...new Set([...(role.permissionCodes || []), 'admin:*'])] : (role.permissionCodes || [])
    for (const departmentId of departments) {
      for (const permissionCode of permissionCodes) addGrant(text(permissionCode), role.moduleCode || '', departmentId)
    }
  }
  const overrides = Array.isArray(MOCK_USER_OVERRIDES.get(user.id)) ? cloneAccess(MOCK_USER_OVERRIDES.get(user.id)) : []
  const deniedPermissions = new Set()
  const scopeMatches = (grant, override) => (
    (!text(override.moduleCode) || text(override.moduleCode) === grant.moduleCode)
    && (!text(override.departmentId) || text(override.departmentId) === grant.departmentId)
  )
  for (const override of overrides) {
    const permissionCode = text(override && override.permissionCode)
    const effect = text(override && override.effect).toUpperCase()
    if (!permissionCode) continue
    if (effect === 'DENY') {
      deniedPermissions.add(permissionCode)
      for (let index = grants.length - 1; index >= 0; index -= 1) {
        if (grants[index].permissionCode === permissionCode && scopeMatches(grants[index], override)) grants.splice(index, 1)
      }
    } else if (effect === 'ALLOW') {
      addGrant(permissionCode, text(override.moduleCode), text(override.departmentId))
    }
  }
  return {
    permissions: [...new Set(grants.map((grant) => grant.permissionCode))].filter((code) => !deniedPermissions.has(code)).sort(),
    scopes: [...new Set(grants.filter((grant) => grant.moduleCode || grant.departmentId).map((grant) => `${grant.moduleCode}:${grant.departmentId}`))].sort(),
    deniedPermissions: [...deniedPermissions].sort(),
  }
}

// Batch/expiry rows are a read-only fixture. They deliberately do not share the
// mutable inventory store so the batch query cannot change total stock behavior.
const MOCK_INVENTORY_BATCHES = [
  {
    batchId: 'P2-6A-EXPIRED', skuId: 'sku-drug-a', itemNo: 'DRUG-A', itemName: '演示阿莫西林胶囊',
    storageUnitId: 'fixture-pharmacy', batchNo: 'B-EXPIRED', productNo: null, supplierId: null,
    productDate: '2025-01-01', expireDate: '2026-10-14', qtyOnHand: '3.000', units: '盒',
    sourceRef: 'FIXTURE-P2-6A', expiryStatus: 'EXPIRED', rowVersion: 'fixture-1',
  },
  {
    batchId: 'P2-6A-EXPIRING', skuId: 'sku-drug-a', itemNo: 'DRUG-A', itemName: '演示阿莫西林胶囊',
    storageUnitId: 'fixture-pharmacy', batchNo: 'B-EXPIRING', productNo: null, supplierId: null,
    productDate: '2026-01-01', expireDate: '2026-10-15', qtyOnHand: '1.000', units: '盒',
    sourceRef: 'FIXTURE-P2-6A', expiryStatus: 'EXPIRING', rowVersion: 'fixture-2',
  },
  {
    batchId: 'P2-6A-EXPIRING-END', skuId: 'sku-drug-a', itemNo: 'DRUG-A', itemName: '演示阿莫西林胶囊',
    storageUnitId: 'fixture-pharmacy', batchNo: 'B-EXPIRING-END', productNo: null, supplierId: null,
    productDate: '2026-01-01', expireDate: '2026-11-14', qtyOnHand: '4.000', units: '盒',
    sourceRef: 'FIXTURE-P2-6A', expiryStatus: 'EXPIRING', rowVersion: 'fixture-3',
  },
  {
    batchId: 'P2-6A-VALID', skuId: 'sku-drug-a', itemNo: 'DRUG-A', itemName: '演示阿莫西林胶囊',
    storageUnitId: 'fixture-pharmacy', batchNo: 'B-VALID', productNo: null, supplierId: null,
    productDate: '2026-01-01', expireDate: '2026-11-15', qtyOnHand: '10.000', units: '盒',
    sourceRef: 'FIXTURE-P2-6A', expiryStatus: 'VALID', rowVersion: 'fixture-4',
  },
  {
    batchId: 'P2-6A-UNKNOWN', skuId: 'sku-drug-a', itemNo: 'DRUG-A', itemName: '演示阿莫西林胶囊',
    storageUnitId: 'fixture-pharmacy', batchNo: 'B-UNKNOWN', productNo: null, supplierId: null,
    productDate: null, expireDate: null, qtyOnHand: '7.000', units: '盒',
    sourceRef: 'FIXTURE-P2-6A', expiryStatus: 'UNKNOWN', rowVersion: 'fixture-5',
  },
]

function accessList(body, label = '权限数据') {
  if (!body || typeof body !== 'object' || !Array.isArray(body.items)) {
    throw apiError(502, 'HTTP_ERROR', `${label}不符合契约`)
  }
  return { items: body.items }
}

const JSON_ACCEPT = 'application/json'

function requestInit(init) {
  return {
    credentials: 'omit',
    cache: 'no-store',
    ...init,
  }
}

function bearer(accessToken) {
  return accessToken ? { Authorization: `Bearer ${accessToken}` } : {}
}

function text(value) {
  return typeof value === 'string' ? value.trim() : ''
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

function asList(body) {
  if (!body || typeof body !== 'object' || !Array.isArray(body.items)) {
    throw apiError(502, 'HTTP_ERROR', '基础数据不符合契约')
  }
  return { items: body.items }
}

function asObject(body) {
  if (!body || typeof body !== 'object' || Array.isArray(body)) {
    throw apiError(502, 'HTTP_ERROR', '基础数据不符合契约')
  }
  return body
}

function withQuery(path, params) {
  const search = new URLSearchParams()
  for (const [key, value] of Object.entries(params || {})) {
    if (value == null) continue
    const token = String(value).trim()
    if (!token) continue
    search.set(key, token)
  }
  const query = search.toString()
  return query ? `${path}?${query}` : path
}

function ledgerDateBoundary(value, exclusiveNextDay = false) {
  const dateText = text(value)
  if (!dateText) return null
  if (!/^\d{4}-\d{2}-\d{2}$/.test(dateText)) throw apiError(400, 'VALIDATION_ERROR', '日期格式必须为 YYYY-MM-DD')
  const date = new Date(`${dateText}T00:00:00+08:00`)
  if (Number.isNaN(date.getTime())) throw apiError(400, 'VALIDATION_ERROR', '日期无效')
  const shanghaiDate = new Date(date.getTime() + 8 * 60 * 60 * 1000).toISOString().slice(0, 10)
  if (shanghaiDate !== dateText) throw apiError(400, 'VALIDATION_ERROR', '日期无效')
  return date.getTime() + (exclusiveNextDay ? 24 * 60 * 60 * 1000 : 0)
}

function ledgerRange(query) {
  const from = ledgerDateBoundary(query.from)
  const to = ledgerDateBoundary(query.to, true)
  if (from != null && to != null && from >= to) throw apiError(400, 'VALIDATION_ERROR', '开始日期不能晚于结束日期')
  return { from, to }
}

function ledgerLimit(value) {
  if (value == null || value === '') return 50
  const limit = Number(value)
  if (!Number.isInteger(limit) || limit < 1) throw apiError(400, 'VALIDATION_ERROR', '每页条数必须是正整数')
  return Math.min(limit, 200)
}

function encodeMockCursor(offset) {
  return `mock-${btoa(`offset:${offset}`).replaceAll('+', '-').replaceAll('/', '_').replaceAll('=', '')}`
}

function decodeMockCursor(cursor) {
  if (!cursor) return 0
  try {
    const encoded = cursor.startsWith('mock-') ? cursor.slice(5) : ''
    const base64 = encoded.replaceAll('-', '+').replaceAll('_', '/')
    const decoded = atob(base64.padEnd(Math.ceil(base64.length / 4) * 4, '='))
    const match = /^offset:(\d+)$/.exec(decoded)
    if (!match) throw new Error('invalid cursor')
    return Number(match[1])
  } catch {
    throw apiError(400, 'VALIDATION_ERROR', '流水游标无效')
  }
}

function mockLedgerPage(skuCode, query) {
  const inventory = listInventoryItems({ skuCode }).items.find((item) => item.skuCode === skuCode || item.itemNo === skuCode)
  if (!inventory) throw apiError(404, 'NOT_FOUND', '未找到库存')
  const bizType = text(query.bizType)
  if (bizType && !['ADJUST', 'DISPENSE', 'RETURN'].includes(bizType)) throw apiError(400, 'VALIDATION_ERROR', '库存流水类型无效')
  const { from, to } = ledgerRange(query)
  const limit = ledgerLimit(query.limit)
  const offset = decodeMockCursor(text(query.cursor))
  const items = listLedger(inventory.skuId).items
    .filter((item) => !bizType || item.bizType === bizType)
    .filter((item) => {
      const createdAt = new Date(item.createdAt).getTime()
      return Number.isFinite(createdAt) && (from == null || createdAt >= from) && (to == null || createdAt < to)
    })
    .map((item) => ({
      ...item,
      skuCode: inventory.skuCode || inventory.itemNo,
      bizId: item.bizId || '',
    }))
  const page = items.slice(offset, offset + limit)
  return {
    items: page,
    nextCursor: offset + page.length < items.length ? encodeMockCursor(offset + page.length) : '',
  }
}

const BATCH_EXPIRY_STATUSES = new Set(['UNKNOWN', 'VALID', 'EXPIRING', 'EXPIRED'])

function batchDate(value, label) {
  const dateText = text(value)
  if (!dateText) return ''
  if (!/^\d{4}-\d{2}-\d{2}$/.test(dateText)) throw apiError(400, 'VALIDATION_ERROR', `${label}格式必须为 YYYY-MM-DD`)
  const date = new Date(`${dateText}T00:00:00+08:00`)
  if (Number.isNaN(date.getTime())) throw apiError(400, 'VALIDATION_ERROR', `${label}无效`)
  const normalized = new Date(date.getTime() + 8 * 60 * 60 * 1000).toISOString().slice(0, 10)
  if (normalized !== dateText) throw apiError(400, 'VALIDATION_ERROR', `${label}无效`)
  return dateText
}

function batchWindow(value) {
  if (value == null || value === '') return 30
  const number = Number(value)
  if (!Number.isInteger(number) || number < 0) throw apiError(400, 'VALIDATION_ERROR', '效期窗口必须是非负整数')
  return number
}

function batchLimit(value) {
  if (value == null || value === '') return 50
  const number = Number(value)
  if (!Number.isInteger(number) || number < 1) throw apiError(400, 'VALIDATION_ERROR', '每页条数必须是正整数')
  return Math.min(number, 200)
}

function batchCursor(cursor) {
  if (!cursor) return 0
  return decodeMockCursor(cursor)
}

function batchSummary(items) {
  const qtyTotal = items.reduce((sum, item) => sum + Number(item.qtyOnHand || 0), 0)
  const qtyUnknown = items.filter((item) => item.expiryStatus === 'UNKNOWN').reduce((sum, item) => sum + Number(item.qtyOnHand || 0), 0)
  return {
    batchCount: items.length,
    qtyTotal: qtyTotal.toFixed(3),
    qtyUnknown: qtyUnknown.toFixed(3),
    duplicateKeyCount: 0,
  }
}

function mockInventoryBatches(query) {
  const skuCode = text(query && query.skuCode).toLowerCase()
  const storageUnitId = text(query && query.storageUnitId).toLowerCase()
  const batchNo = text(query && query.batchNo).toLowerCase()
  const expiryStatus = text(query && query.expiryStatus).toUpperCase()
  if (expiryStatus && !BATCH_EXPIRY_STATUSES.has(expiryStatus)) throw apiError(400, 'VALIDATION_ERROR', '效期状态无效')
  const asOfDate = batchDate(query && query.asOfDate, '参考日期') || '2026-10-15'
  const expiryWindowDays = batchWindow(query && query.expiryWindowDays)
  const limit = batchLimit(query && query.limit)
  const offset = batchCursor(text(query && query.cursor))
  const rows = MOCK_INVENTORY_BATCHES.filter((item) => {
    if (skuCode && ![item.skuId, item.itemNo].some((value) => text(value).toLowerCase() === skuCode)) return false
    if (storageUnitId && text(item.storageUnitId).toLowerCase() !== storageUnitId) return false
    if (batchNo && text(item.batchNo).toLowerCase() !== batchNo) return false
    if (expiryStatus && item.expiryStatus !== expiryStatus) return false
    return true
  })
  const page = rows.slice(offset, offset + limit)
  return {
    items: page,
    summary: batchSummary(rows),
    asOfDate,
    expiryWindowDays,
    nextCursor: offset + page.length < rows.length ? encodeMockCursor(offset + page.length) : '',
  }
}

async function send(accessToken, method, path, { body, ok } = {}) {
  let response
  try {
    response = await fetch(apiUrl(path), requestInit({
      method,
      headers: {
        Accept: JSON_ACCEPT,
        ...(body != null ? { 'Content-Type': 'application/json; charset=utf-8' } : {}),
        ...bearer(accessToken),
      },
      body: body != null ? JSON.stringify(body) : undefined,
    }))
  } catch {
    throw apiError(0, 'HTTP_ERROR', '无法连接基础数据服务')
  }
  const allowed = ok || [200]
  if (!allowed.includes(response.status)) throw await readErrorResponse(response, accessToken)
  if (response.status === 204) return null
  return parseJson(response)
}

function optionalInt(value) {
  if (value == null || value === '') return undefined
  const number = Number(value)
  if (!Number.isFinite(number)) return undefined
  return Math.trunc(number)
}

function departmentBody(body) {
  const src = body && typeof body === 'object' ? body : {}
  const out = {}
  const deptCode = text(src.deptCode)
  const deptName = text(src.deptName)
  if (deptCode) out.deptCode = deptCode
  if (deptName) out.deptName = deptName
  if (text(src.status)) out.status = text(src.status)
  const sortNo = optionalInt(src.sortNo)
  if (sortNo != null) out.sortNo = sortNo
  return out
}

function doctorBody(body) {
  const src = body && typeof body === 'object' ? body : {}
  const out = {}
  const doctorCode = text(src.doctorCode)
  const doctorName = text(src.doctorName)
  const departmentId = text(src.departmentId)
  if (doctorCode) out.doctorCode = doctorCode
  if (doctorName) out.doctorName = doctorName
  if (departmentId) out.departmentId = departmentId
  if (text(src.status)) out.status = text(src.status)
  return out
}

function catalogBody(body, creating) {
  const src = body && typeof body === 'object' ? body : {}
  const out = {}
  for (const key of ['itemNo', 'itemName', 'units']) {
    const value = text(src[key])
    if (value) out[key] = value
  }
  if (creating && text(src.itemType)) out.itemType = text(src.itemType)
  if (src.spec != null) out.spec = text(src.spec)
  if (text(src.status)) out.status = text(src.status)
  if (src.price != null && src.price !== '') {
    const price = Number(src.price)
    if (Number.isFinite(price)) out.price = price
  }
  return out
}

function dictBody(body, creating) {
  const src = body && typeof body === 'object' ? body : {}
  const out = {}
  if (creating && text(src.code)) out.code = text(src.code)
  if (text(src.name)) out.name = text(src.name)
  if (text(src.status)) out.status = text(src.status)
  const sortNo = optionalInt(src.sortNo)
  if (sortNo != null) out.sortNo = sortNo
  return out
}

function userBody(body, creating) {
  const src = body && typeof body === 'object' ? body : {}
  const out = {}
  if (creating && text(src.username)) out.username = text(src.username)
  if (src.displayName != null) out.displayName = text(src.displayName)
  for (const key of ['description', 'inputCode']) {
    if (src[key] != null) out[key] = text(src[key])
  }
  if (src.password != null && text(src.password)) out.password = text(src.password)
  if (src.cardNo != null) out.cardNo = text(src.cardNo)
  if ('serviceStar' in src) {
    out.serviceStar = src.serviceStar == null || src.serviceStar === '' ? null : optionalInt(src.serviceStar)
  }
  for (const key of ['expandInterface1', 'expandInterface2', 'inputCodeCompareMode']) {
    const value = optionalInt(src[key])
    if (value != null) out[key] = value
  }
  if (text(src.status)) out.status = text(src.status)
  if (Array.isArray(src.roles)) {
    out.roles = src.roles.map((role) => text(role).toUpperCase()).filter(Boolean)
  }
  if (Array.isArray(src.roleScopes)) {
    out.roleScopes = src.roleScopes.map((scope) => ({
      roleCode: text(scope && scope.roleCode).toUpperCase(),
      departmentId: text(scope && scope.departmentId),
      status: text(scope && scope.status).toUpperCase(),
    }))
  }
  const lifecycleFields = creating
    ? ['mustChangePassword', 'cannotModifyPassword', 'accountStopped', 'hidden', 'system', 'super', 'developer']
    : ['mustChangePassword', 'cannotModifyPassword', 'accountLocked', 'accountStopped', 'hidden', 'system', 'super', 'developer']
  for (const key of lifecycleFields) {
    if (src[key] != null) out[key] = Boolean(src[key])
  }
  if (Array.isArray(src.permissionOverrides)) {
    out.permissionOverrides = src.permissionOverrides.map((item) => ({
      permissionCode: text(item && item.permissionCode),
      moduleCode: text(item && item.moduleCode),
      departmentId: text(item && item.departmentId),
      effect: text(item && item.effect).toUpperCase(),
    }))
  }
  return out
}

function adjustBody(body) {
  const src = body && typeof body === 'object' ? body : {}
  const out = {
    qtyDelta: Number(src.qtyDelta),
    remark: text(src.remark),
  }
  const rowVersion = text(src.rowVersion)
  if (rowVersion) out.rowVersion = rowVersion
  return out
}

function filterInventory(items, query) {
  const needle = text(query && query.q).toLowerCase()
  if (!needle) return items
  return items.filter((item) => {
    const hay = [item.itemNo, item.itemName, item.name, item.skuCode].filter(Boolean).join(' ').toLowerCase()
    return hay.includes(needle)
  })
}

export async function fetchMasterSummary(accessToken) {
  if (useMock()) {
    void accessToken
    return masterSummary()
  }
  const [departments, doctors, catalog, inventory, dicts, users] = await Promise.all([
    fetchDepartments(accessToken, {}),
    fetchDoctors(accessToken, {}),
    fetchCatalog(accessToken, {}),
    fetchInventory(accessToken, {}),
    fetchDictItems(accessToken, 'register_sort', {}),
    fetchUsers(accessToken, {}),
  ])
  return {
    departments: departments.items.length,
    doctors: doctors.items.length,
    catalog: catalog.items.length,
    inventory: inventory.items.length,
    dicts: dicts.items.length,
    users: users.items.length,
  }
}

export async function fetchDepartments(accessToken, query) {
  if (useMock()) {
    void accessToken
    return listDepartments(query)
  }
  return asList(await send(accessToken, 'GET', withQuery('/departments', {
    status: query && query.status,
    q: query && query.q,
  })))
}

export async function saveDepartment(accessToken, departmentId, body) {
  if (useMock()) {
    void accessToken
    if (departmentId) return patchDepartment(departmentId, body)
    return createDepartment(body)
  }
  const id = text(departmentId)
  if (id) {
    return asObject(await send(accessToken, 'PATCH', `/departments/${encodeURIComponent(id)}`, {
      body: departmentBody(body),
    }))
  }
  return asObject(await send(accessToken, 'POST', '/departments', {
    body: departmentBody(body),
    ok: [201],
  }))
}

export async function fetchDoctors(accessToken, query) {
  if (useMock()) {
    void accessToken
    return listDoctors(query)
  }
  return asList(await send(accessToken, 'GET', withQuery('/doctors', {
    departmentId: query && query.departmentId,
    status: query && query.status,
    q: query && query.q,
  })))
}

export async function saveDoctor(accessToken, doctorId, body) {
  if (useMock()) {
    void accessToken
    if (doctorId) return patchDoctor(doctorId, body)
    return createDoctor(body)
  }
  const id = text(doctorId)
  if (id) {
    return asObject(await send(accessToken, 'PATCH', `/doctors/${encodeURIComponent(id)}`, {
      body: doctorBody(body),
    }))
  }
  return asObject(await send(accessToken, 'POST', '/doctors', {
    body: doctorBody(body),
    ok: [201],
  }))
}

export async function fetchCatalog(accessToken, query) {
  if (useMock()) {
    void accessToken
    return listCatalogItems(query)
  }
  return asList(await send(accessToken, 'GET', withQuery('/catalog-items', {
    itemType: query && query.itemType,
    status: query && query.status,
    q: query && query.q,
  })))
}

export async function saveCatalogItem(accessToken, catalogItemId, body) {
  if (useMock()) {
    void accessToken
    if (catalogItemId) return { item: patchCatalogItem(catalogItemId, body), inventoryCreated: false }
    return createCatalogItem(body)
  }
  const id = text(catalogItemId)
  if (id) {
    const item = asObject(await send(accessToken, 'PATCH', `/catalog-items/${encodeURIComponent(id)}`, {
      body: catalogBody(body, false),
    }))
    return { item, inventoryCreated: false }
  }
  const item = asObject(await send(accessToken, 'POST', '/catalog-items', {
    body: catalogBody(body, true),
    ok: [201],
  }))
  return { item, inventoryCreated: item.itemType === 'DRUG' }
}

export async function fetchInventory(accessToken, query) {
  if (useMock()) {
    void accessToken
    return listInventoryItems(query)
  }
  const params = {}
  const skuCode = text(query && query.skuCode)
  const itemNo = text(query && query.itemNo)
  if (skuCode && itemNo && skuCode !== itemNo) {
    throw apiError(400, 'VALIDATION_ERROR', '药品编码不一致')
  }
  if (skuCode) params.skuCode = skuCode
  else if (itemNo) params.itemNo = itemNo
  const listed = asList(await send(accessToken, 'GET', withQuery('/inventory/items', params)))
  return { items: filterInventory(listed.items, query) }
}

export async function fetchInventoryBatches(accessToken, query = {}) {
  const asOfDate = batchDate(query.asOfDate, '参考日期')
  if (!asOfDate) throw apiError(400, 'VALIDATION_ERROR', '参考日期不能为空')
  const params = {
    skuCode: text(query.skuCode),
    storageUnitId: text(query.storageUnitId),
    batchNo: text(query.batchNo),
    expiryStatus: text(query.expiryStatus).toUpperCase(),
    asOfDate,
    expiryWindowDays: batchWindow(query.expiryWindowDays),
    cursor: text(query.cursor),
    limit: batchLimit(query.limit),
  }
  if (params.expiryStatus && !BATCH_EXPIRY_STATUSES.has(params.expiryStatus)) {
    throw apiError(400, 'VALIDATION_ERROR', '效期状态无效')
  }
  if (useMock()) {
    void accessToken
    return mockInventoryBatches(params)
  }
  const body = asObject(await send(accessToken, 'GET', withQuery('/inventory/batches', params)))
  if (!Array.isArray(body.items) || !body.summary || typeof body.summary !== 'object') {
    throw apiError(502, 'HTTP_ERROR', '库存批次不符合契约')
  }
  return {
    items: body.items,
    summary: body.summary,
    asOfDate: text(body.asOfDate) || params.asOfDate,
    expiryWindowDays: Number.isInteger(Number(body.expiryWindowDays)) ? Number(body.expiryWindowDays) : params.expiryWindowDays,
    nextCursor: text(body.nextCursor),
  }
}

export async function fetchLedger(accessToken, skuCode, query = {}) {
  const code = text(skuCode)
  if (!code) throw apiError(400, 'VALIDATION_ERROR', '请提供药品编码')
  const filters = {
    bizType: text(query.bizType),
    from: text(query.from),
    to: text(query.to),
    cursor: text(query.cursor),
    limit: ledgerLimit(query.limit),
  }
  ledgerRange(filters)
  if (useMock()) {
    void accessToken
    return mockLedgerPage(code, filters)
  }
  const body = asObject(await send(accessToken, 'GET', withQuery('/inventory/ledger', {
    skuCode: code,
    ...filters,
  })))
  if (!Array.isArray(body.items) || typeof body.nextCursor !== 'string') {
    throw apiError(502, 'HTTP_ERROR', '库存流水不符合契约')
  }
  return { items: body.items, nextCursor: body.nextCursor || '' }
}

export async function adjustInventory(accessToken, skuId, body) {
  if (useMock()) {
    void accessToken
    return adjustInventoryMock(skuId, body)
  }
  const id = text(skuId)
  if (!id) throw apiError(400, 'VALIDATION_ERROR', '未找到库存')
  return asObject(await send(accessToken, 'POST', `/inventory/items/${encodeURIComponent(id)}/adjust`, {
    body: adjustBody(body),
  }))
}

export async function fetchDictOverview(accessToken) {
  if (useMock()) {
    void accessToken
    return listDictOverview()
  }
  const body = asObject(await send(accessToken, 'GET', '/dicts'))
  if (!Array.isArray(body.readonlyEnums) || !Array.isArray(body.maintainableTypes)) {
    throw apiError(502, 'HTTP_ERROR', '基础数据不符合契约')
  }
  return body
}

export async function fetchDictItems(accessToken, dictType, query) {
  if (useMock()) {
    void accessToken
    return listDictItems(dictType, query)
  }
  const type = text(dictType)
  return asList(await send(accessToken, 'GET', withQuery(`/dicts/${encodeURIComponent(type)}/items`, {
    status: query && query.status,
  })))
}

export async function saveDictItem(accessToken, dictType, dictItemId, body) {
  if (useMock()) {
    void accessToken
    if (dictItemId) return patchDictItem(dictType, dictItemId, body)
    return createDictItem(dictType, body)
  }
  const type = text(dictType)
  const id = text(dictItemId)
  if (id) {
    return asObject(await send(accessToken, 'PATCH', `/dicts/${encodeURIComponent(type)}/items/${encodeURIComponent(id)}`, {
      body: dictBody(body, false),
    }))
  }
  return asObject(await send(accessToken, 'POST', `/dicts/${encodeURIComponent(type)}/items`, {
    body: dictBody(body, true),
    ok: [201],
  }))
}

export async function fetchUsers(accessToken, query) {
  if (useMock()) {
    void accessToken
    const listed = listUsers(query)
    return {
      ...listed,
      items: listed.items.map((user) => ({
        ...user,
        roleScopes: cloneAccess(MOCK_USER_ROLE_SCOPES.get(user.userId) || []),
      })),
    }
  }
  return asList(await send(accessToken, 'GET', withQuery('/users', {
    status: query && query.status,
    q: query && query.q,
  })))
}

export async function saveUser(accessToken, userId, body) {
  if (useMock()) {
    void accessToken
    const saved = userId ? patchUser(userId, body) : createUser(body)
    if (Array.isArray(body && body.permissionOverrides)) {
      MOCK_USER_OVERRIDES.set(saved.userId, cloneAccess(body.permissionOverrides))
    }
    if (Array.isArray(body && body.roleScopes)) {
      MOCK_USER_ROLE_SCOPES.set(saved.userId, cloneAccess(body.roleScopes))
    } else if (Array.isArray(body && body.roles)) {
      const assignedRoles = new Set(saved.roles || [])
      const retainedScopes = (MOCK_USER_ROLE_SCOPES.get(saved.userId) || []).filter((scope) => assignedRoles.has(scope.roleCode))
      MOCK_USER_ROLE_SCOPES.set(saved.userId, cloneAccess(retainedScopes))
    }
    return saved
  }
  const id = text(userId)
  if (id) {
    return asObject(await send(accessToken, 'PATCH', `/users/${encodeURIComponent(id)}`, {
      body: userBody(body, false),
    }))
  }
  return asObject(await send(accessToken, 'POST', '/users', {
    body: userBody(body, true),
    ok: [201],
  }))
}

export async function resetUserPassword(accessToken, userId) {
  if (useMock()) {
    void accessToken
    return resetUserPasswordMock(userId)
  }
  const id = text(userId)
  if (!id) throw apiError(400, 'VALIDATION_ERROR', '未找到用户')
  await send(accessToken, 'POST', `/users/${encodeURIComponent(id)}/reset-password`, {
    body: {},
    ok: [204],
  })
  return { ok: true }
}

function cloneAccess(value) {
  return JSON.parse(JSON.stringify(value))
}

export async function fetchAccessModules(accessToken) {
  if (useMock()) { void accessToken; return { items: cloneAccess(MOCK_ACCESS.modules) } }
  return accessList(await send(accessToken, 'GET', '/access/modules'), '模块目录')
}

function accessCatalogueBody(body, identifierKey, fields, creating) {
  const src = body && typeof body === 'object' ? body : {}
  const out = {}
  if (creating) out[identifierKey] = text(src[identifierKey])
  for (const key of fields) {
    if (src[key] == null) continue
    if (['hidden', 'needSpecifyDepartment'].includes(key)) out[key] = Boolean(src[key])
    else if (key === 'secondPermissionId') {
      const raw = String(src[key]).trim()
      const value = raw ? Number(raw) : 0
      if (!Number.isSafeInteger(value) || value < 0) throw apiError(400, 'VALIDATION_ERROR', '二次权限记录号必须是非负整数')
      out[key] = value
    } else out[key] = text(src[key])
  }
  return out
}

async function saveAccessCatalogueItem(accessToken, identifier, body, { key, path, collection, fields, label }) {
  const creating = !text(identifier)
  const input = accessCatalogueBody(body, key, fields, creating)
  if (creating && !input[key]) throw apiError(400, 'VALIDATION_ERROR', `请填写${label}编码`)
  if (useMock()) {
    void accessToken
    const rows = MOCK_ACCESS[collection]
    const recordId = creating ? input[key] : text(identifier)
    const index = rows.findIndex((item) => item[key] === recordId)
    if (creating && index >= 0) throw apiError(409, 'DUPLICATE', `${label}编码已存在`)
    if (!creating && index < 0) throw apiError(404, 'NOT_FOUND', `未找到${label}`)
    if (creating) rows.push({ ...input, [key]: recordId, ...(collection === 'modules' ? { id: `module-${recordId}` } : {}), ...(collection === 'categories' ? { code: recordId } : {}) })
    else rows[index] = { ...rows[index], ...input, [key]: recordId }
    return cloneAccess(creating ? rows.at(-1) : rows[index])
  }
  if (creating) return asObject(await send(accessToken, 'POST', path, { body: input, ok: [201] }))
  return asObject(await send(accessToken, 'PATCH', `${path}/${encodeURIComponent(text(identifier))}`, { body: input }))
}

export async function saveAccessModule(accessToken, moduleCode, body) {
  return saveAccessCatalogueItem(accessToken, moduleCode, body, { key: 'code', path: '/access/modules', collection: 'modules', fields: ['name', 'description', 'status', 'hidden', 'departmentFunctions', 'needSpecifyDepartment', 'secondPermissionId', 'homePageUrl'], label: '模块' })
}

export async function fetchAccessCategories(accessToken) {
  if (useMock()) { void accessToken; return { items: cloneAccess(MOCK_ACCESS.categories) } }
  return accessList(await send(accessToken, 'GET', '/access/categories'), '权限分类')
}

export async function savePermissionCategory(accessToken, categoryId, body) {
  return saveAccessCatalogueItem(accessToken, categoryId, body, { key: 'id', path: '/access/categories', collection: 'categories', fields: ['name', 'description', 'status'], label: '分类' })
}

export async function fetchAccessModes(accessToken) {
  if (useMock()) { void accessToken; return { items: cloneAccess(MOCK_ACCESS.modes) } }
  return accessList(await send(accessToken, 'GET', '/access/modes'), '权限模式')
}

export async function savePermissionMode(accessToken, modeCode, body) {
  return saveAccessCatalogueItem(accessToken, modeCode, body, { key: 'code', path: '/access/modes', collection: 'modes', fields: ['name', 'status'], label: '认证模式' })
}

export async function fetchAccessPermissions(accessToken, query = {}) {
  if (useMock()) {
    void accessToken
    const moduleCode = text(query.moduleCode)
    const categoryId = text(query.categoryId)
    return { items: cloneAccess(MOCK_ACCESS.permissions.filter((item) => (!moduleCode || item.moduleCode === moduleCode) && (!categoryId || item.categoryId === categoryId))) }
  }
  return accessList(await send(accessToken, 'GET', withQuery('/access/permissions', {
    moduleCode: query.moduleCode,
    categoryId: query.categoryId,
  })), '权限目录')
}

function accessPermissionBody(body, creating = false) {
  const src = body && typeof body === 'object' ? body : {}
  const out = {}
  if (creating) out.code = text(src.code)
  for (const key of ['name', 'description', 'moduleCode', 'categoryId', 'modeCode', 'secondPermissionCode', 'dbRoleName', 'status']) {
    if (src[key] != null) out[key] = text(src[key])
  }
  out.needSecondPermission = Boolean(src.needSecondPermission)
  out.actions = Array.isArray(src.actions) ? [...new Set(src.actions.map((action) => text(action)).filter(Boolean))] : []
  out.hidden = Boolean(src.hidden)
  return out
}

export async function saveAccessPermission(accessToken, permissionCode, body) {
  const creating = !text(permissionCode)
  const input = accessPermissionBody(body, creating)
  if (useMock()) {
    void accessToken
    if (!input.code && creating) throw apiError(400, 'VALIDATION_ERROR', '请填写权限编码')
    const code = creating ? input.code : text(permissionCode)
    const index = MOCK_ACCESS.permissions.findIndex((item) => item.code === code)
    if (creating && index >= 0) throw apiError(409, 'DUPLICATE', '权限编码已存在')
    if (!creating && index < 0) throw apiError(404, 'NOT_FOUND', '未找到权限')
    if (creating) MOCK_ACCESS.permissions.push({ ...input, code })
    else MOCK_ACCESS.permissions[index] = { ...MOCK_ACCESS.permissions[index], ...input, code }
    return cloneAccess(creating ? MOCK_ACCESS.permissions.at(-1) : MOCK_ACCESS.permissions[index])
  }
  if (creating) return asObject(await send(accessToken, 'POST', '/access/permissions', { body: input, ok: [201] }))
  return asObject(await send(accessToken, 'PATCH', `/access/permissions/${encodeURIComponent(text(permissionCode))}`, { body: input }))
}

export async function fetchAccessRoles(accessToken) {
  if (useMock()) { void accessToken; return { items: cloneAccess(MOCK_ACCESS.roles) } }
  return accessList(await send(accessToken, 'GET', '/access/roles'), '角色目录')
}

function accessRoleBody(body, creating = false) {
  const src = body && typeof body === 'object' ? body : {}
  const out = {}
  if (creating) out.code = text(src.code).toUpperCase()
  for (const key of ['name', 'description', 'moduleCode', 'departmentId', 'status']) {
    if (src[key] != null) out[key] = text(src[key])
  }
  if (src.standName != null) out.standName = text(src.standName)
  const orderSort = optionalInt(src.orderSort)
  if (orderSort != null) out.orderSort = orderSort
  out.hidden = Boolean(src.hidden)
  out.system = Boolean(src.system)
  out.permissionCodes = Array.isArray(src.permissionCodes) ? src.permissionCodes.map((code) => text(code)).filter(Boolean) : []
  return out
}

export async function saveAccessRole(accessToken, roleCode, body) {
  const creating = !text(roleCode)
  if (useMock()) {
    void accessToken
    const input = accessRoleBody(body, creating)
    if (!input.code) throw apiError(400, 'VALIDATION_ERROR', '请填写角色编码')
    const index = MOCK_ACCESS.roles.findIndex((role) => role.code === (creating ? input.code : text(roleCode)))
    if (creating) {
      if (index >= 0) throw apiError(409, 'DUPLICATE', '角色编码已存在')
      MOCK_ACCESS.roles.push({ ...input, code: input.code })
    } else {
      if (index < 0) throw apiError(404, 'NOT_FOUND', '未找到角色')
      MOCK_ACCESS.roles[index] = { ...MOCK_ACCESS.roles[index], ...input, code: MOCK_ACCESS.roles[index].code }
    }
    return cloneAccess(creating ? MOCK_ACCESS.roles.at(-1) : MOCK_ACCESS.roles[index])
  }
  const code = text(roleCode)
  if (code) return asObject(await send(accessToken, 'PATCH', `/access/roles/${encodeURIComponent(code)}`, { body: accessRoleBody(body) }))
  return asObject(await send(accessToken, 'POST', '/access/roles', { body: accessRoleBody(body, true), ok: [201] }))
}

export async function fetchUserPermissionOverrides(accessToken, userId) {
  const id = text(userId)
  if (!id) throw apiError(400, 'VALIDATION_ERROR', '未找到用户')
  if (useMock()) { void accessToken; return { items: cloneAccess(MOCK_USER_OVERRIDES.get(id) || []) } }
  return accessList(await send(accessToken, 'GET', `/access/users/${encodeURIComponent(id)}/overrides`), '用户权限例外')
}

export async function saveUserPermissionOverrides(accessToken, userId, overrides) {
  const id = text(userId)
  if (!id) throw apiError(400, 'VALIDATION_ERROR', '未找到用户')
  const body = { overrides: Array.isArray(overrides) ? overrides : [] }
  if (useMock()) { void accessToken; MOCK_USER_OVERRIDES.set(id, cloneAccess(body.overrides)); return { items: cloneAccess(body.overrides) } }
  return accessList(await send(accessToken, 'PUT', `/access/users/${encodeURIComponent(id)}/overrides`, { body }), '用户权限例外')
}
