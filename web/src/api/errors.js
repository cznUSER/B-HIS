// 2026-10-12 P1-H05 登录界面
// 对齐 OpenAPI 0.3.2 ErrorBody；同时容忍散文契约里的 data 字段和纯 HTTP 错误。

const MESSAGE_BY_CODE = {
  INVALID_CREDENTIALS: '用户名或密码错误',
  ACCOUNT_DISABLED: '账号已停用',
  AUTH_RATE_LIMITED: '登录尝试过于频繁，请稍后再试',
  UNAUTHORIZED: '未登录或登录已失效',
  TOKEN_EXPIRED: '登录已过期，请重新登录',
  TOKEN_INVALID: '登录凭证无效，请重新登录',
  FORBIDDEN: '没有权限执行该操作',
  VALIDATION_ERROR: '请输入用户名和密码',
  DUPLICATE_PATIENT: '该身份证号已存在，不能重复建档',
  DUPLICATE_ID_NUMBER: '该身份证号已存在，不能重复建档',
  DUPLICATE_CARD_NO: '卡号已存在，不能重复建档',
  NOT_FOUND: '未找到患者',
  PATIENT_NOT_FOUND: '患者不存在',
  DEPARTMENT_NOT_FOUND: '科室不存在',
  DOCTOR_NOT_FOUND: '医生不存在',
  REGISTER_NOT_CANCELLABLE: '当前号已使用或就诊中，不能退号',
  REGISTER_BACK_NOT_ALLOWED: '该号已退，不能再次退号',
  // 2026-10-12 P1-H05b-REGISTER-CHECKIN-UI 报到非法状态，对齐契约示例文案。
  ILLEGAL_STATE_TRANSITION: '当前状态不允许该操作',
}

const CODE_BY_STATUS = {
  400: 'VALIDATION_ERROR',
  401: 'UNAUTHORIZED',
  403: 'FORBIDDEN',
  429: 'AUTH_RATE_LIMITED',
}

export class ApiError extends Error {
  constructor({ status, code, message, details = null, requestId = '' }) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
    this.details = details
    this.requestId = requestId
  }
}

export function messageForCode(code) {
  return MESSAGE_BY_CODE[code] || '请求失败，请稍后重试'
}

export function apiError(status, code, message, extra = {}) {
  return new ApiError({
    status,
    code,
    message: message || messageForCode(code),
    details: extra.details ?? null,
    requestId: extra.requestId ?? '',
  })
}

export async function readErrorResponse(response) {
  const text = await response.text()
  let body = null
  if (text) {
    try {
      body = JSON.parse(text)
    } catch {
      body = null
    }
  }

  const envelope = body && typeof body === 'object' ? body : null
  const err = envelope && envelope.error && typeof envelope.error === 'object' ? envelope.error : null
  const code =
    (err && typeof err.code === 'string' && err.code) ||
    CODE_BY_STATUS[response.status] ||
    'HTTP_ERROR'
  const serverMessage = err && typeof err.message === 'string' ? err.message.trim() : ''
  const plainMessage =
    !err && envelope && typeof envelope.message === 'string' ? envelope.message.trim() : ''

  return new ApiError({
    status: response.status,
    code,
    message: serverMessage || plainMessage || messageForCode(code) || `请求失败（HTTP ${response.status}）`,
    details: err && 'details' in err ? err.details : null,
    requestId: envelope && typeof envelope.requestId === 'string' ? envelope.requestId : '',
  })
}
