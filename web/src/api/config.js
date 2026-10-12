// 2026-10-12 P1-H05 登录界面
// 2026-10-12 demo 构建固定走真实 API，供 Windows exe 同源托管

function trimSlash(value) {
  return value.replace(/\/+$/, '')
}

export function useMock() {
  if (import.meta.env.MODE === 'demo') return false
  const flag = String(import.meta.env.VITE_USE_MOCK ?? 'true').trim().toLowerCase()
  return flag !== 'false' && flag !== '0' && flag !== 'no'
}

export function apiBaseUrl() {
  const configured = import.meta.env.VITE_API_BASE_URL
  const base = typeof configured === 'string' && configured.trim() ? configured.trim() : '/api/v1'
  return trimSlash(base)
}

export function apiUrl(path) {
  const suffix = path.startsWith('/') ? path : `/${path}`
  return `${apiBaseUrl()}${suffix}`
}
