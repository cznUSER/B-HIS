/**
 * UI90-ERR-CLEAN-02 probe (Plan B) — static checks for Diagnosis/Dispense/Order.
 * No 90% sign-off. Does not import Vue runtime. Does not stage useErrorDialog.js.
 */
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { execSync } from 'node:child_process'

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../..')
const targets = [
  {
    rel: 'web/src/views/DiagnosisView.vue',
    must: ['role="alert"', 'SubmitStatusBanner', 'errorMessage', 'createSubmitFeedback'],
  },
  {
    rel: 'web/src/views/DispenseView.vue',
    must: ['role="alert"', 'SubmitStatusBanner', 'errorMessage', 'data-testid="dispense-error"'],
  },
  {
    rel: 'web/src/views/OrderView.vue',
    must: ['role="alert"', 'SubmitStatusBanner', 'errorMessage', 'data-testid="order-error"'],
  },
]
const fails = []
function assert(cond, msg) { if (!cond) fails.push(msg) }

for (const t of targets) {
  const abs = path.join(root, t.rel)
  const text = fs.readFileSync(abs, 'utf8')
  assert(!/useErrorDialog/.test(text), `${t.rel} must not reference useErrorDialog`)
  for (const needle of t.must) {
    assert(text.includes(needle), `${t.rel} missing banner/path marker: ${needle}`)
  }
}

const helper = path.join(root, 'web/src/components/feedback/useErrorDialog.js')
const helperOnDisk = fs.existsSync(helper)
let helperTracked = false
try {
  execSync('git ls-files --error-unmatch -- web/src/components/feedback/useErrorDialog.js', {
    cwd: root, stdio: ['ignore', 'ignore', 'ignore'],
  })
  helperTracked = true
} catch {
  helperTracked = false
}
assert(!helperTracked, 'useErrorDialog.js must stay untracked (Plan B)')

let tipGrep = ''
try {
  tipGrep = execSync('git grep useErrorDialog HEAD -- web/src', {
    cwd: root, encoding: 'utf8',
  }).trim()
} catch (e) {
  // git grep exits 1 when no matches
  tipGrep = (e.stdout || '').toString().trim()
}
// After commit tip should be empty; before commit, HEAD still has old refs — probe WT files above.
const wtClean = targets.every((t) => !/useErrorDialog/.test(fs.readFileSync(path.join(root, t.rel), 'utf8')))

console.log(JSON.stringify({
  ok: fails.length === 0,
  fails,
  cleaned: targets.map((t) => t.rel),
  wtThreePagesClean: wtClean,
  tipWebSrcGrepAtProbeTime: tipGrep || '(empty)',
  helperOnDiskButMustStayUntracked: helperOnDisk,
  helperTracked,
  note: 'Plan B close-out: DiagnosisView + DispenseView + OrderView; no 90% sign-off; ERR WIP do-not-commit',
}, null, 2))
process.exit(fails.length === 0 ? 0 : 1)