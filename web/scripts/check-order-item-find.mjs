import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

import { CATALOG } from '../src/api/orderContract.js'
import {
  DEFAULT_FIND_CATEGORY,
  EMPTY_MATCH_MESSAGE,
  FIND_CATEGORIES,
  categoryLabel,
  filterByKeyword,
  findSelectedLabel,
  normalizeKeyword,
} from '../src/workbench/orderItemFind.js'

const here = path.dirname(fileURLToPath(import.meta.url))
const root = path.resolve(here, '..')

assert.deepEqual(
  FIND_CATEGORIES.map((item) => item.id),
  ['DRUG', 'TREAT'],
)
assert.equal(DEFAULT_FIND_CATEGORY, 'DRUG')
assert.equal(categoryLabel('DRUG'), '药品')
assert.equal(categoryLabel('TREAT'), '治疗')
assert.equal(EMPTY_MATCH_MESSAGE, '无匹配项目')

assert.equal(normalizeKeyword('  amox  '), 'amox')
assert.equal(normalizeKeyword(''), '')
assert.equal(normalizeKeyword(null), '')

const drugs = filterByKeyword(CATALOG, { category: 'DRUG', keyword: '' })
assert.equal(drugs.length, 2)
assert.ok(drugs.every((item) => item.itemType === 'DRUG'))

const treats = filterByKeyword(CATALOG, { category: 'TREAT', keyword: '' })
assert.equal(treats.length, 1)
assert.equal(treats[0].itemNo, 'TREAT-1')

const byName = filterByKeyword(CATALOG, { category: 'DRUG', keyword: '  布洛芬  ' })
assert.equal(byName.length, 1)
assert.equal(byName[0].itemNo, 'DRUG-B')

const byNoCase = filterByKeyword(CATALOG, { category: 'DRUG', keyword: 'drug-a' })
assert.equal(byNoCase.length, 1)
assert.equal(byNoCase[0].itemNo, 'DRUG-A')

const bySpec = filterByKeyword(CATALOG, { category: 'DRUG', keyword: '0.25' })
assert.ok(bySpec.some((item) => item.itemNo === 'DRUG-A'))

const empty = filterByKeyword(CATALOG, { category: 'DRUG', keyword: '不存在的项目XYZ' })
assert.equal(empty.length, 0)

const cleared = filterByKeyword(CATALOG, { category: 'DRUG', keyword: '   ' })
assert.equal(cleared.length, 2)

assert.equal(findSelectedLabel(CATALOG, 'DRUG-A'), '演示阿莫西林胶囊')
assert.equal(findSelectedLabel(CATALOG, ''), '')

// 选中回写语义：过滤结果的 itemNo 可直接赋给行；多行独立（纯函数无共享状态）
const line0 = { itemNo: '' }
const line1 = { itemNo: '' }
line0.itemNo = byName[0].itemNo
line1.itemNo = byNoCase[0].itemNo
assert.equal(line0.itemNo, 'DRUG-B')
assert.equal(line1.itemNo, 'DRUG-A')
assert.notEqual(line0.itemNo, line1.itemNo)

const orderView = fs.readFileSync(path.join(root, 'src/views/OrderView.vue'), 'utf8')
assert.match(orderView, /OrderItemFinder/)
assert.match(orderView, /orderItemFind|findItems/)
assert.match(orderView, /:disabled="locked \|\| busy \|\| !canWrite"/)
assert.match(orderView, /常用项目/)
assert.match(orderView, /data-testid="order-common-bar"/)
assert.match(orderView, /最近使用/) // P1-ORDER-RECENT-01 coexistence; FIND filter contract unchanged
assert.equal(/皮试/.test(orderView) && /特殊处方/.test(orderView.replace('普通处方', '')), false)

// PRINT / 提交接线不得被本片拆掉
assert.match(orderView, /OrderPrintPanel/)
assert.match(orderView, /printController/)
assert.match(orderView, /createSubmitFeedback/)
assert.match(orderView, /SubmitStatusBanner/)

const finder = fs.readFileSync(path.join(root, 'src/components/outpatient/OrderItemFinder.vue'), 'utf8')
assert.match(finder, /data-testid="line-item-\$\{|data-testid="\`line-item-\$\{index\}`"|:data-testid="testId"|data-testid="\$\{testId\}"/)
assert.match(finder, /FIND_CATEGORIES/)
assert.match(finder, /filterByKeyword/)
assert.match(finder, /EMPTY_MATCH_MESSAGE|无匹配项目/)
assert.match(finder, /line-item-cat-/)
assert.match(finder, /line-item-q-/)
assert.match(finder, /line-item-empty-/)
assert.match(finder, /disabled/)
assert.equal(finder.includes('window.print'), false)
assert.equal(finder.includes('常用'), false)

const helper = fs.readFileSync(path.join(root, 'src/workbench/orderItemFind.js'), 'utf8')
assert.match(helper, /export function filterByKeyword/)
assert.equal(helper.includes('fetch('), false)
assert.equal(helper.includes('/api/'), false)

console.log('P1_ORDER_ITEM_FIND_OK')