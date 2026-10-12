// 2026-10-12 P1-H05 登录界面
// 2026-10-12 P1-H05 真 API 联调：浏览器走 Vite 注入的 VITE_*；node 脚本可读 process.env。

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
  const flag = configured('VITE_USE_MOCK', 'true').toLowerCase()
  return flag !== 'false' && flag !== '0' && flag !== 'no'
}

export function apiBaseUrl() {
  return trimSlash(configured('VITE_API_BASE_URL', '/api/v1'))
}

export function apiUrl(path) {
  const suffix = path.startsWith('/') ? path : `/${path}`
  return `${apiBaseUrl()}${suffix}`
}
