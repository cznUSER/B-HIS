// 2026-10-13 P1-H05-MASTER-UI 基础数据客户端。
// 2026-10-13 真接口：VITE_USE_MOCK=false 走 OpenAPI 0.3.10-draft / 已合 Go handlers。
// 路径：/departments、/doctors、/catalog-items、/inventory/items + adjust、/dicts、/users。
// 服务端没有库存流水列表，真接口下 fetchLedger 不发请求。

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
  if (!allowed.includes(response.status)) throw await readErrorResponse(response)
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
  if (text(src.status)) out.status = text(src.status)
  if (Array.isArray(src.roles)) {
    out.roles = src.roles.map((role) => text(role).toUpperCase()).filter(Boolean)
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

export async function fetchLedger(accessToken, skuId) {
  if (useMock()) {
    void accessToken
    return listLedger(skuId)
  }
  void accessToken
  void skuId
  return { items: [], supported: false }
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
    return listUsers(query)
  }
  return asList(await send(accessToken, 'GET', withQuery('/users', {
    status: query && query.status,
    q: query && query.q,
  })))
}

export async function saveUser(accessToken, userId, body) {
  if (useMock()) {
    void accessToken
    if (userId) return patchUser(userId, body)
    return createUser(body)
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
