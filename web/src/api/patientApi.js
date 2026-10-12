// 2026-10-12 P1-H05-PATIENT-UI
// POST /api/v1/patients | GET /api/v1/patients | GET /api/v1/patients/{patientId}
// 与登录相同：VITE_USE_MOCK 决定走本地还是真实接口。地址不写入请求体。

import { apiUrl, useMock } from './config.js'
import { apiError, readErrorResponse } from './errors.js'
import {
  assertCreatePatientRequest,
  assertPatientDto,
  assertPatientList,
  assertPatientSearchQuery,
  patientMatchesQuery,
} from './patientContract.js'
import {
  mockCreatePatient,
  mockGetPatient,
  mockLocalAddress,
  mockRememberAddress,
  mockSearchPatients,
} from './mockPatients.js'

const JSON_ACCEPT = 'application/json'
const seen = new Map()

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

function remember(patients) {
  for (const patient of patients) seen.set(patient.patientId, patient)
}

export function resetPatientClientCache() {
  seen.clear()
}

export function presentPatientError(error) {
  if (error.code === 'DUPLICATE_PATIENT') {
    return apiError(error.status || 409, 'DUPLICATE_PATIENT', '该身份证号已存在，不能重复建档', {
      details: error.details,
      requestId: error.requestId,
    })
  }
  if (error.code === 'DUPLICATE_CARD_NO') {
    return apiError(error.status || 409, 'DUPLICATE_CARD_NO', '证件或卡号已存在，无法建档', {
      details: error.details,
      requestId: error.requestId,
    })
  }
  return error
}

export function localPatientAddress(patientId) {
  if (!useMock()) return ''
  return mockLocalAddress(patientId)
}

function searchParams(query) {
  const params = new URLSearchParams()
  if (query.name) params.set('name', query.name)
  if (query.idNumber) params.set('idNumber', query.idNumber)
  return params
}

export async function createPatient(accessToken, input) {
  const request = assertCreatePatientRequest(input)
  const address = input && typeof input.address === 'string' ? input.address.trim() : ''
  const idempotencyKey = crypto.randomUUID()
  if (useMock()) {
    await delay(280)
    const patient = mockCreatePatient(accessToken, request, idempotencyKey)
    if (address) mockRememberAddress(patient.patientId, address)
    remember([patient])
    return patient
  }

  let response
  try {
    response = await fetch(apiUrl('/patients'), requestInit({
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
    throw apiError(0, 'HTTP_ERROR', '无法连接患者服务')
  }

  if (response.status !== 201) throw presentPatientError(await readErrorResponse(response))
  const patient = assertPatientDto(await parseJson(response))
  remember([patient])
  return patient
}

export async function searchPatients(accessToken, input) {
  const query = assertPatientSearchQuery(input)
  if (useMock()) {
    await delay(200)
    const result = mockSearchPatients(accessToken, query)
    remember(result.items)
    return result
  }

  const params = searchParams(query)
  let items
  if ([...params.keys()].length === 0) {
    items = [...seen.values()]
  } else {
    let response
    try {
      response = await fetch(`${apiUrl('/patients')}?${params.toString()}`, requestInit({
        method: 'GET',
        headers: {
          Accept: JSON_ACCEPT,
          ...bearer(accessToken),
        },
      }))
    } catch {
      throw apiError(0, 'HTTP_ERROR', '无法连接患者服务')
    }
    if (!response.ok) throw presentPatientError(await readErrorResponse(response))
    items = assertPatientList(await parseJson(response)).items
  }

  const filtered = items.filter((item) => patientMatchesQuery(item, query))
  remember(filtered)
  return { items: filtered }
}

export async function fetchPatient(accessToken, patientId) {
  const id = typeof patientId === 'string' ? patientId.trim() : ''
  if (!id) throw apiError(400, 'VALIDATION_ERROR', '未找到患者')
  if (useMock()) {
    await delay(160)
    const patient = mockGetPatient(accessToken, id)
    remember([patient])
    return patient
  }

  let response
  try {
    response = await fetch(apiUrl(`/patients/${encodeURIComponent(id)}`), requestInit({
      method: 'GET',
      headers: {
        Accept: JSON_ACCEPT,
        ...bearer(accessToken),
      },
    }))
  } catch {
    throw apiError(0, 'HTTP_ERROR', '无法连接患者服务')
  }
  if (!response.ok) throw presentPatientError(await readErrorResponse(response))
  const patient = assertPatientDto(await parseJson(response))
  remember([patient])
  return patient
}
