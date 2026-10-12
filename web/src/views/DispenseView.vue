<script setup>
// 2026-10-12 P1-H05-DISPENSE-UI 门诊发药 Mock：待发、部分发、退药。不请求 Go。
import { computed, onMounted, ref } from 'vue'
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

const mockMode = useMock()
const chargeTo = { name: 'outpatient-module', params: { module: 'charge' } }

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

const issued = ref([])
const issuedPatient = ref('')
const issuedNo = ref('')
const issuedToday = ref(true)
const issuedLoading = ref(false)
const detailId = ref('')

const drawerOpen = ref(false)
const returnSlip = ref(null)
const returnLines = ref([])
const returnReason = ref('')
const returning = ref(false)
const returnResult = ref(null)
const returnError = ref('')

const busy = computed(() => loading.value || dispensing.value || returning.value)
const operatorName = computed(() => {
  const name = session.user && session.user.displayName
  return typeof name === 'string' && name.trim() ? name.trim() : '当前用户'
})
const singlePatient = computed(() => {
  const ids = new Set(rows.value.map((row) => row.patientId))
  return ids.size === 1 ? rows.value[0] : null
})
const detail = computed(() => issued.value.find((item) => item.dispenseId === detailId.value) || null)

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

async function reloadPending() {
  if (loadMode.value === 'today') {
    applyPending(await listTodayPending(session.accessToken), 'today')
    return
  }
  if (loadMode.value === 'search') {
    applyPending(await searchPending(session.accessToken, query.value), 'search')
  }
}

async function loadIssued() {
  issuedLoading.value = true
  try {
    const listed = await listDispenses(session.accessToken, {
      today: issuedToday.value,
      patient: issuedPatient.value,
      dispenseNo: issuedNo.value,
    })
    issued.value = listed.items
    if (detailId.value && !listed.items.some((item) => item.dispenseId === detailId.value)) detailId.value = ''
  } catch (error) {
    issued.value = []
    if (!errorMessage.value) {
      errorMessage.value = error instanceof ApiError ? error.message : '发药记录读取失败'
      errorCode.value = error instanceof ApiError ? error.code : ''
    }
  } finally {
    issuedLoading.value = false
  }
}

async function onSearch() {
  if (busy.value) return
  clearBanner()
  loading.value = true
  try {
    applyPending(await searchPending(session.accessToken, query.value), 'search')
  } catch (error) {
    showError(error)
  } finally {
    loading.value = false
  }
}

async function onToday() {
  if (busy.value) return
  clearBanner()
  loading.value = true
  try {
    applyPending(await listTodayPending(session.accessToken), 'today')
  } catch (error) {
    showError(error)
  } finally {
    loading.value = false
  }
}

async function onRefreshStock() {
  if (!searched.value || busy.value) return
  clearBanner()
  loading.value = true
  try {
    await reloadPending()
    infoMessage.value = '已刷新库存'
  } catch (error) {
    showError(error)
  } finally {
    loading.value = false
  }
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
  if (dispensing.value) return
  clearBanner()
  dispensing.value = true
  try {
    const result = await confirmDispense(session.accessToken, {
      items: payloadFrom(list),
      idempotencyKey: crypto.randomUUID(),
    })
    const slips = Array.isArray(result.dispenses) && result.dispenses.length > 0 ? result.dispenses : [result]
    const nos = slips.map((item) => item.dispenseNo).join('、')
    infoMessage.value = `已发药 ${nos}。库存已按行扣减。`
    await reloadPending()
    await loadIssued()
  } catch (error) {
    errorMessage.value = error instanceof ApiError ? error.message : '发药失败，请稍后重试'
    errorCode.value = error instanceof ApiError ? error.code : ''
  } finally {
    dispensing.value = false
  }
}

function onDispenseSelected() {
  submit(rows.value.filter((row) => row.selected))
}

function onDispenseAll() {
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
    params: { module: 'charge' },
    query: { charge_id: chargeId },
  }
}

async function openReturn(slip) {
  returnError.value = ''
  returnResult.value = null
  returnReason.value = ''
  try {
    const fresh = await fetchDispense(session.accessToken, slip.dispenseId)
    returnSlip.value = fresh
    returnLines.value = fresh.items
      .filter((item) => item.returnableQty > 0)
      .map((item) => ({
        ...item,
        selected: true,
        returnQty: item.returnableQty,
      }))
    drawerOpen.value = true
  } catch (error) {
    errorMessage.value = error instanceof ApiError ? error.message : '无法打开退药'
    errorCode.value = error instanceof ApiError ? error.code : ''
  }
}

function closeDrawer() {
  if (returning.value) return
  drawerOpen.value = false
}

async function onReturn() {
  if (returning.value || !returnSlip.value) return
  returnError.value = ''
  returning.value = true
  try {
    const result = await returnDispense(session.accessToken, {
      dispenseId: returnSlip.value.dispenseId,
      reason: returnReason.value,
      idempotencyKey: crypto.randomUUID(),
      items: returnLines.value
        .filter((item) => item.selected && Number(item.returnQty) > 0)
        .map((item) => ({
          dispenseItemId: item.dispenseItemId,
          chargeItemId: item.chargeItemId,
          returnQty: Number(item.returnQty),
          selected: true,
        })),
    })
    returnResult.value = result
    infoMessage.value = `${result.stockNote}。可继续退费。`
    if (searched.value) await reloadPending()
    await loadIssued()
    if (detailId.value === result.dispenseId) {
      const fresh = issued.value.find((item) => item.dispenseId === result.dispenseId)
      if (!fresh) detailId.value = ''
    }
  } catch (error) {
    returnError.value = error instanceof ApiError ? error.message : '退药失败，请稍后重试'
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

onMounted(async () => {
  if (!mockMode) return
  searchInput.value?.focus()
  await loadIssued()
})
</script>

<template>
  <section class="dispense-page" aria-labelledby="dispense-title">
    <p class="crumb">
      <RouterLink :to="{ name: 'home' }">首页</RouterLink>
      <span aria-hidden="true"> / </span>
      <RouterLink :to="{ name: 'workbench' }">门诊业务</RouterLink>
      <span aria-hidden="true"> / </span>
      <span>发药</span>
    </p>

    <header class="dash-head">
      <h1 id="dispense-title">门诊发药</h1>
      <p class="dash-flag">只发已收费药品。可改小数量做部分发。已发药品退药后才能去退费。</p>
    </header>

    <div v-if="!mockMode" class="empty-panel" data-testid="dispense-offline">
      <h2>发药接口尚未接入</h2>
      <p class="empty-lead">请使用演示模式</p>
      <p class="muted">把 VITE_USE_MOCK 保持为 true 后重新打开页面。演示账号 system / 1。</p>
    </div>

    <template v-else>
      <form class="patient-panel" @submit.prevent="onSearch">
        <div class="panel-head">
          <h2>检索待发</h2>
          <button class="text-btn" type="button" :disabled="busy" @click="onReset">重置演示数据</button>
        </div>
        <div class="charge-search">
          <label class="field">
            <span>患者 / 挂号 / 收费单 / 处方号</span>
            <input
              ref="searchInput"
              v-model="query"
              data-testid="dispense-search"
              placeholder="例如王全发、GH-FULL-001、SQ-PART-001、RX-SHORT-001"
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
        <div class="demo-chips" aria-label="演示检索">
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

      <p v-if="errorMessage" class="banner" role="alert" data-testid="dispense-error" :data-code="errorCode">{{ errorMessage }}</p>
      <p v-if="infoMessage" class="ok-banner" role="status" data-testid="dispense-ok">{{ infoMessage }}</p>

      <div v-if="!searched" class="empty-panel dispense-empty" data-testid="dispense-prompt">
        <h2>请选择已收费患者或处方</h2>
        <p class="muted">检索后只载入已收费、尚未发完的药品。</p>
      </div>

      <div v-else-if="rows.length === 0" class="empty-panel dispense-empty" data-testid="dispense-empty">
        <h2>{{ emptyKind === 'today' ? '今日无待发' : '暂无待发药品' }}</h2>
        <p v-if="emptyKind === 'done'" class="muted">已发完的药品不在待发列表，可在下方退药。</p>
        <p v-else-if="emptyKind === 'today'" class="muted">今天没有已收费且未发完的药品。</p>
        <p v-else class="muted">未收费或已发完的处方不会出现在这里。</p>
        <RouterLink class="button button-ghost button-compact empty-back" :to="chargeTo">去收费查询</RouterLink>
      </div>

      <form v-else class="patient-panel" data-testid="dispense-lines" @submit.prevent="onDispenseSelected">
        <div class="panel-head">
          <h2>待发明细</h2>
          <p class="field-hint">执行科室 {{ PHARMACY.deptName }} · 操作员 {{ operatorName }}</p>
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
        <p v-else class="field-hint">今日待发含多位患者。确认发药只提交勾选的行，并按收费单分别生成发药单。</p>

        <div class="line-scroll">
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
                  <input v-model="row.selected" type="checkbox" :aria-label="`选择 ${row.itemName}`" :data-testid="`dispense-check-${index}`" />
                </td>
                <td>{{ row.patientName }}</td>
                <td>{{ row.itemName }}</td>
                <td>{{ row.spec || '—' }}</td>
                <td>{{ row.units || '—' }}</td>
                <td>{{ formatQty(row.chargedQty) }}</td>
                <td>{{ formatQty(row.dispensedQty) }}</td>
                <td>{{ formatQty(row.availableQty) }}</td>
                <td class="col-narrow">
                  <input
                    v-model="row.dispenseQty"
                    type="number"
                    min="1"
                    :max="row.availableQty"
                    step="1"
                    :aria-label="`${row.itemName} 本次发药数量`"
                    :data-testid="`dispense-qty-${index}`"
                  />
                  <span v-if="Number(row.dispenseQty) > 0 && Number(row.dispenseQty) < row.availableQty" class="field-hint">部分发</span>
                </td>
                <td :class="{ 'stock-bad': rowShort(row) }">{{ rowHint(row) }}</td>
                <td>
                  <span class="ticket-status" :data-status="row.lineDispenseStatus">{{ lineStatusLabel(row.lineDispenseStatus) }}</span>
                </td>
                <td>{{ row.orderNo }} · {{ row.squareNo }}</td>
              </tr>
            </tbody>
          </table>
        </div>

        <p class="field-hint">全部发药会把每一行改成可发数量再提交。某一行库存不够时，整单不扣库存。</p>
        <div class="panel-actions">
          <button class="button button-compact" type="submit" data-testid="dispense-confirm" :disabled="busy">
            {{ dispensing ? '正在发药…' : '确认发药' }}
          </button>
          <button class="button button-ghost button-compact" type="button" data-testid="dispense-all" :disabled="busy" @click="onDispenseAll">
            全部发药
          </button>
        </div>
      </form>

      <section class="patient-panel" aria-labelledby="issued-title">
        <div class="panel-head">
          <h2 id="issued-title">已发查询</h2>
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
        <p v-else-if="issued.length === 0" class="result-empty" data-testid="issued-empty">暂无发药记录</p>
        <div v-else class="line-scroll">
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
              <tr v-for="item in issued" :key="item.dispenseId">
                <td>{{ formatOperateAt(item.operateAt) }}</td>
                <td>{{ item.dispenseNo }}</td>
                <td>{{ item.patientName }}</td>
                <td>{{ item.registerNo }}</td>
                <td>{{ item.squareNo }}</td>
                <td>{{ item.summary }}</td>
                <td>
                  <span class="ticket-status" :data-status="item.lineDispenseStatus">{{ lineStatusLabel(item.lineDispenseStatus) }}</span>
                </td>
                <td class="ticket-actions">
                  <button class="text-btn" type="button" @click="detailId = detailId === item.dispenseId ? '' : item.dispenseId">详情</button>
                  <button
                    v-if="item.items.some((line) => line.returnableQty > 0)"
                    class="text-btn"
                    type="button"
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
          <h3>{{ detail.dispenseNo }} · {{ detail.patientName }}</h3>
          <p class="field-hint">{{ detail.orderNo }} · {{ detail.squareNo }} · {{ PHARMACY.deptName }}</p>
          <ul>
            <li v-for="item in detail.items" :key="item.dispenseItemId">
              {{ item.itemName }} · 已发 {{ formatQty(item.dispenseQty) }} {{ item.units }} · 已退 {{ formatQty(item.returnedQty) }} · 可退 {{ formatQty(item.returnableQty) }}
            </li>
          </ul>
        </div>
      </section>

      <div v-if="drawerOpen" class="drawer-backdrop" @click.self="closeDrawer">
        <aside class="refund-drawer" role="dialog" aria-modal="true" aria-labelledby="return-title">
          <div class="panel-head">
            <h2 id="return-title">退药</h2>
            <button class="text-btn" type="button" @click="closeDrawer">关闭</button>
          </div>
          <p v-if="returnSlip" class="field-hint">{{ returnSlip.dispenseNo }} · {{ returnSlip.patientName }} · {{ returnSlip.squareNo }}</p>

          <p v-if="returnLines.length === 0" class="result-empty">没有可退明细（未发或已退完）</p>
          <ul v-else class="return-list">
            <li v-for="(item, index) in returnLines" :key="item.dispenseItemId">
              <label class="checkin-demo">
                <input v-model="item.selected" type="checkbox" :aria-label="`退 ${item.itemName}`" />
                {{ item.itemName }}
              </label>
              <p>已发 {{ formatQty(item.dispenseQty) }} · 已退 {{ formatQty(item.returnedQty) }} · 可退 {{ formatQty(item.returnableQty) }} {{ item.units }}</p>
              <label class="field">
                <span>退数量</span>
                <input
                  v-model="item.returnQty"
                  type="number"
                  min="1"
                  :max="item.returnableQty"
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
          <div v-else class="panel-actions">
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
    </template>
  </section>
</template>
