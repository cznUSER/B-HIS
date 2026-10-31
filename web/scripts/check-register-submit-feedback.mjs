import assert from 'node:assert/strict'

import { createSubmitFeedback } from '../src/components/feedback/useSubmitFeedback.js'
import { createRegister } from '../src/api/registerApi.js'
import { mockLogin } from '../src/api/mockAuth.js'
import { mockCreatePatient } from '../src/api/mockPatients.js'
import { resetMockRegisters } from '../src/api/mockRegisters.js'

const feedback = createSubmitFeedback()
assert.equal(feedback.state.status, 'idle')
feedback.beginAttempt('register')
assert.equal(feedback.state.status, 'submitting')
// delay lock: while submitting, a second beginAttempt simply refreshes submitting state (caller must gate); status stays submitting
feedback.beginAttempt('register')
assert.equal(feedback.state.status, 'submitting')
feedback.succeed({ registerId: 'reg-1', registerNo: 'R-1001' })
assert.equal(feedback.state.status, 'success')
assert.equal(feedback.state.message, '挂号成功，流水号 R-1001')
assert.equal(feedback.state.registerNo, 'R-1001')

// sibling action copy matrix (check-in / cancel / complete / fee)
for (const action of ['check-in', 'cancel', 'complete', 'fee-pay']) {
  feedback.beginAttempt(action)
  feedback.succeed({ registerId: 'reg-x', registerNo: 'R-2000' }, action + ' ok，流水号 R-2000')
  assert.equal(feedback.state.status, 'success')
  assert.match(feedback.state.message, /R-2000/)
}

feedback.beginAttempt('register')
feedback.fail('预约时段冲突', 'APPOINTMENT_CONFLICT')
assert.equal(feedback.state.status, 'error')
assert.equal(feedback.state.code, 'APPOINTMENT_CONFLICT')
// 5xx recoverable: fail then new attempt allowed
feedback.beginAttempt('register')
feedback.fail('服务暂不可用', 'HTTP_ERROR')
assert.equal(feedback.state.status, 'error')
assert.equal(feedback.state.code, 'HTTP_ERROR')
feedback.beginAttempt('register')
assert.equal(feedback.state.status, 'submitting')
feedback.reset()
assert.equal(feedback.state.status, 'idle')

const originalMockMode = process.env.VITE_USE_MOCK
process.env.BHIS_WEB_TEST_MODE = '1'
process.env.VITE_USE_MOCK = 'true'
resetMockRegisters()
const token = mockLogin({ username: 'system', password: '1' }).accessToken
const testPatient = mockCreatePatient(token, { name: 'Feedback Test', sex: 'M', birthday: '1990-01-01', idType: 'ID_CARD', fareIdentity: 'SELF', idNumber: '110101199001011234', phone: '13800000000', nationality: '中国' })
const request = {
  patientId: testPatient.patientId, departmentId: 'dept-1', doctorId: 'doc-1',
  visitDate: new Date().toISOString().slice(0, 10), registerFee: 0,
}
const mockCreated = await createRegister(token, request, 'attempt-mock-key')
assert.ok(mockCreated.registerId)
assert.ok(mockCreated.registerNo)
// same key + same body => no second bill
const mockAgain = await createRegister(token, request, 'attempt-mock-key')
assert.equal(mockAgain.registerId, mockCreated.registerId)
assert.equal(mockAgain.registerNo, mockCreated.registerNo)

process.env.VITE_USE_MOCK = 'false'
const originalFetch = globalThis.fetch
let sentKey = ''
let postCount = 0
globalThis.fetch = async (_url, init) => {
  sentKey = init.headers['Idempotency-Key']
  postCount += 1
  return new Response(JSON.stringify({ registerId: 'reg-2', patientId: 'patient-2', status: 'NORMAL', registerNo: 'R-1002' }), { status: 201 })
}
const created = await createRegister(token, {
  patientId: 'patient-2', departmentId: 'dept-1', doctorId: 'doc-1',
  visitDate: new Date().toISOString().slice(0, 10), registerFee: 0,
}, 'attempt-stable-key')
assert.equal(sentKey, 'attempt-stable-key')
assert.equal(created.registerNo, 'R-1002')

globalThis.fetch = async () => new Response(JSON.stringify({ error: { code: 'APPOINTMENT_CONFLICT', message: '时段已有预约' } }), { status: 409 })
await assert.rejects(
  createRegister(token, {
    patientId: 'patient-2', departmentId: 'dept-1', doctorId: 'doc-1',
    visitDate: new Date().toISOString().slice(0, 10), registerFee: 0,
  }, 'attempt-conflict-key'),
  (error) => error.code === 'APPOINTMENT_CONFLICT' && error.status === 409,
)

globalThis.fetch = async () => new Response(JSON.stringify({ error: { code: 'HTTP_ERROR', message: 'server unavailable' } }), { status: 503 })
await assert.rejects(
  createRegister('test-token', {
    patientId: 'patient-2', departmentId: 'dept-1', doctorId: 'doc-1',
    visitDate: new Date().toISOString().slice(0, 10), registerFee: 0,
  }, 'attempt-server-error-key'),
  (error) => error.status === 503,
)

globalThis.fetch = async () => { throw new Error('network offline') }
await assert.rejects(
  createRegister('test-token', {
    patientId: 'patient-2', departmentId: 'dept-1', doctorId: 'doc-1',
    visitDate: new Date().toISOString().slice(0, 10), registerFee: 0,
  }, 'attempt-network-error-key'),
  (error) => error.status === 0 && error.code === 'HTTP_ERROR',
)
assert.equal(JSON.stringify(feedback.state).includes('test-token'), false)
assert.ok(postCount >= 1)

globalThis.fetch = originalFetch
delete process.env.BHIS_WEB_TEST_MODE
if (originalMockMode === undefined) delete process.env.VITE_USE_MOCK
else process.env.VITE_USE_MOCK = originalMockMode

console.log('P0_D1_REGISTER_FEEDBACK_OK')
