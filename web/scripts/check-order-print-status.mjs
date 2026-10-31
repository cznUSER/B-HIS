import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

import {
  PLACEHOLDER_HONESTY,
  SESSION_PERSISTENCE_NOTE,
  PRINT_STATUS,
  buildPreviewModel,
  canPreviewOrderStatus,
  createOrderPrintController,
  printStatusLabel,
} from '../src/workbench/orderPrintStatus.js'

const here = path.dirname(fileURLToPath(import.meta.url))
const root = path.resolve(here, '..')

assert.equal(canPreviewOrderStatus('COMMIT'), true)
assert.equal(canPreviewOrderStatus('PROC'), true)
assert.equal(canPreviewOrderStatus('SAVE'), false)
assert.equal(canPreviewOrderStatus('ABATE'), false)
assert.match(printStatusLabel('PRINTING'), /输出|打印|中/)

assert.ok(PLACEHOLDER_HONESTY.includes('未对接打印服务'))
assert.ok(PLACEHOLDER_HONESTY.includes('未回写服务端打印流水'))
assert.equal(PLACEHOLDER_HONESTY.includes('服务端已打印'), false)
assert.equal(PLACEHOLDER_HONESTY.includes('已对接打印服务'), false)
assert.ok(SESSION_PERSISTENCE_NOTE.includes('未持久化') || SESSION_PERSISTENCE_NOTE.includes('重置') || SESSION_PERSISTENCE_NOTE.includes('刷新'))

const registerA = {
  patientId: 'p-a',
  patientName: '甲患者',
  patientNo: 'PA001',
  registerId: 'reg-a',
  registerNo: 'R-A',
}
const registerB = {
  patientId: 'p-b',
  patientName: '乙患者',
  patientNo: 'PB002',
  registerId: 'reg-b',
  registerNo: 'R-B',
}

const modelA = buildPreviewModel({
  register: registerA,
  orderMeta: {
    patientId: 'p-a',
    orderId: 'ord-a',
    orderNo: 'O-A',
    status: 'COMMIT',
    diagnoseName: '上感',
  },
  lines: [{ itemNo: 'DRUG-A', itemName: '阿莫西林', amount: 1, units: '盒', usageText: '口服', frequencyText: '一日三次' }],
})
assert.equal(modelA.patientId, 'p-a')
assert.equal(modelA.orderNo, 'O-A')
assert.equal(modelA.registerNo, 'R-A')
assert.equal(modelA.lines.length, 1)
assert.equal(modelA.lines[0].itemName, '阿莫西林')

assert.throws(
  () =>
    buildPreviewModel({
      register: registerB,
      orderMeta: { patientId: 'p-a', orderId: 'ord-x', orderNo: 'O-X', status: 'COMMIT' },
      lines: [],
    }),
  /串患|不一致/,
)

const modelB = buildPreviewModel({
  register: registerB,
  orderMeta: { patientId: 'p-b', orderId: 'ord-b', orderNo: 'O-B', status: 'COMMIT' },
  lines: [{ itemNo: 'DRUG-B', itemName: '布洛芬', amount: 2 }],
})
assert.equal(modelB.patientId, 'p-b')
assert.notEqual(modelA.patientId, modelB.patientId)

const ctrl = createOrderPrintController()
assert.equal(ctrl.status, PRINT_STATUS.IDLE)
assert.equal(ctrl.isLocked, false)

const orderStatusSnapshot = 'COMMIT'
assert.equal(ctrl.openPreview(modelA), true)
assert.equal(ctrl.status, PRINT_STATUS.PREVIEW)
assert.equal(ctrl.orderStatusSnapshot, orderStatusSnapshot)
assert.equal(ctrl.isLocked, false)

assert.equal(ctrl.startOutput(), true)
assert.equal(ctrl.status, PRINT_STATUS.PRINTING)
assert.equal(ctrl.isLocked, true)
assert.equal(ctrl.openPreview(modelA), false)

assert.equal(ctrl.confirmPrinted('2026-10-31T10:00:00+08:00'), true)
assert.equal(ctrl.status, PRINT_STATUS.PRINTED)
assert.equal(ctrl.lastPrintedAt, '2026-10-31T10:00:00+08:00')
assert.equal(ctrl.orderStatusSnapshot, 'COMMIT')
assert.equal(ctrl.isLocked, false)

assert.equal(ctrl.reprint(), true)
assert.equal(ctrl.status, PRINT_STATUS.PRINTING)
assert.equal(ctrl.orderStatusSnapshot, 'COMMIT')
assert.equal(ctrl.fail('本机占位失败'), true)
assert.equal(ctrl.status, PRINT_STATUS.FAILED)
assert.match(ctrl.errorMessage, /失败/)

assert.equal(ctrl.retry(), true)
assert.equal(ctrl.status, PRINT_STATUS.PRINTING)
assert.equal(ctrl.confirmPrinted('2026-10-31T10:05:00+08:00'), true)
assert.equal(ctrl.status, PRINT_STATUS.PRINTED)
assert.equal(ctrl.orderStatusSnapshot, 'COMMIT')

ctrl.resetToIdle()
assert.equal(ctrl.status, PRINT_STATUS.IDLE)
assert.equal(ctrl.previewModel, null)
assert.ok(ctrl.sessionPersistenceNote)

const orderView = fs.readFileSync(path.join(root, 'src/views/OrderView.vue'), 'utf8')
assert.match(orderView, /OrderPrintPanel/)
assert.match(orderView, /orderPrintStatus/)
assert.match(orderView, /PLACEHOLDER_HONESTY/)
assert.match(orderView, /recordAction\(['"]打印['"]|recordAction\(label/)
assert.equal(orderView.includes('已对接打印服务'), false)
assert.equal(orderView.includes('服务端已打印'), false)
assert.equal(orderView.includes('已回写服务端打印流水'), false)

const panel = fs.readFileSync(path.join(root, 'src/components/outpatient/OrderPrintPanel.vue'), 'utf8')
assert.match(panel, /data-testid="order-print-panel"/)
assert.match(panel, /data-testid="order-print-status"/)
assert.match(panel, /data-testid="order-print-honesty"/)
assert.match(panel, /data-testid="order-print-retry-btn"/)
assert.match(panel, /data-testid="order-print-reprint-btn"/)
assert.equal(panel.includes('已对接打印服务'), false)
assert.equal(panel.includes('服务端已打印'), false)

console.log('P1_PRINT_STATUS_OK')