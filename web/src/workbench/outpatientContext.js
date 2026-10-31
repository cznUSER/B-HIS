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

export const OUTPATIENT_CONTEXT_STORAGE_KEY = 'bhis.outpatient.ids.v1'
export const outpatientSelection = reactive({ pending: false, draftRevision: 0 })
let switchHandler = null
let selectionVersion = 0

function selectionStorage() {
  try { return globalThis.sessionStorage || null } catch { return null }
}

function selectionIds(value = {}) {
  const ids = {}
  for (const key of ['patientId', 'registerId']) {
    const item = value[key]
    if (item == null || item === '') continue
    if (typeof item !== 'string' || !/^[a-zA-Z0-9][a-zA-Z0-9._:-]{0,127}$/.test(item)) {
      throw new Error('患者或就诊参数无效，请重新选择')
    }
    ids[key] = item
  }
  return ids
}

export function readOutpatientSelection(query = {}) {
  if (Object.hasOwn(query, 'patientId') || Object.hasOwn(query, 'registerId')) return selectionIds(query)
  try {
    return selectionIds(JSON.parse(selectionStorage()?.getItem(OUTPATIENT_CONTEXT_STORAGE_KEY) || '{}'))
  } catch {
    try { selectionStorage()?.removeItem(OUTPATIENT_CONTEXT_STORAGE_KEY) } catch {}
    return {}
  }
}

function persistSelection() {
  try {
    const ids = selectionIds({ patientId: outpatientContext.patient?.patientId, registerId: outpatientContext.register?.registerId })
    const store = selectionStorage()
    if (ids.patientId) store?.setItem(OUTPATIENT_CONTEXT_STORAGE_KEY, JSON.stringify(ids))
    else store?.removeItem(OUTPATIENT_CONTEXT_STORAGE_KEY)
  } catch {}
}

export function setOutpatientSwitchHandler(handler) {
  switchHandler = typeof handler === 'function' ? handler : null
}

export function outpatientSwitchMessage(patient, register = null) {
  const changingPatient = patientIdOf(patient) !== patientIdOf(outpatientContext.patient)
  return {
    title: changingPatient ? '切换患者？' : '切换就诊？',
    body: changingPatient
      ? '将切换到患者「' + patientDisplayName(patient) + '」（' + (patient.patientNo || '系统 Id：' + patient.patientId) + '）。\n确认后会清空上一患者未提交的本机草稿与临时勾选（开单/诊断未保存内容、收费预览勾选、发药数量编辑、退费勾选等）。\n已保存到服务器的数据不会删除。'
      : '将切换到就诊「' + (register?.registerNo || register?.registerId || '未选择') + '」，并清空仅属于上一就诊的未提交草稿与临时勾选。\n已保存到服务器的数据不会删除。',
    confirmLabel: '确认切换',
    cancelLabel: '取消',
  }
}

function needsSwitch(patient, register) {
  const currentId = patientIdOf(outpatientContext.patient)
  if (!currentId) return false
  return currentId !== patientIdOf(patient)
    || Boolean(outpatientContext.register?.registerId && outpatientContext.register.registerId !== register?.registerId)
}

export async function selectOutpatientContext({ patient, register = null }, { isCurrent = () => true } = {}) {
  if (!patientIdOf(patient) || (register && register.patientId !== patient.patientId)) return false
  if (outpatientSelection.pending) return false
  const version = ++selectionVersion
  const changing = needsSwitch(patient, register)
  if (changing) {
    const message = outpatientSwitchMessage(patient, register)
    outpatientSelection.pending = true
    let accepted
    try {
      accepted = switchHandler ? await switchHandler(message) : (typeof window !== 'undefined' && window.confirm(message.title + '\n' + message.body))
    } finally {
      outpatientSelection.pending = false
    }
    if (!accepted || version !== selectionVersion || !isCurrent()) return false
  }
  if (!isCurrent() || version !== selectionVersion) return false
  if (changing) {
    resetEpisode()
  }
  const samePatient = patientIdOf(patient) === patientIdOf(outpatientContext.patient)
  outpatientContext.patient = { ...(samePatient ? outpatientContext.patient : {}), ...patient }
  outpatientContext.register = register ? { ...register } : null
  addHistory('患者资料已读取', patientDisplayName(patient))
  persistSelection()
  if (changing) outpatientSelection.draftRevision++
  return true
}

export async function resolveOutpatientSelection(ids, { authenticated, loadPatient, loadRegisters }) {
  if (!authenticated) throw new Error('请先登录后读取患者资料')
  const selection = selectionIds(ids)
  let register = null
  if (selection.registerId) {
    let offset = 0
    while (true) {
      const listed = await loadRegisters({ patientId: selection.patientId, offset, limit: 100 })
      register = listed.items.find((item) => item.registerId === selection.registerId)
      if (register || listed.hasMore !== true) break
      if (!Number.isInteger(listed.nextOffset) || listed.nextOffset <= offset) throw new Error('就诊列表分页无效，请重新选择')
      offset = listed.nextOffset
    }
    if (!register || (selection.patientId && register.patientId !== selection.patientId)) throw new Error('就诊与患者不匹配或已不可读取，请重新选择')
    selection.patientId = register.patientId
  }
  if (!selection.patientId) return { patient: null, register: null }
  const patient = await loadPatient(selection.patientId)
  if (patient.patientId !== selection.patientId) throw new Error('患者资料不匹配，请重新选择')
  return { patient, register }
}


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
  return String(value.patientId || value.patient_id || '').trim()
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
  selectionVersion++
  Object.assign(outpatientContext, initialContext())
  persistSelection()
}

export function patientDisplayName(patient) {
  if (!patient) return '未选择患者'
  return String(patient.name || patient.patientName || patient.patientNo || patient.patientId || '未命名患者').trim()
}

function dateParts(value) {
  if (typeof value !== 'string' || !/^\d{4}-\d{2}-\d{2}$/.test(value)) return null
  const date = new Date(value + 'T00:00:00Z')
  return Number.isFinite(date.getTime()) && date.toISOString().slice(0, 10) === value ? value.split('-').map(Number) : null
}

export function patientDisplayMeta(patient, serverDate = '') {
  if (!patient) return ''
  const sex = patient.sexLabel || patient.sexName || ({ M: '男', F: '女', U: '未知' }[patient.sex]) || ''
  let age = patient.age == null || patient.age === '' ? null : Number(patient.age)
  if (age == null) {
    const birthday = dateParts(patient.birthday)
    const today = dateParts(serverDate)
    if (birthday && today && patient.birthday <= serverDate) {
      age = today[0] - birthday[0] - (today[1] < birthday[1] || (today[1] === birthday[1] && today[2] < birthday[2]) ? 1 : 0)
    }
  }
  const ageLabel = Number.isInteger(age) && age >= 0 ? age + '岁' : ''
  return [sex, ageLabel].filter(Boolean).join(' · ')
}

export function patientFareDisplay(patient) {
  const fare = String(patient?.fareIdentity || '').trim()
  return '费别：' + ({ SELF: '自费', OTHER: '其他' }[fare] || fare || '未登记')
}

export function patientRiskDisplay(patient) {
  const history = String(patient?.allergyHistory || '').trim()
  return { label: '过敏史：' + (history || '未记录'), highRisk: Boolean(history) }
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

function deferLegacySwitch(patient, register = null) {
  if (outpatientSelection.pending) return true
  if (typeof window === 'undefined' && !switchHandler) return false
  void selectOutpatientContext({ patient, register }).catch(() => {})
  return true
}

export function setPatient(patient) {
  if (!patient || !patientIdOf(patient) || outpatientSelection.pending) return false
  const nextId = patientIdOf(patient)
  const currentId = patientIdOf(outpatientContext.patient)
  const changing = nextId && currentId && nextId !== currentId
  if (changing && deferLegacySwitch(patient)) return false
  if (changing) {
    resetEpisode()
  }
  outpatientContext.patient = { ...(changing ? {} : outpatientContext.patient), ...patient }
  addHistory('患者资料已读取', patient.name || patient.patientName || patient.patientNo || patient.patientId || '')
  persistSelection()
  if (changing) outpatientSelection.draftRevision++
  return true
}

export function setRegister(register) {
  if (!register || outpatientSelection.pending) return false
  const patient = register.patientId && patientIdOf(outpatientContext.patient) !== String(register.patientId).trim()
    ? { patientId: register.patientId, name: register.patientName, patientNo: register.patientNo }
    : outpatientContext.patient
  if (!patientIdOf(patient)) return false
  const changing = needsSwitch(patient, register)
  if (changing && deferLegacySwitch(patient, register)) return false
  if (changing) {
    resetEpisode()
  }
  if (patientIdOf(patient) !== patientIdOf(outpatientContext.patient)) outpatientContext.patient = { ...patient }
  outpatientContext.register = { ...(changing ? {} : outpatientContext.register), ...register }
  addHistory('挂号记录已读取', register.registerNo || register.registerId || '')
  persistSelection()
  if (changing) outpatientSelection.draftRevision++
  return true
}

function belongsToSelection(value) {
  const patientId = patientIdOf(outpatientContext.patient)
  if (!patientId || !value) return false
  if (value.patientId && value.patientId !== patientId) return false
  const registerId = outpatientContext.register?.registerId
  return !value.registerId || !registerId || value.registerId === registerId
}

export function setDiagnosis(diagnosis, list = null) {
  if (outpatientSelection.pending) return false
  if (diagnosis && belongsToSelection(diagnosis)) outpatientContext.diagnosis = { ...outpatientContext.diagnosis, ...diagnosis }
  if (Array.isArray(list)) outpatientContext.diagnoses = list.filter(belongsToSelection).slice(-8)
  if (diagnosis && belongsToSelection(diagnosis)) addHistory('诊断记录已读取', diagnosis.status || diagnosis.diagnoseId || '')
  return true
}

export function setOrders(orders) {
  if (!Array.isArray(orders) || outpatientSelection.pending) return false
  outpatientContext.orders = orders.filter(belongsToSelection).slice(-8)
  const latest = outpatientContext.orders.at(-1)
  if (latest) addHistory('处方记录已读取', latest.orderNo || latest.orderId || '')
  return true
}

export function setCharges(charges) {
  if (!Array.isArray(charges) || outpatientSelection.pending) return false
  outpatientContext.charges = charges.filter(belongsToSelection).slice(-8)
  const latest = outpatientContext.charges.at(-1)
  if (latest) addHistory('收费记录已读取', latest.squareNo || latest.chargeId || '')
  return true
}

export function setDispense(dispense) {
  if (!dispense || outpatientSelection.pending || !belongsToSelection(dispense)) return false
  outpatientContext.dispense = { ...dispense }
  addHistory('发药/退药记录已读取', dispense.dispenseNo || dispense.dispenseId || '')
  return true
}

export function recordAction(label, detail = '') {
  if (outpatientSelection.pending) return
  outpatientContext.lastAction = label
  addHistory(label, detail)
}
