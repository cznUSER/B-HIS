/** P1-DIAG-ICD-FIND-01: 门诊诊断页 · 演示 ICD 短表（Mock / DEMO）。
 * 诚实边界：本文件为 curated 演示码表（约 20–40 条），非完整官方 ICD-10 / 国临编码。
 * 零网络、零 REST、零 Dict；禁止将本表暗示为全量官方码表。
 */
export const ICD_DEMO_DISCLAIMER = '演示码表，非完整官方 ICD'

export const EMPTY_ICD_MATCH_MESSAGE = '无匹配演示码'

/** 默认空关键词时展示前 N 条，避免长列表占满编辑区 */
export const ICD_DEFAULT_PREVIEW_LIMIT = 12

/**
 * DEMO 短表：code + name；可选 pinyin/alias 仅供前端过滤。
 * 编码风格参考常见门诊书写习惯，均为演示用 DEMO 条目。
 */
export const MOCK_ICD_CODES = Object.freeze([
  Object.freeze({ code: 'J06.900', name: '急性上呼吸道感染', pinyin: 'jxsydgr', alias: '上感' }),
  Object.freeze({ code: 'J02.900', name: '急性咽炎', pinyin: 'jxyy', alias: '咽炎' }),
  Object.freeze({ code: 'J00.x00', name: '急性鼻咽炎', pinyin: 'jxbyy', alias: '感冒' }),
  Object.freeze({ code: 'J18.900', name: '肺炎', pinyin: 'fy', alias: '' }),
  Object.freeze({ code: 'J20.900', name: '急性支气管炎', pinyin: 'jxzqgy', alias: '支气' }),
  Object.freeze({ code: 'J45.900', name: '哮喘', pinyin: 'xc', alias: '' }),
  Object.freeze({ code: 'I10.x00', name: '原发性高血压', pinyin: 'yfxgxy', alias: '高血压' }),
  Object.freeze({ code: 'I25.101', name: '冠状动脉粥样硬化性心脏病', pinyin: 'gxbdmzyyhxxzb', alias: '冠心病' }),
  Object.freeze({ code: 'I48.x01', name: '心房颤动', pinyin: 'xfcd', alias: '房颤' }),
  Object.freeze({ code: 'E11.900', name: '2型糖尿病', pinyin: '2xtnb', alias: '糖尿病' }),
  Object.freeze({ code: 'E78.500', name: '高脂血症', pinyin: 'gzxz', alias: '血脂' }),
  Object.freeze({ code: 'E04.101', name: '非毒性甲状腺结节', pinyin: 'fdxjzxjj', alias: '甲结' }),
  Object.freeze({ code: 'K29.500', name: '慢性胃炎', pinyin: 'mxwy', alias: '胃炎' }),
  Object.freeze({ code: 'K21.000', name: '胃食管反流病', pinyin: 'wsgflb', alias: '反流' }),
  Object.freeze({ code: 'K80.200', name: '胆囊结石', pinyin: 'dnjs', alias: '胆石' }),
  Object.freeze({ code: 'K59.000', name: '便秘', pinyin: 'bm', alias: '' }),
  Object.freeze({ code: 'N39.000', name: '泌尿道感染', pinyin: 'mndgr', alias: '尿感' }),
  Object.freeze({ code: 'N40.x00', name: '前列腺增生', pinyin: 'qlxzs', alias: 'BPH' }),
  Object.freeze({ code: 'N20.000', name: '肾结石', pinyin: 'sjs', alias: '' }),
  Object.freeze({ code: 'M54.500', name: '腰痛', pinyin: 'yt', alias: '腰痛' }),
  Object.freeze({ code: 'M17.900', name: '膝关节病', pinyin: 'xgjb', alias: '膝骨关' }),
  Object.freeze({ code: 'M79.300', name: '脂膜炎', pinyin: 'zmy', alias: '' }),
  Object.freeze({ code: 'G43.900', name: '偏头痛', pinyin: 'ptt', alias: '偏头' }),
  Object.freeze({ code: 'G40.900', name: '癫痫', pinyin: 'dx', alias: '' }),
  Object.freeze({ code: 'F41.100', name: '广泛性焦虑障碍', pinyin: 'gfxjlza', alias: '焦虑' }),
  Object.freeze({ code: 'F32.900', name: '抑郁发作', pinyin: 'yyfz', alias: '抑郁' }),
  Object.freeze({ code: 'H10.900', name: '结膜炎', pinyin: 'jmy', alias: '红眼' }),
  Object.freeze({ code: 'H66.900', name: '中耳炎', pinyin: 'zey', alias: '' }),
  Object.freeze({ code: 'L30.900', name: '皮炎', pinyin: 'py', alias: '湿疹' }),
  Object.freeze({ code: 'L50.900', name: '荨麻疹', pinyin: 'xmz', alias: '风团' }),
  Object.freeze({ code: 'B34.900', name: '病毒感染', pinyin: 'bdgr', alias: '' }),
  Object.freeze({ code: 'A09.900', name: '感染性胃肠炎', pinyin: 'grxwcy', alias: '腹泻' }),
  Object.freeze({ code: 'R50.900', name: '发热', pinyin: 'fr', alias: '发烧' }),
  Object.freeze({ code: 'R51.x00', name: '头痛', pinyin: 'tt', alias: '' }),
  Object.freeze({ code: 'R05.x00', name: '咳嗽', pinyin: 'ks', alias: '' }),
  Object.freeze({ code: 'Z00.001', name: '健康查体', pinyin: 'jkct', alias: '体检' }),
])

/** 格式化为 assessment token：`{code} {name}` */
export function formatIcdAssessmentToken(code, name) {
  const c = String(code ?? '').trim()
  const n = String(name ?? '').trim()
  if (!c && !n) return ''
  if (!c) return n
  if (!n) return c
  return `${c} ${n}`
}

export function formatIcdEntryToken(entry) {
  if (!entry) return ''
  return formatIcdAssessmentToken(entry.code, entry.name)
}

export function normalizeIcdKeyword(raw) {
  return String(raw ?? '').trim()
}

/**
 * 按关键词过滤演示码表（code / name / pinyin / alias；大小写不敏感）。
 * keyword 空：返回前 `limit` 条预览；有关键词：返回全部匹配（不截断）。
 */
export function filterIcdByKeyword(entries, keyword = '', { limit = ICD_DEFAULT_PREVIEW_LIMIT } = {}) {
  const list = Array.isArray(entries) ? entries : []
  const q = normalizeIcdKeyword(keyword).toLowerCase()
  if (!q) {
    const n = Number.isFinite(limit) && limit > 0 ? Math.floor(limit) : ICD_DEFAULT_PREVIEW_LIMIT
    return list.slice(0, n)
  }
  return list.filter((item) => {
    if (!item) return false
    const hay = [item.code, item.name, item.pinyin, item.alias]
      .map((part) => String(part ?? '').toLowerCase())
      .join(' ')
    return hay.includes(q)
  })
}

/**
 * 将 ICD token 追加进 assessment；复用与常用诊断相同的「，」整段去重语义。
 */
export function appendIcdToken(current, tokenOrEntry) {
  const token =
    tokenOrEntry && typeof tokenOrEntry === 'object'
      ? formatIcdEntryToken(tokenOrEntry)
      : String(tokenOrEntry ?? '').trim()
  if (!token) return String(current ?? '').trim()
  const cur = String(current ?? '').trim()
  if (!cur) return token
  const parts = cur
    .split('，')
    .map((part) => part.trim())
    .filter(Boolean)
  if (parts.includes(token)) return parts.join('，')
  return [...parts, token].join('，')
}

export function assessmentIncludesIcdToken(current, tokenOrEntry) {
  const token =
    tokenOrEntry && typeof tokenOrEntry === 'object'
      ? formatIcdEntryToken(tokenOrEntry)
      : String(tokenOrEntry ?? '').trim()
  if (!token) return false
  return String(current ?? '')
    .split('，')
    .map((part) => part.trim())
    .filter(Boolean)
    .includes(token)
}
