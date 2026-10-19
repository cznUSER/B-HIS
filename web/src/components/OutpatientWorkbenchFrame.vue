<script setup>
import { computed } from 'vue'
import { session } from '../auth/session.js'
import { decorateModules } from '../workbench/catalog.js'
import WorkflowSteps from './WorkflowSteps.vue'
import {
  deriveWorkflowProgress,
  deriveWorkflowStatus,
  outpatientContext,
  patientDisplayMeta,
  patientDisplayName,
} from '../workbench/outpatientContext.js'
import './outpatient-workbench.css'

const props = defineProps({
  moduleKey: { type: String, required: true },
})

const modules = computed(() => decorateModules(session.user))
const nextModule = computed(() => {
  const currentIndex = modules.value.findIndex((item) => item.key === props.moduleKey)
  return modules.value.slice(currentIndex + 1).find((item) => item.allowed && item.status === 'available') || null
})
const recentHistory = computed(() => outpatientContext.history)
const currentPatient = computed(() => outpatientContext.patient)
const currentPatientName = computed(() => patientDisplayName(currentPatient.value))
const currentPatientMeta = computed(() => patientDisplayMeta(currentPatient.value))
const currentPatientId = computed(() => currentPatient.value?.patientId || currentPatient.value?.patientNo || '')
const currentPatientNo = computed(() => currentPatient.value?.patientNo || currentPatient.value?.patientId || '')
const currentRegister = computed(() => outpatientContext.register)
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
        <span class="outpatient-context-kicker">当前患者</span>
        <strong data-testid="outpatient-patient-name">{{ currentPatientName }}</strong>
        <span v-if="currentPatientNo" class="outpatient-context-id">{{ currentPatientNo }}</span>
        <span v-if="currentPatientMeta" class="outpatient-context-meta">{{ currentPatientMeta }}</span>
        <span v-if="currentRegisterNo" class="outpatient-context-visit">就诊 {{ currentRegisterNo }}</span>
      </div>
      <div class="outpatient-context-actions">
        <span class="outpatient-context-state" :data-tone="workflowStatus.tone" data-testid="outpatient-status">{{ workflowStatus.label }}</span>
        <RouterLink class="outpatient-context-link" :to="workbenchRoute">门诊队列</RouterLink>
        <RouterLink class="outpatient-context-link" :to="patientSearchRoute">切换患者</RouterLink>
      </div>
    </section>

    <section class="outpatient-status-chain" data-testid="outpatient-status-chain" aria-label="流程状态">
      <ol class="outpatient-status-steps">
        <li v-for="step in workflowProgress" :key="step.key" :class="{ 'is-complete': step.complete, 'is-current': step.current }">
          <span class="outpatient-status-dot" aria-hidden="true">{{ step.complete ? '✓' : '·' }}</span>
          <span>{{ step.complete ? step.completeLabel : step.label }}</span>
        </li>
      </ol>
      <span class="outpatient-status-next">下一步：{{ workflowStatus.next }}</span>
    </section>

    <WorkflowSteps />

    <div class="outpatient-main-grid" data-testid="outpatient-two-column">
      <main class="outpatient-frame-content"><slot /></main>

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
