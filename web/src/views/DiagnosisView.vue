<script setup>
import { useErrorDialog } from '../components/feedback/useErrorDialog.js'
import { createSubmitFeedback } from '../components/feedback/useSubmitFeedback.js'
import SubmitStatusBanner from '../components/feedback/SubmitStatusBanner.vue'
// 2026-10-15 P2-DIAGNOSIS-UI 诊断工作区：挂号选择、内容保存、提交和作废。
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
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

const hasAccess = computed(() => canEnter(findModule('diagnosis'), session.user))
const selectedRegister = computed(() => registers.value.find((item) => item.registerId === selectedRegisterId.value) || null)
const busy = computed(() => saving.value || committing.value || abating.value || submitFeedback.state.status === 'submitting')
const canSave = computed(() => Boolean(selectedRegister.value) && !diagnoseId.value && !busy.value)
const canAmend = computed(() => Boolean(diagnoseId.value) && diagnoseStatus.value === 'COMMIT' && !busy.value)
const canCommit = computed(() => Boolean(diagnoseId.value) && diagnoseCanCommit(diagnoseStatus.value) && !busy.value)
const canAbate = computed(() => Boolean(diagnoseId.value) && diagnoseCanAbate(diagnoseStatus.value) && !busy.value)
const editorLocked = computed(() => Boolean(diagnoseId.value) && !['NEW', 'SAVE'].includes(diagnoseStatus.value))
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
}

function resetDraft() {
  clearNotice()
  saveAttempt = null
  commitAttempt = null
  abateAttempt = null
  submitFeedback.reset()
  applyDiagnosis(null)
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
    if (!preferred) resetDraft()
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
  resetDraft()
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
  clearNotice()
  const id = selectedRegisterId.value
  await router.replace({ name: 'outpatient-module', params: { module: 'diagnosis' }, query: id ? { registerId: id } : {} })
  await loadDiagnoses(id)
}

function chooseDiagnosis(row) {
  clearNotice()
  applyDiagnosis(row)
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
  if (id) void loadDiagnoses(id)
}, { immediate: true })

onMounted(loadRegisters)

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
        <button class="button button-ghost button-compact" type="button" :disabled="busy || !selectedRegister" @click="resetDraft">新建诊断</button>
      </section>

      <p v-if="registersError" class="diagnosis-alert diagnosis-alert-error" role="alert">{{ registersError }}</p>
      <p v-if="diagnosesError" class="diagnosis-alert diagnosis-alert-error" role="alert">{{ diagnosesError }}</p>
      <p v-if="errorMessage && submitFeedback.state.status !== 'error'" class="diagnosis-alert diagnosis-alert-error" role="alert">{{ errorMessage }}</p>
      <p v-if="infoMessage && submitFeedback.state.status !== 'success' && submitFeedback.state.status !== 'error'" class="diagnosis-alert diagnosis-alert-success" role="status">{{ infoMessage }}</p>
      <SubmitStatusBanner :feedback="submitFeedback.state" />

      <div class="diagnosis-workspace">
        <aside class="diagnosis-history" aria-label="诊断记录">
          <div class="diagnosis-section-head"><div><p class="diagnosis-kicker">RECORDS</p><h2>本次诊断</h2></div><span>{{ diagnoses.length }} 条</span></div>
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
            <label v-for="field in STRUCTURED_FIELDS" :key="field.key" class="diagnosis-content-field" :class="{ 'diagnosis-content-field-wide': ['presentHistory', 'examination', 'plan'].includes(field.key) }"><span>{{ field.label }}</span><textarea v-model="structured[field.key]" :readonly="editorLocked || busy" maxlength="320" rows="3" :placeholder="field.placeholder" /></label>
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
.diagnosis-content-field textarea { width: 100%; min-height: 76px; resize: vertical; padding: 10px 11px; border: 1px solid #cfe2d5; border-radius: 10px; color: #314e41; background: #fcfefd; font: inherit; line-height: 1.65; }
.diagnosis-content-field textarea:focus { outline: 2px solid #a9d8bc; outline-offset: 1px; }
.diagnosis-content-field textarea[readonly] { color: #768a7e; background: #f5f8f5; }
.diagnosis-editor-foot { display: flex; justify-content: space-between; gap: 12px; margin-top: 7px; color: #95a69b; font-size: 10px; }
.diagnosis-length-error { color: #a34f4f; }
.diagnosis-actions { display: flex; gap: 10px; flex-wrap: wrap; margin-top: 22px; }
.diagnosis-actions .danger { color: #a25f5f; border-color: #eed8d5; background: #fff8f7; }
@media (max-width: 840px) { .diagnosis-workspace { grid-template-columns: 1fr; } .diagnosis-history { order: 2; } .diagnosis-editor { order: 1; } .diagnosis-structured-grid { grid-template-columns: 1fr; } .diagnosis-content-field-wide { grid-column: auto; } }
@media (max-width: 600px) { .diagnosis-selector { align-items: stretch; flex-direction: column; } .diagnosis-register-field { min-width: 0; } .diagnosis-register-meta { align-self: flex-start; } .diagnosis-template-bar { align-items: stretch; flex-direction: column; } .diagnosis-template-bar label { flex-direction: column; align-items: stretch; } .diagnosis-reason-field input { min-width: 0; } }
</style>
