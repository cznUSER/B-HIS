// 2026-10-12 P1-H05-DISPENSE-UI 发药 Mock。不访问 Go，也不调用 fetch。
import { readFileSync } from 'node:fs'
import { ApiError, messageForCode } from '../src/api/errors.js'
import { mockLogin, resetMockAuth } from '../src/api/mockAuth.js'
import { DISPENSE_ROUTES, confirmDispense, searchPending } from '../src/api/dispenseApi.js'
import {
  DEMO_IDS,
  mockConfirmDispense,
  mockListDispenses,
  mockListTodayPending,
  mockReturnDispense,
  mockSearchPending,
  resetMockDispenses,
} from '../src/api/mockDispenses.js'

function assert(condition, message) {
  if (!condition) throw new Error(message)
}

async function expectError(fn, code, status) {
  try {
    await fn()
  } catch (error) {
    assert(error instanceof ApiError, 'expected ApiError')
    assert(error.code === code, `expected code ${code}, got ${error.code}: ${error.message}`)
    if (status != null) assert(error.status === status, `expected status ${status}, got ${error.status}`)
    return error
  }
  throw new Error(`expected ${code}`)
}

const apiSrc = readFileSync(new URL('../src/api/dispenseApi.js', import.meta.url), 'utf8')
const viewSrc = readFileSync(new URL('../src/views/DispenseView.vue', import.meta.url), 'utf8')
assert(!/\bfetch\s*\(/.test(apiSrc), 'dispense client does not call HTTP')
assert(DISPENSE_ROUTES.collection === '/dispenses', 'create path')
assert(DISPENSE_ROUTES.item('dsp-1') === '/dispenses/dsp-1', 'detail path')
assert(DISPENSE_ROUTES.return('dsp-1') === '/dispenses/dsp-1/return', 'return path')
assert(DISPENSE_ROUTES.orderAlias('ord-1') === '/orders/ord-1/dispense', 'deprecated alias')
assert(apiSrc.includes('VITE_USE_MOCK') || apiSrc.includes('useMock'), 'mock switch')

for (const text of ['请选择已收费患者或处方', '暂无待发药品', '今日无待发', '暂无发药记录', '没有可退明细', '确认发药', '全部发药', '去退费', 'charge_id']) {
  assert(viewSrc.includes(text), `missing copy: ${text}`)
}
for (const text of ['采购', '调拨', '盘点', '批次', '追溯', '换算', '审方', 'admin']) {
  assert(!viewSrc.includes(text), `forbidden copy: ${text}`)
}
assert(messageForCode('NOT_CHARGED') === '未收费，不能发药', 'not charged copy')
assert(messageForCode('ALREADY_FULLY_DISPENSED') === '已全部发完，不能再发', 'full copy')
assert(messageForCode('ALREADY_DISPENSED') === '已全部发完，不能再发', 'openapi alias copy')
assert(messageForCode('INSUFFICIENT_STOCK') === '库存不足，无法收费', 'charge stock copy stays')

process.env.VITE_USE_MOCK = 'true'
resetMockAuth()
resetMockDispenses()
const token = mockLogin({ username: 'system', password: '1' }).accessToken

const unpaid = await expectError(() => mockSearchPending(token, '陈未收'), 'NOT_CHARGED', 409)
assert(unpaid.message === '未收费，不能发药', 'unpaid search')
await expectError(() => mockConfirmDispense(token, {
  items: [{ chargeItemId: DEMO_IDS.unpaid, dispenseQty: 1 }],
}), 'NOT_CHARGED', 409)

const today = mockListTodayPending(token)
const names = today.items.map((item) => item.patientName)
assert(names.includes('王全发') && names.includes('李部分') && names.includes('赵缺货'), 'pending cases')
assert(!names.includes('陈未收') && !names.includes('孙已发'), 'unpaid and finished stay out')
assert(today.items.every((item) => item.itemName !== '诊查费'), 'non-drug filtered')
const fullLine = today.items.find((item) => item.chargeItemId === DEMO_IDS.full)
assert(fullLine.lineDispenseStatus === 'NONE' && fullLine.availableQty === 2 && fullLine.units === '盒', 'full line')
assert(mockSearchPending(token, '王全发').items.length === 1, 'one drug line')
assert(mockSearchPending(token, '孙已发').empty === 'done' && mockSearchPending(token, '孙已发').items.length === 0, 'finished search')

await expectError(() => mockConfirmDispense(token, { items: [] }), 'EMPTY_DISPENSE_ITEMS', 400)
const exceeds = await expectError(() => mockConfirmDispense(token, {
  items: [{ chargeItemId: DEMO_IDS.full, dispenseQty: 3 }],
}), 'DISPENSE_EXCEEDS_AVAILABLE', 409)
assert(exceeds.message === '发药数量超过可发数量', 'exceeds copy')
assert(mockSearchPending(token, '王全发').items[0].stockQty === 20, 'exceeds keeps stock')

const short = await expectError(() => mockConfirmDispense(token, {
  items: [
    { chargeItemId: DEMO_IDS.full, dispenseQty: 2 },
    { chargeItemId: DEMO_IDS.short, dispenseQty: 8 },
  ],
}), 'INSUFFICIENT_STOCK', 409)
assert(short.message === '库存不足，无法发药', 'stock copy')
assert(mockSearchPending(token, '王全发').items[0].stockQty === 20, 'short keeps other stock')
assert(mockSearchPending(token, '赵缺货').items[0].stockQty === 2, 'short keeps own stock')

await expectError(() => mockConfirmDispense(token, {
  items: [{ chargeItemId: DEMO_IDS.treat, dispenseQty: 1 }],
}), 'VALIDATION_ERROR', 400)

const dispensed = mockConfirmDispense(token, {
  orderId: 'ord-full',
  chargeId: 'chg-full',
  items: [{ chargeItemId: DEMO_IDS.full, orderItemId: fullLine.orderItemId, dispenseQty: 2, selected: true }],
  idempotencyKey: 'once-full',
})
assert(dispensed.dispenseNo && dispensed.lines[0].lineDispenseStatus === 'FULL', 'full status')
assert(dispensed.lines[0].stockQty === 18, 'stock deducted')
assert(mockSearchPending(token, '王全发').items.length === 0, 'full leaves pending')
const again = await expectError(() => mockConfirmDispense(token, {
  items: [{ chargeItemId: DEMO_IDS.full, dispenseQty: 1 }],
}), 'ALREADY_FULLY_DISPENSED', 409)
assert(again.message === '已全部发完，不能再发', 'already full copy')
const replay = await expectError(() => mockConfirmDispense(token, {
  items: [{ chargeItemId: DEMO_IDS.part, dispenseQty: 1 }],
  idempotencyKey: 'once-full',
}), 'IDEMPOTENT_REJECT', 409)
assert(replay.message === '重复提交已处理，未再次扣减/回滚', 'idempotent copy')
assert(mockSearchPending(token, '李部分').items[0].stockQty === 30, 'replay keeps stock')

const partial = mockConfirmDispense(token, {
  items: [{ chargeItemId: DEMO_IDS.part, dispenseQty: 2 }],
})
assert(partial.lines[0].lineDispenseStatus === 'PARTIAL' && partial.lines[0].availableQty === 4, 'partial status')
assert(partial.lines[0].stockQty === 28, 'partial stock')
const rest = mockConfirmDispense(token, {
  items: [{ chargeItemId: DEMO_IDS.part, dispenseQty: 4 }],
})
assert(rest.lines[0].lineDispenseStatus === 'FULL' && rest.lines[0].stockQty === 24, 'second dispense fills')

const issued = mockListDispenses(token, { today: true, patient: '孙已发' })
assert(issued.items.some((item) => item.dispenseNo === 'FY-DONE-001' && item.status === 'DONE'), 'issued seed')

await expectError(() => mockReturnDispense(token, {
  dispenseId: DEMO_IDS.doneDispense,
  items: [{ dispenseItemId: DEMO_IDS.doneItem, returnQty: 1 }],
}), 'RETURN_REASON_REQUIRED', 400)
await expectError(() => mockReturnDispense(token, {
  dispenseId: 'missing',
  reason: '患者要求',
  items: [{ chargeItemId: DEMO_IDS.full, returnQty: 1 }],
}), 'NOT_DISPENSED', 409)
const tooMuch = await expectError(() => mockReturnDispense(token, {
  dispenseId: DEMO_IDS.doneDispense,
  reason: '患者要求',
  items: [{ dispenseItemId: DEMO_IDS.doneItem, returnQty: 5 }],
}), 'RETURN_EXCEEDS_DISPENSE', 409)
assert(tooMuch.message === '退药数量超过可退数量', 'return exceeds copy')

const returned = mockReturnDispense(token, {
  dispenseId: DEMO_IDS.doneDispense,
  reason: '患者要求',
  items: [{ dispenseItemId: DEMO_IDS.doneItem, returnQty: 1 }],
})
assert(returned.chargeId === 'chg-done' && returned.stockNote === '库存已回加 1', 'return rolls stock')
assert(returned.lines[0].lineDispenseStatus === 'PARTIAL' && returned.lines[0].availableQty === 1, 'return reopens qty')
assert(returned.lines[0].stockQty === 13, 'stock restored')
const after = mockListDispenses(token, { today: true, dispenseNo: 'FY-DONE-001' })
assert(after.items[0].status === 'PARTIAL_RETURNED', 'slip header after partial return')
const pendingAgain = mockSearchPending(token, '孙已发')
assert(pendingAgain.items.length === 1 && pendingAgain.items[0].lineDispenseStatus === 'PARTIAL', 'returned line is pending')

process.env.VITE_USE_MOCK = 'false'
await expectError(() => searchPending(token, '王全发'), 'NOT_IMPLEMENTED', 501)
await expectError(() => confirmDispense(token, { items: [] }), 'NOT_IMPLEMENTED', 501)
process.env.VITE_USE_MOCK = 'true'

console.log('dispense checks passed')
