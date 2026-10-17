import { apiUrl, useMock } from '../api/config.js'
import { apiError, readErrorResponse } from '../api/errors.js'

const AUDIT_SOURCE_FIELDS = [
  'moduleId', 'moduleName', 'logType', 'occurredAt', 'actorUserId', 'actorUsername',
  'departmentId', 'description', 'clientIp', 'clientMac', 'buildVersion', 'requestId', 'outcome', 'errorCode',
]

const AUDIT_FIELD_MAP = {
  moduleId: 'moduleId',
  module: 'moduleName',
  type: 'logType',
  time: 'occurredAt',
  userId: 'actorUserId',
  user: 'actorUsername',
  department: 'departmentId',
  description: 'description',
  ip: 'clientIp',
  mac: 'clientMac',
  version: 'buildVersion',
  requestId: 'requestId',
  outcome: 'outcome',
  errorCode: 'errorCode',
}

const CSV_COLUMNS = [
  ['module', '模块'],
  ['type', '类型'],
  ['time', '时间'],
  ['user', '用户'],
  ['department', '科室'],
  ['description', '说明'],
  ['ip', 'IP'],
  ['mac', 'MAC'],
  ['version', '版本'],
  ['outcome', '结果'],
  ['requestId', '请求编号'],
  ['errorCode', '错误代码'],
]

const MOCK_LOGS = [
  { moduleId: 'module-master', moduleName: '系统设置', logType: '用户重置密码', occurredAt: '2026-10-14T02:42:10+08:00', actorUserId: 'mock-operator', actorUsername: '系统管理员', departmentId: '系统管理', description: '重置用户 U-1002 密码', clientIp: '127.0.0.1', clientMac: '', buildVersion: '0.1.0', outcome: 'SUCCESS', requestId: 'req-mock-204', errorCode: '' },
  { moduleId: 'module-master', moduleName: '系统设置', logType: '用户资料修改', occurredAt: '2026-10-14T02:37:54+08:00', actorUserId: 'mock-operator', actorUsername: '系统管理员', departmentId: '系统管理', description: '修改用户 U-1002', clientIp: '127.0.0.1', clientMac: '', buildVersion: '0.1.0', outcome: 'SUCCESS', requestId: 'req-mock-203', errorCode: '' },
  { moduleId: 'module-outpatient', moduleName: '门诊业务', logType: '患者建档', occurredAt: '2026-10-14T02:31:22+08:00', actorUserId: 'mock-clerk', actorUsername: '门诊挂号员', departmentId: '门诊部', description: '新建患者 P-20261014-001', clientIp: '127.0.0.1', clientMac: '', buildVersion: '0.1.0', outcome: 'SUCCESS', requestId: 'req-mock-202', errorCode: '' },
  { moduleId: 'module-master', moduleName: '系统设置', logType: '角色修改', occurredAt: '2026-10-13T17:16:03+08:00', actorUserId: 'mock-operator', actorUsername: '系统管理员', departmentId: '系统管理', description: '修改角色 CASHIER', clientIp: '127.0.0.1', clientMac: '', buildVersion: '0.1.0', outcome: 'FAILURE', requestId: 'req-mock-201', errorCode: 'FORBIDDEN' },
  { moduleId: 'module-outpatient', moduleName: '门诊业务', logType: '就诊创建', occurredAt: '2026-10-13T16:48:31+08:00', actorUserId: 'mock-doctor', actorUsername: '门诊医生', departmentId: '内科', description: '创建就诊 E-20261013-018', clientIp: '127.0.0.1', clientMac: '', buildVersion: '0.1.0', outcome: 'SUCCESS', requestId: 'req-mock-200', errorCode: '' },
]

function auditError(message = '用户操作日志不符合契约') {
  return apiError(502, 'HTTP_ERROR', message)
}

export function normalizeUserOperationsResponse(body) {
  if (!body || typeof body !== 'object' || Array.isArray(body) || !Array.isArray(body.items)) {
    throw auditError()
  }
  if (!('nextCursor' in body) || (body.nextCursor !== null && typeof body.nextCursor !== 'string')) {
    throw auditError()
  }
  const items = body.items.map((row) => {
    if (!row || typeof row !== 'object' || Array.isArray(row)) throw auditError()
    for (const field of AUDIT_SOURCE_FIELDS) {
      if (typeof row[field] !== 'string') throw auditError()
    }
    const normalized = Object.fromEntries(Object.entries(AUDIT_FIELD_MAP).map(([target, source]) => [target, row[source]]))
    return normalized
  })
  return { items, nextCursor: body.nextCursor || '' }
}

function normalizedLimit(value, maximum = 200) {
  const number = Number(value)
  if (!Number.isFinite(number)) return 50
  return Math.min(maximum, Math.max(1, Math.trunc(number)))
}

function buildQuery(params = {}) {
  const query = new URLSearchParams()
  for (const field of ['moduleId', 'userId', 'begin', 'end', 'cursor']) {
    const value = typeof params[field] === 'string' ? params[field].trim() : ''
    if (value) query.set(field, value)
  }
  query.set('limit', String(normalizedLimit(params.limit)))
  return query.toString()
}

function mockPage(params) {
  const moduleId = String(params.moduleId || '').trim()
  const userId = String(params.userId || '').trim()
  const begin = String(params.begin || '').trim()
  const end = String(params.end || '').trim()
  const limit = normalizedLimit(params.limit)
  const filtered = MOCK_LOGS.filter((row) => {
    const day = row.occurredAt.slice(0, 10)
    return (!moduleId || row.moduleId === moduleId)
      && (!userId || row.actorUserId === userId)
      && (!begin || day >= begin)
      && (!end || day <= end)
  })
  const parsedCursor = /^mock:(\d+)$/.exec(String(params.cursor || ''))
  const offset = parsedCursor ? Number(parsedCursor[1]) : 0
  const items = filtered.slice(offset, offset + limit)
  const nextOffset = offset + items.length
  return normalizeUserOperationsResponse({
    items,
    nextCursor: nextOffset < filtered.length ? `mock:${nextOffset}` : '',
  })
}

export async function fetchUserOperationLogs(accessToken, params = {}) {
  if (useMock()) return mockPage(params)

  const query = buildQuery(params)
  let response
  try {
    response = await fetch(apiUrl(`/audit/user-operations?${query}`), {
      method: 'GET',
      credentials: 'omit',
      cache: 'no-store',
      headers: {
        Accept: 'application/json',
        ...(accessToken ? { Authorization: `Bearer ${accessToken}` } : {}),
      },
    })
  } catch {
    throw apiError(0, 'HTTP_ERROR', '无法连接用户操作日志服务')
  }
  if (response.status !== 200) throw await readErrorResponse(response, accessToken)
  let body
  try {
    body = await response.json()
  } catch {
    throw auditError('用户操作日志响应无法解析')
  }
  return normalizeUserOperationsResponse(body)
}

function csvCell(value) {
  let text = String(value ?? '')
  if (/^[\u0000-\u0020]*[=+@-]/.test(text)) text = `'${text}`
  return `"${text.replaceAll('"', '""')}"`
}

export function buildUserOperationsCsv(rows) {
  const lines = [CSV_COLUMNS.map(([, label]) => csvCell(label)).join(',')]
  for (const row of rows || []) {
    lines.push(CSV_COLUMNS.map(([field]) => csvCell(row?.[field])).join(','))
  }
  return `\uFEFF${lines.join('\r\n')}\r\n`
}
