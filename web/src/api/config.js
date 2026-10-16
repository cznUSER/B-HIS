// 2026-10-12 P1-H05 登录界面
// 2026-10-12 P1-H05 真 API 联调：浏览器走 Vite 注入的 VITE_*；node 脚本可读 process.env。
// 2026-10-12 demo 构建固定走真实 API，供 Windows exe 同源托管

function trimSlash(value) {
  return value.replace(/\/+$/, '')
}

function nodeValue(name) {
  if (typeof process === 'undefined' || !process.env) return undefined
  const value = process.env[name]
  if (value == null) return undefined
  const text = String(value).trim()
  return text || undefined
}

function viteValues() {
  try {
    if (!import.meta.env) return {}
    return {
      VITE_USE_MOCK: import.meta.env.VITE_USE_MOCK,
      VITE_API_BASE_URL: import.meta.env.VITE_API_BASE_URL,
    }
  } catch {
    return {}
  }
}

function configured(name, fallback) {
  const fromVite = viteValues()[name]
  if (fromVite != null && String(fromVite).trim() !== '') return String(fromVite).trim()
  return nodeValue(name) ?? fallback
}

export function useMock() {
  // Product builds always use HTTP. The opt-in test flag exists only for the
  // repository's isolated contract scripts and is never committed to web/.env.
  if (typeof process !== 'undefined' && process.env?.BHIS_WEB_TEST_MODE === '1') {
    const flag = configured('VITE_USE_MOCK', 'false').toLowerCase()
    return flag !== 'false' && flag !== '0' && flag !== 'no'
  }
  return false
}

export function apiBaseUrl() {
  return trimSlash(configured('VITE_API_BASE_URL', '/api/v1'))
}

export function apiUrl(path) {
  const suffix = path.startsWith('/') ? path : `/${path}`
  return `${apiBaseUrl()}${suffix}`
}
