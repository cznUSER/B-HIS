<script setup>
import { useErrorDialog } from '../components/feedback/useErrorDialog.js'
import { createSubmitFeedback } from '../components/feedback/useSubmitFeedback.js'
import SubmitStatusBanner from '../components/feedback/SubmitStatusBanner.vue'
// 2026-10-15 P2-DIAGNOSIS-UI 诊断工作区：挂号选择、内容保存、提交和作废。
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { onBeforeRouteLeave, useRoute, useRouter } from 'vue-router'
import {
  DIRTY_HINT,
  LEAVE_CONFIRM_MESSAGE,
  REGISTER_SWITCH_CONFIRM_MESSAGE,
  NEW_DRAFT_CONFIRM_MESSAGE,
  snapshotOf,
  isDirty as snapshotIsDirty,
  resumeSaveHint,
} from '../workbench/diagnosisDirtyGuard.js'
import {
  COMMON_DIAGNOSES,
  appendDiagnosisName,
  assessmentIncludesName,
} from '../workbench/commonDiagnoses.js'
import {
  MOCK_ICD_CODES,
  ICD_DEMO_DISCLAIMER,
  EMPTY_ICD_MATCH_MESSAGE,
  filterIcdByKeyword,
  formatIcdEntryToken,
  appendIcdToken,
  assessmentIncludesIcdToken,
} from '../workbench/mockIcdCodes.js'
import {
  DIFF_NEED_TWO_HINT,
  DIFF_NO_CHANGES_HINT,
  canOpenVersionDiff,
  defaultDiffPair,
  diffDiagnosisRecords,
} from '../workbench/diagnosisVersionDiff.js'
import { ApiError } from '../api/errors.js'
import { useMock } from '../api/config.js'
import { amendDiagnose, commitDiagnose, abateDiagnose, listDiagnoses, saveDiagnose } from '../api/diagnosisApi.js'
import { diagnoseCanAbate, diagnoseCanCommit, diagnoseStatusLabel } from '../api/diagnosisContract.js'
import { listOrderRegisters } from '../api/orderApi.js'
import { session } from '../auth/session.js'
import BusinessEmptyState from '../components/BusinessEmptyState.vue'
import BusinessPageHeader from '../components/BusinessPageHeader.vue'
import { recordAction, setDiagnosis, setRegister } from '../workbench/outpatientContext.js'
import { deptName, doctorName, statusLabel as registerStatusLabel } from '../register/master.js'
import { canEnter, findModule } from '../workbench/catalog.js'
import './clinical-design.css'

const route = useRoute()
const router = useRouter()
const mockMode = useMock()
const patientSearchRoute = { name: 'patient-search' }
const registerRoute = { name: 'outpatient-module', params: { module: 'register' } }

const registers = ref([])
const selectedRegisterId = ref('')
const diagnoses = ref([])
const diagnoseId = ref('')
const diagnoseStatus = ref('NEW')
const content = ref('')
const templateCode = ref('GENERAL')
const diagnoseVersion = ref(0)
const signedAt = ref('')
const signedBy = ref('')
const amendSourceId = ref('')
const revisionReason = ref('')
const structured = reactive({
  chiefComplaint: '',
  presentHistory: '',
  pastHistory: '',
  examination: '',
  assessment: '',
  plan: '',
})
const loadingRegisters = ref(false)
const loadingDiagnoses = ref(false)
const saving = ref(false)
const committing = ref(false)
const abating = ref(false)
const errorMessage = ref('')
const infoMessage = ref('')
const registersError = ref('')
const diagnosesError = ref('')
let loadSeq = 0
const submitFeedback = createSubmitFeedback()
let saveAttempt = null
let commitAttempt = null
let abateAttempt = null
const baselineSnapshot = ref(snapshotOf({}))
const lastLocalSaveAt = ref(null)
const lastLocalSaveDiagnoseId = ref('')
let loadedRegisterId = ''
let bypassDirtyGuard = false

const hasAccess = computed(() => canEnter(findModule('diagnosis'), session.user))
const selectedRegister = computed(() => registers.value.find((item) => item.registerId === selectedRegisterId.value) || null)
const busy = computed(() => saving.value || committing.value || abating.value || submitFeedback.state.status === 'submitting')
const canSave = computed(() => Boolean(selectedRegister.value) && !diagnoseId.value && !busy.value)
const canAmend = computed(() => Boolean(diagnoseId.value) && diagnoseStatus.value === 'COMMIT' && !busy.value)
const canCommit = computed(() => Boolean(diagnoseId.value) && diagnoseCanCommit(diagnoseStatus.value) && !busy.value)
const canAbate = computed(() => Boolean(diagnoseId.value) && diagnoseCanAbate(diagnoseStatus.value) && !busy.value)
const editorLocked = computed(() => Boolean(diagnoseId.value) && !['NEW', 'SAVE'].includes(diagnoseStatus.value))
const currentEditorSnapshot = computed(() => snapshotOf({
  structured,
  templateCode: templateCode.value,
  revisionReason: revisionReason.value,
  amendSourceId: amendSourceId.value,
}))
const dirty = computed(() => snapshotIsDirty(baselineSnapshot.value, currentEditorSnapshot.value, { editorLocked: editorLocked.value }))
const saveResumeHint = computed(() => resumeSaveHint({
  lastLocalSaveAt: lastLocalSaveAt.value,
  diagnoseStatus: diagnoseStatus.value,
  diagnoseId: diagnoseId.value,
}))
const serializedContent = computed(() => serializeStructured(structured))
const completion = computed(() => Object.values(structured).filter((value) => value.trim()).length)
const contentTooLong = computed(() => serializedContent.value.length > 2000)

const EMR_TEMPLATES = [
  { value: 'initial', label: '全科初诊', values: { chiefComplaint: '因____就诊____天。', presentHistory: '症状起始于____，主要表现为____，伴随____。', pastHistory: '既往史：____；过敏史：____。', examination: '体温____℃，脉搏____次/分；查体及检查：____。', assessment: '初步诊断：____。', plan: '拟予____；必要时完善____；嘱____后复诊。' } },
  { value: 'follow-up', label: '复诊记录', values: { chiefComplaint: '复诊，诉____。', presentHistory: '上次治疗后症状____，用药依从性____，目前____。', pastHistory: '既往诊疗记录已核对；新增过敏史：____。', examination: '复诊查体及检查：____。', assessment: '目前诊断：____；疗效评价：____。', plan: '继续/调整____；复诊时间____；注意事项____。' } },
  { value: 'empty', label: '空白记录', values: { chiefComplaint: '', presentHistory: '', pastHistory: '', examination: '', assessment: '', plan: '' } },
]
const selectedTemplate = ref('')

const STRUCTURED_FIELDS = [
  { key: 'chiefComplaint', label: '主诉', placeholder: '患者本次就诊最主要的不适或需求' },
  { key: 'presentHistory', label: '现病史', placeholder: '症状起始、变化、伴随表现和既往处理' },
  { key: 'pastHistory', label: '既往史 / 过敏史', placeholder: '既往疾病、手术、用药和过敏情况' },
  { key: 'examination', label: '查体与检查', placeholder: '生命体征、查体所见和已完成的检查' },
  { key: 'assessment', label: '初步诊断', placeholder: '本次就诊的诊断或鉴别诊断' },
  { key: 'plan', label: '处置计划', placeholder: '治疗、用药、检查、复诊和健康教育计划' },
]

function serializeStructured(value) {
  return STRUCTURED_FIELDS.map((field) => `${field.label}：${value[field.key]?.trim() || ''}`).join('\n')
}

function parseStructured(raw) {
  const next = Object.fromEntries(STRUCTURED_FIELDS.map((field) => [field.key, '']))
  const text = typeof raw === 'string' ? raw : ''
  let matched = false
  for (const field of STRUCTURED_FIELDS) {
    const line = text.split(/\r?\n/).find((item) => item.startsWith(`${field.label}：`) || item.startsWith(`${field.label}:`))
    if (!line) continue
    matched = true
    next[field.key] = line.slice(field.label.length + 1).trim()
  }
  if (!matched && text.trim()) next.assessment = text.trim()
  Object.assign(structured, next)
}

const icdKeyword = ref('')

const icdFindResults = computed(() => filterIcdByKeyword(MOCK_ICD_CODES, icdKeyword.value))

const diffOpen = ref(false)
const diffLeftId = ref('')
const diffRightId = ref('')
const diffCanOpen = computed(() => canOpenVersionDiff(diagnoses.value))
const diffLeftRow = computed(() => diagnoses.value.find((row) => row.diagnoseId === diffLeftId.value) || null)
const diffRightRow = computed(() => diagnoses.value.find((row) => row.diagnoseId === diffRightId.value) || null)
const diffResult = computed(() => {
  if (!diffLeftRow.value || !diffRightRow.value) return null
  if (diffLeftRow.value.diagnoseId === diffRightRow.value.diagnoseId) return null
  return diffDiagnosisRecords(diffLeftRow.value, diffRightRow.value)
})

function openVersionDiff() {
  if (!diffCanOpen.value) return
  const pair = defaultDiffPair(diagnoses.value, diagnoseId.value)
  if (!pair) return
  diffLeftId.value = pair.left?.diagnoseId || ''
  diffRightId.value = pair.right?.diagnoseId || ''
  diffOpen.value = true
}

function closeVersionDiff() {
  diffOpen.value = false
}

function diagnosisDiffOptionLabel(row) {
  if (!row) return ''
  const ver = row.version || 1
  const status = diagnoseStatusLabel(row.status)
  const id = String(row.diagnoseId || '').slice(0, 8)
  return `v${ver} · ${status} · ${id}`
}

function applyCommonDiagnosis(name) {
  if (editorLocked.value || busy.value) return
  structured.assessment = appendDiagnosisName(structured.assessment, name)
}

function isCommonDiagnosisSelected(name) {
  return assessmentIncludesName(structured.assessment, name)
}

function applyIcdEntry(entry) {
  if (editorLocked.value || busy.value || !entry) return
  structured.assessment = appendIcdToken(structured.assessment, entry)
}

function isIcdEntrySelected(entry) {
  return assessmentIncludesIcdToken(structured.assessment, entry)
}

function applyTemplate() {
  const template = EMR_TEMPLATES.find((item) => item.value === selectedTemplate.value)
  if (!template || editorLocked.value || busy.value) return
  Object.assign(structured, template.values)
  templateCode.value = template.value.toUpperCase()
  selectedTemplate.value = ''
}

function queryText(value) {
  const raw = Array.isArray(value) ? value[0] : value
  return typeof raw === 'string' ? raw.trim() : ''
}

function registerName(row) {
  return row?.patientName || row?.patientNo || row?.patientId || '未命名患者'
}

function registerMeta(row) {
  if (!row) return ''
  const dept = mockMode ? deptName(row.departmentId) : (row.departmentName || row.departmentId || '—')
  const doctor = mockMode ? doctorName(row.doctorId) : (row.doctorName || row.doctorId || '—')
  return `${dept} · ${doctor} · ${registerStatusLabel(row.status)}`
}

function clearNotice() {
  errorMessage.value = ''
  infoMessage.value = ''
}

function captureBaseline() {
  baselineSnapshot.value = snapshotOf({
    structured,
    templateCode: templateCode.value,
    revisionReason: revisionReason.value,
    amendSourceId: amendSourceId.value,
  })
}

function confirmLeaveIfDirty(message = LEAVE_CONFIRM_MESSAGE) {
  if (bypassDirtyGuard || !dirty.value) return true
  return window.confirm(message)
}

function applyDiagnosis(row) {
  if (!row) {
    diagnoseId.value = ''
    diagnoseStatus.value = 'NEW'
    content.value = ''
    templateCode.value = 'GENERAL'
    diagnoseVersion.value = 0
    signedAt.value = ''
    signedBy.value = ''
    amendSourceId.value = ''
    revisionReason.value = ''
    parseStructured('')
    captureBaseline()
    return
  }
  diagnoseId.value = row.diagnoseId
  diagnoseStatus.value = row.status
  content.value = row.content || ''
  templateCode.value = row.templateCode || 'GENERAL'
  diagnoseVersion.value = row.version || 1
  signedAt.value = row.signedAt || ''
  signedBy.value = row.signedBy || ''
  amendSourceId.value = ''
  revisionReason.value = ''
  if (row.structured && Object.values(row.structured).some((value) => String(value || '').trim())) Object.assign(structured, row.structured)
  else parseStructured(row.content || '')
  setDiagnosis(row, diagnoses.value)
  captureBaseline()
}

function resetDraft({ force = false } = {}) {
  if (!force && !confirmLeaveIfDirty(NEW_DRAFT_CONFIRM_MESSAGE)) return false
  clearNotice()
  saveAttempt = null
  commitAttempt = null
  abateAttempt = null
  submitFeedback.reset()
  lastLocalSaveAt.value = null
  lastLocalSaveDiagnoseId.value = ''
  applyDiagnosis(null)
  return true
}

function requestNewDraft() {
  resetDraft()
}

function beginAmend() {
  const source = diagnoses.value.find((row) => row.diagnoseId === diagnoseId.value)
  if (!source || source.status !== 'COMMIT' || busy.value) return
  applyDiagnosis(source)
  amendSourceId.value = source.diagnoseId
  diagnoseId.value = ''
  diagnoseStatus.value = 'NEW'
  diagnoseVersion.value = 0
  signedAt.value = ''
  signedBy.value = ''
  revisionReason.value = ''
  infoMessage.value = `正在修订 v${source.version || 1}，原版本仍会保留。`
}

async function loadRegisters() {
  loadingRegisters.value = true
  registersError.value = ''
  try {
    const listed = await listOrderRegisters(session.accessToken)
    registers.value = listed.items.filter((item) => item.status !== 'BACK')
    const fromQuery = queryText(route.query.registerId)
    const preferred = registers.value.find((item) => item.registerId === fromQuery)
      || registers.value.find((item) => ['DIAGNOSING', 'NORMAL', 'USED'].includes(item.status))
      || registers.value[0]
    if (preferred) setRegister(preferred)
    if (preferred && preferred.registerId !== selectedRegisterId.value) {
      selectedRegisterId.value = preferred.registerId
      await router.replace({ name: 'outpatient-module', params: { module: 'diagnosis' }, query: { registerId: preferred.registerId } })
    }
    if (!preferred) resetDraft({ force: true })
  } catch (error) {
    registersError.value = error instanceof ApiError ? error.message : '暂时无法读取当日挂号'
  } finally {
    loadingRegisters.value = false
  }
}

async function loadDiagnoses(registerId) {
  const id = queryText(registerId)
  const seq = ++loadSeq
  diagnoses.value = []
  diagnosesError.value = ''
  bypassDirtyGuard = true
  resetDraft({ force: true })
  bypassDirtyGuard = false
  loadedRegisterId = id
  if (!id) return
  loadingDiagnoses.value = true
  try {
    const listed = await listDiagnoses(session.accessToken, id)
    if (seq !== loadSeq) return
    diagnoses.value = listed.items
    const latest = listed.items[listed.items.length - 1]
    if (latest) applyDiagnosis(latest)
    else setDiagnosis(null, listed.items)
  } catch (error) {
    if (seq !== loadSeq) return
    diagnosesError.value = error instanceof ApiError ? error.message : '暂时无法读取诊断记录'
  } finally {
    if (seq === loadSeq) loadingDiagnoses.value = false
  }
}

async function chooseRegister() {
  const id = selectedRegisterId.value
  if (id === loadedRegisterId) return
  if (!confirmLeaveIfDirty(REGISTER_SWITCH_CONFIRM_MESSAGE)) {
    selectedRegisterId.value = loadedRegisterId
    return
  }
  clearNotice()
  bypassDirtyGuard = true
  await router.replace({ name: 'outpatient-module', params: { module: 'diagnosis' }, query: id ? { registerId: id } : {} })
  await loadDiagnoses(id)
  bypassDirtyGuard = false
}

function chooseDiagnosis(row) {
  if (row?.diagnoseId && row.diagnoseId === diagnoseId.value) return
  if (!confirmLeaveIfDirty(LEAVE_CONFIRM_MESSAGE)) return
  clearNotice()
  applyDiagnosis(row)
  if (row?.status === 'SAVE' && row.diagnoseId !== lastLocalSaveDiagnoseId.value) {
    lastLocalSaveAt.value = null
  }
}

async function onSave() {
  if (!canSave.value) return
  clearNotice()
  const nextContent = serializedContent.value.trim()
  if (!Object.values(structured).some((value) => value.trim())) {
    errorMessage.value = '请至少填写一项结构化诊疗记录'
    return
  }
  if (contentTooLong.value) {
    errorMessage.value = '结构化诊疗记录超过 2000 个字符，请精简内容'
    return
  }
  const isAmend = Boolean(amendSourceId.value)
  if (isAmend && !revisionReason.value.trim()) {
    errorMessage.value = '请填写病历修订原因'
    return
  }
  content.value = nextContent
  const action = isAmend ? 'amend' : 'save'
  const payload = isAmend
    ? { content: nextContent, structured: { ...structured }, templateCode: templateCode.value, revisionReason: revisionReason.value }
    : { content: nextContent, structured: { ...structured }, templateCode: templateCode.value }
  const fingerprint = JSON.stringify({
    action,
    registerId: selectedRegisterId.value,
    amendSourceId: amendSourceId.value || '',
    payload,
  })
  if (!saveAttempt || saveAttempt.fingerprint !== fingerprint) saveAttempt = { fingerprint, key: crypto.randomUUID() }
  saving.value = true
  submitFeedback.beginAttempt(action)
  try {
    const saved = isAmend
      ? await amendDiagnose(session.accessToken, amendSourceId.value, payload, saveAttempt.key)
      : await saveDiagnose(session.accessToken, selectedRegisterId.value, payload, saveAttempt.key)
    applyDiagnosis(saved)
    diagnoses.value = [...diagnoses.value.filter((row) => row.diagnoseId !== saved.diagnoseId), saved]
    setDiagnosis(saved, diagnoses.value)
    lastLocalSaveAt.value = new Date()
    lastLocalSaveDiagnoseId.value = saved.diagnoseId || ''
    captureBaseline()
    recordAction('诊断已暂存', saved.diagnoseId || '')
    saveAttempt = null
    const msg = (isAmend ? '病历修订已暂存' : '诊断已暂存') + '，诊断号 ' + (saved.diagnoseId || '')
    infoMessage.value = ''
    submitFeedback.succeed({ diagnoseId: saved.diagnoseId || '' }, msg)
  } catch (error) {
    const message = error instanceof ApiError ? error.message : '暂存诊断失败，请稍后重试'
    errorMessage.value = message
    submitFeedback.fail(message, error?.code || '')
  } finally {
    saving.value = false
  }
}

async function onCommit() {
  if (!canCommit.value) return
  clearNotice()
  const fingerprint = JSON.stringify({ action: 'commit', diagnoseId: diagnoseId.value })
  if (!commitAttempt || commitAttempt.fingerprint !== fingerprint) commitAttempt = { fingerprint, key: crypto.randomUUID() }
  committing.value = true
  submitFeedback.beginAttempt('commit')
  try {
    const committed = await commitDiagnose(session.accessToken, diagnoseId.value, commitAttempt.key)
    applyDiagnosis(committed)
    diagnoses.value = diagnoses.value.map((row) => row.diagnoseId === committed.diagnoseId ? committed : row)
    setDiagnosis(committed, diagnoses.value)
    recordAction('诊断已提交', committed.diagnoseId || '')
    commitAttempt = null
    infoMessage.value = ''
    submitFeedback.succeed({ diagnoseId: committed.diagnoseId || '' }, '诊断已提交，诊断号 ' + (committed.diagnoseId || ''))
  } catch (error) {
    const message = error instanceof ApiError ? error.message : '提交诊断失败，请稍后重试'
    errorMessage.value = message
    submitFeedback.fail(message, error?.code || '')
  } finally {
    committing.value = false
  }
}

async function onAbate() {
  if (!canAbate.value) return
  clearNotice()
  const fingerprint = JSON.stringify({ action: 'abate', diagnoseId: diagnoseId.value })
  if (!abateAttempt || abateAttempt.fingerprint !== fingerprint) abateAttempt = { fingerprint, key: crypto.randomUUID() }
  abating.value = true
  submitFeedback.beginAttempt('abate')
  try {
    const abated = await abateDiagnose(session.accessToken, diagnoseId.value, abateAttempt.key)
    applyDiagnosis(abated)
    diagnoses.value = diagnoses.value.map((row) => row.diagnoseId === abated.diagnoseId ? abated : row)
    setDiagnosis(abated, diagnoses.value)
    recordAction('诊断已作废', abated.diagnoseId || '')
    abateAttempt = null
    infoMessage.value = ''
    submitFeedback.succeed({ diagnoseId: abated.diagnoseId || '' }, '诊断已作废，诊断号 ' + (abated.diagnoseId || ''))
  } catch (error) {
    const message = error instanceof ApiError ? error.message : '作废诊断失败，请稍后重试'
    errorMessage.value = message
    submitFeedback.fail(message, error?.code || '')
  } finally {
    abating.value = false
  }
}

watch(() => queryText(route.query.registerId), (id) => {
  if (id && id !== selectedRegisterId.value) selectedRegisterId.value = id
  if (bypassDirtyGuard) {
    if (id) void loadDiagnoses(id)
    else {
      bypassDirtyGuard = true
      resetDraft({ force: true })
      bypassDirtyGuard = false
      loadedRegisterId = ''
    }
    return
  }
  if (id === loadedRegisterId) return
  if (dirty.value && !confirmLeaveIfDirty(REGISTER_SWITCH_CONFIRM_MESSAGE)) {
    selectedRegisterId.value = loadedRegisterId
    void router.replace({
      name: 'outpatient-module',
      params: { module: 'diagnosis' },
      query: loadedRegisterId ? { registerId: loadedRegisterId } : {},
    })
    return
  }
  if (id) void loadDiagnoses(id)
  else {
    bypassDirtyGuard = true
    resetDraft({ force: true })
    bypassDirtyGuard = false
    loadedRegisterId = ''
  }
}, { immediate: true })

function beforeUnloadGuard(event) {
  if (!dirty.value) return
  event.preventDefault()
  event.returnValue = ''
}

onMounted(() => {
  window.addEventListener('beforeunload', beforeUnloadGuard)
  void loadRegisters()
})
onBeforeUnmount(() => window.removeEventListener('beforeunload', beforeUnloadGuard))
onBeforeRouteLeave(() => {
  if (!dirty.value || window.confirm(LEAVE_CONFIRM_MESSAGE)) return true
  return false
})

useErrorDialog([
  () => errorMessage.value,
  () => registersError.value,
  () => diagnosesError.value,
])
</script>

<template>
  <section class="diagnosis-page" aria-labelledby="diagnosis-title">
    <p class="crumb"><RouterLink :to="{ name: 'home' }">首页</RouterLink><span aria-hidden="true"> / </span><RouterLink :to="{ name: 'workbench' }">门诊业务</RouterLink><span aria-hidden="true"> / </span><span>诊断</span></p>
    <BusinessPageHeader title-id="diagnosis-title" title="诊断" eyebrow="门诊业务 · 可选录入" icon="diagnosis" description="按挂号记录诊断内容，可先暂存，再提交或作废。临床规则当前保持关闭。">
      <template #actions><span class="diagnosis-scope">当前账号：{{ hasAccess ? '门诊医生' : '只读' }}</span></template>
    </BusinessPageHeader>

    <BusinessEmptyState v-if="!hasAccess" title="无权限" description="当前账号不能录入诊断，请使用门诊医生或系统账号。" icon="diagnosis" eyebrow="访问权限" />
    <BusinessEmptyState
      v-else-if="!loadingRegisters && !registers.length"
      variant="revisit"
      module-label="诊断"
      queue-label="今日待诊"
      title="选择一位患者开始诊断"
      description="先完成患者建档和挂号，患者会按就诊顺序进入左侧待诊队列。"
      empty-label="暂无待诊患者"
      empty-description="完成建档和挂号后，待诊患者会自动出现在这里。"
      primary-text="去挂号"
      :primary-to="registerRoute"
      secondary-text="患者查询"
      :secondary-to="patientSearchRoute"
      test-id="diagnosis-queue-workspace"
    />
    <div v-else class="diagnosis-shell">
      <section class="diagnosis-selector" aria-label="挂号选择">
        <label class="diagnosis-register-field"><span>当前挂号</span><select v-model="selectedRegisterId" :disabled="loadingRegisters || busy" @change="chooseRegister"><option value="">请选择挂号</option><option v-for="row in registers" :key="row.registerId" :value="row.registerId">{{ registerName(row) }} · {{ row.registerNo || row.registerId.slice(0, 8) }}</option></select></label>
        <p v-if="selectedRegister" class="diagnosis-register-meta">{{ registerMeta(selectedRegister) }}</p>
        <button class="button button-ghost button-compact" type="button" :disabled="busy || !selectedRegister" @click="requestNewDraft">新建诊断</button>
      </section>

      <p v-if="registersError" class="diagnosis-alert diagnosis-alert-error" role="alert">{{ registersError }}</p>
      <p v-if="diagnosesError" class="diagnosis-alert diagnosis-alert-error" role="alert">{{ diagnosesError }}</p>
      <p v-if="errorMessage && submitFeedback.state.status !== 'error'" class="diagnosis-alert diagnosis-alert-error" role="alert">{{ errorMessage }}</p>
      <p v-if="infoMessage && submitFeedback.state.status !== 'success' && submitFeedback.state.status !== 'error'" class="diagnosis-alert diagnosis-alert-success" role="status">{{ infoMessage }}</p>
      <p v-if="dirty" class="diagnosis-alert diagnosis-alert-dirty" role="status" data-testid="diagnosis-dirty-hint">{{ DIRTY_HINT }}</p>
      <p v-else-if="saveResumeHint" class="diagnosis-alert diagnosis-alert-resume" role="status" data-testid="diagnosis-save-hint">{{ saveResumeHint }}</p>
      <SubmitStatusBanner :feedback="submitFeedback.state" />

      <div class="diagnosis-workspace">
        <aside class="diagnosis-history" aria-label="诊断记录">
          <div class="diagnosis-section-head"><div><p class="diagnosis-kicker">RECORDS</p><h2>本次诊断</h2></div><span>{{ diagnoses.length }} 条</span></div>
          <div class="diagnosis-version-diff-toolbar">
            <button
              type="button"
              class="button button-ghost button-compact"
              data-testid="diagnosis-version-diff-open"
              :disabled="!diffCanOpen || busy"
              :title="diffCanOpen ? '对比两个历史版本' : DIFF_NEED_TWO_HINT"
              @click="openVersionDiff"
            >对比版本</button>
            <p
              v-if="!diffCanOpen"
              class="diagnosis-muted diagnosis-version-diff-need-two"
              data-testid="diagnosis-version-diff-need-two"
            >{{ DIFF_NEED_TWO_HINT }}</p>
          </div>
          <div
            v-if="diffOpen"
            class="diagnosis-version-diff-panel"
            data-testid="diagnosis-version-diff-panel"
            role="region"
            aria-label="诊断版本对比"
          >
            <div class="diagnosis-version-diff-panel-head">
              <strong>版本对比</strong>
              <button type="button" class="button button-ghost button-compact" data-testid="diagnosis-version-diff-close" @click="closeVersionDiff">关闭</button>
            </div>
            <div class="diagnosis-version-diff-pickers">
              <label>
                <span>版本 A</span>
                <select v-model="diffLeftId" data-testid="diagnosis-version-diff-left">
                  <option v-for="row in diagnoses" :key="'L-' + row.diagnoseId" :value="row.diagnoseId">{{ diagnosisDiffOptionLabel(row) }}</option>
                </select>
              </label>
              <label>
                <span>版本 B</span>
                <select v-model="diffRightId" data-testid="diagnosis-version-diff-right">
                  <option v-for="row in diagnoses" :key="'R-' + row.diagnoseId" :value="row.diagnoseId">{{ diagnosisDiffOptionLabel(row) }}</option>
                </select>
              </label>
            </div>
            <p
              v-if="diffResult && diffResult.identical"
              class="diagnosis-muted"
              data-testid="diagnosis-version-diff-empty"
            >{{ DIFF_NO_CHANGES_HINT }}</p>
            <ul v-else-if="diffResult" class="diagnosis-version-diff-fields">
              <li
                v-for="field in diffResult.fields"
                :key="field.key"
                class="diagnosis-version-diff-field"
                :class="{ 'is-changed': field.changed }"
                :data-testid="'diagnosis-version-diff-field-' + field.key"
              >
                <span class="diagnosis-version-diff-label">{{ field.label }}</span>
                <span class="diagnosis-version-diff-side">
                  <span class="diagnosis-version-diff-side-tag">A</span>
                  <span>{{ field.left || '（空）' }}</span>
                </span>
                <span class="diagnosis-version-diff-side">
                  <span class="diagnosis-version-diff-side-tag">B</span>
                  <span>{{ field.right || '（空）' }}</span>
                </span>
              </li>
            </ul>
            <p v-else class="diagnosis-muted">请选择两个不同的诊断版本。</p>
          </div>
          <p v-if="loadingDiagnoses" class="diagnosis-muted">读取中…</p>
          <p v-else-if="!diagnoses.length" class="diagnosis-muted">尚无已保存诊断。</p>
          <button v-for="row in diagnoses" :key="row.diagnoseId" class="diagnosis-record" :class="{ 'is-active': row.diagnoseId === diagnoseId }" type="button" @click="chooseDiagnosis(row)">
            <span class="diagnosis-record-status"><span>{{ diagnoseStatusLabel(row.status) }}</span><span>v{{ row.version || 1 }} · {{ row.templateCode || 'GENERAL' }}</span></span><span v-if="row.supersedesDiagnoseId" class="diagnosis-record-revision">修订自前一版本 · {{ row.revisionReason || '已记录原因' }}</span><span class="diagnosis-record-text">{{ row.content }}</span>
          </button>
        </aside>

        <main class="diagnosis-editor" aria-label="结构化诊疗记录编辑">
          <div class="diagnosis-section-head"><div><p class="diagnosis-kicker">EDITOR</p><h2>{{ amendSourceId ? '病历修订草稿' : (diagnoseId ? '诊断详情' : '结构化诊疗记录') }}</h2></div><div class="diagnosis-editor-meta"><span class="diagnosis-status" :class="`status-${diagnoseStatus.toLowerCase()}`">{{ diagnoseStatusLabel(diagnoseStatus) }}</span><span v-if="diagnoseId" class="diagnosis-version">v{{ diagnoseVersion || 1 }} · {{ templateCode }}</span><span v-if="amendSourceId" class="diagnosis-version">修订前 v{{ diagnoses.find((row) => row.diagnoseId === amendSourceId)?.version || 1 }}</span></div></div>
          <p class="diagnosis-structured-note">结构化栏目、所用模板和版本会随诊断记录保存；已提交版本保持不可变，修订会生成新草稿并保留前序记录和原因。</p>
          <div class="diagnosis-template-bar"><label><span>记录模板</span><select v-model="selectedTemplate" :disabled="editorLocked || busy" @change="applyTemplate"><option value="">选择模板快速起草</option><option v-for="template in EMR_TEMPLATES" :key="template.value" :value="template.value">{{ template.label }}</option></select></label><label v-if="amendSourceId" class="diagnosis-reason-field"><span>修订原因 *</span><input v-model="revisionReason" maxlength="256" :disabled="busy" placeholder="说明为什么需要修订" /></label><span>模板只填充草稿，不会绕过保存、提交和权限校验。</span><span v-if="signedAt" class="diagnosis-signed">已签名 · {{ signedAt }}<template v-if="signedBy"> · {{ signedBy }}</template></span></div>
          <div class="diagnosis-structured-grid">
            <div
              v-for="field in STRUCTURED_FIELDS"
              :key="field.key"
              class="diagnosis-content-field"
              :class="{ 'diagnosis-content-field-wide': ['presentHistory', 'examination', 'plan'].includes(field.key) }"
            >
              <label class="diagnosis-content-field-label">
                <span>{{ field.label }}</span>
                <textarea v-model="structured[field.key]" :readonly="editorLocked || busy" maxlength="320" rows="3" :placeholder="field.placeholder" />
              </label>
              <div
                v-if="field.key === 'assessment'"
                class="diagnosis-common-diagnoses"
                data-testid="diagnosis-common-chips"
                aria-label="常用诊断"
              >
                <span class="diagnosis-common-label">常用诊断</span>
                <button
                  v-for="name in COMMON_DIAGNOSES"
                  :key="name"
                  type="button"
                  class="diagnosis-common-chip"
                  data-testid="diagnosis-common-chip"
                  :disabled="editorLocked || busy || isCommonDiagnosisSelected(name)"
                  :title="isCommonDiagnosisSelected(name) ? '已在初步诊断中' : ('填入 ' + name)"
                  @click="applyCommonDiagnosis(name)"
                >+ {{ name }}</button>
              </div>
              <div
                v-if="field.key === 'assessment'"
                class="diagnosis-icd-find"
                data-testid="diagnosis-icd-find"
                aria-label="ICD 检索（演示）"
              >
                <div class="diagnosis-icd-find-head">
                  <span class="diagnosis-icd-find-label">ICD 检索（演示）</span>
                  <span class="diagnosis-icd-find-disclaimer" data-testid="icd-find-disclaimer">{{ ICD_DEMO_DISCLAIMER }}</span>
                </div>
                <input
                  v-model="icdKeyword"
                  class="diagnosis-icd-find-input"
                  data-testid="icd-find-input"
                  type="search"
                  maxlength="64"
                  placeholder="关键词过滤演示码表（编码/名称/别名）"
                  :disabled="editorLocked || busy"
                  autocomplete="off"
                />
                <ul
                  class="diagnosis-icd-find-results"
                  data-testid="icd-find-results"
                  role="listbox"
                  aria-label="演示 ICD 结果"
                >
                  <li v-if="!icdFindResults.length" class="diagnosis-icd-find-empty" data-testid="icd-find-empty">{{ EMPTY_ICD_MATCH_MESSAGE }}</li>
                  <li
                    v-for="entry in icdFindResults"
                    :key="entry.code"
                    class="diagnosis-icd-find-item"
                  >
                    <button
                      type="button"
                      class="diagnosis-icd-find-item-btn"
                      :data-testid="'icd-find-item-' + entry.code"
                      :disabled="editorLocked || busy || isIcdEntrySelected(entry)"
                      :title="isIcdEntrySelected(entry) ? '已在初步诊断中' : ('填入 ' + formatIcdEntryToken(entry))"
                      @click="applyIcdEntry(entry)"
                    >
                      <span class="diagnosis-icd-find-code">{{ entry.code }}</span>
                      <span class="diagnosis-icd-find-name">{{ entry.name }}</span>
                    </button>
                  </li>
                </ul>
              </div>
            </div>
          </div>
          <div class="diagnosis-editor-foot"><span :class="{ 'diagnosis-length-error': contentTooLong }">{{ serializedContent.length }}/2000 · 已填写 {{ completion }}/{{ STRUCTURED_FIELDS.length }} 项</span><span v-if="editorLocked">已提交或已作废的诊断不能直接修改，可从已提交版本发起修订</span><span v-else-if="amendSourceId">修订草稿将保存为 v{{ (diagnoses.length || 0) + 1 }}</span></div>
          <div class="diagnosis-actions"><button v-if="canAmend" class="button button-ghost" type="button" :disabled="busy" @click="beginAmend">修订此版本</button><button v-if="!diagnoseId" class="button button-primary" type="button" :disabled="!canSave" @click="onSave">{{ saving ? '保存中…' : (amendSourceId ? '保存修订' : '保存诊断') }}</button><button v-if="canCommit" class="button button-primary" type="button" :disabled="busy" @click="onCommit">{{ committing ? '提交中…' : '提交' }}</button><button v-if="canAbate" class="button button-ghost danger" type="button" :disabled="busy" @click="onAbate">{{ abating ? '作废中…' : '作废' }}</button></div>
        </main>
      </div>
    </div>
  </section>
</template>

<style scoped>
.diagnosis-page { display: flex; flex-direction: column; gap: 18px; max-width: 1480px; margin: 0 auto; }
.diagnosis-scope { display: inline-flex; align-items: center; min-height: 34px; padding: 0 12px; border: 1px solid #d9e9df; border-radius: 10px; color: #4e7966; background: #f1f8f3; font-size: 11px; }
.diagnosis-selector { display: flex; align-items: flex-end; gap: 16px; flex-wrap: wrap; padding: 18px 20px; border: 1px solid #dfebe2; border-radius: 14px; background: #fff; }
.diagnosis-register-field { display: flex; flex-direction: column; gap: 7px; min-width: min(420px, 100%); color: #687f73; font-size: 11px; }
.diagnosis-register-field select { width: 100%; min-width: 0; min-height: 40px; padding: 0 12px; border: 1px solid #cfe2d5; border-radius: 9px; color: #304d40; background: #fbfefc; }
.diagnosis-register-meta { align-self: center; color: #7c9185; font-size: 12px; }
.diagnosis-alert { padding: 11px 14px; border-radius: 10px; font-size: 12px; }
.diagnosis-alert-error { color: #a94e4e; background: #fff3f2; border: 1px solid #f0d6d4; }
.diagnosis-alert-success { color: #357c61; background: #edf8f0; border: 1px solid #d1e8d8; }
.diagnosis-alert-dirty { color: #8a6d3b; background: #fff8e8; border: 1px solid #f0e0b8; }
.diagnosis-alert-resume { color: #3f6f5c; background: #f2f8f4; border: 1px solid #d5e7dc; }
.diagnosis-workspace { display: grid; grid-template-columns: minmax(240px, 320px) minmax(0, 1fr); gap: 16px; }
.diagnosis-history, .diagnosis-editor { min-width: 0; padding: 20px; border: 1px solid #dfebe2; border-radius: 14px; background: #fff; }
.diagnosis-section-head { display: flex; align-items: flex-start; justify-content: space-between; gap: 10px; margin-bottom: 16px; }
.diagnosis-section-head h2 { color: #355448; font-size: 16px; font-weight: 650; }
.diagnosis-section-head > span { color: #8ba095; font-size: 11px; }
.diagnosis-editor-meta { display: flex; align-items: center; gap: 7px; flex-wrap: wrap; justify-content: flex-end; }
.diagnosis-editor-meta > span { color: #8ba095; font-size: 11px; }
.diagnosis-editor-meta .diagnosis-status { color: #4d8b6d; }
.diagnosis-version { padding: 5px 8px; border: 1px solid #dcebe1; border-radius: 7px; background: #f8fcf9; }
.diagnosis-kicker { margin-bottom: 5px; color: #84a18f; font-size: 9px; letter-spacing: .12em; }
.diagnosis-muted { padding: 24px 5px; color: #99a89f; font-size: 12px; line-height: 1.7; }
.diagnosis-record { display: flex; flex-direction: column; align-items: flex-start; width: 100%; gap: 7px; margin-bottom: 8px; padding: 12px; text-align: left; border: 1px solid #e1ebe4; border-radius: 10px; color: #526c5d; background: #fbfefc; cursor: pointer; }
.diagnosis-record:hover, .diagnosis-record.is-active { border-color: #9ed1b5; background: #f0f9f2; }
.diagnosis-record-status { display: flex; justify-content: space-between; width: 100%; gap: 8px; color: #4f9879; font-size: 10px; }
.diagnosis-record-status > span:last-child { color: #8a9d92; white-space: nowrap; }
.diagnosis-record-revision { color: #8d7355; font-size: 10px; line-height: 1.45; }
.diagnosis-record-text { display: -webkit-box; overflow: hidden; color: #3f594c; font-size: 12px; line-height: 1.6; -webkit-box-orient: vertical; -webkit-line-clamp: 3; }
.diagnosis-status { min-height: 24px; padding: 5px 9px; border-radius: 7px; background: #edf6ef; color: #4d8b6d !important; }
.status-commit { background: #e8f2fa; color: #4b789c !important; }.status-abate { background: #f8eeee; color: #a26a6a !important; }
.diagnosis-structured-note { margin: -4px 0 13px; padding: 9px 11px; border-left: 3px solid #a7d6b7; color: #6f877b; background: #f4faf5; font-size: 11px; line-height: 1.6; }
.diagnosis-template-bar { display: flex; align-items: flex-end; justify-content: space-between; gap: 12px; margin-bottom: 13px; padding: 10px 11px; border: 1px solid #e0ece3; border-radius: 9px; background: #fbfefc; color: #8a9d92; font-size: 10px; }
.diagnosis-template-bar label { display: flex; align-items: center; gap: 8px; color: #678074; font-size: 11px; }
.diagnosis-template-bar select { min-height: 32px; padding: 0 9px; border: 1px solid #cfe2d5; border-radius: 7px; color: #355746; background: #fff; }
.diagnosis-reason-field { display: flex; align-items: center; gap: 8px; color: #678074; font-size: 11px; }
.diagnosis-reason-field input { min-height: 32px; min-width: 220px; padding: 0 9px; border: 1px solid #cfe2d5; border-radius: 7px; color: #355746; background: #fff; font: inherit; }
.diagnosis-signed { color: #4d8b6d; white-space: nowrap; }
.diagnosis-structured-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 12px; }
.diagnosis-content-field-wide { grid-column: 1 / -1; }
.diagnosis-content-field { display: flex; flex-direction: column; gap: 8px; color: #667f72; font-size: 11px; }
.diagnosis-content-field-label { display: flex; flex-direction: column; gap: 8px; }
.diagnosis-common-diagnoses { display: flex; flex-wrap: wrap; align-items: center; gap: 8px; margin-top: 2px; }
.diagnosis-common-label { color: #84a18f; font-size: 10px; letter-spacing: .04em; }
.diagnosis-common-chip { border: 0; padding: 2px 0; color: #1e6e5d; background: transparent; font: inherit; font-size: 11px; cursor: pointer; }
.diagnosis-common-chip:disabled { color: #a0b0ab; cursor: default; }
.diagnosis-icd-find { display: flex; flex-direction: column; gap: 6px; margin-top: 4px; padding: 8px 10px; border: 1px dashed #cfe2d5; border-radius: 10px; background: #f8fcf9; }
.diagnosis-icd-find-head { display: flex; flex-wrap: wrap; align-items: baseline; gap: 8px; }
.diagnosis-icd-find-label { color: #4e7966; font-size: 10px; letter-spacing: .04em; font-weight: 600; }
.diagnosis-icd-find-disclaimer { color: #9a7a3a; font-size: 10px; }
.diagnosis-icd-find-input { width: 100%; min-height: 32px; padding: 0 10px; border: 1px solid #cfe2d5; border-radius: 8px; color: #314e41; background: #fff; font: inherit; font-size: 12px; }
.diagnosis-icd-find-input:disabled { color: #8a9b93; background: #f3f6f4; }
.diagnosis-icd-find-results { list-style: none; margin: 0; padding: 0; max-height: 148px; overflow: auto; display: flex; flex-direction: column; gap: 2px; }
.diagnosis-icd-find-empty { color: #95a69b; font-size: 11px; padding: 4px 2px; }
.diagnosis-icd-find-item-btn { display: flex; gap: 10px; width: 100%; align-items: baseline; border: 0; padding: 4px 2px; background: transparent; color: #1e6e5d; font: inherit; font-size: 11px; text-align: left; cursor: pointer; }
.diagnosis-icd-find-item-btn:disabled { color: #a0b0ab; cursor: default; }
.diagnosis-icd-find-code { flex: 0 0 auto; min-width: 4.5em; color: #5b7d6c; font-variant-numeric: tabular-nums; }
.diagnosis-icd-find-name { flex: 1 1 auto; }
.diagnosis-content-field textarea { width: 100%; min-height: 76px; resize: vertical; padding: 10px 11px; border: 1px solid #cfe2d5; border-radius: 10px; color: #314e41; background: #fcfefd; font: inherit; line-height: 1.65; }
.diagnosis-content-field textarea:focus { outline: 2px solid #a9d8bc; outline-offset: 1px; }
.diagnosis-content-field textarea[readonly] { color: #768a7e; background: #f5f8f5; }
.diagnosis-editor-foot { display: flex; justify-content: space-between; gap: 12px; margin-top: 7px; color: #95a69b; font-size: 10px; }
.diagnosis-length-error { color: #a34f4f; }
.diagnosis-actions { display: flex; gap: 10px; flex-wrap: wrap; margin-top: 22px; }
.diagnosis-actions .danger { color: #a25f5f; border-color: #eed8d5; background: #fff8f7; }
@media (max-width: 840px) { .diagnosis-workspace { grid-template-columns: 1fr; } .diagnosis-history { order: 2; } .diagnosis-editor { order: 1; } .diagnosis-structured-grid { grid-template-columns: 1fr; } .diagnosis-content-field-wide { grid-column: auto; } }
@media (max-width: 600px) { .diagnosis-selector { align-items: stretch; flex-direction: column; } .diagnosis-register-field { min-width: 0; } .diagnosis-register-meta { align-self: flex-start; } .diagnosis-template-bar { align-items: stretch; flex-direction: column; } .diagnosis-template-bar label { flex-direction: column; align-items: stretch; } .diagnosis-reason-field input { min-width: 0; } }

.diagnosis-version-diff-toolbar { display: flex; flex-wrap: wrap; align-items: center; gap: 8px; margin: 0 0 10px; }
.diagnosis-version-diff-need-two { margin: 0; font-size: 11px; }
.diagnosis-version-diff-panel { display: flex; flex-direction: column; gap: 8px; margin: 0 0 12px; padding: 10px; border: 1px dashed #cfe2d5; border-radius: 10px; background: #f8fcf9; }
.diagnosis-version-diff-panel-head { display: flex; align-items: center; justify-content: space-between; gap: 8px; }
.diagnosis-version-diff-panel-head strong { color: #2f5d4a; font-size: 13px; }
.diagnosis-version-diff-pickers { display: grid; grid-template-columns: 1fr 1fr; gap: 8px; }
.diagnosis-version-diff-pickers label { display: flex; flex-direction: column; gap: 4px; color: #4e7966; font-size: 11px; }
.diagnosis-version-diff-pickers select { min-height: 30px; padding: 0 8px; border: 1px solid #cfe2d5; border-radius: 8px; color: #314e41; background: #fff; font: inherit; font-size: 12px; }
.diagnosis-version-diff-fields { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: 6px; max-height: 280px; overflow: auto; }
.diagnosis-version-diff-field { display: grid; grid-template-columns: 88px 1fr 1fr; gap: 6px; padding: 6px 8px; border: 1px solid #e4efe8; border-radius: 8px; background: #fff; font-size: 11px; color: #5b6f66; }
.diagnosis-version-diff-field.is-changed { border-color: #e6c97a; background: #fff9eb; color: #4a4030; }
.diagnosis-version-diff-label { font-weight: 600; color: #3f6a56; }
.diagnosis-version-diff-side { display: flex; gap: 6px; min-width: 0; white-space: pre-wrap; word-break: break-word; }
.diagnosis-version-diff-side-tag { flex: 0 0 auto; color: #7a9186; font-weight: 600; }
@media (max-width: 840px) {
  .diagnosis-version-diff-pickers { grid-template-columns: 1fr; }
  .diagnosis-version-diff-field { grid-template-columns: 1fr; }
}
</style>
