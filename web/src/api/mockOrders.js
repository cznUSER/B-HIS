// 2026-10-12 P1-H05-ORDER-UI 本地开单 Mock，形状对齐 OpenAPI 0.3.7。
// 无有效 registerId → ENCOUNTER_REQUIRED。提交走 commit，作废仅未收费。不访问网络。

import { apiError } from './errors.js'
import { mockMe } from './mockAuth.js'
import { mockAllStoredRegisters, mockGetStoredRegister } from './mockRegisters.js'
import { mockGetPatient } from './mockPatients.js'
import { todayInShanghai } from './patientContract.js'
import { deptName, doctorName } from '../register/master.js'
import {
  assertCreateOrderRequest,
  assertOrderDto,
  assertPatchOrderRequest,
  registerIsOrderable,
  round2,
} from './orderContract.js'

const STORAGE_KEY = 'bhis.orders.v1'

export const DEMO_REGISTERS = [
  {
    registerId: '11111111-1111-4111-8111-111111111111',
    registerNo: 'G202610120001',
    patientId: '22222222-2222-4222-8222-222222222221',
    patientName: '李演示',
    patientNo: 'P202610120001',
    departmentId: 'dept-1',
    doctorId: 'doc-1',
    visitDate: '2026-10-12',
    status: 'NORMAL',
  },
  {
    registerId: '11111111-1111-4111-8111-111111111112',
    registerNo: 'G202610120002',
    patientId: '22222222-2222-4222-8222-222222222222',
    patientName: '王演示',
    patientNo: 'P202610120002',
    departmentId: 'dept-2',
    doctorId: 'doc-2',
    visitDate: '2026-10-12',
    status: 'NORMAL',
  },
  {
    registerId: '11111111-1111-4111-8111-111111111113',
    registerNo: 'G202610120003',
    patientId: '22222222-2222-4222-8222-222222222223',
    patientName: '赵已退',
    patientNo: 'P202610120003',
    departmentId: 'dept-1',
    doctorId: 'doc-1',
    visitDate: '2026-10-12',
    status: 'BACK',
  },
]

const memory = {
  seq: 0,
  orders: [],
  idempotency: {},
}

function storage() {
  try {
    return typeof sessionStorage === 'undefined' ? null : sessionStorage
  } catch {
    return null
  }
}

function readState() {
  const store = storage()
  if (!store) return memory
  try {
    const raw = store.getItem(STORAGE_KEY)
    if (!raw) return { seq: 0, orders: [], idempotency: {} }
    const parsed = JSON.parse(raw)
    return {
      seq: typeof parsed.seq === 'number' ? parsed.seq : 0,
      orders: Array.isArray(parsed.orders) ? parsed.orders : [],
      idempotency: parsed.idempotency && typeof parsed.idempotency === 'object' ? parsed.idempotency : {},
    }
  } catch {
    return { seq: 0, orders: [], idempotency: {} }
  }
}

function writeState(state) {
  const store = storage()
  if (!store) {
    memory.seq = state.seq
    memory.orders = state.orders
    memory.idempotency = state.idempotency
    return
  }
  store.setItem(STORAGE_KEY, JSON.stringify(state))
}

function requestId() {
  return `mock-${crypto.randomUUID()}`
}

function encounterRequired() {
  throw apiError(409, 'ENCOUNTER_REQUIRED', '请先挂号', { requestId: requestId() })
}

function presentRegister(accessToken, raw) {
  let patientName = typeof raw.patientName === 'string' ? raw.patientName : ''
  let patientNo = typeof raw.patientNo === 'string' ? raw.patientNo : ''
  if (!patientName && raw.patientId) {
    try {
      const patient = mockGetPatient(accessToken, raw.patientId)
      patientName = patient.name || ''
      patientNo = patient.patientNo || ''
    } catch {
      patientName = ''
    }
  }
  return {
    registerId: raw.registerId,
    registerNo: raw.registerNo || '',
    patientId: raw.patientId,
    patientName,
    patientNo,
    departmentId: raw.departmentId || '',
    departmentName: deptName(raw.departmentId),
    doctorId: raw.doctorId || '',
    doctorName: doctorName(raw.doctorId),
    visitDate: raw.visitDate || '',
    status: raw.status,
  }
}

function findRegister(accessToken, registerId) {
  const demo = DEMO_REGISTERS.find((item) => item.registerId === registerId)
  if (demo) return presentRegister(accessToken, demo)
  const stored = mockGetStoredRegister(accessToken, registerId)
  return stored ? presentRegister(accessToken, stored) : null
}

function requireOrderable(accessToken, registerId) {
  const register = findRegister(accessToken, registerId)
  if (!registerIsOrderable(register)) encounterRequired()
  return register
}

export function resetMockOrders() {
  memory.seq = 0
  memory.orders = []
  memory.idempotency = {}
  const store = storage()
  if (store) store.removeItem(STORAGE_KEY)
}

export function mockListOrderRegisters(accessToken) {
  mockMe(accessToken)
  const map = new Map()
  for (const item of DEMO_REGISTERS) map.set(item.registerId, presentRegister(accessToken, item))
  for (const item of mockAllStoredRegisters(accessToken)) {
    map.set(item.registerId, presentRegister(accessToken, item))
  }
  return { items: [...map.values()] }
}

export function mockGetOrderRegister(accessToken, registerId) {
  mockMe(accessToken)
  const id = typeof registerId === 'string' ? registerId.trim() : ''
  const found = findRegister(accessToken, id)
  if (!found) throw apiError(404, 'NOT_FOUND', '未找到有效挂号', { requestId: requestId() })
  return found
}

function copyOrder(order) {
  return assertOrderDto(order)
}

export function mockListOrders(accessToken, input) {
  mockMe(accessToken)
  const source = input && typeof input === 'object' ? input : {}
  const registerId = (typeof source.registerId === 'string' && source.registerId.trim())
    || (typeof source.encounterId === 'string' && source.encounterId.trim())
    || ''
  const state = readState()
  const items = state.orders
    .filter((item) => !registerId || item.registerId === registerId)
    .map((item) => copyOrder(item))
  return { items }
}

function applyHeader(register, request, previous) {
  const patientId = request.patientId || (previous && previous.patientId) || register.patientId
  if (patientId !== register.patientId) {
    throw apiError(400, 'VALIDATION_ERROR', '患者与挂号不一致', { requestId: requestId() })
  }
  return {
    patientId,
    departmentId: request.departmentId || (previous && previous.departmentId) || register.departmentId,
    doctorId: request.doctorId || (previous && previous.doctorId) || register.doctorId,
    diagnoseName: Object.prototype.hasOwnProperty.call(request, 'diagnoseName')
      ? request.diagnoseName
      : (previous && previous.diagnoseName) || null,
    recipeSort: request.recipeSort || (previous && previous.recipeSort) || 'NORMAL',
  }
}

function materializeItems(items, doctorId) {
  return items.map((item) => ({
    ...item,
    orderItemId: crypto.randomUUID(),
    doctorId,
  }))
}

export function mockCreateOrder(accessToken, input, idempotencyKey) {
  const user = mockMe(accessToken)
  const request = assertCreateOrderRequest(input)
  const register = requireOrderable(accessToken, request.registerId)
  const state = readState()
  const key = typeof idempotencyKey === 'string' ? idempotencyKey.trim() : ''
  if (key && state.idempotency[key]) {
    const prior = state.idempotency[key]
    if (prior.fingerprint !== JSON.stringify(request)) {
      throw apiError(409, 'IDEMPOTENCY_KEY_CONFLICT', '重复提交的内容不一致', { requestId: requestId() })
    }
    return copyOrder(prior.order)
  }
  const header = applyHeader(register, request, null)
  state.seq += 1
  const day = todayInShanghai().replace(/-/g, '')
  const now = new Date().toISOString()
  const items = materializeItems(request.items, header.doctorId)
  const order = {
    orderId: crypto.randomUUID(),
    orderNo: `R${day}${String(state.seq).padStart(4, '0')}`,
    registerId: register.registerId,
    encounterId: register.registerId,
    patientId: header.patientId,
    departmentId: header.departmentId,
    doctorId: header.doctorId,
    diagnoseName: header.diagnoseName,
    recipeSort: header.recipeSort,
    status: 'SAVE',
    totalFee: round2(items.reduce((sum, item) => sum + item.totalFee, 0)),
    charged: false,
    items,
    submittedAt: null,
    abatedAt: null,
    abatedBy: null,
    operatedAt: now,
    operatorId: user.id,
  }
  state.orders.unshift(order)
  if (key) state.idempotency[key] = { fingerprint: JSON.stringify(request), order }
  writeState(state)
  return copyOrder(order)
}

export function mockPatchOrder(accessToken, orderId, input) {
  const user = mockMe(accessToken)
  const request = assertPatchOrderRequest(input)
  const id = typeof orderId === 'string' ? orderId.trim() : ''
  const state = readState()
  const current = state.orders.find((item) => item.orderId === id)
  if (!current) throw apiError(404, 'NOT_FOUND', '未找到处方', { requestId: requestId() })
  if (current.status !== 'SAVE' || current.charged) {
    throw apiError(409, 'ORDER_NOT_EDITABLE', '已提交的处方不能修改', { requestId: requestId() })
  }
  if (request.registerId && request.registerId !== current.registerId) {
    throw apiError(409, 'ORDER_NOT_EDITABLE', '不能更换处方所属挂号', { requestId: requestId() })
  }
  const register = requireOrderable(accessToken, current.registerId)
  const header = applyHeader(register, request, current)
  const items = Object.prototype.hasOwnProperty.call(request, 'items')
    ? materializeItems(request.items, header.doctorId)
    : current.items
  const order = {
    ...current,
    patientId: header.patientId,
    departmentId: header.departmentId,
    doctorId: header.doctorId,
    diagnoseName: header.diagnoseName,
    recipeSort: header.recipeSort,
    encounterId: current.registerId,
    items,
    totalFee: round2(items.reduce((sum, item) => sum + item.totalFee, 0)),
    operatedAt: new Date().toISOString(),
    operatorId: user.id,
  }
  const index = state.orders.findIndex((item) => item.orderId === id)
  state.orders.splice(index, 1, order)
  writeState(state)
  return copyOrder(order)
}

export function mockCommitOrder(accessToken, orderId) {
  const user = mockMe(accessToken)
  const id = typeof orderId === 'string' ? orderId.trim() : ''
  const state = readState()
  const current = state.orders.find((item) => item.orderId === id)
  if (!current) throw apiError(404, 'NOT_FOUND', '未找到处方', { requestId: requestId() })
  if (current.status !== 'SAVE' || current.charged) {
    throw apiError(409, 'ORDER_NOT_COMMITTABLE', '当前状态不能提交', { requestId: requestId() })
  }
  requireOrderable(accessToken, current.registerId)
  if (!Array.isArray(current.items) || current.items.length === 0) {
    throw apiError(409, 'ORDER_EMPTY_ITEMS', '请至少添加一行明细', { requestId: requestId() })
  }
  for (const item of current.items) {
    if (!(Number(item.amount) > 0)) {
      throw apiError(409, 'ORDER_INVALID_AMOUNT', '数量必须大于 0', { requestId: requestId() })
    }
  }
  const now = new Date().toISOString()
  const order = {
    ...current,
    encounterId: current.registerId,
    status: 'COMMIT',
    totalFee: round2(current.items.reduce((sum, item) => sum + item.totalFee, 0)),
    submittedAt: now,
    operatedAt: now,
    operatorId: user.id,
  }
  const index = state.orders.findIndex((item) => item.orderId === id)
  state.orders.splice(index, 1, order)
  writeState(state)
  return copyOrder(order)
}

export function mockAbateOrder(accessToken, orderId) {
  const user = mockMe(accessToken)
  const id = typeof orderId === 'string' ? orderId.trim() : ''
  const state = readState()
  const current = state.orders.find((item) => item.orderId === id)
  if (!current) throw apiError(404, 'NOT_FOUND', '未找到处方', { requestId: requestId() })
  if (current.charged) {
    throw apiError(409, 'ORDER_CHARGED_NOT_ABATABLE', '已收费，不能作废', { requestId: requestId() })
  }
  if (current.status !== 'SAVE' && current.status !== 'COMMIT') {
    throw apiError(409, 'ILLEGAL_STATE_TRANSITION', '当前状态不能作废', { requestId: requestId() })
  }
  const now = new Date().toISOString()
  const order = {
    ...current,
    encounterId: current.registerId,
    status: 'ABATE',
    abatedAt: now,
    abatedBy: user.id,
    operatedAt: now,
    operatorId: user.id,
  }
  const index = state.orders.findIndex((item) => item.orderId === id)
  state.orders.splice(index, 1, order)
  writeState(state)
  return copyOrder(order)
}

export function mockMarkOrderCharged(orderId) {
  const state = readState()
  const current = state.orders.find((item) => item.orderId === orderId)
  if (!current) throw apiError(404, 'NOT_FOUND', '未找到处方', { requestId: requestId() })
  current.charged = true
  writeState(state)
  return copyOrder(current)
}
