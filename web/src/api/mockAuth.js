// 2026-10-12 P1-H05 登录界面
// 本地 Mock，响应形状与 OpenAPI 0.3.2 login / logout / me 一致。
// accessToken 只是演示用的 JWT 三段结构，签名固定为 mock，不是服务端 HS256。

import { apiError } from './errors.js'
import { assertLoginRequest, assertLoginResponse, assertUserDto } from './contract.js'

const REVOKED_KEY = 'bhis.revoked.v1'
const TOKEN_TTL_SECONDS = 28800
const FAILURE_LIMIT = 5

const ACCOUNTS = {
  admin: {
    password: 'admin123',
    disabled: false,
    user: {
      id: 'u-admin',
      username: 'admin',
      displayName: '系统管理员',
      roles: ['SYSTEM'],
    },
  },
  disabled: {
    password: 'disabled123',
    disabled: true,
    user: {
      id: 'u-disabled',
      username: 'disabled',
      displayName: '已停用账号',
      roles: ['REGISTRAR'],
    },
  },
}

const failureCount = new Map()
const memoryRevoked = new Set()

function base64url(value) {
  const json = typeof value === 'string' ? value : JSON.stringify(value)
  const bytes = new TextEncoder().encode(json)
  let binary = ''
  for (const byte of bytes) binary += String.fromCharCode(byte)
  return btoa(binary).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/g, '')
}

function decodeBase64url(segment) {
  const padded = segment.replace(/-/g, '+').replace(/_/g, '/') + '==='.slice((segment.length + 3) % 4)
  const binary = atob(padded)
  const bytes = Uint8Array.from(binary, (char) => char.charCodeAt(0))
  return JSON.parse(new TextDecoder().decode(bytes))
}

function readRevoked() {
  if (typeof sessionStorage === 'undefined') return new Set(memoryRevoked)
  try {
    const raw = sessionStorage.getItem(REVOKED_KEY)
    const list = raw ? JSON.parse(raw) : []
    return new Set(Array.isArray(list) ? list.filter((item) => typeof item === 'string') : [])
  } catch {
    return new Set()
  }
}

function writeRevoked(set) {
  if (typeof sessionStorage === 'undefined') {
    memoryRevoked.clear()
    for (const id of set) memoryRevoked.add(id)
    return
  }
  sessionStorage.setItem(REVOKED_KEY, JSON.stringify([...set]))
}

function issueToken(user) {
  const now = Math.floor(Date.now() / 1000)
  const payload = {
    sub: user.id,
    username: user.username,
    displayName: user.displayName,
    roles: user.roles,
    iat: now,
    exp: now + TOKEN_TTL_SECONDS,
    jti: crypto.randomUUID(),
  }
  const header = { alg: 'HS256', typ: 'JWT' }
  return `${base64url(header)}.${base64url(payload)}.mock`
}

function readToken(accessToken) {
  if (typeof accessToken !== 'string' || accessToken.split('.').length !== 3) {
    throw apiError(401, 'TOKEN_INVALID', '登录凭证无效，请重新登录')
  }
  const [headerSeg, payloadSeg, signature] = accessToken.split('.')
  if (signature !== 'mock') {
    throw apiError(401, 'TOKEN_INVALID', '登录凭证无效，请重新登录')
  }
  let header
  let payload
  try {
    header = decodeBase64url(headerSeg)
    payload = decodeBase64url(payloadSeg)
  } catch {
    throw apiError(401, 'TOKEN_INVALID', '登录凭证无效，请重新登录')
  }
  if (!header || header.typ !== 'JWT' || header.alg !== 'HS256') {
    throw apiError(401, 'TOKEN_INVALID', '登录凭证无效，请重新登录')
  }
  if (!payload || typeof payload.exp !== 'number' || typeof payload.jti !== 'string') {
    throw apiError(401, 'TOKEN_INVALID', '登录凭证无效，请重新登录')
  }
  if (payload.exp * 1000 <= Date.now()) {
    throw apiError(401, 'TOKEN_EXPIRED', '登录已过期，请重新登录')
  }
  if (readRevoked().has(payload.jti)) {
    throw apiError(401, 'TOKEN_INVALID', '登录凭证无效，请重新登录')
  }
  return payload
}

export function resetMockAuth() {
  failureCount.clear()
  memoryRevoked.clear()
  if (typeof sessionStorage !== 'undefined') sessionStorage.removeItem(REVOKED_KEY)
}

export function mockLogin(request) {
  const { username, password } = assertLoginRequest(request)
  const attempts = failureCount.get(username) || 0
  if (attempts >= FAILURE_LIMIT) {
    throw apiError(429, 'AUTH_RATE_LIMITED', '登录尝试过于频繁，请稍后再试', {
      requestId: `mock-${crypto.randomUUID()}`,
    })
  }

  const account = ACCOUNTS[username]
  if (!account || account.password !== password) {
    failureCount.set(username, attempts + 1)
    throw apiError(401, 'INVALID_CREDENTIALS', '用户名或密码错误', {
      requestId: `mock-${crypto.randomUUID()}`,
      details: {},
    })
  }
  if (account.disabled) {
    throw apiError(403, 'ACCOUNT_DISABLED', '账号已停用', {
      requestId: `mock-${crypto.randomUUID()}`,
    })
  }

  failureCount.delete(username)
  const user = assertUserDto(account.user)
  return assertLoginResponse({
    accessToken: issueToken(user),
    tokenType: 'Bearer',
    expiresIn: TOKEN_TTL_SECONDS,
    user,
  })
}

export function mockLogout(accessToken) {
  const payload = readToken(accessToken)
  const revoked = readRevoked()
  revoked.add(payload.jti)
  writeRevoked(revoked)
}

export function mockMe(accessToken) {
  const payload = readToken(accessToken)
  return assertUserDto({
    id: payload.sub,
    username: payload.username,
    displayName: payload.displayName,
    roles: payload.roles,
  })
}
