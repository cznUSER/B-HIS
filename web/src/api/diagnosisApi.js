// 2026-10-15 P2-DIAGNOSIS-UI 诊断真接口：挂在挂号下，支持保存、提交、作废。
import { apiUrl, useMock } from './config.js'
import { apiError, readErrorResponse } from './errors.js'
import {
  assertCreateDiagnoseRequest,
  assertDiagnoseDto,
  assertDiagnoseList,
} from './diagnosisContract.js'
import {
  mockAbateDiagnose,
  mockCommitDiagnose,
  mockCreateDiagnose,
  mockListDiagnoses,
} from './mockDiagnoses.js'

const JSON_ACCEPT = 'application/json'

export const DIAGNOSIS_ROUTES = {
  collection: (registerId) => `/registers/${encodeURIComponent(registerId)}/diagnoses`,
  commit: (diagnoseId) => `/diagnoses/${encodeURIComponent(diagnoseId)}/commit`,
  abate: (diagnoseId) => `/diagnoses/${encodeURIComponent(diagnoseId)}/abate`,
}

function bearer(accessToken) { return accessToken ? { Authorization: `Bearer ${accessToken}` } : {} }
function requestInit(init) { return { credentials: 'omit', cache: 'no-store', ...init } }
function delay(ms) { return new Promise((resolve) => setTimeout(resolve, ms)) }

async function parseJson(response) {
  const text = await response.text()
  if (!text) return null
  try { return JSON.parse(text) } catch { throw apiError(response.status, 'HTTP_ERROR', '响应无法解析') }
}

async function request(accessToken, path, init, offlineMessage) {
  let response
  try {
    response = await fetch(apiUrl(path), requestInit({
      ...init,
      headers: { Accept: JSON_ACCEPT, ...bearer(accessToken), ...(init && init.headers) },
    }))
  } catch {
    throw apiError(0, 'HTTP_ERROR', offlineMessage)
  }
  if (!response.ok) throw await readErrorResponse(response, accessToken)
  return parseJson(response)
}

function requireId(value, label) {
  const id = typeof value === 'string' ? value.trim() : ''
  if (!id) throw apiError(404, 'NOT_FOUND', `未找到${label}`)
  return id
}

export async function listDiagnoses(accessToken, registerId) {
  const id = requireId(registerId, '挂号')
  if (useMock()) { await delay(80); return mockListDiagnoses(accessToken, id) }
  return assertDiagnoseList(await request(accessToken, DIAGNOSIS_ROUTES.collection(id), {
    method: 'GET',
  }, '无法连接诊断服务'))
}

export async function saveDiagnose(accessToken, registerId, input) {
  const id = requireId(registerId, '挂号')
  const body = assertCreateDiagnoseRequest(input)
  if (useMock()) { await delay(140); return mockCreateDiagnose(accessToken, id, body) }
  return assertDiagnoseDto(await request(accessToken, DIAGNOSIS_ROUTES.collection(id), {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json; charset=utf-8',
      'Idempotency-Key': crypto.randomUUID(),
    },
    body: JSON.stringify(body),
  }, '无法连接诊断服务'))
}

async function diagnoseAction(accessToken, diagnoseId, action, mockAction) {
  const id = requireId(diagnoseId, '诊断')
  if (useMock()) { await delay(120); return mockAction(accessToken, id) }
  return assertDiagnoseDto(await request(accessToken, action(id), {
    method: 'POST',
    headers: { 'Idempotency-Key': crypto.randomUUID() },
  }, '无法连接诊断服务'))
}

export function commitDiagnose(accessToken, diagnoseId) {
  return diagnoseAction(accessToken, diagnoseId, DIAGNOSIS_ROUTES.commit, mockCommitDiagnose)
}

export function abateDiagnose(accessToken, diagnoseId) {
  return diagnoseAction(accessToken, diagnoseId, DIAGNOSIS_ROUTES.abate, mockAbateDiagnose)
}
