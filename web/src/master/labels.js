// 2026-10-13 P1-H05-MASTER-UI 基础数据页展示文案。

export function statusLabel(status) {
  if (status === 'ACTIVE') return '启用'
  if (status === 'INACTIVE' || status === 'DISABLED') return '停用'
  return status || '—'
}

export function typeLabel(itemType) {
  if (itemType === 'DRUG') return '药品'
  if (itemType === 'TREAT') return '治疗'
  return itemType || '—'
}

export function bizLabel(bizType) {
  if (bizType === 'ADJUST') return '调整'
  if (bizType === 'DISPENSE') return '发药'
  if (bizType === 'RETURN') return '退药'
  return bizType || '—'
}

export function formatMoney(value) {
  const number = Number(value)
  if (!Number.isFinite(number)) return '—'
  return number.toFixed(2)
}

export function formatQty(value) {
  const number = Number(value)
  if (!Number.isFinite(number)) return '—'
  return String(Math.round((number + Number.EPSILON) * 1000) / 1000)
}

export function formatTime(iso) {
  if (!iso) return '—'
  const date = new Date(iso)
  if (Number.isNaN(date.getTime())) return String(iso)
  const pad = (value) => String(value).padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}`
}

export const ENUM_LABELS = {
  M: '男',
  F: '女',
  U: '未知',
  ID_CARD: '身份证',
  CASH: '现金',
  WECHAT: '微信',
  ALIPAY: '支付宝',
  OTHER_E: '其他电子',
  OTHER: '其他',
  SELF: '自费',
}

export function enumLabel(code) {
  return ENUM_LABELS[code] || code
}
