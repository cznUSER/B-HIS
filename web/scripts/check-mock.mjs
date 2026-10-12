// 2026-10-12 P1-H05 登录界面
import { ApiError, readErrorResponse } from '../src/api/errors.js'
import { mockLogin, mockLogout, mockMe, resetMockAuth } from '../src/api/mockAuth.js'

function assert(condition, message) {
  if (!condition) throw new Error(message)
}

async function expectError(fn, code, status) {
  try {
    await fn()
  } catch (error) {
    assert(error instanceof ApiError, 'expected ApiError')
    assert(error.code === code, `expected code ${code}, got ${error.code}`)
    assert(error.status === status, `expected status ${status}, got ${error.status}`)
    assert(!JSON.stringify(error).includes('admin123'), 'error leaked password')
    return error
  }
  throw new Error(`expected ${code}`)
}

resetMockAuth()

const ok = mockLogin({ username: 'admin', password: 'admin123' })
assert(ok.tokenType === 'Bearer', 'tokenType')
assert(typeof ok.accessToken === 'string' && ok.accessToken.endsWith('.mock'), 'mock jwt')
assert(ok.expiresIn === 28800, 'expiresIn')
assert(ok.user.username === 'admin', 'username')
assert(ok.user.displayName === '系统管理员', 'displayName')
assert(Array.isArray(ok.user.roles) && ok.user.roles.includes('SYSTEM'), 'roles')

const me = mockMe(ok.accessToken)
assert(me.displayName === '系统管理员', 'me displayName')
assert(me.id === ok.user.id, 'me id')

mockLogout(ok.accessToken)
await expectError(() => mockMe(ok.accessToken), 'TOKEN_INVALID', 401)
await expectError(() => mockLogout(ok.accessToken), 'TOKEN_INVALID', 401)

resetMockAuth()
await expectError(() => mockLogin({ username: 'admin', password: 'wrong' }), 'INVALID_CREDENTIALS', 401)
await expectError(() => mockLogin({ username: 'nobody', password: 'x' }), 'INVALID_CREDENTIALS', 401)
await expectError(
  () => mockLogin({ username: 'disabled', password: 'disabled123' }),
  'ACCOUNT_DISABLED',
  403,
)
await expectError(() => mockLogin({ username: '', password: 'x' }), 'VALIDATION_ERROR', 400)

resetMockAuth()
for (let i = 0; i < 5; i += 1) {
  await expectError(() => mockLogin({ username: 'admin', password: 'bad' }), 'INVALID_CREDENTIALS', 401)
}
await expectError(() => mockLogin({ username: 'admin', password: 'admin123' }), 'AUTH_RATE_LIMITED', 429)

const envelope = new Response(
  JSON.stringify({
    success: false,
    data: null,
    error: { code: 'INVALID_CREDENTIALS', message: '用户名或密码错误', details: {} },
    requestId: 'req-1',
  }),
  { status: 401, headers: { 'Content-Type': 'application/json' } },
)
const parsed = await readErrorResponse(envelope)
assert(parsed.code === 'INVALID_CREDENTIALS', 'envelope code')
assert(parsed.requestId === 'req-1', 'requestId')
assert(parsed.message === '用户名或密码错误', 'envelope message')

const plain = new Response('Unauthorized', { status: 401 })
const plainError = await readErrorResponse(plain)
assert(plainError.code === 'UNAUTHORIZED', 'plain http code')
assert(plainError.status === 401, 'plain http status')

console.log('mock auth checks passed')
