// 2026-10-12 P1-H05-REGISTER-UI 挂号请求对齐 OpenAPI 0.3.6 Register。
// 必填 patientId、departmentId、doctorId、visitDate。不使用 deptCode。

import { useMock } from './config.js'
import { apiError } from './errors.js'
import { findDepartment, findDoctor } from '../master/store.js'
import { REGISTER_SORT } from '../register/master.js'

const STATUSES = new Set(['NORMAL', 'USED', 'BACK', 'DIAGNOSING', 'DIAGNOSED', 'PAUSED'])

function asText(value) {
  return typeof value === 'string' ? value.trim() : ''
}

export function assertCreateRegisterRequest(input) {
  const src = input && typeof input === 'object' ? input : {}
  const patientId = asText(src.patientId)
  if (!patientId) throw apiError(400, 'VALIDATION_ERROR', '请先选择或建档患者')

  const departmentId = asText(src.departmentId)
  const doctorId = asText(src.doctorId)
  if (useMock()) {
    const department = findDepartment(departmentId)
    if (!department || department.status !== 'ACTIVE') {
      throw apiError(400, 'VALIDATION_ERROR', '请选择科室')
    }
    const doctor = findDoctor(doctorId)
    if (!doctor || doctor.status !== 'ACTIVE' || doctor.departmentId !== departmentId) {
      throw apiError(400, 'VALIDATION_ERROR', '请选择医生')
    }
  } else if (!departmentId) {
    throw apiError(400, 'VALIDATION_ERROR', '请选择科室')
  } else if (!doctorId) {
    throw apiError(400, 'VALIDATION_ERROR', '请选择医生')
  }

  const visitDate = asText(src.visitDate)
  if (!/^\d{4}-\d{2}-\d{2}$/.test(visitDate)) {
    throw apiError(400, 'VALIDATION_ERROR', '就诊日期无效')
  }

  const registerSort = asText(src.registerSort) || REGISTER_SORT.registerSort
  const fee = src.registerFee
  const registerFee = typeof fee === 'number' && Number.isFinite(fee) ? fee : 0
  return { patientId, departmentId, doctorId, visitDate, registerSort, registerFee }
}

function optionalText(body, key, target) {
  if (body[key] == null || body[key] === '') return
  if (typeof body[key] !== 'string') {
    throw apiError(502, 'HTTP_ERROR', '挂号单不符合契约')
  }
  target[key] = body[key]
}

export function assertRegisterDto(body) {
  if (!body || typeof body !== 'object') {
    throw apiError(502, 'HTTP_ERROR', '挂号单不符合契约')
  }
  const registerId = asText(body.registerId)
  const patientId = asText(body.patientId)
  const status = asText(body.status)
  if (!registerId || !patientId || !STATUSES.has(status)) {
    throw apiError(502, 'HTTP_ERROR', '挂号单不符合契约')
  }
  const dto = {
    registerId,
    patientId,
    status,
    registerFee: typeof body.registerFee === 'number' && Number.isFinite(body.registerFee) ? body.registerFee : 0,
    registerCharged: body.registerCharged === true,
  }
  for (const key of ['registerNo', 'departmentId', 'doctorId', 'registerSort', 'visitDate', 'registeredAt', 'backOperatorId', 'backOperatedAt']) {
    optionalText(body, key, dto)
  }
  return dto
}

export function assertRegisterList(body) {
  if (!body || typeof body !== 'object' || !Array.isArray(body.items)) {
    throw apiError(502, 'HTTP_ERROR', '挂号列表不符合契约')
  }
  return { items: body.items.map((item) => assertRegisterDto(item)) }
}
