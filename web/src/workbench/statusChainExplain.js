import { deriveWorkflowProgress, deriveWorkflowStatus } from './outpatientContext.js'

export const STATUS_CHAIN_STATE_LABELS = Object.freeze({
  done: '已完成', current: '当前', pending: '待处理', blocked: '阻塞', skipped: '跳过', void: '作废',
})

const DOMAIN_STEPS = { registers: 'register', orders: 'order', charges: 'charge', dispenses: 'dispense', diagnoses: 'diagnosis' }
const DOMAIN_LABELS = { registers: '挂号', orders: '开单', charges: '收费', dispenses: '发药', diagnoses: '诊断' }

export function classifyDomainError(error) {
  const forbidden = [error?.status, error?.statusCode, error?.response?.status].some(status => Number(status) === 403)
    || [error?.code, error?.error?.code, error?.response?.data?.code].some(code => String(code).toUpperCase() === 'FORBIDDEN')
  return { forbidden, message: forbidden ? '该域无权限，请联系管理员确认授权' : '该域读取失败，请重试；不能据此判断无数据' }
}

function rows(payload) {
  return Array.isArray(payload?.items) ? payload.items : []
}

function ordered(items) {
  return [...items].sort((left, right) => String(left.updatedAt || left.operateAt || left.dispensedAt || left.createdAt || '')
    .localeCompare(String(right.updatedAt || right.operateAt || right.dispensedAt || right.createdAt || '')))
}

export function contextFromStatusChainPayloads(anchor, payloads) {
  const register = rows(payloads.registers).find(row => row.registerId === anchor.registerId && row.patientId === anchor.patientId)
  if (!register) throw new Error('未找到当前挂号，不能使用其他就诊的数据')
  if (payloads.patient?.patientId !== anchor.patientId) throw new Error('患者与当前挂号不一致')
  const scoped = payload => ordered(rows(payload).filter(row => row.registerId === anchor.registerId && (!row.patientId || row.patientId === anchor.patientId)))
  const diagnoses = scoped(payloads.diagnoses)
  const dispenses = scoped(payloads.dispenses)
  return {
    patient: { ...payloads.patient }, register: { ...register },
    orders: scoped(payloads.orders), charges: scoped(payloads.charges), diagnoses,
    diagnosis: diagnoses.at(-1) || null, dispense: dispenses.at(-1) || null,
  }
}

export function buildStatusChainSummary(context, options = {}) {
  const source = context || {}
  const domainAccess = Object.fromEntries(Object.entries(options.domainAccess || {}).map(([domain, access]) => [domain, {
    ...access, ...(access.forbidden ? { ok: false, empty: false, message: '该域无权限，请联系管理员确认授权' } : {}),
  }]))
  const view = { ...source }
  for (const [domain, access] of Object.entries(domainAccess)) {
    if (access.ok !== false) continue
    if (domain === 'registers') view.register = null
    else if (domain === 'diagnoses') { view.diagnosis = null; view.diagnoses = [] }
    else if (domain === 'dispenses') view.dispense = null
    else if (domain === 'orders' || domain === 'charges') view[domain] = []
  }
  const status = deriveWorkflowStatus(view)
  const steps = deriveWorkflowProgress(view).map(step => ({
    ...step,
    state: step.current ? 'current' : step.complete ? 'done' : 'pending',
    reason: step.current ? status.next : step.complete ? step.completeLabel : '等待前序流程或该域处理',
  }))
  const stepOf = key => steps.find(step => step.key === key)
  const mark = (key, state, reason) => Object.assign(stepOf(key), {
    state, reason, complete: state === 'done', current: state === 'current' || state === 'blocked',
  })
  const partialReasons = []
  const blockedReasons = []
  const activeOrders = (view.orders || []).filter(row => row.status !== 'ABATE')
  const activeCharges = (view.charges || []).filter(row => row.status !== 'ABATE')
  const diagnosis = view.diagnosis
  const dispense = view.dispense
  const paid = charge => charge.paid === true || Boolean(charge.paidAt)
  const documents = [...(view.orders || []), ...(view.charges || []), ...(view.diagnoses || []), ...(diagnosis ? [diagnosis] : [])]
  const terminalAbate = documents.length > 0 && documents.every(row => row.status === 'ABATE') && !dispense
    && Object.values(domainAccess).every(access => access.ok !== false)
  const isBack = view.register?.status === 'BACK'
  let kind = status.key === 'PATIENT_REQUIRED' ? 'empty' : 'normal'

  if (diagnosis?.status === 'ABATE') mark('diagnosis', 'void', '诊断已作废，不代表诊断完成')
  for (const [key, entries] of [['order', view.orders || []], ['charge', view.charges || []]]) {
    if (entries.length && entries.every(row => row.status === 'ABATE')) mark(key, 'void', '该域单据均已作废')
  }
  if (dispense?.status === 'ABATE') mark('dispense', 'void', '发药单已作废，不代表发药完成')
  else if (dispense) {
    const lineStates = (dispense.items || []).map(row => row.lineDispenseStatus).filter(Boolean)
    const allFull = lineStates.length ? lineStates.every(state => state === 'FULL') : dispense.lineDispenseStatus === 'FULL'
    const partial = dispense.lineDispenseStatus === 'PARTIAL' || lineStates.includes('PARTIAL') || (lineStates.includes('FULL') && lineStates.includes('NONE'))
    if (allFull) mark('dispense', 'done', '发药明细均已发完')
    else {
      const reason = partial ? '部分发药：仍有明细未发完，请在药房核对剩余数量' : '发药尚未完成，请核对发药明细'
      mark('dispense', 'current', reason)
      partialReasons.push(reason)
    }
  }
  if (activeCharges.some(paid) && !dispense) {
    partialReasons.push('收费已完成，尚无发药记录；是否需要发药请核对处方明细')
    mark('dispense', 'current', partialReasons.at(-1))
  }
  if (activeOrders.some(row => row.status === 'SAVE') && activeOrders.some(row => ['COMMIT', 'PROC'].includes(row.status))) {
    partialReasons.push('部分处方已提交，仍有草稿待处理')
  }
  const unpaid = activeCharges.some(row => !paid(row)) || activeOrders.some(row => row.status === 'COMMIT' && row.charged !== true
    && !activeCharges.some(charge => charge.orderId === row.orderId && paid(charge)))
  if (unpaid) {
    blockedReasons.push('存在未收费或未支付单据，后续发药需先核对收费结果')
    mark('charge', 'blocked', blockedReasons.at(-1))
    if (stepOf('dispense').state === 'current') mark('dispense', 'pending', '等待收费结果核对')
  }
  if (view.register?.status === 'PAUSED') {
    blockedReasons.push('当前就诊已暂停，请在有权限的业务页面核对恢复条件')
    mark('register', 'blocked', blockedReasons.at(-1))
  }
  for (const [domain, access] of Object.entries(domainAccess)) {
    if (access.ok !== false || !DOMAIN_STEPS[domain]) continue
    const reason = DOMAIN_LABELS[domain] + '：' + (access.forbidden ? '该域无权限' : '读取失败，请重试')
    blockedReasons.push(reason)
    mark(DOMAIN_STEPS[domain], 'blocked', reason)
  }
  let explain = status.label + '；下一步：' + status.next
  let label = status.label
  let next = status.next
  let tone = status.tone
  if (partialReasons.length) { kind = 'partial'; explain = partialReasons.join('；') }
  if (blockedReasons.length) { kind = 'blocked'; explain = blockedReasons.join('；') }
  if (isBack || terminalAbate) {
    kind = 'void'
    explain = isBack ? '当前挂号已退号，后续流程不再继续；已有业务单据仍须分别核对' : '当前已读取的业务单据均已作废，不代表诊疗完成'
    if (isBack) {
      mark('register', 'void', '挂号已退号')
      for (const key of ['diagnosis', 'order', 'charge', 'dispense']) if (!stepOf(key).complete && stepOf(key).state !== 'blocked') mark(key, 'skipped', '本次挂号已退号，不继续此步骤')
    } else { label = '单据已作废'; next = '核对作废单据后按业务规则继续'; tone = 'danger' }
  }
  return {
    kind, label, next, tone, key: status.key, registerId: String(source.register?.registerId || ''),
    steps, blockReason: blockedReasons.join('；'), explain, partialReasons, domainAccess,
    ...(options.refreshedAt ? { refreshedAt: options.refreshedAt } : {}),
  }
}
