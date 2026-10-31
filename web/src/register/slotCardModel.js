import { apiError } from '../api/errors.js'

// The board is authoritative: null means no schedule, zero means unavailable.
export function assertSlotBoard(body) {
  const invalid = () => { throw apiError(502, 'HTTP_ERROR', '号源响应不符合契约') }
  const count = value => value === null || (Number.isInteger(value) && value >= 0)
  if (!body || !/^\d{4}-\d{2}-\d{2}$/.test(body.visitDate) || !Array.isArray(body.departments) || !Array.isArray(body.doctors)) invalid()
  for (const dept of body.departments) if (!dept.departmentId || !dept.departmentName || !count(dept.remainTotal)) invalid()
  for (const doctor of body.doctors) {
    if (!doctor.doctorId || !doctor.departmentId || !doctor.doctorName || typeof doctor.scheduleConfigured !== 'boolean' || !count(doctor.remain) || !Array.isArray(doctor.slots)) invalid()
    if (!doctor.scheduleConfigured && (doctor.remain !== null || doctor.slots.length)) invalid()
    if (doctor.scheduleConfigured && doctor.remain === null) invalid()
    for (const slot of doctor.slots) {
      if (!Number.isFinite(Date.parse(slot.start)) || !Number.isFinite(Date.parse(slot.end)) || Date.parse(slot.start) >= Date.parse(slot.end) || !['AM', 'PM'].includes(slot.period)) invalid()
      if (![slot.capacity, slot.used, slot.remain].every(n => Number.isInteger(n) && n >= 0) || slot.remain > slot.capacity) invalid()
    }
  }
  return body
}

export function reduceSelection(state, event) {
  if (event.type === 'patient') return { ...state, departmentId: '', doctorId: '', slot: null, registerFee: 0, paymentPreference: 'CASH' }
  if (event.type === 'date') return { ...state, doctorId: '', slot: null }
  if (event.type === 'department') return { ...state, departmentId: event.id, doctorId: '', slot: null }
  if (event.type === 'doctor') return { ...state, doctorId: event.id, slot: null }
  return state
}

export function canConfirmSelection({ patientId, departmentId, doctor, slot, appointmentMode = false }) {
  if (!patientId || !departmentId || !doctor?.doctorId) return false
  if (doctor.scheduleConfigured === false) return !appointmentMode
  return doctor.scheduleConfigured === true && doctor.remain > 0 && Boolean(slot && slot.remain > 0)
}

export function historySince(today) {
  return new Date(Date.parse(`${today}T00:00:00+08:00`) - 89 * 86400000).toLocaleDateString('en-CA', { timeZone: 'Asia/Shanghai' })
}

export function patientHistory(items, patientId, today) {
  const since = historySince(today)
  return items.filter(row => row.patientId === patientId && row.visitDate >= since)
    .sort((a, b) => String(b.visitDate + (b.appointmentStart || b.registeredAt || '')).localeCompare(String(a.visitDate + (a.appointmentStart || a.registeredAt || ''))))
}

export function slotTime(value) {
  if (!Number.isFinite(Date.parse(value))) return '—'
  return new Intl.DateTimeFormat('zh-CN', { timeZone: 'Asia/Shanghai', hour: '2-digit', minute: '2-digit', hour12: false }).format(new Date(value))
}

export function paymentSummary(row, charge) {
  if (charge?.paid) return charge.payType === 'CASH' ? '现金 · 已收讫' : '已收费 · 方式未返回'
  if (row.registerCharged) return '已收费 · 方式未返回'
  return Number(row.registerFee) > 0 ? '待收费' : '无需收费'
}
