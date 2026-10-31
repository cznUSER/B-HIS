/**
 * P0-B-QUEUE-01 shared outpatient queue helpers (morph A: zero new API).
 * Filter maps follow H02 p0-field-freeze §6; 「待开单」primary anchor = DIAGNOSED.
 */
function text(value) {
  return typeof value === 'string' ? value.trim() : ''
}

function searchable(row) {
  return [
    row?.patientName,
    row?.patientNo,
    row?.registerNo,
    row?.registerId,
    row?.patientId,
    row?.idNumber,
    row?.idNumberMasked,
  ]
    .map(text)
    .join(' ')
    .toLowerCase()
}

function isChargePaid(charge) {
  if (!charge) return false
  if (charge.paid === true) return true
  return Boolean(text(charge.paidAt))
}

function isChargeActive(charge) {
  return Boolean(charge) && charge.status !== 'ABATE'
}

function relatedByRegister(list, registerId) {
  const id = text(registerId)
  if (!id) return []
  return (Array.isArray(list) ? list : []).filter((row) => text(row?.registerId) === id)
}

function rowWaitCharge(row) {
  const orders = Array.isArray(row?.orders) ? row.orders : []
  const charges = Array.isArray(row?.charges) ? row.charges : []
  const hasCommit = orders.some((order) => order?.status === 'COMMIT')
  if (!hasCommit) return false
  const paid = orders.some((order) => order?.charged === true)
    || charges.some((charge) => isChargeActive(charge) && isChargePaid(charge))
  return !paid
}

function rowWaitDispense(row) {
  const charges = Array.isArray(row?.charges) ? row.charges : []
  const dispenses = Array.isArray(row?.dispenses) ? row.dispenses : []
  const paid = charges.some((charge) => isChargeActive(charge) && isChargePaid(charge))
  if (!paid) return false
  const done = dispenses.some((slip) => {
    const status = text(slip?.status).toUpperCase()
    if (status === 'DONE') return true
    const lines = Array.isArray(slip?.items) ? slip.items : Array.isArray(slip?.lines) ? slip.lines : []
    if (!lines.length) return status === 'DONE'
    return lines.every((line) => text(line?.lineDispenseStatus || line?.dispenseStatus).toUpperCase() === 'FULL')
  })
  return !done
}

export function buildOutpatientQueueRows({ registers, orders, charges, dispenses } = {}) {
  const registerList = Array.isArray(registers) ? registers : []
  const orderList = Array.isArray(orders) ? orders : []
  const chargeList = Array.isArray(charges) ? charges : []
  const dispenseList = Array.isArray(dispenses) ? dispenses : []

  return registerList.map((register) => {
    const relatedOrders = relatedByRegister(orderList, register?.registerId)
    const relatedCharges = relatedByRegister(chargeList, register?.registerId)
    const relatedDispenses = relatedByRegister(dispenseList, register?.registerId)
    const row = {
      ...register,
      orders: relatedOrders,
      charges: relatedCharges,
      dispenses: relatedDispenses,
    }
    row.flags = {
      waitOrder: register?.status === 'DIAGNOSED',
      waitCharge: rowWaitCharge(row),
      waitDispense: rowWaitDispense(row),
    }
    return row
  })
}

/**
 * @param {Array} rows
 * @param {'waiting'|'received'|'diagnosing'|'order'|'charge'|'dispense'|'paused'|'back'|'all'} tab
 * @param {string} query
 */
export function filterOutpatientQueue(rows, tab = 'waiting', query = '') {
  const source = Array.isArray(rows) ? rows : []
  const needle = text(query).toLowerCase()
  return source.filter((row) => {
    if (!row) return false
    if (needle && !searchable(row).includes(needle)) return false
    const status = text(row.status)
    switch (tab) {
      case 'waiting':
        return status === 'NORMAL'
      case 'received':
        return status === 'USED'
      case 'diagnosing':
        return status === 'DIAGNOSING'
      case 'order':
        // H02 §6.3 / H03: 「待开单」primary anchor DIAGNOSED
        return status === 'DIAGNOSED'
      case 'charge':
        return rowWaitCharge(row)
      case 'dispense':
        return rowWaitDispense(row)
      case 'paused':
        return status === 'PAUSED'
      case 'back':
        return status === 'BACK'
      case 'all':
        return true
      default:
        return status === 'NORMAL'
    }
  })
}

function captureError(domain, reason) {
  const err = reason && typeof reason === 'object' ? reason : new Error(String(reason || 'load failed'))
  return {
    domain,
    status: typeof err.status === 'number' ? err.status : 0,
    code: text(err.code),
    message: text(err.message) || '加载失败',
  }
}

export async function loadOutpatientQueue(apis = {}, token = '') {
  const visitDate = text(apis.visitDate)
  const listRegisters = apis.listRegisters
  const listOrders = apis.listOrders
  const listCharges = apis.listCharges
  const listDispenses = apis.listDispenses
  if (typeof listRegisters !== 'function' || typeof listOrders !== 'function'
    || typeof listCharges !== 'function' || typeof listDispenses !== 'function') {
    throw new Error('loadOutpatientQueue requires listRegisters/listOrders/listCharges/listDispenses')
  }

  const registerQuery = visitDate ? { visitDate } : {}
  const settled = await Promise.allSettled([
    listRegisters(token, registerQuery),
    listOrders(token, visitDate ? { visitDate } : {}),
    listCharges(token, {}),
    listDispenses(token, {}),
  ])

  const bags = { registers: [], orders: [], charges: [], dispenses: [] }
  const errors = []
  const domains = ['registers', 'orders', 'charges', 'dispenses']
  settled.forEach((result, index) => {
    const domain = domains[index]
    if (result.status === 'fulfilled') {
      const value = result.value
      bags[domain] = Array.isArray(value?.items) ? value.items : Array.isArray(value) ? value : []
      return
    }
    errors.push(captureError(domain, result.reason))
  })

  return {
    ...bags,
    rows: buildOutpatientQueueRows(bags),
    errors,
  }
}

export async function selectQueueRegister(register, apis = {}, token = '') {
  if (!register || !text(register.patientId)) return false
  if (typeof apis.fetchPatient !== 'function' || typeof apis.selectOutpatientContext !== 'function') {
    throw new Error('selectQueueRegister requires fetchPatient and selectOutpatientContext')
  }
  const patient = await apis.fetchPatient(token, register.patientId)
  if (!patient || !text(patient.patientId)) return false
  return Boolean(await apis.selectOutpatientContext({ patient, register }))
}

export const OUTPATIENT_QUEUE_TABS = Object.freeze([
  { id: 'waiting', label: '候诊' },
  { id: 'received', label: '已接诊' },
  { id: 'diagnosing', label: '诊疗中' },
  { id: 'order', label: '待开单' },
  { id: 'charge', label: '待收费' },
  { id: 'dispense', label: '待发药' },
  { id: 'paused', label: '暂停' },
  { id: 'back', label: '退号' },
])
