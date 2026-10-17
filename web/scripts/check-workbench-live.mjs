// P2-2 isolated API acceptance: write an outpatient fixture, refresh the
// dashboard, verify role filtering, and exercise duplicate charge requests.
// This script deliberately refuses production-like targets.

import { spawn } from 'node:child_process'
import { fileURLToPath } from 'node:url'

const API_BASE = String(process.env.BHIS_PHASE2_API_BASE || 'http://127.0.0.1:8081/api/v1').replace(/\/+$/, '')
const DATABASE_NAME = String(process.env.BHIS_PHASE2_DB_NAME || '')
const EXPECTED_DATABASE = 'BHIS_PHASE2_TEST_20261015'

function assert(condition, message) {
  if (!condition) throw new Error(message)
}

function requiredEnvironment() {
  assert(DATABASE_NAME === EXPECTED_DATABASE, `refusing writes: BHIS_PHASE2_DB_NAME must be ${EXPECTED_DATABASE}`)
  const target = new URL(API_BASE)
  assert(target.protocol === 'http:' && target.hostname === '127.0.0.1', 'refusing writes: API must be local HTTP')
  assert(['8081', '8080', '9999'].includes(target.port), 'refusing writes: API port must be 8080, 8081, or 9999')
  assert(target.pathname === '/api/v1', 'refusing writes: API path must be /api/v1')
}

async function request(path, { method = 'GET', token = '', body, expected = [200], idempotency = false } = {}) {
  const headers = { Accept: 'application/json' }
  if (token) headers.Authorization = `Bearer ${token}`
  if (body !== undefined) {
    headers['Content-Type'] = 'application/json; charset=utf-8'
    if (idempotency) headers['Idempotency-Key'] = crypto.randomUUID()
  }
  let response
  try {
    response = await fetch(`${API_BASE}${path}`, {
      method,
      headers,
      cache: 'no-store',
      body: body === undefined ? undefined : JSON.stringify(body),
    })
  } catch (error) {
    throw new Error(`API unreachable at ${API_BASE}: ${error instanceof Error ? error.message : String(error)}`)
  }
  const text = await response.text()
  let payload = null
  if (text) {
    try { payload = JSON.parse(text) } catch { payload = { raw: text } }
  }
  if (!expected.includes(response.status)) {
    const code = payload && payload.error && payload.error.code ? payload.error.code : 'UNKNOWN'
    throw new Error(`${method} ${path} expected ${expected.join('/')} got ${response.status} (${code})`)
  }
  return { status: response.status, body: payload }
}

async function health() {
  const origin = new URL(API_BASE).origin
  const response = await fetch(`${origin}/health`, { cache: 'no-store' })
  const body = await response.json()
  assert(response.ok && body && body.status === 'ok', 'isolated API health')
  return body
}

async function runConcurrentChargeClients(orderId, token) {
  const script = fileURLToPath(import.meta.url)
  const runOne = () => new Promise((resolve, reject) => {
    const child = spawn(process.execPath, [script, '--charge-once'], {
      env: { ...process.env, BHIS_PHASE2_CHILD_ORDER: orderId, BHIS_PHASE2_CHILD_TOKEN: token },
      stdio: ['ignore', 'pipe', 'pipe'],
    })
    let stdout = ''
    let stderr = ''
    child.stdout.on('data', (chunk) => { stdout += chunk })
    child.stderr.on('data', (chunk) => { stderr += chunk })
    child.on('error', reject)
    child.on('close', (code) => {
      const line = stdout.trim().split(/\r?\n/).filter(Boolean).at(-1)
      if (code !== 0 || !line) {
        reject(new Error(`concurrent client failed (${code}): ${stderr.trim() || stdout.trim()}`))
        return
      }
      try { resolve(JSON.parse(line)) } catch { reject(new Error(`concurrent client output invalid: ${line}`)) }
    })
  })
  return Promise.all([runOne(), runOne()])
}

if (process.argv[2] === '--charge-once') {
  requiredEnvironment()
  const childResult = await request('/charges', {
    method: 'POST',
    token: String(process.env.BHIS_PHASE2_CHILD_TOKEN || ''),
    idempotency: true,
    expected: [201, 409],
    body: { orderId: String(process.env.BHIS_PHASE2_CHILD_ORDER || '') },
  })
  console.log(JSON.stringify({ status: childResult.status, chargeId: childResult.body && childResult.body.chargeId }))
  process.exit(0)
}

function todayInShanghai() {
  return new Intl.DateTimeFormat('en-CA', {
    timeZone: 'Asia/Shanghai', year: 'numeric', month: '2-digit', day: '2-digit',
  }).format(new Date())
}

function summaryShape(body, period = 'TODAY') {
  assert(body && body.period === period, `summary period ${period}`)
  assert(Date.parse(body.rangeStart) < Date.parse(body.rangeEnd), `summary range ${period}`)
  for (const field of ['revenueCents', 'refundCents', 'netRevenueCents', 'chargeCount', 'encounterCount', 'pendingChargeCount', 'pendingDispenseCount']) {
    assert(Number.isSafeInteger(Number(body[field])) && Number(body[field]) >= 0, `summary field ${field}`)
  }
  assert(Number(body.netRevenueCents) === Number(body.revenueCents) - Number(body.refundCents), 'summary net revenue')
  assert(body.available && typeof body.available === 'object', 'summary availability')
  return body
}

async function login(username, password) {
  const response = await request('/auth/login', { method: 'POST', body: { username, password }, expected: [200] })
  assert(response.body && typeof response.body.accessToken === 'string' && response.body.accessToken, `${username} login token`)
  return response.body
}

function userSummary(body) {
  return {
    charges: Boolean(body.available && body.available.charges),
    encounters: Boolean(body.available && body.available.encounters),
    dispenses: Boolean(body.available && body.available.dispenses),
  }
}

requiredEnvironment()
const suffix = `${Date.now().toString(36)}${Math.random().toString(36).slice(2, 7)}`
const patientName = `P2工作台验收${suffix}`
const username = `p2a_${suffix}`
let systemToken = ''
let roleToken = ''
let userId = ''
let registerId = ''
let chargeId = ''
let acceptance = null
const cleanup = { refund: 'not-needed', register: 'not-needed', user: 'not-needed' }

try {
  await health()
  const system = await login('system', '1')
  systemToken = system.accessToken
  const before = summaryShape((await request('/workbench/summary?period=TODAY', { token: systemToken })).body)

  const patient = await request('/patients', {
    method: 'POST',
    token: systemToken,
    idempotency: true,
    expected: [201],
    body: {
      name: patientName,
      sex: 'U',
      birthday: '1990-01-01',
      fareIdentity: 'SELF',
    },
  })
  const patientId = patient.body && patient.body.patientId
  assert(typeof patientId === 'string' && patientId, 'patient create response')

  const register = await request('/registers', {
    method: 'POST',
    token: systemToken,
    idempotency: true,
    expected: [201],
    body: { patientId, departmentId: 'dept-1', doctorId: 'doc-1', visitDate: todayInShanghai(), registerFee: 0 },
  })
  registerId = register.body && register.body.registerId
  assert(typeof registerId === 'string' && registerId, 'register create response')

  const afterRegister = summaryShape((await request('/workbench/summary?period=TODAY', { token: systemToken })).body)
  assert(afterRegister.encounterCount >= before.encounterCount + 1, 'dashboard encounter count did not refresh after register write')

  const user = await request('/users', {
    method: 'POST',
    token: systemToken,
    idempotency: true,
    expected: [201],
    body: {
      username,
      displayName: 'P2 工作台范围验收',
      password: 'Phase2!123',
      roles: ['REGISTRAR'],
      roleScopes: [{ roleCode: 'REGISTRAR', departmentId: 'dept-1', status: 'ACTIVE' }],
      mustChangePassword: false,
    },
  })
  userId = user.body && user.body.userId
  assert(typeof userId === 'string' && userId, 'role user create response')
  const role = await login(username, 'Phase2!123')
  roleToken = role.accessToken
  const roleSummary = summaryShape((await request('/workbench/summary?period=TODAY', { token: roleToken })).body)
  const availability = userSummary(roleSummary)
  assert(availability.encounters === true, 'registrar cannot see encounters')
  assert(availability.charges === false && availability.dispenses === false, 'registrar metric permission leaked')

  const order = await request('/orders', {
    method: 'POST',
    token: systemToken,
    idempotency: true,
    expected: [201],
    body: {
      registerId,
      patientId,
      departmentId: 'dept-1',
      doctorId: 'doc-1',
      diagnoseName: 'P2 工作台并发验收',
      items: [{ itemNo: 'TREAT-1', itemType: 'TREAT', itemName: '演示诊查费', units: '次', amount: 1, price: 15 }],
    },
  })
  const orderId = order.body && order.body.orderId
  assert(typeof orderId === 'string' && orderId, 'order create response')
  const committed = await request(`/orders/${encodeURIComponent(orderId)}/commit`, {
    method: 'POST', token: systemToken, idempotency: true, expected: [200], body: {},
  })
  assert(committed.body && committed.body.status === 'COMMIT', 'order commit response')

  const concurrent = await runConcurrentChargeClients(orderId, systemToken)
  const successes = concurrent.filter((item) => item.status === 201)
  const conflicts = concurrent.filter((item) => item.status === 409)
  assert(successes.length === 1 && conflicts.length === 1, `concurrent charge results ${concurrent.map((item) => item.status).join(',')}`)
  chargeId = successes[0].chargeId
  assert(typeof chargeId === 'string' && chargeId, 'charge create response')
  const charge = (await request(`/charges/${encodeURIComponent(chargeId)}`, { token: systemToken, expected: [200] })).body
  const payment = await request(`/charges/${encodeURIComponent(chargeId)}/pay`, {
    method: 'POST', token: systemToken, idempotency: true, expected: [200],
    body: { payType: 'CASH', amount: charge.squareFee },
  })
  assert(payment.body && payment.body.payType === 'CASH', 'cash payment response')

  const afterPayment = summaryShape((await request('/workbench/summary?period=TODAY', { token: systemToken })).body)
  assert(afterPayment.chargeCount >= before.chargeCount + 1, 'dashboard charge count did not refresh after payment')
  assert(afterPayment.revenueCents >= before.revenueCents + Math.round(Number(charge.squareFee) * 100), 'dashboard revenue did not refresh after payment')
  acceptance = {
    database: EXPECTED_DATABASE,
    patientWritten: true,
    encounterDelta: afterRegister.encounterCount - before.encounterCount,
    paymentWritten: true,
    chargeDelta: afterPayment.chargeCount - before.chargeCount,
    revenueDeltaCents: afterPayment.revenueCents - before.revenueCents,
    registrarAvailability: availability,
    concurrentChargeStatuses: concurrent.map((item) => item.status),
    result: 'PASS',
  }
} finally {
  if (chargeId && systemToken) {
    const charge = await request(`/charges/${encodeURIComponent(chargeId)}`, { token: systemToken, expected: [200] }).catch(() => null)
    const items = charge && charge.body && Array.isArray(charge.body.items) ? charge.body.items : []
    if (items.length) {
      const refunded = await request(`/charges/${encodeURIComponent(chargeId)}/refund`, {
        method: 'POST', token: systemToken, idempotency: true, expected: [200, 409],
        body: { payType: 'CASH', reason: 'P2-2 隔离验收清理', items: items.map((item) => ({ chargeItemId: item.chargeItemId, refundQty: item.qty })) },
      }).catch(() => null)
      cleanup.refund = refunded ? `HTTP_${refunded.status}` : 'FAILED'
    }
  }
  if (registerId && systemToken) {
    const cancelled = await request(`/registers/${encodeURIComponent(registerId)}/cancel`, { method: 'POST', token: systemToken, idempotency: true, expected: [200, 409], body: {} }).catch(() => null)
    cleanup.register = cancelled ? `HTTP_${cancelled.status}` : 'FAILED'
  }
  if (userId && systemToken) {
    const stopped = await request(`/users/${encodeURIComponent(userId)}`, { method: 'PATCH', token: systemToken, expected: [200], body: { status: 'INACTIVE' } }).catch(() => null)
    cleanup.user = stopped ? `HTTP_${stopped.status}` : 'FAILED'
  }
}

if (acceptance) console.log(JSON.stringify({ ...acceptance, cleanup }))
