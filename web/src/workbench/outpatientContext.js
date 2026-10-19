import { reactive } from 'vue'

// Shared read-only context for the outpatient frame. Each module keeps its
// existing API calls and writes the successful response here so the shell can
// show the same patient and workflow state while the user moves between pages.
function initialContext() {
  return {
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
  }
}

export const outpatientContext = reactive(initialContext())

export const OUTPATIENT_WORKFLOW_STEPS = Object.freeze([
  { key: 'patient', label: '患者', completeLabel: '已选患者' },
  { key: 'register', label: '挂号', completeLabel: '已挂号' },
  { key: 'diagnosis', label: '诊断', completeLabel: '已记录' },
  { key: 'order', label: '开单', completeLabel: '已提交' },
  { key: 'charge', label: '收费', completeLabel: '已收费' },
  { key: 'dispense', label: '发药', completeLabel: '已处理' },
])

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

function patientIdOf(value) {
  if (!value || typeof value !== 'object') return ''
  return String(value.patientId || value.patientNo || value.patient_id || '').trim()
}

function resetEpisode() {
  outpatientContext.register = null
  outpatientContext.diagnosis = null
  outpatientContext.diagnoses = []
  outpatientContext.orders = []
  outpatientContext.charges = []
  outpatientContext.dispense = null
  outpatientContext.history = []
  outpatientContext.lastAction = ''
}

export function clearOutpatientContext() {
  Object.assign(outpatientContext, initialContext())
}

export function patientDisplayName(patient) {
  if (!patient) return '未选择患者'
  return String(patient.name || patient.patientName || patient.patientNo || patient.patientId || '未命名患者').trim()
}

export function patientDisplayMeta(patient) {
  if (!patient) return ''
  const parts = [patient.sexLabel || patient.sexName || patient.sex, patient.age == null ? '' : `${patient.age}岁`]
    .map((value) => String(value || '').trim())
    .filter(Boolean)
  return parts.join(' · ')
}

export function deriveWorkflowStatus(context = outpatientContext) {
  const patient = context?.patient
  if (!patientIdOf(patient)) return { key: 'PATIENT_REQUIRED', label: '待选择患者', next: '从患者查询或队列选择患者', tone: 'neutral' }

  const register = context?.register
  if (register?.status === 'BACK') return { key: 'REGISTER_BACK', label: '已退号', next: '重新挂号后继续', tone: 'danger' }
  const latestOrder = Array.isArray(context?.orders) ? context.orders.at(-1) : null
  const latestCharge = Array.isArray(context?.charges) ? context.charges.at(-1) : null
  const diagnosis = context?.diagnosis
  const dispense = context?.dispense

  if (latestCharge && latestCharge.status !== 'ABATE' && latestCharge.paid !== true && latestCharge.paidAt == null) {
    return { key: 'WAIT_CHARGE', label: '待收费', next: '核对费用并完成收费', tone: 'warning' }
  }
  if (latestOrder?.status === 'COMMIT' && latestOrder.charged !== true && latestCharge?.paid !== true) {
    return { key: 'WAIT_CHARGE', label: '待收费', next: '核对费用并完成收费', tone: 'warning' }
  }
  if (latestOrder?.status === 'COMMIT' && (latestOrder.charged === true || latestCharge?.paid === true) && !dispense) {
    return { key: 'WAIT_DISPENSE', label: '待发药', next: '核对库存并完成发药', tone: 'info' }
  }
  if (diagnosis && ['NEW', 'SAVE'].includes(diagnosis.status)) {
    return { key: 'WAIT_DIAGNOSIS', label: '待提交诊断', next: '补充结构化诊疗记录并提交', tone: 'warning' }
  }
  if (!register) return { key: 'REGISTER_REQUIRED', label: '待挂号', next: '先完成挂号或选择有效就诊', tone: 'neutral' }
  if (register.status === 'NORMAL') return { key: 'WAIT_CHECK_IN', label: '待报到', next: '确认患者到诊', tone: 'info' }
  if (['USED', 'DIAGNOSING'].includes(register.status)) return { key: 'IN_PROGRESS', label: '诊疗中', next: '完成诊断和开单', tone: 'info' }
  if (register.status === 'DIAGNOSED') return { key: 'WAIT_ORDER', label: '待开单', next: '录入并提交医嘱', tone: 'info' }
  return { key: 'IN_PROGRESS', label: '处理中', next: '按当前模块继续处理', tone: 'info' }
}

export function deriveWorkflowProgress(context = outpatientContext) {
  const status = deriveWorkflowStatus(context)
  const currentStepKey = {
    PATIENT_REQUIRED: 'patient',
    REGISTER_REQUIRED: 'register',
    WAIT_CHECK_IN: 'register',
    WAIT_DIAGNOSIS: 'diagnosis',
    WAIT_ORDER: 'order',
    WAIT_CHARGE: 'charge',
    WAIT_DISPENSE: 'dispense',
  }[status.key]
  const patient = context?.patient
  const register = context?.register
  const diagnosis = context?.diagnosis
  const orders = Array.isArray(context?.orders) ? context.orders : []
  const charges = Array.isArray(context?.charges) ? context.charges : []
  const dispense = context?.dispense
  const order = orders.at(-1)
  const charge = charges.at(-1)
  return OUTPATIENT_WORKFLOW_STEPS.map((step) => {
    const complete = step.key === 'patient'
      ? Boolean(patientIdOf(patient))
      : step.key === 'register'
        ? Boolean(register && register.status !== 'BACK')
        : step.key === 'diagnosis'
          ? Boolean(diagnosis && ['SAVE', 'COMMIT'].includes(diagnosis.status))
          : step.key === 'order'
            ? Boolean(order && ['COMMIT', 'PROC'].includes(order.status))
            : step.key === 'charge'
              ? Boolean(charge && (charge.paid === true || charge.paidAt))
              : Boolean(dispense)
    return { ...step, complete, current: currentStepKey === step.key }
  })
}

export function setPatient(patient) {
  if (!patient) return
  const nextId = patientIdOf(patient)
  const currentId = patientIdOf(outpatientContext.patient)
  if (nextId && currentId && nextId !== currentId) resetEpisode()
  outpatientContext.patient = { ...outpatientContext.patient, ...patient }
  addHistory('患者资料已读取', patient.name || patient.patientName || patient.patientNo || patient.patientId || '')
}

export function setRegister(register) {
  if (!register) return
  if (register.patientId && patientIdOf(outpatientContext.patient) !== String(register.patientId).trim()) {
    setPatient({ patientId: register.patientId, name: register.patientName, patientNo: register.patientNo })
  }
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
  const current = orders.find((item) => item?.patientId)
  if (current?.patientId && patientIdOf(outpatientContext.patient) !== String(current.patientId).trim()) {
    setPatient({ patientId: current.patientId, name: current.patientName, patientNo: current.patientNo })
  }
  outpatientContext.orders = orders.slice(-8)
  const latest = orders[orders.length - 1]
  if (latest) addHistory('处方记录已读取', latest.orderNo || latest.orderId || '')
}

export function setCharges(charges) {
  if (!Array.isArray(charges)) return
  const current = charges.find((item) => item?.patientId)
  if (current?.patientId && patientIdOf(outpatientContext.patient) !== String(current.patientId).trim()) {
    setPatient({ patientId: current.patientId, name: current.patientName, patientNo: current.patientNo })
  }
  outpatientContext.charges = charges.slice(-8)
  const latest = charges[charges.length - 1]
  if (latest) addHistory('收费记录已读取', latest.squareNo || latest.chargeId || '')
}

export function setDispense(dispense) {
  if (!dispense) return
  if (dispense.patientId && patientIdOf(outpatientContext.patient) !== String(dispense.patientId).trim()) {
    setPatient({ patientId: dispense.patientId, name: dispense.patientName, patientNo: dispense.patientNo })
  }
  outpatientContext.dispense = { ...outpatientContext.dispense, ...dispense }
  addHistory('发药/退药记录已读取', dispense.dispenseNo || dispense.dispenseId || '')
}

export function recordAction(label, detail = '') {
  outpatientContext.lastAction = label
  addHistory(label, detail)
}
