// 2026-10-12 P1-H05-PATIENT-UI
// POST /api/v1/patients | GET /api/v1/patients | GET /api/v1/patients/{patientId}
// 与登录相同：VITE_USE_MOCK 决定走本地还是真实接口。真接口默认经 Vite 代理到 127.0.0.1:9999。

import { apiUrl, useMock } from './config.js'
import { apiError, readErrorResponse } from './errors.js'
import {
  assertCreatePatientRequest,
  assertPatientDto,
  assertPatientList,
  assertPatientSearchQuery,
  patientMatchesQuery,
} from './patientContract.js'
import { mockCreatePatient, mockGetPatient, mockSearchPatients, mockSearchRegistrationPatients } from './mockPatients.js'

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
  if (error.code === 'DUPLICATE_PATIENT' || error.code === 'DUPLICATE_ID_NUMBER') {
    return apiError(error.status || 409, error.code, '该身份证号已存在，不能重复建档', {
      details: error.details,
      requestId: error.requestId,
    })
  }
  if (error.code === 'DUPLICATE_CARD_NO') {
    return apiError(error.status || 409, 'DUPLICATE_CARD_NO', '卡号已存在，不能重复建档', {
      details: error.details,
      requestId: error.requestId,
    })
  }
  return error
}

function searchParams(query, { offset = 0, limit = 0 } = {}) {
  const params = new URLSearchParams()
  if (query.name) params.set('name', query.name)
  if (query.idNumber) params.set('idNumber', query.idNumber)
  if (query.patientNo) params.set('patientNo', query.patientNo)
  if (query.hospPkNo) params.set('hospPkNo', query.hospPkNo)
  if (offset > 0) params.set('offset', String(offset))
  if (limit > 0) params.set('limit', String(limit))
  return params
}

export async function createPatient(accessToken, input) {
  const request = assertCreatePatientRequest(input)
  const idempotencyKey = crypto.randomUUID()
  if (useMock()) {
    await delay(280)
    const patient = mockCreatePatient(accessToken, request, idempotencyKey)
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

  if (response.status !== 201) throw presentPatientError(await readErrorResponse(response, accessToken))
  const patient = assertPatientDto(await parseJson(response))
  remember([patient])
  return patient
}

export async function searchPatients(accessToken, input, { offset = 0, limit = 0 } = {}) {
  const query = assertPatientSearchQuery(input)
  if (!Number.isSafeInteger(offset) || offset < 0) throw apiError(400, 'VALIDATION_ERROR', '患者列表位置无效')
  if (!Number.isSafeInteger(limit) || limit < 0 || limit > 100) throw apiError(400, 'VALIDATION_ERROR', '患者列表大小无效')
  if (useMock()) {
    await delay(200)
    const result = mockSearchPatients(accessToken, query, { offset, limit })
    remember(result.items)
    return result
  }

  const params = searchParams(query, { offset, limit })
  let response
  try {
    const suffix = params.toString() ? `?${params.toString()}` : ''
    response = await fetch(`${apiUrl('/patients')}${suffix}`, requestInit({
      method: 'GET',
      headers: {
        Accept: JSON_ACCEPT,
        ...bearer(accessToken),
      },
    }))
  } catch {
    throw apiError(0, 'HTTP_ERROR', '无法连接患者服务')
  }
  if (!response.ok) throw presentPatientError(await readErrorResponse(response, accessToken))
  const result = assertPatientList(await parseJson(response))
  const items = result.items
  const filtered = items.filter((item) => patientMatchesQuery(item, query))
  remember(filtered)
  return { ...result, items: filtered }
}

function assertRegistrationPatientList(body) {
  if (!body || typeof body !== 'object' || !Array.isArray(body.items)) {
    throw apiError(502, 'HTTP_ERROR', '患者查询响应不符合契约')
  }
  const total = Number(body.total)
  const nextOffset = body.nextOffset == null ? null : Number(body.nextOffset)
  if (!Number.isSafeInteger(total) || total < 0 || (nextOffset !== null && (!Number.isSafeInteger(nextOffset) || nextOffset < 0))) {
    throw apiError(502, 'HTTP_ERROR', '患者查询响应不符合契约')
  }
  const items = body.items.map((item) => {
    if (!item || typeof item !== 'object' || typeof item.patientId !== 'string' || !item.patientId || typeof item.name !== 'string' ||
      typeof item.patientNo !== 'string' || typeof item.sex !== 'string' || typeof item.idType !== 'string' ||
      typeof item.idNumberMasked !== 'string' || typeof item.phoneMasked !== 'string' ||
      !['ACTIVE', 'INACTIVE'].includes(item.status) || typeof item.selectable !== 'boolean' || item.selectable !== (item.status === 'ACTIVE')) {
      throw apiError(502, 'HTTP_ERROR', '患者查询响应不符合契约')
    }
    return {
      patientId: item.patientId,
      patientNo: item.patientNo,
      name: item.name,
      sex: item.sex,
      idType: item.idType,
      idNumberMasked: item.idNumberMasked,
      phoneMasked: item.phoneMasked,
      status: item.status,
      selectable: item.selectable,
    }
  })
  return { items, total, nextOffset }
}

export async function searchRegistrationPatients(accessToken, keyword = '', { offset = 0, limit = 50 } = {}) {
  const normalized = typeof keyword === 'string' ? keyword.trim() : ''
  if (normalized.length > 64 || !Number.isSafeInteger(offset) || offset < 0 || !Number.isSafeInteger(limit) || limit < 1 || limit > 50) {
    throw apiError(400, 'VALIDATION_ERROR', '患者查询条件无效')
  }
  if (useMock()) {
    await delay(180)
    const result = await mockSearchRegistrationPatients(accessToken, normalized, { offset, limit })
    const items = result.items.map((item) => {
      const phone = item.phone || ''
      const idNumber = item.idNumber || ''
      return {
        patientId: item.patientId,
        patientNo: item.patientNo || '',
        name: item.name,
        sex: item.sex || 'U',
        idType: item.idType || '',
        idNumberMasked: idNumber.length > 4 ? `${idNumber.slice(0, 2)}${'*'.repeat(idNumber.length - 4)}${idNumber.slice(-2)}` : '*'.repeat(idNumber.length),
        phoneMasked: phone.length > 7 ? `${phone.slice(0, 3)}${'*'.repeat(phone.length - 7)}${phone.slice(-4)}` : '*'.repeat(phone.length),
        status: item.status || 'ACTIVE',
        selectable: item.status !== 'INACTIVE',
      }
    })
    return { items, total: result.total, nextOffset: result.nextOffset }
  }

  let response
  try {
    response = await fetch(apiUrl('/patients/registration-search'), requestInit({
      method: 'POST',
      headers: {
        Accept: JSON_ACCEPT,
        'Content-Type': 'application/json; charset=utf-8',
        ...bearer(accessToken),
      },
      body: JSON.stringify({ keyword: normalized, offset, limit }),
    }))
  } catch {
    throw apiError(0, 'HTTP_ERROR', '无法连接患者服务')
  }
  if (!response.ok) throw presentPatientError(await readErrorResponse(response, accessToken))
  return assertRegistrationPatientList(await parseJson(response))
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
  if (!response.ok) throw presentPatientError(await readErrorResponse(response, accessToken))
  const patient = assertPatientDto(await parseJson(response))
  remember([patient])
  return patient
}
