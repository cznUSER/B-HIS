<script setup>
import { createSubmitFeedback } from '../components/feedback/useSubmitFeedback.js'
import SubmitStatusBanner from '../components/feedback/SubmitStatusBanner.vue'
// 2026-10-12 P1-H05-DISPENSE-UI 门诊发药：待发、部分发、退药。
// 2026-10-12 VITE_USE_MOCK=false 时走 /dispenses；Mock 仍可演示。
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { ApiError } from '../api/errors.js'
import { useMock } from '../api/config.js'
import {
  confirmDispense,
  fetchDispense,
  listDispenses,
  listTodayPending,
  resetDispenseDemo,
  returnDispense,
  searchPending,
} from '../api/dispenseApi.js'
import {
  PHARMACY,
  formatOperateAt,
  formatQty,
  lineStatusLabel,
  stockHint,
  stockIsShort,
} from '../api/dispenseContract.js'
import { DEMO_CASES } from '../api/mockDispenses.js'
import { session } from '../auth/session.js'
import BusinessPageHeader from '../components/BusinessPageHeader.vue'
import BusinessEmptyState from '../components/BusinessEmptyState.vue'
import { recordAction, setDispense, setPatient } from '../workbench/outpatientContext.js'

const props = defineProps({ returnOnly: { type: Boolean, default: false } })

const route = useRoute()
const mockMode = useMock()
const viewMode = ref(props.returnOnly || route.query.view === 'issued' ? 'issued' : 'pending')
const chargeContext = ref('')
const chargeTo = computed(() => ({ name: 'outpatient-module', params: { module: 'charge' }, query: { view: 'records', ...(chargeContext.value ? { charge_id: chargeContext.value } : {}) } }))

const searchInput = ref(null)
const query = ref('')
const searched = ref(false)
const emptyKind = ref('prompt')
const rows = ref([])
const loading = ref(false)
const dispensing = ref(false)
const errorMessage = ref('')
const errorCode = ref('')
const infoMessage = ref('')
const loadMode = ref('')
const refreshWarning = ref('')
const pendingNeedsRefresh = ref(false)

const issued = ref([])
const issuedPatient = ref('')
const issuedNo = ref('')
const issuedToday = ref(true)
const issuedLoading = ref(false)
const detailId = ref('')
const issuedError = ref('')

const drawerOpen = ref(false)
const returnSlip = ref(null)
const returnLines = ref([])
const returnReason = ref('')
const returning = ref(false)
const returnResult = ref(null)
const returnError = ref('')
const dispenseWrite = reactive({ fingerprint: '', key: '' })
const returnWrite = reactive({ fingerprint: '', key: '' })
const submitFeedback = createSubmitFeedback()

const submitting = computed(() => submitFeedback.state.status === 'submitting')
const busy = computed(() => loading.value || dispensing.value || returning.value || submitting.value)
const searchPlaceholder = computed(() => (
  mockMode
    ? '例如王全发、GH-FULL-001、SQ-PART-001、RX-SHORT-001'
    : '患者姓名、患者号、挂号号、收费单号或处方号'
))
const operatorName = computed(() => {
  const name = session.user && session.user.displayName
  return typeof name === 'string' && name.trim() ? name.trim() : '当前用户'
})
const singlePatient = computed(() => {
  const ids = new Set(rows.value.map((row) => row.patientId))
  return ids.size === 1 ? rows.value[0] : null
})
const detail = computed(() => issued.value.find((item) => item.dispenseId === detailId.value) || null)
const selectedRows = computed(() => rows.value.filter((row) => row.selected))
const selectedPatientCount = computed(() => new Set(selectedRows.value.map((row) => row.patientId)).size)
const shortageCount = computed(() => rows.value.filter((row) => rowShort(row)).length)

function clearBanner() {
  errorMessage.value = ''
  errorCode.value = ''
  infoMessage.value = ''
}

function toRow(item, singlePatient) {
  const canFill = item.availableQty > 0 && item.stockQty >= item.availableQty
  return {
    ...item,
    selected: item.availableQty > 0 && (singlePatient || canFill),
    dispenseQty: item.availableQty,
  }
}

function applyPending(result, mode) {
  const onePatient = result.items.length > 0 && new Set(result.items.map((item) => item.patientId)).size === 1
  rows.value = result.items.map((item) => toRow(item, onePatient))
  emptyKind.value = result.items.length > 0 ? '' : (result.empty || 'none')
  loadMode.value = mode
  searched.value = true
  pendingNeedsRefresh.value = false
  const first = result.items[0]
  if (first) setPatient({ patientId: first.patientId, name: first.patientName, patientNo: first.patientNo })
  setDispense({ chargeId: first?.chargeId, items: result.items })
}

function showError(error) {
  rows.value = []
  emptyKind.value = 'rejected'
  searched.value = true
  if (error instanceof ApiError) {
    errorMessage.value = error.message
    errorCode.value = error.code
    return
  }
  errorMessage.value = '发药失败，请稍后重试'
  errorCode.value = ''
}

let pendingRequest = 0
async function readPending(mode, keyword = query.value) {
  const request = ++pendingRequest
  const context = chargeContext.value
  const isCurrent = () => request === pendingRequest && context === chargeContext.value
  loading.value = true
  loadMode.value = mode
  try {
    const result = mode === 'today'
      ? await listTodayPending(session.accessToken)
      : await searchPending(session.accessToken, keyword)
    if (!isCurrent()) return false
    applyPending(context ? { ...result, items: result.items.filter((item) => item.chargeId === context) } : result, mode)
    return true
  } catch (error) {
    if (isCurrent()) throw error
    return false
  } finally {
    if (request === pendingRequest) loading.value = false
  }
}

async function reloadPending() {
  return loadMode.value ? readPending(loadMode.value) : true
}

let issuedRequest = 0
async function loadIssued(options = {}) {
  const request = ++issuedRequest
  const context = chargeContext.value
  issuedLoading.value = true
  issuedError.value = ''
  try {
    const listed = await listDispenses(session.accessToken, {
      today: issuedToday.value,
      patient: issuedPatient.value,
      dispenseNo: issuedNo.value,
    })
    if (request !== issuedRequest) return false
    issued.value = context ? listed.items.filter((item) => item.chargeId === context) : listed.items
    if (issued.value[0]) {
      setPatient({ patientId: issued.value[0].patientId, name: issued.value[0].patientName, patientNo: issued.value[0].patientNo })
      setDispense(issued.value[0])
    }
    if (detailId.value && !issued.value.some((item) => item.dispenseId === detailId.value)) detailId.value = ''
    return true
  } catch (error) {
    if (request !== issuedRequest) return false
    issuedError.value = error instanceof ApiError ? error.message : '发药记录读取失败'
    if (!options.afterWrite && !errorMessage.value) {
      errorMessage.value = error instanceof ApiError ? error.message : '发药记录读取失败'
      errorCode.value = error instanceof ApiError ? error.code : ''
    }
    return false
  } finally {
    if (request === issuedRequest) issuedLoading.value = false
  }
}

async function onSearch() {
  if (busy.value) return
  clearBanner()
  try {
    await readPending('search')
  } catch (error) {
    showError(error)
  }
}

async function onToday() {
  if (busy.value) return
  clearBanner()
  try {
    await readPending('today')
  } catch (error) {
    showError(error)
  }
}

async function onRefreshStock() {
  if (!searched.value || busy.value) return
  clearBanner()
  try {
    if (await reloadPending()) {
      refreshWarning.value = ''
      infoMessage.value = '已刷新库存'
    }
  } catch (error) {
    showError(error)
  }
}

// 2026-10-14: refresh failures never change a successful dispense/return into a failure.
async function refreshAfterWrite() {
  const context = chargeContext.value
  const routeRevision = routeRequest
  let failed = false
  if (searched.value) {
    try { await reloadPending() } catch { failed = true; if (context === chargeContext.value && routeRevision === routeRequest) pendingNeedsRefresh.value = true }
  }
  if (context !== chargeContext.value || routeRevision !== routeRequest) return
  if (!(await loadIssued({ afterWrite: true }))) failed = true
  if (context !== chargeContext.value || routeRevision !== routeRequest) return
  refreshWarning.value = failed ? '操作已成功，页面数据刷新失败。请重试刷新，无需再次提交。' : ''
}

function payloadFrom(list) {
  return list
    .filter((row) => row.selected !== false && Number(row.dispenseQty) > 0)
    .map((row) => ({
      orderId: row.orderId,
      chargeId: row.chargeId,
      orderItemId: row.orderItemId,
      chargeItemId: row.chargeItemId,
      dispenseQty: Number(row.dispenseQty),
      selected: true,
    }))
}

async function submit(list) {
  if (dispensing.value || submitting.value || pendingNeedsRefresh.value) return
  const writeContext = chargeContext.value
  const writeRoute = routeRequest
  clearBanner()
  const items = payloadFrom(list)
  const fingerprint = JSON.stringify(items)
  if (dispenseWrite.fingerprint !== fingerprint) {
    dispenseWrite.fingerprint = fingerprint
    dispenseWrite.key = crypto.randomUUID()
  }
  dispensing.value = true
  submitFeedback.beginAttempt('dispense')
  try {
    const result = await confirmDispense(session.accessToken, {
      items,
      idempotencyKey: dispenseWrite.key,
    })
    const slips = Array.isArray(result.dispenses) && result.dispenses.length > 0 ? result.dispenses : [result]
    const nos = slips.map((item) => item.dispenseNo).join('、')
    const tip = `已发药 ${nos}。库存已按行扣减。`
    infoMessage.value = tip
    submitFeedback.succeed({
      dispenseId: slips[0]?.dispenseId || '',
      dispenseNo: slips[0]?.dispenseNo || '',
    }, tip)
    setDispense(slips[0])
    recordAction('发药已完成', nos)
    issued.value = [...slips, ...issued.value.filter((item) => !slips.some((slip) => slip.dispenseId === item.dispenseId))]
    dispenseWrite.fingerprint = ''
    dispenseWrite.key = ''
    if (writeContext === chargeContext.value && writeRoute === routeRequest) await refreshAfterWrite()
    if (result.refreshWarning) refreshWarning.value = result.refreshWarning
  } catch (error) {
    const slips = Array.isArray(error.partialDispenses) ? error.partialDispenses : []
    const message = error instanceof ApiError ? error.message : '发药失败，请稍后重试'
    errorMessage.value = slips.length ? `其余发药失败：${message}` : message
    errorCode.value = error instanceof ApiError ? error.code : ''
    if (slips.length) {
      const tip = `部分发药成功：${slips.map((item) => item.dispenseNo).join('、')}。请核对剩余明细后继续。`
      infoMessage.value = tip
      submitFeedback.succeed({
        dispenseId: slips[0]?.dispenseId || '',
        dispenseNo: slips[0]?.dispenseNo || '',
      }, tip)
      issued.value = [...slips, ...issued.value.filter((item) => !slips.some((slip) => slip.dispenseId === item.dispenseId))]
      if (writeContext === chargeContext.value && writeRoute === routeRequest) {
        pendingNeedsRefresh.value = true
        await refreshAfterWrite()
      }
      return
    }
    submitFeedback.fail(message, error instanceof ApiError ? error.code : '')
    if (searched.value) {
      try {
        await reloadPending()
      } catch {
        // 保留本次发药错误，不改成刷新失败文案。
      }
    }
    try {
      await loadIssued()
    } catch {
      // 已发列表刷新失败时仍显示发药错误。
    }
  } finally {
    dispensing.value = false
  }
}

function onDispenseSelected() {
  submit(rows.value.filter((row) => row.selected))
}

function onDispenseAll() {
  if (busy.value || pendingNeedsRefresh.value) return
  for (const row of rows.value) {
    row.selected = true
    row.dispenseQty = row.availableQty
  }
  submit(rows.value)
}

function rowHint(row) {
  return stockHint(row.stockQty, Number(row.dispenseQty))
}

function rowShort(row) {
  return stockIsShort(row.stockQty, Number(row.dispenseQty))
}

function refundTo(chargeId) {
  return {
    name: 'outpatient-module',
    params: { module: props.returnOnly ? 'refund' : 'charge' },
    query: { view: 'records', charge_id: chargeId },
  }
}

const allDispenseTo = computed(() => ({
  name: 'outpatient-module',
  params: { module: props.returnOnly ? 'refund' : 'dispense' },
  query: { view: 'issued', ...(chargeContext.value ? { charge_id: chargeContext.value } : {}) },
}))

async function openReturn(slip) {
  if (returning.value || submitting.value) return
  errorMessage.value = ''
  returnError.value = ''
  returnResult.value = null
  returnReason.value = ''
  try {
    const fresh = await fetchDispense(session.accessToken, slip.dispenseId)
    setDispense(fresh)
    returnSlip.value = fresh
    returnLines.value = fresh.items
      .filter((item) => item.returnableQty > 0)
      .map((item) => ({
        ...item,
        selected: true,
        returnQty: item.returnableQty,
      }))
    returnWrite.fingerprint = ''
    returnWrite.key = ''
    drawerOpen.value = true
  } catch (error) {
    errorMessage.value = error instanceof ApiError ? error.message : '无法打开退药'
    errorCode.value = error instanceof ApiError ? error.code : ''
  }
}

function closeDrawer() {
  if (returning.value || submitting.value) return
  drawerOpen.value = false
}

async function onReturn() {
  if (returning.value || submitting.value || !returnSlip.value || returnResult.value) return
  returnError.value = ''
  const items = returnLines.value
    .filter((item) => item.selected && Number(item.returnQty) > 0)
    .map((item) => ({
      dispenseItemId: item.dispenseItemId,
      chargeItemId: item.chargeItemId,
      returnQty: Number(item.returnQty),
      selected: true,
    }))
  const fingerprint = JSON.stringify({
    dispenseId: returnSlip.value.dispenseId,
    reason: returnReason.value,
    items,
  })
  if (returnWrite.fingerprint !== fingerprint) {
    returnWrite.fingerprint = fingerprint
    returnWrite.key = crypto.randomUUID()
  }
  returning.value = true
  submitFeedback.beginAttempt('return')
  try {
    const result = await returnDispense(session.accessToken, {
      dispenseId: returnSlip.value.dispenseId,
      reason: returnReason.value,
      idempotencyKey: returnWrite.key,
      items,
    })
    returnResult.value = result
    setDispense({ ...result, items: returnLines.value })
    recordAction('退药已完成', result.dispenseNo || result.dispenseId || '')
    const tip = `${result.stockNote}。可继续退费。`
    infoMessage.value = tip
    submitFeedback.succeed({
      dispenseId: result.dispenseId || returnSlip.value.dispenseId || '',
      dispenseNo: result.dispenseNo || returnSlip.value.dispenseNo || '',
    }, tip)
    returnWrite.fingerprint = ''
    returnWrite.key = ''
    returnSlip.value = { ...returnSlip.value, ...result }
    issued.value = issued.value.map((item) => item.dispenseId === result.dispenseId ? { ...item, ...result } : item)
    await refreshAfterWrite()
    if (result.refreshWarning) refreshWarning.value = result.refreshWarning
    if (detailId.value === result.dispenseId) {
      const fresh = issued.value.find((item) => item.dispenseId === result.dispenseId)
      if (!fresh) detailId.value = ''
    }
  } catch (error) {
    returnError.value = error instanceof ApiError ? error.message : '退药失败，请稍后重试'
    submitFeedback.fail(returnError.value, error instanceof ApiError ? error.code : '')
  } finally {
    returning.value = false
  }
}

async function onReset() {
  if (busy.value) return
  clearBanner()
  await resetDispenseDemo(session.accessToken)
  rows.value = []
  searched.value = false
  emptyKind.value = 'prompt'
  loadMode.value = ''
  detailId.value = ''
  drawerOpen.value = false
  issuedPatient.value = ''
  issuedNo.value = ''
  issuedToday.value = true
  infoMessage.value = '已重置演示数据'
  await loadIssued()
}

function useDemo(text) {
  query.value = text
  onSearch()
}

function queryText(value) {
  const raw = Array.isArray(value) ? value[0] : value
  return typeof raw === 'string' ? raw.trim() : ''
}

let routeRequest = 0
watch([() => route.query.view, () => queryText(route.query.charge_id ?? route.query.chargeId)], async ([view, context], before) => {
  const request = ++routeRequest
  pendingRequest += 1
  loading.value = false
  viewMode.value = props.returnOnly || view === 'issued' ? 'issued' : 'pending'
  chargeContext.value = context
  drawerOpen.value = false
  if (!context || view === 'issued') loading.value = false
  if (context) issuedToday.value = false
  else if (before?.[1]) { issuedToday.value = true; query.value = ''; rows.value = []; searched.value = false; pendingNeedsRefresh.value = false }
  if (context && view !== 'issued' && !props.returnOnly) {
    query.value = context
    clearBanner()
    try {
      await readPending('search', context)
    } catch (error) { if (request === routeRequest) showError(error) }
  }
  if (before) await loadIssued()
}, { immediate: true })

onMounted(async () => {
  searchInput.value?.focus()
  await loadIssued()
})

</script>

<template>
  <section class="dispense-page" aria-labelledby="dispense-title">
    <p v-if="!returnOnly" class="crumb">
      <RouterLink :to="{ name: 'home' }">首页</RouterLink>
      <span aria-hidden="true"> / </span>
      <RouterLink :to="{ name: 'workbench' }">门诊业务</RouterLink>
      <span aria-hidden="true"> / </span>
      <span>发药</span>
    </p>

    <BusinessPageHeader v-if="!returnOnly" title-id="dispense-title" title="门诊发药" eyebrow="OUTPATIENT · PHARMACY" icon="dispense" description="只发已收费药品。核对患者、药品与本次数量，支持部分发药。已发药品须先退药，再去退费。">
      <template #actions><span class="dispense-header-note"><span></span>{{ PHARMACY.deptName }}</span></template>
    </BusinessPageHeader>
    <p v-if="mockMode" class="dispense-demo-note">演示账号 system / 1</p>

    <div v-if="!returnOnly" class="dispense-workbar">
      <div class="dispense-view-switch" role="group" aria-label="发药工作区">
        <button type="button" :class="{ 'is-active': viewMode === 'pending' }" :aria-pressed="viewMode === 'pending'" @click="viewMode = 'pending'">待发工作台 <span>{{ searched ? rows.length : '—' }}</span></button>
        <button type="button" :class="{ 'is-active': viewMode === 'issued' }" :aria-pressed="viewMode === 'issued'" @click="viewMode = 'issued'">已发查询 / 退药 <span>{{ issued.length }}</span></button>
      </div>
      <p class="dispense-workbar-note">{{ PHARMACY.deptName }} · 操作员 {{ operatorName }}</p>
    </div>

    <SubmitStatusBanner :feedback="submitFeedback.state" />
    <p v-if="errorMessage" class="banner" role="alert" data-testid="dispense-error" :data-code="errorCode">{{ errorMessage }}</p>
    <p v-if="infoMessage" class="ok-banner" role="status" data-testid="dispense-ok">{{ infoMessage }}</p>
    <div v-if="refreshWarning" class="banner refresh-warning" role="status" data-testid="dispense-refresh-warning"><span>{{ refreshWarning }}</span><button class="text-btn" type="button" :disabled="busy || issuedLoading" @click="refreshAfterWrite">重试刷新</button></div>
    <p v-if="chargeContext" class="dispense-context-note" role="status">当前收费单：{{ chargeContext }} <RouterLink class="text-btn" :to="allDispenseTo">查看全部</RouterLink></p>

    <div v-if="!returnOnly && viewMode === 'pending'" class="dispense-pending-workspace">
    <form class="patient-panel dispense-search-panel" @submit.prevent="onSearch">
        <div class="panel-head">
          <div><p class="dispense-section-kicker">01 · LOCATE</p><h2>检索待发</h2></div>
          <button v-if="mockMode" class="text-btn" type="button" :disabled="busy" @click="onReset">重置演示数据</button>
        </div>
        <div class="charge-search">
          <label class="field">
            <span>患者 / 挂号 / 收费单 / 处方号</span>
            <input
              ref="searchInput"
              v-model="query"
              data-testid="dispense-search"
              :placeholder="searchPlaceholder"
              autocomplete="off"
            />
          </label>
          <button class="button button-compact" type="submit" data-testid="dispense-submit-search" :disabled="busy">检索</button>
          <button class="button button-ghost button-compact" type="button" data-testid="dispense-today" :disabled="busy" @click="onToday">
            今日待发
          </button>
          <button class="button button-ghost button-compact" type="button" :disabled="busy || !searched" @click="onRefreshStock">
            刷新库存
          </button>
        </div>
        <div v-if="mockMode" class="demo-chips" aria-label="演示检索">
          <button
            v-for="item in DEMO_CASES"
            :key="item.query"
            class="demo-chip"
            type="button"
            :disabled="busy"
            @click="useDemo(item.query)"
          >
            {{ item.label }}
          </button>
        </div>
      </form>

      <p v-if="loading" class="result-empty dispense-loading" role="status">正在载入待发药品…</p>
      <BusinessEmptyState
        v-else-if="!searched"
        variant="revisit"
        module-label="发药"
        queue-label="今日待发"
        title="请选择已收费患者或处方"
        description="检索后只载入已收费、尚未发完的药品，库存与本次发药数量会在右侧展开。"
        empty-label="尚未检索待发队列"
        empty-description="可以按患者、挂号、收费单或处方号检索，也可以直接查看今日待发。"
        test-id="dispense-queue-workspace"
      >
        <template #actions>
          <button class="button button-primary button-compact" type="button" :disabled="busy" @click="onToday">查看今日待发</button>
          <RouterLink class="button button-ghost button-compact" :to="chargeTo">去收费查询</RouterLink>
        </template>
      </BusinessEmptyState>

      <BusinessEmptyState
        v-else-if="emptyKind === 'rejected' || rows.length === 0"
        variant="revisit"
        module-label="发药"
        queue-label="今日待发"
        :title="emptyKind === 'rejected' ? '待发读取失败' : (emptyKind === 'today' ? '今日无待发' : '暂无待发药品')"
        :description="emptyKind === 'rejected' ? errorMessage : (emptyKind === 'done' ? '已发完的药品不在待发列表，可在已发查询中办理退药。' : emptyKind === 'today' ? '今天没有已收费且未发完的药品。' : '未收费或已发完的处方不会出现在这里。')"
        :empty-label="emptyKind === 'rejected' ? '待发队列读取失败' : '暂无待发药品'"
        empty-description="调整检索条件或完成收费后，符合条件的患者会进入待发队列。"
        test-id="dispense-empty"
      >
        <template #actions>
          <button v-if="emptyKind === 'rejected'" class="button button-primary button-compact" type="button" :disabled="busy" @click="loadMode === 'today' ? onToday() : onSearch()">重试读取</button>
          <RouterLink v-else class="button button-primary button-compact" :to="chargeTo">去收费查询</RouterLink>
          <button v-if="emptyKind === 'done'" class="button button-ghost button-compact" type="button" @click="viewMode = 'issued'">查看已发记录</button>
        </template>
      </BusinessEmptyState>

      <form v-else class="patient-panel dispense-lines-panel" data-testid="dispense-lines" @submit.prevent="onDispenseSelected">
        <div class="panel-head">
          <div><p class="dispense-section-kicker">02 · VERIFY & DISPENSE</p><h2>待发明细</h2></div>
          <span class="dispense-counter">{{ rows.length }} 项药品</span>
        </div>

        <dl v-if="singlePatient" class="summary-grid">
          <div>
            <dt>患者</dt>
            <dd>{{ singlePatient.patientName }}</dd>
          </div>
          <div>
            <dt>患者号</dt>
            <dd>{{ singlePatient.patientNo }}</dd>
          </div>
          <div>
            <dt>挂号</dt>
            <dd>{{ singlePatient.registerNo }}</dd>
          </div>
          <div>
            <dt>收费单</dt>
            <dd>{{ singlePatient.squareNo }}</dd>
          </div>
          <div>
            <dt>处方号</dt>
            <dd>{{ singlePatient.orderNo }}</dd>
          </div>
        </dl>
        <p v-else class="dispense-multi-note">今日待发含多位患者。确认发药只提交勾选的行，并按收费单分别生成发药单。</p>

        <div class="dispense-line-summary" aria-label="本次发药概况">
          <div><span>已选明细</span><strong>{{ selectedRows.length }}<small> / {{ rows.length }} 项</small></strong></div>
          <div><span>涉及患者</span><strong>{{ selectedPatientCount }}<small>位患者</small></strong></div>
          <div :class="{ 'is-short': shortageCount > 0 }"><span>库存不足</span><strong>{{ shortageCount }}<small>项明细</small></strong></div>
          <p>请逐行核对药品、可发数量和当前库存。</p>
        </div>

        <p id="dispense-pending-scroll" class="table-scroll-hint">左右滑动可查看完整药品、数量与库存</p>
        <div class="line-scroll" role="region" aria-label="待发明细，可横向滚动" aria-describedby="dispense-pending-scroll" tabindex="0">
          <table class="line-table">
            <caption class="sr-only">待发药品</caption>
            <thead>
              <tr>
                <th>发</th>
                <th>患者</th>
                <th>药品</th>
                <th>规格</th>
                <th>单位</th>
                <th>已收费</th>
                <th>已发</th>
                <th>可发</th>
                <th>本次发药</th>
                <th>库存</th>
                <th>行态</th>
                <th>处方 / 收费</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="(row, index) in rows" :key="row.chargeItemId" :class="{ 'is-short': rowShort(row) }">
                <td>
                  <input v-model="row.selected" type="checkbox" :disabled="busy || pendingNeedsRefresh" :aria-label="`选择 ${row.itemName}`" :data-testid="`dispense-check-${index}`" />
                </td>
                <td class="dispense-patient-name">{{ row.patientName }}</td>
                <td class="dispense-drug-name">{{ row.itemName }}</td>
                <td>{{ row.spec || '—' }}</td>
                <td>{{ row.units || '—' }}</td>
                <td>{{ formatQty(row.chargedQty) }}</td>
                <td>{{ formatQty(row.dispensedQty) }}</td>
                <td class="dispense-available-qty">{{ formatQty(row.availableQty) }}</td>
                <td class="col-narrow">
                  <input
                    v-model="row.dispenseQty"
                    :disabled="busy || pendingNeedsRefresh"
                    type="number"
                    min="1"
                    :max="row.availableQty"
                    step="1"
                    :aria-label="`${row.itemName} 本次发药数量`"
                    :data-testid="`dispense-qty-${index}`"
                  />
                  <span v-if="Number(row.dispenseQty) > 0 && Number(row.dispenseQty) < row.availableQty" class="field-hint">部分发</span>
                </td>
                <td><span class="dispense-stock-state" :class="{ 'stock-bad': rowShort(row) }">{{ rowHint(row) }}</span></td>
                <td>
                  <span class="ticket-status" :data-status="row.lineDispenseStatus">{{ lineStatusLabel(row.lineDispenseStatus) }}</span>
                </td>
                <td class="dispense-reference"><span>{{ row.orderNo }}</span><span>{{ row.squareNo }}</span></td>
              </tr>
            </tbody>
          </table>
        </div>

        <div class="dispense-confirm-bar">
        <div class="dispense-confirm-copy"><strong>已勾选 {{ selectedRows.length }} 项药品</strong><p class="field-hint">全部发药会把每一行改成可发数量再提交。某一行库存不够时，整单不扣库存。</p></div>
        <div class="panel-actions">
          <button class="button button-compact" type="submit" data-testid="dispense-confirm" :disabled="busy || pendingNeedsRefresh || selectedRows.length === 0">
            {{ dispensing ? '正在发药…' : '确认发药' }}
          </button>
          <button class="button button-ghost button-compact" type="button" data-testid="dispense-all" :disabled="busy || pendingNeedsRefresh" @click="onDispenseAll">
            全部发药
          </button>
        </div>
        </div>
      </form>
      </div>

      <section v-show="viewMode === 'issued'" class="patient-panel dispense-issued-panel" aria-labelledby="issued-title">
        <div class="panel-head">
          <div><p class="dispense-section-kicker">DISPENSE HISTORY</p><h2 id="issued-title">已发查询</h2></div><span class="dispense-counter">{{ issued.length }} 条记录</span>
        </div>
        <form class="issued-filters" @submit.prevent="loadIssued">
          <label class="checkin-demo">
            <input v-model="issuedToday" type="checkbox" @change="loadIssued" />
            今日
          </label>
          <label class="field">
            <span>患者号 / 姓名</span>
            <input v-model="issuedPatient" placeholder="例如孙已发" autocomplete="off" />
          </label>
          <label class="field">
            <span>发药单号</span>
            <input v-model="issuedNo" placeholder="例如 FY-DONE-001" autocomplete="off" />
          </label>
          <button class="button button-ghost button-compact" type="submit" :disabled="issuedLoading">查询</button>
        </form>

        <p v-if="issuedLoading" class="result-empty" role="status">正在读取发药记录…</p>
        <BusinessEmptyState v-else-if="issuedError" icon="dispense" title="发药记录读取失败" :description="issuedError" eyebrow="读取失败"><button class="button button-ghost button-compact" type="button" @click="loadIssued">重试查询</button></BusinessEmptyState>
        <BusinessEmptyState
          v-else-if="issued.length === 0"
          variant="revisit"
          module-label="退药"
          queue-label="已发待退"
          title="暂无发药记录"
          description="调整患者、发药单号或日期范围后，符合条件的记录会进入左侧队列。"
          empty-label="暂无可退药记录"
          empty-description="查询到已发药记录后，可在右侧核对数量、原因并办理退药。"
          primary-text="查看待发"
          :primary-to="{ name: 'outpatient-module', params: { module: 'dispense' } }"
          secondary-text="收费查询"
          :secondary-to="{ name: 'outpatient-module', params: { module: 'charge' }, query: { view: 'records' } }"
          test-id="issued-empty"
        />
        <p v-if="issued.length && !issuedLoading && !issuedError" id="dispense-issued-scroll" class="table-scroll-hint">左右滑动可查看完整记录与退药操作</p>
        <div v-if="issued.length && !issuedLoading && !issuedError" class="line-scroll" role="region" aria-label="已发记录，可横向滚动" aria-describedby="dispense-issued-scroll" tabindex="0">
          <table class="line-table">
            <caption class="sr-only">发药单</caption>
            <thead>
              <tr>
                <th>时间</th>
                <th>发药单号</th>
                <th>患者</th>
                <th>挂号</th>
                <th>收费单</th>
                <th>明细</th>
                <th>行态</th>
                <th>操作</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="item in issued" :key="item.dispenseId" :class="{ 'is-active': detailId === item.dispenseId }">
                <td>{{ formatOperateAt(item.operateAt) }}</td>
                <td class="dispense-record-no">{{ item.dispenseNo }}</td>
                <td class="dispense-patient-name">{{ item.patientName }}</td>
                <td>{{ item.registerNo }}</td>
                <td>{{ item.squareNo }}</td>
                <td>{{ item.summary }}</td>
                <td>
                  <span class="ticket-status" :data-status="item.lineDispenseStatus">{{ lineStatusLabel(item.lineDispenseStatus) }}</span>
                </td>
                <td class="ticket-actions">
                  <button class="text-btn" type="button" :aria-label="`查看发药单 ${item.dispenseNo} 详情`" :aria-expanded="detailId === item.dispenseId" @click="detailId = detailId === item.dispenseId ? '' : item.dispenseId">详情</button>
                  <button
                    v-if="item.items.some((line) => line.returnableQty > 0)"
                    class="text-btn"
                    type="button"
                    :aria-label="`退药发药单 ${item.dispenseNo}`"
                    :disabled="issuedLoading || Boolean(issuedError)"
                    data-testid="issued-return"
                    @click="openReturn(item)"
                  >
                    退药
                  </button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <div v-if="detail" class="dispense-detail" data-testid="dispense-detail">
          <div class="dispense-detail-head"><div><p class="dispense-section-kicker">RECORD DETAIL</p><h3>{{ detail.dispenseNo }} · {{ detail.patientName }}</h3><p class="field-hint">{{ detail.orderNo }} · {{ detail.squareNo }} · {{ PHARMACY.deptName }}</p></div><button class="text-btn" type="button" @click="detailId = ''">收起</button></div>
          <ul>
            <li v-for="item in detail.items" :key="item.dispenseItemId">
              <strong>{{ item.itemName }} <span>{{ item.units }}</span></strong>
              <dl><div><dt>已发</dt><dd>{{ formatQty(item.dispenseQty) }}</dd></div><div><dt>已退</dt><dd>{{ formatQty(item.returnedQty) }}</dd></div><div><dt>可退</dt><dd>{{ formatQty(item.returnableQty) }}</dd></div></dl>
            </li>
          </ul>
        </div>
      </section>

      <div v-if="drawerOpen" class="drawer-backdrop" @click.self="closeDrawer" @keydown.esc="closeDrawer">
        <aside class="refund-drawer" role="dialog" aria-modal="true" aria-labelledby="return-title">
          <div class="panel-head">
            <div><p class="dispense-section-kicker">RETURN WORKSPACE</p><h2 id="return-title">退药</h2></div>
            <button class="text-btn" type="button" :disabled="returning" @click="closeDrawer">关闭</button>
          </div>
          <dl v-if="returnSlip" class="return-slip-summary"><div><dt>患者</dt><dd>{{ returnSlip.patientName }}</dd></div><div><dt>发药单</dt><dd>{{ returnSlip.dispenseNo }}</dd></div><div><dt>收费单</dt><dd>{{ returnSlip.squareNo }}</dd></div></dl>
          <p class="return-drawer-note">先确认可退数量与原因。退药成功后库存回加，再继续办理退费。</p>

          <p v-if="returnLines.length === 0" class="result-empty">没有可退明细（未发或已退完）</p>
          <ul v-else class="return-list">
            <li v-for="(item, index) in returnLines" :key="item.dispenseItemId">
              <label class="checkin-demo">
                <input v-model="item.selected" type="checkbox" :aria-label="`退 ${item.itemName}`" :disabled="Boolean(returnResult)" />
                {{ item.itemName }}
              </label>
              <p class="return-quantity-note">已发 {{ formatQty(item.dispenseQty) }} · 已退 {{ formatQty(item.returnedQty) }} · 可退 <strong>{{ formatQty(item.returnableQty) }} {{ item.units }}</strong></p>
              <label class="field">
                <span>退数量</span>
                <input
                  v-model="item.returnQty"
                  type="number"
                  min="1"
                  :max="item.returnableQty"
                  :disabled="Boolean(returnResult)"
                  step="1"
                  :aria-label="`${item.itemName} 退数量`"
                  :data-testid="`return-qty-${index}`"
                />
              </label>
            </li>
          </ul>

          <label v-if="!returnResult" class="field">
            <span>退药原因 <abbr class="req" title="必填">*</abbr></span>
            <textarea
              v-model="returnReason"
              data-testid="return-reason"
              maxlength="80"
              placeholder="请填写，例如患者要求"
            ></textarea>
          </label>

          <p v-if="returnError" class="banner" role="alert" data-testid="return-error">{{ returnError }}</p>
          <div v-if="returnResult" class="ok-banner" role="status" data-testid="return-ok">
            <span class="ok-title">已退药</span>
            <span>{{ returnResult.stockNote }}</span>
            <RouterLink class="button button-ghost button-compact" data-testid="go-refund" :to="refundTo(returnResult.chargeId)">
              去退费
            </RouterLink>
          </div>
          <div v-else class="panel-actions return-confirm-bar">
            <p>退药后可继续退费</p>
            <button
              class="button button-compact"
              type="button"
              data-testid="return-confirm"
              :disabled="returning || returnLines.length === 0"
              @click="onReturn"
            >
              {{ returning ? '正在退药…' : '确认退药' }}
            </button>
          </div>
        </aside>
      </div>
  </section>
</template>

<style scoped src="./dispense-design.css"></style>
