// 2026-10-13 P1-H05-MASTER-UI 基础数据 Mock：种子可改，错误码与下拉过滤。
import { readFileSync } from 'node:fs'
import { ApiError } from '../src/api/errors.js'
import { mockLogin, resetMockAuth } from '../src/api/mockAuth.js'
import { openCatalog } from '../src/api/orderContract.js'
import { adjustInventory, fetchDepartments } from '../src/master/api.js'
import { canMaintainMaster } from '../src/master/access.js'
import {
  activeDepartments,
  adjustInventory as adjustInventoryStore,
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
  loginAccountFromMaster,
  patchCatalogItem,
  patchDepartment,
  patchDoctor,
  patchUser,
  resetMasterStore,
  resetUserPassword,
} from '../src/master/store.js'
import { doctorsForDept, selectableDepartments } from '../src/register/master.js'

function assert(condition, message) {
  if (!condition) throw new Error(message)
}

function expectError(fn, code, status) {
  try {
    fn()
  } catch (error) {
    assert(error instanceof ApiError, 'expected ApiError')
    assert(error.code === code, `expected code ${code}, got ${error.code}`)
    assert(error.status === status, `expected status ${status}, got ${error.status}`)
    return error
  }
  throw new Error(`expected ${code}`)
}

resetMasterStore()
resetMockAuth()

const depts = listDepartments().items
assert(depts.map((item) => item.departmentId).join(',') === 'dept-1,dept-2', 'dept seeds')
assert(depts[0].deptCode === 'DEPT1' && depts[0].deptName === '内科', 'dept-1 editable seed')
const renamed = patchDepartment('dept-1', { deptName: '心内科' })
assert(renamed.departmentId === 'dept-1' && renamed.deptName === '心内科', 'seed dept can be renamed')
expectError(() => patchDepartment('dept-1', { deptCode: 'DEPT2' }), 'DEPT_CODE_DUPLICATE', 409)
expectError(() => createDepartment({ deptCode: 'dept2', deptName: '重复' }), 'DEPT_CODE_DUPLICATE', 409)
const stopped = patchDepartment('dept-1', { status: 'INACTIVE' })
assert(stopped.status === 'INACTIVE', 'dept can stop')
assert(selectableDepartments().every((item) => item.departmentId !== 'dept-1'), 'stopped dept leaves register list')
assert(doctorsForDept('dept-1').length === 0, 'stopped dept has no new doctors')
assert(doctorsForDept('').length === 0, 'empty dept still has no doctors')
patchDepartment('dept-1', { status: 'ACTIVE', deptName: '内科' })
assert(doctorsForDept('dept-1').map((item) => item.doctorId).join(',') === 'doc-1', 'dept-1 doctor returns')
assert(selectableDepartments().some((item) => item.name === '内科'), 'register names follow master')

const cold = createDepartment({ deptCode: 'DEPTX', deptName: '停用科', status: 'INACTIVE' })
expectError(
  () => createDoctor({ doctorCode: 'DOCX', doctorName: '无科', departmentId: cold.departmentId }),
  'DEPARTMENT_INACTIVE',
  409,
)
expectError(
  () => createDoctor({ doctorCode: 'DOC1', doctorName: '重复', departmentId: 'dept-2' }),
  'DOCTOR_CODE_DUPLICATE',
  409,
)
expectError(() => createDoctor({ doctorCode: 'DOC9', doctorName: '无', departmentId: 'missing' }), 'DEPARTMENT_NOT_FOUND', 404)
const extra = createDoctor({ doctorCode: 'DOC9', doctorName: '周医生', departmentId: 'dept-1' })
assert(extra.doctorId && listDoctors({ departmentId: 'dept-1' }).items.some((item) => item.doctorId === extra.doctorId), 'filter by dept')
patchDoctor(extra.doctorId, { status: 'INACTIVE' })
assert(!doctorsForDept('dept-1').some((item) => item.doctorId === extra.doctorId), 'stopped doctor hidden')

const catalog = listCatalogItems().items.map((item) => item.itemNo).join(',')
assert(catalog === 'DRUG-A,DRUG-B,TREAT-1', 'catalog seeds')
assert(listCatalogItems().items.every((item) => item.controlledFlag === false && item.isControlled === false), 'controlled stays off')
const before = listInventoryItems().items.length
const treat = createCatalogItem({ itemNo: 'TREAT-9', itemName: '换药', itemType: 'TREAT', units: '次', price: 3 })
assert(treat.inventoryCreated === false, 'treat skips stock')
assert(listInventoryItems().items.length === before, 'treat not in inventory')
const drug = createCatalogItem({ itemNo: 'DRUG-C', itemName: '新药', itemType: 'DRUG', units: '盒', price: 2, spec: '1片' })
assert(drug.inventoryCreated === true, 'drug opens stock')
const sku = listInventoryItems().items.find((item) => item.itemNo === 'DRUG-C')
assert(sku && sku.qtyOnHand === 0, 'new drug qty 0')
expectError(
  () => createCatalogItem({ itemNo: 'drug-a', itemName: '重复', itemType: 'DRUG', units: '盒', price: 1 }),
  'ITEM_NO_DUPLICATE',
  409,
)
expectError(
  () => createCatalogItem({ itemNo: 'DRUG-Z', itemName: '坏价', itemType: 'DRUG', units: '盒', price: -1 }),
  'INVALID_PRICE',
  400,
)
const drugB = listCatalogItems().items.find((item) => item.itemNo === 'DRUG-B')
assert(openCatalog().some((item) => item.itemNo === 'DRUG-B'), 'active drug stays on order list')
patchCatalogItem(drugB.catalogItemId, { status: 'INACTIVE', price: 9 })
assert(!openCatalog().some((item) => item.itemNo === 'DRUG-B'), 'stopped catalog leaves order list')
assert(listCatalogItems().items.find((item) => item.itemNo === 'DRUG-B').price === 9, 'seed price editable')
patchCatalogItem(drugB.catalogItemId, { status: 'ACTIVE' })

const stockA = listInventoryItems().items.find((item) => item.itemNo === 'DRUG-A')
assert(stockA.qtyOnHand === 100, 'inventory seed 100')
expectError(() => adjustInventoryStore(stockA.skuId, { qtyDelta: -1000, remark: '盘亏' }), 'ADJUST_RESULT_NEGATIVE', 409)
assert(listInventoryItems().items.find((item) => item.skuId === stockA.skuId).qtyOnHand === 100, 'rejected adjust keeps qty')
expectError(() => adjustInventoryStore(stockA.skuId, { qtyDelta: -1, remark: '  ' }), 'VALIDATION_ERROR', 400)
const adjusted = adjustInventoryStore(stockA.skuId, { qtyDelta: -5, remark: '盘点', operatorName: '系统管理员' })
assert(adjusted.qtyOnHand === 95, 'adjust delta')
const ledger = listLedger(stockA.skuId).items
assert(ledger[0].bizType === 'ADJUST' && ledger[0].qtyDelta === -5 && ledger[0].remark === '盘点', 'ledger adjust')
assert(ledger.every((item) => item.bizType === 'ADJUST'), 'mock ledger is adjust only')

const overview = listDictOverview()
assert(overview.readonlyEnums.some((item) => item.dictType === 'sex' && item.values.join(',') === 'M,F,U'), 'sex enum')
assert(overview.maintainableTypes.some((item) => item.dictType === 'register_sort'), 'register sort type')
expectError(() => createDictItem('sex', { code: 'X', name: '其他' }), 'DICT_READONLY', 409)
expectError(() => createDictItem('register_sort', { code: 'sort-normal', name: '再来一个' }), 'DICT_CODE_DUPLICATE', 409)
const sort = createDictItem('register_sort', { code: 'sort-expert', name: '专家号' })
assert(listDictItems('register_sort').items.some((item) => item.dictItemId === sort.dictItemId), 'sort row')
const normal = listDictItems('register_sort').items.find((item) => item.code === 'sort-normal')
assert(normal && normal.name === '普通号', 'sort-normal seed')

const users = listUsers().items
assert(users.some((item) => item.username === 'system' && item.roles.includes('SYSTEM')), 'system user seed')
assert(!Object.prototype.hasOwnProperty.call(users[0], 'password'), 'user dto hides password')
expectError(
  () => createUser({ username: 'system', displayName: '另一个', roles: ['SYSTEM'] }),
  'USERNAME_DUPLICATE',
  409,
)
const created = createUser({ username: 'reg1', displayName: '挂号甲', roles: ['registrar'] })
assert(!JSON.stringify(created).includes('password'), 'create hides password')
const account = loginAccountFromMaster('reg1')
assert(account && account.password === '1' && account.disabled === false, 'default password')
assert(account.user.roles.includes('REGISTRAR'), 'login role maps')
resetUserPassword(created.userId)
assert(loginAccountFromMaster('reg1').password === '1', 'reset password')
patchUser(created.userId, { status: 'DISABLED' })
assert(loginAccountFromMaster('reg1').disabled === true, 'disabled flag')
expectError(() => mockLogin({ username: 'reg1', password: '1' }), 'ACCOUNT_DISABLED', 403)
resetMockAuth()
const systemLogin = mockLogin({ username: 'system', password: '1' })
assert(systemLogin.user.username === 'system' && systemLogin.user.roles.includes('SYSTEM'), 'system login still works')

assert(canMaintainMaster({ username: 'system', roles: [] }) === true, 'demo system')
assert(canMaintainMaster({ username: 'ops', roles: ['SYSTEM'] }) === true, 'SYSTEM role')
assert(canMaintainMaster({ username: 'ops', roles: ['admin:master'] }) === true, 'admin scope')
assert(canMaintainMaster({ username: 'reg', roles: ['REGISTRAR'] }) === false, 'registrar blocked')
assert(canMaintainMaster(null) === false, 'anonymous blocked')

const routerSrc = readFileSync(new URL('../src/router/index.js', import.meta.url), 'utf8')
for (const path of ['settings/master', 'departments', 'doctors', 'catalog', 'inventory', 'dicts', 'users']) {
  assert(routerSrc.includes(path), `route ${path}`)
}
const navSrc = readFileSync(new URL('../src/workbench/nav.js', import.meta.url), 'utf8')
assert(navSrc.includes("label: '基础数据'"), 'nav label')
assert(!navSrc.includes("label: '基础字典'"), 'old placeholder gone')
const apiSrc = readFileSync(new URL('../src/master/api.js', import.meta.url), 'utf8')
assert(!apiSrc.includes('fetch('), 'mock api does not fetch')
assert(apiSrc.includes('基础数据接口尚未接入'), 'offline copy')

process.env.VITE_USE_MOCK = 'false'
let fetched = false
globalThis.fetch = () => {
  fetched = true
  throw new Error('network')
}
try {
  await fetchDepartments('token')
  throw new Error('expected offline')
} catch (error) {
  assert(error instanceof ApiError && error.code === 'NOT_IMPLEMENTED' && error.status === 501, 'offline code')
}
assert(fetched === false, 'offline does not fetch')
process.env.VITE_USE_MOCK = 'true'
const online = await fetchDepartments('token')
assert(online.items.length >= 2, 'mock list returns rows')
try {
  await adjustInventory('token', 'missing', { qtyDelta: 1, remark: '期初' })
  throw new Error('expected missing sku')
} catch (error) {
  assert(error instanceof ApiError && error.status === 404, 'missing sku')
}

console.log('master checks passed')
