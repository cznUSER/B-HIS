// 2026-10-12 P1-H05-PATIENT-UI 本地患者 Mock。
// 响应形状对齐 PatientDto（含 address）。证件重复拒绝，不合并。

import { apiError } from './errors.js'
import { mockMe } from './mockAuth.js'
import {
  assertCreatePatientRequest,
  assertPatientDto,
  assertPatientSearchQuery,
  patientMatchesQuery,
  todayInShanghai,
} from './patientContract.js'

const STORAGE_KEY = 'bhis.patients.v1'

const memory = {
  seq: 0,
  patients: [],
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
    if (!raw) return { seq: 0, patients: [], idempotency: {} }
    const parsed = JSON.parse(raw)
    return {
      seq: typeof parsed.seq === 'number' ? parsed.seq : 0,
      patients: Array.isArray(parsed.patients) ? parsed.patients : [],
      idempotency: parsed.idempotency && typeof parsed.idempotency === 'object' ? parsed.idempotency : {},
    }
  } catch {
    return { seq: 0, patients: [], idempotency: {} }
  }
}

function writeState(state) {
  const store = storage()
  if (!store) {
    memory.seq = state.seq
    memory.patients = state.patients
    memory.idempotency = state.idempotency
    return
  }
  store.setItem(STORAGE_KEY, JSON.stringify(state))
}

function requireSession(accessToken) {
  mockMe(accessToken)
}

function copyPatient(patient) {
  return assertPatientDto(patient)
}

export function resetMockPatients() {
  memory.seq = 0
  memory.patients = []
  memory.idempotency = {}
  const store = storage()
  if (store) store.removeItem(STORAGE_KEY)
}

export function mockCreatePatient(accessToken, input, idempotencyKey) {
  requireSession(accessToken)
  const request = assertCreatePatientRequest(input)
  const state = readState()
  const key = typeof idempotencyKey === 'string' ? idempotencyKey.trim() : ''
  if (key && state.idempotency[key]) {
    const prior = state.idempotency[key]
    if (prior.fingerprint !== JSON.stringify(request)) {
      throw apiError(409, 'IDEMPOTENCY_KEY_CONFLICT', '重复提交的内容不一致', {
        requestId: `mock-${crypto.randomUUID()}`,
      })
    }
    return copyPatient(prior.patient)
  }

  if (request.idNumber) {
    const existing = state.patients.find((item) => item.idNumber === request.idNumber)
    if (existing) {
      throw apiError(409, 'DUPLICATE_ID_NUMBER', '该身份证号已存在，不能重复建档', {
        details: { patientId: existing.patientId, fields: ['idNumber'] },
        requestId: `mock-${crypto.randomUUID()}`,
      })
    }
  }
  if (request.cardNo) {
    const existing = state.patients.find((item) => item.cardNo === request.cardNo)
    if (existing) {
      throw apiError(409, 'DUPLICATE_CARD_NO', '卡号已存在，不能重复建档', {
        details: { patientId: existing.patientId, fields: ['cardNo'] },
        requestId: `mock-${crypto.randomUUID()}`,
      })
    }
  }

  state.seq += 1
  const day = todayInShanghai().replace(/-/g, '')
  const serial = String(state.seq).padStart(4, '0')
  const patient = assertPatientDto({
    patientId: crypto.randomUUID(),
    hospPkNo: `H${day}${serial}`,
    patientNo: `P${day}${serial}`,
    cardNo: request.cardNo || '',
    name: request.name,
    sex: request.sex,
    birthday: request.birthday,
    idNumber: request.idNumber || '',
    phone: request.phone || '',
    address: request.address || '',
    fareIdentity: request.fareIdentity,
    cardStatus: 'NOT_SENT',
    status: 'ACTIVE',
    createdAt: new Date().toISOString(),
  })
  state.patients.unshift(patient)
  if (key) state.idempotency[key] = { fingerprint: JSON.stringify(request), patient }
  writeState(state)
  return copyPatient(patient)
}

export function mockSearchPatients(accessToken, input) {
  requireSession(accessToken)
  const query = assertPatientSearchQuery(input)
  const state = readState()
  const items = state.patients.filter((item) => patientMatchesQuery(item, query)).map((item) => copyPatient(item))
  return { items }
}

export function mockGetPatient(accessToken, patientId) {
  requireSession(accessToken)
  const id = typeof patientId === 'string' ? patientId.trim() : ''
  const state = readState()
  const found = state.patients.find((item) => item.patientId === id)
  if (!found) {
    throw apiError(404, 'NOT_FOUND', '未找到患者', {
      requestId: `mock-${crypto.randomUUID()}`,
    })
  }
  return copyPatient(found)
}
