// 2026-10-12 P1-H05-REGISTER-UI 挂号主数据与退号口径。
// 2026-10-12 对齐 OpenAPI 0.3.6：科室键为 departmentId（dept-1 / dept-2）。
// 2026-10-12 P1-H05b-REGISTER-CHECKIN-UI 报到：NORMAL 或 USED 可报到。
// 号别固定占位普通号。无排班日历。

export const DEPARTMENTS = [
  { departmentId: 'dept-1', name: '内科' },
  { departmentId: 'dept-2', name: '外科' },
]

export const DOCTORS = [
  { doctorId: 'doc-1', name: '张医生', departmentId: 'dept-1' },
  { doctorId: 'doc-2', name: '李医生', departmentId: 'dept-2' },
]

export const REGISTER_SORT = { registerSort: 'sort-normal', name: '普通号' }

export const STATUS_LABELS = {
  NORMAL: '正常号',
  USED: '已用',
  BACK: '退号',
  DIAGNOSING: '就诊中',
  DIAGNOSED: '就诊完成',
  PAUSED: '挂起',
}

const USED_REJECT = '当前号已使用或就诊中，不能退号'
const BACK_REJECT = '该号已退，不能再次退号'

const REJECT_MESSAGE = {
  USED: USED_REJECT,
  DIAGNOSING: USED_REJECT,
  DIAGNOSED: USED_REJECT,
  PAUSED: USED_REJECT,
  BACK: BACK_REJECT,
}

export function doctorsForDept(departmentId) {
  return DOCTORS.filter((item) => item.departmentId === departmentId)
}

export function deptName(departmentId) {
  const found = DEPARTMENTS.find((item) => item.departmentId === departmentId)
  return found ? found.name : departmentId || '—'
}

export function doctorName(doctorId) {
  const found = DOCTORS.find((item) => item.doctorId === doctorId)
  return found ? found.name : doctorId || '—'
}

export function registerSortName(registerSort) {
  if (!registerSort || registerSort === REGISTER_SORT.registerSort) return REGISTER_SORT.name
  return registerSort
}

export function statusLabel(status) {
  return STATUS_LABELS[status] || status || '—'
}

export function canCancel(status) {
  return status === 'NORMAL'
}

export function canCheckIn(status) {
  return status === 'NORMAL' || status === 'USED'
}

export function cancelRejectMessage(status) {
  if (canCancel(status)) return ''
  return REJECT_MESSAGE[status] || USED_REJECT
}

export function maskIdNumber(value) {
  const text = typeof value === 'string' ? value.trim() : ''
  if (!text) return '—'
  if (text.length <= 8) return text
  return `${text.slice(0, 4)}**********${text.slice(-4)}`
}
