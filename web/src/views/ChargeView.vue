<script setup>
// 2026-10-12 P1-H05-CHARGE-UI 门诊收费：待收、现金、查询、退费。侧栏仍由 AppShell 提供。
// 2026-10-12 VITE_USE_MOCK=false 时走 charge-preview / charges / pay / refund。
// 2026-10-12 P1-H05-DISPENSE-UI 发药页「去退费」带 charge_id 时打开退费。
import { computed, onMounted, reactive, ref } from 'vue'
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

const route = useRoute()
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

const refundOpen = ref(false)
const refundTarget = ref(null)
const refundReason = ref('')
const refundPicks = reactive({})
const refundQty = reactive({})
const refunding = ref(false)
const refundError = ref('')
const refundCode = ref('')
const refundMessage = ref('')

const preview = computed(() => (pending.value && pending.value.preview) || null)
const scene = computed(() => (pending.value && pending.value.scene) || '')
const stockBlocked = computed(() => Boolean(preview.value && preview.value.stockBlocked))
const canPay = computed(() => Boolean(preview.value) && !stockBlocked.value && !paying.value)

const refundRows = computed(() => (refundTarget.value && refundTarget.value.items) || [])
const refundHasDispensed = computed(() => refundRows.value.some((item) => item.dispenseBlocked))
const refundSelectable = computed(() => refundRows.value.filter((item) => item.canBack && !item.dispenseBlocked && item.refundableQty > 0))
const refundChosen = computed(() => refundSelectable.value.filter((item) => refundPicks[item.chargeItemId]))
const refundBlocked = computed(() => refundHasDispensed.value && refundSelectable.value.length === 0)

function clearError() {
  errorMessage.value = ''
  errorCode.value = ''
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

async function refreshSubjects() {
  subjects.value = (await lookupChargeSubjects(session.accessToken, keyword.value)).items
  if (!activeKey.value && subjects.value.length > 0) {
    const preferred = subjects.value.find((item) => item.scene === 'PENDING' || item.scene === 'STOCK') || subjects.value[0]
    await openSubject(preferred)
  }
}

async function refreshList() {
  loadingList.value = true
  try {
    const found = await listCharges(session.accessToken, {
      keyword: queryKeyword.value,
      todayOnly: todayOnly.value,
    })
    charges.value = found.items
  } catch (error) {
    charges.value = []
    applyError(error, '查询失败，请稍后重试')
  } finally {
    loadingList.value = false
  }
}

async function openSubject(subject, options = {}) {
  if (!subject || loadingPending.value) return
  activeSubject.value = subject
  activeKey.value = subject.key || subject.patientId
  loadingPending.value = true
  clearError()
  if (!options.keepNotice) {
    payMessage.value = ''
    paidSquareNo.value = ''
  }
  try {
    pending.value = await loadPendingCharge(session.accessToken, subject)
  } catch (error) {
    pending.value = null
    applyError(error, '待收载入失败')
  } finally {
    loadingPending.value = false
  }
}

async function onSearchSubjects() {
  clearError()
  try {
    await refreshSubjects()
  } catch (error) {
    applyError(error, '检索失败')
  }
}

async function onPay() {
  if (!canPay.value || !preview.value) return
  paying.value = true
  clearError()
  payMessage.value = ''
  try {
    const result = await payCharge(session.accessToken, {
      orderId: preview.value.orderId,
      registerId: pending.value.register && pending.value.register.registerId,
      payType: PAY_CASH,
      amount: preview.value.squareFee,
      amountCents: preview.value.amountDueCents,
    })
    paidSquareNo.value = result.charge.squareNo
    await refreshSubjects()
    const next = subjects.value.find((item) => item.key === activeKey.value) || activeSubject.value
    await openSubject(next, { keepNotice: true })
    payMessage.value = '收费成功，可去发药。'
    await refreshList()
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
  for (const key of Object.keys(refundPicks)) delete refundPicks[key]
  for (const key of Object.keys(refundQty)) delete refundQty[key]
  for (const item of charge.items) {
    if (item.canBack && !item.dispenseBlocked && item.refundableQty > 0) {
      refundPicks[item.chargeItemId] = false
      refundQty[item.chargeItemId] = item.refundableQty
    }
  }
}

async function openRefund(chargeId) {
  refundMessage.value = ''
  refundOpen.value = true
  refundTarget.value = null
  refundError.value = ''
  refundCode.value = ''
  try {
    resetRefundForm(await fetchCharge(session.accessToken, chargeId))
  } catch (error) {
    refundCode.value = error instanceof ApiError ? error.code : ''
    refundError.value = error instanceof ApiError ? error.message : '未找到收费单'
  }
}

function closeRefund() {
  if (refunding.value) return
  refundOpen.value = false
}

async function onRefund() {
  if (!refundTarget.value || refunding.value || refundBlocked.value) return
  const items = refundChosen.value.map((item) => ({
    chargeItemId: item.chargeItemId,
    refundQty: Number(refundQty[item.chargeItemId]),
  }))
  if (items.length === 0) {
    refundCode.value = 'INVALID_AMOUNT'
    refundError.value = '数量或金额不合法'
    return
  }
  refunding.value = true
  refundError.value = ''
  refundCode.value = ''
  try {
    const result = await refundCharge(session.accessToken, refundTarget.value.chargeId, {
      reason: refundReason.value,
      payType: PAY_CASH,
      items,
    })
    refundOpen.value = false
    if (result.fullRefund || !result.newSquare) {
      refundMessage.value = '已全额退费。原单已作废，未生成新收费单。'
    } else {
      refundMessage.value = `已部分退费。原单已作废，未退部分已生成新收费单 ${result.newSquare.squareNo}（${formatYuan(result.newSquare.squareFeeCents)}）。`
    }
    await refreshList()
    if (activeSubject.value) await openSubject(activeSubject.value, { keepNotice: true })
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

onMounted(async () => {
  try {
    await refreshSubjects()
    await refreshList()
    const chargeId = queryChargeId()
    if (chargeId) await openRefund(chargeId)
  } catch (error) {
    applyError(error, '收费页载入失败')
  }
})
</script>

<template>
  <section class="charge-page" aria-labelledby="charge-title">
    <p class="crumb">
      <RouterLink :to="{ name: 'home' }">首页</RouterLink>
      <span aria-hidden="true"> / </span>
      <RouterLink :to="{ name: 'workbench' }">门诊业务</RouterLink>
      <span aria-hidden="true"> / </span>
      <span>收费</span>
    </p>

    <header class="dash-head">
      <h1 id="charge-title">门诊收费</h1>
      <p class="dash-flag">从已提交处方载入待收。支付方式为现金，实收等于应收。电子方式只作预留。</p>
    </header>

    <div class="charge-layout">
      <section class="patient-panel" aria-labelledby="pending-title">
        <div class="panel-head">
          <h2 id="pending-title">待收</h2>
        </div>

        <form class="charge-search" data-testid="charge-search" @submit.prevent="onSearchSubjects">
          <label class="field">
            <span>患者 / 挂号 / 处方</span>
            <input v-model="keyword" name="charge-keyword" autocomplete="off" placeholder="姓名、患者号、挂号号或处方号" />
          </label>
          <button class="button button-compact" type="submit">检索</button>
        </form>

        <div class="demo-chips" aria-label="演示患者">
          <button
            v-for="item in subjects"
            :key="item.patientId"
            class="demo-chip"
            :class="{ 'is-active': item.key === activeKey }"
            type="button"
            :data-testid="`charge-chip-${item.scene.toLowerCase()}`"
            @click="openSubject(item)"
          >
            {{ item.patientName }}
            <span>{{ item.sceneLabel }}</span>
          </button>
        </div>

        <p v-if="errorMessage" class="banner" role="alert" data-testid="charge-error">
          <span class="banner-body">
            <span>{{ errorMessage }}</span>
            <span v-if="errorCode" class="banner-meta">
              <span class="meta-chip">{{ errorCode }}</span>
            </span>
          </span>
        </p>

        <div v-if="payMessage" class="ok-banner" role="status" data-testid="charge-paid">
          <p class="ok-title">{{ payMessage }}</p>
          <p v-if="paidSquareNo">收费单号 <strong>{{ paidSquareNo }}</strong></p>
          <RouterLink class="button button-compact" :to="{ name: 'outpatient-module', params: { module: 'dispense' } }">去发药</RouterLink>
        </div>

        <div v-if="loadingPending" class="result-empty">正在载入待收…</div>

        <div v-else-if="!pending" class="charge-empty" data-testid="charge-empty-register">
          <h3>请先挂号</h3>
          <p>未挂号/无有效就诊，不能收费。</p>
          <RouterLink class="button button-ghost button-compact" :to="{ name: 'outpatient-module', params: { module: 'register' } }">去挂号</RouterLink>
        </div>

        <div v-else-if="scene === 'NO_REGISTER'" class="charge-empty" data-testid="charge-empty-register">
          <h3>请先挂号</h3>
          <p>{{ pending.patient.patientName }}（{{ pending.patient.patientNo }}）还没有有效挂号。</p>
          <p class="field-hint">未挂号/无有效就诊，不能收费。</p>
          <RouterLink class="button button-ghost button-compact" :to="{ name: 'outpatient-module', params: { module: 'register' } }">去挂号</RouterLink>
        </div>

        <div v-else-if="scene === 'NO_COMMIT'" class="charge-empty" data-testid="charge-empty-order">
          <h3>请先提交处方</h3>
          <p>{{ pending.patient.patientName }} 已挂号（{{ pending.register.summary }}），还没有已提交处方。</p>
          <RouterLink class="button button-ghost button-compact" :to="{ name: 'outpatient-module', params: { module: 'order' } }">去开单</RouterLink>
        </div>

        <div v-else-if="scene === 'CHARGED' || scene === 'DISPENSED'" class="charge-empty" data-testid="charge-already">
          <h3>该处方已收费</h3>
          <p>{{ pending.patient.patientName }} · {{ pending.squareNo || '已生成收费单' }}。退费请在右侧已收查询中办理。</p>
          <p v-if="scene === 'DISPENSED'" class="field-hint">已发药的明细须先退药，才能退费。</p>
        </div>

        <template v-else-if="preview">
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

          <div class="result-wrap">
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
                  <td>{{ item.itemName }}</td>
                  <td>{{ item.qty }}{{ item.unit }}</td>
                  <td>{{ formatYuan(item.unitPriceCents) }}</td>
                  <td>{{ formatYuan(item.lineFeeCents) }}</td>
                  <td>{{ item.stockHint }}</td>
                </tr>
              </tbody>
            </table>
          </div>

          <div class="charge-pay" data-testid="charge-pay">
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

            <div class="panel-actions">
              <button class="button button-compact" type="button" data-testid="charge-confirm" :disabled="!canPay" @click="onPay">
                {{ paying ? '正在收费…' : '确认收费' }}
              </button>
              <button class="button button-ghost button-compact" type="button" :disabled="paying" @click="pending = null; activeKey = ''; activeSubject = null">
                返回
              </button>
            </div>
          </div>
        </template>
      </section>

      <section class="patient-panel" aria-labelledby="query-title">
        <div class="panel-head">
          <h2 id="query-title">已收查询</h2>
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

        <div class="result-wrap">
          <p v-if="charges.length === 0" class="result-empty">暂无收费记录</p>
          <table v-else class="patient-table">
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
                <td>{{ item.squareNo }}</td>
                <td>{{ item.patientName }}</td>
                <td>{{ item.registerSummary }}</td>
                <td>{{ formatYuan(item.squareFeeCents) }}</td>
                <td>
                  <span class="charge-status" :class="`is-${item.status.toLowerCase()}`">{{ item.paid === false ? '待支付' : statusLabel(item.status) }}</span>
                  <span class="field-hint">{{ item.status }}</span>
                </td>
                <td>
                  <button class="text-btn" type="button" @click="openRefund(item.chargeId)">详情</button>
                  <button
                    v-if="item.canRefund"
                    class="text-btn"
                    type="button"
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

    <div v-if="refundOpen" class="drawer-backdrop" @click.self="closeRefund">
      <section
        class="refund-drawer"
        role="dialog"
        aria-modal="true"
        aria-labelledby="refund-title"
        data-testid="charge-refund-dialog"
      >
        <div class="panel-head">
          <h2 id="refund-title">退费</h2>
          <button class="text-btn" type="button" :disabled="refunding" @click="closeRefund">关闭</button>
        </div>

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
              <RouterLink class="text-btn text-btn-danger" :to="{ name: 'outpatient-module', params: { module: 'refund' } }">去退药</RouterLink>
            </span>
          </p>

          <p v-if="refundSelectable.length === 0" class="result-empty">没有可退明细（已退完或不可退）</p>

          <div class="result-wrap">
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

          <div class="panel-actions">
            <button
              class="button button-compact"
              type="button"
              data-testid="charge-refund-confirm"
              :disabled="refunding || refundTarget.status !== 'NORMAL' || refundTarget.paid === false || refundBlocked || refundChosen.length === 0"
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
