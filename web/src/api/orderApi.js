// 2026-10-12 P1-H05-ORDER-UI
// 形状对齐 OpenAPI 0.3.7，但先不发 HTTP。
// 待 H04 ORDER-API 合并且 H01 标明可联调后，再接 POST/PATCH /orders 与 /commit、/abate。

import { useMock } from './config.js'
import { apiError } from './errors.js'
import {
  mockAbateOrder,
  mockCommitOrder,
  mockCreateOrder,
  mockGetOrderRegister,
  mockListOrderRegisters,
  mockListOrders,
  mockPatchOrder,
} from './mockOrders.js'

export const ORDER_ROUTES = {
  collection: '/orders',
  item: (orderId) => `/orders/${orderId}`,
  commit: (orderId) => `/orders/${orderId}/commit`,
  abate: (orderId) => `/orders/${orderId}/abate`,
}

function delay(ms) {
  return new Promise((resolve) => {
    setTimeout(resolve, ms)
  })
}

function pending() {
  return apiError(501, 'NOT_IMPLEMENTED', '开单接口尚未联调')
}

function assertMock() {
  if (!useMock()) throw pending()
}

export async function listOrderRegisters(accessToken) {
  assertMock()
  await delay(120)
  return mockListOrderRegisters(accessToken)
}

export async function fetchOrderRegister(accessToken, registerId) {
  assertMock()
  await delay(120)
  return mockGetOrderRegister(accessToken, registerId)
}

export async function listOrders(accessToken, registerId) {
  assertMock()
  await delay(80)
  return mockListOrders(accessToken, { registerId })
}

export async function saveOrder(accessToken, input) {
  assertMock()
  await delay(180)
  const orderId = input && typeof input.orderId === 'string' ? input.orderId.trim() : ''
  if (orderId) return mockPatchOrder(accessToken, orderId, input)
  return mockCreateOrder(accessToken, input, crypto.randomUUID())
}

export async function commitOrder(accessToken, orderId) {
  assertMock()
  await delay(140)
  return mockCommitOrder(accessToken, orderId)
}

export async function abateOrder(accessToken, orderId) {
  assertMock()
  await delay(140)
  return mockAbateOrder(accessToken, orderId)
}
