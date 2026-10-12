// 2026-10-12 P1-H05 真 API 联调
// 本地假 HTTP，形状对齐 Go：LoginResponse、ErrorBody（含 data: null）、logout 204。
// 不访问本机 Go API，可随 npm run check 在 CI 里跑。

import http from 'node:http'

function assert(condition, message) {
  if (!condition) throw new Error(message)
}

const server = http.createServer(async (req, res) => {
  const url = new URL(req.url || '/', 'http://127.0.0.1')
  if (req.method === 'POST' && url.pathname === '/api/v1/auth/login') {
    const chunks = []
    for await (const chunk of req) chunks.push(chunk)
    const body = JSON.parse(Buffer.concat(chunks).toString() || '{}')
    if (body.password === 'good') {
      res.writeHead(200, { 'Content-Type': 'application/json; charset=utf-8' })
      res.end(JSON.stringify({
        accessToken: 'header.payload.sig',
        tokenType: 'Bearer',
        expiresIn: 28800,
        user: {
          id: '00000000-0000-4000-8000-000000000001',
          username: body.username,
          displayName: '系统管理员',
          roles: ['SYSTEM'],
        },
      }))
      return
    }
    res.writeHead(401, { 'Content-Type': 'application/json; charset=utf-8' })
    res.end(JSON.stringify({
      success: false,
      data: null,
      error: { code: 'INVALID_CREDENTIALS', message: '用户名或密码错误', details: {} },
      requestId: 'req-fixture',
    }))
    return
  }

  if (req.method === 'POST' && url.pathname === '/api/v1/auth/logout') {
    const auth = req.headers.authorization || ''
    if (!auth.startsWith('Bearer ')) {
      res.writeHead(401, { 'Content-Type': 'application/json; charset=utf-8' })
      res.end(JSON.stringify({
        success: false,
        data: null,
        error: { code: 'UNAUTHORIZED', message: '未认证', details: {} },
        requestId: 'req-logout',
      }))
      return
    }
    if (auth.endsWith('fail200')) {
      res.writeHead(200, { 'Content-Type': 'application/json; charset=utf-8' })
      res.end(JSON.stringify({ success: false }))
      return
    }
    if (auth.endsWith('ok200')) {
      res.writeHead(200, { 'Content-Type': 'application/json; charset=utf-8' })
      res.end(JSON.stringify({ success: true }))
      return
    }
    res.writeHead(204)
    res.end()
    return
  }

  if (req.method === 'GET' && url.pathname === '/api/v1/auth/me') {
    const auth = req.headers.authorization || ''
    if (auth.endsWith('header.payload.sig')) {
      res.writeHead(200, { 'Content-Type': 'application/json; charset=utf-8' })
      res.end(JSON.stringify({
        id: '00000000-0000-4000-8000-000000000001',
        username: 'system',
        displayName: '系统管理员',
        roles: ['SYSTEM'],
      }))
      return
    }
    res.writeHead(401, { 'Content-Type': 'application/json; charset=utf-8' })
    res.end(JSON.stringify({
      success: false,
      data: null,
      error: { code: 'TOKEN_INVALID', message: 'accessToken 无效或已注销', details: {} },
      requestId: 'req-me',
    }))
    return
  }

  res.writeHead(404)
  res.end()
})

await new Promise((resolve) => {
  server.listen(0, '127.0.0.1', resolve)
})

const { port } = server.address()
process.env.VITE_USE_MOCK = 'false'
process.env.VITE_API_BASE_URL = `http://127.0.0.1:${port}/api/v1`

try {
  const { login, logout, fetchMe } = await import('../src/api/authApi.js')
  const { ApiError } = await import('../src/api/errors.js')

  async function expectError(fn, code, status) {
    try {
      await fn()
    } catch (error) {
      assert(error instanceof ApiError, 'expected ApiError')
      assert(error.code === code, `expected code ${code}, got ${error.code}`)
      assert(error.status === status, `expected status ${status}, got ${error.status}`)
      return error
    }
    throw new Error(`expected ${code}`)
  }

  const bad = await expectError(() => login('system', 'wrong-password'), 'INVALID_CREDENTIALS', 401)
  assert(bad.message === '用户名或密码错误', 'error message')
  assert(bad.requestId === 'req-fixture', 'requestId')
  assert(bad.details && typeof bad.details === 'object', 'details object')

  const ok = await login('system', 'good')
  assert(ok.tokenType === 'Bearer', 'tokenType')
  assert(ok.accessToken === 'header.payload.sig', 'real-shaped token')
  assert(!ok.accessToken.endsWith('.mock'), 'not a mock token')
  assert(ok.expiresIn === 28800, 'expiresIn')
  assert(ok.user.displayName === '系统管理员', 'displayName')
  assert(ok.user.roles.includes('SYSTEM'), 'roles')

  const me = await fetchMe(ok.accessToken)
  assert(me.username === 'system', 'me username')

  await logout(ok.accessToken)
  await logout('token-ok200')
  await expectError(() => logout('token-fail200'), 'HTTP_ERROR', 200)
  const revoked = await expectError(() => fetchMe('revoked'), 'TOKEN_INVALID', 401)
  assert(revoked.message === 'accessToken 无效或已注销', 'token message')

  console.log('real-api client shape checks passed')
} finally {
  await new Promise((resolve, reject) => {
    server.close((error) => (error ? reject(error) : resolve()))
  })
}
