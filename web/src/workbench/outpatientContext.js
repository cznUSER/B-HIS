import { reactive } from 'vue'

// Shared read-only context for the outpatient frame. Each module keeps its
// existing API calls and writes the successful response here so the shell can
// show the same patient and workflow state while the user moves between pages.
export const outpatientContext = reactive({
  patient: null,
  register: null,
  diagnosis: null,
  diagnoses: [],
  orders: [],
  charges: [],
  dispense: null,
  history: [],
  lastAction: '',
  updatedAt: '',
})

function now() {
  return new Date().toISOString()
}

function addHistory(label, detail = '') {
  outpatientContext.history = [
    { label, detail, at: now() },
    ...outpatientContext.history,
  ].slice(0, 6)
  outpatientContext.updatedAt = now()
}

export function setPatient(patient) {
  if (!patient) return
  outpatientContext.patient = { ...outpatientContext.patient, ...patient }
  addHistory('患者资料已读取', patient.name || patient.patientName || patient.patientNo || patient.patientId || '')
}

export function setRegister(register) {
  if (!register) return
  outpatientContext.register = { ...outpatientContext.register, ...register }
  addHistory('挂号记录已读取', register.registerNo || register.registerId || '')
}

export function setDiagnosis(diagnosis, list = null) {
  if (diagnosis) outpatientContext.diagnosis = { ...outpatientContext.diagnosis, ...diagnosis }
  if (Array.isArray(list)) outpatientContext.diagnoses = list.slice(-8)
  if (diagnosis) addHistory('诊断记录已读取', diagnosis.status || diagnosis.diagnoseId || '')
}

export function setOrders(orders) {
  if (!Array.isArray(orders)) return
  outpatientContext.orders = orders.slice(-8)
  const current = orders[orders.length - 1]
  if (current) addHistory('处方记录已读取', current.orderNo || current.orderId || '')
}

export function setCharges(charges) {
  if (!Array.isArray(charges)) return
  outpatientContext.charges = charges.slice(-8)
  const current = charges[charges.length - 1]
  if (current) addHistory('收费记录已读取', current.squareNo || current.chargeId || '')
}

export function setDispense(dispense) {
  if (!dispense) return
  outpatientContext.dispense = { ...outpatientContext.dispense, ...dispense }
  addHistory('发药/退药记录已读取', dispense.dispenseNo || dispense.dispenseId || '')
}

export function recordAction(label, detail = '') {
  outpatientContext.lastAction = label
  addHistory(label, detail)
}
