import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

import {
  STRUCTURED_DIFF_KEYS,
  STRUCTURED_DIFF_LABELS,
  META_DIFF_KEYS,
  META_DIFF_LABELS,
  DIFF_NEED_TWO_HINT,
  DIFF_NO_CHANGES_HINT,
  text,
  pickStructured,
  diffDiagnosisRecords,
  canOpenVersionDiff,
  defaultDiffPair,
} from '../src/workbench/diagnosisVersionDiff.js'

const here = path.dirname(fileURLToPath(import.meta.url))
const root = path.resolve(here, '..')

assert.deepEqual([...STRUCTURED_DIFF_KEYS], [
  'chiefComplaint',
  'presentHistory',
  'pastHistory',
  'examination',
  'assessment',
  'plan',
])
assert.equal(Object.isFrozen(STRUCTURED_DIFF_KEYS), true)
assert.equal(STRUCTURED_DIFF_LABELS.assessment, '初步诊断')
assert.deepEqual([...META_DIFF_KEYS], ['templateCode', 'status', 'version', 'revisionReason'])
assert.equal(META_DIFF_LABELS.version, '版本号')
assert.equal(DIFF_NEED_TWO_HINT, '至少需要 2 条记录才能对比')
assert.equal(DIFF_NO_CHANGES_HINT, '无字段差异')
assert.equal(text(null), '')
assert.equal(text(undefined), '')
assert.equal(text(2), '2')

assert.equal(canOpenVersionDiff([]), false)
assert.equal(canOpenVersionDiff([{ diagnoseId: 'a' }]), false)
assert.equal(canOpenVersionDiff([{ diagnoseId: 'a' }, { diagnoseId: 'b' }]), true)
assert.equal(canOpenVersionDiff(null), false)

const baseStructured = {
  chiefComplaint: '发热',
  presentHistory: '3天',
  pastHistory: '',
  examination: '咽红',
  assessment: '上呼道感染',
  plan: '休息',
}

const older = {
  diagnoseId: 'd-old',
  version: 1,
  status: 'COMMIT',
  templateCode: 'GENERAL',
  revisionReason: '',
  structured: { ...baseStructured },
  content: 'old',
}

const newer = {
  diagnoseId: 'd-new',
  version: 2,
  status: 'SAVE',
  templateCode: 'GENERAL',
  revisionReason: '补充主诉',
  supersedesDiagnoseId: 'd-old',
  structured: { ...baseStructured, assessment: '急性咽炎' },
  content: 'new',
}

const picked = pickStructured(newer)
assert.equal(picked.diagnoseId, 'd-new')
assert.equal(picked.version, '2')
assert.equal(picked.structured.assessment, '急性咽炎')
assert.equal(pickStructured({}).version, '1')
assert.equal(pickStructured({ structured: null }).structured.assessment, '')

const changed = diffDiagnosisRecords(older, newer)
assert.equal(changed.identical, false)
assert.ok(changed.changedCount >= 1)
const assessmentField = changed.fields.find((f) => f.key === 'assessment')
assert.ok(assessmentField)
assert.equal(assessmentField.changed, true)
assert.equal(assessmentField.left, '上呼道感染')
assert.equal(assessmentField.right, '急性咽炎')
assert.equal(assessmentField.group, 'structured')
const versionField = changed.fields.find((f) => f.key === 'version')
assert.equal(versionField.changed, true)
assert.equal(versionField.left, '1')
assert.equal(versionField.right, '2')

const same = diffDiagnosisRecords(older, { ...older, diagnoseId: 'd-copy' })
assert.equal(same.identical, true)
assert.equal(same.changedCount, 0)

const pair = defaultDiffPair([older, newer], 'd-new')
assert.equal(pair.left.diagnoseId, 'd-old')
assert.equal(pair.right.diagnoseId, 'd-new')
const pairFallback = defaultDiffPair([
  { diagnoseId: 'x', version: 1 },
  { diagnoseId: 'y', version: 1 },
])
assert.equal(pairFallback.left.diagnoseId, 'x')
assert.equal(pairFallback.right.diagnoseId, 'y')
assert.equal(defaultDiffPair([older]), null)

const helper = fs.readFileSync(path.join(root, 'src/workbench/diagnosisVersionDiff.js'), 'utf8')
assert.equal(helper.includes('fetch('), false)
assert.equal(helper.includes('/api/'), false)
assert.equal(helper.includes('localStorage'), false)
assert.equal(helper.includes('sessionStorage'), false)
assert.equal(helper.includes('diagnosisContract'), false)

const view = fs.readFileSync(path.join(root, 'src/views/DiagnosisView.vue'), 'utf8')
for (const marker of [
  'diagnosisVersionDiff',
  'DIFF_NEED_TWO_HINT',
  'DIFF_NO_CHANGES_HINT',
  'canOpenVersionDiff',
  'defaultDiffPair',
  'diffDiagnosisRecords',
  'openVersionDiff',
  'closeVersionDiff',
  'data-testid="diagnosis-version-diff-open"',
  'data-testid="diagnosis-version-diff-panel"',
  'diagnosis-version-diff-field-',
  'diagnosisDirtyGuard',
  'commonDiagnoses',
  'mockIcdCodes',
  'createSubmitFeedback',
  'SubmitStatusBanner',
]) {
  assert.ok(view.includes(marker), `missing view marker: ${marker}`)
}
assert.equal(view.includes('localStorage'), false)
assert.equal(view.includes('sessionStorage'), false)
assert.equal(view.includes('OrderView'), false)
assert.equal(view.includes('DoctorDiagnosisPanel'), false)

const openBody = view.match(/function openVersionDiff\(\)\s*\{([\s\S]*?)\n\}/)?.[1] || ''
assert.match(openBody, /defaultDiffPair/)
assert.equal(openBody.includes('applyDiagnosis'), false)
assert.equal(openBody.includes('chooseDiagnosis'), false)
assert.equal(openBody.includes('resetDraft'), false)

const closeBody = view.match(/function closeVersionDiff\(\)\s*\{([\s\S]*?)\n\}/)?.[1] || ''
assert.equal(closeBody.includes('applyDiagnosis'), false)
assert.equal(closeBody.includes('resetDraft'), false)

console.log('P1_DIAG_VERSION_DIFF_OK')
