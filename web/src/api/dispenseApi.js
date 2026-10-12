// 2026-10-12 P1-H05-DISPENSE-UI
// 与收费、开单共用 VITE_USE_MOCK。默认只走 Mock。
// 路径名对齐 OpenAPI 0.3.9-draft Dispenses，真请求等 H04，本文件不调用 fetch。

import { useMock } from './config.js'
import { apiError } from './errors.js'
import {
  mockConfirmDispense,
  mockGetDispense,
  mockListDispenses,
  mockListTodayPending,
  mockReturnDispense,
  mockSearchPending,
  resetMockDispenses,
} from './mockDispenses.js'

// 主路径 POST/GET /dispenses。POST /orders/{orderId}/dispense 是 deprecated 别名。
export const DISPENSE_ROUTES = {
  collection: '/dispenses',
  item: (dispenseId) => `/dispenses/${dispenseId}`,
  return: (dispenseId) => `/dispenses/${dispenseId}/return`,
  orderAlias: (orderId) => `/orders/${orderId}/dispense`,
}

function delay(ms) {
  return new Promise((resolve) => {
    setTimeout(resolve, ms)
  })
}

function assertMock() {
  if (!useMock()) throw apiError(501, 'NOT_IMPLEMENTED', '发药接口尚未接入')
}

export async function searchPending(accessToken, query) {
  assertMock()
  await delay(120)
  return mockSearchPending(accessToken, query)
}

export async function listTodayPending(accessToken) {
  assertMock()
  await delay(120)
  return mockListTodayPending(accessToken)
}

export async function confirmDispense(accessToken, input) {
  assertMock()
  await delay(160)
  return mockConfirmDispense(accessToken, input)
}

export async function listDispenses(accessToken, input) {
  assertMock()
  await delay(80)
  return mockListDispenses(accessToken, input)
}

export async function fetchDispense(accessToken, dispenseId) {
  assertMock()
  await delay(80)
  return mockGetDispense(accessToken, dispenseId)
}

export async function returnDispense(accessToken, input) {
  assertMock()
  await delay(160)
  return mockReturnDispense(accessToken, input)
}

export async function resetDispenseDemo(accessToken) {
  assertMock()
  resetMockDispenses(accessToken)
}
