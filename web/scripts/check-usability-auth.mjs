// 2026-10-14 P1-H05-WEB-USABILITY 深链回跳行为回归。
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { parse, compileScript } from '@vue/compiler-sfc'
import { transform } from 'esbuild'
import * as vue from 'vue'

const values = new Map()
globalThis.sessionStorage = {
  getItem(key) { return values.get(key) ?? null },
  setItem(key, value) { values.set(key, String(value)) },
  removeItem(key) { values.delete(key) },
}

const { safeReturnTo, rememberReturnTo, takeReturnTo } = await import('../src/router/returnTo.js')
const { session, applyLogin, clearSession } = await import('../src/auth/session.js')
const { readErrorResponse } = await import('../src/api/errors.js')
const { outpatientTarget } = await import('../src/workbench/context.js')
assert.deepEqual(outpatientTarget('dispense', { patientId: 'p-1', charge_id: 'c-1', view: 'records', junk: 'x' }), {
  name: 'outpatient-module', params: { module: 'dispense' }, query: { patientId: 'p-1', charge_id: 'c-1' },
})
assert.deepEqual(outpatientTarget('charge', { chargeId: 'c-2', dispenseId: ['d-1','d-2'] }, { view: 'records' }), {
  name: 'outpatient-module', params: { module: 'charge' }, query: { charge_id: 'c-2', dispenseId: 'd-1', view: 'records' },
})

function signIn(token) {
  applyLogin({ accessToken: token, tokenType: 'Bearer', expiresIn: 3600, user: { username: 'tester' } })
}
function errorResponse(status, code) {
  return new Response(JSON.stringify({ error: { code, message: 'fixture' } }), { status })
}
signIn('current-token')
await readErrorResponse(errorResponse(401, 'TOKEN_INVALID'), 'current-token')
assert.equal(session.accessToken, '', 'current authenticated 401 must clear the session')
signIn('new-token')
await readErrorResponse(errorResponse(401, 'TOKEN_EXPIRED'), 'old-token')
assert.equal(session.accessToken, 'new-token', 'late old request must not log out a new login')
await readErrorResponse(errorResponse(403, 'FORBIDDEN'), 'new-token')
assert.equal(session.accessToken, 'new-token', 'permission failure must not clear login')
await readErrorResponse(errorResponse(401, 'INVALID_CREDENTIALS'))
assert.equal(session.accessToken, 'new-token', 'bad login credentials must not expire an existing session')
clearSession()

async function shell(auth) {
  const route = { name: 'home', fullPath: '/app', params: {}, meta: {}, query: {} }
  const navigation = []
  const mounted = []
  const { descriptor } = parse(readFileSync(new URL('../src/components/AppShell.vue', import.meta.url), 'utf8'))
  const script = compileScript(descriptor, { id: 'auth-regression' })
  const output = await transform(script.content, { format: 'cjs' })
  const dependencies = {
    vue: { ...vue, onMounted: (fn) => mounted.push(fn), onBeforeUnmount() {}, watch() {} },
    'vue-router': { useRoute: () => route, useRouter: () => ({ replace: async (target) => navigation.push(target) }) },
    '../api/authApi.js': auth,
    '../api/config.js': { useMock: () => false },
    '../api/errors.js': { ApiError: (await import('../src/api/errors.js')).ApiError },
    '../auth/session.js': await import('../src/auth/session.js'),
    '../router/returnTo.js': await import('../src/router/returnTo.js'),
    '../workbench/catalog.js': await import('../src/workbench/catalog.js'),
  }
  const module = { exports: {} }
  new Function('require', 'module', 'exports', output.code)((name) => dependencies[name] || { default: {} }, module, module.exports)
  return { state: module.exports.default.setup({}, { expose() {} }), mounted, navigation }
}
signIn('old-logout')
let finishLogout
const logoutShell = await shell({ fetchMe: async () => ({}), logout: () => new Promise((resolve) => { finishLogout = resolve }) })
const pendingLogout = logoutShell.state.onLogout()
signIn('new-login')
finishLogout()
await pendingLogout
assert.equal(session.accessToken, 'new-login', 'late logout must not clear a newer login')
assert.deepEqual(logoutShell.navigation, [], 'late logout must not navigate the new session')
clearSession()

assert.equal(safeReturnTo('/app/outpatient/charge?charge_id=c-1'), '/app/outpatient/charge?charge_id=c-1')
assert.equal(safeReturnTo('//evil.example/login'), '')
assert.equal(safeReturnTo('/login?next=/app'), '')
assert.equal(safeReturnTo('/app/no-such-page'), '')
assert.equal(safeReturnTo('/app\\\\evil.example'), '')
assert.equal(safeReturnTo('/app/settings/users', { roles: ['REGISTRAR'], permissions: [], scopes: [], accessControlLoaded: true }), '')
assert.equal(safeReturnTo('/app/settings/users/roles', { permissions: ['admin:role'], scopes: [], accessControlLoaded: true }), '/app/settings/users/roles')
rememberReturnTo('/app/outpatient/order?registerId=r-1')
assert.equal(takeReturnTo(), '/app/outpatient/order?registerId=r-1')
assert.equal(takeReturnTo(), '')
rememberReturnTo('https://evil.example')
assert.equal(takeReturnTo(), '')

console.log('check-usability-auth: PASS')
