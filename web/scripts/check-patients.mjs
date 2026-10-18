// 2026-10-12 P1-H05-PATIENT-UI 建档校验、重复拒绝、查询和真实接口形状。
import { readFileSync } from 'node:fs'
import http from 'node:http'
import { ApiError } from '../src/api/errors.js'
import { mockLogin, resetMockAuth } from '../src/api/mockAuth.js'
import { mockCreatePatient, mockSearchPatients, resetMockPatients } from '../src/api/mockPatients.js'
import { presentPatientError } from '../src/api/patientApi.js'
import { createPatient, fetchPatient, resetPatientClientCache, searchPatients } from '../src/api/patientApi.js'
import {
  FARE_OPTIONS,
  assertCreatePatientRequest,
  assertPatientSearchQuery,
  patientMatchesQuery,
  todayInShanghai,
} from '../src/api/patientContract.js'

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

function tomorrow() {
  const [year, month, day] = todayInShanghai().split('-').map((part) => Number(part))
  const date = new Date(Date.UTC(year, month - 1, day + 1))
  return date.toISOString().slice(0, 10)
}

function sample(overrides = {}) {
  return {
    name: '张三',
    sex: 'M',
    birthday: '1990-01-15',
    fareIdentity: 'SELF',
    idNumber: '110101199001151234',
    phone: '13800000000',
    address: '太原市示范路 1 号',
    ...overrides,
  }
}

process.env.VITE_USE_MOCK = 'true'
resetMockAuth()
resetMockPatients()
resetPatientClientCache()

const token = mockLogin({ username: 'system', password: '1' }).accessToken
const today = todayInShanghai()

await expectError(() => assertCreatePatientRequest(sample({ name: '  ' })), 'VALIDATION_ERROR', 400)
await expectError(() => assertCreatePatientRequest(sample({ sex: '' })), 'VALIDATION_ERROR', 400)
await expectError(() => assertCreatePatientRequest(sample({ birthday: '1900-01-01' })), 'VALIDATION_ERROR', 400)
await expectError(() => assertCreatePatientRequest(sample({ birthday: tomorrow() })), 'VALIDATION_ERROR', 400)
await expectError(() => assertCreatePatientRequest(sample({ birthday: '1990-02-31' })), 'VALIDATION_ERROR', 400)
await expectError(() => assertCreatePatientRequest(sample({ fareIdentity: '' })), 'VALIDATION_ERROR', 400)
await expectError(() => assertCreatePatientRequest(sample({ idNumber: '12' })), 'VALIDATION_ERROR', 400)

const emptyName = await expectError(() => assertCreatePatientRequest(sample({ name: '' })), 'VALIDATION_ERROR', 400)
assert(emptyName.message === '请填写姓名', 'name message')
const badDay = await expectError(() => assertCreatePatientRequest(sample({ birthday: '' })), 'VALIDATION_ERROR', 400)
assert(badDay.message === '出生日期不合法', 'birthday message')
const noFare = await expectError(() => assertCreatePatientRequest(sample({ fareIdentity: ' ' })), 'VALIDATION_ERROR', 400)
assert(noFare.message === '请选择费别', 'fare message')

const body = assertCreatePatientRequest(sample())
assert(body.idType === 'ID_CARD', 'idType')
assert(body.idNumber === '110101199001151234', 'idNumber')
assert(body.fareIdentity === 'SELF', 'fareIdentity')
assert(body.address === '太原市示范路 1 号', 'address is sent when filled')
assert(!Object.prototype.hasOwnProperty.call(body, 'cardNo'), 'card number stays off when empty')
const noAddress = assertCreatePatientRequest(sample({ address: '  ' }))
assert(!Object.prototype.hasOwnProperty.call(noAddress, 'address'), 'blank address omitted')

const noId = assertCreatePatientRequest(sample({ idNumber: '', phone: '' }))
assert(!Object.prototype.hasOwnProperty.call(noId, 'idNumber'), 'empty id omitted')
assert(!Object.prototype.hasOwnProperty.call(noId, 'idType'), 'empty id type omitted')
assert(!Object.prototype.hasOwnProperty.call(noId, 'phone'), 'empty phone omitted')

const detailedFields = {
  cardNo: 'C-20261017',
  memberCardNo: 'M-1001',
  memberLevel: '普通会员',
  contractUnit: '示范单位',
  patientCertificateNo: '病人证-001',
  birthPlace: '山西省太原市',
  ethnicity: '汉族',
  nationality: '中国',
  maritalStatus: '已婚',
  occupationCategory: '专业技术人员',
  occupation: '医生',
  education: '本科',
  workUnit: '示范医院',
  workUnitPhone: '03511234567',
  workUnitPostcode: '030000',
  householdAddress: '山西省太原市小店区',
  householdPhone: '03517654321',
  householdPostcode: '030012',
  contactName: '李四',
  contactAddress: '太原市示范路',
  contactPhone: '13900000000',
  contactRelation: '配偶',
}
const detailedBody = assertCreatePatientRequest(sample(detailedFields))
for (const [key, value] of Object.entries(detailedFields)) {
  assert(detailedBody[key] === value, `detailed field ${key} is sent`)
}
const detailedDto = (await import('../src/api/patientContract.js')).assertPatientDto({ patientId: 'p-1', name: '张三', ...detailedFields })
for (const [key, value] of Object.entries(detailedFields)) {
  assert(detailedDto[key] === value, `detailed dto field ${key}`)
}

const created = mockCreatePatient(token, sample())
assert(created.patientId, 'patientId')
assert(created.patientNo && created.patientNo.startsWith('P'), 'patientNo')
assert(created.hospPkNo && created.hospPkNo.startsWith('H'), 'hospPkNo')
assert(created.cardStatus === 'NOT_SENT', 'card not issued')
assert(created.address === '太原市示范路 1 号', 'dto keeps address')

const duplicate = await expectError(() => mockCreatePatient(token, sample({ name: '张小三' })), 'DUPLICATE_ID_NUMBER', 409)
assert(duplicate.message === '该身份证号已存在，不能重复建档', 'duplicate copy')
assert(duplicate.details.patientId === created.patientId, 'existing id')

const anon = mockCreatePatient(token, sample({ idNumber: '', name: '李四' }))
const anonAgain = mockCreatePatient(token, sample({ idNumber: '', name: '李四二' }))
assert(anon.patientId !== anonAgain.patientId, 'blank id may repeat')

const todayPatient = mockCreatePatient(token, sample({ idNumber: '', name: '今日', birthday: today }))
assert(todayPatient.birthday === today, 'today is allowed')

const fuzzy = mockSearchPatients(token, { name: '张' })
assert(fuzzy.items.length === 1 && fuzzy.items[0].patientId === created.patientId, 'fuzzy name')
const exactId = mockSearchPatients(token, { idNumber: '110101199001151234' })
assert(exactId.items.length === 1, 'exact id')
const prefixId = mockSearchPatients(token, { idNumber: '11010119900115123' })
assert(prefixId.items.length === 0, 'id is not a prefix search')
const byNo = mockSearchPatients(token, { patientNo: created.patientNo })
assert(byNo.items.length === 1 && byNo.items[0].name === '张三', 'exact patientNo')
const byHospNo = mockSearchPatients(token, { hospPkNo: created.hospPkNo })
assert(byHospNo.items.length === 1 && byHospNo.items[0].patientId === created.patientId, 'exact hospPkNo')
const prefixNo = mockSearchPatients(token, { patientNo: created.patientNo.slice(0, -1) })
assert(prefixNo.items.length === 0, 'patientNo is not a prefix search')
assertPatientSearchQuery({ hospPkNo: ' H0001 ' })
assert(patientMatchesQuery({ hospPkNo: 'H0001' }, { hospPkNo: 'H0001' }), 'hospPkNo exact match')
assert(!patientMatchesQuery({ hospPkNo: 'H00010' }, { hospPkNo: 'H0001' }), 'hospPkNo is not a prefix search')
const allMockPatients = mockSearchPatients(token, {})
assert(allMockPatients.items.length >= 2, 'empty patient query returns all mock archives')
assert(allMockPatients.nextOffset === null, 'mock archive list has no extra page')

mockCreatePatient(token, sample({ idNumber: '', cardNo: 'C1', name: '有卡' }))
const cardDup = await expectError(
  () => mockCreatePatient(token, sample({ idNumber: '', cardNo: 'C1', name: '另一人' })),
  'DUPLICATE_CARD_NO',
  409,
)
assert(cardDup.message === '卡号已存在，不能重复建档', 'card copy')
assert(FARE_OPTIONS.some((item) => item.value === 'SELF' && item.label === '自费'), 'self pay label')

const viaApi = await createPatient(token, sample({ idNumber: '', name: '王五', address: '太原市并州路' }))
assert(viaApi.address === '太原市并州路', 'api mock returns address')
const found = await searchPatients(token, { name: '王' })
assert(found.items.some((item) => item.patientId === viaApi.patientId), 'api fuzzy search')
const loaded = await fetchPatient(token, viaApi.patientId)
assert(loaded.name === '王五', 'api get')

const view = readFileSync(new URL('../src/views/PatientView.vue', import.meta.url), 'utf8')
const template = view.split('<template>')[1]
for (const text of ['姓名', '性别', '出生日期', '费别', '证件号码', '手机号', '地址', '卡号', '会员卡号', '会员等级', '合同单位', '病人证号', '出生地', '民族', '国籍', '婚姻状况', '职业分类', '职业', '学历', '工作单位', '邮政编码', '户口地址', '联系人', '关系', '在患者查询中查看已有档案', '建档成功', '患者号', '去挂号']) {
  assert(template.includes(text), `missing copy: ${text}`)
}
for (const text of ['patient-create-layout', 'patient-summary', 'patient-mode-switch', '简易建档', '详细建档', 'patient-detail-fields']) {
  assert(template.includes(text), `missing patient registration layout: ${text}`)
}
const summaryIndex = template.indexOf('data-testid="patient-summary"')
const formIndex = template.indexOf('data-testid="patient-form"')
assert(summaryIndex >= 0 && summaryIndex < formIndex, 'left summary precedes the create form')
assert(template.includes('v-if="detailMode" class="clinical-section" data-testid="patient-detail-fields"'), 'detail fields are mode-gated')
assert(template.includes('data-testid="patient-simple-fields"'), 'simple fields have a mode boundary')
assert(template.includes('v-model="form.address"'), 'address remains available in detailed mode')
const simpleFields = template.slice(template.indexOf('data-testid="patient-simple-fields"'), template.indexOf('data-testid="patient-detail-fields"'))
assert(!simpleFields.includes('患者地址') && !simpleFields.includes('patient-address'), 'address is not part of simple registration')
assert(view.includes('const detailMode = ref(false)'), 'simple mode is the default')
assert(!template.includes('patient-search') && !template.includes('searchPatients'), '建档页不含患者查询')
const searchView = readFileSync(new URL('../src/views/PatientSearchView.vue', import.meta.url), 'utf8')
for (const text of ['患者查询', '全部患者档案', 'searchPatients', 'fetchPatient', '暂无患者档案']) assert(searchView.includes(text), `search page ${text}`)
const apiSrc = readFileSync(new URL('../src/api/patientApi.js', import.meta.url), 'utf8')
assert(apiSrc.includes('该身份证号已存在，不能重复建档'), 'duplicate copy is fixed')
assert(apiSrc.includes('DUPLICATE_ID_NUMBER'), 'real duplicate code')
assert(apiSrc.includes('卡号已存在，不能重复建档'), 'card copy is fixed')
const aliased = presentPatientError({ code: 'DUPLICATE_PATIENT', status: 409, message: 'duplicate', details: null, requestId: 'r' })
assert(aliased.code === 'DUPLICATE_PATIENT' && aliased.message === '该身份证号已存在，不能重复建档', 'legacy code')
for (const text of ['发卡', '充值', '患者来源', '首诊症状', '介绍人', '信用额度', '支付方式', '收据号', '合并']) {
  assert(!template.includes(text), `forbidden copy: ${text}`)
}
assert(readFileSync(new URL('../src/router/index.js', import.meta.url), 'utf8').includes('OutpatientModuleView'), 'router')
assert(
  readFileSync(new URL('../src/views/ModulePlaceholderView.vue', import.meta.url), 'utf8').includes('接口就绪后开放'),
  'other modules stay empty',
)

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
  if (req.method === 'POST' && url.pathname === '/api/v1/patients') {
    const payload = JSON.parse(raw || '{}')
    if (payload.idNumber === '110101199001151234') {
      json(409, {
        success: false,
        data: null,
        error: {
          code: 'DUPLICATE_ID_NUMBER',
          message: 'duplicate',
          details: { fields: ['idNumber'] },
        },
        requestId: 'req-dup',
      })
      return
    }
    json(201, {
      ...payload,
      patientId: 'created-1',
      hospPkNo: 'H0001',
      patientNo: 'P0001',
      name: payload.name,
      sex: payload.sex,
      birthday: payload.birthday,
      idNumber: payload.idNumber || '',
      phone: payload.phone || '',
      address: payload.address || '',
      fareIdentity: payload.fareIdentity,
      cardStatus: 'NOT_SENT',
      status: 'ACTIVE',
      createdAt: '2026-10-12T00:00:00Z',
    })
    return
  }
  if (req.method === 'GET' && url.pathname === '/api/v1/patients') {
    const payload = {
      items: [
        {
          patientId: 'created-1',
          hospPkNo: 'H0001',
          patientNo: 'P0001',
          name: '张三',
          sex: 'M',
          birthday: '1990-01-15',
          idNumber: '110101199001151234',
          fareIdentity: 'SELF',
        },
        {
          patientId: 'other-1',
          hospPkNo: 'H0002',
          patientNo: 'P0002',
          name: '李四',
          sex: 'F',
          birthday: '1992-02-02',
          idNumber: '110101199202022222',
          fareIdentity: 'SELF',
        },
      ],
    }
    if (!url.searchParams.has('offset')) payload.nextOffset = 100
    json(200, payload)
    return
  }
  if (req.method === 'GET' && url.pathname === '/api/v1/patients/created-1') {
    json(200, {
      patientId: 'created-1',
      patientNo: 'P0001',
      name: '张三',
      sex: 'M',
      birthday: '1990-01-15',
      fareIdentity: 'SELF',
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
resetPatientClientCache()

try {
  const real = await createPatient('header.payload.sig', sample({ ...detailedFields, idNumber: '', name: '赵六', address: '太原市并州路' }))
  assert(real.patientNo === 'P0001', 'real patientNo')
  assert(real.patientId === 'created-1', 'real patientId')
  assert(real.address === '太原市并州路', 'real address')
  assert(real.memberCardNo === detailedFields.memberCardNo && real.householdPostcode === detailedFields.householdPostcode, 'real response keeps detailed patient fields')
  const post = calls.find((item) => item.method === 'POST')
  const posted = JSON.parse(post.body)
  assert(posted.address === '太原市并州路', 'post sends address')
  assert(posted.cardNo === detailedFields.cardNo, 'post sends card')
  assert(posted.memberCardNo === detailedFields.memberCardNo && posted.householdPostcode === detailedFields.householdPostcode, 'post sends detailed patient fields')
  assert(posted.fareIdentity === 'SELF', 'post fare')
  assert(posted.name === '赵六', 'post name')
  assert(typeof post.headers['idempotency-key'] === 'string' && post.headers['idempotency-key'].length > 10, 'idempotency')

  const dup = await expectError(
    () => createPatient('header.payload.sig', sample()),
    'DUPLICATE_ID_NUMBER',
    409,
  )
  assert(dup.message === '该身份证号已存在，不能重复建档', 'real duplicate copy')
  assert(Array.isArray(dup.details.fields) && dup.details.fields.includes('idNumber'), 'duplicate field')
  assert(dup.requestId === 'req-dup', 'request id kept')

  resetPatientClientCache()
  calls.length = 0
  const allReal = await searchPatients('header.payload.sig', {})
  assert(allReal.items.length === 2, 'empty patient query calls real API')
  assert(allReal.nextOffset === 100, 'real patient list exposes the next page')
  const allCall = calls.find((item) => item.method === 'GET' && item.path === '/api/v1/patients')
  assert(allCall && allCall.search === '', 'empty patient query sends no filters')

  resetPatientClientCache()
  calls.length = 0
  const nextReal = await searchPatients('header.payload.sig', {}, { offset: 100 })
  assert(nextReal.items.length === 2, 'real patient list loads another page')
  const nextCall = calls.find((item) => item.method === 'GET' && item.path === '/api/v1/patients')
  assert(nextCall && new URLSearchParams(nextCall.search).get('offset') === '100', 'next offset is sent exactly')

  resetPatientClientCache()
  calls.length = 0
  const byPatientNo = await searchPatients('header.payload.sig', { patientNo: 'P0001' })
  assert(byPatientNo.items.length === 1 && byPatientNo.items[0].patientId === 'created-1', 'real patientNo search')
  const patientNoCall = calls.find((item) => item.method === 'GET' && item.path === '/api/v1/patients')
  assert(patientNoCall && new URLSearchParams(patientNoCall.search).get('patientNo') === 'P0001', 'patientNo query is sent exactly')
  assert([...new URLSearchParams(patientNoCall.search).keys()].length === 1, 'patientNo search sends no other filters')

  resetPatientClientCache()
  calls.length = 0
  const byName = await searchPatients('header.payload.sig', { name: '张' })
  assert(byName.items.length === 1 && byName.items[0].name === '张三', 'client keeps fuzzy name')
  const nameCall = calls.find((item) => item.method === 'GET' && item.path === '/api/v1/patients')
  assert(nameCall.search.includes('name='), 'name query')
  assert(!nameCall.search.includes('patientNo'), 'no invented patientNo param')
  assert(!nameCall.search.includes('hospPkNo'), 'no invented hospPkNo param')
  assert(!nameCall.search.includes('address'), 'no address query')

  resetPatientClientCache()
  calls.length = 0
  const byHospPkNo = await searchPatients('header.payload.sig', { hospPkNo: 'H0001' })
  assert(byHospPkNo.items.length === 1 && byHospPkNo.items[0].patientId === 'created-1', 'real hospPkNo search')
  const hospPkNoCall = calls.find((item) => item.method === 'GET' && item.path === '/api/v1/patients')
  assert(hospPkNoCall && new URLSearchParams(hospPkNoCall.search).get('hospPkNo') === 'H0001', 'hospPkNo query is sent exactly')
  assert([...new URLSearchParams(hospPkNoCall.search).keys()].length === 1, 'hospPkNo search sends no other filters')

  const one = await fetchPatient('header.payload.sig', 'created-1')
  assert(one.patientNo === 'P0001', 'real get')
  await expectError(() => createPatient('', sample({ idNumber: '' })), 'UNAUTHORIZED', 401)
} finally {
  await new Promise((resolve) => server.close(resolve))
}

console.log('patient checks passed')
