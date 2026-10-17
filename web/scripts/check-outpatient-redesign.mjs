import fs from 'node:fs'
import path from 'node:path'
import process from 'node:process'
import { fileURLToPath } from 'node:url'

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const read = (file) => fs.readFileSync(path.join(root, file), 'utf8')
const required = [
  'src/components/OutpatientWorkbenchFrame.vue',
  'src/components/outpatient-workbench.css',
  'src/views/OutpatientModuleView.vue',
  'src/views/PatientView.vue',
  'src/views/RegisterView.vue',
  'src/views/DiagnosisView.vue',
  'src/views/OrderView.vue',
  'src/views/ChargeView.vue',
  'src/views/DispenseView.vue',
  'src/views/RefundView.vue',
]
for (const file of required) {
  if (!fs.existsSync(path.join(root, file))) throw new Error(`missing redesign file: ${file}`)
}

const frame = read('src/components/OutpatientWorkbenchFrame.vue')
const moduleView = read('src/views/OutpatientModuleView.vue')
const steps = read('src/components/WorkflowSteps.vue')
const expectedKeys = ['patient', 'register', 'diagnosis', 'order', 'charge', 'dispense', 'refund']
for (const key of expectedKeys) {
  if (!moduleView.includes(`show${key[0].toUpperCase()}${key.slice(1)}`)) throw new Error(`module route is not mounted: ${key}`)
}
for (const anchor of ['outpatient-context', 'outpatient-status']) {
  if (!frame.includes(`data-testid="${anchor}"`)) throw new Error(`missing shared anchor: ${anchor}`)
}
for (const anchor of ['outpatient-three-column', 'outpatient-left-rail', 'outpatient-right-rail', 'outpatient-status-chain', 'outpatient-fee-summary', 'outpatient-inventory-summary']) {
  if (!frame.includes(anchor)) throw new Error(`missing three-column anchor: ${anchor}`)
}
if (!fs.existsSync(path.join(root, 'src/workbench/outpatientContext.js'))) throw new Error('missing shared outpatient context')
if (!steps.includes('data-testid="outpatient-step-nav"')) throw new Error('missing step navigation anchor')

const boardPath = path.resolve(root, '..', '.codex', '实时进度', '任务看板.json')
const boardHtmlPath = path.resolve(root, '..', '.codex', '实时进度', '任务看板.html')
const board = JSON.parse(fs.readFileSync(boardPath, 'utf8'))
if (!Array.isArray(board.tasks) || board.tasks.length < 1) throw new Error('task board has no tasks')
if (!board.counts || board.counts.total !== board.tasks.length) throw new Error('task board counts do not match tasks')
const boardHtml = fs.readFileSync(boardHtmlPath, 'utf8')
for (const token of ['任务进程看板', '任务看板.json', 'setInterval(load, 5000)', '当前任务']) {
  if (!boardHtml.includes(token)) throw new Error(`task board is missing ${token}`)
}

console.log(`REDESIGN_CHECK_OK modules=${expectedKeys.length} boardTasks=${board.tasks.length} sharedAnchors=9`)
