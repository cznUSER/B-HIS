/** P1-PRINT-STATUS-01: pure frontend order print status machine (zero network). */
export const PRINT_STATUS = Object.freeze({
  IDLE: 'IDLE',
  PREVIEW: 'PREVIEW',
  PRINTING: 'PRINTING',
  PRINTED: 'PRINTED',
  FAILED: 'FAILED',
})

export const PRINT_STATUS_LABELS = Object.freeze({
  IDLE: '未输出',
  PREVIEW: '预览中',
  PRINTING: '正在输出…',
  PRINTED: '本机已确认输出',
  FAILED: '输出失败',
})

/** Honest placeholder: must never claim server-side print audit. */
export const PLACEHOLDER_HONESTY =
  '本机占位输出：未对接打印服务，未回写服务端打印流水。刷新后打印态不持久化。'

export const SESSION_PERSISTENCE_NOTE = '本机会话未持久化打印态；刷新后回到未输出。'

const PRINTABLE = new Set(['COMMIT', 'PROC'])

function text(value) {
  return value == null ? '' : String(value).trim()
}

function first(...values) {
  for (const value of values) {
    const item = text(value)
    if (item) return item
  }
  return ''
}

export function canPreviewOrderStatus(status) {
  return PRINTABLE.has(text(status).toUpperCase())
}

export function printStatusLabel(status) {
  const key = text(status).toUpperCase()
  return PRINT_STATUS_LABELS[key] || PRINT_STATUS_LABELS.IDLE
}

function normalizeLine(line = {}, catalogLookup) {
  const itemNo = first(line.itemNo, line.itemId)
  const catalog = typeof catalogLookup === 'function' ? catalogLookup(itemNo, line) : null
  return {
    itemNo,
    itemName: first(line.itemName, line.name, catalog?.itemName, itemNo),
    itemType: first(line.itemType, line.type, catalog?.itemType),
    amount: line.amount ?? '',
    units: first(line.units, line.unit, catalog?.units),
    usageText: first(line.usageText, line.usage),
    frequencyText: first(line.frequencyText, line.frequency),
    dosage: line.dosage ?? '',
    dosageUnit: first(line.dosageUnit),
  }
}

/**
 * Build a read-only preview model bound to the given order/register.
 * Caller must pass the currently selected order fields so patientId cannot drift.
 */
export function buildPreviewModel({ register = {}, orderMeta = {}, lines = [], catalogLookup } = {}) {
  const patientId = first(orderMeta.patientId, register.patientId)
  const registerPatientId = first(register.patientId)
  if (patientId && registerPatientId && patientId !== registerPatientId) {
    throw new Error('预览患者与就诊不一致，已阻止串患')
  }
  const normalizedLines = (Array.isArray(lines) ? lines : [])
    .filter((line) => first(line.itemNo, line.itemId))
    .map((line) => normalizeLine(line, catalogLookup))
  return {
    patientId,
    patientName: first(orderMeta.patientName, register.patientName, register.name),
    patientNo: first(orderMeta.patientNo, register.patientNo),
    registerNo: first(orderMeta.registerNo, register.registerNo),
    registerId: first(orderMeta.registerId, register.registerId),
    orderNo: first(orderMeta.orderNo),
    orderId: first(orderMeta.orderId),
    orderStatus: first(orderMeta.status),
    diagnoseName: first(orderMeta.diagnoseName),
    lines: normalizedLines,
    generatedAt: new Date().toISOString(),
  }
}

export function createOrderPrintController() {
  const state = {
    status: PRINT_STATUS.IDLE,
    previewModel: null,
    errorMessage: '',
    lastPrintedAt: null,
    orderStatusSnapshot: '',
    honestyNote: PLACEHOLDER_HONESTY,
    sessionPersistenceNote: SESSION_PERSISTENCE_NOTE,
  }

  function openPreview(previewModel) {
    if (!previewModel || !first(previewModel.orderId, previewModel.orderNo)) return false
    if (![PRINT_STATUS.IDLE, PRINT_STATUS.PREVIEW, PRINT_STATUS.PRINTED, PRINT_STATUS.FAILED].includes(state.status)) {
      return false
    }
    state.previewModel = previewModel
    state.orderStatusSnapshot = text(previewModel.orderStatus)
    state.errorMessage = ''
    state.status = PRINT_STATUS.PREVIEW
    return true
  }

  function startOutput() {
    if (![PRINT_STATUS.PREVIEW, PRINT_STATUS.FAILED].includes(state.status) || !state.previewModel) return false
    state.errorMessage = ''
    state.status = PRINT_STATUS.PRINTING
    return true
  }

  function confirmPrinted(printedAt = new Date().toISOString()) {
    if (state.status !== PRINT_STATUS.PRINTING) return false
    state.status = PRINT_STATUS.PRINTED
    state.errorMessage = ''
    state.lastPrintedAt = printedAt
    return true
  }

  function fail(message = '输出失败，请重试') {
    if (state.status !== PRINT_STATUS.PRINTING) return false
    state.status = PRINT_STATUS.FAILED
    state.errorMessage = text(message) || '输出失败，请重试'
    return true
  }

  function retry() {
    if (state.status !== PRINT_STATUS.FAILED) return false
    return startOutput()
  }

  function reprint() {
    if (state.status !== PRINT_STATUS.PRINTED || !state.previewModel) return false
    state.errorMessage = ''
    state.status = PRINT_STATUS.PRINTING
    return true
  }

  function resetToIdle() {
    state.status = PRINT_STATUS.IDLE
    state.previewModel = null
    state.errorMessage = ''
    state.lastPrintedAt = null
    state.orderStatusSnapshot = ''
    return true
  }

  return {
    state,
    get status() {
      return state.status
    },
    get previewModel() {
      return state.previewModel
    },
    get errorMessage() {
      return state.errorMessage
    },
    get lastPrintedAt() {
      return state.lastPrintedAt
    },
    get orderStatusSnapshot() {
      return state.orderStatusSnapshot
    },
    get honestyNote() {
      return state.honestyNote
    },
    get sessionPersistenceNote() {
      return state.sessionPersistenceNote
    },
    get isLocked() {
      return state.status === PRINT_STATUS.PRINTING
    },
    openPreview,
    startOutput,
    confirmPrinted,
    fail,
    retry,
    reprint,
    resetToIdle,
  }
}