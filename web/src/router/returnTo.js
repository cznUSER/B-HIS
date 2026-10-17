// 2026-10-14 P1-H05-WEB-USABILITY 深链登录回跳。

import { canEnter, findModule } from '../workbench/catalog.js'
import { canMaintainMaster } from '../master/access.js'
import { canOpenAccessRoute } from '../master/accessControlPolicy.js'

const STORAGE_KEY = 'bhis.returnTo.v1'

function store() {
  try {
    return typeof sessionStorage === 'undefined' ? null : sessionStorage
  } catch {
    return null
  }
}

export function safeReturnTo(value, user) {
  if (typeof value !== 'string' || !value.startsWith('/') || value.startsWith('//')) return ''
  if (/[\\\u0000-\u0020]/.test(value)) return ''
  const target = new URL(value, 'http://bhis.local')
  if (target.origin !== 'http://bhis.local') return ''
  const path = target.pathname
  if (path === '/app' || path === '/app/outpatient') return `${path}${target.search}${target.hash}`
  const module = path.match(/^\/app\/outpatient\/([^/]+)$/)
  if (module) {
    const entry = findModule(module[1])
    if (!entry || (user && !canEnter(entry, user))) return ''
    return `${path}${target.search}${target.hash}`
  }
  if (path === '/app/settings/master/users') {
    if (user && !canOpenAccessRoute(user, 'access-users')) return ''
    return `${path}${target.search}${target.hash}`
  }
  if (/^\/app\/settings\/master(?:\/(?:departments|doctors|catalog|inventory|dicts))?$/.test(path)) {
    if (user && !canMaintainMaster(user)) return ''
    return `${path}${target.search}${target.hash}`
  }
  const accessRoute = path.match(/^\/app\/settings\/users(?:\/(roles|permissions|logs))?$/)
  if (accessRoute) {
    if (user) {
      const routeName = accessRoute[1] === 'roles'
        ? 'access-roles'
        : accessRoute[1] === 'permissions'
          ? 'access-permissions'
          : accessRoute[1] === 'logs'
            ? 'access-user-logs'
            : 'access-users'
      if (!canOpenAccessRoute(user, routeName)) return ''
    }
    return `${path}${target.search}${target.hash}`
  }
  return ''
}

export function rememberReturnTo(value) {
  const path = safeReturnTo(value)
  const target = store()
  if (!target) return path
  if (path) target.setItem(STORAGE_KEY, path)
  else target.removeItem(STORAGE_KEY)
  return path
}

export function takeReturnTo() {
  const target = store()
  if (!target) return ''
  const path = safeReturnTo(target.getItem(STORAGE_KEY) || '')
  target.removeItem(STORAGE_KEY)
  return path
}
