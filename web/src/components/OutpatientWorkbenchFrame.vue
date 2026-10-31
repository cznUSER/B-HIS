<script setup>
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { isAuthenticated, session } from '../auth/session.js'
import { fetchPatient } from '../api/patientApi.js'
import { listRegisters } from '../api/registerApi.js'
import { fetchDepartments, fetchDoctors } from '../master/api.js'
import { decorateModules } from '../workbench/catalog.js'
import WorkflowSteps from './WorkflowSteps.vue'
import {
  deriveWorkflowProgress,
  deriveWorkflowStatus,
  outpatientContext,
  patientDisplayMeta,
  patientDisplayName,
  patientFareDisplay,
  patientRiskDisplay,
  clearOutpatientContext,
  outpatientSelection,
  readOutpatientSelection,
  resolveOutpatientSelection,
  selectOutpatientContext,
  setOutpatientSwitchHandler,
} from '../workbench/outpatientContext.js'
import './outpatient-workbench.css'

const props = defineProps({
  moduleKey: { type: String, required: true },
})

const route = useRoute()
const router = useRouter()
const contextReady = ref(false)
const contextLoading = ref(false)
const contextError = ref('')
const serverDate = ref('')
const departments = ref([])
const doctors = ref([])
const masterLoading = ref(false)
const switchDialog = ref(null)
const switchMessage = ref(null)
const switchCancel = ref(null)
const slotRevision = ref(0)
let finishSwitch = null
let loadVersion = 0
let restoredSignature = ''
let disposed = false
let guardActive = false
let needsRemount = false

function finishConfirmation(accepted) {
  switchDialog.value?.close()
  const finish = finishSwitch
  finishSwitch = null
  switchMessage.value = null
  finish?.(accepted)
}

setOutpatientSwitchHandler(async (message) => {
  switchMessage.value = message
  const decision = new Promise((resolve) => { finishSwitch = resolve })
  await nextTick()
  if (disposed || !isAuthenticated.value) {
    finishConfirmation(false)
    return false
  }
  switchDialog.value.showModal()
  switchCancel.value?.focus()
  return decision
})

function signature(ids) {
  return (ids.patientId || '') + '|' + (ids.registerId || '')
}

function currentIds() {
  return {
    ...(outpatientContext.patient?.patientId ? { patientId: outpatientContext.patient.patientId } : {}),
    ...(outpatientContext.register?.registerId ? { registerId: outpatientContext.register.registerId } : {}),
  }
}

async function restoreContext(target = route) {
  if (!isAuthenticated.value) return false
  const version = ++loadVersion
  const token = session.accessToken
  contextLoading.value = true
  contextError.value = ''
  const isCurrent = () => !disposed && version === loadVersion && token === session.accessToken && isAuthenticated.value
  try {
    const ids = readOutpatientSelection(target.query)
    if (!ids.patientId && !ids.registerId) {
      contextReady.value = true
      return true
    }
    if (contextReady.value && signature(ids) === restoredSignature) return true
    const selected = await resolveOutpatientSelection(ids, {
      authenticated: isAuthenticated.value,
      loadPatient: (patientId) => fetchPatient(token, patientId),
      loadRegisters: (query) => listRegisters(token, query),
    })
    if (!isCurrent()) return false
    const accepted = await selectOutpatientContext(selected, { isCurrent })
    if (!accepted || !isCurrent()) return false
    restoredSignature = signature(currentIds())
    contextReady.value = true
    return true
  } catch {
    if (isCurrent()) contextError.value = '无法恢复患者/就诊上下文。请重试或重新选择患者；不会使用缓存的患者详情。'
    return false
  } finally {
    if (version === loadVersion) contextLoading.value = false
  }
}

const removeGuard = router.beforeEach(async (to) => {
  if (to.name !== 'outpatient-module' || !isAuthenticated.value) return true
  guardActive = true
  try {
    const ids = readOutpatientSelection(to.query)
    if (contextReady.value && signature(ids) === signature(currentIds())) return true
    return await restoreContext(to)
  } catch {
    contextError.value = '患者或就诊参数无效，请重新选择。'
    return false
  } finally {
    guardActive = false
  }
})

const removeAfterEach = router.afterEach((_to, _from, failure) => {
  if (!failure && needsRemount) {
    slotRevision.value++
    needsRemount = false
  }
})

watch(() => outpatientSelection.draftRevision, async () => {
  needsRemount = true
  if (guardActive || route.name !== 'outpatient-module') return
  const ids = currentIds()
  await router.replace({ path: route.path, query: { ...route.query, patientId: ids.patientId, registerId: ids.registerId } })
  if (!disposed && needsRemount) {
    slotRevision.value++
    needsRemount = false
  }
}, { flush: 'sync' })

watch(() => route.fullPath, async () => {
  if (route.name !== 'outpatient-module') return
  const accepted = await restoreContext()
  if (!accepted || disposed) return
  const ids = currentIds()
  if (ids.patientId && (route.query.patientId !== ids.patientId || (route.query.registerId || '') !== (ids.registerId || ''))) {
    await router.replace({ path: route.path, query: { ...route.query, patientId: ids.patientId, registerId: ids.registerId } })
  }
}, { immediate: true })

watch(() => session.accessToken, (token, previous) => {
  if (token === previous) return
  loadVersion++
  finishConfirmation(false)
  clearOutpatientContext()
  departments.value = []
  doctors.value = []
  contextReady.value = false
  restoredSignature = ''
}, { flush: 'sync' })

async function loadDisplayData() {
  if (!isAuthenticated.value) return
  const token = session.accessToken
  masterLoading.value = true
  const results = await Promise.allSettled([
    fetchDepartments(token, {}),
    fetchDoctors(token, {}),
    fetch('/health', { cache: 'no-store', credentials: 'omit' }).then((response) => {
      const date = new Date(response.headers.get('Date') || '')
      return Number.isFinite(date.getTime()) ? new Intl.DateTimeFormat('en-CA', { timeZone: 'Asia/Shanghai', year: 'numeric', month: '2-digit', day: '2-digit' }).format(date) : ''
    }),
  ])
  if (disposed || token !== session.accessToken || !isAuthenticated.value) return
  departments.value = results[0].status === 'fulfilled' ? results[0].value.items : []
  doctors.value = results[1].status === 'fulfilled' ? results[1].value.items : []
  serverDate.value = results[2].status === 'fulfilled' ? results[2].value : ''
  masterLoading.value = false
}

void loadDisplayData()
onBeforeUnmount(() => {
  disposed = true
  loadVersion++
  finishConfirmation(false)
  setOutpatientSwitchHandler(null)
  removeGuard()
  removeAfterEach()
})

const modules = computed(() => decorateModules(session.user || {}))
const nextModule = computed(() => {
  const currentIndex = modules.value.findIndex((item) => item.key === props.moduleKey)
  return modules.value.slice(currentIndex + 1).find((item) => item.allowed && item.status === 'available') || null
})
const recentHistory = computed(() => outpatientContext.history)
const currentPatient = computed(() => isAuthenticated.value ? outpatientContext.patient : null)
const currentPatientName = computed(() => patientDisplayName(currentPatient.value))
const currentPatientMeta = computed(() => patientDisplayMeta(currentPatient.value, serverDate.value))
const currentPatientId = computed(() => currentPatient.value?.patientId || currentPatient.value?.patientNo || '')
const currentPatientNo = computed(() => currentPatient.value?.patientNo || (currentPatient.value?.patientId ? '系统 Id：' + currentPatient.value.patientId : ''))
const currentFare = computed(() => patientFareDisplay(currentPatient.value))
const currentRisk = computed(() => patientRiskDisplay(currentPatient.value))
const currentRegister = computed(() => isAuthenticated.value ? outpatientContext.register : null)
const currentModuleName = computed(() => modules.value.find((item) => item.key === props.moduleKey)?.name || props.moduleKey)
const departmentLabel = computed(() => currentRegister.value?.departmentName || departments.value.find((item) => item.departmentId === currentRegister.value?.departmentId)?.deptName || (masterLoading.value ? '加载中' : '未知'))
const doctorLabel = computed(() => currentRegister.value?.doctorName || doctors.value.find((item) => item.doctorId === currentRegister.value?.doctorId)?.doctorName || (masterLoading.value ? '加载中' : '未知'))
const currentRegisterNo = computed(() => currentRegister.value?.registerNo || currentRegister.value?.registerId || '')
const workflowStatus = computed(() => deriveWorkflowStatus(outpatientContext))
const workflowProgress = computed(() => deriveWorkflowProgress(outpatientContext))
const patientSearchRoute = computed(() => ({
  name: 'patient-search',
  query: currentPatientId.value ? { patientId: currentPatientId.value } : {},
}))
const workbenchRoute = computed(() => ({ name: 'workbench' }))
const latestCharge = computed(() => outpatientContext.charges.at(-1) || null)
const chargeSummary = computed(() => {
  const charge = latestCharge.value
  if (!charge) return null
  const amount = charge.squareFee ?? charge.amountDue ?? charge.totalFee ?? charge.paidFee
  const refundable = charge.refundableFee ?? charge.refundableAmount
  return {
    amount: amount == null ? '' : `¥${Number(amount).toFixed(2)}`,
    status: charge.status || (charge.paid ? '已收费' : '待收费'),
    refundable: refundable == null ? '' : `可退 ¥${Number(refundable).toFixed(2)}`,
  }
})
const inventorySummary = computed(() => {
  const items = Array.isArray(outpatientContext.dispense?.items) ? outpatientContext.dispense.items : []
  if (!items.length) return null
  return {
    count: items.length,
    stock: items.reduce((sum, item) => sum + Number(item.stockQty ?? item.availableQty ?? 0), 0),
    returnable: items.reduce((sum, item) => sum + Number(item.returnableQty ?? item.refundableQty ?? 0), 0),
  }
})
</script>

<template>
  <section class="outpatient-frame" :data-module="moduleKey">
    <section class="outpatient-context-bar" data-testid="outpatient-context" aria-label="当前患者上下文">
      <div class="outpatient-context-identity">
        <span class="outpatient-context-kicker">当前患者 · 当前模块：{{ currentModuleName }}</span>
        <strong data-testid="outpatient-patient-name">{{ currentPatientName }}</strong>
        <span v-if="currentPatientNo" class="outpatient-context-id">{{ currentPatientNo }}</span>
        <span v-if="currentPatientMeta" class="outpatient-context-meta">{{ currentPatientMeta }}</span>
        <span v-if="currentRegisterNo" class="outpatient-context-visit">就诊 {{ currentRegisterNo }}</span>
        <span v-if="currentPatient" class="outpatient-context-meta" data-testid="outpatient-fare">{{ currentFare }}</span>
        <template v-if="currentRegister">
          <span class="outpatient-context-meta" data-testid="outpatient-department">科室：{{ departmentLabel }}</span>
          <span class="outpatient-context-meta" data-testid="outpatient-doctor">医生：{{ doctorLabel }}</span>
          <span class="outpatient-context-meta" data-testid="outpatient-visit-date">就诊日：{{ currentRegister.visitDate || '未记录' }}</span>
        </template>
        <details v-if="currentPatient && currentRisk.highRisk" :key="currentPatient.patientId" class="outpatient-context-risk" data-testid="outpatient-allergy" data-risk="high">
          <summary>过敏史：已记录 · 高风险（展开查看）</summary>
          <span>{{ currentRisk.label }}</span>
        </details>
        <span v-else-if="currentPatient" class="outpatient-context-risk" data-testid="outpatient-allergy" data-risk="unknown">{{ currentRisk.label }}</span>
        <span class="outpatient-context-meta">下一步：{{ workflowStatus.next }}</span>
      </div>
      <div class="outpatient-context-actions">
        <span class="outpatient-context-state" :data-tone="workflowStatus.tone" data-testid="outpatient-status">{{ workflowStatus.label }}</span>
        <RouterLink class="outpatient-context-link" :to="workbenchRoute">门诊队列</RouterLink>
        <RouterLink class="outpatient-context-link" :to="patientSearchRoute">切换患者</RouterLink>
      </div>
    </section>

    <p v-if="!currentPatient" class="outpatient-context-notice" data-testid="outpatient-preconditions">尚未选择患者：请先从患者查询或队列选择患者，再选择有效就诊，方可按权限进行诊断、开单、收费、发药或退费。</p>
    <p v-if="contextLoading" class="outpatient-context-notice" role="status">正在从服务器读取患者/就诊上下文……</p>
    <p v-if="contextError" class="outpatient-context-notice outpatient-context-error" role="alert">{{ contextError }} <button type="button" :disabled="contextLoading" @click="restoreContext()">重试</button> <RouterLink :to="patientSearchRoute">重新选择患者</RouterLink></p>

    <dialog ref="switchDialog" class="outpatient-context-dialog" aria-labelledby="outpatient-switch-title" aria-describedby="outpatient-switch-body" @cancel.prevent="finishConfirmation(false)">
      <template v-if="switchMessage">
        <h2 id="outpatient-switch-title">{{ switchMessage.title }}</h2>
        <p id="outpatient-switch-body">{{ switchMessage.body }}</p>
        <div class="outpatient-context-dialog-actions">
          <button ref="switchCancel" type="button" data-testid="outpatient-switch-cancel" @click="finishConfirmation(false)">{{ switchMessage.cancelLabel }}</button>
          <button type="button" data-testid="outpatient-switch-confirm" @click="finishConfirmation(true)">{{ switchMessage.confirmLabel }}</button>
        </div>
      </template>
    </dialog>

    <section class="outpatient-status-chain" data-testid="outpatient-status-chain" aria-label="流程状态">
      <ol class="outpatient-status-steps">
        <li v-for="step in workflowProgress" :key="step.key" :class="{ 'is-complete': step.complete, 'is-current': step.current }">
          <span class="outpatient-status-dot" aria-hidden="true">{{ step.complete ? '✓' : '·' }}</span>
          <span>{{ step.complete ? step.completeLabel : step.label }}</span>
        </li>
      </ol>
      <span class="outpatient-status-next">下一步：{{ workflowStatus.next }}</span>
    </section>

    <WorkflowSteps v-if="isAuthenticated" />

    <div class="outpatient-main-grid" data-testid="outpatient-two-column">
      <main class="outpatient-frame-content" :inert="outpatientSelection.pending || !isAuthenticated"><div v-if="contextReady && isAuthenticated" :key="slotRevision"><slot /></div></main>

      <aside class="outpatient-right-rail" data-testid="outpatient-right-rail" aria-label="业务明细">
        <section class="outpatient-rail-block">
          <div class="outpatient-rail-heading"><span>最近记录</span><span class="outpatient-rail-count">{{ recentHistory.length }}</span></div>
          <ul v-if="recentHistory.length" class="outpatient-history-list">
            <li v-for="item in recentHistory" :key="`${item.at}-${item.label}`"><strong>{{ item.label }}</strong><span>{{ item.detail || '已更新' }}</span></li>
          </ul>
          <p v-else class="outpatient-rail-empty">进入模块后，读取成功的记录会显示在这里。</p>
        </section>
        <section v-if="chargeSummary" class="outpatient-rail-block" data-testid="outpatient-fee-summary">
          <div class="outpatient-rail-heading"><span>费用摘要</span><span class="outpatient-rail-count">{{ chargeSummary.status }}</span></div>
          <strong class="outpatient-metric">{{ chargeSummary.amount || '金额待返回' }}</strong>
          <span v-if="chargeSummary.refundable" class="outpatient-rail-note">{{ chargeSummary.refundable }}</span>
        </section>
        <section v-if="inventorySummary" class="outpatient-rail-block" data-testid="outpatient-inventory-summary">
          <div class="outpatient-rail-heading"><span>库存与可退</span><span class="outpatient-rail-count">{{ inventorySummary.count }} 项</span></div>
          <dl class="outpatient-mini-dl">
            <div><dt>库存/可用</dt><dd>{{ inventorySummary.stock }}</dd></div>
            <div><dt>可退数量</dt><dd>{{ inventorySummary.returnable }}</dd></div>
          </dl>
        </section>
        <section class="outpatient-rail-next">
          <span>当前建议</span><strong>{{ nextModule ? `下一步：${nextModule.name}` : '按当前状态继续处理' }}</strong>
        </section>
      </aside>
    </div>
  </section>
</template>
