import assert from 'node:assert/strict'
import {
  clearOutpatientContext,
  deriveWorkflowStatus,
  deriveWorkflowProgress,
  outpatientContext,
  setPatient,
  setRegister,
} from '../src/workbench/outpatientContext.js'

clearOutpatientContext()

assert.equal(deriveWorkflowStatus({ patient: null }).key, 'PATIENT_REQUIRED')
assert.equal(deriveWorkflowStatus({ patient: { patientId: 'p1' } }).key, 'REGISTER_REQUIRED')
assert.equal(
  deriveWorkflowStatus({ patient: { patientId: 'p1' }, register: { status: 'NORMAL' } }).key,
  'WAIT_CHECK_IN',
)
assert.equal(
  deriveWorkflowStatus({
    patient: { patientId: 'p1' },
    register: { status: 'DIAGNOSED' },
    orders: [{ status: 'COMMIT', charged: false }],
  }).key,
  'WAIT_CHARGE',
)
assert.equal(
  deriveWorkflowProgress({ patient: { patientId: 'p1' }, register: { status: 'NORMAL' } }).find((step) => step.key === 'register').current,
  true,
)

setPatient({ patientId: 'p1', name: '患者一', patientNo: 'P001' })
setRegister({ registerId: 'r1', patientId: 'p1', status: 'NORMAL' })
outpatientContext.orders = [{ orderId: 'o1', status: 'COMMIT' }]
setPatient({ patientId: 'p2', name: '患者二', patientNo: 'P002' })
assert.equal(outpatientContext.patient.patientId, 'p2')
assert.equal(outpatientContext.register, null)
assert.deepEqual(outpatientContext.orders, [])

console.log('outpatient context checks passed')
