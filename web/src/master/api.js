// 2026-10-13 P1-H05-MASTER-UI 基础数据客户端。
// VITE_USE_MOCK 打开时走本地内存；关闭时不发请求，提示尚未接入。
// 路径名对齐 OpenAPI 0.3.10-draft，本模块不调用 fetch。

import { apiError } from '../api/errors.js'
import { useMock } from '../api/config.js'
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

function gate() {
  if (!useMock()) throw apiError(501, 'NOT_IMPLEMENTED', '基础数据接口尚未接入')
}

export async function fetchMasterSummary(accessToken) {
  gate()
  void accessToken
  return masterSummary()
}

export async function fetchDepartments(accessToken, query) {
  gate()
  void accessToken
  return listDepartments(query)
}

export async function saveDepartment(accessToken, departmentId, body) {
  gate()
  void accessToken
  if (departmentId) return patchDepartment(departmentId, body)
  return createDepartment(body)
}

export async function fetchDoctors(accessToken, query) {
  gate()
  void accessToken
  return listDoctors(query)
}

export async function saveDoctor(accessToken, doctorId, body) {
  gate()
  void accessToken
  if (doctorId) return patchDoctor(doctorId, body)
  return createDoctor(body)
}

export async function fetchCatalog(accessToken, query) {
  gate()
  void accessToken
  return listCatalogItems(query)
}

export async function saveCatalogItem(accessToken, catalogItemId, body) {
  gate()
  void accessToken
  if (catalogItemId) return { item: patchCatalogItem(catalogItemId, body), inventoryCreated: false }
  return createCatalogItem(body)
}

export async function fetchInventory(accessToken, query) {
  gate()
  void accessToken
  return listInventoryItems(query)
}

export async function fetchLedger(accessToken, skuId) {
  gate()
  void accessToken
  return listLedger(skuId)
}

export async function adjustInventory(accessToken, skuId, body) {
  gate()
  void accessToken
  return adjustInventoryMock(skuId, body)
}

export async function fetchDictOverview(accessToken) {
  gate()
  void accessToken
  return listDictOverview()
}

export async function fetchDictItems(accessToken, dictType, query) {
  gate()
  void accessToken
  return listDictItems(dictType, query)
}

export async function saveDictItem(accessToken, dictType, dictItemId, body) {
  gate()
  void accessToken
  if (dictItemId) return patchDictItem(dictType, dictItemId, body)
  return createDictItem(dictType, body)
}

export async function fetchUsers(accessToken, query) {
  gate()
  void accessToken
  return listUsers(query)
}

export async function saveUser(accessToken, userId, body) {
  gate()
  void accessToken
  if (userId) return patchUser(userId, body)
  return createUser(body)
}

export async function resetUserPassword(accessToken, userId) {
  gate()
  void accessToken
  return resetUserPasswordMock(userId)
}
