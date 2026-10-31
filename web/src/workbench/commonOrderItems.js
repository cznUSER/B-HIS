/** P1-ORDER-COMMON-01: outpatient order quick-fill constants (形态 A · 零网络 · 无 storage).
 * 演示/产品冻结短表，非个人收藏；零网络；无 web storage。
 */
export const COMMON_ORDER_ITEMS = Object.freeze([
  Object.freeze({ itemNo: 'DRUG-A', label: '演示阿莫西林胶囊', itemType: 'DRUG' }),
  Object.freeze({ itemNo: 'DRUG-B', label: '演示布洛芬片', itemType: 'DRUG' }),
  Object.freeze({ itemNo: 'TREAT-1', label: '演示诊查费', itemType: 'TREAT' }),
])

function asItemNo(value) {
  return String(value ?? '').trim()
}

function findConstant(itemNo) {
  const key = asItemNo(itemNo)
  if (!key) return null
  return COMMON_ORDER_ITEMS.find((item) => item.itemNo === key) || null
}

/**
 * Resolve a common chip against an ACTIVE/selectable catalog list.
 * When catalogItems is omitted, returns the frozen constant entry (demo table only).
 * When provided, requires exact itemNo match; rows with status !== ACTIVE are rejected.
 */
export function resolveCommonItem(itemNo, catalogItems) {
  const key = asItemNo(itemNo)
  if (!key) return null
  if (catalogItems === undefined || catalogItems === null) {
    return findConstant(key)
  }
  const list = Array.isArray(catalogItems) ? catalogItems : []
  const hit = list.find((item) => item && asItemNo(item.itemNo) === key)
  if (!hit) return null
  if (Object.prototype.hasOwnProperty.call(hit, 'status') && hit.status !== 'ACTIVE') return null
  return hit
}

export function isCommonItemResolvable(itemNo, catalogItems) {
  return Boolean(resolveCommonItem(itemNo, catalogItems))
}

/**
 * Fill the first blank line itemNo; if none, append via addBlankLine then fill.
 * Does not auto-save/commit; leaves amount/usage/etc untouched.
 */
export function applyCommonItemToLines(lines, item, { addBlankLine } = {}) {
  const itemNo = asItemNo(item?.itemNo)
  const list = Array.isArray(lines) ? lines.map((line) => ({ ...line })) : []
  if (!itemNo) return list
  let idx = list.findIndex((line) => !asItemNo(line?.itemNo))
  if (idx < 0) {
    const blank =
      typeof addBlankLine === 'function'
        ? addBlankLine()
        : { key: `blank-${list.length + 1}`, itemNo: '', amount: '', usageText: '', frequencyText: '', dosage: '', dosageUnit: '' }
    list.push({ ...blank })
    idx = list.length - 1
  }
  list[idx] = { ...list[idx], itemNo }
  return list
}
