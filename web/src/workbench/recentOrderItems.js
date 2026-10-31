/** P1-ORDER-RECENT-01: outpatient order recent itemNo MRU (形态 A).
 * 会话内存 · 非持久化 · 禁 web storage；零网络；非跨会话收藏。
 */
export const MAX_RECENT_ORDER_ITEMS = 8

function asItemNo(value) {
  return String(value ?? '').trim()
}

function normalizeMax(max) {
  if (max === undefined || max === null) return MAX_RECENT_ORDER_ITEMS
  const value = Number(max)
  if (!Number.isFinite(value)) return MAX_RECENT_ORDER_ITEMS
  return Math.max(0, Math.floor(value))
}

/**
 * Keep first-seen order (already MRU-front), drop empties/dupes, truncate.
 */
export function normalizeRecentItemNos(list, { max } = {}) {
  const limit = normalizeMax(max)
  if (!Array.isArray(list) || limit === 0) return []
  const seen = new Set()
  const result = []
  for (const value of list) {
    const itemNo = asItemNo(value)
    if (!itemNo || seen.has(itemNo)) continue
    seen.add(itemNo)
    result.push(itemNo)
    if (result.length >= limit) break
  }
  return result
}

/** Push itemNo to MRU front; dedupe + truncate. Empty itemNo is a no-op. */
export function pushRecentItemNo(list, itemNo, { max } = {}) {
  const key = asItemNo(itemNo)
  if (!key) return normalizeRecentItemNos(list, { max })
  return normalizeRecentItemNos([key, ...(Array.isArray(list) ? list : [])], { max })
}

let sessionRecentItemNos = []

export function getSessionRecentItemNos() {
  return sessionRecentItemNos.slice()
}

export function rememberSessionItemNo(itemNo) {
  sessionRecentItemNos = pushRecentItemNo(sessionRecentItemNos, itemNo)
  return getSessionRecentItemNos()
}

/** Test helper: clear in-memory session ring. */
export function clearSessionRecentItemNos() {
  sessionRecentItemNos = []
}

/**
 * Resolve recent chip against ACTIVE/selectable catalog.
 * Missing or non-ACTIVE → null (no ghost write).
 */
export function resolveRecentItem(itemNo, catalogItems) {
  const key = asItemNo(itemNo)
  if (!key) return null
  const list = Array.isArray(catalogItems) ? catalogItems : []
  const hit = list.find((item) => item && asItemNo(item.itemNo) === key)
  if (!hit) return null
  if (Object.prototype.hasOwnProperty.call(hit, 'status') && hit.status !== 'ACTIVE') return null
  return hit
}