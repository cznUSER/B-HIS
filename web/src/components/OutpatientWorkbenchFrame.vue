<script setup>
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { session } from '../auth/session.js'
import { decorateModules, findModule } from '../workbench/catalog.js'
import { outpatientTarget } from '../workbench/context.js'
import { outpatientContext } from '../workbench/outpatientContext.js'
import './outpatient-workbench.css'

const props = defineProps({
  moduleKey: { type: String, required: true },
  title: { type: String, default: '' },
  description: { type: String, default: '' },
  context: { type: Object, default: () => ({}) },
  status: { type: String, default: '' },
  statusTone: { type: String, default: '' },
})

const route = useRoute()
const moduleInfo = computed(() => findModule(props.moduleKey))
const modules = computed(() => decorateModules(session.user))
const nextModule = computed(() => {
  const currentIndex = modules.value.findIndex((item) => item.key === props.moduleKey)
  return modules.value.slice(currentIndex + 1).find((item) => item.allowed && item.status === 'available') || null
})
const nextTarget = computed(() => nextModule.value ? outpatientTarget(nextModule.value.key, route.query) : { name: 'workbench' })
const contextValues = computed(() => {
  const source = props.context || {}
  return [
    ['患者', source.patientId || outpatientContext.patient?.patientId, true],
    ['挂号', source.registerId || outpatientContext.register?.registerId, false],
    ['医嘱', source.orderId || outpatientContext.orders.at(-1)?.orderId, false],
    ['收费', source.chargeId || source.charge_id || outpatientContext.charges.at(-1)?.chargeId, false],
  ].filter(([, value]) => value)
})
const patientSummary = computed(() => ({
  ...(outpatientContext.patient || {}),
  ...(props.context || {}),
}))
const statusSteps = computed(() => {
  const source = props.context || {}
  const completed = {
    patient: Boolean(outpatientContext.patient?.patientId || source.patientId),
    register: Boolean(outpatientContext.register?.registerId || source.registerId),
    diagnosis: Boolean(outpatientContext.diagnosis?.diagnoseId),
    order: Boolean(outpatientContext.orders.length || source.orderId),
    charge: Boolean(outpatientContext.charges.length || source.chargeId || source.charge_id),
    dispense: Boolean(outpatientContext.dispense?.dispenseId || source.dispenseId),
    refund: /^退/.test(outpatientContext.lastAction),
  }
  return [
    ['register', '挂号'],
    ['diagnosis', '诊断'],
    ['order', '开单'],
    ['charge', '收费'],
    ['dispense', '发药'],
    ['refund', '退药/退费'],
  ].map(([key, name]) => ({
    key,
    name,
    state: completed[key] ? 'done' : key === props.moduleKey ? 'current' : 'pending',
  }))
})
const recentHistory = computed(() => outpatientContext.history)
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
const statusText = computed(() => {
  if (props.status) return props.status
  if (!moduleInfo.value?.allowed) return '当前账号无此模块权限'
  if (props.moduleKey !== 'patient' && !contextValues.value.length) return '等待患者或就诊上下文'
  return '工作区已就绪'
})
const tone = computed(() => props.statusTone || (statusText.value.includes('无') ? 'blocked' : statusText.value.includes('等待') ? 'waiting' : 'ready'))
const displayValue = (value) => String(value).length > 18 ? `${String(value).slice(0, 8)}…${String(value).slice(-6)}` : value
</script>

<template>
  <section class="outpatient-frame" :data-module="moduleKey">
    <header class="outpatient-frame-head">
      <div class="outpatient-frame-title">
        <div class="outpatient-frame-kicker"><span class="outpatient-frame-index">{{ String((moduleInfo && moduleInfo.key ? modules.findIndex((item) => item.key === moduleKey) + 1 : 0)).padStart(2, '0') }}</span><span>门诊工作区</span><span class="outpatient-frame-divider">/</span><span>{{ moduleInfo?.name || title }}</span></div>
        <h1>{{ title || moduleInfo?.name || '门诊业务' }}</h1>
        <p v-if="description">{{ description }}</p>
      </div>
      <div class="outpatient-frame-actions">
        <RouterLink class="outpatient-frame-link" :to="{ name: 'workbench' }">流程总览</RouterLink>
        <RouterLink v-if="nextModule" class="outpatient-frame-next" :to="nextTarget">下一步：{{ nextModule.name }} <span aria-hidden="true">→</span></RouterLink>
      </div>
    </header>

    <div class="outpatient-context" data-testid="outpatient-context" aria-label="患者和就诊上下文">
      <div class="outpatient-context-main">
        <span class="outpatient-context-avatar" aria-hidden="true">{{ contextValues.length ? '患' : '—' }}</span>
        <div><span class="outpatient-context-label">当前患者</span><strong>{{ patientSummary.name || patientSummary.patientName || (patientSummary.patientId ? '已选择患者' : '未选择患者') }}</strong></div>
        <span class="outpatient-context-state" :class="`is-${tone}`">{{ patientSummary.patientId ? '已关联' : moduleKey === 'patient' ? '建档工作区' : '需前置' }}</span>
      </div>
      <div class="outpatient-context-values">
        <span v-for="([label, value]) in contextValues" :key="label" class="outpatient-context-value"><small>{{ label }}</small><strong :title="value">{{ displayValue(value) }}</strong></span>
        <span v-if="!contextValues.length" class="outpatient-context-empty">从患者查询开始</span>
      </div>
      <RouterLink v-if="moduleKey !== 'patient' && !patientSummary.patientId" class="outpatient-context-cta" :to="{ name: 'patient-search' }">去选择患者 <span aria-hidden="true">↗</span></RouterLink>
    </div>

    <div class="outpatient-status" data-testid="outpatient-status" :data-tone="tone" role="status">
      <span class="outpatient-status-dot" aria-hidden="true"></span><strong>{{ statusText }}</strong><span v-if="tone === 'waiting'">完成前置后，当前页面会自动保留工作区。</span><span v-else-if="tone === 'blocked'">请联系管理员或切换到有权限的岗位。</span><span v-else>业务数据以服务端实时结果为准。</span>
    </div>

    <div class="outpatient-main-grid" data-testid="outpatient-three-column">
      <aside class="outpatient-left-rail" data-testid="outpatient-left-rail" aria-label="患者与流程摘要">
        <section class="outpatient-rail-block outpatient-patient-summary">
          <div class="outpatient-rail-heading"><span>患者摘要</span><span class="outpatient-rail-count">{{ patientSummary.patientNo || patientSummary.patientId || '未选择' }}</span></div>
          <strong class="outpatient-patient-name">{{ patientSummary.name || patientSummary.patientName || '未选择患者' }}</strong>
          <dl class="outpatient-mini-dl">
            <div><dt>性别</dt><dd>{{ patientSummary.sex || '—' }}</dd></div>
            <div><dt>出生日期</dt><dd>{{ patientSummary.birthday || '—' }}</dd></div>
            <div><dt>费别</dt><dd>{{ patientSummary.fareIdentity || '—' }}</dd></div>
          </dl>
        </section>
        <section class="outpatient-rail-block outpatient-status-chain" data-testid="outpatient-status-chain" aria-label="门诊状态链">
          <div class="outpatient-rail-heading"><span>流程状态</span><span class="outpatient-rail-count">{{ statusSteps.filter((item) => item.state === 'done').length }}/{{ statusSteps.length }}</span></div>
          <ol>
            <li v-for="step in statusSteps" :key="step.key" :class="`is-${step.state}`">
              <span class="outpatient-step-marker" aria-hidden="true">{{ step.state === 'done' ? '✓' : step.state === 'current' ? '●' : '·' }}</span>
              <span>{{ step.name }}</span>
              <small>{{ step.state === 'done' ? '已完成' : step.state === 'current' ? '当前' : '待进入' }}</small>
            </li>
          </ol>
        </section>
      </aside>

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
