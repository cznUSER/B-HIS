import { readFileSync } from 'node:fs'
import {
  buildUserOperationsCsv,
  fetchUserOperationLogs,
  normalizeUserOperationsResponse,
} from '../src/master/accessAuditApi.js'

function assert(condition, message) {
  if (!condition) throw new Error(message)
}

const component = readFileSync(new URL('../src/views/access/UserOperationsLogPanel.vue', import.meta.url), 'utf8')
assert(component.includes('用户操作日志') && component.includes('导出已加载'), 'log panel search/list/export controls')
for (const field of ['moduleId', 'userId', 'begin', 'end', 'module', 'type', 'time', 'user', 'department', 'description', 'ip', 'mac', 'version', 'requestId', 'errorCode', 'outcome']) {
  assert(component.includes(field), `log panel column ${field}`)
}

const fixture = {
  items: [{
    moduleId: '1',
    moduleName: '门诊',
    logType: 'USER_RESET_PASSWORD',
    occurredAt: '2026-10-14T02:30:00+08:00',
    actorUserId: 'operator-1',
    actorUsername: 'operator',
    departmentId: '挂号室',
    description: '重置账号密码',
    clientIp: '127.0.0.1',
    clientMac: '00:00:00:00:00:01',
    buildVersion: '1.0.0',
    requestId: 'REQ-1',
    errorCode: '',
    outcome: 'SUCCESS',
  }],
  nextCursor: 'cursor-next',
}
assert(normalizeUserOperationsResponse(fixture).nextCursor === 'cursor-next', 'response cursor contract')
assert(normalizeUserOperationsResponse(fixture).items[0].user === 'operator', 'response item contract')
assert(normalizeUserOperationsResponse(fixture).items[0].userId === 'operator-1' && normalizeUserOperationsResponse(fixture).items[0].department === '挂号室', 'backend audit fields map to display rows')
for (const invalid of [null, {}, { items: {} }, { items: [{}] }, { items: [], nextCursor: 42 }]) {
  let rejected = false
  try { normalizeUserOperationsResponse(invalid) } catch { rejected = true }
  assert(rejected, 'malformed response rejected')
}

const previousFetch = globalThis.fetch
let request
globalThis.fetch = async (url, init) => {
  request = { url: String(url), init }
  return new Response(JSON.stringify(fixture), { status: 200, headers: { 'Content-Type': 'application/json' } })
}
process.env.VITE_USE_MOCK = 'false'
const result = await fetchUserOperationLogs('test-token', {
  moduleId: '1', userId: 'U 1', begin: '2026-10-01', end: '2026-10-14', cursor: 'cursor/a', limit: 25,
})
assert(request.url === '/api/v1/audit/user-operations?moduleId=1&userId=U+1&begin=2026-10-01&end=2026-10-14&cursor=cursor%2Fa&limit=25', 'live audit route and query encoding')
assert(request.init.method === 'GET' && request.init.headers.Authorization === 'Bearer test-token', 'live audit request headers')
assert(result.items.length === 1 && result.nextCursor === 'cursor-next', 'live audit response')
const csv = buildUserOperationsCsv([{
  module: '门诊', type: 'USER_UPDATE', time: '2026-10-14T02:30:00+08:00', user: '=HYPERLINK("x")',
  department: '内科', description: 'update,profile', ip: '+cmd|x', mac: 'line\nbreak', version: '1.0.0',
  outcome: 'SUCCESS', requestId: 'REQ-1', errorCode: '',
}])
assert(csv.startsWith('\uFEFF"模块","类型","时间","用户","科室","说明","IP","MAC","版本","结果","请求编号","错误代码"\r\n'), 'CSV heading and UTF-8 BOM')
assert(csv.includes('"\'=HYPERLINK(""x"")"') && csv.includes('"update,profile"') && csv.includes('"line\nbreak"'), 'CSV quote and spreadsheet formula escaping')
assert(csv.includes("'+cmd|x"), 'CSV formula-like values are neutralized')

globalThis.fetch = previousFetch
process.env.VITE_USE_MOCK = 'true'
const mock = await fetchUserOperationLogs('', { userId: 'mock-operator', limit: 2 })
assert(mock.items.length > 0 && mock.items.length <= 2, 'mock-compatible filtered page')
const loadedCsv = buildUserOperationsCsv(mock.items)
assert(loadedCsv.includes('用户重置密码') && !loadedCsv.includes('患者建档'), 'CSV exports only the currently loaded rows')
console.log('user operations log checks passed')
