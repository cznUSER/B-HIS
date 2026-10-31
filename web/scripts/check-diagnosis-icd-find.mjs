import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

import {
  MOCK_ICD_CODES,
  ICD_DEMO_DISCLAIMER,
  EMPTY_ICD_MATCH_MESSAGE,
  ICD_DEFAULT_PREVIEW_LIMIT,
  filterIcdByKeyword,
  formatIcdAssessmentToken,
  formatIcdEntryToken,
  appendIcdToken,
  assessmentIncludesIcdToken,
  normalizeIcdKeyword,
} from '../src/workbench/mockIcdCodes.js'

const here = path.dirname(fileURLToPath(import.meta.url))
const root = path.resolve(here, '..')

assert.equal(ICD_DEMO_DISCLAIMER.includes('演示'), true)
assert.match(ICD_DEMO_DISCLAIMER, /非完整官方 ICD/)
assert.equal(EMPTY_ICD_MATCH_MESSAGE, '无匹配演示码')
assert.equal(ICD_DEFAULT_PREVIEW_LIMIT, 12)

assert.equal(Object.isFrozen(MOCK_ICD_CODES), true)
assert.ok(MOCK_ICD_CODES.length >= 20 && MOCK_ICD_CODES.length <= 40)
assert.ok(MOCK_ICD_CODES.every((row) => row && row.code && row.name))
assert.equal(new Set(MOCK_ICD_CODES.map((row) => row.code)).size, MOCK_ICD_CODES.length)

assert.equal(formatIcdAssessmentToken('J06.900', '急性上呼吸道感染'), 'J06.900 急性上呼吸道感染')
assert.equal(formatIcdAssessmentToken('', '仅名'), '仅名')
assert.equal(formatIcdAssessmentToken('X', ''), 'X')
assert.equal(formatIcdAssessmentToken('', ''), '')
assert.equal(formatIcdEntryToken(MOCK_ICD_CODES[0]), 'J06.900 急性上呼吸道感染')

assert.equal(normalizeIcdKeyword('  高血压  '), '高血压')
assert.equal(normalizeIcdKeyword(null), '')

const preview = filterIcdByKeyword(MOCK_ICD_CODES, '')
assert.equal(preview.length, ICD_DEFAULT_PREVIEW_LIMIT)
assert.equal(preview[0].code, MOCK_ICD_CODES[0].code)

const byAlias = filterIcdByKeyword(MOCK_ICD_CODES, '高血压')
assert.ok(byAlias.some((row) => row.code === 'I10.x00'))

const byCode = filterIcdByKeyword(MOCK_ICD_CODES, 'j06.900')
assert.equal(byCode.length, 1)
assert.equal(byCode[0].name, '急性上呼吸道感染')

const byName = filterIcdByKeyword(MOCK_ICD_CODES, '咽炎')
assert.ok(byName.some((row) => row.code === 'J02.900'))

const byPinyin = filterIcdByKeyword(MOCK_ICD_CODES, 'yfxgxy')
assert.ok(byPinyin.some((row) => row.code === 'I10.x00'))

const empty = filterIcdByKeyword(MOCK_ICD_CODES, '不存在的演示码XYZ')
assert.equal(empty.length, 0)

const token = formatIcdEntryToken(MOCK_ICD_CODES[0])
assert.equal(appendIcdToken('', MOCK_ICD_CODES[0]), token)
assert.equal(appendIcdToken('  ', token), token)
assert.equal(appendIcdToken('胃炎', MOCK_ICD_CODES[0]), `胃炎，${token}`)
assert.equal(appendIcdToken(`胃炎，${token}`, MOCK_ICD_CODES[0]), `胃炎，${token}`)
assert.equal(appendIcdToken(token, ''), token)
assert.equal(assessmentIncludesIcdToken(`胃炎，${token}`, MOCK_ICD_CODES[0]), true)
assert.equal(assessmentIncludesIcdToken('胃炎', MOCK_ICD_CODES[0]), false)

const helper = fs.readFileSync(path.join(root, 'src/workbench/mockIcdCodes.js'), 'utf8')
assert.match(helper, /非完整官方 ICD/)
assert.match(helper, /演示/)
assert.match(helper, /Mock|DEMO|演示码表/)
assert.equal(helper.includes('fetch('), false)
assert.equal(helper.includes('/api/'), false)
assert.equal(helper.includes('localStorage'), false)
assert.equal(helper.includes('sessionStorage'), false)
assert.equal(/完整官方 ICD-10/.test(helper) && helper.includes('本文件即为官方全量'), false)

const view = fs.readFileSync(path.join(root, 'src/views/DiagnosisView.vue'), 'utf8')
assert.match(view, /mockIcdCodes/)
assert.match(view, /ICD_DEMO_DISCLAIMER/)
assert.match(view, /filterIcdByKeyword/)
assert.match(view, /appendIcdToken/)
assert.match(view, /applyIcdEntry/)
assert.match(view, /data-testid="icd-find-input"/)
assert.match(view, /data-testid="icd-find-results"/)
assert.match(view, /icd-find-item-/)
assert.match(view, /icd-find-disclaimer/)
assert.match(view, /ICD 检索（演示）/)
assert.match(view, /editorLocked \|\| busy/)
assert.match(view, /COMMON_DIAGNOSES/)
assert.match(view, /data-testid="diagnosis-common-chips"/)
assert.match(view, /applyCommonDiagnosis/)
assert.equal(view.includes('localStorage'), false)
assert.equal(view.includes('sessionStorage'), false)
assert.equal(view.includes('fetch('), false)
assert.equal(view.includes('OrderView'), false)
assert.equal(view.includes('DoctorDiagnosisPanel'), false)
assert.match(view, /diagnosisDirtyGuard/)
assert.match(view, /createSubmitFeedback/)
assert.match(view, /SubmitStatusBanner/)

// COMMON 五条不得被吞
assert.match(view, /常用诊断/)
assert.equal((view.match(/diagnosis-common-chip/g) || []).length >= 2, true)

console.log('P1_DIAG_ICD_FIND_OK')