// 2026-10-12 P1-H05-REGISTER-UI
// POST /api/v1/registers | GET /api/v1/registers | POST /api/v1/registers/{registerId}/cancel
// 与建档相同：VITE_USE_MOCK=false 时走真实接口（.env.local）。科室字段是 departmentId。

import { apiUrl, useMock } from './config.js'
import { apiError, readErrorResponse } from './errors.js'
import { assertCreateRegisterRequest, assertRegisterDto, assertRegisterList } from './registerContract.js'
import { mockCancelRegister, mockCreateRegister, mockListRegisters } from './mockRegisters.js'
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
