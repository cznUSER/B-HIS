// 2026-10-12 P1-H05-REGISTER-UI 挂号校验、退号规则，以及 /registers 真实接口形状。
// 2026-10-12 P1-H05b-REGISTER-CHECKIN-UI 报到：Mock 成功/409，以及 check-in 请求形状。
import { readFileSync } from 'node:fs'
import http from 'node:http'
import { ApiError } from '../src/api/errors.js'
import { mockLogin, resetMockAuth } from '../src/api/mockAuth.js'
import { mockCreatePatient, resetMockPatients } from '../src/api/mockPatients.js'
import { resetPatientClientCache } from '../src/api/patientApi.js'
import { todayInShanghai } from '../src/api/patientContract.js'
import { assertCreateRegisterRequest } from '../src/api/registerContract.js'
import { cancelRegister, checkInRegister, createRegister, listRegisters, presentCheckInError, presentRegisterError } from '../src/api/registerApi.js'
import {
  mockCancelRegister,
  mockCheckInRegister,
  mockCreateRegister,
  mockListRegisters,
  resetMockRegisters,
  stageMockRegister,
} from '../src/api/mockRegisters.js'
import {
  cancelRejectMessage,
  canCancel,
  canCheckIn,
  doctorsForDept,
  statusLabel,
} from '../src/register/master.js'

function assert(condition, message) {
  if (!condition) throw new Error(message)
}

async function expectError(fn, code, status) {
  try {
    await fn()
  } catch (error) {
    assert(error instanceof ApiError, 'expected ApiError')
    assert(error.code === code, `expected code ${code}, got ${error.code}`)
    if (status != null) assert(error.status === status, `expected status ${status}, got ${error.status}`)
    return error
  }
  throw new Error(`expected ${code}`)
}

process.env.VITE_USE_MOCK = 'true'
resetMockAuth()
resetMockPatients()
resetMockRegisters()
resetPatientClientCache()

const token = mockLogin({ username: 'system', password: '1' }).accessToken
const today = todayInShanghai()

assert(doctorsForDept('dept-1').map((item) => item.doctorId).join(',') === 'doc-1', '内科只有张医生')
assert(doctorsForDept('dept-2').map((item) => item.name).join(',') === '李医生', '外科只有李医生')
assert(doctorsForDept('').length === 0, '未选科室没有医生')
assert(canCancel('NORMAL') === true, '正常号可退')
for (const status of ['USED', 'DIAGNOSING', 'DIAGNOSED', 'PAUSED', 'BACK']) {
  assert(canCancel(status) === false, `${status} 不可退`)
}
for (const status of ['NORMAL', 'USED']) {
  assert(canCheckIn(status) === true, `${status} 可报到`)
}
for (const status of ['DIAGNOSING', 'DIAGNOSED', 'PAUSED', 'BACK', '']) {
  assert(canCheckIn(status) === false, `${status || '空'} 不可报到`)
}
assert(statusLabel('NORMAL') === '正常号', '正常号文案')
assert(statusLabel('BACK') === '退号', '退号文案')
assert(cancelRejectMessage('USED') === '当前号已使用或就诊中，不能退号', '已用')
assert(cancelRejectMessage('DIAGNOSING') === '当前号已使用或就诊中，不能退号', '就诊中')
assert(cancelRejectMessage('DIAGNOSED') === '当前号已使用或就诊中，不能退号', '就诊完成')
assert(cancelRejectMessage('PAUSED') === '当前号已使用或就诊中，不能退号', '挂起')
assert(cancelRejectMessage('BACK') === '该号已退，不能再次退号', '已退')

const noPatient = await expectError(() => assertCreateRegisterRequest({}), 'VALIDATION_ERROR', 400)
assert(noPatient.message === '请先选择或建档患者', '缺患者')
const noDept = await expectError(
  () => assertCreateRegisterRequest({ patientId: 'p', visitDate: today, doctorId: 'doc-1' }),
  'VALIDATION_ERROR',
  400,
)
assert(noDept.message === '请选择科室', '缺科室')
const legacyDept = await expectError(
  () => assertCreateRegisterRequest({ patientId: 'p', deptCode: 'dept-1', doctorId: 'doc-1', visitDate: today }),
  'VALIDATION_ERROR',
  400,
)
assert(legacyDept.message === '请选择科室', '不认 deptCode')
const wrongDoctor = await expectError(
  () => assertCreateRegisterRequest({ patientId: 'p', departmentId: 'dept-1', doctorId: 'doc-2', visitDate: today }),
  'VALIDATION_ERROR',
  400,
)
assert(wrongDoctor.message === '请选择医生', '医生须属科室')

const payload = assertCreateRegisterRequest({
  patientId: 'p',
  departmentId: 'dept-1',
  doctorId: 'doc-1',
  visitDate: today,
  registerFee: 0,
})
for (const key of ['patientId', 'departmentId', 'doctorId', 'visitDate']) {
  assert(Object.prototype.hasOwnProperty.call(payload, key), `payload ${key}`)
}
assert(!Object.prototype.hasOwnProperty.call(payload, 'deptCode'), 'payload 无 deptCode')
assert(payload.registerSort === 'sort-normal', '号别占位')
assert(payload.registerFee === 0, '费用默认 0')

const patient = mockCreatePatient(token, {
  name: '王小明',
  sex: 'M',
  birthday: '1990-05-16',
  fareIdentity: 'SELF',
  idNumber: '110101199005161234',
})

const created = mockCreateRegister(token, {
  patientId: patient.patientId,
  departmentId: 'dept-2',
  doctorId: 'doc-2',
  visitDate: today,
  registerFee: 0,
})
assert(created.status === 'NORMAL', '保存后正常号')
assert(created.registerFee === 0 && created.registerCharged === false, '费用隐藏默认')
assert(created.departmentId === 'dept-2' && created.doctorId === 'doc-2', '科室医生写入')
assert(created.patientId === patient.patientId && created.visitDate === today, '患者与就诊日')

const listed = mockListRegisters(token, { patientId: patient.patientId, visitDate: today })
assert(listed.items.some((item) => item.registerId === created.registerId && statusLabel(item.status) === '正常号'), '列表可见正常号')
assert(listed.items.every((item) => item.status === 'NORMAL'), '列表不塞演示号')

for (const status of ['USED', 'DIAGNOSING', 'DIAGNOSED', 'PAUSED']) {
  const row = stageMockRegister(token, {
    patientId: patient.patientId,
    visitDate: today,
    status,
    registerId: `stage-${status}`,
  })
  const rejected = await expectError(() => mockCancelRegister(token, row.registerId), 'REGISTER_NOT_CANCELLABLE', 409)
  assert(rejected.message === '当前号已使用或就诊中，不能退号', `拒绝文案 ${status}`)
  const still = mockListRegisters(token, { patientId: patient.patientId, visitDate: today })
  assert(still.items.find((item) => item.registerId === row.registerId).status === status, `${status} 状态不变`)
}

const backed = mockCancelRegister(token, created.registerId)
assert(backed.status === 'BACK', '退号成功')
assert(backed.backOperatorId && backed.backOperatedAt, '留下退号操作')
const twice = await expectError(() => mockCancelRegister(token, created.registerId), 'REGISTER_BACK_NOT_ALLOWED', 409)
assert(twice.message === '该号已退，不能再次退号', '不可再退')

const viaApi = await createRegister(token, {
  patientId: patient.patientId,
  departmentId: 'dept-1',
  doctorId: 'doc-1',
  visitDate: today,
})
assert(viaApi.status === 'NORMAL', 'api mock 正常号')
const apiList = await listRegisters(token, { patientId: patient.patientId, visitDate: today })
assert(apiList.items.some((item) => item.registerId === viaApi.registerId && item.status === 'NORMAL'), 'api 列表')
const apiBack = await cancelRegister(token, viaApi.registerId)
assert(apiBack.status === 'BACK', 'api 退号')

const checkInSource = mockCreateRegister(token, {
  patientId: patient.patientId,
  departmentId: 'dept-1',
  doctorId: 'doc-1',
  visitDate: today,
})
const checked = mockCheckInRegister(token, checkInSource.registerId)
assert(checked.status === 'DIAGNOSING', '报到后就诊中')
assert(statusLabel(checked.status) === '就诊中', '就诊中文案')
const checkedAgain = await expectError(
  () => mockCheckInRegister(token, checkInSource.registerId),
  'ILLEGAL_STATE_TRANSITION',
  409,
)
assert(checkedAgain.message === '当前状态不允许该操作', '重复报到')
const stillDiagnosing = mockListRegisters(token, { patientId: patient.patientId, visitDate: today })
assert(
  stillDiagnosing.items.find((item) => item.registerId === checkInSource.registerId).status === 'DIAGNOSING',
  '失败后仍是就诊中',
)

const usedRow = stageMockRegister(token, {
  patientId: patient.patientId,
  visitDate: today,
  status: 'USED',
  registerId: 'stage-checkin-used',
})
const fromUsed = mockCheckInRegister(token, usedRow.registerId)
assert(fromUsed.status === 'DIAGNOSING', '已用号可报到')

for (const status of ['DIAGNOSED', 'PAUSED', 'BACK']) {
  const row = stageMockRegister(token, {
    patientId: patient.patientId,
    visitDate: today,
    status,
    registerId: `stage-checkin-${status}`,
  })
  const deniedCheckIn = await expectError(
    () => mockCheckInRegister(token, row.registerId),
    'ILLEGAL_STATE_TRANSITION',
    409,
  )
  assert(deniedCheckIn.message === '当前状态不允许该操作', `报到拒绝 ${status}`)
  const unchanged = mockListRegisters(token, { patientId: patient.patientId, visitDate: today })
  assert(unchanged.items.find((item) => item.registerId === row.registerId).status === status, `${status} 报到失败不改状态`)
}

const conflictSource = mockCreateRegister(token, {
  patientId: patient.patientId,
  departmentId: 'dept-1',
  doctorId: 'doc-1',
  visitDate: today,
})
const conflicted = await expectError(
  () => mockCheckInRegister(token, conflictSource.registerId, { simulateConflict: true }),
  'ILLEGAL_STATE_TRANSITION',
  409,
)
assert(conflicted.message === '当前状态不允许该操作', '模拟冲突文案')
const conflictList = mockListRegisters(token, { patientId: patient.patientId, visitDate: today })
assert(
  conflictList.items.find((item) => item.registerId === conflictSource.registerId).status === 'NORMAL',
  '模拟冲突不改状态',
)
const conflictViaApi = await expectError(
  () => checkInRegister(token, conflictSource.registerId, { simulateConflict: true }),
  'ILLEGAL_STATE_TRANSITION',
  409,
)
assert(conflictViaApi.message === '当前状态不允许该操作', 'api mock 模拟冲突')
const arrived = await checkInRegister(token, conflictSource.registerId)
assert(arrived.status === 'DIAGNOSING', 'api mock 报到成功')

const missingCheckIn = await expectError(() => mockCheckInRegister(token, 'missing-reg'), 'NOT_FOUND', 404)
assert(missingCheckIn.message === '未找到挂号单', '报到找不到号')

const blankConflict = presentCheckInError({
  code: 'ILLEGAL_STATE_TRANSITION',
  status: 409,
  message: '',
  details: null,
  requestId: 'c1',
})
assert(blankConflict.message === '当前状态不允许该操作', '409 缺省文案')
const serverConflict = presentCheckInError({
  code: 'ILLEGAL_STATE_TRANSITION',
  status: 409,
  message: '该号已在就诊中',
  details: null,
  requestId: 'c2',
})
assert(serverConflict.message === '该号已在就诊中', '409 沿用服务端文案')
const missingMapped = presentCheckInError({
  code: 'NOT_FOUND',
  status: 404,
  message: '未找到患者',
  details: null,
  requestId: 'c3',
})
assert(missingMapped.message === '未找到挂号单', '404 改成挂号文案')
const downMapped = presentCheckInError({
  code: 'HTTP_ERROR',
  status: 500,
  message: '请求失败，请稍后重试',
  details: null,
  requestId: 'c4',
})
assert(downMapped.message === '报到失败，请稍后重试', '5xx 缺省文案')
const downServer = presentCheckInError({
  code: 'HTTP_ERROR',
  status: 503,
  message: '挂号服务暂不可用',
  details: null,
  requestId: 'c5',
})
assert(downServer.message === '挂号服务暂不可用', '5xx 沿用服务端文案')

const view = readFileSync(new URL('../src/views/RegisterView.vue', import.meta.url), 'utf8')
const template = view.split('<template>')[1]
for (const text of ['请先选择或建档患者', '请选择科室', '请选择医生', '普通号', '去建档', '回建档页', '退号', '报到', '模拟报到冲突', '姓名', '性别', '出生日期', '患者号', '身份证']) {
  assert(template.includes(text), `missing copy: ${text}`)
}
assert(template.includes('v-if="canCheckIn(row.status)"'), '仅可报到状态显示按钮')
assert(!template.includes('type="date"'), '无日期控件')
assert(!template.includes('排班'), '无排班文案')
assert(!template.includes('挂号费'), '费用不展示')
assert(!view.includes('deptCode'), '页面不用 deptCode')
assert(view.includes('departmentId:'), '提交 departmentId')
assert(view.includes('registerFee: 0'), '提交费用为 0')
const apiSrc = readFileSync(new URL('../src/api/registerApi.js', import.meta.url), 'utf8')
assert(apiSrc.includes("apiUrl('/registers')"), 'create path')
assert(apiSrc.includes('/registers/${encodeURIComponent(id)}/cancel'), 'cancel path')
assert(apiSrc.includes('/registers/${encodeURIComponent(id)}/check-in'), 'check-in path')
assert(apiSrc.includes('useMock()'), 'mock switch')
assert(!apiSrc.includes('/encounters'), '不再走 encounters')
assert(!apiSrc.includes('deptCode'), '客户端不用 deptCode')

const usedTip = presentRegisterError({
  code: 'REGISTER_NOT_CANCELLABLE',
  status: 409,
  message: '当前状态不可退号',
  details: null,
  requestId: 'r1',
})
assert(usedTip.message === '当前号已使用或就诊中，不能退号', '不可退号中文')
const backTip = presentRegisterError({
  code: 'REGISTER_BACK_NOT_ALLOWED',
  status: 409,
  message: '已退号，不能再次退号',
  details: null,
  requestId: 'r2',
})
assert(backTip.message === '该号已退，不能再次退号', '已退中文')

const calls = []
const server = http.createServer(async (req, res) => {
  const url = new URL(req.url || '/', 'http://127.0.0.1')
  const chunks = []
  for await (const chunk of req) chunks.push(chunk)
  const raw = Buffer.concat(chunks).toString()
  calls.push({ method: req.method, path: url.pathname, search: url.search, body: raw, headers: req.headers })
  const json = (status, payload) => {
    res.writeHead(status, { 'Content-Type': 'application/json; charset=utf-8' })
    res.end(JSON.stringify(payload))
  }
  if (!String(req.headers.authorization || '').startsWith('Bearer ')) {
    json(401, {
      success: false,
      data: null,
      error: { code: 'UNAUTHORIZED', message: '未认证', details: {} },
      requestId: 'req-auth',
    })
    return
  }
  if (req.method === 'POST' && url.pathname === '/api/v1/registers') {
    const body = JSON.parse(raw || '{}')
    json(201, {
      registerId: 'reg-1',
      registerNo: 'GH0001',
      patientId: body.patientId,
      departmentId: body.departmentId,
      doctorId: body.doctorId,
      registerSort: body.registerSort,
      visitDate: body.visitDate,
      status: 'NORMAL',
      registerFee: body.registerFee,
      registerCharged: false,
      registeredAt: '2026-10-12T01:00:00Z',
      backOperatorId: null,
      backOperatedAt: null,
    })
    return
  }
  if (req.method === 'GET' && url.pathname === '/api/v1/registers') {
    json(200, {
      items: [
        {
          registerId: 'reg-used',
          registerNo: 'GH9001',
          patientId: 'created-1',
          departmentId: 'dept-1',
          doctorId: 'doc-1',
          registerSort: 'sort-normal',
          visitDate: today,
          status: 'USED',
          registerFee: 0,
          registerCharged: false,
          registeredAt: '2026-10-12T00:00:00Z',
          backOperatorId: null,
          backOperatedAt: null,
        },
      ],
    })
    return
  }
  if (req.method === 'POST' && url.pathname === '/api/v1/registers/reg-1/cancel') {
    json(200, {
      registerId: 'reg-1',
      registerNo: 'GH0001',
      patientId: 'created-1',
      departmentId: 'dept-1',
      doctorId: 'doc-1',
      registerSort: 'sort-normal',
      visitDate: today,
      status: 'BACK',
      registerFee: 0,
      registerCharged: false,
      registeredAt: '2026-10-12T01:00:00Z',
      backOperatorId: 'u-system',
      backOperatedAt: '2026-10-12T02:00:00Z',
    })
    return
  }
  if (req.method === 'POST' && url.pathname === '/api/v1/registers/reg-used/cancel') {
    json(409, {
      success: false,
      data: null,
      error: { code: 'REGISTER_NOT_CANCELLABLE', message: '当前状态不可退号', details: {} },
      requestId: 'req-used',
    })
    return
  }
  if (req.method === 'POST' && url.pathname === '/api/v1/registers/reg-back/cancel') {
    json(409, {
      success: false,
      data: null,
      error: { code: 'REGISTER_BACK_NOT_ALLOWED', message: '已退号，不能再次退号', details: {} },
      requestId: 'req-back',
    })
    return
  }
  if (req.method === 'POST' && url.pathname === '/api/v1/registers/reg-used/check-in') {
    json(200, {
      registerId: 'reg-used',
      registerNo: 'GH9001',
      patientId: 'created-1',
      departmentId: 'dept-1',
      doctorId: 'doc-1',
      registerSort: 'sort-normal',
      visitDate: today,
      status: 'DIAGNOSING',
      registerFee: 0,
      registerCharged: false,
      registeredAt: '2026-10-12T00:00:00Z',
      backOperatorId: null,
      backOperatedAt: null,
    })
    return
  }
  if (req.method === 'POST' && url.pathname === '/api/v1/registers/reg-back/check-in') {
    json(409, {
      success: false,
      data: null,
      error: { code: 'ILLEGAL_STATE_TRANSITION', message: '当前状态不允许该操作', details: {} },
      requestId: 'req-checkin',
    })
    return
  }
  if (req.method === 'POST' && url.pathname === '/api/v1/registers/reg-missing/check-in') {
    json(404, {
      success: false,
      data: null,
      error: { code: 'NOT_FOUND', message: '', details: {} },
      requestId: 'req-miss-checkin',
    })
    return
  }
  if (req.method === 'POST' && url.pathname === '/api/v1/registers/reg-down/check-in') {
    json(500, {
      success: false,
      data: null,
      error: { code: 'HTTP_ERROR', message: '', details: {} },
      requestId: 'req-down',
    })
    return
  }
  json(404, {
    success: false,
    data: null,
    error: { code: 'NOT_FOUND', message: 'missing', details: {} },
    requestId: 'req-miss',
  })
})

await new Promise((resolve) => {
  server.listen(0, '127.0.0.1', resolve)
})
const { port } = server.address()
process.env.VITE_USE_MOCK = 'false'
process.env.VITE_API_BASE_URL = `http://127.0.0.1:${port}/api/v1`

try {
  const real = await createRegister('header.payload.sig', {
    patientId: 'created-1',
    departmentId: 'dept-1',
    doctorId: 'doc-1',
    visitDate: today,
    registerFee: 0,
  })
  assert(real.status === 'NORMAL' && real.registerId === 'reg-1', 'real status')
  const post = calls.find((item) => item.method === 'POST' && item.path === '/api/v1/registers')
  const posted = JSON.parse(post.body)
  assert(posted.patientId === 'created-1', 'post patientId')
  assert(posted.departmentId === 'dept-1', 'post departmentId')
  assert(posted.doctorId === 'doc-1', 'post doctorId')
  assert(posted.visitDate === today, 'post visitDate')
  assert(posted.registerFee === 0, 'post fee 0')
  assert(posted.registerSort === 'sort-normal', 'post registerSort')
  assert(!Object.prototype.hasOwnProperty.call(posted, 'deptCode'), 'post 无 deptCode')
  assert(typeof post.headers['idempotency-key'] === 'string', 'idempotency')

  const rows = await listRegisters('header.payload.sig', { patientId: 'created-1', visitDate: today })
  assert(rows.items.length === 1 && rows.items[0].status === 'USED', 'real list')
  const listCall = calls.find((item) => item.method === 'GET' && item.path === '/api/v1/registers')
  assert(listCall.search.includes('patientId=created-1'), 'list patient')
  assert(listCall.search.includes('visitDate='), 'list date')

  const cancelled = await cancelRegister('header.payload.sig', 'reg-1')
  assert(cancelled.status === 'BACK', 'real cancel')
  const denied = await expectError(() => cancelRegister('header.payload.sig', 'reg-used'), 'REGISTER_NOT_CANCELLABLE', 409)
  assert(denied.message === '当前号已使用或就诊中，不能退号', 'real used copy')
  const again = await expectError(() => cancelRegister('header.payload.sig', 'reg-back'), 'REGISTER_BACK_NOT_ALLOWED', 409)
  assert(again.message === '该号已退，不能再次退号', 'real back copy')

  const checkedIn = await checkInRegister('header.payload.sig', 'reg-used', { simulateConflict: true })
  assert(checkedIn.status === 'DIAGNOSING', 'real check-in')
  const checkInCall = calls.find((item) => item.method === 'POST' && item.path === '/api/v1/registers/reg-used/check-in')
  assert(checkInCall, 'check-in called')
  assert(!checkInCall.body, 'check-in 无请求体')
  assert(checkInCall.headers['idempotency-key'] == null, 'check-in 不带幂等键')
  assert(String(checkInCall.headers.authorization || '').startsWith('Bearer '), 'check-in bearer')
  const illegal = await expectError(() => checkInRegister('header.payload.sig', 'reg-back'), 'ILLEGAL_STATE_TRANSITION', 409)
  assert(illegal.message === '当前状态不允许该操作', 'real 409 copy')
  const missingReal = await expectError(() => checkInRegister('header.payload.sig', 'reg-missing'), 'NOT_FOUND', 404)
  assert(missingReal.message === '未找到挂号单', 'real 404 copy')
  const down = await expectError(() => checkInRegister('header.payload.sig', 'reg-down'), 'HTTP_ERROR', 500)
  assert(down.message === '报到失败，请稍后重试', 'real 500 copy')
  await expectError(() => createRegister('', {
    patientId: 'created-1',
    departmentId: 'dept-1',
    doctorId: 'doc-1',
    visitDate: today,
  }), 'UNAUTHORIZED', 401)
} finally {
  await new Promise((resolve) => server.close(resolve))
}

console.log('register checks passed')
