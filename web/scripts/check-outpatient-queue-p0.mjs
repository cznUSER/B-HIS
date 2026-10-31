import assert from 'node:assert/strict'
import { buildOutpatientQueueRows, filterOutpatientQueue, loadOutpatientQueue, selectQueueRegister } from '../src/workbench/outpatientQueue.js'

const registers = [
  { registerId: 'r1', patientId: 'p1', patientName: '甲', status: 'NORMAL', visitDate: '2026-10-31' },
  { registerId: 'r2', patientId: 'p2', patientName: '乙', status: 'DIAGNOSED', visitDate: '2026-10-31' },
]
const orders = [{ orderId: 'o1', registerId: 'r2', status: 'COMMIT' }]
const charges = [{ chargeId: 'c1', registerId: 'r2', orderId: 'o1', status: 'NORMAL', paidAt: '' }]
const dispenses = []
const rows = buildOutpatientQueueRows({ registers, orders, charges, dispenses })
assert.equal(filterOutpatientQueue(rows, 'waiting').length, 1)
assert.equal(filterOutpatientQueue(rows, 'order').length, 1, 'DIAGNOSED is the primary order tab anchor')
assert.equal(filterOutpatientQueue(rows, 'charge').length, 1, 'COMMIT with unpaid charge maps to waiting-for-charge')
assert.equal(filterOutpatientQueue(rows, 'waiting', '乙').length, 0)

const calls = []
const loaded = await loadOutpatientQueue({
  visitDate: '2026-10-31',
  listRegisters: async (_token, query) => { calls.push(['registers', query]); return { items: registers } },
  listOrders: async (_token, query) => { calls.push(['orders', query]); return { items: orders } },
  listCharges: async (_token, query) => { calls.push(['charges', query]); return { items: charges } },
  listDispenses: async (_token, query) => { calls.push(['dispenses', query]); return { items: dispenses } },
}, 'token')
assert.equal(calls.length, 4)
assert.equal(loaded.errors.length, 0)

const denied = await loadOutpatientQueue({
  visitDate: '2026-10-31',
  listRegisters: async () => { throw Object.assign(new Error('forbidden'), { status: 403 }) },
  listOrders: async () => ({ items: [] }), listCharges: async () => ({ items: [] }), listDispenses: async () => ({ items: [] }),
}, 'token')
assert.equal(denied.errors[0].status, 403, '403 remains an explicit failed state, not an empty result')

const selected = []
const accepted = await selectQueueRegister(registers[0], {
  fetchPatient: async (_token, patientId) => ({ patientId, patientName: '甲' }),
  selectOutpatientContext: async (context) => { selected.push(context); return true },
}, 'token')
assert.equal(accepted, true)
assert.deepEqual(selected, [{ patient: { patientId: 'p1', patientName: '甲' }, register: registers[0] }])
console.log('P0_B_QUEUE_CHECK_OK')
