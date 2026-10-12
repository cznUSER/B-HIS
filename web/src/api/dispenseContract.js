// 2026-10-12 P1-H05-DISPENSE-UI 发药 Mock 字段与文案。对齐拍板 #23，不调用 Go。

export const PHARMACY = {
  deptId: 'dept-pharmacy',
  deptName: '门诊药房',
}

export const LINE_STATUS_LABEL = {
  NONE: '未发',
  PARTIAL: '部分',
  FULL: '已发',
}

export function round2(value) {
  const number = Number(value)
  if (!Number.isFinite(number)) return 0
  return Math.round((number + Number.EPSILON) * 100) / 100
}

export function formatQty(value) {
  const number = round2(value)
  if (Number.isInteger(number)) return String(number)
  return number.toFixed(2)
}

export function lineDispenseStatus(chargedQty, dispensedNet) {
  const charged = round2(chargedQty)
  const net = round2(dispensedNet)
  if (!(net > 0)) return 'NONE'
  if (net + 1e-9 < charged) return 'PARTIAL'
  return 'FULL'
}

export function lineStatusLabel(status) {
  return LINE_STATUS_LABEL[status] || '未发'
}

export function stockHint(stockQty, dispenseQty) {
  const stock = round2(stockQty)
  const qty = round2(dispenseQty)
  if (!(stock > 0)) return '库存不足，无法发药'
  if (qty > stock) return `库存 ${formatQty(stock)}，不足本次 ${formatQty(qty)}`
  return `库存 ${formatQty(stock)}`
}

export function stockIsShort(stockQty, dispenseQty) {
  return !(round2(stockQty) > 0) || round2(dispenseQty) > round2(stockQty)
}

export function formatOperateAt(iso) {
  const date = new Date(iso)
  if (Number.isNaN(date.getTime())) return typeof iso === 'string' && iso ? iso : '—'
  const parts = new Intl.DateTimeFormat('en-CA', {
    timeZone: 'Asia/Shanghai',
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    hourCycle: 'h23',
  }).formatToParts(date)
  const pick = (type) => parts.find((part) => part.type === type)?.value || ''
  return `${pick('year')}-${pick('month')}-${pick('day')} ${pick('hour')}:${pick('minute')}`
}
