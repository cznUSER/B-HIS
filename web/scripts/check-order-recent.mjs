import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

import {
  MAX_RECENT_ORDER_ITEMS,
  clearSessionRecentItemNos,
  getSessionRecentItemNos,
  normalizeRecentItemNos,
  pushRecentItemNo,
  rememberSessionItemNo,
  resolveRecentItem,
} from '../src/workbench/recentOrderItems.js'

const here = path.dirname(fileURLToPath(import.meta.url))
const root = path.resolve(here, '..')

assert.equal(MAX_RECENT_ORDER_ITEMS, 8)
assert.deepEqual(pushRecentItemNo([' A ', 'B', 'A'], ' B '), ['B', 'A'])
assert.deepEqual(normalizeRecentItemNos([' A ', '', null, 'B', 'A', 'C']), ['A', 'B', 'C'])
assert.deepEqual(normalizeRecentItemNos(['A', 'B', 'C'], { max: 2 }), ['A', 'B'])
assert.deepEqual(pushRecentItemNo(['A', 'B'], 'C', { max: 2 }), ['C', 'A'])
assert.deepEqual(normalizeRecentItemNos(['', ' ', null]), [])
assert.deepEqual(
  pushRecentItemNo(['1', '2', '3', '4', '5', '6', '7', '8'], '9'),
  ['9', '1', '2', '3', '4', '5', '6', '7'],
)

clearSessionRecentItemNos()
rememberSessionItemNo('DRUG-A')
rememberSessionItemNo('DRUG-B')
rememberSessionItemNo('DRUG-A')
assert.deepEqual(getSessionRecentItemNos(), ['DRUG-A', 'DRUG-B'])
assert.deepEqual(getSessionRecentItemNos(), ['DRUG-A', 'DRUG-B'])
clearSessionRecentItemNos()
assert.deepEqual(getSessionRecentItemNos(), [])

const catalog = [
  { itemNo: 'ACTIVE-1', itemName: 'Active', status: 'ACTIVE' },
  { itemNo: 'INACTIVE-1', itemName: 'Inactive', status: 'INACTIVE' },
  { itemNo: 'NO-STATUS-1', label: 'Selectable without status' },
]
assert.equal(resolveRecentItem('ACTIVE-1', catalog).itemName, 'Active')
assert.equal(resolveRecentItem('NO-STATUS-1', catalog).label, 'Selectable without status')
assert.equal(resolveRecentItem('INACTIVE-1', catalog), null)
assert.equal(resolveRecentItem('MISSING', catalog), null)
assert.equal(resolveRecentItem('', catalog), null)

const helper = fs.readFileSync(path.join(root, 'src/workbench/recentOrderItems.js'), 'utf8')
assert.equal(helper.includes('fetch('), false)
assert.equal(helper.includes('/api/'), false)
assert.equal(helper.includes('localStorage'), false)
assert.equal(helper.includes('sessionStorage'), false)
assert.match(helper, /会话内存/)
assert.match(helper, /非持久化/)

const view = fs.readFileSync(path.join(root, 'src/views/OrderView.vue'), 'utf8')
assert.equal(view.includes('localStorage'), false)
assert.equal(view.includes('sessionStorage'), false)
assert.match(view, /recentOrderItems/)
assert.match(view, /rememberSessionItemNo/)
assert.match(view, /getSessionRecentItemNos/)
assert.match(view, /resolveRecentItem/)
assert.match(view, /applyCommonItemToLines/)
assert.match(view, /data-testid="order-recent-bar"/)
assert.match(view, /order-recent-item-\$\{item\.itemNo\}/)
assert.match(view, /data-testid="order-recent-empty"/)
assert.match(view, /本会话暂无最近使用/)
assert.match(view, /刷新后清空/)
assert.match(view, /:disabled="locked \|\| busy \|\| !canWrite \|\| !item\.resolved"/)
assert.match(view, /applyRecentOrderItem/)
assert.match(view, /OrderItemFinder/)
assert.match(view, /order-common-bar/)
assert.match(view, /OrderPrintPanel/)
assert.match(view, /createSubmitFeedback/)

console.log('P1_ORDER_RECENT_OK')