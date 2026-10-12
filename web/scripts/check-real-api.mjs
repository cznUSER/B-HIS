// 2026-10-12 P1-H05 真 API 联调
// 打可配置的真 API：登录成功、错误口令、退出 204。不打印 token 或口令。
// VITE_USE_MOCK 不是 false 时直接跳过（退出 0）。不要放进 CI。

import { readFile } from 'node:fs/promises'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const webRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const WANTED = ['VITE_USE_MOCK', 'VITE_API_BASE_URL', 'VITE_API_PROXY_TARGET']

function parseEnv(text) {
  const out = {}
  for (const rawLine of text.split('\n')) {
    let line = rawLine.trim().replace(/\r$/, '')
    if (!line || line.startsWith('#')) continue
    if (line.startsWith('export ')) line = line.slice(7).trim()
    const eq = line.indexOf('=')
    if (eq <= 0) continue
    const key = line.slice(0, eq).trim()
    if (!WANTED.includes(key)) continue
    let value = line.slice(eq + 1).trim()
    if (
      value.length >= 2 &&
      ((value.startsWith('"') && value.endsWith('"')) || (value.startsWith("'") && value.endsWith("'")))
    ) {
      value = value.slice(1, -1)
    }
    out[key] = value
  }
  return out
}

async function readOptional(name) {
  try {
    return await readFile(path.join(webRoot, name), 'utf8')
  } catch (error) {
    if (error && error.code === 'ENOENT') return ''
    throw error
  }
}

function applyFileEnv() {
  return Promise.all([readOptional('.env.local'), readOptional('.env')]).then(([localText, envText]) => {
    const merged = { ...parseEnv(envText), ...parseEnv(localText) }
    for (const key of WANTED) {
      if (process.env[key] == null || String(process.env[key]).trim() === '') {
        if (merged[key] != null) process.env[key] = merged[key]
      }
    }
  })
}

function mockOff() {
  const flag = String(process.env.VITE_USE_MOCK ?? 'true').trim().toLowerCase()
  return flag === 'false' || flag === '0' || flag === 'no'
}

function resolveBase() {
  const configured = String(process.env.VITE_API_BASE_URL || '/api/v1').trim()
  if (/^https?:\/\//i.test(configured)) return configured.replace(/\/+$/, '')
  const proxy = String(process.env.VITE_API_PROXY_TARGET || 'http://127.0.0.1:8080').trim().replace(/\/+$/, '')
  const suffix = configured.startsWith('/') ? configured : `/${configured}`
  return `${proxy}${suffix}`.replace(/\/+$/, '')
}

await applyFileEnv()

if (!mockOff()) {
  console.log('check:api skipped (set VITE_USE_MOCK=false to call a real API; CI does not run this)')
  process.exit(0)
}

process.env.VITE_USE_MOCK = 'false'
process.env.VITE_API_BASE_URL = resolveBase()
const base = process.env.VITE_API_BASE_URL

const { login, logout, fetchMe } = await import('../src/api/authApi.js')
const { ApiError } = await import('../src/api/errors.js')

function assert(condition, message) {
  if (!condition) throw new Error(message)
}

async function expectError(fn, code, status) {
  try {
    await fn()
  } catch (error) {
    if (error instanceof ApiError && error.code === 'HTTP_ERROR' && error.status === 0) {
      throw new Error(`API unreachable at ${base}`)
    }
    assert(error instanceof ApiError, 'expected ApiError')
    assert(error.code === code, `expected code ${code}, got ${error.code}`)
    assert(error.status === status, `expected status ${status}, got ${error.status}`)
    return error
  }
  throw new Error(`expected ${code}`)
}

// 口令与仓库根 README「内存演示账号」相同，只放在请求体里。
const demoUser = 'system'
const demoPassword = '1'

try {
  const ok = await login(demoUser, demoPassword)
  assert(ok.tokenType === 'Bearer', 'tokenType')
  assert(typeof ok.accessToken === 'string' && ok.accessToken.length > 0, 'accessToken')
  assert(!ok.accessToken.endsWith('.mock'), 'mock token was sent to the real API')
  assert(ok.expiresIn > 0, 'expiresIn')
  assert(ok.user.username === demoUser, 'username')
  assert(ok.user.displayName === '系统管理员', 'displayName')
  assert(Array.isArray(ok.user.roles) && ok.user.roles.includes('SYSTEM'), 'roles')
  console.log(`login 200 user=${ok.user.username} expiresIn=${ok.expiresIn}`)

  const bad = await expectError(() => login(demoUser, 'wrong-password'), 'INVALID_CREDENTIALS', 401)
  assert(bad.message === '用户名或密码错误', 'failure message')
  console.log(`login 401 code=${bad.code}`)

  const disabled = await expectError(() => login('disabled', demoPassword), 'ACCOUNT_DISABLED', 403)
  console.log(`login 403 code=${disabled.code}`)

  await logout(ok.accessToken)
  console.log('logout 204')

  const revoked = await expectError(() => fetchMe(ok.accessToken), 'TOKEN_INVALID', 401)
  console.log(`me 401 code=${revoked.code}`)
  console.log(`real API checks passed (${base})`)
} catch (error) {
  if (error instanceof ApiError && error.status === 0) {
    console.error(`API unreachable at ${base}`)
  } else if (error instanceof ApiError) {
    console.error(`unexpected status=${error.status} code=${error.code}`)
  } else {
    console.error(error instanceof Error ? error.message : String(error))
  }
  process.exit(1)
}
