// 2026-10-12 P1-H05-HOME-DASH 首页占位。数字只有 0；扇区等分，不表示占比。

export const PLACEHOLDER_VALUE = 0

export const PERIODS = ['今天', '昨天', '本周', '本月']

export const TODO_CARDS = [
  { key: 'charge', label: '待收费', value: 0, module: 'charge' },
  { key: 'dispense', label: '待发药', value: 0, module: 'dispense' },
  { key: 'upload', label: '进销存未上传', value: null, phase: '二期' },
  { key: 'standard', label: '未贯标提醒', value: null, phase: '二期' },
  { key: 'stock', label: '库存预警', value: 0, phase: '占位' },
]

export const CHARGE_METRICS = ['营业收入', '现金', '微信', '支付宝', '银行卡', '医保', '减免']

export const VISIT_METRICS = ['营业收入', '门诊客流量', '门诊客单价']

export const FEE_SLICES = [
  { key: 'lab', label: '检验', color: '#3cba6e' },
  { key: 'exam', label: '检查', color: '#f2c14b' },
  { key: 'west', label: '西成药', color: '#ee8d3a' },
  { key: 'herb', label: '中草药', color: '#e15b5b' },
  { key: 'treat', label: '治疗及其他', color: '#5eb4e8' },
  { key: 'material', label: '卫材', color: '#3a78c2' },
]

export const PAY_SLICES = [
  { key: 'cash', label: '现金', color: '#3a78c2' },
  { key: 'wechat', label: '微信', color: '#3cba6e' },
  { key: 'alipay', label: '支付宝', color: '#f2c14b' },
  { key: 'card', label: '银行卡', color: '#e15b5b' },
  { key: 'insure', label: '医保', color: '#5eb4e8' },
  { key: 'waiver', label: '减免', color: '#2f9a86' },
]

// 圆周按 100 计。等分只为摆出环，间隙不编码任何业务量。
export function equalArcs(count) {
  const gap = 1.15
  const span = 100 / count
  return Array.from({ length: count }, (_, index) => ({
    dash: `${(span - gap).toFixed(3)} ${(100 - (span - gap)).toFixed(3)}`,
    offset: (-index * span).toFixed(3),
  }))
}
