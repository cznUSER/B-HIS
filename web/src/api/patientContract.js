// 2026-10-12 P1-H05-PATIENT-UI 建档请求与患者查询。
// JSON 键对齐 OpenAPI 0.3.5：name、sex、birthday、idNumber、phone、address、fareIdentity、cardNo。
// 业务必填：姓名、性别、生日、费别。地址选填，有内容才提交。

import { apiError } from './errors.js'

const SEX = new Set(['M', 'F', 'U'])
const CARD_STATUS = new Set(['NOT_SENT', 'NORMAL', 'LOST', 'CANCELLED'])
const PATIENT_STATUS = new Set(['ACTIVE', 'INACTIVE'])
const FIRST_INVALID_DAY = '1900-01-01'

// 一期手机号默认选填。改为 true 后保存前会拦截空号，不另做设置页。
export const PHONE_REQUIRED = false

export const SEX_OPTIONS = [
  { value: 'M', label: '男' },
  { value: 'F', label: '女' },
  { value: 'U', label: '未知' },
]

export const FARE_OPTIONS = [
  { value: 'SELF', label: '自费' },
  { value: 'OTHER', label: '其他（占位）' },
]

export function todayInShanghai(now = new Date()) {
  const text = new Intl.DateTimeFormat('en-CA', {
    timeZone: 'Asia/Shanghai',
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
  }).format(now)
  return text
}

function asText(value) {
  return typeof value === 'string' ? value.trim() : ''
}

function isCalendarDay(iso) {
  if (!/^\d{4}-\d{2}-\d{2}$/.test(iso)) return false
  const [year, month, day] = iso.split('-').map((part) => Number(part))
  const date = new Date(Date.UTC(year, month - 1, day))
  return date.getUTCFullYear() === year && date.getUTCMonth() === month - 1 && date.getUTCDate() === day
}

export function sexLabel(sex) {
  const found = SEX_OPTIONS.find((item) => item.value === sex)
  return found ? found.label : sex || '—'
}

export function fareLabel(code) {
  const found = FARE_OPTIONS.find((item) => item.value === code)
  return found ? found.label : code || '—'
}

export function assertCreatePatientRequest(input) {
  const src = input && typeof input === 'object' ? input : {}
  const name = asText(src.name)
  if (!name) throw apiError(400, 'VALIDATION_ERROR', '请填写姓名')

  const sex = asText(src.sex)
  if (!SEX.has(sex)) throw apiError(400, 'VALIDATION_ERROR', '请选择性别')

  const birthday = asText(src.birthday)
  if (!isCalendarDay(birthday) || birthday <= FIRST_INVALID_DAY || birthday > todayInShanghai()) {
    throw apiError(400, 'VALIDATION_ERROR', '出生日期不合法')
  }

  const fareIdentity = asText(src.fareIdentity)
  if (!fareIdentity) throw apiError(400, 'VALIDATION_ERROR', '请选择费别')

  const idNumber = asText(src.idNumber).toUpperCase()
  if (idNumber && !/^(\d{15}|\d{17}[\dX])$/.test(idNumber)) {
    throw apiError(400, 'VALIDATION_ERROR', '身份证号格式不正确')
  }

  const phone = asText(src.phone)
  if (PHONE_REQUIRED && !phone) throw apiError(400, 'VALIDATION_ERROR', '请填写手机号')
  if (phone && !/^[0-9+\-()\s]{5,20}$/.test(phone)) {
    throw apiError(400, 'VALIDATION_ERROR', '手机号格式不正确')
  }

  const address = asText(src.address)
  if ([...address].length > 256) throw apiError(400, 'VALIDATION_ERROR', '地址请少于 256 字')

  const body = { name, sex, birthday, fareIdentity }
  if (idNumber) {
    body.idType = 'ID_CARD'
    body.idNumber = idNumber
  }
  if (phone) body.phone = phone
  if (address) body.address = address
  const cardNo = asText(src.cardNo)
  if (cardNo) body.cardNo = cardNo
  return body
}

export function assertPatientSearchQuery(input) {
  const src = input && typeof input === 'object' ? input : {}
  const name = asText(src.name)
  const idNumber = asText(src.idNumber).toUpperCase()
  const patientNo = asText(src.patientNo)
  if (!name && !idNumber && !patientNo) {
    throw apiError(400, 'VALIDATION_ERROR', '请填写姓名、身份证号或患者号')
  }
  return { name, idNumber, patientNo }
}

export function patientMatchesQuery(patient, query) {
  if (query.name && !String(patient.name || '').toLowerCase().includes(query.name.toLowerCase())) return false
  if (query.idNumber && String(patient.idNumber || '') !== query.idNumber) return false
  if (query.patientNo && String(patient.patientNo || '') !== query.patientNo) return false
  return true
}

function optionalString(body, key, target) {
  if (body[key] == null || body[key] === '') return
  if (typeof body[key] !== 'string') {
    throw apiError(502, 'HTTP_ERROR', '患者信息不符合契约')
  }
  target[key] = body[key]
}

export function assertPatientDto(body) {
  if (!body || typeof body !== 'object') {
    throw apiError(502, 'HTTP_ERROR', '患者信息不符合契约')
  }
  const patientId = asText(body.patientId)
  const name = asText(body.name)
  if (!patientId || !name) {
    throw apiError(502, 'HTTP_ERROR', '患者信息不符合契约')
  }
  const dto = { patientId, name }
  for (const key of ['hospPkNo', 'patientNo', 'cardNo', 'sex', 'birthday', 'idNumber', 'phone', 'address', 'fareIdentity', 'createdAt']) {
    optionalString(body, key, dto)
  }
  if (body.cardStatus != null && body.cardStatus !== '') {
    if (!CARD_STATUS.has(body.cardStatus)) throw apiError(502, 'HTTP_ERROR', '患者信息不符合契约')
    dto.cardStatus = body.cardStatus
  }
  if (body.status != null && body.status !== '') {
    if (!PATIENT_STATUS.has(body.status)) throw apiError(502, 'HTTP_ERROR', '患者信息不符合契约')
    dto.status = body.status
  }
  return dto
}

export function assertPatientList(body) {
  if (!body || typeof body !== 'object' || !Array.isArray(body.items)) {
    throw apiError(502, 'HTTP_ERROR', '患者查询响应不符合契约')
  }
  return { items: body.items.map((item) => assertPatientDto(item)) }
}
