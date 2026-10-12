// 2026-10-12 P1-H05-CHARGE-UI 收费金额用分，避免浮点累加。
// 2026-10-12 真接口金额沿用服务端 squareFee，界面仍按分展示。

import { apiError } from './errors.js'
import { findCatalogItem } from './orderContract.js'

export const PAY_CASH = 'CASH'
export const RESERVED_PAY_TYPES = [
  { value: 'WECHAT', label: '微信' },
  { value: 'ALIPAY', label: '支付宝' },
]

export function yuanToCents(value) {
  const text = typeof value === 'number' ? value.toFixed(2) : String(value ?? '').trim()
  if (!/^\d+\.\d{2}$/.test(text)) {
    throw apiError(400, 'INVALID_AMOUNT', '数量或金额不合法')
  }
  const [yuan, frac] = text.split('.')
  return Number(yuan) * 100 + Number(frac)
}

export function formatYuan(cents) {
  const amount = Number(cents)
  if (!Number.isInteger(amount)) return '—'
  const sign = amount < 0 ? '-' : ''
  const abs = Math.abs(amount)
  const yuan = Math.floor(abs / 100)
  const frac = String(abs % 100).padStart(2, '0')
  return `${sign}¥${yuan}.${frac}`
}

export function moneyCents(value) {
  const number = typeof value === 'number' ? value : Number(String(value ?? '').trim())
  if (!Number.isFinite(number)) throw apiError(400, 'INVALID_AMOUNT', '数量或金额不合法')
  return Math.round(number * 100)
}

export function lineStock(itemNo, qty) {
  const catalog = findCatalogItem(itemNo)
  const amount = Number(qty)
  if (!catalog || catalog.stockQty == null || !Number.isFinite(amount)) {
    return { sufficient: true, available: null, hint: '不占库存' }
  }
  if (amount > catalog.stockQty) {
    return {
      sufficient: false,
      available: catalog.stockQty,
      hint: `库存不足（可用 ${catalog.stockQty}，需要 ${amount}）`,
    }
  }
  return { sufficient: true, available: catalog.stockQty, hint: `可用 ${catalog.stockQty}` }
}

export function formatOperate(iso) {
  if (typeof iso !== 'string' || !iso) return '—'
  const date = new Date(iso)
  if (Number.isNaN(date.getTime())) return '—'
  const parts = new Intl.DateTimeFormat('en-CA', {
    timeZone: 'Asia/Shanghai',
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    hourCycle: 'h23',
  }).formatToParts(date)
  const pick = (type) => {
    const found = parts.find((part) => part.type === type)
    return found ? found.value : ''
  }
  return `${pick('year')}-${pick('month')}-${pick('day')} ${pick('hour')}:${pick('minute')}`
}

export function statusLabel(status) {
  if (status === 'NORMAL') return '有效'
  if (status === 'ABATE') return '已作废'
  return status || '—'
}
