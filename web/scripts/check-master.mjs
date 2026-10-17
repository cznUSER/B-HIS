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
const mockLedgerPage1 = await fetchLedger('token', 'DRUG-A', { from: '2026-10-01', to: '2026-11-01', limit: 1 })
assert(mockLedgerPage1.items.length === 1 && mockLedgerPage1.nextCursor, 'mock ledger returns an opaque next cursor')
const mockLedgerPage2 = await fetchLedger('token', 'DRUG-A', { from: '2026-10-01', to: '2026-11-01', limit: 1, cursor: mockLedgerPage1.nextCursor })
assert(mockLedgerPage2.items.length === 1 && !mockLedgerPage2.nextCursor, 'mock ledger cursor returns the final page')
const mockLedgerReturnOnly = await fetchLedger('token', 'DRUG-A', { bizType: 'RETURN', from: '2026-10-01', to: '2026-11-01' })
assert(mockLedgerReturnOnly.items.length === 0, 'mock ledger filters by business type and inclusive date range')
const mockLedgerShanghaiDay = await fetchLedger('token', 'DRUG-A', { bizType: 'ADJUST', from: '2026-10-12', to: '2026-10-12' })
assert(mockLedgerShanghaiDay.items.length === 1 && mockLedgerShanghaiDay.items[0].ledgerId === 'led-drug-a-seed', 'mock ledger includes a Shanghai calendar day boundary')
let badDateError
try { await fetchLedger('token', 'DRUG-A', { from: '2026-02-30' }) } catch (error) { badDateError = error }
assert(badDateError instanceof ApiError && badDateError.code === 'VALIDATION_ERROR' && badDateError.status === 400, 'mock ledger rejects invalid calendar dates')
let badBizTypeError
try { await fetchLedger('token', 'DRUG-A', { bizType: 'UNKNOWN' }) } catch (error) { badBizTypeError = error }
assert(badBizTypeError instanceof ApiError && badBizTypeError.code === 'VALIDATION_ERROR' && badBizTypeError.status === 400, 'mock ledger rejects an unknown business type')

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
const created = createUser({ username: 'reg1', displayName: '挂号甲', description: '门诊挂号', cardNo: 'CARD-1', inputCode: 'ghj', roles: ['registrar'], accountStopped: false, mustChangePassword: true, cannotModifyPassword: false, hidden: true, system: false, super: false, developer: false })
assert(!JSON.stringify(created).includes('"password":'), 'create hides password')
assert(created.description === '门诊挂号' && created.cardNo === 'CARD-1' && created.inputCode === 'ghj', 'user profile metadata')
assert(created.mustChangePassword === true && created.hidden === true && created.super === false && created.developer === false, 'user lifecycle flags round-trip')
const account = loginAccountFromMaster('reg1')
assert(account && account.password === '1' && account.disabled === false, 'default password')
assert(account.user.roles.includes('REGISTRAR'), 'login role maps')
assert(account.user.mustChangePassword === true && account.user.cannotModifyPassword === false, 'login lifecycle flags')
resetUserPassword(created.userId)
assert(loginAccountFromMaster('reg1').password === '1' && listUsers().items.find((item) => item.userId === created.userId).mustChangePassword, 'reset password requires change')
patchUser(created.userId, { description: '夜班挂号', cardNo: 'CARD-2', inputCode: 'ybgh', mustChangePassword: false, cannotModifyPassword: true, accountStopped: true, hidden: false, system: true, super: true, developer: true })
const updatedUser = listUsers().items.find((item) => item.userId === created.userId)
assert(updatedUser.description === '夜班挂号' && updatedUser.cardNo === 'CARD-2' && updatedUser.inputCode === 'ybgh', 'user profile patch')
assert(updatedUser.cannotModifyPassword && updatedUser.accountStopped && updatedUser.system && updatedUser.super && updatedUser.developer, 'user protected lifecycle patch')
assert(loginAccountFromMaster('reg1').disabled === true, 'stopped account blocked')
patchUser(created.userId, { status: 'DISABLED' })
assert(loginAccountFromMaster('reg1').disabled === true, 'disabled flag')
expectError(() => mockLogin({ username: 'reg1', password: '1' }), 'ACCOUNT_DISABLED', 403)
resetMockAuth()
const systemLogin = mockLogin({ username: 'system', password: '1' })
assert(systemLogin.user.username === 'system' && systemLogin.user.roles.includes('SYSTEM'), 'system login still works')

assert(canMaintainMaster({ username: 'system', roles: [] }) === true, 'demo system')
assert(canMaintainMaster({ username: 'ops', roles: ['SYSTEM'] }) === true, 'SYSTEM role')
assert(canMaintainMaster({ username: 'ops', roles: ['admin:master'] }) === true, 'admin scope')
assert(canMaintainMaster({ roles: ['SYSTEM'], permissions: [], scopes: [], accessControlLoaded: true }) === false, 'empty effective master permissions block access')
assert(canMaintainMaster({ permissions: ['admin:user'], scopes: ['master:'] }) === true, 'effective admin permission opens master')
assert(canMaintainMaster({ permissions: ['admin:user'], scopes: ['outpatient:dept-1'] }) === false, 'outpatient-only scope blocks master')
assert(canMaintainMaster({ permissions: ['admin:*'], deniedPermissions: ['admin:*'] }) === false, 'master admin deny wins')
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
    if (req.method === 'GET' && url.pathname === '/api/v1/inventory/ledger') {
      json(200, {
        items: [{ ledgerId: 'ledger-1', skuId: 'sku-a', skuCode: 'DRUG-A', bizType: 'DISPENSE', bizId: 'disp-1', qtyDelta: -1, qtyAfter: 99, remark: '', operatorName: '药师', createdAt: '2026-10-15T01:00:00Z' }],
        nextCursor: 'opaque-next-2',
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
  const ledgerPage = await fetchLedger('tok', 'DRUG-A', {
    bizType: 'DISPENSE', from: '2026-10-01', to: '2026-10-15', cursor: 'opaque-next', limit: 50,
  })
  assert(ledgerPage.items[0]?.ledgerId === 'ledger-1' && ledgerPage.nextCursor === 'opaque-next-2', 'real ledger returns a page')
  const ledgerCall = calls.find((item) => item.method === 'GET' && item.path === '/api/v1/inventory/ledger')
  assert(ledgerCall, 'real ledger endpoint requested')
  assert(ledgerCall.search.includes('skuCode=DRUG-A') && ledgerCall.search.includes('bizType=DISPENSE'), 'ledger filters are sent')
  assert(ledgerCall.search.includes('from=2026-10-01') && ledgerCall.search.includes('to=2026-10-15'), 'ledger calendar dates are sent')
  assert(ledgerCall.search.includes('cursor=opaque-next') && ledgerCall.search.includes('limit=50'), 'ledger pagination is sent')
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
  const createdUser = await saveUser('tok', '', { username: 'reg1', displayName: '挂号甲', description: '门诊挂号', cardNo: 'CARD-1', inputCode: 'ghj', serviceStar: 4, expandInterface1: 10, expandInterface2: 11, inputCodeCompareMode: 2, roles: ['REGISTRAR'], mustChangePassword: true, cannotModifyPassword: false, accountStopped: false, hidden: true, system: false, super: false, developer: false, password: 'Stronger!9' })
  assert(createdUser.userId === 'u-1' && !JSON.stringify(createdUser).includes('"password":'), 'user create hides password')
  const userPost = calls.find((item) => item.method === 'POST' && item.path === '/api/v1/users')
  const userBody = JSON.parse(userPost.body)
  assert(userBody.password === 'Stronger!9', 'user body carries optional initial password')
  assert(userBody.roles[0] === 'REGISTRAR', 'roles upper')
  assert(userBody.description === '门诊挂号' && userBody.cardNo === 'CARD-1' && userBody.inputCode === 'ghj', 'user create profile body')
  assert(userBody.mustChangePassword === true && userBody.cannotModifyPassword === false && userBody.hidden === true && userBody.super === false, 'user create lifecycle body')
  assert(userBody.serviceStar === 4 && userBody.expandInterface1 === 10 && userBody.expandInterface2 === 11 && userBody.inputCodeCompareMode === 2, 'user numeric metadata body')
  const patchedUser = await saveUser('tok', 'u-1', { status: 'DISABLED', roles: ['REGISTRAR'], description: '', cardNo: '', inputCode: 'ybgh', serviceStar: null, expandInterface1: 5, expandInterface2: 6, inputCodeCompareMode: 2, mustChangePassword: false, cannotModifyPassword: true, accountLocked: false, accountStopped: true, hidden: false, system: false, super: true, developer: false })
  assert(patchedUser.status === 'DISABLED', 'user patch')
  const userPatch = calls.find((item) => item.method === 'PATCH' && item.path === '/api/v1/users/u-1')
  const patchBody = JSON.parse(userPatch.body)
  assert(patchBody.description === '' && patchBody.cardNo === '' && patchBody.inputCode === 'ybgh', 'user patch profile body includes empty values')
  assert(patchBody.mustChangePassword === false && patchBody.accountLocked === false && patchBody.accountStopped === true && patchBody.super === true, 'user patch lifecycle body preserves false')
  assert(patchBody.serviceStar === null && patchBody.expandInterface1 === 5 && patchBody.expandInterface2 === 6 && patchBody.inputCodeCompareMode === 2, 'user patch numeric metadata and nullable star')
  await saveUser('tok', 'u-1', { displayName: '仅更新资料' })
  const limitedUserPatch = calls.at(-1)
  const limitedUserBody = JSON.parse(limitedUserPatch.body)
  assert(['hidden', 'system', 'developer', 'super', 'cardNo'].every((key) => !Object.hasOwn(limitedUserBody, key)), 'user API keeps omitted protected fields omitted')
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
