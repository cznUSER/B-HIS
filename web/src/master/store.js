// 2026-10-13 P1-H05-MASTER-UI 基础数据内存库。
// 形状对齐 OpenAPI 0.3.10-draft：/departments、/doctors、/catalog-items、
// /inventory/items + adjust、/dicts、/users。不发 HTTP，不写库。

import { apiError } from '../api/errors.js'

export const MASTER_ROLES = [
  { code: 'REGISTRAR', label: '建档挂号员' },
  { code: 'DOCTOR', label: '门诊医生' },
  { code: 'CASHIER', label: '收费员' },
  { code: 'PHARMACIST', label: '药师' },
  { code: 'SYSTEM', label: '系统' },
]

const SESSION_ROLE = {
  registrar: 'REGISTRAR',
  doctor: 'DOCTOR',
  cashier: 'CASHIER',
  pharmacist: 'PHARMACIST',
  system: 'SYSTEM',
}

const SEED_AT = '2026-10-12T00:00:00.000Z'
const DEMO_PASSWORD = '1'

const READONLY_ENUMS = [
  { dictType: 'sex', name: '性别', values: ['M', 'F', 'U'] },
  { dictType: 'id_type', name: '证件类型', values: ['ID_CARD'] },
  { dictType: 'pay_type', name: '支付方式', values: ['CASH', 'WECHAT', 'ALIPAY', 'OTHER_E', 'OTHER'] },
  { dictType: 'fare_identity', name: '费别', values: ['SELF', 'OTHER'] },
]

const MAINTAINABLE_TYPES = [{ dictType: 'register_sort', name: '号别' }]

function text(value) {
  return typeof value === 'string' ? value.trim() : ''
}

function nowIso() {
  return new Date().toISOString()
}

function sameCode(left, right) {
  return text(left).toUpperCase() === text(right).toUpperCase()
}

function clone(value) {
  return JSON.parse(JSON.stringify(value))
}

function seedState() {
  return {
    seq: 100,
    departments: [
      { departmentId: 'dept-1', deptCode: 'DEPT1', deptName: '内科', status: 'ACTIVE', sortNo: 1, createdAt: SEED_AT, updatedAt: SEED_AT },
      { departmentId: 'dept-2', deptCode: 'DEPT2', deptName: '外科', status: 'ACTIVE', sortNo: 2, createdAt: SEED_AT, updatedAt: SEED_AT },
    ],
    doctors: [
      { doctorId: 'doc-1', doctorCode: 'DOC1', doctorName: '张医生', departmentId: 'dept-1', status: 'ACTIVE', createdAt: SEED_AT, updatedAt: SEED_AT },
      { doctorId: 'doc-2', doctorCode: 'DOC2', doctorName: '李医生', departmentId: 'dept-2', status: 'ACTIVE', createdAt: SEED_AT, updatedAt: SEED_AT },
    ],
    catalog: [
      itemSeed('cat-drug-a', 'DRUG-A', '演示阿莫西林胶囊', 'DRUG', '0.25g×24', '盒', 10),
      itemSeed('cat-drug-b', 'DRUG-B', '演示布洛芬片', 'DRUG', '0.2g×20', '盒', 8),
      itemSeed('cat-treat-1', 'TREAT-1', '演示诊查费', 'TREAT', '', '次', 15),
    ],
    inventory: [
      skuSeed('sku-drug-a', 'cat-drug-a', 'DRUG-A', '演示阿莫西林胶囊', 100, '盒', 1),
      skuSeed('sku-drug-b', 'cat-drug-b', 'DRUG-B', '演示布洛芬片', 100, '盒', 2),
    ],
    ledger: [
      { ledgerId: 'led-drug-a-seed', skuId: 'sku-drug-a', bizType: 'ADJUST', qtyDelta: 100, qtyAfter: 100, remark: '期初', operatorName: '系统管理员', createdAt: SEED_AT, seq: 1 },
      { ledgerId: 'led-drug-b-seed', skuId: 'sku-drug-b', bizType: 'ADJUST', qtyDelta: 100, qtyAfter: 100, remark: '期初', operatorName: '系统管理员', createdAt: SEED_AT, seq: 2 },
    ],
    dictItems: [
      { dictItemId: 'dict-sort-normal', dictType: 'register_sort', code: 'sort-normal', name: '普通号', status: 'ACTIVE', sortNo: 0, createdAt: SEED_AT, updatedAt: SEED_AT },
    ],
    users: [
      {
        userId: 'u-system',
        username: 'system',
        loginName: 'system',
        displayName: '系统管理员',
        status: 'ACTIVE',
        roles: ['SYSTEM'],
        password: DEMO_PASSWORD,
        createdAt: SEED_AT,
        updatedAt: SEED_AT,
      },
    ],
  }
}

function itemSeed(catalogItemId, itemNo, itemName, itemType, spec, units, price) {
  return {
    catalogItemId,
    itemNo,
    itemName,
    itemType,
    spec,
    units,
    price,
    status: 'ACTIVE',
    controlledFlag: false,
    isControlled: false,
    createdAt: SEED_AT,
    updatedAt: SEED_AT,
  }
}

function skuSeed(skuId, catalogItemId, itemNo, itemName, qtyOnHand, units, seq) {
  return {
    skuId,
    catalogItemId,
    skuCode: itemNo,
    itemNo,
    name: itemName,
    itemName,
    qtyOnHand,
    units,
    status: 'ACTIVE',
    rowVersion: String(seq),
    updatedAt: SEED_AT,
  }
}

let state = seedState()

export function resetMasterStore() {
  state = seedState()
}

function nextId(prefix) {
  state.seq += 1
  return `${prefix}-${state.seq}`
}

function nextSeq() {
  state.seq += 1
  return state.seq
}

function asStatus(value, fallback = 'ACTIVE') {
  if (value == null || value === '') return fallback
  if (value === 'ACTIVE' || value === 'INACTIVE') return value
  throw apiError(400, 'VALIDATION_ERROR', '状态无效')
}

function asSort(value, fallback = 0) {
  if (value == null || value === '') return fallback
  const number = Number(value)
  if (!Number.isFinite(number) || !Number.isInteger(number)) {
    throw apiError(400, 'VALIDATION_ERROR', '排序须为整数')
  }
  return number
}

function asPrice(value) {
  if (value == null || value === '') throw apiError(400, 'INVALID_PRICE', '单价不合法')
  const number = Number(value)
  if (!Number.isFinite(number) || number < 0) throw apiError(400, 'INVALID_PRICE', '单价不合法')
  return Math.round((number + Number.EPSILON) * 100) / 100
}

function matches(query, fields) {
  const needle = text(query).toLowerCase()
  if (!needle) return true
  return fields.some((field) => text(field).toLowerCase().includes(needle))
}

function sortBy(list, compare) {
  return list.slice().sort(compare)
}

function publicDepartment(row) {
  return clone(row)
}

function publicDoctor(row) {
  return clone(row)
}

function publicCatalog(row) {
  return {
    catalogItemId: row.catalogItemId,
    itemNo: row.itemNo,
    itemName: row.itemName,
    itemType: row.itemType,
    spec: row.spec || null,
    units: row.units,
    price: row.price,
    status: row.status,
    controlledFlag: false,
    isControlled: false,
    createdAt: row.createdAt,
    updatedAt: row.updatedAt,
  }
}

function publicInventory(row) {
  return {
    skuId: row.skuId,
    skuCode: row.skuCode,
    itemNo: row.itemNo,
    name: row.name,
    itemName: row.itemName,
    qtyOnHand: row.qtyOnHand,
    units: row.units,
    status: row.status,
    rowVersion: row.rowVersion,
    updatedAt: row.updatedAt,
  }
}

function publicLedger(row) {
  return {
    ledgerId: row.ledgerId,
    skuId: row.skuId,
    bizType: row.bizType,
    qtyDelta: row.qtyDelta,
    qtyAfter: row.qtyAfter,
    remark: row.remark,
    operatorName: row.operatorName,
    createdAt: row.createdAt,
  }
}

function publicDict(row) {
  return clone(row)
}

function publicUser(row) {
  return {
    userId: row.userId,
    username: row.username,
    loginName: row.loginName,
    displayName: row.displayName,
    status: row.status,
    roles: [...row.roles],
    createdAt: row.createdAt,
    updatedAt: row.updatedAt,
  }
}

function requireDepartment(departmentId, { allowInactiveId = '' } = {}) {
  const id = text(departmentId)
  const found = state.departments.find((item) => item.departmentId === id)
  if (!found) throw apiError(404, 'DEPARTMENT_NOT_FOUND', '归属科室无效')
  if (found.status !== 'ACTIVE' && found.departmentId !== allowInactiveId) {
    throw apiError(409, 'DEPARTMENT_INACTIVE', '归属科室已停用，不能新挂')
  }
  return found
}

function canonicalRole(role) {
  const raw = text(role)
  const found = MASTER_ROLES.find((item) => item.code.toLowerCase() === raw.toLowerCase())
  return found ? found.code : ''
}

function normalizeRoles(input) {
  const list = Array.isArray(input) ? input : []
  if (list.length === 0) throw apiError(400, 'VALIDATION_ERROR', '请至少选择一个角色')
  const codes = []
  for (const role of list) {
    const code = canonicalRole(role)
    if (!code) throw apiError(400, 'VALIDATION_ERROR', '只能选择预置角色')
    if (!codes.includes(code)) codes.push(code)
  }
  return codes
}

function toSessionRoles(roles) {
  return roles.map((role) => SESSION_ROLE[String(role).toLowerCase()] || role)
}

export function findDepartment(departmentId) {
  const found = state.departments.find((item) => item.departmentId === departmentId)
  return found ? publicDepartment(found) : null
}

export function findDoctor(doctorId) {
  const found = state.doctors.find((item) => item.doctorId === doctorId)
  return found ? publicDoctor(found) : null
}

export function activeDepartments() {
  return sortBy(
    state.departments.filter((item) => item.status === 'ACTIVE'),
    (left, right) => left.sortNo - right.sortNo || left.deptCode.localeCompare(right.deptCode),
  ).map(publicDepartment)
}

export function activeDoctors(departmentId) {
  const activeDeptIds = new Set(
    state.departments.filter((item) => item.status === 'ACTIVE').map((item) => item.departmentId),
  )
  const id = text(departmentId)
  return state.doctors
    .filter((item) => {
      if (item.status !== 'ACTIVE') return false
      if (!activeDeptIds.has(item.departmentId)) return false
      if (id && item.departmentId !== id) return false
      return true
    })
    .map(publicDoctor)
}

export function catalogSelectable(itemNo) {
  const key = text(itemNo)
  if (!key) return false
  const row = state.catalog.find((item) => sameCode(item.itemNo, key))
  if (!row) return true
  return row.status === 'ACTIVE'
}

export function listDepartments(query = {}) {
  const status = text(query.status)
  const items = sortBy(state.departments, (left, right) => left.sortNo - right.sortNo || left.deptCode.localeCompare(right.deptCode))
    .filter((item) => !status || item.status === status)
    .filter((item) => matches(query.q, [item.deptCode, item.deptName]))
    .map(publicDepartment)
  return { items }
}

export function createDepartment(input = {}) {
  const deptCode = text(input.deptCode)
  const deptName = text(input.deptName)
  if (!deptCode || !deptName) throw apiError(400, 'VALIDATION_ERROR', '请填写科室代码和名称')
  if (state.departments.some((item) => sameCode(item.deptCode, deptCode))) {
    throw apiError(409, 'DEPT_CODE_DUPLICATE', '科室代码已存在')
  }
  const now = nowIso()
  const row = {
    departmentId: nextId('dept'),
    deptCode,
    deptName,
    status: asStatus(input.status, 'ACTIVE'),
    sortNo: asSort(input.sortNo, 0),
    createdAt: now,
    updatedAt: now,
  }
  state.departments.push(row)
  return publicDepartment(row)
}

export function patchDepartment(departmentId, input = {}) {
  const row = state.departments.find((item) => item.departmentId === departmentId)
  if (!row) throw apiError(404, 'NOT_FOUND', '未找到科室')
  if ('deptCode' in input) {
    const deptCode = text(input.deptCode)
    if (!deptCode) throw apiError(400, 'VALIDATION_ERROR', '请填写科室代码')
    if (state.departments.some((item) => item.departmentId !== row.departmentId && sameCode(item.deptCode, deptCode))) {
      throw apiError(409, 'DEPT_CODE_DUPLICATE', '科室代码已存在')
    }
    row.deptCode = deptCode
  }
  if ('deptName' in input) {
    const deptName = text(input.deptName)
    if (!deptName) throw apiError(400, 'VALIDATION_ERROR', '请填写科室名称')
    row.deptName = deptName
  }
  if ('status' in input) row.status = asStatus(input.status, row.status)
  if ('sortNo' in input) row.sortNo = asSort(input.sortNo, row.sortNo)
  row.updatedAt = nowIso()
  return publicDepartment(row)
}

export function listDoctors(query = {}) {
  const status = text(query.status)
  const departmentId = text(query.departmentId)
  const items = sortBy(state.doctors, (left, right) => left.doctorCode.localeCompare(right.doctorCode))
    .filter((item) => !status || item.status === status)
    .filter((item) => !departmentId || item.departmentId === departmentId)
    .filter((item) => matches(query.q, [item.doctorCode, item.doctorName]))
    .map(publicDoctor)
  return { items }
}

export function createDoctor(input = {}) {
  const doctorCode = text(input.doctorCode)
  const doctorName = text(input.doctorName)
  if (!doctorCode || !doctorName) throw apiError(400, 'VALIDATION_ERROR', '请填写医生代码和姓名')
  const department = requireDepartment(input.departmentId)
  if (state.doctors.some((item) => sameCode(item.doctorCode, doctorCode))) {
    throw apiError(409, 'DOCTOR_CODE_DUPLICATE', '医生代码已存在')
  }
  const now = nowIso()
  const row = {
    doctorId: nextId('doc'),
    doctorCode,
    doctorName,
    departmentId: department.departmentId,
    status: asStatus(input.status, 'ACTIVE'),
    createdAt: now,
    updatedAt: now,
  }
  state.doctors.push(row)
  return publicDoctor(row)
}

export function patchDoctor(doctorId, input = {}) {
  const row = state.doctors.find((item) => item.doctorId === doctorId)
  if (!row) throw apiError(404, 'NOT_FOUND', '未找到医生')
  if ('doctorCode' in input) {
    const doctorCode = text(input.doctorCode)
    if (!doctorCode) throw apiError(400, 'VALIDATION_ERROR', '请填写医生代码')
    if (state.doctors.some((item) => item.doctorId !== row.doctorId && sameCode(item.doctorCode, doctorCode))) {
      throw apiError(409, 'DOCTOR_CODE_DUPLICATE', '医生代码已存在')
    }
    row.doctorCode = doctorCode
  }
  if ('doctorName' in input) {
    const doctorName = text(input.doctorName)
    if (!doctorName) throw apiError(400, 'VALIDATION_ERROR', '请填写医生姓名')
    row.doctorName = doctorName
  }
  if ('departmentId' in input) {
    const department = requireDepartment(input.departmentId, { allowInactiveId: row.departmentId })
    row.departmentId = department.departmentId
  }
  if ('status' in input) row.status = asStatus(input.status, row.status)
  row.updatedAt = nowIso()
  return publicDoctor(row)
}

export function listCatalogItems(query = {}) {
  const status = text(query.status)
  const itemType = text(query.itemType)
  const items = sortBy(state.catalog, (left, right) => left.itemNo.localeCompare(right.itemNo))
    .filter((item) => !status || item.status === status)
    .filter((item) => !itemType || item.itemType === itemType)
    .filter((item) => matches(query.q, [item.itemNo, item.itemName]))
    .map(publicCatalog)
  return { items }
}

function ensureDrugSku(item) {
  const existing = state.inventory.find((row) => row.catalogItemId === item.catalogItemId || sameCode(row.itemNo, item.itemNo))
  if (existing) {
    existing.catalogItemId = item.catalogItemId
    existing.skuCode = item.itemNo
    existing.itemNo = item.itemNo
    existing.name = item.itemName
    existing.itemName = item.itemName
    existing.units = item.units
    existing.status = item.status
    existing.updatedAt = item.updatedAt
    return false
  }
  const seq = nextSeq()
  state.inventory.push({
    skuId: `sku-${seq}`,
    catalogItemId: item.catalogItemId,
    skuCode: item.itemNo,
    itemNo: item.itemNo,
    name: item.itemName,
    itemName: item.itemName,
    qtyOnHand: 0,
    units: item.units,
    status: item.status,
    rowVersion: '1',
    updatedAt: item.updatedAt,
  })
  return true
}

export function createCatalogItem(input = {}) {
  const itemNo = text(input.itemNo)
  const itemName = text(input.itemName)
  const units = text(input.units)
  const itemType = text(input.itemType)
  if (!itemNo || !itemName || !units) throw apiError(400, 'VALIDATION_ERROR', '请填写编码、名称和单位')
  if (itemType !== 'DRUG' && itemType !== 'TREAT') throw apiError(400, 'VALIDATION_ERROR', '类型只能是药品或治疗')
  if (state.catalog.some((item) => sameCode(item.itemNo, itemNo))) {
    throw apiError(409, 'ITEM_NO_DUPLICATE', '项目编码已存在')
  }
  const now = nowIso()
  const row = {
    catalogItemId: nextId('cat'),
    itemNo,
    itemName,
    itemType,
    spec: text(input.spec),
    units,
    price: asPrice(input.price),
    status: asStatus(input.status, 'ACTIVE'),
    controlledFlag: false,
    isControlled: false,
    createdAt: now,
    updatedAt: now,
  }
  state.catalog.push(row)
  const inventoryCreated = row.itemType === 'DRUG' ? ensureDrugSku(row) : false
  return { item: publicCatalog(row), inventoryCreated }
}

export function patchCatalogItem(catalogItemId, input = {}) {
  const row = state.catalog.find((item) => item.catalogItemId === catalogItemId)
  if (!row) throw apiError(404, 'NOT_FOUND', '未找到目录项目')
  if ('itemNo' in input) {
    const itemNo = text(input.itemNo)
    if (!itemNo) throw apiError(400, 'VALIDATION_ERROR', '请填写项目编码')
    if (state.catalog.some((item) => item.catalogItemId !== row.catalogItemId && sameCode(item.itemNo, itemNo))) {
      throw apiError(409, 'ITEM_NO_DUPLICATE', '项目编码已存在')
    }
    const clash = state.inventory.find((sku) => sku.catalogItemId !== row.catalogItemId && sameCode(sku.itemNo, itemNo))
    if (clash) throw apiError(409, 'ITEM_NO_DUPLICATE', '项目编码已存在')
    row.itemNo = itemNo
  }
  if ('itemName' in input) {
    const itemName = text(input.itemName)
    if (!itemName) throw apiError(400, 'VALIDATION_ERROR', '请填写项目名称')
    row.itemName = itemName
  }
  if ('spec' in input) row.spec = text(input.spec)
  if ('units' in input) {
    const units = text(input.units)
    if (!units) throw apiError(400, 'VALIDATION_ERROR', '请填写单位')
    row.units = units
  }
  if ('price' in input) row.price = asPrice(input.price)
  if ('status' in input) row.status = asStatus(input.status, row.status)
  row.controlledFlag = false
  row.isControlled = false
  row.updatedAt = nowIso()
  if (row.itemType === 'DRUG') ensureDrugSku(row)
  return publicCatalog(row)
}

export function listInventoryItems(query = {}) {
  const status = text(query.status)
  const code = text(query.itemNo) || text(query.skuCode)
  const items = sortBy(state.inventory, (left, right) => left.itemNo.localeCompare(right.itemNo))
    .filter((item) => !status || item.status === status)
    .filter((item) => !code || sameCode(item.itemNo, code) || sameCode(item.skuCode, code))
    .filter((item) => matches(query.q, [item.itemNo, item.itemName, item.skuCode]))
    .map(publicInventory)
  return { items }
}

export function listLedger(skuId) {
  const sku = state.inventory.find((item) => item.skuId === skuId)
  if (!sku) throw apiError(404, 'NOT_FOUND', '未找到库存')
  const items = state.ledger
    .filter((item) => item.skuId === skuId)
    .slice()
    .sort((left, right) => right.seq - left.seq)
    .map(publicLedger)
  return { items }
}

export function adjustInventory(skuId, input = {}) {
  const sku = state.inventory.find((item) => item.skuId === skuId)
  if (!sku) throw apiError(404, 'NOT_FOUND', '未找到库存')
  const catalog = state.catalog.find((item) => item.catalogItemId === sku.catalogItemId || sameCode(item.itemNo, sku.itemNo))
  if (catalog && catalog.itemType !== 'DRUG') throw apiError(409, 'NOT_DRUG_SKU', '治疗项目无库存')
  if (input.qtyDelta == null || input.qtyDelta === '') throw apiError(400, 'VALIDATION_ERROR', '请填写调整数量')
  const qtyDelta = Number(input.qtyDelta)
  if (!Number.isFinite(qtyDelta) || qtyDelta === 0) throw apiError(400, 'VALIDATION_ERROR', '请填写非零调整数量')
  const delta = Math.round((qtyDelta + Number.EPSILON) * 1000) / 1000
  const remark = text(input.remark)
  if (!remark) throw apiError(400, 'VALIDATION_ERROR', '请填写调整原因')
  const qtyAfter = Math.round((sku.qtyOnHand + delta + Number.EPSILON) * 1000) / 1000
  if (qtyAfter < 0) throw apiError(409, 'ADJUST_RESULT_NEGATIVE', '调整后数量不能为负')
  const now = nowIso()
  sku.qtyOnHand = qtyAfter
  sku.rowVersion = String(Number(sku.rowVersion) + 1 || 1)
  sku.updatedAt = now
  state.ledger.push({
    ledgerId: nextId('led'),
    skuId: sku.skuId,
    bizType: 'ADJUST',
    qtyDelta: delta,
    qtyAfter,
    remark,
    operatorName: text(input.operatorName) || '系统管理员',
    createdAt: now,
    seq: nextSeq(),
  })
  return publicInventory(sku)
}

export function listDictOverview() {
  return {
    readonlyEnums: clone(READONLY_ENUMS),
    maintainableTypes: clone(MAINTAINABLE_TYPES),
  }
}

export function listDictItems(dictType, query = {}) {
  const type = text(dictType)
  if (!MAINTAINABLE_TYPES.some((item) => item.dictType === type)) {
    throw apiError(409, 'DICT_READONLY', '该字典为系统枚举，不能维护')
  }
  const status = text(query.status)
  const items = sortBy(
    state.dictItems.filter((item) => item.dictType === type),
    (left, right) => left.sortNo - right.sortNo || left.code.localeCompare(right.code),
  )
    .filter((item) => !status || item.status === status)
    .map(publicDict)
  return { items }
}

export function createDictItem(dictType, input = {}) {
  const type = text(dictType)
  if (!MAINTAINABLE_TYPES.some((item) => item.dictType === type)) {
    throw apiError(409, 'DICT_READONLY', '该字典为系统枚举，不能维护')
  }
  const code = text(input.code)
  const name = text(input.name)
  if (!code || !name) throw apiError(400, 'VALIDATION_ERROR', '请填写代码和名称')
  if (state.dictItems.some((item) => item.dictType === type && sameCode(item.code, code))) {
    throw apiError(409, 'DICT_CODE_DUPLICATE', '代码已存在')
  }
  const now = nowIso()
  const row = {
    dictItemId: nextId('dict'),
    dictType: type,
    code,
    name,
    status: asStatus(input.status, 'ACTIVE'),
    sortNo: asSort(input.sortNo, 0),
    createdAt: now,
    updatedAt: now,
  }
  state.dictItems.push(row)
  return publicDict(row)
}

export function patchDictItem(dictType, dictItemId, input = {}) {
  const type = text(dictType)
  const row = state.dictItems.find((item) => item.dictType === type && item.dictItemId === dictItemId)
  if (!row) throw apiError(404, 'NOT_FOUND', '未找到码表项')
  if ('name' in input) {
    const name = text(input.name)
    if (!name) throw apiError(400, 'VALIDATION_ERROR', '请填写名称')
    row.name = name
  }
  if ('status' in input) row.status = asStatus(input.status, row.status)
  if ('sortNo' in input) row.sortNo = asSort(input.sortNo, row.sortNo)
  row.updatedAt = nowIso()
  return publicDict(row)
}

export function listUsers(query = {}) {
  const status = text(query.status)
  const items = sortBy(state.users, (left, right) => left.username.localeCompare(right.username))
    .filter((item) => !status || item.status === status)
    .filter((item) => matches(query.q, [item.username, item.displayName]))
    .map(publicUser)
  return { items }
}

export function createUser(input = {}) {
  const username = text(input.username) || text(input.loginName)
  const displayName = text(input.displayName)
  if (!username || !displayName) throw apiError(400, 'VALIDATION_ERROR', '请填写登录名和显示名')
  if (/\s/.test(username)) throw apiError(400, 'VALIDATION_ERROR', '登录名不能包含空格')
  if (state.users.some((item) => item.username.toLowerCase() === username.toLowerCase())) {
    throw apiError(409, 'USERNAME_DUPLICATE', '登录名已存在')
  }
  const now = nowIso()
  const row = {
    userId: nextId('user'),
    username,
    loginName: username,
    displayName,
    status: input.status === 'DISABLED' || input.status === 'INACTIVE' ? input.status : 'ACTIVE',
    roles: normalizeRoles(input.roles),
    password: DEMO_PASSWORD,
    createdAt: now,
    updatedAt: now,
  }
  state.users.push(row)
  return publicUser(row)
}

export function patchUser(userId, input = {}) {
  const row = state.users.find((item) => item.userId === userId)
  if (!row) throw apiError(404, 'NOT_FOUND', '未找到用户')
  if ('displayName' in input) {
    const displayName = text(input.displayName)
    if (!displayName) throw apiError(400, 'VALIDATION_ERROR', '请填写显示名')
    row.displayName = displayName
  }
  if ('status' in input) {
    const status = text(input.status)
    if (status !== 'ACTIVE' && status !== 'DISABLED' && status !== 'INACTIVE') {
      throw apiError(400, 'VALIDATION_ERROR', '状态无效')
    }
    row.status = status
  }
  if ('roles' in input) row.roles = normalizeRoles(input.roles)
  row.updatedAt = nowIso()
  return publicUser(row)
}

export function resetUserPassword(userId) {
  const row = state.users.find((item) => item.userId === userId)
  if (!row) throw apiError(404, 'NOT_FOUND', '未找到用户')
  row.password = DEMO_PASSWORD
  row.updatedAt = nowIso()
  return { ok: true }
}

export function loginAccountFromMaster(username) {
  const key = text(username).toLowerCase()
  if (!key) return null
  const row = state.users.find((item) => item.username.toLowerCase() === key)
  if (!row) return null
  return {
    password: row.password,
    disabled: row.status !== 'ACTIVE',
    user: {
      id: row.userId,
      username: row.username,
      displayName: row.displayName,
      roles: toSessionRoles(row.roles),
    },
  }
}

export function masterSummary() {
  return {
    departments: state.departments.length,
    doctors: state.doctors.length,
    catalog: state.catalog.length,
    inventory: state.inventory.length,
    dicts: state.dictItems.length,
    users: state.users.length,
  }
}
