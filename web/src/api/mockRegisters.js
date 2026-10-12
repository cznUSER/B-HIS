// 2026-10-12 P1-H05-REGISTER-UI 本地挂号 Mock，形状与 POST/GET /registers 相同。
// 仅在 VITE_USE_MOCK 打开时使用。保存后 status=NORMAL；退号只接受正常号。

import { apiError } from './errors.js'
import { mockMe } from './mockAuth.js'
import { mockGetPatient } from './mockPatients.js'
import { assertCreateRegisterRequest, assertRegisterDto } from './registerContract.js'
import { cancelRejectMessage, canCancel } from '../register/master.js'
import { todayInShanghai } from './patientContract.js'

const STORAGE_KEY = 'bhis.registers.v1'

const memory = {
  seq: 0,
  registers: [],
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
    if (!raw) return { seq: 0, registers: [], idempotency: {} }
    const parsed = JSON.parse(raw)
    return {
      seq: typeof parsed.seq === 'number' ? parsed.seq : 0,
      registers: Array.isArray(parsed.registers) ? parsed.registers : [],
      idempotency: parsed.idempotency && typeof parsed.idempotency === 'object' ? parsed.idempotency : {},
    }
  } catch {
    return { seq: 0, registers: [], idempotency: {} }
  }
}

function writeState(state) {
  const store = storage()
  if (!store) {
    memory.seq = state.seq
    memory.registers = state.registers
    memory.idempotency = state.idempotency
    return
  }
  store.setItem(STORAGE_KEY, JSON.stringify(state))
}

function requireSession(accessToken) {
  return mockMe(accessToken)
}

function copyRegister(row) {
  return assertRegisterDto(row)
}

function rejectCancel(status) {
  const code = status === 'BACK' ? 'REGISTER_BACK_NOT_ALLOWED' : 'REGISTER_NOT_CANCELLABLE'
  throw apiError(409, code, cancelRejectMessage(status), {
    requestId: `mock-${crypto.randomUUID()}`,
  })
}

export function resetMockRegisters() {
  memory.seq = 0
  memory.registers = []
  memory.idempotency = {}
  const store = storage()
  if (store) store.removeItem(STORAGE_KEY)
}

export function stageMockRegister(accessToken, input) {
  requireSession(accessToken)
  const src = input && typeof input === 'object' ? input : {}
  const state = readState()
  const row = copyRegister({
    registerId: typeof src.registerId === 'string' && src.registerId.trim() ? src.registerId.trim() : crypto.randomUUID(),
    registerNo: typeof src.registerNo === 'string' && src.registerNo.trim() ? src.registerNo.trim() : 'GHSTAGE',
    patientId: typeof src.patientId === 'string' ? src.patientId.trim() : '',
    departmentId: typeof src.departmentId === 'string' ? src.departmentId.trim() : 'dept-1',
    doctorId: typeof src.doctorId === 'string' ? src.doctorId.trim() : 'doc-1',
    registerSort: 'sort-normal',
    visitDate: typeof src.visitDate === 'string' ? src.visitDate.trim() : todayInShanghai(),
    status: src.status,
    registerFee: 0,
    registerCharged: false,
    registeredAt: typeof src.registeredAt === 'string' ? src.registeredAt : '2026-10-12T00:00:00.000Z',
  })
  state.registers.unshift(row)
  writeState(state)
  return copyRegister(row)
}

export function mockCreateRegister(accessToken, input, idempotencyKey) {
  requireSession(accessToken)
  const request = assertCreateRegisterRequest(input)
  mockGetPatient(accessToken, request.patientId)
  const state = readState()
  const key = typeof idempotencyKey === 'string' ? idempotencyKey.trim() : ''
  if (key && state.idempotency[key]) {
    const prior = state.idempotency[key]
    if (prior.fingerprint !== JSON.stringify(request)) {
      throw apiError(409, 'IDEMPOTENCY_KEY_CONFLICT', '重复提交的内容不一致', {
        requestId: `mock-${crypto.randomUUID()}`,
      })
    }
    return copyRegister(prior.register)
  }

  state.seq += 1
  const serial = String(state.seq).padStart(4, '0')
  const row = copyRegister({
    registerId: crypto.randomUUID(),
    registerNo: `GH${request.visitDate.replace(/-/g, '')}${serial}`,
    patientId: request.patientId,
    departmentId: request.departmentId,
    doctorId: request.doctorId,
    registerSort: request.registerSort,
    visitDate: request.visitDate,
    status: 'NORMAL',
    registerFee: request.registerFee,
    registerCharged: false,
    registeredAt: new Date().toISOString(),
  })
  state.registers.unshift(row)
  if (key) state.idempotency[key] = { fingerprint: JSON.stringify(request), register: row }
  writeState(state)
  return copyRegister(row)
}

export function mockListRegisters(accessToken, input) {
  requireSession(accessToken)
  const patientId = input && typeof input.patientId === 'string' ? input.patientId.trim() : ''
  if (!patientId) throw apiError(400, 'VALIDATION_ERROR', '请先选择或建档患者')
  mockGetPatient(accessToken, patientId)
  const visitDate = input && typeof input.visitDate === 'string' && input.visitDate.trim()
    ? input.visitDate.trim()
    : todayInShanghai()
  const state = readState()
  const items = state.registers
    .filter((item) => item.patientId === patientId && item.visitDate === visitDate)
    .map((item) => copyRegister(item))
    .sort((a, b) => (a.registeredAt < b.registeredAt ? 1 : -1))
  return { items }
}

export function mockCancelRegister(accessToken, registerId, idempotencyKey) {
  const user = requireSession(accessToken)
  const id = typeof registerId === 'string' ? registerId.trim() : ''
  if (!id) throw apiError(404, 'NOT_FOUND', '未找到挂号单')
  const state = readState()
  const key = typeof idempotencyKey === 'string' ? idempotencyKey.trim() : ''
  if (key && state.idempotency[key]) {
    const prior = state.idempotency[key]
    if (prior.registerId !== id) {
      throw apiError(409, 'IDEMPOTENCY_KEY_CONFLICT', '重复提交的内容不一致', {
        requestId: `mock-${crypto.randomUUID()}`,
      })
    }
    return copyRegister(prior.register)
  }

  const index = state.registers.findIndex((item) => item.registerId === id)
  if (index < 0) throw apiError(404, 'NOT_FOUND', '未找到挂号单')
  const current = state.registers[index]
  if (!canCancel(current.status)) rejectCancel(current.status)

  const next = copyRegister({
    ...current,
    status: 'BACK',
    backOperatorId: user.id,
    backOperatedAt: new Date().toISOString(),
  })
  state.registers.splice(index, 1, next)
  if (key) state.idempotency[key] = { registerId: id, register: next }
  writeState(state)
  return copyRegister(next)
}
