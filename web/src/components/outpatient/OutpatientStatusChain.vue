<script setup>
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { isAuthenticated, session } from '../../auth/session.js'
import { fetchPatient } from '../../api/patientApi.js'
import { listRegisters } from '../../api/registerApi.js'
import { listOrders } from '../../api/orderApi.js'
import { listCharges } from '../../api/chargeApi.js'
import { listDispenses } from '../../api/dispenseApi.js'
import { listDiagnoses } from '../../api/diagnosisApi.js'
import { outpatientContext, outpatientSelection, setPatient, setRegister, setDiagnosis, setOrders, setCharges, setDispense } from '../../workbench/outpatientContext.js'
import { buildStatusChainSummary, classifyDomainError, contextFromStatusChainPayloads, STATUS_CHAIN_STATE_LABELS } from '../../workbench/statusChainExplain.js'

const DOMAIN_FIELDS = { patient: ['patient'], registers: ['register'], orders: ['orders'], charges: ['charges'], diagnoses: ['diagnosis', 'diagnoses'], dispenses: ['dispense'] }
const DOMAIN_LABELS = { registers: '挂号', orders: '开单', charges: '收费', diagnoses: '诊断', dispenses: '发药' }
const loading = ref(false)
const refreshError = ref('')
const snapshot = ref(null)
const registerId = computed(() => outpatientContext.register?.registerId || '')
const canRefresh = computed(() => Boolean(registerId.value && outpatientContext.patient?.patientId && isAuthenticated.value && !outpatientSelection.pending))
let requestVersion = 0
let disposed = false

function fingerprint(domain) {
  return JSON.stringify(DOMAIN_FIELDS[domain].map(field => outpatientContext[field]))
}

const summary = computed(() => {
  const context = { ...outpatientContext }
  const domainAccess = { ...snapshot.value?.access }
  if (snapshot.value) {
    for (const [domain, fields] of Object.entries(DOMAIN_FIELDS)) {
      if (fingerprint(domain) === snapshot.value.baseline[domain]) {
        for (const field of fields) context[field] = snapshot.value.context[field]
      } else delete domainAccess[domain]
    }
  }
  return buildStatusChainSummary(context, { domainAccess, refreshedAt: snapshot.value?.refreshedAt })
})
const accessBanners = computed(() => Object.entries(summary.value.domainAccess).filter(([, access]) => access.ok === false))

async function readRegister(token, anchor) {
  let offset = 0
  const seen = new Set()
  while (!seen.has(offset)) {
    seen.add(offset)
    const result = await listRegisters(token, { patientId: anchor.patientId, limit: 100, offset })
    const found = result.items.find(row => row.registerId === anchor.registerId && row.patientId === anchor.patientId)
    if (found) return { items: [found] }
    if (!result.hasMore) break
    offset = result.nextOffset
  }
  throw new Error('未找到当前挂号，请重新选择有效就诊')
}

async function refresh() {
  if (!canRefresh.value || loading.value) return
  const version = ++requestVersion
  const token = session.accessToken
  const anchor = { registerId: registerId.value, patientId: outpatientContext.patient.patientId }
  const before = Object.fromEntries(Object.keys(DOMAIN_FIELDS).map(domain => [domain, fingerprint(domain)]))
  const isCurrent = () => !disposed && version === requestVersion && token === session.accessToken && canRefresh.value
    && registerId.value === anchor.registerId && outpatientContext.patient?.patientId === anchor.patientId
  loading.value = true
  refreshError.value = ''
  try {
    const loaders = {
      patient: () => fetchPatient(token, anchor.patientId),
      registers: () => readRegister(token, anchor),
      orders: () => listOrders(token, { registerId: anchor.registerId, patientId: anchor.patientId }),
      charges: () => listCharges(token, { registerId: anchor.registerId, todayOnly: false }),
      dispenses: () => listDispenses(token, { patient: anchor.patientId, today: false }),
      diagnoses: () => listDiagnoses(token, anchor.registerId),
    }
    const domains = Object.keys(loaders)
    const results = await Promise.allSettled(domains.map(domain => loaders[domain]()))
    if (!isCurrent()) return
    if (domains.some(domain => before[domain] !== fingerprint(domain))) {
      refreshError.value = '读取期间业务记录已更新，为避免覆盖新数据，请重新刷新'
      return
    }
    const payloads = {}
    const access = {}
    results.forEach((result, index) => {
      const domain = domains[index]
      if (result.status === 'fulfilled') payloads[domain] = result.value
      else if (domain === 'patient') refreshError.value = '患者资料：' + classifyDomainError(result.reason).message
      else access[domain] = { ok: false, ...classifyDomainError(result.reason) }
    })
    const context = contextFromStatusChainPayloads(anchor, {
      ...payloads,
      patient: payloads.patient || outpatientContext.patient,
      registers: payloads.registers || { items: [{ ...anchor }] },
    })
    for (const domain of domains.filter(domain => domain !== 'patient' && !access[domain])) {
      const fields = DOMAIN_FIELDS[domain]
      const value = context[fields.at(-1)]
      access[domain] = { ok: true, empty: Array.isArray(value) ? value.length === 0 : !value }
    }
    if (payloads.patient) setPatient(context.patient)
    if (access.registers.ok) setRegister(context.register)
    if (access.orders.ok) setOrders(context.orders)
    if (access.charges.ok) setCharges(context.charges)
    if (access.diagnoses.ok) setDiagnosis(context.diagnosis, context.diagnoses)
    if (access.dispenses.ok && context.dispense) setDispense(context.dispense)
    snapshot.value = {
      context, access, refreshedAt: new Date().toISOString(),
      baseline: Object.fromEntries(domains.map(domain => [domain, fingerprint(domain)])),
    }
    if (Object.values(access).some(value => !value.ok)) refreshError.value ||= '部分域读取失败，状态不完整；请查看各域提示后重试'
  } catch {
    if (isCurrent()) refreshError.value = '无法核对当前患者/挂号的服务器数据，请重新刷新或选择有效就诊'
  } finally {
    if (version === requestVersion) loading.value = false
  }
}

watch(() => [registerId.value, outpatientContext.patient?.patientId, session.accessToken, isAuthenticated.value, outpatientSelection.pending], () => {
  requestVersion++
  loading.value = false
  snapshot.value = null
  refreshError.value = ''
  if (canRefresh.value) void refresh()
}, { immediate: true })
onBeforeUnmount(() => { disposed = true; requestVersion++ })
</script>

<template>
  <section class="outpatient-status-chain" data-testid="outpatient-status-chain" aria-label="流程状态" :data-kind="summary.kind" :aria-busy="loading">
    <div class="status-headline">
      <strong :data-tone="summary.tone">{{ summary.label }}</strong>
      <span>下一步：{{ summary.next }}</span>
      <button type="button" :disabled="!canRefresh || loading" @click="refresh">{{ loading ? '读取中…' : refreshError ? '重试刷新状态' : '刷新状态' }}</button>
    </div>
    <p class="status-explain" aria-live="polite">{{ summary.explain }}</p>
    <ol class="status-steps">
      <li v-for="step in summary.steps" :key="step.key" :data-state="step.state" :aria-current="step.current ? 'step' : undefined">
        <span>{{ step.complete ? step.completeLabel : step.label }} · {{ STATUS_CHAIN_STATE_LABELS[step.state] }}</span>
        <small>{{ step.reason }}</small>
      </li>
    </ol>
    <p v-for="[domain, access] in accessBanners" :key="domain" class="status-warning" role="status" :data-domain="domain">
      {{ DOMAIN_LABELS[domain] }}：{{ access.forbidden ? '该域无权限' : '该域读取失败' }}，{{ access.message }}
    </p>
    <p v-if="refreshError" class="status-warning" role="alert">{{ refreshError }}</p>
    <small v-if="summary.refreshedAt">服务器回读时间：{{ summary.refreshedAt }}</small>
    <small v-else-if="!canRefresh && summary.registerId">登录有效且就诊选择稳定后可刷新</small>
  </section>
</template>

<style scoped>
.outpatient-status-chain { display:grid; gap:7px; min-width:0; padding:10px 12px; border:1px solid #d8e7ec; border-radius:7px; background:#f7fbfc; color:#334b57; }
.status-headline { display:flex; align-items:center; flex-wrap:wrap; gap:8px 14px; font-size:12px; }
.status-headline button { margin-left:auto; padding:4px 10px; border:1px solid #aac6ce; border-radius:4px; background:#fff; color:#23566b; cursor:pointer; }
.status-headline button:disabled { opacity:.65; cursor:default; }
.status-headline button:focus-visible { outline:2px solid #23566b; outline-offset:2px; }
.status-explain, .status-warning { margin:0; font-size:12px; line-height:1.5; overflow-wrap:anywhere; }
.status-warning { color:#843e17; }
.status-steps { display:flex; flex-wrap:wrap; gap:6px 14px; margin:0; padding:0; list-style:none; }
.status-steps li { display:grid; gap:2px; flex:1 1 115px; font-size:12px; }
.status-steps small, section > small { font-size:11px; line-height:1.4; color:#506975; overflow-wrap:anywhere; }
[data-state="done"] > span { color:#23634f; }
[data-state="current"] > span { color:#175a76; font-weight:700; }
[data-state="blocked"] > span, [data-state="void"] > span { color:#873c25; font-weight:700; }
</style>
