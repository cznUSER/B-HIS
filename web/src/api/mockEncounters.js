// 2026-10-12 P1-H05-REGISTER-UI 本地挂号 Mock。
// 保存后 status=NORMAL（正常号）。退号只接受正常号，成功后为 BACK（退号）。

import { apiError } from './errors.js'
import { mockMe } from './mockAuth.js'
import { mockGetPatient } from './mockPatients.js'
import { assertCreateEncounterRequest, assertEncounterDto } from './encounterContract.js'
import { cancelRejectMessage, canCancel } from '../register/master.js'
import { todayInShanghai } from './patientContract.js'

const STORAGE_KEY = 'bhis.encounters.v1'

const memory = {
  seq: 0,
  encounters: [],
  idempotency: {},
}

const knownSeeds = new Map()

const SEED_SPECS = [
  { status: 'USED', serial: '9001', deptCode: 'dept-1', doctorId: 'doc-1' },
  { status: 'DIAGNOSING', serial: '9002', deptCode: 'dept-1', doctorId: 'doc-1' },
  { status: 'DIAGNOSED', serial: '9003', deptCode: 'dept-2', doctorId: 'doc-2' },
  { status: 'PAUSED', serial: '9004', deptCode: 'dept-2', doctorId: 'doc-2' },
  { status: 'BACK', serial: '9005', deptCode: 'dept-1', doctorId: 'doc-1' },
]

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
    if (!raw) return { seq: 0, encounters: [], idempotency: {} }
    const parsed = JSON.parse(raw)
    return {
      seq: typeof parsed.seq === 'number' ? parsed.seq : 0,
      encounters: Array.isArray(parsed.encounters) ? parsed.encounters : [],
      idempotency: parsed.idempotency && typeof parsed.idempotency === 'object' ? parsed.idempotency : {},
    }
  } catch {
    return { seq: 0, encounters: [], idempotency: {} }
  }
}

function writeState(state) {
  const store = storage()
  if (!store) {
    memory.seq = state.seq
    memory.encounters = state.encounters
    memory.idempotency = state.idempotency
    return
  }
  store.setItem(STORAGE_KEY, JSON.stringify(state))
}

function requireSession(accessToken) {
  return mockMe(accessToken)
}

function copyEncounter(encounter) {
  return assertEncounterDto(encounter)
}

function seedId(patientId, status) {
  const raw = `${status}:${patientId}`
  const seeds = [2166136261, 2166136261 ^ 0x9e3779b9, 2166136261 ^ 0x85ebca6b, 2166136261 ^ 0xc2b2ae35]
  const mul = [16777619, 2246822519, 3266489917, 668265263]
  const hash = seeds.slice()
  for (let i = 0; i < raw.length; i += 1) {
    const code = raw.charCodeAt(i)
    for (let part = 0; part < hash.length; part += 1) {
      hash[part] = Math.imul(hash[part] ^ code, mul[part]) >>> 0
    }
  }
  const mark = { USED: '1', DIAGNOSING: '2', DIAGNOSED: '3', PAUSED: '4', BACK: '5' }[status] || '0'
  const hex = hash.map((part) => part.toString(16).padStart(8, '0')).join('')
  return `${hex.slice(0, 8)}-${mark}${hex.slice(9, 12)}-4${hex.slice(13, 16)}-a${hex.slice(17, 20)}-${hex.slice(20, 32)}`
}

function seedEncounters(patientId, visitDate) {
  const day = visitDate.replace(/-/g, '')
  return SEED_SPECS.map((spec, index) => {
    const encounter = {
      encounterId: seedId(patientId, spec.status),
      encounterNo: `GH${day}${spec.serial}`,
      patientId,
      deptCode: spec.deptCode,
      doctorId: spec.doctorId,
      clinicSort: 'sort-normal',
      visitDate,
      status: spec.status,
      registerFee: 0,
      registerCharged: false,
      registeredAt: `${visitDate}T00:00:0${index}.000Z`,
    }
    if (spec.status === 'BACK') {
      encounter.backOperatorId = 'seed'
      encounter.backOperatedAt = `${visitDate}T00:10:00.000Z`
    }
    return copyEncounter(encounter)
  })
}

function rejectCancel(status) {
  throw apiError(409, 'REGISTER_BACK_NOT_ALLOWED', cancelRejectMessage(status), {
    requestId: `mock-${crypto.randomUUID()}`,
  })
}

export function resetMockEncounters() {
  memory.seq = 0
  memory.encounters = []
  memory.idempotency = {}
  knownSeeds.clear()
  const store = storage()
  if (store) store.removeItem(STORAGE_KEY)
}

export function mockCreateEncounter(accessToken, input, idempotencyKey) {
  requireSession(accessToken)
  const request = assertCreateEncounterRequest(input)
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
    return copyEncounter(prior.encounter)
  }

  state.seq += 1
  const serial = String(state.seq).padStart(4, '0')
  const encounter = copyEncounter({
    encounterId: crypto.randomUUID(),
    encounterNo: `GH${request.visitDate.replace(/-/g, '')}${serial}`,
    patientId: request.patientId,
    deptCode: request.deptCode,
    doctorId: request.doctorId,
    clinicSort: request.clinicSort,
    visitDate: request.visitDate,
    status: 'NORMAL',
    registerFee: request.registerFee,
    registerCharged: false,
    registeredAt: new Date().toISOString(),
  })
  state.encounters.unshift(encounter)
  if (key) state.idempotency[key] = { fingerprint: JSON.stringify(request), encounter }
  writeState(state)
  return copyEncounter(encounter)
}

export function mockListEncounters(accessToken, input) {
  requireSession(accessToken)
  const patientId = input && typeof input.patientId === 'string' ? input.patientId.trim() : ''
  if (!patientId) throw apiError(400, 'VALIDATION_ERROR', '请先选择或建档患者')
  mockGetPatient(accessToken, patientId)
  const visitDate = input && typeof input.visitDate === 'string' && input.visitDate.trim()
    ? input.visitDate.trim()
    : todayInShanghai()
  const state = readState()
  const saved = state.encounters
    .filter((item) => item.patientId === patientId && item.visitDate === visitDate)
    .map((item) => copyEncounter(item))
  const savedIds = new Set(saved.map((item) => item.encounterId))
  const seeds = seedEncounters(patientId, visitDate).filter((item) => !savedIds.has(item.encounterId))
  for (const item of seeds) knownSeeds.set(item.encounterId, item)
  const items = [...saved, ...seeds].sort((a, b) => (a.registeredAt < b.registeredAt ? 1 : -1))
  return { items }
}

export function mockCancelEncounter(accessToken, encounterId, idempotencyKey) {
  const user = requireSession(accessToken)
  const id = typeof encounterId === 'string' ? encounterId.trim() : ''
  if (!id) throw apiError(404, 'NOT_FOUND', '未找到挂号单')
  const state = readState()
  const key = typeof idempotencyKey === 'string' ? idempotencyKey.trim() : ''
  if (key && state.idempotency[key]) {
    const prior = state.idempotency[key]
    if (prior.encounterId !== id) {
      throw apiError(409, 'IDEMPOTENCY_KEY_CONFLICT', '重复提交的内容不一致', {
        requestId: `mock-${crypto.randomUUID()}`,
      })
    }
    return copyEncounter(prior.encounter)
  }

  const index = state.encounters.findIndex((item) => item.encounterId === id)
  const current = index >= 0 ? state.encounters[index] : knownSeeds.get(id)
  if (!current) throw apiError(404, 'NOT_FOUND', '未找到挂号单')
  if (!canCancel(current.status)) rejectCancel(current.status)

  const next = copyEncounter({
    ...current,
    status: 'BACK',
    backOperatorId: user.id,
    backOperatedAt: new Date().toISOString(),
  })
  if (index >= 0) state.encounters.splice(index, 1, next)
  else state.encounters.unshift(next)
  if (key) state.idempotency[key] = { encounterId: id, encounter: next }
  writeState(state)
  return copyEncounter(next)
}
