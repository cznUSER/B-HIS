import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

import {
  COMMON_DIAGNOSES,
  appendDiagnosisName,
  assessmentIncludesName,
} from '../src/workbench/commonDiagnoses.js'

const here = path.dirname(fileURLToPath(import.meta.url))
const root = path.resolve(here, '..')

assert.deepEqual([...COMMON_DIAGNOSES], [
  '上呼吸道感染',
  '急性咽炎',
  '高血压',
  '2型糖尿病',
  '胃炎',
])
assert.equal(Object.isFrozen(COMMON_DIAGNOSES), true)

assert.equal(appendDiagnosisName('', '上呼吸道感染'), '上呼吸道感染')
assert.equal(appendDiagnosisName('  ', '胃炎'), '胃炎')
assert.equal(appendDiagnosisName('高血压', '胃炎'), '高血压，胃炎')
assert.equal(appendDiagnosisName('高血压，胃炎', '高血压'), '高血压，胃炎')
assert.equal(appendDiagnosisName('高血压，胃炎', ' 高血压 '), '高血压，胃炎')
assert.equal(appendDiagnosisName('上呼吸道感染', ''), '上呼吸道感染')
assert.equal(appendDiagnosisName(null, '急性咽炎'), '急性咽炎')
assert.equal(appendDiagnosisName('急性咽炎', null), '急性咽炎')

assert.equal(assessmentIncludesName('高血压，胃炎', '胃炎'), true)
assert.equal(assessmentIncludesName('高血压，胃炎', '2型糖尿病'), false)
assert.equal(assessmentIncludesName('', '高血压'), false)

const helper = fs.readFileSync(path.join(root, 'src/workbench/commonDiagnoses.js'), 'utf8')
assert.equal(helper.includes('fetch('), false)
assert.equal(helper.includes('/api/'), false)
assert.equal(helper.includes('localStorage'), false)
assert.equal(helper.includes('sessionStorage'), false)
assert.equal(/icd/i.test(helper), false)

const view = fs.readFileSync(path.join(root, 'src/views/DiagnosisView.vue'), 'utf8')
assert.match(view, /commonDiagnoses/)
assert.match(view, /COMMON_DIAGNOSES/)
assert.match(view, /appendDiagnosisName/)
assert.match(view, /applyCommonDiagnosis/)
assert.match(view, /data-testid="diagnosis-common-chips"/)
assert.match(view, /data-testid="diagnosis-common-chip"/)
assert.match(view, /常用诊断/)
assert.match(view, /editorLocked \|\| busy/)
assert.match(view, /diagnosis-common-diagnoses/)
assert.equal(view.includes('localStorage'), false)
assert.equal(view.includes('sessionStorage'), false)

assert.match(view, /diagnosisDirtyGuard/)
assert.match(view, /createSubmitFeedback/)
assert.match(view, /SubmitStatusBanner/)
assert.match(view, /DIRTY_HINT|有未暂存修改/)

assert.equal(view.includes('OrderView'), false)
assert.equal(view.includes('DoctorDiagnosisPanel'), false)

console.log('P1_DIAG_COMMON_OK')
