/**
 * UI90-ERR-CLEAN-01 probe (Plan B) — static file checks for first 3 pages.
 * No 90% sign-off. Does not import Vue runtime. Does not touch Diagnosis/Dispense/Order.
 */
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { execSync } from 'node:child_process'

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../..')
const targets = [
  {
    rel: 'web/src/views/RegisterView.vue',
    must: ['role="alert"', 'SubmitStatusBanner', 'loadError', 'class="banner"'],
  },
  {
    rel: 'web/src/components/RegisterPatientPicker.vue',
    must: ['register-picker-error', 'role="alert"', "error.value"],
  },
  {
    rel: 'web/src/views/ChargeView.vue',
    must: ['role="alert"', 'SubmitStatusBanner', 'errorMessage', 'data-testid="charge-error"'],
  },
]
const residual = [
  'web/src/views/DiagnosisView.vue',
  'web/src/views/DispenseView.vue',
  'web/src/views/OrderView.vue',
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

const residualHits = residual.filter((rel) => {
  const text = fs.readFileSync(path.join(root, rel), 'utf8')
  return /useErrorDialog/.test(text)
})
assert(residualHits.length === 3, `ERR-CLEAN-02 residual expected 3 pages, got ${residualHits.length}: ${residualHits.join(',')}`)

console.log(JSON.stringify({
  ok: fails.length === 0,
  fails,
  cleaned: targets.map((t) => t.rel),
  residualForErrClean02: residualHits,
  helperOnDiskButMustStayUntracked: helperOnDisk,
  helperTracked,
  note: 'Plan B first slice: RegisterView + RegisterPatientPicker + ChargeView; no 90% sign-off; Diagnosis/Dispense/Order deferred',
}, null, 2))
process.exit(fails.length === 0 ? 0 : 1)