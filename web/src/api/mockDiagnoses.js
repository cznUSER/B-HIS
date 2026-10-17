// 2026-10-15 P2-DIAGNOSIS-UI 本地诊断 Mock，仅在 VITE_USE_MOCK=true 时使用。
import { apiError } from './errors.js'
import { mockMe } from './mockAuth.js'
import { mockGetOrderRegister } from './mockOrders.js'
import { assertCreateDiagnoseRequest, assertDiagnoseDto } from './diagnosisContract.js'

const STORAGE_KEY = 'bhis.diagnoses.v1'
const memory = { diagnoses: [] }

function storage() {
  try { return typeof sessionStorage === 'undefined' ? null : sessionStorage } catch { return null }
}

function readState() {
  const store = storage()
  if (!store) return memory
  try {
    const parsed = JSON.parse(store.getItem(STORAGE_KEY) || '{}')
    return { diagnoses: Array.isArray(parsed.diagnoses) ? parsed.diagnoses : [] }
  } catch {
    return { diagnoses: [] }
  }
}

function writeState(state) {
  const store = storage()
  if (!store) {
    memory.diagnoses = state.diagnoses
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

export function resetMockDiagnoses() {
  memory.diagnoses = []
  const store = storage()
  if (store) store.removeItem(STORAGE_KEY)
}

export function mockListDiagnoses(accessToken, registerId) {
  requireRegister(accessToken, registerId)
  return { items: readState().diagnoses.filter((item) => item.encounterId === registerId).map(copy) }
}

export function mockCreateDiagnose(accessToken, registerId, input) {
  requireRegister(accessToken, registerId)
  const request = assertCreateDiagnoseRequest(input)
  const state = readState()
  const row = copy({
    diagnoseId: crypto.randomUUID(),
    encounterId: registerId,
    status: 'SAVE',
    content: request.content,
  })
  state.diagnoses.push(row)
  writeState(state)
  return copy(row)
}

function action(accessToken, diagnoseId, nextStatus) {
  mockMe(accessToken)
  const id = typeof diagnoseId === 'string' ? diagnoseId.trim() : ''
  if (!id) throw apiError(404, 'NOT_FOUND', '未找到诊断')
  const state = readState()
  const index = state.diagnoses.findIndex((item) => item.diagnoseId === id)
  if (index < 0) throw apiError(404, 'NOT_FOUND', '未找到诊断')
  const current = state.diagnoses[index]
  if (nextStatus === 'COMMIT' && !['SAVE', 'NEW'].includes(current.status)) {
    throw apiError(409, 'ILLEGAL_STATE_TRANSITION', '当前诊断状态不允许提交')
  }
  if (nextStatus === 'ABATE' && !['SAVE', 'COMMIT'].includes(current.status)) {
    throw apiError(409, 'ILLEGAL_STATE_TRANSITION', '当前诊断状态不允许作废')
  }
  const row = copy({ ...current, status: nextStatus })
  state.diagnoses.splice(index, 1, row)
  writeState(state)
  return copy(row)
}

export function mockCommitDiagnose(accessToken, diagnoseId) { return action(accessToken, diagnoseId, 'COMMIT') }
export function mockAbateDiagnose(accessToken, diagnoseId) { return action(accessToken, diagnoseId, 'ABATE') }
