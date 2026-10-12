// 2026-10-12 P1-H05-REGISTER-UI 挂号校验、退号规则、Mock 与真实接口形状。
import { readFileSync } from 'node:fs'
import http from 'node:http'
import { ApiError } from '../src/api/errors.js'
import { mockLogin, resetMockAuth } from '../src/api/mockAuth.js'
import { mockCreatePatient, resetMockPatients } from '../src/api/mockPatients.js'
import { resetPatientClientCache } from '../src/api/patientApi.js'
import { todayInShanghai } from '../src/api/patientContract.js'
import { assertCreateEncounterRequest } from '../src/api/encounterContract.js'
import { cancelEncounter, createEncounter, listEncounters, presentEncounterError } from '../src/api/encounterApi.js'
import {
  mockCancelEncounter,
  mockCreateEncounter,
  mockListEncounters,
  resetMockEncounters,
} from '../src/api/mockEncounters.js'
import {
  cancelRejectMessage,
  canCancel,
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
resetMockEncounters()
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
assert(statusLabel('NORMAL') === '正常号', '正常号文案')
assert(statusLabel('BACK') === '退号', '退号文案')
assert(cancelRejectMessage('USED') === '当前号已使用或就诊中，不能退号', '已用')
assert(cancelRejectMessage('DIAGNOSING') === '当前号已使用或就诊中，不能退号', '就诊中')
assert(cancelRejectMessage('DIAGNOSED') === '当前号已使用或就诊中，不能退号', '就诊完成')
assert(cancelRejectMessage('PAUSED') === '当前号已使用或就诊中，不能退号', '挂起')
assert(cancelRejectMessage('BACK') === '该号已退，不能再次退号', '已退')

const noPatient = await expectError(() => assertCreateEncounterRequest({}), 'VALIDATION_ERROR', 400)
assert(noPatient.message === '请先选择或建档患者', '缺患者')
const noDept = await expectError(
  () => assertCreateEncounterRequest({ patientId: 'p', visitDate: today, doctorId: 'doc-1' }),
  'VALIDATION_ERROR',
  400,
)
assert(noDept.message === '请选择科室', '缺科室')
const noDoctor = await expectError(
  () => assertCreateEncounterRequest({ patientId: 'p', deptCode: 'dept-1', visitDate: today }),
  'VALIDATION_ERROR',
  400,
)
assert(noDoctor.message === '请选择医生', '缺医生')
const wrongDoctor = await expectError(
  () => assertCreateEncounterRequest({ patientId: 'p', deptCode: 'dept-1', doctorId: 'doc-2', visitDate: today }),
  'VALIDATION_ERROR',
  400,
)
assert(wrongDoctor.message === '请选择医生', '医生须属科室')

const payload = assertCreateEncounterRequest({
  patientId: 'p',
  deptCode: 'dept-1',
  doctorId: 'doc-1',
  visitDate: today,
  registerFee: 0,
})
for (const key of ['patientId', 'deptCode', 'doctorId', 'visitDate']) {
  assert(Object.prototype.hasOwnProperty.call(payload, key), `payload ${key}`)
}
assert(payload.clinicSort === 'sort-normal', '号别占位')
assert(payload.registerFee === 0, '费用默认 0')
assert(!Object.prototype.hasOwnProperty.call(payload, 'schedule'), '无排班字段')

const patient = mockCreatePatient(token, {
  name: '王小明',
  sex: 'M',
  birthday: '1990-05-16',
  fareIdentity: 'SELF',
  idNumber: '110101199005161234',
})

const created = mockCreateEncounter(token, {
  patientId: patient.patientId,
  deptCode: 'dept-2',
  doctorId: 'doc-2',
  visitDate: today,
  registerFee: 0,
})
assert(created.status === 'NORMAL', '保存后正常号')
assert(created.registerFee === 0 && created.registerCharged === false, '费用隐藏默认')
assert(created.deptCode === 'dept-2' && created.doctorId === 'doc-2', '科室医生写入')
assert(created.patientId === patient.patientId, '患者写入')
assert(created.visitDate === today, '就诊日写入')

const listed = mockListEncounters(token, { patientId: patient.patientId, visitDate: today })
const fresh = listed.items.find((item) => item.encounterId === created.encounterId)
assert(fresh && statusLabel(fresh.status) === '正常号', '列表可见正常号')
const seedIds = listed.items.filter((item) => item.encounterId !== created.encounterId).map((item) => item.encounterId)
assert(new Set(seedIds).size === seedIds.length, '演示号单号互不相同')
for (const status of ['USED', 'DIAGNOSING', 'DIAGNOSED', 'PAUSED', 'BACK']) {
  const row = listed.items.find((item) => item.status === status)
  assert(row, `列表含 ${status}`)
  const rejected = await expectError(() => mockCancelEncounter(token, row.encounterId), 'REGISTER_BACK_NOT_ALLOWED', 409)
  assert(rejected.message === cancelRejectMessage(status), `拒绝文案 ${status}`)
}
const again = mockListEncounters(token, { patientId: patient.patientId, visitDate: today })
for (const status of ['USED', 'DIAGNOSING', 'DIAGNOSED', 'PAUSED', 'BACK']) {
  assert(again.items.some((item) => item.status === status), `${status} 状态不变`)
}

const backed = mockCancelEncounter(token, created.encounterId)
assert(backed.status === 'BACK', '退号成功')
assert(backed.backOperatorId && backed.backOperatedAt, '留下退号操作')
const twice = await expectError(() => mockCancelEncounter(token, created.encounterId), 'REGISTER_BACK_NOT_ALLOWED', 409)
assert(twice.message === '该号已退，不能再次退号', '不可再退')

const viaApi = await createEncounter(token, {
  patientId: patient.patientId,
  deptCode: 'dept-1',
  doctorId: 'doc-1',
  visitDate: today,
})
assert(viaApi.status === 'NORMAL', 'api mock 正常号')
const apiList = await listEncounters(token, { patientId: patient.patientId, visitDate: today })
assert(apiList.items.some((item) => item.encounterId === viaApi.encounterId && item.status === 'NORMAL'), 'api 列表')
const apiBack = await cancelEncounter(token, viaApi.encounterId)
assert(apiBack.status === 'BACK', 'api 退号')

const view = readFileSync(new URL('../src/views/RegisterView.vue', import.meta.url), 'utf8')
const template = view.split('<template>')[1]
for (const text of ['请先选择或建档患者', '请选择科室', '请选择医生', '普通号', '去建档', '回建档页', '退号', '姓名', '性别', '出生日期', '患者号', '身份证']) {
  assert(template.includes(text), `missing copy: ${text}`)
}
assert(!template.includes('type="date"'), '无日期控件')
assert(!template.includes('排班'), '无排班文案')
assert(!template.includes('挂号费'), '费用不展示')
assert(!template.toLowerCase().includes('admin'), '无 admin')
assert(view.includes('registerFee: 0'), '提交费用为 0')
const apiSrc = readFileSync(new URL('../src/api/encounterApi.js', import.meta.url), 'utf8')
assert(apiSrc.includes("apiUrl('/encounters')"), 'create path')
assert(apiSrc.includes('/cancel'), 'cancel path')
assert(apiSrc.includes('useMock()'), 'mock switch')
assert(readFileSync(new URL('../src/views/OutpatientModuleView.vue', import.meta.url), 'utf8').includes('RegisterView'), 'route shell')

const english = presentEncounterError({
  code: 'REGISTER_BACK_NOT_ALLOWED',
  status: 409,
  message: 'not allowed',
  details: null,
  requestId: 'r',
})
assert(english.message === '当前号已使用或就诊中，不能退号', '英文拒绝改成中文')

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
  if (req.method === 'POST' && url.pathname === '/api/v1/encounters') {
    const payload = JSON.parse(raw || '{}')
    json(201, {
      encounterId: 'enc-1',
      encounterNo: 'GH0001',
      patientId: payload.patientId,
      deptCode: payload.deptCode,
      doctorId: payload.doctorId,
      clinicSort: payload.clinicSort,
      visitDate: payload.visitDate,
      status: 'NORMAL',
      registerFee: payload.registerFee,
      registerCharged: false,
      registeredAt: '2026-10-12T01:00:00Z',
    })
    return
  }
  if (req.method === 'GET' && url.pathname === '/api/v1/encounters') {
    json(200, {
      items: [
        {
          encounterId: 'enc-used',
          encounterNo: 'GH9001',
          patientId: 'created-1',
          deptCode: 'dept-1',
          doctorId: 'doc-1',
          visitDate: today,
          status: 'USED',
          registerFee: 0,
          registerCharged: false,
        },
      ],
    })
    return
  }
  if (req.method === 'POST' && url.pathname === '/api/v1/encounters/enc-1/cancel') {
    json(200, {
      encounterId: 'enc-1',
      encounterNo: 'GH0001',
      patientId: 'created-1',
      deptCode: 'dept-1',
      doctorId: 'doc-1',
      visitDate: today,
      status: 'BACK',
      registerFee: 0,
      registerCharged: false,
      backOperatorId: 'u-system',
      backOperatedAt: '2026-10-12T02:00:00Z',
    })
    return
  }
  if (req.method === 'POST' && url.pathname === '/api/v1/encounters/enc-used/cancel') {
    json(409, {
      success: false,
      data: null,
      error: { code: 'REGISTER_BACK_NOT_ALLOWED', message: 'not allowed', details: {} },
      requestId: 'req-back',
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
  const real = await createEncounter('header.payload.sig', {
    patientId: 'created-1',
    deptCode: 'dept-1',
    doctorId: 'doc-1',
    visitDate: today,
    registerFee: 0,
  })
  assert(real.status === 'NORMAL', 'real status')
  assert(real.encounterId === 'enc-1', 'real id')
  const post = calls.find((item) => item.method === 'POST' && item.path === '/api/v1/encounters')
  const posted = JSON.parse(post.body)
  assert(posted.patientId === 'created-1', 'post patientId')
  assert(posted.deptCode === 'dept-1', 'post deptCode')
  assert(posted.doctorId === 'doc-1', 'post doctorId')
  assert(posted.visitDate === today, 'post visitDate')
  assert(posted.registerFee === 0, 'post fee 0')
  assert(posted.clinicSort === 'sort-normal', 'post clinic sort')
  assert(typeof post.headers['idempotency-key'] === 'string', 'idempotency')

  const rows = await listEncounters('header.payload.sig', { patientId: 'created-1', visitDate: today })
  assert(rows.items.length === 1 && rows.items[0].status === 'USED', 'real list')
  const listCall = calls.find((item) => item.method === 'GET')
  assert(listCall.search.includes('patientId=created-1'), 'list patient')
  assert(listCall.search.includes('visitDate='), 'list date')

  const cancelled = await cancelEncounter('header.payload.sig', 'enc-1')
  assert(cancelled.status === 'BACK', 'real cancel')
  const denied = await expectError(() => cancelEncounter('header.payload.sig', 'enc-used'), 'REGISTER_BACK_NOT_ALLOWED', 409)
  assert(denied.message === '当前号已使用或就诊中，不能退号', 'real reject copy')
  await expectError(() => createEncounter('', {
    patientId: 'created-1',
    deptCode: 'dept-1',
    doctorId: 'doc-1',
    visitDate: today,
  }), 'UNAUTHORIZED', 401)
} finally {
  await new Promise((resolve) => server.close(resolve))
}

console.log('register checks passed')
