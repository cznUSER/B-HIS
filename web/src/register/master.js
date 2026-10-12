// 2026-10-12 P1-H05-REGISTER-UI 挂号主数据与退号口径。
// 科室、医生用演示种子；号别固定占位普通号。无排班日历。

export const DEPARTMENTS = [
  { deptCode: 'dept-1', name: '内科' },
  { deptCode: 'dept-2', name: '外科' },
]

export const DOCTORS = [
  { doctorId: 'doc-1', name: '张医生', deptCode: 'dept-1' },
  { doctorId: 'doc-2', name: '李医生', deptCode: 'dept-2' },
]

export const CLINIC_SORT = { clinicSort: 'sort-normal', name: '普通号' }

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

export function doctorsForDept(deptCode) {
  return DOCTORS.filter((item) => item.deptCode === deptCode)
}

export function deptName(deptCode) {
  const found = DEPARTMENTS.find((item) => item.deptCode === deptCode)
  return found ? found.name : deptCode || '—'
}

export function doctorName(doctorId) {
  const found = DOCTORS.find((item) => item.doctorId === doctorId)
  return found ? found.name : doctorId || '—'
}

export function clinicSortName(clinicSort) {
  if (!clinicSort || clinicSort === CLINIC_SORT.clinicSort) return CLINIC_SORT.name
  return clinicSort
}

export function statusLabel(status) {
  return STATUS_LABELS[status] || status || '—'
}

export function canCancel(status) {
  return status === 'NORMAL'
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
