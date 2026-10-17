import { apiUrl, useMock } from './config.js'
import { apiError, readErrorResponse } from './errors.js'

export const DASHBOARD_SUMMARY_ROUTE = '/workbench/summary'
export const DASHBOARD_PERIODS = ['TODAY', 'YESTERDAY', 'WEEK', 'MONTH']

// A period switch or manual refresh can leave an older request in flight. The
// view uses this gate to ignore responses that no longer represent the latest
// selected period and permission snapshot.
export function createDashboardRequestGate() {
  let sequence = 0
  return {
    begin() {
      sequence += 1
      return sequence
    },
    isCurrent(requestId) {
      return requestId === sequence
    },
  }
}

function nonNegativeInteger(value, field) {
  if (!Number.isSafeInteger(value) || value < 0) {
    throw apiError(502, 'HTTP_ERROR', `工作台汇总的${field}无效`)
  }
  return value
}

export function assertDashboardSummary(raw, expectedPeriod) {
  if (!raw || typeof raw !== 'object' || raw.period !== expectedPeriod || !DASHBOARD_PERIODS.includes(raw.period)) {
    throw apiError(502, 'HTTP_ERROR', '工作台汇总响应无法读取')
  }
  for (const field of ['rangeStart', 'rangeEnd']) {
    if (typeof raw[field] !== 'string' || Number.isNaN(Date.parse(raw[field]))) {
      throw apiError(502, 'HTTP_ERROR', '工作台汇总响应无法读取')
    }
  }
  if (Date.parse(raw.rangeStart) >= Date.parse(raw.rangeEnd)) {
    throw apiError(502, 'HTTP_ERROR', '工作台汇总响应无法读取')
  }
  const revenueCents = nonNegativeInteger(raw.revenueCents, '收费金额')
  const refundCents = nonNegativeInteger(raw.refundCents, '退费金额')
  const netRevenueCents = Number(raw.netRevenueCents)
  if (!Number.isSafeInteger(netRevenueCents) || netRevenueCents !== revenueCents - refundCents) {
    throw apiError(502, 'HTTP_ERROR', '工作台汇总响应无法读取')
  }
  if (!raw.available || typeof raw.available !== 'object' || ['charges', 'encounters', 'dispenses'].some((key) => typeof raw.available[key] !== 'boolean')) {
    throw apiError(502, 'HTTP_ERROR', '工作台汇总响应无法读取')
  }
  return {
    period: raw.period,
    rangeStart: raw.rangeStart,
    rangeEnd: raw.rangeEnd,
    revenueCents,
    refundCents,
    netRevenueCents,
    chargeCount: nonNegativeInteger(raw.chargeCount, '收费笔数'),
    refundCount: nonNegativeInteger(raw.refundCount, '退费笔数'),
    encounterCount: nonNegativeInteger(raw.encounterCount, '门诊人次'),
    averageTicketCents: nonNegativeInteger(raw.averageTicketCents, '平均客单价'),
    pendingChargeCount: nonNegativeInteger(raw.pendingChargeCount, '待收费数'),
    pendingDispenseCount: nonNegativeInteger(raw.pendingDispenseCount, '待发药数'),
    available: {
      charges: raw.available.charges,
      encounters: raw.available.encounters,
      dispenses: raw.available.dispenses,
    },
  }
}

export async function fetchDashboardSummary(accessToken, period) {
  if (!DASHBOARD_PERIODS.includes(period)) throw apiError(400, 'VALIDATION_ERROR', '时间范围无效')
  if (useMock()) return { source: 'mock' }
  let response
  try {
    response = await fetch(`${apiUrl(DASHBOARD_SUMMARY_ROUTE)}?period=${encodeURIComponent(period)}`, {
      method: 'GET',
      credentials: 'omit',
      cache: 'no-store',
      headers: {
        Accept: 'application/json',
        ...(accessToken ? { Authorization: `Bearer ${accessToken}` } : {}),
      },
    })
  } catch {
    throw apiError(0, 'HTTP_ERROR', '无法连接工作台汇总服务')
  }
  if (!response.ok) throw await readErrorResponse(response, accessToken)
  let body
  try {
    body = await response.json()
  } catch {
    throw apiError(502, 'HTTP_ERROR', '工作台汇总响应无法读取')
  }
  return assertDashboardSummary(body, period)
}
