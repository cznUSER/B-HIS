// Workbench summary labels and data-backed pending tasks.

export const PERIOD_OPTIONS = [
  { label: '今天', value: 'TODAY' },
  { label: '昨天', value: 'YESTERDAY' },
  { label: '本周', value: 'WEEK' },
  { label: '本月', value: 'MONTH' },
]

export const TODO_CARDS = [
  { key: 'charge', label: '待收费', valueKey: 'pendingChargeCount', module: 'charge' },
  { key: 'dispense', label: '待发药', valueKey: 'pendingDispenseCount', module: 'dispense' },
  { key: 'upload', label: '进销存未上传', value: null, phase: '二期' },
  { key: 'standard', label: '未贯标提醒', value: null, phase: '二期' },
  { key: 'stock', label: '库存预警', value: null, phase: '未配置' },
]

export const CHARGE_METRICS = [
  { label: '收费金额', key: 'revenueCents', format: 'money' },
  { label: '退费金额', key: 'refundCents', format: 'money' },
  { label: '净收入', key: 'netRevenueCents', format: 'money' },
  { label: '收费笔数', key: 'chargeCount', format: 'count' },
]

export const VISIT_METRICS = [
  { label: '门诊人次', key: 'encounterCount', format: 'count' },
  { label: '平均客单价', key: 'averageTicketCents', format: 'money' },
]
