import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

import {
  DIRTY_HINT,
  LEAVE_CONFIRM_MESSAGE,
  REGISTER_SWITCH_CONFIRM_MESSAGE,
  NEW_DRAFT_CONFIRM_MESSAGE,
  LOCAL_SAVE_HINT_PREFIX,
  SERVER_SAVED_HINT,
  snapshotOf,
  isDirty,
  formatLocalSaveAt,
  localSaveHint,
  resumeSaveHint,
} from '../src/workbench/diagnosisDirtyGuard.js'

const here = path.dirname(fileURLToPath(import.meta.url))
const root = path.resolve(here, '..')

const empty = snapshotOf({})
assert.equal(empty.templateCode, 'GENERAL')
assert.equal(empty.revisionReason, '')
assert.equal(empty.amendSourceId, '')
assert.equal(empty.structured.chiefComplaint, '')

const baseline = snapshotOf({
  structured: { chiefComplaint: '发热', assessment: '上感' },
  templateCode: 'GENERAL',
  revisionReason: '',
  amendSourceId: '',
})
assert.equal(isDirty(baseline, baseline), false)
assert.equal(
  isDirty(
    baseline,
    snapshotOf({
      structured: { chiefComplaint: '发热', assessment: '上感' },
      templateCode: 'GENERAL',
    }),
  ),
  false,
)
assert.equal(
  isDirty(
    baseline,
    snapshotOf({
      structured: { chiefComplaint: '咳嗽', assessment: '上感' },
      templateCode: 'GENERAL',
    }),
  ),
  true,
)
assert.equal(
  isDirty(
    baseline,
    snapshotOf({
      structured: { chiefComplaint: '发热', assessment: '上感' },
      templateCode: 'INITIAL',
    }),
  ),
  true,
)
assert.equal(
  isDirty(
    baseline,
    snapshotOf({
      structured: { chiefComplaint: '发热', assessment: '上感' },
      templateCode: 'GENERAL',
      revisionReason: '补写',
    }),
  ),
  true,
)
assert.equal(
  isDirty(
    baseline,
    snapshotOf({
      structured: { chiefComplaint: '发热', assessment: '上感' },
      templateCode: 'GENERAL',
      amendSourceId: 'diag-1',
    }),
  ),
  true,
)
// editorLocked never dirty
assert.equal(
  isDirty(
    baseline,
    snapshotOf({
      structured: { chiefComplaint: '完全不同' },
      templateCode: 'X',
    }),
    { editorLocked: true },
  ),
  false,
)

const stamped = new Date(2026, 9, 31, 18, 7, 9)
assert.equal(formatLocalSaveAt(stamped), '18:07:09')
assert.equal(localSaveHint(stamped), `${LOCAL_SAVE_HINT_PREFIX} · 18:07:09`)
assert.equal(localSaveHint(null), '')
assert.equal(
  resumeSaveHint({ lastLocalSaveAt: stamped, diagnoseStatus: 'SAVE', diagnoseId: 'd1' }),
  `${LOCAL_SAVE_HINT_PREFIX} · 18:07:09`,
)
assert.equal(
  resumeSaveHint({ lastLocalSaveAt: null, diagnoseStatus: 'SAVE', diagnoseId: 'd1' }),
  SERVER_SAVED_HINT,
)
assert.equal(resumeSaveHint({ lastLocalSaveAt: null, diagnoseStatus: 'COMMIT', diagnoseId: 'd1' }), '')
assert.equal(resumeSaveHint({ lastLocalSaveAt: null, diagnoseStatus: 'SAVE', diagnoseId: '' }), '')

assert.equal(DIRTY_HINT, '有未暂存修改')
assert.match(LEAVE_CONFIRM_MESSAGE, /未暂存/)
assert.match(REGISTER_SWITCH_CONFIRM_MESSAGE, /切换挂号/)
assert.match(NEW_DRAFT_CONFIRM_MESSAGE, /新建/)

const helper = fs.readFileSync(path.join(root, 'src/workbench/diagnosisDirtyGuard.js'), 'utf8')
assert.equal(helper.includes('fetch('), false)
assert.equal(helper.includes('/api/'), false)
assert.equal(helper.includes('localStorage'), false)
assert.equal(helper.includes('sessionStorage'), false)

const view = fs.readFileSync(path.join(root, 'src/views/DiagnosisView.vue'), 'utf8')
assert.match(view, /diagnosisDirtyGuard/)
assert.match(view, /snapshotOf|isDirty/)
assert.match(view, /onBeforeRouteLeave/)
assert.match(view, /beforeunload/)
assert.match(view, /lastLocalSaveAt/)
assert.match(view, /DIRTY_HINT|有未暂存修改/)
assert.match(view, /resumeSaveHint|saveResumeHint/)
assert.match(helper, /SERVER_SAVED_HINT/)
assert.match(helper, /已暂存（服务端）/)
assert.match(view, /REGISTER_SWITCH_CONFIRM_MESSAGE|LEAVE_CONFIRM_MESSAGE/)
assert.match(view, /createSubmitFeedback/)
assert.match(view, /SubmitStatusBanner/)
assert.equal(view.includes('localStorage'), false)
assert.equal(view.includes('sessionStorage'), false)
// fail path must not wipe list blindly after catch — catch keeps editor
assert.match(view, /submitFeedback\.fail/)
assert.match(view, /captureBaseline|baselineSnapshot/)

// D2 wiring intact
assert.match(view, /saveDiagnose/)
assert.match(view, /amendDiagnose/)

console.log('P1_DIAG_RESUME_OK')
