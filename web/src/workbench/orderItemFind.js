/** P1-ORDER-FIND-01: 开单页找项 · 纯前端分类 + 关键词过滤（零网络） */
export const FIND_CATEGORIES = Object.freeze([
  { id: 'DRUG', label: '药品' },
  { id: 'TREAT', label: '治疗' },
]);

export const DEFAULT_FIND_CATEGORY = 'DRUG';

export const EMPTY_MATCH_MESSAGE = '无匹配项目';

export function categoryLabel(itemType) {
  const hit = FIND_CATEGORIES.find((item) => item.id === itemType);
  return hit ? hit.label : String(itemType || '');
}

/** 规范化关键词：trim；空串视为无过滤 */
export function normalizeKeyword(raw) {
  return String(raw ?? '').trim();
}

/**
 * 按当前分类 + 关键词过滤目录项。
 * 匹配字段：itemNo / itemName / spec（大小写不敏感；前后空白忽略）。
 * keyword 空则返回该分类全量（调用方已筛 ACTIVE）。
 */
export function filterByKeyword(items, { category = DEFAULT_FIND_CATEGORY, keyword = '' } = {}) {
  const list = Array.isArray(items) ? items : [];
  const cat = category === 'TREAT' ? 'TREAT' : 'DRUG';
  const q = normalizeKeyword(keyword).toLowerCase();
  const typed = list.filter((item) => item && item.itemType === cat);
  if (!q) return typed;
  return typed.filter((item) => {
    const hay = [item.itemNo, item.itemName, item.spec]
      .map((part) => String(part ?? '').toLowerCase())
      .join(' ');
    return hay.includes(q);
  });
}

export function findSelectedLabel(items, itemNo) {
  const key = String(itemNo ?? '').trim();
  if (!key) return '';
  const hit = (Array.isArray(items) ? items : []).find((item) => item && item.itemNo === key);
  return hit ? hit.itemName || key : key;
}