<script setup>
// 2026-10-13 P1-H05-MASTER-UI 药品库存：只读当前量，弹窗调整并记 ADJUST 流水。结果不能为负。
import { computed, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { ApiError } from '../../api/errors.js'
import { session } from '../../auth/session.js'
import { adjustInventory, fetchInventory, fetchLedger } from '../../master/api.js'
import { bizLabel, formatQty, formatTime, statusLabel } from '../../master/labels.js'

const rows = ref([])
const ledger = ref([])
const loading = ref(false)
const saving = ref(false)
const q = ref('')
const selectedId = ref('')
const adjusting = ref(null)
const notice = ref('')
const noticeError = ref(false)
const errorCode = ref('')
const form = reactive({ qtyDelta: '', remark: '' })

const selected = computed(() => rows.value.find((item) => item.skuId === selectedId.value) || null)
const preview = computed(() => {
  if (!adjusting.value || form.qtyDelta === '' || form.qtyDelta == null) return null
  const delta = Number(form.qtyDelta)
  if (!Number.isFinite(delta)) return null
  return Math.round((adjusting.value.qtyOnHand + delta + Number.EPSILON) * 1000) / 1000
})
const canAdjust = computed(() => {
  const delta = Number(form.qtyDelta)
  return preview.value != null && Number.isFinite(delta) && delta !== 0 && preview.value >= 0
})

function show(message, isError = false, code = '') {
  notice.value = message
  noticeError.value = isError
  errorCode.value = code
}

async function load() {
  loading.value = true
  try {
    const listed = await fetchInventory(session.accessToken, { q: q.value })
    rows.value = listed.items
    if (selectedId.value && !rows.value.some((item) => item.skuId === selectedId.value)) {
      selectedId.value = ''
      ledger.value = []
    } else if (selectedId.value) {
      await loadLedger(selectedId.value)
    }
  } catch (error) {
    show(error instanceof ApiError ? error.message : '无法读取库存', true, error instanceof ApiError ? error.code : '')
  } finally {
    loading.value = false
  }
}

async function loadLedger(skuId) {
  const listed = await fetchLedger(session.accessToken, skuId)
  ledger.value = listed.items
}

async function selectRow(row) {
  selectedId.value = row.skuId
  try {
    await loadLedger(row.skuId)
  } catch (error) {
    ledger.value = []
    show(error instanceof ApiError ? error.message : '无法读取流水', true, error instanceof ApiError ? error.code : '')
  }
}

function openAdjust(row) {
  adjusting.value = row
  form.qtyDelta = ''
  form.remark = ''
  errorCode.value = ''
}

function closeAdjust() {
  adjusting.value = null
}

async function onAdjust() {
  if (saving.value || !adjusting.value) return
  saving.value = true
  const skuId = adjusting.value.skuId
  try {
    await adjustInventory(session.accessToken, skuId, {
      qtyDelta: form.qtyDelta,
      remark: form.remark,
      operatorName: session.user && session.user.displayName,
    })
    closeAdjust()
    show('已调整库存', false)
    await load()
    selectedId.value = skuId
    await loadLedger(skuId)
  } catch (error) {
    show(error instanceof ApiError ? error.message : '调整失败', true, error instanceof ApiError ? error.code : '')
  } finally {
    saving.value = false
  }
}

function onKeydown(event) {
  if (event.key === 'Escape') closeAdjust()
}

onMounted(() => {
  window.addEventListener('keydown', onKeydown)
  load()
})

onBeforeUnmount(() => {
  window.removeEventListener('keydown', onKeydown)
})
</script>

<template>
  <section class="patient-panel" aria-labelledby="stock-heading">
    <div class="panel-head">
      <h2 id="stock-heading">药品库存</h2>
    </div>
    <p class="field-hint">当前数量只读。调整须填写原因，记一笔调整流水。本期没有批次和效期。</p>

    <form class="master-filters" @submit.prevent="load">
      <label class="field">
        <span>编码 / 品名</span>
        <input v-model="q" type="search" data-testid="stock-q" />
      </label>
      <button class="button button-ghost button-compact" type="submit">查询</button>
    </form>

    <p v-if="notice && !adjusting" class="banner" :class="{ 'banner-ok': !noticeError }" :role="noticeError ? 'alert' : 'status'" data-testid="stock-notice">
      {{ notice }}<span v-if="errorCode">（{{ errorCode }}）</span>
    </p>
    <p v-if="loading" class="shell-status">正在读取…</p>
    <p v-else-if="rows.length === 0" class="result-empty" data-testid="stock-empty">
      {{ q ? '没有符合条件的库存' : '暂无药品库存（无 DRUG 或未建 SKU）' }}
    </p>
    <div v-else class="result-wrap">
      <table class="register-table" data-testid="stock-list">
        <caption class="sr-only">药品库存</caption>
        <thead>
          <tr>
            <th>编码</th>
            <th>品名</th>
            <th>当前数量</th>
            <th>单位</th>
            <th>状态</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="row in rows" :key="row.skuId" :class="{ 'is-active': row.skuId === selectedId }">
            <td class="mono">{{ row.itemNo }}</td>
            <td>{{ row.itemName }}</td>
            <td>{{ formatQty(row.qtyOnHand) }}</td>
            <td>{{ row.units }}</td>
            <td><span class="status-pill" :data-status="row.status">{{ statusLabel(row.status) }}</span></td>
            <td class="ticket-actions">
              <button class="text-btn" type="button" @click="selectRow(row)">流水</button>
              <button class="text-btn" type="button" :data-testid="`stock-adjust-${row.itemNo}`" @click="openAdjust(row)">调整</button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <div v-if="selected" class="master-ledger">
      <h3>流水 · {{ selected.itemName }}</h3>
      <p v-if="ledger.length === 0" class="result-empty">暂无流水</p>
      <div v-else class="result-wrap">
        <table class="register-table" data-testid="ledger-list">
          <caption class="sr-only">库存流水</caption>
          <thead>
            <tr>
              <th>时间</th>
              <th>类型</th>
              <th>数量</th>
              <th>结存</th>
              <th>原因</th>
              <th>操作员</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="row in ledger" :key="row.ledgerId">
              <td>{{ formatTime(row.createdAt) }}</td>
              <td>{{ bizLabel(row.bizType) }}</td>
              <td>{{ formatQty(row.qtyDelta) }}</td>
              <td>{{ formatQty(row.qtyAfter) }}</td>
              <td>{{ row.remark }}</td>
              <td>{{ row.operatorName || '—' }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <div v-if="adjusting" class="master-modal" role="presentation" @click.self="closeAdjust">
      <form class="master-modal-card" role="dialog" aria-modal="true" aria-labelledby="adjust-title" data-testid="adjust-dialog" @submit.prevent="onAdjust">
        <h3 id="adjust-title">调整库存</h3>
        <p class="field-hint">{{ adjusting.itemNo }} · {{ adjusting.itemName }} · 当前 {{ formatQty(adjusting.qtyOnHand) }} {{ adjusting.units }}</p>
        <label class="field">
          <span>调整数量 <abbr class="req" title="必填">*</abbr></span>
          <input v-model="form.qtyDelta" type="number" step="0.001" required data-testid="adjust-delta" />
          <span class="field-hint">正数增加，负数减少。</span>
        </label>
        <label class="field">
          <span>原因 <abbr class="req" title="必填">*</abbr></span>
          <input v-model="form.remark" required data-testid="adjust-remark" placeholder="期初 / 盘点 / 纠错" autocomplete="off" />
        </label>
        <p v-if="preview != null" class="field-hint" :class="{ 'is-danger': preview < 0 }">
          调整后数量 {{ formatQty(preview) }}{{ preview < 0 ? '，不能为负' : '' }}
        </p>
        <p v-if="notice && noticeError" class="banner" role="alert">
          {{ notice }}<span v-if="errorCode">（{{ errorCode }}）</span>
        </p>
        <div class="panel-actions">
          <button class="button button-compact" type="submit" data-testid="adjust-save" :disabled="saving || !canAdjust">
            {{ saving ? '正在保存…' : '确认调整' }}
          </button>
          <button class="button button-ghost button-compact" type="button" @click="closeAdjust">取消</button>
        </div>
      </form>
    </div>
  </section>
</template>
