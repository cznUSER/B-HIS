/** P1-DIAG-COMMON-01: outpatient diagnosis quick-fill constants (形态 A · 零网络 · 禁 storage). */
export const COMMON_DIAGNOSES = Object.freeze([
  '上呼吸道感染',
  '急性咽炎',
  '高血压',
  '2型糖尿病',
  '胃炎',
])

/**
 * Append a diagnosis name into assessment text.
 * Empty -> write; non-empty -> join with Chinese comma and dedupe exact tokens.
 */
export function appendDiagnosisName(current, name) {
  const next = String(name ?? '').trim()
  if (!next) return String(current ?? '').trim()
  const cur = String(current ?? '').trim()
  if (!cur) return next
  const parts = cur
    .split('，')
    .map((part) => part.trim())
    .filter(Boolean)
  if (parts.includes(next)) return parts.join('，')
  return [...parts, next].join('，')
}

export function assessmentIncludesName(current, name) {
  const needle = String(name ?? '').trim()
  if (!needle) return false
  return String(current ?? '')
    .split('，')
    .map((part) => part.trim())
    .filter(Boolean)
    .includes(needle)
}
