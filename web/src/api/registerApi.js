// 2026-10-12 P1-H05-REGISTER-UI
// POST /api/v1/registers | GET /api/v1/registers | POST /api/v1/registers/{registerId}/cancel
// 2026-10-12 P1-H05b-REGISTER-CHECKIN-UI POST /api/v1/registers/{registerId}/check-in
// 2026-10-12 对齐已合并的 H04b：VITE_USE_MOCK=false 时打真实接诊。无请求体、无 Idempotency-Key。
// NORMAL 或 USED → DIAGNOSING。其它状态 409 ILLEGAL_STATE_TRANSITION，服务端文案「当前状态不允许该操作」。
// 无服务端 message 时页面仍用这句。simulateConflict 只在 Mock 生效，真实请求忽略。

import { apiUrl, useMock } from './config.js'
import { apiError, messageForCode, readErrorResponse } from './errors.js'
import { assertCreateRegisterRequest, assertRegisterDto, assertRegisterList } from './registerContract.js'
import { mockCancelRegister, mockCheckInRegister, mockCreateRegister, mockListRegisters } from './mockRegisters.js'
import { cancelRejectMessage } from '../register/master.js'

const JSON_ACCEPT = 'application/json'

function requestInit(init) {
  return {
    credentials: 'omit',
    cache: 'no-store',
    ...init,
  }
}

function delay(ms) {
  return new Promise((resolve) => {
    setTimeout(resolve, ms)
  })
}

function bearer(accessToken) {
  return accessToken ? { Authorization: `Bearer ${accessToken}` } : {}
}

async function parseJson(response) {
  const text = await response.text()
  if (!text) return null
  try {
    return JSON.parse(text)
  } catch {
    throw apiError(response.status, 'HTTP_ERROR', '响应无法解析')
  }
}

const GENERIC_FAILURE = '请求失败，请稍后重试'
const CHECKIN_CONFLICT = '当前状态不允许该操作'
const CHECKIN_FAILED = '报到失败，请稍后重试'
const CHECKIN_MISSING = '未找到挂号单'

function errorExtra(error) {
  return {
    details: error.details,
    requestId: error.requestId,
  }
}

function serverText(error) {
  return typeof error.message === 'string' ? error.message.trim() : ''
}

export function presentCheckInError(error) {
  const extra = errorExtra(error)
  const server = serverText(error)
  if (error.status === 409 || error.code === 'ILLEGAL_STATE_TRANSITION') {
    const message = server && server !== GENERIC_FAILURE ? server : CHECKIN_CONFLICT
    return apiError(error.status || 409, error.code || 'ILLEGAL_STATE_TRANSITION', message, extra)
  }
  if (error.status === 404 || error.code === 'NOT_FOUND') {
    const message = server && server !== '未找到患者' && server !== GENERIC_FAILURE ? server : CHECKIN_MISSING
    return apiError(error.status || 404, error.code || 'NOT_FOUND', message, extra)
  }
  if (typeof error.status === 'number' && error.status >= 500) {
    const message = server && server !== GENERIC_FAILURE && server !== messageForCode('HTTP_ERROR')
      ? server
      : CHECKIN_FAILED
    return apiError(error.status, error.code || 'HTTP_ERROR', message, extra)
  }
  return error
}

export function presentRegisterError(error) {
  if (error.code === 'REGISTER_NOT_CANCELLABLE') {
    return apiError(error.status || 409, error.code, cancelRejectMessage('USED'), {
      details: error.details,
      requestId: error.requestId,
    })
  }
  if (error.code === 'REGISTER_BACK_NOT_ALLOWED') {
    return apiError(error.status || 409, error.code, cancelRejectMessage('BACK'), {
      details: error.details,
      requestId: error.requestId,
    })
  }
  return error
}

export async function createRegister(accessToken, input) {
  const request = assertCreateRegisterRequest(input)
  const idempotencyKey = crypto.randomUUID()
  if (useMock()) {
    await delay(280)
    return mockCreateRegister(accessToken, request, idempotencyKey)
  }

  let response
  try {
    response = await fetch(apiUrl('/registers'), requestInit({
      method: 'POST',
      headers: {
        Accept: JSON_ACCEPT,
        'Content-Type': 'application/json; charset=utf-8',
        'Idempotency-Key': idempotencyKey,
        ...bearer(accessToken),
      },
      body: JSON.stringify(request),
    }))
  } catch {
    throw apiError(0, 'HTTP_ERROR', '无法连接挂号服务')
  }
  if (response.status !== 201) throw presentRegisterError(await readErrorResponse(response))
  return assertRegisterDto(await parseJson(response))
}

export async function listRegisters(accessToken, input) {
  const patientId = input && typeof input.patientId === 'string' ? input.patientId.trim() : ''
  const visitDate = input && typeof input.visitDate === 'string' ? input.visitDate.trim() : ''
  if (useMock()) {
    await delay(180)
    return mockListRegisters(accessToken, { patientId, visitDate })
  }

  const params = new URLSearchParams()
  if (patientId) params.set('patientId', patientId)
  if (visitDate) params.set('visitDate', visitDate)
  let response
  try {
    response = await fetch(`${apiUrl('/registers')}?${params.toString()}`, requestInit({
      method: 'GET',
      headers: {
        Accept: JSON_ACCEPT,
        ...bearer(accessToken),
      },
    }))
  } catch {
    throw apiError(0, 'HTTP_ERROR', '无法连接挂号服务')
  }
  if (!response.ok) throw presentRegisterError(await readErrorResponse(response))
  const listed = assertRegisterList(await parseJson(response))
  listed.items.sort((a, b) => (String(a.registeredAt || '') < String(b.registeredAt || '') ? 1 : -1))
  return listed
}

export async function cancelRegister(accessToken, registerId) {
  const id = typeof registerId === 'string' ? registerId.trim() : ''
  if (!id) throw apiError(404, 'NOT_FOUND', '未找到挂号单')
  const idempotencyKey = crypto.randomUUID()
  if (useMock()) {
    await delay(220)
    return mockCancelRegister(accessToken, id, idempotencyKey)
  }

  let response
  try {
    response = await fetch(apiUrl(`/registers/${encodeURIComponent(id)}/cancel`), requestInit({
      method: 'POST',
      headers: {
        Accept: JSON_ACCEPT,
        'Idempotency-Key': idempotencyKey,
        ...bearer(accessToken),
      },
    }))
  } catch {
    throw apiError(0, 'HTTP_ERROR', '无法连接挂号服务')
  }
  if (!response.ok) throw presentRegisterError(await readErrorResponse(response))
  return assertRegisterDto(await parseJson(response))
}

export async function checkInRegister(accessToken, registerId, options) {
  const id = typeof registerId === 'string' ? registerId.trim() : ''
  if (!id) throw apiError(404, 'NOT_FOUND', CHECKIN_MISSING)
  const simulateConflict = Boolean(options && options.simulateConflict)
  if (useMock()) {
    await delay(220)
    return mockCheckInRegister(accessToken, id, { simulateConflict })
  }

  let response
  try {
    response = await fetch(apiUrl(`/registers/${encodeURIComponent(id)}/check-in`), requestInit({
      method: 'POST',
      headers: {
        Accept: JSON_ACCEPT,
        ...bearer(accessToken),
      },
    }))
  } catch {
    throw apiError(0, 'HTTP_ERROR', '无法连接挂号服务')
  }
  if (!response.ok) throw presentCheckInError(await readErrorResponse(response))
  return assertRegisterDto(await parseJson(response))
}
