<script setup>
// 2026-10-14 P1-H05-WEB-USABILITY 列表与流水区分失败和空结果，写成功提示独立保留。
// 2026-10-13 P1-H05-MASTER-UI 药品库存：只读当前量，弹窗调整并记 ADJUST 流水。结果不能为负。
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { ApiError } from '../../api/errors.js'
import { session } from '../../auth/session.js'
import { adjustInventory, fetchInventory, fetchInventoryBatches, fetchLedger } from '../../master/api.js'
import { bizLabel, formatQty, formatTime, statusLabel } from '../../master/labels.js'
import MasterPanelHeader from './MasterPanelHeader.vue'
import MasterSymbol from './MasterSymbol.vue'

const rows = ref([])
const ledger = ref([])
const loading = ref(false)
const listReady = ref(false)
const loadError = ref('')
const loadErrorCode = ref('')
const ledgerLoading = ref(false)
const ledgerReady = ref(false)
const ledgerError = ref('')
const ledgerErrorCode = ref('')
const batchRows = ref([])
const batchSummary = ref(null)
const batchLoading = ref(false)
const batchReady = ref(false)
const batchError = ref('')
const batchErrorCode = ref('')
const batchNextCursor = ref('')
const saving = ref(false)
const q = ref('')
const selectedId = ref('')
const ledgerNextCursor = ref('')
const ledgerFilter = reactive({ bizType: '', from: '', to: '' })
const adjusting = ref(null)
const notice = ref('')
const noticeError = ref(false)
const errorCode = ref('')
const adjustDialog = ref(null)
const adjustInput = ref(null)
const searchInput = ref(null)
const batchSkuInput = ref(null)
let adjustReturnFocus = null
let ledgerReadId = 0
let batchReadId = 0
const form = reactive({ qtyDelta: '', remark: '' })
const batchFilter = reactive({
  skuCode: '',
  storageUnitId: '',
  batchNo: '',
  expiryStatus: '',
  asOfDate: '2026-10-15',
  expiryWindowDays: '30',
})

const batchStatusLabels = {
  UNKNOWN: '未知效期',
  VALID: '有效',
  EXPIRING: '近效期',
  EXPIRED: '已过期',
}

const canMaintain = computed(() => listReady.value && !loading.value && !saving.value)
const selected = computed(() => listReady.value ? rows.value.find((item) => item.skuId === selectedId.value) || null : null)
const preview = computed(() => {
  if (!adjusting.value || form.qtyDelta === '' || form.qtyDelta == null) return null
  const delta = Number(form.qtyDelta)
  if (!Number.isFinite(delta)) return null
  return Math.round((adjusting.value.qtyOnHand + delta + Number.EPSILON) * 1000) / 1000
})
const canAdjust = computed(() => {
  const delta = Number(form.qtyDelta)
  return canMaintain.value && preview.value != null && Number.isFinite(delta) && delta !== 0 && preview.value >= 0 && Boolean(form.remark.trim())
})

function show(message, isError = false, code = '') {
  notice.value = message
  noticeError.value = isError
  errorCode.value = code
}

async function load() {
  if (loading.value) return
  loading.value = true
  listReady.value = false
  loadError.value = ''
  loadErrorCode.value = ''
  try {
    const listed = await fetchInventory(session.accessToken, { q: q.value })
    rows.value = listed.items
    listReady.value = true
    if (selectedId.value && !rows.value.some((item) => item.skuId === selectedId.value)) {
      selectedId.value = ''
      ledger.value = []
      ledgerNextCursor.value = ''
      ledgerReady.value = false
    } else if (selectedId.value) {
      await loadLedger(selectedId.value)
    }
  } catch (error) {
    loadError.value = error instanceof ApiError ? error.message : '无法读取库存'
    loadErrorCode.value = error instanceof ApiError ? error.code : ''
  } finally {
    loading.value = false
  }
}

async function loadLedger(skuId, { cursor = '', append = false } = {}) {
  if (!skuId) return
  const sku = rows.value.find((item) => item.skuId === skuId)
  if (!sku) return
  const readId = ++ledgerReadId
  ledgerLoading.value = true
  ledgerReady.value = false
  ledgerError.value = ''
  ledgerErrorCode.value = ''
  try {
    const listed = await fetchLedger(session.accessToken, sku.itemNo || sku.skuCode, {
      ...ledgerFilter,
      cursor,
      limit: 50,
    })
    if (readId !== ledgerReadId || selectedId.value !== skuId) return
    ledger.value = append ? [...ledger.value, ...listed.items] : listed.items
    ledgerNextCursor.value = listed.nextCursor || ''
    ledgerReady.value = true
  } catch (error) {
    if (readId !== ledgerReadId || selectedId.value !== skuId) return
    ledgerError.value = error instanceof ApiError ? error.message : '无法读取流水'
    ledgerErrorCode.value = error instanceof ApiError ? error.code : ''
  } finally {
    if (readId === ledgerReadId) ledgerLoading.value = false
  }
}

async function selectRow(row) {
  if (!canMaintain.value) return
  selectedId.value = row.skuId
  ledger.value = []
  ledgerNextCursor.value = ''
  await loadLedger(row.skuId)
}

async function applyLedgerFilters() {
  if (!selectedId.value || ledgerLoading.value) return
  ledger.value = []
  ledgerNextCursor.value = ''
  await loadLedger(selectedId.value)
}

async function loadMoreLedger() {
  if (!selectedId.value || !ledgerNextCursor.value || ledgerLoading.value) return
  await loadLedger(selectedId.value, { cursor: ledgerNextCursor.value, append: true })
}

function batchStatusLabel(status) {
  return batchStatusLabels[status] || status || '—'
}

function batchQuantity(value) {
  return formatQty(value)
}

function summarizeBatchRows(rows) {
  const keyCounts = new Map()
  let qtyTotal = 0
  let qtyUnknown = 0
  for (const row of rows) {
    const quantity = Number(row.qtyOnHand)
    if (Number.isFinite(quantity)) {
      qtyTotal += quantity
      if (row.expiryStatus === 'UNKNOWN') qtyUnknown += quantity
    }
    const key = [row.storageUnitId || '', row.skuId || '', row.batchNo || '', row.sourceRef || ''].join('\u0000')
    keyCounts.set(key, (keyCounts.get(key) || 0) + 1)
  }
  return {
    batchCount: rows.length,
    qtyTotal: qtyTotal.toFixed(3),
    qtyUnknown: qtyUnknown.toFixed(3),
    duplicateKeyCount: [...keyCounts.values()].filter((count) => count > 1).length,
  }
}

async function loadBatches({ cursor = '', append = false } = {}) {
  if (batchLoading.value) return
  const readId = ++batchReadId
  batchLoading.value = true
  batchError.value = ''
  batchErrorCode.value = ''
  if (!append) {
    batchReady.value = false
    batchRows.value = []
    batchSummary.value = null
    batchNextCursor.value = ''
  }
  try {
    const listed = await fetchInventoryBatches(session.accessToken, {
      ...batchFilter,
      cursor,
      limit: 50,
    })
    if (readId !== batchReadId) return
    const loadedRows = append ? [...batchRows.value, ...listed.items] : listed.items
    batchRows.value = loadedRows
    batchSummary.value = summarizeBatchRows(loadedRows)
    batchNextCursor.value = listed.nextCursor || ''
    batchReady.value = true
  } catch (error) {
    if (readId !== batchReadId) return
    batchError.value = error instanceof ApiError ? error.message : '无法读取批次/效期'
    batchErrorCode.value = error instanceof ApiError ? error.code : ''
  } finally {
    if (readId === batchReadId) batchLoading.value = false
  }
}

function prepareBatchFilter(row) {
  if (!row) return
  batchFilter.skuCode = row.itemNo || row.skuCode || ''
  nextTick(() => batchSkuInput.value?.focus())
}

async function loadMoreBatches() {
  if (!batchNextCursor.value || batchLoading.value) return
  await loadBatches({ cursor: batchNextCursor.value, append: true })
}

function openAdjust(row) {
  if (!canMaintain.value) return
  adjustReturnFocus = document.activeElement
  adjusting.value = row
  form.qtyDelta = ''
  form.remark = ''
  show('', false)
  nextTick(() => adjustInput.value?.focus())
}

function closeAdjust() {
  if (saving.value) return
  adjusting.value = null
  nextTick(() => {
    if (adjustReturnFocus?.isConnected) adjustReturnFocus.focus()
    else searchInput.value?.focus()
  })
}

async function onAdjust() {
  if (!canAdjust.value || !adjusting.value) return
  saving.value = true
  const skuId = adjusting.value.skuId
  try {
    await adjustInventory(session.accessToken, skuId, {
      qtyDelta: form.qtyDelta,
      remark: form.remark,
      rowVersion: adjusting.value.rowVersion,
      operatorName: session.user && session.user.displayName,
    })
    adjusting.value = null
    show('已调整库存', false)
    selectedId.value = skuId
    await load()
    nextTick(() => searchInput.value?.focus())
  } catch (error) {
    show(error instanceof ApiError ? error.message : '调整失败', true, error instanceof ApiError ? error.code : '')
  } finally {
    saving.value = false
  }
}

function onKeydown(event) {
  if (!adjusting.value) return
  if (event.key === 'Escape') {
    event.preventDefault()
    closeAdjust()
  } else if (event.key === 'Tab') {
    const controls = [...(adjustDialog.value?.querySelectorAll('button:not([disabled]), input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex="0"]') || [])]
    const first = controls[0]
    const last = controls.at(-1)
    if (!first) {
      event.preventDefault()
      adjustDialog.value?.focus()
    } else if (!controls.includes(document.activeElement)) {
      event.preventDefault()
      if (event.shiftKey) last?.focus()
      else first.focus()
    } else if (event.shiftKey && document.activeElement === first) {
      event.preventDefault()
      last?.focus()
    } else if (!event.shiftKey && document.activeElement === last) {
      event.preventDefault()
      first?.focus()
    }
  }
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
  <section class="patient-panel master-data-panel master-inventory-panel" aria-labelledby="stock-heading">
    <MasterPanelHeader title="药品库存" title-id="stock-heading" icon="inventory" description="核对当前药品数量，查看流水并进行库存调整。" :count="listReady ? rows.length : null" :loading="loading" />
    <p class="field-hint master-inline-note">当前数量仍按一药品一 SKU 总量账运行。批次/效期为独立只读查询，不参与库存调整、发药或退药。</p>

    <form class="master-filters" @submit.prevent="!saving && load()">
      <label class="field">
        <span>编码 / 品名</span>
        <input ref="searchInput" v-model="q" type="search" placeholder="搜索药品编码或品名" data-testid="stock-q" />
      </label>
      <button class="button button-ghost button-compact" type="submit" :disabled="loading || saving">查询</button>
    </form>

    <p v-if="notice && !adjusting" class="banner" :class="{ 'banner-ok': !noticeError }" :role="noticeError ? 'alert' : 'status'" data-testid="stock-notice">
      {{ notice }}<span v-if="errorCode">（{{ errorCode }}）</span>
    </p>
    <div class="master-inventory-layout">
      <div class="master-inventory-list">
    <p v-if="loading" class="shell-status" role="status">正在读取…</p>
    <div v-else-if="loadError" class="master-load-error" role="alert" data-testid="stock-load-error">
      <p>{{ loadError }}<span v-if="loadErrorCode">（{{ loadErrorCode }}）</span></p>
      <button class="button button-ghost button-compact" type="button" data-testid="stock-retry" @click="load">重新读取</button>
    </div>
    <p v-else-if="listReady && rows.length === 0" class="result-empty" data-testid="stock-empty">
      {{ q ? '没有符合条件的库存' : '暂无药品库存（无 DRUG 或未建 SKU）' }}
    </p>
    <div v-else-if="listReady" class="result-wrap" tabindex="0" role="region" aria-label="药品库存列表，可横向滚动">
      <div class="master-table-head"><h3>库存列表</h3><span>{{ rows.length }} 条查询结果</span></div>
      <table class="register-table" data-testid="stock-list">
        <caption class="sr-only">药品库存</caption>
        <thead>
          <tr>
            <th scope="col">编码</th>
            <th scope="col">品名</th>
            <th scope="col">当前数量</th>
            <th scope="col">单位</th>
            <th scope="col">状态</th>
            <th scope="col">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="row in rows" :key="row.skuId" :class="{ 'is-active': row.skuId === selectedId }">
            <td class="mono">{{ row.itemNo }}</td>
            <td class="master-primary-cell">{{ row.itemName }}</td>
            <td class="master-qty-cell" :class="{ 'is-zero': Number(row.qtyOnHand) === 0 }">{{ formatQty(row.qtyOnHand) }}</td>
            <td>{{ row.units }}</td>
            <td><span class="status-pill" :data-status="row.status">{{ statusLabel(row.status) }}</span></td>
            <td class="ticket-actions">
              <button class="text-btn" type="button" :disabled="!canMaintain" @click="selectRow(row)">流水</button>
              <button class="text-btn" type="button" :data-testid="`stock-batch-${row.itemNo}`" :disabled="!canMaintain" @click="prepareBatchFilter(row)">批次/效期</button>
              <button class="text-btn" type="button" :data-testid="`stock-adjust-${row.itemNo}`" :disabled="!canMaintain" @click="openAdjust(row)">调整</button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
      </div>

    <div v-if="selected" class="master-ledger">
      <div class="master-ledger-head"><div><p class="master-eyebrow">库存明细</p><h3>流水 · {{ selected.itemName }}</h3></div><span class="master-soft-badge">{{ selected.itemNo }}</span></div>
      <div class="master-selected-stock"><span>当前库存</span><strong>{{ formatQty(selected.qtyOnHand) }}<small>{{ selected.units }}</small></strong></div>
      <form class="master-filters" data-testid="ledger-filters" @submit.prevent="applyLedgerFilters">
        <label class="field">
          <span>业务类型</span>
          <select v-model="ledgerFilter.bizType" data-testid="ledger-biz-type" :disabled="ledgerLoading">
            <option value="">全部</option>
            <option value="ADJUST">调整</option>
            <option value="DISPENSE">发药</option>
            <option value="RETURN">退药</option>
          </select>
        </label>
        <label class="field">
          <span>开始日期</span>
          <input v-model="ledgerFilter.from" type="date" data-testid="ledger-from" :disabled="ledgerLoading" />
        </label>
        <label class="field">
          <span>结束日期</span>
          <input v-model="ledgerFilter.to" type="date" data-testid="ledger-to" :disabled="ledgerLoading" />
        </label>
        <button class="button button-ghost button-compact" type="submit" data-testid="ledger-search" :disabled="ledgerLoading">筛选</button>
      </form>
      <p v-if="ledgerLoading" class="shell-status" role="status">正在读取流水…</p>
      <div v-else-if="ledgerError" class="master-load-error" role="alert" data-testid="ledger-load-error">
        <p>{{ ledgerError }}<span v-if="ledgerErrorCode">（{{ ledgerErrorCode }}）</span></p>
        <button class="button button-ghost button-compact" type="button" data-testid="ledger-retry" :disabled="!canMaintain" @click="loadLedger(selectedId)">重新读取流水</button>
      </div>
      <p v-else-if="ledgerReady && ledger.length === 0" class="result-empty">暂无流水</p>
      <div v-else-if="ledgerReady" class="result-wrap" tabindex="0" role="region" aria-label="库存流水，可横向滚动">
        <table class="register-table" data-testid="ledger-list">
          <caption class="sr-only">库存流水</caption>
          <thead>
            <tr>
              <th scope="col">时间</th>
              <th scope="col">类型</th>
              <th scope="col">数量</th>
              <th scope="col">结存</th>
              <th scope="col">原因</th>
              <th scope="col">操作员</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="row in ledger" :key="row.ledgerId">
              <td>{{ formatTime(row.createdAt) }}</td>
              <td>{{ bizLabel(row.bizType) }}</td>
              <td class="master-delta-cell" :class="{ 'is-positive': Number(row.qtyDelta) > 0 }">{{ Number(row.qtyDelta) > 0 ? '+' : '' }}{{ formatQty(row.qtyDelta) }}</td>
              <td>{{ formatQty(row.qtyAfter) }}</td>
              <td>{{ row.remark }}</td>
              <td>{{ row.operatorName || '—' }}</td>
            </tr>
          </tbody>
        </table>
      </div>
      <button v-if="ledgerReady && ledgerNextCursor" class="button button-ghost button-compact" type="button" data-testid="ledger-more" :disabled="ledgerLoading" @click="loadMoreLedger">更多</button>
    </div>
    <div v-else class="master-ledger master-ledger-empty"><span class="master-ledger-empty-symbol"><MasterSymbol name="inventory" /></span><h3>查看库存流水</h3><p>点击药品行的“流水”，查看该药品的数量与调整记录。</p></div>
    </div>

    <section class="master-ledger master-batch-read" aria-labelledby="batch-heading">
      <div class="master-ledger-head"><div><p class="master-eyebrow">库存批次</p><h3 id="batch-heading">批次 / 效期只读查询</h3></div><span class="master-soft-badge">只读</span></div>
      <p class="field-hint">效期分类按参考日期和窗口计算：未知、有效、近效期、已过期。查询需要独立的 inventory:batch-read 权限。</p>
      <form class="master-filters" data-testid="batch-filters" @submit.prevent="loadBatches">
        <label class="field">
          <span>SKU / 编码</span>
          <input ref="batchSkuInput" v-model="batchFilter.skuCode" type="search" placeholder="可选，例 DRUG-A" data-testid="batch-sku" :disabled="batchLoading" />
        </label>
        <label class="field">
          <span>仓 / 药房</span>
          <input v-model="batchFilter.storageUnitId" type="search" placeholder="可选" data-testid="batch-storage" :disabled="batchLoading" />
        </label>
        <label class="field">
          <span>批号</span>
          <input v-model="batchFilter.batchNo" type="search" placeholder="可选" data-testid="batch-no" :disabled="batchLoading" />
        </label>
        <label class="field">
          <span>效期状态</span>
          <select v-model="batchFilter.expiryStatus" data-testid="batch-status" :disabled="batchLoading">
            <option value="">全部</option>
            <option value="UNKNOWN">未知效期</option>
            <option value="VALID">有效</option>
            <option value="EXPIRING">近效期</option>
            <option value="EXPIRED">已过期</option>
          </select>
        </label>
        <label class="field">
          <span>参考日期</span>
          <input v-model="batchFilter.asOfDate" type="date" required data-testid="batch-as-of" :disabled="batchLoading" />
        </label>
        <label class="field">
          <span>近效期窗口（天）</span>
          <input v-model="batchFilter.expiryWindowDays" type="number" min="0" step="1" required data-testid="batch-window" :disabled="batchLoading" />
        </label>
        <button class="button button-ghost button-compact" type="submit" data-testid="batch-search" :disabled="batchLoading">{{ batchLoading ? '读取中…' : '查询批次' }}</button>
      </form>
      <p v-if="batchLoading" class="shell-status" role="status" data-testid="batch-loading">正在读取批次/效期…</p>
      <div v-else-if="batchError" class="master-load-error" role="alert" data-testid="batch-load-error">
        <p>{{ batchError }}<span v-if="batchErrorCode">（{{ batchErrorCode }}）</span></p>
        <button class="button button-ghost button-compact" type="button" data-testid="batch-retry" @click="loadBatches">重新读取</button>
      </div>
      <p v-else-if="batchReady && batchRows.length === 0" class="result-empty" data-testid="batch-empty">暂无符合条件的批次/效期记录</p>
      <template v-else-if="batchReady">
        <div v-if="batchSummary" class="master-selected-stock" data-testid="batch-summary">
          <span>批次 {{ batchSummary.batchCount ?? batchRows.length }} 条 · 未知效期 {{ batchQuantity(batchSummary.qtyUnknown) }} · 重复键 {{ batchSummary.duplicateKeyCount ?? 0 }}</span>
          <strong>{{ batchQuantity(batchSummary.qtyTotal) }}<small>合计数量</small></strong>
        </div>
        <div class="result-wrap" tabindex="0" role="region" aria-label="药品批次和效期列表，可横向滚动">
          <div class="master-table-head"><h3>批次结果</h3><span>{{ batchRows.length }} 条已读取</span></div>
          <table class="register-table" data-testid="batch-list">
            <caption class="sr-only">药品批次与效期</caption>
            <thead><tr><th scope="col">批号</th><th scope="col">编码 / 品名</th><th scope="col">药房</th><th scope="col">生产日期</th><th scope="col">效期</th><th scope="col">数量</th><th scope="col">效期状态</th></tr></thead>
            <tbody>
              <tr v-for="row in batchRows" :key="row.batchId">
                <td class="mono">{{ row.batchNo || '—' }}</td>
                <td class="master-primary-cell">{{ row.itemNo }}<small>{{ row.itemName }}</small></td>
                <td>{{ row.storageUnitId || '—' }}</td>
                <td>{{ row.productDate || '—' }}</td>
                <td>{{ row.expireDate || '—' }}</td>
                <td class="master-qty-cell">{{ batchQuantity(row.qtyOnHand) }} {{ row.units || '' }}</td>
                <td><span class="status-pill" :data-status="row.expiryStatus">{{ batchStatusLabel(row.expiryStatus) }}</span></td>
              </tr>
            </tbody>
          </table>
        </div>
        <button v-if="batchNextCursor" class="button button-ghost button-compact" type="button" data-testid="batch-more" :disabled="batchLoading" @click="loadMoreBatches">更多</button>
      </template>
      <p v-else class="result-empty" data-testid="batch-idle">填写条件后查询批次/效期；此查询不会写入库存或流水。</p>
    </section>

    <div v-if="adjusting" class="master-modal" role="presentation" @click.self="closeAdjust">
      <form ref="adjustDialog" class="master-modal-card" tabindex="-1" role="dialog" aria-modal="true" aria-labelledby="adjust-title" :aria-busy="saving" data-testid="adjust-dialog" @submit.prevent="onAdjust">
        <div class="master-adjust-head"><span class="master-panel-symbol"><MasterSymbol name="inventory" /></span><div><p class="master-eyebrow">库存管理</p><h3 id="adjust-title">调整库存</h3></div><button type="button" class="master-modal-close" aria-label="关闭库存调整" :disabled="saving" @click="closeAdjust"><svg viewBox="0 0 20 20" aria-hidden="true"><path d="m5 5 10 10M15 5 5 15"/></svg></button></div>
        <div class="master-adjust-item"><strong>{{ adjusting.itemName }}</strong><span>{{ adjusting.itemNo }} · 当前 {{ formatQty(adjusting.qtyOnHand) }} {{ adjusting.units }}</span></div>
        <label class="field">
          <span>调整数量 <abbr class="req" title="必填">*</abbr></span>
          <input ref="adjustInput" v-model="form.qtyDelta" type="number" step="0.001" required data-testid="adjust-delta" :disabled="saving" />
          <span class="field-hint">正数增加，负数减少。</span>
        </label>
        <label class="field">
          <span>原因 <abbr class="req" title="必填">*</abbr></span>
          <input v-model="form.remark" required data-testid="adjust-remark" placeholder="期初 / 盘点 / 纠错" autocomplete="off" :disabled="saving" />
        </label>
        <div class="master-adjust-preview" :class="{ 'is-danger': preview != null && preview < 0 }"><div><span>当前数量</span><strong>{{ formatQty(adjusting.qtyOnHand) }}<small>{{ adjusting.units }}</small></strong></div><svg viewBox="0 0 20 20" aria-hidden="true"><path d="M4 10h12M11 5l5 5-5 5"/></svg><div><span>调整后数量</span><strong>{{ preview != null ? formatQty(preview) : '—' }}<small>{{ adjusting.units }}</small></strong></div></div>
        <p v-if="preview != null" class="field-hint master-preview-note" :class="{ 'is-danger': preview < 0 }">调整后数量 {{ formatQty(preview) }}{{ preview < 0 ? '，不能为负' : '' }}</p>
        <p v-if="notice && noticeError" class="banner" role="alert">
          {{ notice }}<span v-if="errorCode">（{{ errorCode }}）</span>
        </p>
        <div class="panel-actions">
          <button class="button button-compact" type="submit" data-testid="adjust-save" :disabled="saving || !canAdjust">
            {{ saving ? '正在保存…' : '确认调整' }}
          </button>
          <button class="button button-ghost button-compact" type="button" :disabled="saving" @click="closeAdjust">取消</button>
        </div>
      </form>
    </div>
  </section>
</template>
