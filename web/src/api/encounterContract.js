// 2026-10-12 P1-H05-REGISTER-UI 挂号请求对齐 OpenAPI Encounter。
// 提交字段：patientId、deptCode、doctorId、visitDate，号别与挂号费一并带上。

import { apiError } from './errors.js'
import { CLINIC_SORT, DEPARTMENTS, DOCTORS } from '../register/master.js'

const STATUSES = new Set(['NORMAL', 'USED', 'BACK', 'DIAGNOSING', 'DIAGNOSED', 'PAUSED'])

function asText(value) {
  return typeof value === 'string' ? value.trim() : ''
}

export function assertCreateEncounterRequest(input) {
  const src = input && typeof input === 'object' ? input : {}
  const patientId = asText(src.patientId)
  if (!patientId) throw apiError(400, 'VALIDATION_ERROR', '请先选择或建档患者')

  const deptCode = asText(src.deptCode)
  if (!DEPARTMENTS.some((item) => item.deptCode === deptCode)) {
    throw apiError(400, 'VALIDATION_ERROR', '请选择科室')
  }

  const doctorId = asText(src.doctorId)
  const doctor = DOCTORS.find((item) => item.doctorId === doctorId)
  if (!doctor || doctor.deptCode !== deptCode) {
    throw apiError(400, 'VALIDATION_ERROR', '请选择医生')
  }

  const visitDate = asText(src.visitDate)
  if (!/^\d{4}-\d{2}-\d{2}$/.test(visitDate)) {
    throw apiError(400, 'VALIDATION_ERROR', '就诊日期无效')
  }

  const clinicSort = asText(src.clinicSort) || CLINIC_SORT.clinicSort
  const fee = src.registerFee
  const registerFee = typeof fee === 'number' && Number.isFinite(fee) ? fee : 0
  return { patientId, deptCode, doctorId, clinicSort, visitDate, registerFee }
}

export function assertEncounterDto(body) {
  if (!body || typeof body !== 'object') {
    throw apiError(502, 'HTTP_ERROR', '挂号单不符合契约')
  }
  const encounterId = asText(body.encounterId)
  const patientId = asText(body.patientId)
  const status = asText(body.status)
  if (!encounterId || !patientId || !STATUSES.has(status)) {
    throw apiError(502, 'HTTP_ERROR', '挂号单不符合契约')
  }
  const dto = {
    encounterId,
    patientId,
    status,
    registerFee: typeof body.registerFee === 'number' && Number.isFinite(body.registerFee) ? body.registerFee : 0,
    registerCharged: body.registerCharged === true,
  }
  for (const key of ['encounterNo', 'deptCode', 'doctorId', 'clinicSort', 'visitDate', 'registeredAt', 'backOperatorId', 'backOperatedAt']) {
    if (typeof body[key] === 'string') dto[key] = body[key]
  }
  return dto
}

export function assertEncounterList(body) {
  if (!body || typeof body !== 'object' || !Array.isArray(body.items)) {
    throw apiError(502, 'HTTP_ERROR', '挂号列表不符合契约')
  }
  return { items: body.items.map((item) => assertEncounterDto(item)) }
}
