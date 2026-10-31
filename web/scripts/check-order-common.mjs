import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

import { CATALOG } from '../src/api/orderContract.js'
import {
  COMMON_ORDER_ITEMS,
  applyCommonItemToLines,
  isCommonItemResolvable,
  resolveCommonItem,
} from '../src/workbench/commonOrderItems.js'

const here = path.dirname(fileURLToPath(import.meta.url))
const root = path.resolve(here, '..')

assert.equal(Object.isFrozen(COMMON_ORDER_ITEMS), true)
assert.deepEqual(
  COMMON_ORDER_ITEMS.map((item) => item.itemNo),
  ['DRUG-A', 'DRUG-B', 'TREAT-1'],
)
assert.ok(COMMON_ORDER_ITEMS.every((item) => Object.isFrozen(item)))

const catalogItems = CATALOG.map((item) => ({ ...item, status: 'ACTIVE' }))
assert.equal(resolveCommonItem('DRUG-A', catalogItems).itemNo, 'DRUG-A')
assert.equal(resolveCommonItem('DRUG-A', [{ itemNo: 'DRUG-A', status: 'INACTIVE' }]), null)
assert.equal(resolveCommonItem('MISSING', catalogItems), null)
assert.equal(resolveCommonItem('DRUG-A').itemNo, 'DRUG-A')
assert.equal(resolveCommonItem('MISSING'), null)
assert.equal(isCommonItemResolvable('TREAT-1', catalogItems), true)
assert.equal(isCommonItemResolvable('TREAT-1', []), false)

const original = [{ key: 'line-1', itemNo: '', amount: '', usageText: '', dosage: '' }]
const filled = applyCommonItemToLines(original, catalogItems[0])
assert.equal(filled[0].itemNo, 'DRUG-A')
assert.equal(filled[0].amount, '')
assert.equal(filled[0].usageText, '')
assert.equal(filled[0].dosage, '')
assert.equal(original[0].itemNo, '')
assert.notEqual(filled, original)

const filledLines = [{ key: 'line-1', itemNo: 'DRUG-B', amount: '2', usageText: '口服' }]
let addBlankCount = 0
const added = applyCommonItemToLines(filledLines, catalogItems[2], {
  addBlankLine: () => {
    addBlankCount += 1
    return { key: 'line-2', itemNo: '', amount: '', usageText: '' }
  },
})
assert.equal(addBlankCount, 1)
assert.equal(added.length, 2)
assert.equal(added[1].itemNo, 'TREAT-1')
assert.equal(added[1].amount, '')
assert.equal(filledLines.length, 1)

const helperSrc = fs.readFileSync(path.join(root, 'src/workbench/commonOrderItems.js'), 'utf8')
assert.equal(helperSrc.includes('fetch('), false)
assert.equal(helperSrc.includes('/api/'), false)
assert.equal(helperSrc.includes('localStorage'), false)
assert.equal(helperSrc.includes('sessionStorage'), false)
assert.match(helperSrc, /非个人收藏/)

const view = fs.readFileSync(path.join(root, 'src/views/OrderView.vue'), 'utf8')
assert.match(view, /commonOrderItems/)
assert.match(view, /COMMON_ORDER_ITEMS/)
assert.match(view, /resolveCommonItem/)
assert.match(view, /applyCommonItemToLines/)
assert.match(view, /data-testid="order-common-bar"/)
assert.match(view, /order-common-item-\$\{item\.itemNo\}/)
assert.match(view, /常用项目/)
assert.match(view, /演示短表（非个人收藏）/)
assert.match(view, /:disabled="locked \|\| busy \|\| !canWrite \|\| !item\.resolved"/)
assert.equal(view.includes('localStorage'), false)
assert.equal(view.includes('sessionStorage'), false)

assert.match(view, /OrderItemFinder/)
assert.match(view, /OrderPrintPanel/)
assert.match(view, /createSubmitFeedback/)
assert.match(view, /SubmitStatusBanner/)

console.log('P1_ORDER_COMMON_OK')
