// 2026-10-15 P2-DIAGNOSIS-UI 本地诊断 Mock，仅在 VITE_USE_MOCK=true 时使用。
import { apiError } from './errors.js'
import { mockMe } from './mockAuth.js'
import { mockGetOrderRegister } from './mockOrders.js'
import { assertAmendDiagnoseRequest, assertCreateDiagnoseRequest, assertDiagnoseDto } from './diagnosisContract.js'

const STORAGE_KEY = 'bhis.diagnoses.v1'
const memory = { diagnoses: [], idempotency: {} }

function storage() {
  try { return typeof sessionStorage === 'undefined' ? null : sessionStorage } catch { return null }
}

function readState() {
  const store = storage()
  if (!store) return memory
  try {
    const parsed = JSON.parse(store.getItem(STORAGE_KEY) || '{}')
    return {
      diagnoses: Array.isArray(parsed.diagnoses) ? parsed.diagnoses : [],
      idempotency: parsed.idempotency && typeof parsed.idempotency === 'object' ? parsed.idempotency : {},
    }
  } catch {
    return { diagnoses: [], idempotency: {} }
  }
}

function writeState(state) {
  const store = storage()
  if (!store) {
    memory.diagnoses = state.diagnoses
    memory.idempotency = state.idempotency
    return
  }
  store.setItem(STORAGE_KEY, JSON.stringify(state))
}

function requireRegister(accessToken, registerId) {
  mockMe(accessToken)
  const id = typeof registerId === 'string' ? registerId.trim() : ''
  if (!id) throw apiError(404, 'NOT_FOUND', '未找到挂号')
  const register = mockGetOrderRegister(accessToken, id)
  if (!register || register.status === 'BACK') throw apiError(409, 'ENCOUNTER_REQUIRED', '该挂号当前不能录入诊断')
  return register
}

function copy(row) { return assertDiagnoseDto(row) }

function recallIdempotent(state, key, fingerprint) {
  if (!key || !state.idempotency[key]) return null
  const prior = state.idempotency[key]
  if (prior.fingerprint !== fingerprint) {
    throw apiError(409, 'IDEMPOTENCY_KEY_CONFLICT', '重复提交的内容不一致', {
      requestId: `mock-${crypto.randomUUID()}`,
    })
  }
  return copy(prior.diagnose)
}

function storeIdempotent(state, key, fingerprint, diagnose) {
  if (!key) return
  state.idempotency[key] = { fingerprint, diagnose }
}

export function resetMockDiagnoses() {
  memory.diagnoses = []
  memory.idempotency = {}
  const store = storage()
  if (store) store.removeItem(STORAGE_KEY)
}

export function mockListDiagnoses(accessToken, registerId) {
  requireRegister(accessToken, registerId)
  return { items: readState().diagnoses.filter((item) => item.encounterId === registerId).map(copy) }
}

export function mockCreateDiagnose(accessToken, registerId, input, idempotencyKey) {
  requireRegister(accessToken, registerId)
  const request = assertCreateDiagnoseRequest(input)
  const state = readState()
  const key = typeof idempotencyKey === 'string' ? idempotencyKey.trim() : ''
  const fingerprint = JSON.stringify({ kind: 'create', registerId, request })
  const recalled = recallIdempotent(state, key, fingerprint)
  if (recalled) return recalled
  const row = copy({
    diagnoseId: crypto.randomUUID(),
    encounterId: registerId,
    status: 'SAVE',
    content: request.content,
    structured: request.structured,
    templateCode: request.templateCode,
    version: state.diagnoses.filter((item) => item.encounterId === registerId).length + 1,
  })
  state.diagnoses.push(row)
  storeIdempotent(state, key, fingerprint, row)
  writeState(state)
  return copy(row)
}

export function mockAmendDiagnose(accessToken, diagnoseId, input, idempotencyKey) {
  mockMe(accessToken)
  const id = typeof diagnoseId === 'string' ? diagnoseId.trim() : ''
  if (!id) throw apiError(404, 'NOT_FOUND', '未找到诊断')
  const state = readState()
  const previous = state.diagnoses.find((item) => item.diagnoseId === id)
  if (!previous) throw apiError(404, 'NOT_FOUND', '未找到诊断')
  if (previous.status !== 'COMMIT') throw apiError(409, 'DIAGNOSE_NOT_AMENDABLE', '只有已提交诊断可以发起修订')
  const request = assertAmendDiagnoseRequest(input)
  const key = typeof idempotencyKey === 'string' ? idempotencyKey.trim() : ''
  const fingerprint = JSON.stringify({ kind: 'amend', diagnoseId: id, request })
  const recalled = recallIdempotent(state, key, fingerprint)
  if (recalled) return recalled
  const row = copy({
    diagnoseId: crypto.randomUUID(),
    encounterId: previous.encounterId,
    status: 'SAVE',
    content: request.content,
    structured: request.structured,
    templateCode: request.templateCode || previous.templateCode,
    version: state.diagnoses.filter((item) => item.encounterId === previous.encounterId).length + 1,
    supersedesDiagnoseId: previous.diagnoseId,
    revisionReason: request.revisionReason,
  })
  state.diagnoses.push(row)
  storeIdempotent(state, key, fingerprint, row)
  writeState(state)
  return copy(row)
}

function action(accessToken, diagnoseId, nextStatus, idempotencyKey) {
  mockMe(accessToken)
  const id = typeof diagnoseId === 'string' ? diagnoseId.trim() : ''
  if (!id) throw apiError(404, 'NOT_FOUND', '未找到诊断')
  const state = readState()
  const key = typeof idempotencyKey === 'string' ? idempotencyKey.trim() : ''
  const fingerprint = JSON.stringify({ kind: nextStatus === 'COMMIT' ? 'commit' : 'abate', diagnoseId: id })
  const recalled = recallIdempotent(state, key, fingerprint)
  if (recalled) return recalled
  const index = state.diagnoses.findIndex((item) => item.diagnoseId === id)
  if (index < 0) throw apiError(404, 'NOT_FOUND', '未找到诊断')
  const current = state.diagnoses[index]
  if (nextStatus === 'COMMIT' && !['SAVE', 'NEW'].includes(current.status)) {
    throw apiError(409, 'ILLEGAL_STATE_TRANSITION', '当前诊断状态不允许提交')
  }
  if (nextStatus === 'ABATE' && !['SAVE', 'COMMIT'].includes(current.status)) {
    throw apiError(409, 'ILLEGAL_STATE_TRANSITION', '当前诊断状态不允许作废')
  }
  const row = copy({
    ...current,
    status: nextStatus,
    signedAt: nextStatus === 'COMMIT' ? new Date().toISOString() : current.signedAt,
    signedBy: nextStatus === 'COMMIT' ? mockMe(accessToken).id : current.signedBy,
  })
  state.diagnoses.splice(index, 1, row)
  storeIdempotent(state, key, fingerprint, row)
  writeState(state)
  return copy(row)
}

export function mockCommitDiagnose(accessToken, diagnoseId, idempotencyKey) {
  return action(accessToken, diagnoseId, 'COMMIT', idempotencyKey)
}
export function mockAbateDiagnose(accessToken, diagnoseId, idempotencyKey) {
  return action(accessToken, diagnoseId, 'ABATE', idempotencyKey)
}
