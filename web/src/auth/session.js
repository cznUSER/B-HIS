// 2026-10-12 P1-H05 登录界面
// 访问令牌只放内存和 sessionStorage，刷新后仍可进入首页；退出时清除。

import { computed, reactive } from 'vue'

const STORAGE_KEY = 'bhis.session.v1'

export const session = reactive({
  accessToken: '',
  tokenType: 'Bearer',
  expiresIn: 0,
  expiresAt: 0,
  user: null,
})

export const isAuthenticated = computed(() => {
  return Boolean(session.accessToken) && session.expiresAt > Date.now()
})

function storage() {
  try {
    return typeof sessionStorage === 'undefined' ? null : sessionStorage
  } catch {
    return null
  }
}

function persist() {
  const store = storage()
  if (!store) return
  if (!session.accessToken) {
    store.removeItem(STORAGE_KEY)
    return
  }
  store.setItem(
    STORAGE_KEY,
    JSON.stringify({
      accessToken: session.accessToken,
      tokenType: session.tokenType,
      expiresIn: session.expiresIn,
      expiresAt: session.expiresAt,
      user: session.user,
    }),
  )
}

export function clearSession() {
  session.accessToken = ''
  session.tokenType = 'Bearer'
  session.expiresIn = 0
  session.expiresAt = 0
  session.user = null
  persist()
}

export function applyLogin(payload) {
  session.accessToken = payload.accessToken
  session.tokenType = payload.tokenType
  session.expiresIn = payload.expiresIn
  session.expiresAt = Date.now() + payload.expiresIn * 1000
  session.user = payload.user
  persist()
}

export function applyUser(user) {
  session.user = user
  persist()
}

export function restoreSession() {
  const store = storage()
  if (!store) return
  const raw = store.getItem(STORAGE_KEY)
  if (!raw) return
  try {
    const saved = JSON.parse(raw)
    if (!saved || typeof saved.accessToken !== 'string' || typeof saved.expiresAt !== 'number') {
      store.removeItem(STORAGE_KEY)
      return
    }
    if (saved.expiresAt <= Date.now()) {
      store.removeItem(STORAGE_KEY)
      return
    }
    session.accessToken = saved.accessToken
    session.tokenType = saved.tokenType === 'Bearer' ? 'Bearer' : 'Bearer'
    session.expiresIn = typeof saved.expiresIn === 'number' ? saved.expiresIn : 0
    session.expiresAt = saved.expiresAt
    session.user = saved.user || null
  } catch {
    store.removeItem(STORAGE_KEY)
  }
}
