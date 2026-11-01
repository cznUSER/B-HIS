/**
 * UI90-04D-FIX-02b CLEAN probe (Plan B) — static file checks.
 * No 90% sign-off. Does not import Vue runtime.
 */
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../..')
const panel = path.join(root, 'web/src/views/master/DepartmentsPanel.vue')
const text = fs.readFileSync(panel, 'utf8')
const fails = []

function assert(cond, msg) {
  if (!cond) fails.push(msg)
}

assert(!/useErrorDialog/.test(text), 'DepartmentsPanel must not reference useErrorDialog')
assert(/data-testid="dept-notice"/.test(text), 'dept-notice banner missing')
assert(/data-testid="dept-load-error"/.test(text), 'dept-load-error banner missing')
assert(/const loadError = ref\(''\)/.test(text), 'loadError ref missing')
assert(/const notice = ref\(''\)/.test(text), 'notice ref missing')
assert(/data-testid="dept-nursing-list"/.test(text), 'nursing list missing')
assert(/data-testid="dept-nursing-add"/.test(text), 'nursing add missing')
assert(/data-testid="dept-nursing-del"/.test(text), 'nursing del missing')
assert(/--dept-nursing-list-h:\s*257px/.test(text), 'nursing list height token missing')
assert(/--dept-editor-w:\s*408px/.test(text), 'editor width token missing')
assert(/data-testid="dept-editor-tabs"/.test(text), 'FIX-01 tabs missing')
assert(/deptCode:\s*form\.deptCode/.test(text), 'saveDepartment deptCode missing')
assert(/deptName:\s*form\.deptName/.test(text), 'saveDepartment deptName missing')
assert(/status:\s*form\.status/.test(text), 'saveDepartment status missing')
assert(/sortNo:\s*form\.sortNo/.test(text), 'saveDepartment sortNo missing')
assert(!/nursingMembers/.test(text.match(/await saveDepartment[\s\S]*?\n\s*\}\)/)?.[0] || ''), 'saveDepartment must not include nursingMembers')

const helperTrackedHint = path.join(root, 'web/src/components/feedback/useErrorDialog.js')
const helperExistsOnDisk = fs.existsSync(helperTrackedHint)
console.log(JSON.stringify({
  ok: fails.length === 0,
  fails,
  panel,
  helperExistsOnDiskButMustStayUntracked: helperExistsOnDisk,
  note: 'Plan B: do not git-add useErrorDialog.js; banners + nursing shell conserved; no 90% sign-off',
}, null, 2))
process.exit(fails.length === 0 ? 0 : 1)