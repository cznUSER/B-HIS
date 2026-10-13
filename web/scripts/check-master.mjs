// 2026-10-13 P1-H05-MASTER-UI 基础数据 Mock：种子可改，错误码与下拉过滤。
// 2026-10-13 关 Mock 时用本地 HTTP 假服务核对路径、状态码和请求体。
import { readFileSync } from 'node:fs'
import http from 'node:http'
import { ApiError } from '../src/api/errors.js'
import { mockLogin, resetMockAuth } from '../src/api/mockAuth.js'
import { openCatalog } from '../src/api/orderContract.js'
import {
  adjustInventory,
  fetchCatalog,
  fetchDepartments,
  fetchDictItems,
  fetchDictOverview,
  fetchDoctors,
  fetchInventory,
  fetchLedger,
  fetchMasterSummary,
  fetchUsers,
  resetUserPassword as resetUserPasswordApi,
  saveCatalogItem,
  saveDepartment,
  saveDictItem,
  saveDoctor,
  saveUser,
} from '../src/master/api.js'
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
assert(apiSrc.includes('fetch('), 'real client fetches')
for (const path of ['/departments', '/doctors', '/catalog-items', '/inventory/items', '/dicts', '/users', 'reset-password']) {
  assert(apiSrc.includes(path), `client path ${path}`)
}
assert(!apiSrc.includes('基础数据接口尚未接入'), 'offline wall removed')
const viewSrc = readFileSync(new URL('../src/views/MasterView.vue', import.meta.url), 'utf8')
assert(viewSrc.includes('system / 1'), 'login hint stays system')
assert(!viewSrc.includes('基础数据接口尚未接入'), 'page mounts in real mode')

process.env.VITE_USE_MOCK = 'true'
const online = await fetchDepartments('token')
assert(online.items.length >= 2, 'mock list returns rows')
try {
  await adjustInventory('token', 'missing', { qtyDelta: 1, remark: '期初' })
  throw new Error('expected missing sku')
} catch (error) {
  assert(error instanceof ApiError && error.status === 404, 'missing sku')
}

process.env.VITE_USE_MOCK = 'false'
const originalFetch = globalThis.fetch
globalThis.fetch = () => {
  throw new Error('network')
}
try {
  await fetchDepartments('token')
  throw new Error('expected network')
} catch (error) {
  assert(error instanceof ApiError && error.status === 0, 'network status')
  assert(error.message === '无法连接基础数据服务', 'network copy')
}
let ledgerHits = 0
globalThis.fetch = (...args) => {
  ledgerHits += 1
  return originalFetch(...args)
}
const quietLedger = await fetchLedger('token', 'sku-a')
assert(quietLedger.supported === false && quietLedger.items.length === 0, 'real ledger is local empty')
assert(ledgerHits === 0, 'real ledger does not fetch')
globalThis.fetch = originalFetch

const calls = []
const server = http.createServer((req, res) => {
  const url = new URL(req.url, 'http://127.0.0.1')
  let raw = ''
  req.on('data', (chunk) => {
    raw += chunk
  })
  req.on('end', () => {
    calls.push({
      method: req.method,
      path: url.pathname,
      search: url.search,
      authorization: req.headers.authorization || '',
      body: raw,
    })
    const json = (status, body) => {
      res.writeHead(status, { 'Content-Type': 'application/json; charset=utf-8' })
      res.end(JSON.stringify(body))
    }
    if (req.method === 'GET' && url.pathname === '/api/v1/departments') {
      json(200, { items: [{ departmentId: 'dept-9', deptCode: 'DEPT9', deptName: '测试科', status: 'ACTIVE', sortNo: 3 }] })
      return
    }
    if (req.method === 'POST' && url.pathname === '/api/v1/departments') {
      const posted = JSON.parse(raw)
      if (String(posted.deptCode).toLowerCase() === 'dept1') {
        json(409, { error: { code: 'DEPT_CODE_DUPLICATE', message: '科室代码已存在' } })
        return
      }
      json(201, { departmentId: 'dept-new', deptCode: posted.deptCode, deptName: posted.deptName, status: 'ACTIVE', sortNo: posted.sortNo || 0 })
      return
    }
    if (req.method === 'PATCH' && url.pathname === '/api/v1/departments/dept-9') {
      json(200, { departmentId: 'dept-9', deptCode: 'DEPT9', deptName: '测试科改', status: 'INACTIVE', sortNo: 4 })
      return
    }
    if (req.method === 'GET' && url.pathname === '/api/v1/doctors') {
      json(200, { items: [{ doctorId: 'doc-9', doctorCode: 'DOC9', doctorName: '周医生', departmentId: 'dept-9', status: 'ACTIVE' }] })
      return
    }
    if (req.method === 'POST' && url.pathname === '/api/v1/doctors') {
      json(201, { doctorId: 'doc-new', doctorCode: 'DOC8', doctorName: '新医生', departmentId: 'dept-9', status: 'ACTIVE' })
      return
    }
    if (req.method === 'PATCH' && url.pathname === '/api/v1/doctors/doc-9') {
      json(200, { doctorId: 'doc-9', doctorCode: 'DOC9', doctorName: '周医生改', departmentId: 'dept-9', status: 'INACTIVE' })
      return
    }
    if (req.method === 'GET' && url.pathname === '/api/v1/catalog-items') {
      json(200, { items: [{ catalogItemId: 'cat-1', itemNo: 'DRUG-A', itemName: '药', itemType: 'DRUG', units: '盒', price: 10, status: 'ACTIVE', controlledFlag: false, isControlled: false }] })
      return
    }
    if (req.method === 'POST' && url.pathname === '/api/v1/catalog-items') {
      const posted = JSON.parse(raw)
      json(201, {
        catalogItemId: 'cat-c',
        itemNo: posted.itemNo,
        itemName: posted.itemName,
        itemType: posted.itemType,
        units: posted.units,
        price: posted.price,
        status: 'ACTIVE',
        controlledFlag: false,
        isControlled: false,
      })
      return
    }
    if (req.method === 'PATCH' && url.pathname === '/api/v1/catalog-items/cat-1') {
      json(200, { catalogItemId: 'cat-1', itemNo: 'DRUG-A', itemName: '药', itemType: 'DRUG', units: '盒', price: 12, status: 'ACTIVE', controlledFlag: false, isControlled: false })
      return
    }
    if (req.method === 'GET' && url.pathname === '/api/v1/inventory/items') {
      json(200, {
        items: [
          { skuId: 'sku-a', skuCode: 'DRUG-A', itemNo: 'DRUG-A', name: '药A', itemName: '药A', qtyOnHand: 100, units: '盒', status: 'ACTIVE', rowVersion: '7' },
          { skuId: 'sku-b', skuCode: 'DRUG-B', itemNo: 'DRUG-B', name: '药B', itemName: '药B', qtyOnHand: 1, units: '盒', status: 'ACTIVE', rowVersion: '1' },
        ],
      })
      return
    }
    if (req.method === 'POST' && url.pathname === '/api/v1/inventory/items/sku-a/adjust') {
      json(200, { skuId: 'sku-a', skuCode: 'DRUG-A', itemNo: 'DRUG-A', name: '药A', itemName: '药A', qtyOnHand: 101, units: '盒', status: 'ACTIVE', rowVersion: '8' })
      return
    }
    if (req.method === 'GET' && url.pathname === '/api/v1/dicts') {
      json(200, {
        readonlyEnums: [{ dictType: 'sex', values: ['M', 'F', 'U'] }],
        maintainableTypes: [{ dictType: 'register_sort', name: '号别' }],
      })
      return
    }
    if (req.method === 'GET' && url.pathname === '/api/v1/dicts/register_sort/items') {
      json(200, { items: [{ dictItemId: 'dict-sort-normal', dictType: 'register_sort', code: 'sort-normal', name: '普通号', status: 'ACTIVE', sortNo: 1 }] })
      return
    }
    if (req.method === 'POST' && url.pathname === '/api/v1/dicts/register_sort/items') {
      json(201, { dictItemId: 'dict-new', dictType: 'register_sort', code: 'sort-expert', name: '专家号', status: 'ACTIVE', sortNo: 2 })
      return
    }
    if (req.method === 'PATCH' && url.pathname === '/api/v1/dicts/register_sort/items/dict-sort-normal') {
      json(200, { dictItemId: 'dict-sort-normal', dictType: 'register_sort', code: 'sort-normal', name: '普通号改', status: 'ACTIVE', sortNo: 1 })
      return
    }
    if (req.method === 'GET' && url.pathname === '/api/v1/users') {
      json(200, { items: [{ userId: 'u-system', username: 'system', loginName: 'system', displayName: '系统管理员', status: 'ACTIVE', roles: ['SYSTEM'] }] })
      return
    }
    if (req.method === 'POST' && url.pathname === '/api/v1/users') {
      const posted = JSON.parse(raw)
      json(201, { userId: 'u-1', username: posted.username, loginName: posted.username, displayName: posted.displayName, status: 'ACTIVE', roles: posted.roles })
      return
    }
    if (req.method === 'PATCH' && url.pathname === '/api/v1/users/u-1') {
      json(200, { userId: 'u-1', username: 'reg1', loginName: 'reg1', displayName: '挂号甲', status: 'DISABLED', roles: ['REGISTRAR'] })
      return
    }
    if (req.method === 'POST' && url.pathname === '/api/v1/users/u-1/reset-password') {
      res.writeHead(204)
      res.end()
      return
    }
    json(404, { error: { code: 'NOT_FOUND', message: 'missing' } })
  })
})

await new Promise((resolve) => {
  server.listen(0, '127.0.0.1', resolve)
})
const { port } = server.address()
process.env.VITE_API_BASE_URL = `http://127.0.0.1:${port}/api/v1`

try {
  const summary = await fetchMasterSummary('tok')
  assert(summary.departments === 1 && summary.doctors === 1 && summary.catalog === 1, 'summary counts')
  assert(summary.inventory === 2 && summary.dicts === 1 && summary.users === 1, 'summary rest')
  assert(calls.every((item) => String(item.authorization).startsWith('Bearer tok')), 'bearer')

  const duplicate = await (async () => {
    try {
      await saveDepartment('tok', '', { deptCode: 'dept1', deptName: '重复', sortNo: '3' })
      return null
    } catch (error) {
      return error
    }
  })()
  assert(duplicate instanceof ApiError && duplicate.code === 'DEPT_CODE_DUPLICATE' && duplicate.status === 409, 'dept duplicate')
  const createdDept = await saveDepartment('tok', '', { deptCode: 'DEPT9', deptName: '测试科', sortNo: '3' })
  assert(createdDept.departmentId === 'dept-new', 'dept create')
  const deptPost = calls.find((item) => item.method === 'POST' && item.path === '/api/v1/departments' && item.body.includes('DEPT9'))
  assert(JSON.parse(deptPost.body).sortNo === 3, 'sortNo integer')
  const patchedDept = await saveDepartment('tok', 'dept-9', { deptName: '测试科改', status: 'INACTIVE', sortNo: 4 })
  assert(patchedDept.status === 'INACTIVE', 'dept patch')

  const doctors = await fetchDoctors('tok', { departmentId: 'dept-9', status: 'ACTIVE' })
  assert(doctors.items[0].doctorId === 'doc-9', 'doctor list')
  const doctorCall = calls.find((item) => item.method === 'GET' && item.path === '/api/v1/doctors' && item.search.includes('departmentId='))
  assert(doctorCall && doctorCall.search.includes('departmentId=dept-9'), 'doctor query')
  const createdDoctor = await saveDoctor('tok', '', { doctorCode: 'DOC8', doctorName: '新医生', departmentId: 'dept-9' })
  assert(createdDoctor.doctorId === 'doc-new', 'doctor create')
  const patchedDoctor = await saveDoctor('tok', 'doc-9', { status: 'INACTIVE' })
  assert(patchedDoctor.status === 'INACTIVE', 'doctor patch')

  const catalog = await fetchCatalog('tok', { itemType: 'DRUG', status: 'ACTIVE' })
  assert(catalog.items[0].itemNo === 'DRUG-A', 'catalog list')
  const savedDrug = await saveCatalogItem('tok', '', {
    itemNo: 'DRUG-C',
    itemName: '新药',
    itemType: 'DRUG',
    units: '盒',
    price: '1.5',
    controlledFlag: true,
    isControlled: true,
  })
  assert(savedDrug.inventoryCreated === true && savedDrug.item.itemType === 'DRUG', 'drug create wraps inventory')
  const catalogPost = calls.find((item) => item.method === 'POST' && item.path === '/api/v1/catalog-items')
  const catalogBody = JSON.parse(catalogPost.body)
  assert(catalogBody.price === 1.5, 'price number')
  assert(!Object.prototype.hasOwnProperty.call(catalogBody, 'controlledFlag'), 'omit controlled')
  assert(!Object.prototype.hasOwnProperty.call(catalogBody, 'isControlled'), 'omit isControlled')
  const patchedCatalog = await saveCatalogItem('tok', 'cat-1', { price: 12, itemType: 'TREAT' })
  assert(patchedCatalog.inventoryCreated === false && patchedCatalog.item.price === 12, 'catalog patch')
  const catalogPatch = calls.find((item) => item.method === 'PATCH' && item.path === '/api/v1/catalog-items/cat-1')
  assert(!Object.prototype.hasOwnProperty.call(JSON.parse(catalogPatch.body), 'itemType'), 'patch omits itemType')

  const stock = await fetchInventory('tok', { q: '药B' })
  assert(stock.items.length === 1 && stock.items[0].itemNo === 'DRUG-B', 'inventory client filter')
  const stockCall = calls.find((item) => item.method === 'GET' && item.path === '/api/v1/inventory/items')
  assert(!stockCall.search.includes('q='), 'inventory has no q')
  const adjusted = await adjustInventory('tok', 'sku-a', { qtyDelta: '1', remark: ' 盘盈 ', rowVersion: '7', operatorName: '系统管理员' })
  assert(adjusted.qtyOnHand === 101, 'adjust result')
  const adjustCall = calls.find((item) => item.method === 'POST' && item.path === '/api/v1/inventory/items/sku-a/adjust')
  const adjustBody = JSON.parse(adjustCall.body)
  assert(adjustBody.qtyDelta === 1 && adjustBody.remark === '盘盈' && adjustBody.rowVersion === '7', 'adjust body')
  assert(!Object.prototype.hasOwnProperty.call(adjustBody, 'operatorName'), 'adjust omits operator')

  const overview = await fetchDictOverview('tok')
  assert(overview.readonlyEnums[0].dictType === 'sex' && !overview.readonlyEnums[0].name, 'enum has no name')
  const sorts = await fetchDictItems('tok', 'register_sort', {})
  assert(sorts.items[0].code === 'sort-normal', 'sort items')
  const createdSort = await saveDictItem('tok', 'register_sort', '', { code: 'sort-expert', name: '专家号', sortNo: '2' })
  assert(createdSort.dictItemId === 'dict-new', 'dict create')
  const patchedSort = await saveDictItem('tok', 'register_sort', 'dict-sort-normal', { name: '普通号改' })
  assert(patchedSort.name === '普通号改', 'dict patch')

  const users = await fetchUsers('tok', { q: 'sys' })
  assert(users.items[0].username === 'system', 'user list')
  const createdUser = await saveUser('tok', '', { username: 'reg1', displayName: '挂号甲', roles: ['REGISTRAR'], password: '1' })
  assert(createdUser.userId === 'u-1' && !JSON.stringify(createdUser).includes('password'), 'user create hides password')
  const userPost = calls.find((item) => item.method === 'POST' && item.path === '/api/v1/users')
  const userBody = JSON.parse(userPost.body)
  assert(!Object.prototype.hasOwnProperty.call(userBody, 'password'), 'user body has no password')
  assert(userBody.roles[0] === 'REGISTRAR', 'roles upper')
  const patchedUser = await saveUser('tok', 'u-1', { status: 'DISABLED', roles: ['REGISTRAR'] })
  assert(patchedUser.status === 'DISABLED', 'user patch')
  const reset = await resetUserPasswordApi('tok', 'u-1')
  assert(reset.ok === true, 'reset ok')
  const resetCall = calls.find((item) => item.path === '/api/v1/users/u-1/reset-password')
  assert(resetCall && resetCall.method === 'POST', 'reset path')
} finally {
  await new Promise((resolve) => server.close(resolve))
  process.env.VITE_USE_MOCK = 'true'
  delete process.env.VITE_API_BASE_URL
}

console.log('master checks passed')
