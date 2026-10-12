// 2026-10-12 P1-H05-REGISTER-UI
// POST /api/v1/encounters | GET /api/v1/encounters | POST /api/v1/encounters/{encounterId}/cancel
// 与建档相同：VITE_USE_MOCK 决定走本地还是真实接口。真接口未开放时不阻断页面编译。

import { apiUrl, useMock } from './config.js'
import { apiError, readErrorResponse } from './errors.js'
import { assertCreateEncounterRequest, assertEncounterDto, assertEncounterList } from './encounterContract.js'
import { mockCancelEncounter, mockCreateEncounter, mockListEncounters } from './mockEncounters.js'

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

export function presentEncounterError(error) {
  if (error.code !== 'REGISTER_BACK_NOT_ALLOWED') return error
  const message = typeof error.message === 'string' ? error.message : ''
  if (message.includes('不能退号') || message.includes('已退')) return error
  return apiError(error.status || 409, error.code, '当前号已使用或就诊中，不能退号', {
    details: error.details,
    requestId: error.requestId,
  })
}

export async function createEncounter(accessToken, input) {
  const request = assertCreateEncounterRequest(input)
  const idempotencyKey = crypto.randomUUID()
  if (useMock()) {
    await delay(280)
    return mockCreateEncounter(accessToken, request, idempotencyKey)
  }

  let response
  try {
    response = await fetch(apiUrl('/encounters'), requestInit({
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
  if (response.status !== 201) throw presentEncounterError(await readErrorResponse(response))
  return assertEncounterDto(await parseJson(response))
}

export async function listEncounters(accessToken, input) {
  const patientId = input && typeof input.patientId === 'string' ? input.patientId.trim() : ''
  const visitDate = input && typeof input.visitDate === 'string' ? input.visitDate.trim() : ''
  if (useMock()) {
    await delay(180)
    return mockListEncounters(accessToken, { patientId, visitDate })
  }

  const params = new URLSearchParams()
  if (patientId) params.set('patientId', patientId)
  if (visitDate) params.set('visitDate', visitDate)
  let response
  try {
    response = await fetch(`${apiUrl('/encounters')}?${params.toString()}`, requestInit({
      method: 'GET',
      headers: {
        Accept: JSON_ACCEPT,
        ...bearer(accessToken),
      },
    }))
  } catch {
    throw apiError(0, 'HTTP_ERROR', '无法连接挂号服务')
  }
  if (!response.ok) throw presentEncounterError(await readErrorResponse(response))
  return assertEncounterList(await parseJson(response))
}

export async function cancelEncounter(accessToken, encounterId) {
  const id = typeof encounterId === 'string' ? encounterId.trim() : ''
  if (!id) throw apiError(404, 'NOT_FOUND', '未找到挂号单')
  const idempotencyKey = crypto.randomUUID()
  if (useMock()) {
    await delay(220)
    return mockCancelEncounter(accessToken, id, idempotencyKey)
  }

  let response
  try {
    response = await fetch(apiUrl(`/encounters/${encodeURIComponent(id)}/cancel`), requestInit({
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
  if (!response.ok) throw presentEncounterError(await readErrorResponse(response))
  return assertEncounterDto(await parseJson(response))
}
