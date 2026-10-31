/** P1-DIAG-RESUME-01: pure frontend dirty snapshot helpers (zero network; never persist clinical text). */
export const DIRTY_HINT = '有未暂存修改'
export const LEAVE_CONFIRM_MESSAGE = '当前诊断有未暂存修改，离开将丢失未保存内容。确定离开吗？'
export const REGISTER_SWITCH_CONFIRM_MESSAGE = '当前诊断有未暂存修改，切换挂号将丢失未保存内容。确定切换吗？'
export const NEW_DRAFT_CONFIRM_MESSAGE = '当前诊断有未暂存修改，新建将清空编辑器。确定继续吗？'
export const LOCAL_SAVE_HINT_PREFIX = '本机上次暂存成功'
export const SERVER_SAVED_HINT = '已暂存（服务端）'

const STRUCTURED_KEYS = [
  'chiefComplaint',
  'presentHistory',
  'pastHistory',
  'examination',
  'assessment',
  'plan',
]

function text(value) {
  return value == null ? '' : String(value)
}

/**
 * @param {{ structured?: Record<string, string>, templateCode?: string, revisionReason?: string, amendSourceId?: string }} state
 */
export function snapshotOf(state = {}) {
  const structured = state.structured || {}
  const next = {
    templateCode: text(state.templateCode || 'GENERAL'),
    revisionReason: text(state.revisionReason),
    amendSourceId: text(state.amendSourceId),
    structured: {},
  }
  for (const key of STRUCTURED_KEYS) {
    next.structured[key] = text(structured[key]).trim()
  }
  return next
}

/**
 * @param {ReturnType<typeof snapshotOf>|null|undefined} baseline
 * @param {ReturnType<typeof snapshotOf>|null|undefined} current
 * @param {{ editorLocked?: boolean }} [opts]
 */
export function isDirty(baseline, current, opts = {}) {
  if (opts.editorLocked) return false
  const left = baseline || snapshotOf({})
  const right = current || snapshotOf({})
  if (left.templateCode !== right.templateCode) return true
  if (left.revisionReason !== right.revisionReason) return true
  if (left.amendSourceId !== right.amendSourceId) return true
  for (const key of STRUCTURED_KEYS) {
    if (text(left.structured?.[key]) !== text(right.structured?.[key])) return true
  }
  return false
}

/** Format a Date as HH:mm:ss for local session hint (not server clock). */
export function formatLocalSaveAt(date) {
  if (!(date instanceof Date) || Number.isNaN(date.getTime())) return ''
  const hh = String(date.getHours()).padStart(2, '0')
  const mm = String(date.getMinutes()).padStart(2, '0')
  const ss = String(date.getSeconds()).padStart(2, '0')
  return `${hh}:${mm}:${ss}`
}

export function localSaveHint(date) {
  const stamp = formatLocalSaveAt(date)
  return stamp ? `${LOCAL_SAVE_HINT_PREFIX} · ${stamp}` : ''
}

export function resumeSaveHint({ lastLocalSaveAt, diagnoseStatus, diagnoseId } = {}) {
  const local = localSaveHint(lastLocalSaveAt)
  if (local) return local
  if (diagnoseId && String(diagnoseStatus || '').toUpperCase() === 'SAVE') return SERVER_SAVED_HINT
  return ''
}
