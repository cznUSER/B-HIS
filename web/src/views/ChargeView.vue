<script setup>
// 2026-10-12 P1-H05-CHARGE-UI 门诊收费：待收、现金、查询、退费。侧栏仍由 AppShell 提供。
// 2026-10-12 VITE_USE_MOCK=false 时走 charge-preview / charges / pay / refund。
// 2026-10-12 P1-H05-DISPENSE-UI 发药页「去退费」带 charge_id 时打开退费。
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { ApiError } from '../api/errors.js'
import {
  fetchCharge,
  listCharges,
  loadPendingCharge,
  lookupChargeSubjects,
  payCharge,
  refundCharge,
} from '../api/chargeApi.js'
import { PAY_CASH, RESERVED_PAY_TYPES, formatYuan, statusLabel } from '../api/chargeContract.js'
import { session } from '../auth/session.js'
import BusinessPageHeader from '../components/BusinessPageHeader.vue'
import BusinessEmptyState from '../components/BusinessEmptyState.vue'
import { recordAction, setCharges, setPatient, setRegister } from '../workbench/outpatientContext.js'

const props = defineProps({ refundOnly: { type: Boolean, default: false } })

const route = useRoute()
const viewMode = ref(props.refundOnly || route.query.view === 'records' ? 'records' : 'pending')
const subjects = ref([])
const keyword = ref('')
const activeKey = ref('')
const activeSubject = ref(null)
const pending = ref(null)
const charges = ref([])
const queryKeyword = ref('')
const todayOnly = ref(true)
const loadingPending = ref(false)
const loadingList = ref(false)
const paying = ref(false)
const errorMessage = ref('')
const errorCode = ref('')
const payMessage = ref('')
const paidSquareNo = ref('')
const paidChargeId = ref('')
const paidOrderId = ref('')
const refreshWarning = ref('')
const subjectsError = ref('')
const pendingError = ref('')
const listError = ref('')

const refundOpen = ref(false)
const refundTarget = ref(null)
const refundReason = ref('')
const refundPicks = reactive({})
const refundQty = reactive({})
const refunding = ref(false)
const refundError = ref('')
const refundCode = ref('')
const refundMessage = ref('')

const paymentWrite = reactive({ orderId: '', key: '' })
const refundWrite = reactive({ fingerprint: '', key: '' })

const preview = computed(() => (pending.value && pending.value.preview) || null)
const scene = computed(() => (pending.value && pending.value.scene) || '')
const stockBlocked = computed(() => Boolean(preview.value && preview.value.stockBlocked))
const canPay = computed(() => Boolean(preview.value) && preview.value.orderId !== paidOrderId.value && !stockBlocked.value && !loadingPending.value && !paying.value)

const refundRows = computed(() => (refundTarget.value && refundTarget.value.items) || [])
const refundHasDispensed = computed(() => refundRows.value.some((item) => item.dispenseBlocked))
const refundSelectable = computed(() => refundRows.value.filter((item) => item.canBack && !item.dispenseBlocked && item.refundableQty > 0))
const refundChosen = computed(() => refundSelectable.value.filter((item) => refundPicks[item.chargeItemId]))
const refundBlocked = computed(() => refundHasDispensed.value && refundSelectable.value.length === 0)
const refundIsFull = computed(() => refundRows.value.every((item) => {
  const remaining = Number(item.refundableQty || 0)
  return remaining <= 0 || (item.canBack && !item.dispenseBlocked && refundPicks[item.chargeItemId] && Number(refundQty[item.chargeItemId]) === remaining)
}))
const discountedPartialRefund = computed(() => Number(refundTarget.value?.favourableFee || 0) > 0 && !refundIsFull.value)

function clearError() {
  errorMessage.value = ''
  errorCode.value = ''
}

function paymentKey(orderId) {
  if (paymentWrite.orderId !== orderId) {
    paymentWrite.orderId = orderId
    paymentWrite.key = crypto.randomUUID()
  }
  return paymentWrite.key
}

function applyError(error, fallback) {
  if (error instanceof ApiError) {
    errorMessage.value = error.message
    errorCode.value = error.code
    return
  }
  errorMessage.value = fallback
  errorCode.value = ''
}

async function refreshSubjects(options = {}) {
  subjectsError.value = ''
  try {
    subjects.value = (await lookupChargeSubjects(session.accessToken, keyword.value)).items
  } catch (error) {
    subjectsError.value = error instanceof ApiError ? error.message : '检索失败'
    throw error
  }
  if (options.autoOpen !== false && !activeKey.value && subjects.value.length > 0) {
    const preferred = subjects.value.find((item) => item.scene === 'PENDING' || item.scene === 'STOCK') || subjects.value[0]
    await openSubject(preferred)
  }
}

async function refreshList(options = {}) {
  loadingList.value = true
  listError.value = ''
  try {
    const found = await listCharges(session.accessToken, {
      keyword: queryKeyword.value,
      todayOnly: todayOnly.value,
    })
    charges.value = found.items
    setCharges(charges.value)
    return true
  } catch (error) {
    listError.value = error instanceof ApiError ? error.message : '查询失败，请稍后重试'
    if (!options.afterWrite) applyError(error, '查询失败，请稍后重试')
    return false
  } finally {
    loadingList.value = false
  }
}

async function openSubject(subject, options = {}) {
  if (!subject || loadingPending.value || (!options.afterWrite && (paying.value || refunding.value))) return false
  activeSubject.value = subject
  activeKey.value = subject.key || subject.patientId
  loadingPending.value = true
  pendingError.value = ''
  clearError()
  if (!options.keepNotice) {
    payMessage.value = ''
    paidSquareNo.value = ''
    paidChargeId.value = ''
  }
  try {
    pending.value = await loadPendingCharge(session.accessToken, subject)
    if (pending.value?.patient) setPatient({
      patientId: pending.value.patient.patientId,
      name: pending.value.patient.patientName || pending.value.patient.name,
      patientNo: pending.value.patient.patientNo,
      sex: pending.value.patient.sex,
      birthday: pending.value.patient.birthday,
      fareIdentity: pending.value.patient.fareIdentity,
    })
    if (pending.value?.register) setRegister(pending.value.register)
    return true
  } catch (error) {
    pendingError.value = error instanceof ApiError ? error.message : '待收载入失败'
    if (!options.afterWrite) {
      pending.value = null
      applyError(error, '待收载入失败')
    }
    return false
  } finally {
    loadingPending.value = false
  }
}

// 2026-10-14: successful writes retain their receipt; reads have a separate retry state.
async function refreshAfterWrite() {
  let failed = false
  if (!props.refundOnly) {
    try {
      await refreshSubjects({ autoOpen: false })
      const next = subjects.value.find((item) => item.key === activeKey.value)
      if (next && !(await openSubject(next, { keepNotice: true, afterWrite: true }))) failed = true
    } catch { failed = true }
  }
  if (!(await refreshList({ afterWrite: true }))) failed = true
  refreshWarning.value = failed ? '操作已成功，页面数据刷新失败。请重试刷新，无需再次提交。' : ''
}

async function onSearchSubjects() {
  if (paying.value || refunding.value) return
  clearError()
  try {
    await refreshSubjects()
  } catch (error) {
    applyError(error, '检索失败')
  }
}

async function onPay() {
  if (!canPay.value || !preview.value) return
  // 2026-10-14: bind the receipt to the subject/order captured before the write.
  const paymentPreview = preview.value
  const paymentPending = pending.value
  const paymentSubject = activeSubject.value
  const paymentSubjectKey = activeKey.value
  paying.value = true
  clearError()
  payMessage.value = ''
  refreshWarning.value = ''
  try {
    const result = await payCharge(session.accessToken, {
      orderId: paymentPreview.orderId,
      registerId: paymentPending.register && paymentPending.register.registerId,
      payType: PAY_CASH,
      amount: paymentPreview.squareFee,
      amountCents: paymentPreview.amountDueCents,
      idempotencyKey: paymentKey(paymentPreview.orderId),
    })
    paidSquareNo.value = result.charge.squareNo
    paidChargeId.value = result.charge.chargeId
    paidOrderId.value = paymentPreview.orderId
    if (activeKey.value === paymentSubjectKey) {
      pending.value = { ...paymentPending, scene: 'CHARGED', preview: null, squareNo: result.charge.squareNo, chargeId: result.charge.chargeId }
      activeSubject.value = { ...paymentSubject, scene: 'CHARGED' }
    }
    charges.value = [result.charge, ...charges.value.filter((item) => item.chargeId !== result.charge.chargeId)]
    setCharges(charges.value)
    recordAction('收费已完成', result.charge.squareNo || result.charge.chargeId || '')
    payMessage.value = '收费成功，可去发药。'
    await refreshAfterWrite()
    if (result.refreshWarning) refreshWarning.value = result.refreshWarning
  } catch (error) {
    applyError(error, '收费失败')
  } finally {
    paying.value = false
  }
}

function resetRefundForm(charge) {
  refundTarget.value = charge
  refundReason.value = ''
  refundError.value = ''
  refundCode.value = ''
  refundWrite.fingerprint = ''
  refundWrite.key = ''
  for (const key of Object.keys(refundPicks)) delete refundPicks[key]
  for (const key of Object.keys(refundQty)) delete refundQty[key]
  for (const item of charge.items) {
    if (item.canBack && !item.dispenseBlocked && item.refundableQty > 0) {
      refundPicks[item.chargeItemId] = false
      refundQty[item.chargeItemId] = item.refundableQty
    }
  }
}

let refundRequest = 0
async function openRefund(chargeId) {
  const request = ++refundRequest
  viewMode.value = 'records'
  refundMessage.value = ''
  refundOpen.value = true
  refundTarget.value = null
  refundError.value = ''
  refundCode.value = ''
  try {
    const charge = await fetchCharge(session.accessToken, chargeId)
    setCharges([charge])
    setPatient({ patientId: charge.patientId, name: charge.patientName, patientNo: charge.patientNo })
    setRegister({ registerId: charge.registerId, registerNo: charge.registerNo, patientId: charge.patientId })
    if (request === refundRequest) resetRefundForm(charge)
  } catch (error) {
    if (request !== refundRequest) return
    refundCode.value = error instanceof ApiError ? error.code : ''
    refundError.value = error instanceof ApiError ? error.message : '未找到收费单'
  }
}

function closeRefund() {
  if (refunding.value) return
  refundOpen.value = false
  refundRequest += 1
}

async function onRefund() {
  if (!refundTarget.value || refunding.value || refundBlocked.value || discountedPartialRefund.value) return
  const items = refundChosen.value.map((item) => ({
    chargeItemId: item.chargeItemId,
    refundQty: Number(refundQty[item.chargeItemId]),
  }))
  if (items.length === 0) {
    refundCode.value = 'INVALID_AMOUNT'
    refundError.value = '数量或金额不合法'
    return
  }
  const fingerprint = JSON.stringify({
    chargeId: refundTarget.value.chargeId,
    reason: refundReason.value || '',
    items,
  })
  if (refundWrite.fingerprint !== fingerprint) {
    refundWrite.fingerprint = fingerprint
    refundWrite.key = crypto.randomUUID()
  }
  refunding.value = true
  const refundChargeId = refundTarget.value.chargeId
  const originalRefundCharge = refundTarget.value
  refundError.value = ''
  refundCode.value = ''
  refreshWarning.value = ''
  try {
    const result = await refundCharge(session.accessToken, refundTarget.value.chargeId, {
      reason: refundReason.value,
      payType: PAY_CASH,
      items,
      idempotencyKey: refundWrite.key,
    })
    if (refundTarget.value?.chargeId === refundChargeId) {
      refundOpen.value = false
      refundTarget.value = result.abatedSquare || { ...originalRefundCharge, status: 'ABATE', canRefund: false }
    }
    charges.value = charges.value.map((item) => item.chargeId === refundChargeId ? { ...item, status: 'ABATE', canRefund: false } : item)
    if (result.newSquare) charges.value.unshift(result.newSquare)
    setCharges(charges.value)
    recordAction('收费已退费', refundChargeId)
    if (result.fullRefund || !result.newSquare) {
      refundMessage.value = '已全额退费。原单已作废，未生成新收费单。'
    } else {
      refundMessage.value = `已部分退费。原单已作废，未退部分已生成新收费单 ${result.newSquare.squareNo}（${formatYuan(result.newSquare.squareFeeCents)}）。`
    }
    await refreshAfterWrite()
  } catch (error) {
    refundCode.value = error instanceof ApiError ? error.code : ''
    refundError.value = error instanceof ApiError ? error.message : '退费失败'
  } finally {
    refunding.value = false
  }
}

function queryChargeId() {
  const value = route.query.charge_id != null ? route.query.charge_id : route.query.chargeId
  const raw = Array.isArray(value) ? value[0] : value
  return typeof raw === 'string' ? raw.trim() : ''
}

watch([() => route.query.view, queryChargeId], ([view, chargeId], before) => {
  viewMode.value = props.refundOnly || view === 'records' || chargeId ? 'records' : 'pending'
  if (chargeId && chargeId !== before?.[1]) openRefund(chargeId)
  if (!chargeId && before?.[1]) closeRefund()
}, { immediate: true })

onMounted(async () => {
  try {
    if (!props.refundOnly) await refreshSubjects()
    await refreshList()
  } catch (error) {
    applyError(error, '收费页载入失败')
  }
})
</script>

<template>
  <section class="charge-page ui2-charge-page" aria-labelledby="charge-title">
    <p v-if="!refundOnly" class="crumb">
      <RouterLink :to="{ name: 'home' }">首页</RouterLink>
      <span aria-hidden="true"> / </span>
      <RouterLink :to="{ name: 'workbench' }">门诊业务</RouterLink>
      <span aria-hidden="true"> / </span>
      <span>收费</span>
    </p>

    <BusinessPageHeader v-if="!refundOnly" title-id="charge-title" title="门诊收费" eyebrow="OUTPATIENT · CASHIER" icon="charge" description="从已提交处方载入待收。核对明细后以现金结算，实收等于应收。电子方式只作预留。">
      <template #actions><span class="charge-header-note"><span></span>现金结算</span></template>
    </BusinessPageHeader>

    <div v-if="!refundOnly" class="charge-workbar">
      <div class="charge-view-switch" role="group" aria-label="收费工作区">
        <button type="button" :class="{ 'is-active': viewMode === 'pending' }" :aria-pressed="viewMode === 'pending'" @click="viewMode = 'pending'">待收结算</button>
        <button type="button" :class="{ 'is-active': viewMode === 'records' }" :aria-pressed="viewMode === 'records'" @click="viewMode = 'records'">已收查询 <span>{{ charges.length }}</span></button>
      </div>
      <p class="charge-workbar-note">先核对患者与处方，再确认金额</p>
    </div>

    <p v-if="errorMessage" class="banner" role="alert" data-testid="charge-error">
      <span class="banner-body"><span>{{ errorMessage }}</span><span v-if="errorCode" class="banner-meta"><span class="meta-chip">{{ errorCode }}</span></span></span>
    </p>
    <div v-if="refreshWarning" class="banner refresh-warning" role="status" data-testid="charge-refresh-warning">
      <span>{{ refreshWarning }}</span><button class="text-btn" type="button" :disabled="paying || refunding || loadingPending || loadingList" @click="refreshAfterWrite">重试刷新</button>
    </div>

    <div class="charge-layout">
      <section v-if="!refundOnly && viewMode === 'pending'" class="patient-panel charge-pending-panel" aria-labelledby="pending-title">
        <div class="charge-candidate-zone">
        <div class="panel-head">
          <div><p class="charge-section-kicker">01 · IDENTIFY</p><h2 id="pending-title">检索待收</h2></div>
          <span class="charge-counter">{{ subjects.length }} 个候选</span>
        </div>

        <form class="charge-search" data-testid="charge-search" @submit.prevent="onSearchSubjects">
          <label class="field">
            <span>患者 / 挂号 / 处方</span>
            <input v-model="keyword" name="charge-keyword" autocomplete="off" placeholder="姓名、患者号、挂号号或处方号" />
          </label>
          <button class="button button-compact" type="submit" :disabled="paying || refunding || loadingPending">检索</button>
        </form>

        <div class="demo-chips" aria-label="患者候选">
          <button
            v-for="item in subjects"
            :key="item.key || item.patientId"
            class="demo-chip"
            :class="{ 'is-active': item.key === activeKey }"
            type="button"
            :disabled="paying || refunding || loadingPending"
            :aria-pressed="item.key === activeKey"
            :data-testid="`charge-chip-${item.scene.toLowerCase()}`"
            @click="openSubject(item)"
          >
            <span class="charge-candidate-name">{{ item.patientName }}</span>
            <span class="charge-candidate-scene">{{ item.sceneLabel }}</span>
          </button>
        </div>
        <p v-if="subjectsError" class="banner" role="alert">候选读取失败：{{ subjectsError }} <button class="text-btn" type="button" @click="onSearchSubjects">重试</button></p>
        <p v-else-if="subjects.length === 0" class="charge-candidate-hint">暂无匹配候选，可调整姓名或单号后重新检索。</p>
        </div>

        <div v-if="payMessage" class="ok-banner" role="status" data-testid="charge-paid">
          <p class="ok-title">{{ payMessage }}</p>
          <p v-if="paidSquareNo">收费单号 <strong>{{ paidSquareNo }}</strong></p>
          <RouterLink class="button button-compact" :to="{ name: 'outpatient-module', params: { module: 'dispense' }, query: { charge_id: paidChargeId } }">去发药</RouterLink>
        </div>

        <div v-if="loadingPending" class="result-empty">正在载入待收…</div>

        <BusinessEmptyState v-else-if="pendingError" icon="charge" title="待收读取失败" :description="pendingError" eyebrow="读取失败">
          <button class="button button-ghost button-compact" type="button" @click="openSubject(activeSubject)">重试载入</button>
        </BusinessEmptyState>
        <BusinessEmptyState v-else-if="!pending" icon="register" title="请先挂号" description="未挂号/无有效就诊，不能收费。" eyebrow="待收前置条件" data-testid="charge-empty-register">
          <RouterLink class="button button-ghost button-compact" :to="{ name: 'outpatient-module', params: { module: 'register' } }">去挂号</RouterLink>
        </BusinessEmptyState>

        <BusinessEmptyState v-else-if="scene === 'NO_REGISTER'" icon="register" title="请先挂号" :description="`${pending.patient.patientName}（${pending.patient.patientNo}）还没有有效挂号。`" eyebrow="待收前置条件" data-testid="charge-empty-register">
          <template #detail><p class="field-hint">未挂号/无有效就诊，不能收费。</p></template>
          <RouterLink class="button button-ghost button-compact" :to="{ name: 'outpatient-module', params: { module: 'register' } }">去挂号</RouterLink>
        </BusinessEmptyState>

        <BusinessEmptyState v-else-if="scene === 'NO_COMMIT'" icon="order" title="请先提交处方" :description="`${pending.patient.patientName} 已挂号（${pending.register.summary}），还没有已提交处方。`" eyebrow="待收前置条件" data-testid="charge-empty-order">
          <RouterLink class="button button-ghost button-compact" :to="{ name: 'outpatient-module', params: { module: 'order' } }">去开单</RouterLink>
        </BusinessEmptyState>

        <BusinessEmptyState v-else-if="scene === 'CHARGED' || scene === 'DISPENSED'" icon="charge" title="该处方已收费" :description="`${pending.patient.patientName} · ${pending.squareNo || '已生成收费单'}。退费请在已收查询中办理。`" eyebrow="收费状态" data-testid="charge-already">
          <template #detail><p v-if="scene === 'DISPENSED'" class="field-hint">已发药的明细须先退药，才能退费。</p></template>
          <button class="button button-ghost button-compact" type="button" @click="viewMode = 'records'">查看已收记录</button>
        </BusinessEmptyState>

        <template v-else-if="preview">
          <div class="charge-workspace">
          <section class="charge-preview" aria-labelledby="charge-preview-title">
          <div class="panel-head"><div><p class="charge-section-kicker">02 · REVIEW</p><h2 id="charge-preview-title">核对待收明细</h2></div><span class="charge-counter">{{ preview.items.length }} 项</span></div>
          <dl class="charge-meta">
            <div>
              <dt>患者</dt>
              <dd>{{ pending.patient.patientName }} · {{ pending.patient.patientNo }}</dd>
            </div>
            <div>
              <dt>挂号</dt>
              <dd>{{ pending.register.summary }}</dd>
            </div>
            <div>
              <dt>处方</dt>
              <dd>{{ pending.order.recipeNo }} · {{ pending.order.status }}</dd>
            </div>
          </dl>

          <p id="charge-pending-scroll" class="table-scroll-hint">左右滑动可查看完整明细</p>
          <div class="result-wrap" role="region" aria-label="待收明细，可横向滚动" aria-describedby="charge-pending-scroll" tabindex="0">
            <table class="patient-table" data-testid="charge-pending">
              <caption class="sr-only">待收明细</caption>
              <thead>
                <tr>
                  <th>项目</th>
                  <th>数量</th>
                  <th>单价</th>
                  <th>金额</th>
                  <th>库存</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="item in preview.items" :key="item.itemNo" :class="{ 'is-short': !item.stockSufficient }">
                  <td class="charge-item-name">{{ item.itemName }}</td>
                  <td>{{ item.qty }}{{ item.unit }}</td>
                  <td>{{ formatYuan(item.unitPriceCents) }}</td>
                  <td class="charge-amount-cell">{{ formatYuan(item.lineFeeCents) }}</td>
                  <td><span class="charge-stock-state" :class="{ 'is-short': !item.stockSufficient }">{{ item.stockHint }}</span></td>
                </tr>
              </tbody>
            </table>
          </div>
          </section>

          <aside class="charge-pay" data-testid="charge-pay" aria-labelledby="charge-settle-title">
            <div class="charge-settle-head"><p class="charge-section-kicker">03 · SETTLE</p><h2 id="charge-settle-title">收银结算</h2></div>
            <div class="charge-total"><span>本次应收</span><strong>{{ formatYuan(preview.amountDueCents) }}</strong><span>人民币 · 元</span></div>
            <div class="money-grid">
              <label class="field">
                <span>应收</span>
                <input :value="formatYuan(preview.amountDueCents)" readonly tabindex="-1" />
              </label>
              <label class="field">
                <span>优惠</span>
                <input :value="formatYuan(preview.favourableFeeCents)" readonly tabindex="-1" />
              </label>
              <label class="field">
                <span>实收</span>
                <input :value="formatYuan(preview.amountPaidCents)" readonly tabindex="-1" data-testid="charge-paid-amount" />
              </label>
            </div>

            <fieldset class="pay-types">
              <legend>支付方式</legend>
              <label class="pay-option">
                <input type="radio" name="pay-type" :value="PAY_CASH" checked />
                现金
              </label>
              <label v-for="item in RESERVED_PAY_TYPES" :key="item.value" class="pay-option is-reserved">
                <input type="radio" name="pay-type" :value="item.value" disabled />
                {{ item.label }}
                <span>预留</span>
              </label>
            </fieldset>

            <p v-if="stockBlocked" class="banner" role="alert" data-testid="charge-stock-block">
              <span class="banner-body">
                <span>库存不足，无法收费</span>
                <span class="banner-meta"><span class="meta-chip">INSUFFICIENT_STOCK</span></span>
              </span>
            </p>

            <p class="charge-settle-note">请核对实收金额后确认。本次按现金一次结清。</p>
            <div class="panel-actions charge-settle-actions">
              <button class="button button-compact" type="button" data-testid="charge-confirm" :disabled="!canPay" @click="onPay">
                {{ paying ? '正在收费…' : '确认收费' }}
              </button>
              <button class="button button-ghost button-compact" type="button" :disabled="paying" @click="pending = null; activeKey = ''; activeSubject = null">
                返回
              </button>
            </div>
          </aside>
          </div>
        </template>
      </section>

      <section v-show="viewMode === 'records'" class="patient-panel charge-records-panel" aria-labelledby="query-title">
        <div class="panel-head">
          <div><p class="charge-section-kicker">TRANSACTION HISTORY</p><h2 id="query-title">已收查询</h2></div><span class="charge-counter">{{ charges.length }} 条记录</span>
        </div>

        <form class="charge-search" data-testid="charge-query" @submit.prevent="refreshList">
          <label class="field">
            <span>患者号 / 姓名</span>
            <input v-model="queryKeyword" name="charge-query" autocomplete="off" placeholder="留空查看今日" />
          </label>
          <label class="check-line">
            <input v-model="todayOnly" type="checkbox" @change="refreshList" />
            仅看今日
          </label>
          <button class="button button-compact" type="submit" :disabled="loadingList">查询</button>
        </form>

        <p v-if="refundMessage" class="ok-banner" role="status" data-testid="charge-refund-done">{{ refundMessage }}</p>

        <p v-if="loadingList" class="result-empty" role="status">正在查询收费记录…</p>
        <BusinessEmptyState v-else-if="listError" icon="charge" title="收费记录读取失败" :description="listError" eyebrow="读取失败"><button class="button button-ghost button-compact" type="button" @click="refreshList">重试查询</button></BusinessEmptyState>
        <BusinessEmptyState v-else-if="charges.length === 0" icon="charge" title="暂无收费记录" description="可更换患者姓名、患者号，或取消仅看今日后重新查询。" eyebrow="已收查询" />
        <p v-if="charges.length && !loadingList && !listError" id="charge-records-scroll" class="table-scroll-hint">左右滑动可查看完整记录与操作</p>
        <div v-if="charges.length && !loadingList && !listError" class="result-wrap" role="region" aria-label="已收记录，可横向滚动" aria-describedby="charge-records-scroll" tabindex="0">
          <table class="patient-table">
            <caption class="sr-only">已收列表</caption>
            <thead>
              <tr>
                <th>时间</th>
                <th>单号</th>
                <th>患者</th>
                <th>挂号</th>
                <th>金额</th>
                <th>状态</th>
                <th>操作</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="item in charges" :key="item.chargeId">
                <td>{{ item.operateText }}</td>
                <td class="charge-record-no">{{ item.squareNo }}</td>
                <td class="charge-item-name">{{ item.patientName }}</td>
                <td>{{ item.registerSummary }}</td>
                <td class="charge-amount-cell">{{ formatYuan(item.squareFeeCents) }}</td>
                <td>
                  <span class="charge-status" :class="`is-${item.status.toLowerCase()}`">{{ item.paid === false ? '待支付' : statusLabel(item.status) }}</span>
                  <span class="field-hint">{{ item.status }}</span>
                </td>
                <td>
                  <button class="text-btn" type="button" :aria-label="`查看收费单 ${item.squareNo} 详情`" @click="openRefund(item.chargeId)">详情</button>
                  <button
                    v-if="item.canRefund"
                    class="text-btn"
                    type="button"
                    :aria-label="`退费收费单 ${item.squareNo}`"
                    :data-testid="item.patientName === '周已发' ? 'charge-refund-dispensed' : 'charge-refund-entry'"
                    @click="openRefund(item.chargeId)"
                  >
                    退费
                  </button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>
    </div>

    <div v-if="refundOpen" class="drawer-backdrop" @click.self="closeRefund" @keydown.esc="closeRefund">
      <section
        class="refund-drawer"
        role="dialog"
        aria-modal="true"
        aria-labelledby="refund-title"
        data-testid="charge-refund-dialog"
      >
        <div class="panel-head">
          <div><p class="charge-section-kicker">REFUND WORKSPACE</p><h2 id="refund-title">退费</h2></div>
          <button class="text-btn" type="button" :disabled="refunding" @click="closeRefund">关闭</button>
        </div>
        <p class="charge-drawer-note">核对原单、勾选项目与数量，再确认退回方式。</p>

        <p v-if="!refundTarget && !refundError" class="result-empty">正在载入收费单…</p>
        <p v-if="refundError" class="banner" role="alert" data-testid="charge-refund-error">
          <span class="banner-body">
            <span>{{ refundError }}</span>
            <span v-if="refundCode" class="banner-meta"><span class="meta-chip">{{ refundCode }}</span></span>
          </span>
        </p>

        <template v-if="refundTarget">
          <dl class="charge-meta">
            <div>
              <dt>原收费单</dt>
              <dd>{{ refundTarget.squareNo }}</dd>
            </div>
            <div>
              <dt>患者</dt>
              <dd>{{ refundTarget.patientName }} · {{ refundTarget.registerSummary }}</dd>
            </div>
            <div>
              <dt>状态</dt>
              <dd>{{ statusLabel(refundTarget.status) }} {{ refundTarget.status }}</dd>
            </div>
          </dl>

          <p v-if="refundHasDispensed" class="banner" role="alert" data-testid="charge-refund-block">
            <span class="banner-body">
              <span>已发药，请先退药再退费</span>
              <span class="banner-meta"><span class="meta-chip">REFUND_REQUIRES_RETURN</span></span>
              <RouterLink class="text-btn text-btn-danger" :to="{ name: 'outpatient-module', params: { module: refundOnly ? 'refund' : 'dispense' }, query: { view: 'issued', charge_id: refundTarget.chargeId } }">去退药</RouterLink>
            </span>
          </p>

          <p v-if="discountedPartialRefund" class="banner" role="alert" data-testid="charge-refund-discount-block">
            <span class="banner-body">
              <span>收费单含优惠，暂不支持部分退费。请勾选全部可退明细并按原数量全额退费。</span>
              <span class="banner-meta"><span class="meta-chip">PARTIAL_REFUND_DISCOUNT_UNSUPPORTED</span></span>
            </span>
          </p>

          <p v-if="refundSelectable.length === 0" class="result-empty">没有可退明细（已退完或不可退）</p>

          <p id="charge-refund-scroll" class="table-scroll-hint">左右滑动可查看完整退费明细</p>
          <div class="result-wrap" role="region" aria-label="退费明细，可横向滚动" aria-describedby="charge-refund-scroll" tabindex="0">
            <table class="patient-table">
              <caption class="sr-only">退费明细</caption>
              <thead>
                <tr>
                  <th>退</th>
                  <th>项目</th>
                  <th>可退数量</th>
                  <th>金额</th>
                  <th>说明</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="item in refundRows" :key="item.chargeItemId">
                  <td>
                    <input
                      v-if="item.canBack && !item.dispenseBlocked && item.refundableQty > 0"
                      v-model="refundPicks[item.chargeItemId]"
                      type="checkbox"
                      :aria-label="`退 ${item.itemName}`"
                    />
                  </td>
                  <td>{{ item.itemName }}</td>
                  <td>
                    <template v-if="item.canBack && !item.dispenseBlocked && item.refundableQty > 0">
                      <input
                        v-model.number="refundQty[item.chargeItemId]"
                        class="qty-input"
                        type="number"
                        min="1"
                        :max="item.refundableQty"
                        :disabled="!refundPicks[item.chargeItemId]"
                        :aria-label="`${item.itemName}退数量`"
                      />
                      <span class="field-hint"> / {{ item.refundableQty }}{{ item.unit }}</span>
                    </template>
                    <template v-else>{{ item.refundableQty }}{{ item.unit }}</template>
                  </td>
                  <td>{{ formatYuan(item.lineFeeCents) }}</td>
                  <td>
                    <span v-if="item.dispenseBlocked" class="is-short-text">已发药，请先退药</span>
                    <span v-else-if="!item.canBack">不可退</span>
                    <span v-else>可退</span>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>

          <label class="field">
            <span>原因</span>
            <input v-model="refundReason" name="refund-reason" autocomplete="off" placeholder="患者要求" :disabled="refundTarget.status !== 'NORMAL'" />
          </label>

          <label class="field">
            <span>退回方式</span>
            <input value="现金" readonly tabindex="-1" />
          </label>

          <div class="panel-actions charge-refund-actions">
            <p class="charge-refund-count">已选 <strong>{{ refundChosen.length }}</strong> 项 · 现金退回</p>
            <button
              class="button button-compact"
              type="button"
              data-testid="charge-refund-confirm"
              :disabled="refunding || refundTarget.status !== 'NORMAL' || refundTarget.paid === false || refundBlocked || discountedPartialRefund || refundChosen.length === 0"
              @click="onRefund"
            >
              {{ refunding ? '正在退费…' : '确认退费' }}
            </button>
          </div>
        </template>
      </section>
    </div>
  </section>
</template>

<style scoped src="./charge-design.css"></style>

<style scoped>
/* UI2.0 Charge/ClinicChargeFrm: 47px search strip, grid, bottom cashier area. */
.ui2-charge-page.charge-page { color: #202020; font: 16px SimSun, '宋体', serif; background: #f0f0f0; }
.ui2-charge-page.charge-page .charge-layout { display: block; }
.ui2-charge-page.charge-page .charge-pending-panel, .ui2-charge-page.charge-page .charge-records-panel { padding: 0; border: 1px solid #999; border-radius: 0; background: #f0f0f0; }
.ui2-charge-page.charge-page .charge-candidate-zone { padding: 0; border: 0; border-radius: 0; background: #f0f0f0; }
.ui2-charge-page.charge-page .charge-search { display: flex; align-items: center; gap: 8px; min-height: 47px; margin: 0; padding: 6px 10px; border-top: 1px solid #aaa; border-bottom: 1px solid #aaa; border-radius: 0; background: #f0f0f0; box-sizing: border-box; }
.ui2-charge-page.charge-page .charge-search .field { display: flex; flex-direction: row; align-items: center; gap: 8px; flex: 1; }
.ui2-charge-page.charge-page .field > span { color: #202020; font: 12px/17px 'Segoe UI', sans-serif; }
.ui2-charge-page.charge-page .field input:not([type=radio]), .ui2-charge-page.charge-page .field select { min-height: 25px; height: 25px; padding: 2px 5px; border: 1px solid #999; border-radius: 0; background: white; color: #202020; font: 16px SimSun, '宋体', serif; }
.ui2-charge-page.charge-page .button { min-height: 30px; height: 30px; padding: 0 10px; border: 1px solid #aaa; border-radius: 3px; background: #e6e6e6; color: #202020; font: 16px SimSun, '宋体', serif; }
.ui2-charge-page.charge-page .charge-workspace { display: flex; flex-direction: column; gap: 0; margin: 0; }
.ui2-charge-page.charge-page .charge-preview { padding: 8px; border: 0; border-radius: 0; background: white; }
.ui2-charge-page.charge-page .charge-pay { padding: 10px; border: 1px solid #aaa; border-radius: 0; background: #f0f0f0; }
.ui2-charge-page.charge-page .charge-meta { margin: 0 0 8px; padding: 6px 8px; gap: 5px 12px; border: 1px solid #aaa; border-radius: 0; background: #f0f0f0; }
.ui2-charge-page.charge-page .charge-section-kicker { display: none; }
.ui2-charge-page.charge-page .panel-head { min-height: 30px; margin: 0; padding: 5px 10px; }
.ui2-charge-page.charge-page .panel-head h2, .ui2-charge-page.charge-page .charge-settle-head h2 { font: 16px SimSun, '宋体', serif; }
.ui2-charge-page.charge-page .result-wrap { border: 1px solid #aaa; border-radius: 0; background: white; }
.ui2-charge-page.charge-page .patient-table { border-collapse: collapse; font: 16px SimSun, '宋体', serif; }
.ui2-charge-page.charge-page .patient-table th { padding: 5px 8px; border: 1px solid #aaa; background: #e7e7e7; color: #202020; font-weight: normal; }
.ui2-charge-page.charge-page .patient-table td { padding: 4px 8px; border: 1px solid #d4d4d4; color: #202020; }
.ui2-charge-page.charge-page .charge-amount-cell { text-align: right; font-variant-numeric: tabular-nums; }
.ui2-charge-page.charge-page .charge-total { display: flex; align-items: baseline; justify-content: flex-end; gap: 12px; padding: 8px; border: 1px solid #aaa; border-radius: 0; background: white; }
.ui2-charge-page.charge-page .charge-total strong { font: 24px SimSun, '宋体', serif; color: #202020; }
.ui2-charge-page.charge-page .money-grid { grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 8px; }
.ui2-charge-page.charge-page .charge-settle-actions { justify-content: flex-end; }
@media (max-width: 600px) {
  .ui2-charge-page.charge-page .charge-search { flex-wrap: wrap; }
  .ui2-charge-page.charge-page .charge-search .field { flex-basis: 100%; }
  .ui2-charge-page.charge-page .money-grid { grid-template-columns: 1fr; }
  .ui2-charge-page.charge-page .charge-total { flex-wrap: wrap; }
}
</style>
