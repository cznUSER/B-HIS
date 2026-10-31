import assert from 'node:assert/strict'

import { createSubmitFeedback } from '../src/components/feedback/useSubmitFeedback.js'
import { saveDiagnose, commitDiagnose, abateDiagnose, amendDiagnose } from '../src/api/diagnosisApi.js'
import { mockLogin } from '../src/api/mockAuth.js'
import { mockCreatePatient } from '../src/api/mockPatients.js'
import { createRegister } from '../src/api/registerApi.js'
import { resetMockRegisters } from '../src/api/mockRegisters.js'
import { resetMockDiagnoses } from '../src/api/mockDiagnoses.js'

const feedback = createSubmitFeedback()
assert.equal(feedback.state.status, 'idle')
feedback.beginAttempt('save')
assert.equal(feedback.state.status, 'submitting')
feedback.beginAttempt('save')
assert.equal(feedback.state.status, 'submitting')
feedback.succeed({ diagnoseId: 'diag-1' })
assert.equal(feedback.state.status, 'success')
assert.match(feedback.state.message, /diag-1/)
assert.equal(feedback.state.diagnoseId, 'diag-1')

for (const action of ['amend', 'commit', 'abate']) {
  feedback.beginAttempt(action)
  feedback.succeed({ diagnoseId: 'diag-x' }, action + ' ok，诊断号 diag-x')
  assert.equal(feedback.state.status, 'success')
  assert.match(feedback.state.message, /diag-x/)
}

// Register path must remain intact after diagnoseId additive
feedback.beginAttempt('register')
feedback.succeed({ registerId: 'reg-1', registerNo: 'R-1001' })
assert.equal(feedback.state.message, '挂号成功，流水号 R-1001')
assert.equal(feedback.state.registerNo, 'R-1001')
assert.equal(feedback.state.diagnoseId, '')

feedback.beginAttempt('commit')
feedback.fail('当前诊断状态不允许提交', 'ILLEGAL_STATE_TRANSITION')
assert.equal(feedback.state.status, 'error')
assert.equal(feedback.state.code, 'ILLEGAL_STATE_TRANSITION')

feedback.beginAttempt('save')
feedback.fail('服务暂不可用', 'HTTP_ERROR')
assert.equal(feedback.state.status, 'error')
assert.equal(feedback.state.code, 'HTTP_ERROR')
feedback.beginAttempt('save')
assert.equal(feedback.state.status, 'submitting')
feedback.reset()
assert.equal(feedback.state.status, 'idle')

const originalMockMode = process.env.VITE_USE_MOCK
process.env.BHIS_WEB_TEST_MODE = '1'
process.env.VITE_USE_MOCK = 'true'
resetMockRegisters()
resetMockDiagnoses()
const token = mockLogin({ username: 'system', password: '1' }).accessToken
const testPatient = mockCreatePatient(token, {
  name: 'Diagnosis Feedback',
  sex: 'M',
  birthday: '1990-01-01',
  idType: 'ID_CARD',
  fareIdentity: 'SELF',
  idNumber: '110101199001011299',
  phone: '13800000099',
  nationality: '中国',
})
const register = await createRegister(token, {
  patientId: testPatient.patientId,
  departmentId: 'dept-1',
  doctorId: 'doc-1',
  visitDate: new Date().toISOString().slice(0, 10),
  registerFee: 0,
}, 'diag-register-key')

const payload = {
  content: '主诉：发热\n初步诊断：上感',
  structured: {
    chiefComplaint: '发热',
    presentHistory: '',
    pastHistory: '',
    examination: '',
    assessment: '上感',
    plan: '',
  },
  templateCode: 'GENERAL',
}

const saved = await saveDiagnose(token, register.registerId, payload, 'diag-save-key')
assert.ok(saved.diagnoseId)
const savedAgain = await saveDiagnose(token, register.registerId, payload, 'diag-save-key')
assert.equal(savedAgain.diagnoseId, saved.diagnoseId)

const committed = await commitDiagnose(token, saved.diagnoseId, 'diag-commit-key')
assert.equal(committed.status, 'COMMIT')
const committedAgain = await commitDiagnose(token, saved.diagnoseId, 'diag-commit-key')
assert.equal(committedAgain.diagnoseId, committed.diagnoseId)
assert.equal(committedAgain.status, 'COMMIT')

const amended = await amendDiagnose(token, committed.diagnoseId, {
  content: '主诉：发热复查\n初步诊断：上感',
  structured: {
    chiefComplaint: '发热复查',
    presentHistory: '',
    pastHistory: '',
    examination: '',
    assessment: '上感',
    plan: '',
  },
  templateCode: 'GENERAL',
  revisionReason: '补充复查记录',
}, 'diag-amend-key')
assert.ok(amended.diagnoseId)
assert.notEqual(amended.diagnoseId, committed.diagnoseId)
const amendedAgain = await amendDiagnose(token, committed.diagnoseId, {
  content: '主诉：发热复查\n初步诊断：上感',
  structured: {
    chiefComplaint: '发热复查',
    presentHistory: '',
    pastHistory: '',
    examination: '',
    assessment: '上感',
    plan: '',
  },
  templateCode: 'GENERAL',
  revisionReason: '补充复查记录',
}, 'diag-amend-key')
assert.equal(amendedAgain.diagnoseId, amended.diagnoseId)

const abated = await abateDiagnose(token, amended.diagnoseId, 'diag-abate-key')
assert.equal(abated.status, 'ABATE')
const abatedAgain = await abateDiagnose(token, amended.diagnoseId, 'diag-abate-key')
assert.equal(abatedAgain.diagnoseId, abated.diagnoseId)
assert.equal(abatedAgain.status, 'ABATE')

process.env.VITE_USE_MOCK = 'false'
const originalFetch = globalThis.fetch
let sentKey = ''
let postCount = 0
globalThis.fetch = async (_url, init) => {
  sentKey = init.headers['Idempotency-Key']
  postCount += 1
  return new Response(JSON.stringify({
    diagnoseId: 'diag-2',
    encounterId: 'reg-2',
    status: 'SAVE',
    content: '主诉：头痛',
    structured: { chiefComplaint: '头痛', presentHistory: '', pastHistory: '', examination: '', assessment: '', plan: '' },
    templateCode: 'GENERAL',
    version: 1,
  }), { status: 201 })
}
const created = await saveDiagnose(token, 'reg-2', {
  content: '主诉：头痛',
  structured: { chiefComplaint: '头痛', presentHistory: '', pastHistory: '', examination: '', assessment: '', plan: '' },
  templateCode: 'GENERAL',
}, 'attempt-stable-diag-key')
assert.equal(sentKey, 'attempt-stable-diag-key')
assert.equal(created.diagnoseId, 'diag-2')

globalThis.fetch = async () => new Response(JSON.stringify({ error: { code: 'ILLEGAL_STATE_TRANSITION', message: '当前诊断状态不允许提交' } }), { status: 409 })
await assert.rejects(
  commitDiagnose(token, 'diag-2', 'attempt-illegal-key'),
  (error) => error.code === 'ILLEGAL_STATE_TRANSITION' && error.status === 409,
)

globalThis.fetch = async () => new Response(JSON.stringify({ error: { code: 'DIAGNOSE_NOT_AMENDABLE', message: '只有已提交诊断可以发起修订' } }), { status: 409 })
await assert.rejects(
  amendDiagnose(token, 'diag-2', {
    content: '主诉：改',
    structured: { chiefComplaint: '改', presentHistory: '', pastHistory: '', examination: '', assessment: '', plan: '' },
    templateCode: 'GENERAL',
    revisionReason: '测试',
  }, 'attempt-not-amendable-key'),
  (error) => error.code === 'DIAGNOSE_NOT_AMENDABLE' && error.status === 409,
)

globalThis.fetch = async () => new Response(JSON.stringify({ error: { code: 'HTTP_ERROR', message: 'server unavailable' } }), { status: 503 })
await assert.rejects(
  saveDiagnose(token, 'reg-2', {
    content: '主诉：头痛',
    structured: { chiefComplaint: '头痛', presentHistory: '', pastHistory: '', examination: '', assessment: '', plan: '' },
    templateCode: 'GENERAL',
  }, 'attempt-server-error-key'),
  (error) => error.status === 503,
)

globalThis.fetch = async () => { throw new Error('network offline') }
await assert.rejects(
  saveDiagnose(token, 'reg-2', {
    content: '主诉：头痛',
    structured: { chiefComplaint: '头痛', presentHistory: '', pastHistory: '', examination: '', assessment: '', plan: '' },
    templateCode: 'GENERAL',
  }, 'attempt-network-error-key'),
  (error) => error.status === 0 && error.code === 'HTTP_ERROR',
)

assert.equal(JSON.stringify(feedback.state).includes(token), false)
assert.ok(postCount >= 1)

globalThis.fetch = originalFetch
delete process.env.BHIS_WEB_TEST_MODE
if (originalMockMode === undefined) delete process.env.VITE_USE_MOCK
else process.env.VITE_USE_MOCK = originalMockMode

console.log('P0_D2_DIAGNOSIS_FEEDBACK_OK')