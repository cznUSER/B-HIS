<script setup>
/**
 * P0-B-QUEUE-01 shared outpatient waiting/pending queue (not doctor-station local).
 * Morph A: assemble from existing listRegisters/listOrders/listCharges/listDispenses.
 * Selecting a row calls selectOutpatientContext (P0-A confirm flow) — no write-semantic changes.
 */
import { computed, onMounted, ref, watch } from 'vue'
import { session } from '../../auth/session.js'
import { listRegisters } from '../../api/registerApi.js'
import { listOrders } from '../../api/orderApi.js'
import { listCharges } from '../../api/chargeApi.js'
import { listDispenses } from '../../api/dispenseApi.js'
import { fetchPatient } from '../../api/patientApi.js'
import { outpatientContext, selectOutpatientContext } from '../../workbench/outpatientContext.js'
import {
  OUTPATIENT_QUEUE_TABS,
  filterOutpatientQueue,
  loadOutpatientQueue,
  selectQueueRegister,
} from '../../workbench/outpatientQueue.js'

const props = defineProps({
  visitDate: { type: String, default: '' },
})

const emit = defineEmits(['selected'])

const activeTab = ref('waiting')
const search = ref('')
const loading = ref(false)
const rows = ref([])
const errors = ref([])
const loadMessage = ref('')
const selectingId = ref('')

const today = computed(() => {
  if (props.visitDate) return props.visitDate
  try {
    return new Intl.DateTimeFormat('en-CA', { timeZone: 'Asia/Shanghai' }).format(new Date())
  } catch {
    const now = new Date()
    const y = now.getFullYear()
    const m = String(now.getMonth() + 1).padStart(2, '0')
    const d = String(now.getDate()).padStart(2, '0')
    return `${y}-${m}-${d}`
  }
})

const token = computed(() => session.accessToken || '')
const forbidden = computed(() => errors.value.some((item) => item.status === 403 || item.status === 401))
const hasHardError = computed(() => errors.value.length > 0 && !rows.value.length)
const visibleRows = computed(() => filterOutpatientQueue(rows.value, activeTab.value, search.value))
const selectedRegisterId = computed(() => outpatientContext.register?.registerId || '')
const selectedPatientId = computed(() => outpatientContext.patient?.patientId || '')

const emptyCopy = computed(() => {
  const tab = OUTPATIENT_QUEUE_TABS.find((item) => item.id === activeTab.value)
  const label = tab?.label || '当前'
  if (search.value.trim()) {
    return { title: `没有匹配「${search.value.trim()}」的${label}患者`, hint: '换个关键词，或清空搜索后再试' }
  }
  return { title: `暂无${label}患者`, hint: '符合今日就诊与当前筛选项的患者会显示在这里' }
})

function statusLabel(status) {
  return {
    NORMAL: '候诊',
    USED: '已接诊',
    DIAGNOSING: '诊疗中',
    DIAGNOSED: '待开单',
    PAUSED: '暂停',
    BACK: '退号',
  }[status] || status || '—'
}

function sexLabel(value) {
  return { M: '男', F: '女', U: '未知' }[value] || value || ''
}

function tabCount(tabId) {
  return filterOutpatientQueue(rows.value, tabId, search.value).length
}

async function reload() {
  if (!token.value) {
    rows.value = []
    errors.value = [{ domain: 'auth', status: 401, message: '请先登录后查看门诊队列' }]
    loadMessage.value = '请先登录'
    return
  }
  loading.value = true
  loadMessage.value = ''
  try {
    const loaded = await loadOutpatientQueue({
      visitDate: today.value,
      listRegisters,
      listOrders,
      listCharges,
      listDispenses,
    }, token.value)
    // Keep previously selected patient while filters/reload change — do not touch outpatientContext here.
    rows.value = loaded.rows
    errors.value = loaded.errors
    if (loaded.errors.length && loaded.rows.length) {
      loadMessage.value = `部分列表加载失败（${loaded.errors.map((e) => e.domain).join('、')}），已显示可用数据`
    }
  } catch (error) {
    rows.value = []
    errors.value = [{
      domain: 'queue',
      status: typeof error?.status === 'number' ? error.status : 0,
      message: error?.message || '门诊队列加载失败',
    }]
    loadMessage.value = error?.message || '门诊队列加载失败'
  } finally {
    loading.value = false
  }
}

async function onSelect(row) {
  if (!row?.registerId || selectingId.value) return
  selectingId.value = row.registerId
  try {
    const accepted = await selectQueueRegister(row, {
      fetchPatient,
      selectOutpatientContext,
    }, token.value)
    if (accepted) emit('selected', row)
  } catch (error) {
    loadMessage.value = error?.message || '选择患者失败'
  } finally {
    selectingId.value = ''
  }
}

watch(activeTab, () => {
  // Filter-only: never clear outpatientContext / current patient.
})

onMounted(reload)
</script>

<template>
  <section class="outpatient-queue" aria-label="门诊共享队列" data-testid="outpatient-queue">
    <header class="outpatient-queue-head">
      <div>
        <p class="outpatient-queue-eyebrow">SHARED QUEUE · {{ today }}</p>
        <h2>门诊队列</h2>
        <p class="outpatient-queue-sub">候诊与待处理一览；选中行写入同一门诊上下文（切换他患走确认）。筛选不丢当前患者。</p>
      </div>
      <button class="outpatient-queue-refresh" type="button" :disabled="loading" @click="reload">
        {{ loading ? '刷新中…' : '刷新' }}
      </button>
    </header>

    <div class="outpatient-queue-tabs" role="tablist" aria-label="队列筛选项">
      <button
        v-for="tab in OUTPATIENT_QUEUE_TABS"
        :key="tab.id"
        type="button"
        role="tab"
        class="outpatient-queue-tab"
        :class="{ 'is-active': activeTab === tab.id }"
        :aria-selected="activeTab === tab.id"
        @click="activeTab = tab.id"
      >
        <span>{{ tab.label }}</span>
        <em>{{ tabCount(tab.id) }}</em>
      </button>
    </div>

    <label class="outpatient-queue-search">
      <span class="sr-only">搜索患者</span>
      <input v-model="search" type="search" placeholder="姓名 / 患者号 / 就诊号" autocomplete="off">
    </label>

    <p v-if="selectedPatientId" class="outpatient-queue-current" data-testid="outpatient-queue-current">
      当前患者保留中：{{ outpatientContext.patient?.name || outpatientContext.patient?.patientName || selectedPatientId }}
      <template v-if="selectedRegisterId"> · 就诊 {{ outpatientContext.register?.registerNo || selectedRegisterId }}</template>
    </p>

    <div v-if="forbidden" class="outpatient-queue-state is-forbidden" role="alert" data-testid="outpatient-queue-forbidden">
      <strong>无权限查看门诊队列</strong>
      <span>{{ errors[0]?.message || '当前账号缺少挂号/医嘱/收费/发药列表权限' }}</span>
      <button type="button" @click="reload">重试</button>
    </div>
    <div v-else-if="hasHardError && !loading" class="outpatient-queue-state is-error" role="alert" data-testid="outpatient-queue-error">
      <strong>队列加载失败</strong>
      <span>{{ errors.map((item) => item.message).join('；') }}</span>
      <button type="button" @click="reload">重试</button>
    </div>
    <div v-else-if="loading" class="outpatient-queue-state" data-testid="outpatient-queue-loading">正在读取今日队列…</div>
    <div v-else-if="!visibleRows.length" class="outpatient-queue-state is-empty" data-testid="outpatient-queue-empty">
      <strong>{{ emptyCopy.title }}</strong>
      <span>{{ emptyCopy.hint }}</span>
      <button type="button" @click="reload">重新加载</button>
    </div>
    <div v-else class="outpatient-queue-table-wrap" data-testid="outpatient-queue-table">
      <table class="outpatient-queue-table">
        <thead>
          <tr>
            <th scope="col">患者</th>
            <th scope="col">就诊号</th>
            <th scope="col">挂号态</th>
            <th scope="col">科室/医生</th>
            <th scope="col">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="row in visibleRows"
            :key="row.registerId"
            :class="{ 'is-selected': selectedRegisterId === row.registerId }"
          >
            <td>
              <strong>{{ row.patientName || '未命名患者' }}</strong>
              <small>{{ sexLabel(row.sex) }}<template v-if="row.age !== '' && row.age != null"> / {{ row.age }}岁</template></small>
            </td>
            <td>{{ row.registerNo || row.registerId }}</td>
            <td><span class="outpatient-queue-tag" :data-status="row.status">{{ statusLabel(row.status) }}</span></td>
            <td>
              <span>{{ row.departmentName || row.departmentId || '—' }}</span>
              <small>{{ row.doctorName || row.doctorId || '' }}</small>
            </td>
            <td>
              <button
                type="button"
                class="outpatient-queue-select"
                :disabled="selectingId === row.registerId"
                @click="onSelect(row)"
              >
                {{ selectedRegisterId === row.registerId ? '当前' : (selectingId === row.registerId ? '切换中…' : '选入') }}
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <p v-if="loadMessage && !forbidden" class="outpatient-queue-note" role="status">{{ loadMessage }}</p>
  </section>
</template>

<style scoped>
.outpatient-queue {
  display: flex;
  flex-direction: column;
  gap: 12px;
  min-height: 0;
  max-height: min(62vh, 640px);
  padding: 18px 18px 14px;
  border: 1px solid #dce8df;
  border-radius: 18px;
  background: #fbfdfb;
  box-shadow: 0 10px 28px #1f4b3510;
}
.outpatient-queue-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}
.outpatient-queue-eyebrow {
  margin: 0 0 4px;
  color: #8fa697;
  font-size: 10px;
  letter-spacing: .08em;
}
.outpatient-queue-head h2 {
  margin: 0;
  color: #2f4f44;
  font-size: 18px;
  font-weight: 650;
}
.outpatient-queue-sub {
  margin: 6px 0 0;
  max-width: 46em;
  color: #6f8579;
  font-size: 12px;
  line-height: 1.6;
}
.outpatient-queue-refresh,
.outpatient-queue-select,
.outpatient-queue-state button {
  height: 34px;
  padding: 0 12px;
  border: 1px solid #d5e5da;
  border-radius: 10px;
  background: #eef7f0;
  color: #2f7a5c;
  font-size: 12px;
  cursor: pointer;
}
.outpatient-queue-refresh:disabled,
.outpatient-queue-select:disabled {
  opacity: .6;
  cursor: default;
}
.outpatient-queue-tabs {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
.outpatient-queue-tab {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  height: 32px;
  padding: 0 10px;
  border: 1px solid #d9e6dd;
  border-radius: 999px;
  background: #fff;
  color: #5d7368;
  font-size: 12px;
  cursor: pointer;
}
.outpatient-queue-tab em {
  min-width: 1.2em;
  color: #8aa092;
  font-style: normal;
  font-variant-numeric: tabular-nums;
}
.outpatient-queue-tab.is-active {
  border-color: #b9d8c4;
  background: #e7f5ec;
  color: #246b52;
  font-weight: 600;
}
.outpatient-queue-search input {
  width: 100%;
  height: 38px;
  padding: 0 12px;
  border: 1px solid #d7e5db;
  border-radius: 12px;
  background: #fff;
  color: #314940;
  font-size: 13px;
}
.outpatient-queue-current {
  margin: 0;
  color: #4d6b5d;
  font-size: 12px;
}
.outpatient-queue-state {
  display: grid;
  gap: 8px;
  justify-items: start;
  padding: 22px 16px;
  border: 1px dashed #d5e3d9;
  border-radius: 14px;
  background: #f7faf7;
  color: #678074;
  font-size: 13px;
}
.outpatient-queue-state strong {
  color: #355347;
  font-size: 14px;
}
.outpatient-queue-state.is-error,
.outpatient-queue-state.is-forbidden {
  border-color: #e7cfc8;
  background: #fff7f4;
  color: #8a5b52;
}
.outpatient-queue-state.is-forbidden strong,
.outpatient-queue-state.is-error strong {
  color: #8a3b2f;
}
.outpatient-queue-table-wrap {
  min-height: 0;
  flex: 1 1 auto;
  overflow: auto;
  border: 1px solid #e1ebe4;
  border-radius: 14px;
  background: #fff;
}
.outpatient-queue-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 13px;
}
.outpatient-queue-table th,
.outpatient-queue-table td {
  padding: 10px 12px;
  border-bottom: 1px solid #eef3ef;
  text-align: left;
  vertical-align: top;
}
.outpatient-queue-table th {
  position: sticky;
  top: 0;
  z-index: 1;
  background: #f5f9f6;
  color: #6b8074;
  font-size: 11px;
  font-weight: 600;
}
.outpatient-queue-table td strong {
  display: block;
  color: #29463d;
}
.outpatient-queue-table td small {
  display: block;
  margin-top: 2px;
  color: #7b8f84;
  font-size: 11px;
}
.outpatient-queue-table tr.is-selected {
  background: #eef8f1;
}
.outpatient-queue-tag {
  display: inline-flex;
  align-items: center;
  height: 22px;
  padding: 0 8px;
  border-radius: 999px;
  background: #eef3ef;
  color: #4f6a5d;
  font-size: 11px;
}
.outpatient-queue-tag[data-status='DIAGNOSED'] { background: #e7f1fb; color: #3a6d9b; }
.outpatient-queue-tag[data-status='DIAGNOSING'] { background: #f4f0e4; color: #8a7340; }
.outpatient-queue-tag[data-status='PAUSED'],
.outpatient-queue-tag[data-status='BACK'] { background: #f8ece9; color: #8a5246; }
.outpatient-queue-note {
  margin: 0;
  color: #8a6a3d;
  font-size: 12px;
}
.sr-only {
  position: absolute;
  width: 1px;
  height: 1px;
  padding: 0;
  margin: -1px;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
  white-space: nowrap;
  border: 0;
}
@media (max-width: 860px) {
  .outpatient-queue { max-height: min(70vh, 720px); }
  .outpatient-queue-table th:nth-child(4),
  .outpatient-queue-table td:nth-child(4) { display: none; }
}
</style>
