// P2-1 refund workspace contract: the existing dispense/charge flows must be reachable from one entry.
import { readFileSync } from 'node:fs'
import { OUTPATIENT_MODULES } from '../src/workbench/catalog.js'

function assert(condition, message) {
  if (!condition) throw new Error(message)
}

const refund = OUTPATIENT_MODULES.find((item) => item.key === 'refund')
assert(refund && refund.status === 'available', 'refund module is available')
assert(refund.roles.join(',') === 'CASHIER,PHARMACIST,SYSTEM', 'refund roles')

const moduleView = readFileSync(new URL('../src/views/OutpatientModuleView.vue', import.meta.url), 'utf8')
assert(moduleView.includes("import RefundView from './RefundView.vue'"), 'refund view import')
assert(moduleView.includes("allowed('refund')"), 'refund permission gate')
assert(moduleView.includes('<RefundView v-else-if="showRefund" />'), 'refund view mount')

const refundView = readFileSync(new URL('../src/views/RefundView.vue', import.meta.url), 'utf8')
for (const text of ['退药/退费工作区', '退药', '退费', 'DispenseView', 'ChargeView', 'charge_id']) {
  assert(refundView.includes(text), `refund workspace copy: ${text}`)
}
assert(refundView.includes("view: 'issued'"), 'issued deep link')
assert(refundView.includes("view: 'records'"), 'refund deep link')
assert(refundView.includes('<DispenseView v-if="activeTab === \'return\' && canReturn" return-only />'), 'return-only component mount')
assert(refundView.includes('<ChargeView v-else-if="canRefund" refund-only />'), 'refund-only component mount')
assert(refundView.includes("hasEffectivePermission(session.user, ['dispense:create', 'dispense:return'])"), 'return permission controls the tab')
assert(refundView.includes("hasEffectivePermission(session.user, ['charge:refund'])"), 'refund permission controls the tab')
assert(refundView.includes('含优惠收费单暂不支持部分退费'), 'discount refund scope is shown')
const mockAuth = readFileSync(new URL('../src/api/mockAuth.js', import.meta.url), 'utf8')
assert(mockAuth.includes("permissions.add('admin:*')"), 'demo admin receives access permissions')
assert(mockAuth.includes("permissions.add('charge:refund')"), 'demo cashier receives refund permission')
assert(mockAuth.includes("permissions.add('dispense:return')"), 'demo pharmacist receives return permission')

const dispenseView = readFileSync(new URL('../src/views/DispenseView.vue', import.meta.url), 'utf8')
assert(dispenseView.includes('returnOnly'), 'return-only mode')
assert(dispenseView.includes("props.returnOnly || route.query.view === 'issued' ? 'issued' : 'pending'"), 'return-only defaults to issued list')
assert(dispenseView.includes("props.returnOnly ? 'refund' : 'charge'"), 'return-only continues into refund workspace')

const chargeView = readFileSync(new URL('../src/views/ChargeView.vue', import.meta.url), 'utf8')
assert(chargeView.includes('refundOnly'), 'refund-only mode')
assert(chargeView.includes("refundOnly ? 'refund' : 'dispense'"), 'refund-only returns to return workspace')
assert(chargeView.includes('discountedPartialRefund'), 'discounted partial refund is blocked in the UI')

console.log('refund workspace checks passed')
