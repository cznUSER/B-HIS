<script setup>
// 2026-10-15 P2-DIAGNOSIS-UI 诊断工作区：挂号选择、内容保存、提交和作废。
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ApiError } from '../api/errors.js'
import { useMock } from '../api/config.js'
import { commitDiagnose, abateDiagnose, listDiagnoses, saveDiagnose } from '../api/diagnosisApi.js'
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

const registers = ref([])
const selectedRegisterId = ref('')
const diagnoses = ref([])
const diagnoseId = ref('')
const diagnoseStatus = ref('NEW')
const content = ref('')
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

const hasAccess = computed(() => canEnter(findModule('diagnosis'), session.user))
const selectedRegister = computed(() => registers.value.find((item) => item.registerId === selectedRegisterId.value) || null)
const busy = computed(() => saving.value || committing.value || abating.value)
const canSave = computed(() => Boolean(selectedRegister.value) && !diagnoseId.value && !busy.value)
const canCommit = computed(() => Boolean(diagnoseId.value) && diagnoseCanCommit(diagnoseStatus.value) && !busy.value)
const canAbate = computed(() => Boolean(diagnoseId.value) && diagnoseCanAbate(diagnoseStatus.value) && !busy.value)
const editorLocked = computed(() => Boolean(diagnoseId.value) && !['NEW', 'SAVE'].includes(diagnoseStatus.value))

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
    return
  }
  diagnoseId.value = row.diagnoseId
  diagnoseStatus.value = row.status
  content.value = row.content || ''
  setDiagnosis(row, diagnoses.value)
}

function resetDraft() {
  clearNotice()
  applyDiagnosis(null)
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
  saving.value = true
  try {
    const saved = await saveDiagnose(session.accessToken, selectedRegisterId.value, { content: content.value })
    applyDiagnosis(saved)
    diagnoses.value = [...diagnoses.value, saved]
    setDiagnosis(saved, diagnoses.value)
    recordAction('诊断已暂存', saved.diagnoseId || '')
    infoMessage.value = '诊断已暂存，可继续提交或作废。'
  } catch (error) {
    errorMessage.value = error instanceof ApiError ? error.message : '暂存诊断失败，请稍后重试'
  } finally {
    saving.value = false
  }
}

async function onCommit() {
  if (!canCommit.value) return
  clearNotice()
  committing.value = true
  try {
    const committed = await commitDiagnose(session.accessToken, diagnoseId.value)
    applyDiagnosis(committed)
    diagnoses.value = diagnoses.value.map((row) => row.diagnoseId === committed.diagnoseId ? committed : row)
    setDiagnosis(committed, diagnoses.value)
    recordAction('诊断已提交', committed.diagnoseId || '')
    infoMessage.value = '诊断已提交。'
  } catch (error) {
    errorMessage.value = error instanceof ApiError ? error.message : '提交诊断失败，请稍后重试'
  } finally {
    committing.value = false
  }
}

async function onAbate() {
  if (!canAbate.value) return
  clearNotice()
  abating.value = true
  try {
    const abated = await abateDiagnose(session.accessToken, diagnoseId.value)
    applyDiagnosis(abated)
    diagnoses.value = diagnoses.value.map((row) => row.diagnoseId === abated.diagnoseId ? abated : row)
    setDiagnosis(abated, diagnoses.value)
    recordAction('诊断已作废', abated.diagnoseId || '')
    infoMessage.value = '诊断已作废。'
  } catch (error) {
    errorMessage.value = error instanceof ApiError ? error.message : '作废诊断失败，请稍后重试'
  } finally {
    abating.value = false
  }
}

watch(() => queryText(route.query.registerId), (id) => {
  if (id && id !== selectedRegisterId.value) selectedRegisterId.value = id
  if (id) void loadDiagnoses(id)
}, { immediate: true })

onMounted(loadRegisters)
</script>

<template>
  <section class="diagnosis-page" aria-labelledby="diagnosis-title">
    <p class="crumb"><RouterLink :to="{ name: 'home' }">首页</RouterLink><span aria-hidden="true"> / </span><RouterLink :to="{ name: 'workbench' }">门诊业务</RouterLink><span aria-hidden="true"> / </span><span>诊断</span></p>
    <BusinessPageHeader title-id="diagnosis-title" title="诊断" eyebrow="门诊业务 · 可选录入" icon="diagnosis" description="按挂号记录诊断内容，可先暂存，再提交或作废。临床规则当前保持关闭。">
      <template #actions><span class="diagnosis-scope">当前账号：{{ hasAccess ? '门诊医生' : '只读' }}</span></template>
    </BusinessPageHeader>

    <BusinessEmptyState v-if="!hasAccess" title="无权限" description="当前账号不能录入诊断，请使用门诊医生或系统账号。" icon="diagnosis" eyebrow="访问权限" />
    <BusinessEmptyState v-else-if="!loadingRegisters && !registers.length" title="暂无可用挂号" description="请先完成患者建档和挂号，再进入诊断工作区。" icon="diagnosis" eyebrow="选择挂号" />
    <div v-else class="diagnosis-shell">
      <section class="diagnosis-selector" aria-label="挂号选择">
        <label class="diagnosis-register-field"><span>当前挂号</span><select v-model="selectedRegisterId" :disabled="loadingRegisters || busy" @change="chooseRegister"><option value="">请选择挂号</option><option v-for="row in registers" :key="row.registerId" :value="row.registerId">{{ registerName(row) }} · {{ row.registerNo || row.registerId.slice(0, 8) }}</option></select></label>
        <p v-if="selectedRegister" class="diagnosis-register-meta">{{ registerMeta(selectedRegister) }}</p>
        <button class="button button-ghost button-compact" type="button" :disabled="busy || !selectedRegister" @click="resetDraft">新建诊断</button>
      </section>

      <p v-if="registersError" class="diagnosis-alert diagnosis-alert-error" role="alert">{{ registersError }}</p>
      <p v-if="diagnosesError" class="diagnosis-alert diagnosis-alert-error" role="alert">{{ diagnosesError }}</p>
      <p v-if="errorMessage" class="diagnosis-alert diagnosis-alert-error" role="alert">{{ errorMessage }}</p>
      <p v-if="infoMessage" class="diagnosis-alert diagnosis-alert-success" role="status">{{ infoMessage }}</p>

      <div class="diagnosis-workspace">
        <aside class="diagnosis-history" aria-label="诊断记录">
          <div class="diagnosis-section-head"><div><p class="diagnosis-kicker">RECORDS</p><h2>本次诊断</h2></div><span>{{ diagnoses.length }} 条</span></div>
          <p v-if="loadingDiagnoses" class="diagnosis-muted">读取中…</p>
          <p v-else-if="!diagnoses.length" class="diagnosis-muted">尚无已保存诊断。</p>
          <button v-for="row in diagnoses" :key="row.diagnoseId" class="diagnosis-record" :class="{ 'is-active': row.diagnoseId === diagnoseId }" type="button" @click="chooseDiagnosis(row)">
            <span class="diagnosis-record-status">{{ diagnoseStatusLabel(row.status) }}</span><span class="diagnosis-record-text">{{ row.content }}</span>
          </button>
        </aside>

        <main class="diagnosis-editor" aria-label="诊断编辑">
          <div class="diagnosis-section-head"><div><p class="diagnosis-kicker">EDITOR</p><h2>{{ diagnoseId ? '诊断详情' : '录入诊断' }}</h2></div><span class="diagnosis-status" :class="`status-${diagnoseStatus.toLowerCase()}`">{{ diagnoseStatusLabel(diagnoseStatus) }}</span></div>
          <label class="diagnosis-content-field"><span>诊断内容</span><textarea v-model="content" :readonly="editorLocked || busy" maxlength="2000" rows="10" placeholder="填写本次就诊诊断内容" /></label>
          <div class="diagnosis-editor-foot"><span>{{ content.length }}/2000</span><span v-if="editorLocked">已提交或已作废的诊断不能修改</span></div>
          <div class="diagnosis-actions"><button v-if="!diagnoseId" class="button button-primary" type="button" :disabled="!canSave" @click="onSave">{{ saving ? '保存中…' : '保存诊断' }}</button><button v-if="canCommit" class="button button-primary" type="button" :disabled="busy" @click="onCommit">{{ committing ? '提交中…' : '提交' }}</button><button v-if="canAbate" class="button button-ghost danger" type="button" :disabled="busy" @click="onAbate">{{ abating ? '作废中…' : '作废' }}</button></div>
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
.diagnosis-kicker { margin-bottom: 5px; color: #84a18f; font-size: 9px; letter-spacing: .12em; }
.diagnosis-muted { padding: 24px 5px; color: #99a89f; font-size: 12px; line-height: 1.7; }
.diagnosis-record { display: flex; flex-direction: column; align-items: flex-start; width: 100%; gap: 7px; margin-bottom: 8px; padding: 12px; text-align: left; border: 1px solid #e1ebe4; border-radius: 10px; color: #526c5d; background: #fbfefc; cursor: pointer; }
.diagnosis-record:hover, .diagnosis-record.is-active { border-color: #9ed1b5; background: #f0f9f2; }
.diagnosis-record-status { color: #4f9879; font-size: 10px; }
.diagnosis-record-text { display: -webkit-box; overflow: hidden; color: #3f594c; font-size: 12px; line-height: 1.6; -webkit-box-orient: vertical; -webkit-line-clamp: 3; }
.diagnosis-status { min-height: 24px; padding: 5px 9px; border-radius: 7px; background: #edf6ef; color: #4d8b6d !important; }
.status-commit { background: #e8f2fa; color: #4b789c !important; }.status-abate { background: #f8eeee; color: #a26a6a !important; }
.diagnosis-content-field { display: flex; flex-direction: column; gap: 8px; color: #667f72; font-size: 11px; }
.diagnosis-content-field textarea { width: 100%; min-height: 230px; resize: vertical; padding: 13px; border: 1px solid #cfe2d5; border-radius: 10px; color: #314e41; background: #fcfefd; font: inherit; line-height: 1.8; }
.diagnosis-content-field textarea:focus { outline: 2px solid #a9d8bc; outline-offset: 1px; }
.diagnosis-content-field textarea[readonly] { color: #768a7e; background: #f5f8f5; }
.diagnosis-editor-foot { display: flex; justify-content: space-between; gap: 12px; margin-top: 7px; color: #95a69b; font-size: 10px; }
.diagnosis-actions { display: flex; gap: 10px; flex-wrap: wrap; margin-top: 22px; }
.diagnosis-actions .danger { color: #a25f5f; border-color: #eed8d5; background: #fff8f7; }
@media (max-width: 840px) { .diagnosis-workspace { grid-template-columns: 1fr; } .diagnosis-history { order: 2; } .diagnosis-editor { order: 1; } }
@media (max-width: 600px) { .diagnosis-selector { align-items: stretch; flex-direction: column; } .diagnosis-register-field { min-width: 0; } .diagnosis-register-meta { align-self: flex-start; } }
</style>
