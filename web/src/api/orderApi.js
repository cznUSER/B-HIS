// 2026-10-12 P1-H05-ORDER-UI
// 2026-10-12 P1-H05-ORDER-UI 真接口：VITE_USE_MOCK=false 时走 /api/v1/orders。
// POST 创建 SAVE；PATCH 仅改暂存；POST /commit；POST /abate；GET 按 registerId 等查询。
// 挂号只读（GET /registers/{id}、当日列表），不调用接诊。

import { apiUrl, useMock } from './config.js'
import { apiError, readErrorResponse } from './errors.js'
import {
  assertCreateOrderRequest,
  assertOrderDto,
  assertOrderList,
  assertPatchOrderRequest,
} from './orderContract.js'
import { fetchPatient } from './patientApi.js'
import { todayInShanghai } from './patientContract.js'
import { assertRegisterDto, assertRegisterList } from './registerContract.js'
import { deptName, doctorName } from '../register/master.js'
import {
  mockAbateOrder,
  mockCommitOrder,
  mockCreateOrder,
  mockGetOrderRegister,
  mockListOrderRegisters,
  mockListOrders,
  mockPatchOrder,
} from './mockOrders.js'

const JSON_ACCEPT = 'application/json'

export const ORDER_ROUTES = {
  collection: '/orders',
  item: (orderId) => `/orders/${orderId}`,
  commit: (orderId) => `/orders/${orderId}/commit`,
  abate: (orderId) => `/orders/${orderId}/abate`,
}

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

export function presentOrderError(error) {
  if (error && error.code === 'ENCOUNTER_REQUIRED') {
    return apiError(error.status || 409, error.code, '请先挂号', {
      details: error.details,
      requestId: error.requestId,
    })
  }
  return error
}

async function callJson(path, init, offlineMessage) {
  let response
  try {
    response = await fetch(apiUrl(path), requestInit(init))
  } catch {
    throw apiError(0, 'HTTP_ERROR', offlineMessage)
  }
  return response
}

function presentRegister(dto, patient) {
  return {
    registerId: dto.registerId,
    registerNo: dto.registerNo || '',
    patientId: dto.patientId,
    patientName: patient && patient.name ? patient.name : '',
    patientNo: patient && patient.patientNo ? patient.patientNo : '',
    departmentId: dto.departmentId || '',
    departmentName: deptName(dto.departmentId),
    doctorId: dto.doctorId || '',
    doctorName: doctorName(dto.doctorId),
    visitDate: dto.visitDate || '',
    status: dto.status,
  }
}

async function withPatient(accessToken, dto) {
  let patient = null
  if (dto.patientId) {
    try {
      patient = await fetchPatient(accessToken, dto.patientId)
    } catch {
      patient = null
    }
  }
  return presentRegister(dto, patient)
}

function orderSearchParams(input) {
  const source = typeof input === 'string' ? { registerId: input } : (input && typeof input === 'object' ? input : {})
  const registerId = typeof source.registerId === 'string' ? source.registerId.trim() : ''
  const encounterId = typeof source.encounterId === 'string' ? source.encounterId.trim() : ''
  if (registerId && encounterId && registerId !== encounterId) {
    throw apiError(400, 'VALIDATION_ERROR', 'registerId 与 encounterId 不一致')
  }
  const params = new URLSearchParams()
  const primary = registerId || encounterId
  if (primary) params.set('registerId', primary)
  for (const key of ['patientId', 'status', 'orderNo']) {
    const value = typeof source[key] === 'string' ? source[key].trim() : ''
    if (value) params.set(key, value)
  }
  return params
}

function patchPayload(input) {
  const request = assertPatchOrderRequest(input)
  delete request.registerId
  return request
}

export async function listOrderRegisters(accessToken) {
  if (useMock()) {
    await delay(120)
    return mockListOrderRegisters(accessToken)
  }
  const params = new URLSearchParams({ visitDate: todayInShanghai() })
  const response = await callJson(`/registers?${params.toString()}`, {
    method: 'GET',
    headers: {
      Accept: JSON_ACCEPT,
      ...bearer(accessToken),
    },
  }, '无法连接挂号服务')
  if (!response.ok) throw presentOrderError(await readErrorResponse(response))
  const listed = assertRegisterList(await parseJson(response))
  const items = []
  for (const item of listed.items) items.push(await withPatient(accessToken, item))
  return { items }
}

export async function fetchOrderRegister(accessToken, registerId) {
  const id = typeof registerId === 'string' ? registerId.trim() : ''
  if (!id) throw apiError(404, 'NOT_FOUND', '未找到有效挂号')
  if (useMock()) {
    await delay(120)
    return mockGetOrderRegister(accessToken, id)
  }
  const response = await callJson(`/registers/${encodeURIComponent(id)}`, {
    method: 'GET',
    headers: {
      Accept: JSON_ACCEPT,
      ...bearer(accessToken),
    },
  }, '无法连接挂号服务')
  if (!response.ok) throw presentOrderError(await readErrorResponse(response))
  return withPatient(accessToken, assertRegisterDto(await parseJson(response)))
}

export async function listOrders(accessToken, input) {
  const params = orderSearchParams(input)
  if (useMock()) {
    await delay(80)
    return mockListOrders(accessToken, {
      registerId: params.get('registerId') || '',
      patientId: params.get('patientId') || '',
      status: params.get('status') || '',
      orderNo: params.get('orderNo') || '',
    })
  }
  const query = params.toString()
  const response = await callJson(query ? `${ORDER_ROUTES.collection}?${query}` : ORDER_ROUTES.collection, {
    method: 'GET',
    headers: {
      Accept: JSON_ACCEPT,
      ...bearer(accessToken),
    },
  }, '无法连接开单服务')
  if (!response.ok) throw presentOrderError(await readErrorResponse(response))
  return assertOrderList(await parseJson(response))
}

export async function saveOrder(accessToken, input) {
  const orderId = input && typeof input.orderId === 'string' ? input.orderId.trim() : ''
  if (useMock()) {
    await delay(180)
    if (orderId) return mockPatchOrder(accessToken, orderId, input)
    return mockCreateOrder(accessToken, input, crypto.randomUUID())
  }
  if (orderId) {
    const response = await callJson(ORDER_ROUTES.item(orderId), {
      method: 'PATCH',
      headers: {
        Accept: JSON_ACCEPT,
        'Content-Type': 'application/json; charset=utf-8',
        ...bearer(accessToken),
      },
      body: JSON.stringify(patchPayload(input)),
    }, '无法连接开单服务')
    if (!response.ok) throw presentOrderError(await readErrorResponse(response))
    return assertOrderDto(await parseJson(response))
  }
  const response = await callJson(ORDER_ROUTES.collection, {
    method: 'POST',
    headers: {
      Accept: JSON_ACCEPT,
      'Content-Type': 'application/json; charset=utf-8',
      'Idempotency-Key': crypto.randomUUID(),
      ...bearer(accessToken),
    },
    body: JSON.stringify(assertCreateOrderRequest(input)),
  }, '无法连接开单服务')
  if (response.status !== 201) throw presentOrderError(await readErrorResponse(response))
  return assertOrderDto(await parseJson(response))
}

async function postOrderAction(accessToken, path) {
  const response = await callJson(path, {
    method: 'POST',
    headers: {
      Accept: JSON_ACCEPT,
      'Idempotency-Key': crypto.randomUUID(),
      ...bearer(accessToken),
    },
  }, '无法连接开单服务')
  if (!response.ok) throw presentOrderError(await readErrorResponse(response))
  return assertOrderDto(await parseJson(response))
}

export async function commitOrder(accessToken, orderId) {
  const id = typeof orderId === 'string' ? orderId.trim() : ''
  if (!id) throw apiError(404, 'NOT_FOUND', '未找到处方')
  if (useMock()) {
    await delay(140)
    return mockCommitOrder(accessToken, id)
  }
  return postOrderAction(accessToken, ORDER_ROUTES.commit(id))
}

export async function abateOrder(accessToken, orderId) {
  const id = typeof orderId === 'string' ? orderId.trim() : ''
  if (!id) throw apiError(404, 'NOT_FOUND', '未找到处方')
  if (useMock()) {
    await delay(140)
    return mockAbateOrder(accessToken, id)
  }
  return postOrderAction(accessToken, ORDER_ROUTES.abate(id))
}
