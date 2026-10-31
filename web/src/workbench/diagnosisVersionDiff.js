/** P1-DIAG-VERSION-DIFF-01: pure read-only diagnosis version field diff (形态 A · 零网络 · 零 storage). */

export const STRUCTURED_DIFF_KEYS = Object.freeze([
  'chiefComplaint',
  'presentHistory',
  'pastHistory',
  'examination',
  'assessment',
  'plan',
])

export const STRUCTURED_DIFF_LABELS = Object.freeze({
  chiefComplaint: '主诉',
  presentHistory: '现病史',
  pastHistory: '既往史 / 过敏史',
  examination: '查体与检查',
  assessment: '初步诊断',
  plan: '处置计划',
})

export const META_DIFF_KEYS = Object.freeze([
  'templateCode',
  'status',
  'version',
  'revisionReason',
])

export const META_DIFF_LABELS = Object.freeze({
  templateCode: '模板码',
  status: '状态',
  version: '版本号',
  revisionReason: '修订原因',
})

export const DIFF_NEED_TWO_HINT = '至少需要 2 条记录才能对比'
export const DIFF_NO_CHANGES_HINT = '无字段差异'

export function text(value) {
  return value == null ? '' : String(value)
}

/**
 * @param {Record<string, any>|null|undefined} row
 */
export function pickStructured(row = {}) {
  const source = row && typeof row === 'object' ? row : {}
  const structuredIn = source.structured && typeof source.structured === 'object' ? source.structured : {}
  const structured = {}
  for (const key of STRUCTURED_DIFF_KEYS) {
    structured[key] = text(structuredIn[key]).trim()
  }
  const versionRaw = source.version
  const version = versionRaw == null || versionRaw === '' ? '1' : text(versionRaw)
  return {
    diagnoseId: text(source.diagnoseId),
    content: text(source.content),
    templateCode: text(source.templateCode) || 'GENERAL',
    status: text(source.status),
    version,
    revisionReason: text(source.revisionReason),
    structured,
  }
}

/**
 * @param {Record<string, any>|null|undefined} a
 * @param {Record<string, any>|null|undefined} b
 */
export function diffDiagnosisRecords(a, b) {
  const left = pickStructured(a)
  const right = pickStructured(b)
  const fields = []

  for (const key of STRUCTURED_DIFF_KEYS) {
    const l = text(left.structured[key])
    const r = text(right.structured[key])
    fields.push({
      key,
      label: STRUCTURED_DIFF_LABELS[key] || key,
      left: l,
      right: r,
      changed: l !== r,
      group: 'structured',
    })
  }

  for (const key of META_DIFF_KEYS) {
    const l = text(left[key])
    const r = text(right[key])
    fields.push({
      key,
      label: META_DIFF_LABELS[key] || key,
      left: l,
      right: r,
      changed: l !== r,
      group: 'meta',
    })
  }

  const changedCount = fields.filter((f) => f.changed).length
  return {
    fields,
    changedCount,
    identical: changedCount === 0,
  }
}

export function canOpenVersionDiff(diagnoses) {
  return Array.isArray(diagnoses) && diagnoses.length >= 2
}

/**
 * Prefer preferred row vs its supersedesDiagnoseId target; else first amend-chain pair; else first two list rows.
 * @param {Array<Record<string, any>>} diagnoses
 * @param {string} [preferredId]
 */
export function defaultDiffPair(diagnoses, preferredId) {
  if (!canOpenVersionDiff(diagnoses)) return null
  const list = diagnoses
  const byId = new Map(list.map((row) => [text(row.diagnoseId), row]))
  const preferred = preferredId ? byId.get(text(preferredId)) : null
  if (preferred) {
    const priorId = text(preferred.supersedesDiagnoseId)
    if (priorId && byId.has(priorId)) {
      return { left: byId.get(priorId), right: preferred }
    }
  }
  for (const row of list) {
    const priorId = text(row.supersedesDiagnoseId)
    if (priorId && byId.has(priorId)) {
      return { left: byId.get(priorId), right: row }
    }
  }
  return { left: list[0], right: list[1] }
}
