<script setup>
import { computed, nextTick, onMounted, reactive, ref } from 'vue'
import { ApiError } from '../../api/errors.js'
import { session } from '../../auth/session.js'
import { fetchAccessModules, fetchUsers } from '../../master/api.js'
import { buildUserOperationsCsv, fetchUserOperationLogs } from '../../master/accessAuditApi.js'
import MasterPanelHeader from '../master/MasterPanelHeader.vue'

const rows = ref([])
const nextCursor = ref('')
const modules = ref([])
const filters = reactive({ moduleId: '', userId: '', begin: '', end: '' })
const selectedUserLabel = ref('')
const pickerOpen = ref(false)
const pickerQ = ref('')
const pickerUsers = ref([])
const pickerLoading = ref(false)
const pickerError = ref('')
const pickerButton = ref(null)
const pickerInput = ref(null)
let pickerRequestId = 0
const loading = ref(false)
const exporting = ref(false)
const ready = ref(false)
const error = ref('')
const limit = 50

const canLoadMore = computed(() => ready.value && Boolean(nextCursor.value) && !loading.value)
const canExport = computed(() => ready.value && rows.value.length > 0 && !loading.value && !exporting.value)

function userLabel(user) {
  const name = user.displayName || user.username
  return name ? `${name}(${user.userId})` : user.userId
}

async function loadPickerUsers() {
  const requestId = ++pickerRequestId
  pickerLoading.value = true
  pickerError.value = ''
  pickerUsers.value = []
  try {
    const page = await fetchUsers(session.accessToken, { q: pickerQ.value })
    if (requestId === pickerRequestId) pickerUsers.value = page.items
  } catch (cause) {
    if (requestId === pickerRequestId) pickerError.value = cause instanceof Error ? cause.message : '读取用户列表失败'
  } finally {
    if (requestId === pickerRequestId) pickerLoading.value = false
  }
}

async function openUserPicker() {
  pickerOpen.value = true
  const pending = loadPickerUsers()
  await nextTick()
  pickerInput.value?.focus()
  await pending
}

function closeUserPicker() {
  pickerOpen.value = false
  pickerRequestId += 1
  pickerLoading.value = false
  pickerButton.value?.focus()
}

function selectUser(user) {
  filters.userId = user.userId
  selectedUserLabel.value = userLabel(user)
  closeUserPicker()
}

function clearUser() {
  filters.userId = ''
  selectedUserLabel.value = ''
  closeUserPicker()
}

function message(errorValue) {
  return errorValue instanceof ApiError || errorValue instanceof Error ? errorValue.message : '读取用户操作日志失败'
}

function queryParams(cursor = '') {
  return { ...filters, cursor, limit }
}

async function loadPage({ append = false } = {}) {
  if (loading.value) return
  loading.value = true
  error.value = ''
  try {
    const page = await fetchUserOperationLogs(session.accessToken, queryParams(append ? nextCursor.value : ''))
    rows.value = append ? [...rows.value, ...page.items] : page.items
    nextCursor.value = page.nextCursor
    ready.value = true
  } catch (cause) {
    error.value = message(cause)
    if (!append) {
      rows.value = []
      nextCursor.value = ''
      ready.value = false
    }
  } finally {
    loading.value = false
  }
}

function search() {
  nextCursor.value = ''
  return loadPage()
}

async function exportCsv() {
  if (!canExport.value) return
  exporting.value = true
  try {
    const blob = new Blob([buildUserOperationsCsv(rows.value)], { type: 'text/csv;charset=utf-8' })
    const url = URL.createObjectURL(blob)
    const anchor = document.createElement('a')
    anchor.href = url
    anchor.download = `user-operations-${new Date().toISOString().slice(0, 10)}.csv`
    document.body.append(anchor)
    anchor.click()
    anchor.remove()
    URL.revokeObjectURL(url)
  } catch (cause) {
    error.value = message(cause)
  } finally {
    exporting.value = false
  }
}

function formatTime(value) {
  const parsed = new Date(value)
  return Number.isNaN(parsed.getTime()) ? value || '—' : parsed.toLocaleString('zh-CN', { hour12: false })
}

onMounted(async () => {
  const [moduleList] = await Promise.all([fetchAccessModules(session.accessToken).catch(() => ({ items: [] })), loadPage()])
  modules.value = moduleList.items || []
})
</script>

<template>
  <section class="patient-panel master-data-panel access-data-panel user-operations-panel" aria-labelledby="user-operations-heading">
    <MasterPanelHeader title="用户操作日志查询" title-id="user-operations-heading" icon="overview" description="查询账号与权限相关的操作记录。" :count="ready ? rows.length : null" count-label="已加载记录" :loading="loading">
      <template #actions><RouterLink class="button button-ghost button-compact uol-hdr-return" :to="{ name: 'access-users' }">返回</RouterLink><button class="button button-compact uol-hdr-search" type="submit" form="user-log-search" :disabled="loading">{{ loading ? '正在查询…' : '检索(S)' }}</button><button class="button button-ghost button-compact uol-hdr-export" type="button" :disabled="!canExport" title="导出已加载" @click="exportCsv">{{ exporting ? '正在导出…' : '导出(E)' }}</button></template>
    </MasterPanelHeader>

    <form id="user-log-search" class="master-filters user-operations-filters" @submit.prevent="search">
      <p class="user-operations-filters-legend uol-abs-legend">检索条件</p>
      <label class="field"><span class="uol-abs-label-module">模块名称</span><select v-model="filters.moduleId" class="uol-abs-ctrl-module"><option value="">全部模块</option><option v-for="item in modules" :key="item.id" :value="String(item.id)">{{ item.name }}（{{ item.id }}）</option></select></label>
      <div class="field user-operations-user-field">
        <span id="user-operations-user-label" class="uol-abs-label-user">用户</span>
        <div class="user-operations-user-button-edit uol-abs-ctrl-user">
          <input :value="selectedUserLabel || filters.userId" type="text" readonly placeholder="点击省略号选择用户" aria-labelledby="user-operations-user-label" />
          <button ref="pickerButton" type="button" aria-label="选择用户" aria-haspopup="dialog" :aria-expanded="pickerOpen" :aria-controls="pickerOpen ? 'user-operations-user-picker' : undefined" @click="pickerOpen ? closeUserPicker() : openUserPicker()">…</button>
        </div>
        <div v-if="pickerOpen" id="user-operations-user-picker" class="user-operations-user-picker" role="dialog" aria-label="选择用户" @keydown.esc.stop.prevent="closeUserPicker">
          <div class="user-operations-user-picker-actions">
            <strong>选择用户</strong>
            <button type="button" :disabled="!filters.userId" @click="clearUser">清除</button>
            <button type="button" @click="closeUserPicker">关闭</button>
          </div>
          <div class="user-operations-user-picker-search">
            <input ref="pickerInput" v-model.trim="pickerQ" type="text" aria-label="筛选用户" placeholder="输入用户名或姓名" @keydown.enter.prevent="loadPickerUsers" />
            <button type="button" :disabled="pickerLoading" @click="loadPickerUsers">筛选</button>
          </div>
          <p v-if="pickerLoading" role="status">正在读取用户…</p>
          <p v-else-if="pickerError" role="alert">{{ pickerError }} <button type="button" @click="loadPickerUsers">重试</button></p>
          <ul v-else-if="pickerUsers.length" class="user-operations-user-picker-list">
            <li v-for="user in pickerUsers" :key="user.userId"><button type="button" :aria-pressed="filters.userId === user.userId" @click="selectUser(user)">{{ userLabel(user) }}</button></li>
          </ul>
          <p v-else role="status">未找到用户</p>
        </div>
      </div>
      <div class="field user-operations-record-date">
        <span class="uol-abs-label-begin">记录日期</span>
        <div class="user-operations-date-range">
          <input v-model="filters.begin" class="uol-abs-ctrl-begin" type="date" aria-label="记录日期起" />
          <span class="user-operations-date-sep uol-abs-label-sep" aria-hidden="true">——</span>
          <input v-model="filters.end" class="uol-abs-ctrl-end" type="date" aria-label="记录日期止" />
        </div>
      </div>
    </form>

    <p v-if="error" class="banner" role="alert">{{ error }} <button class="button button-ghost button-compact" type="button" :disabled="loading" @click="search">重试</button></p>
    <p v-if="loading && !ready" class="shell-status">正在读取操作日志…</p>
    <p v-else-if="ready && !rows.length" class="result-empty">没有符合条件的操作记录</p>
    <div v-else-if="ready" class="result-wrap user-operations-results" tabindex="0" role="region" aria-label="用户操作日志列表，可横向滚动">
      <div class="master-table-head"><h3>操作记录</h3><span>{{ rows.length }} 条已加载</span></div>
      <table class="register-table">
        <thead><tr><th>模块名称</th><th>日志类型</th><th>记录时间</th><th>用户名称</th><th>登陆科室</th><th>日志内容</th><th>IP地址</th><th>MAC地址</th><th>程序版本号</th><th>结果</th><th>请求编号</th><th>错误代码</th></tr></thead>
        <tbody>
          <tr v-for="(row, index) in rows" :key="`${row.requestId}-${row.time}-${index}`">
            <td>{{ row.module || '—' }}</td>
            <td>{{ row.type || '—' }}</td>
            <td class="user-operations-time">{{ formatTime(row.time) }}</td>
            <td><span>{{ row.user || row.userId || '—' }}</span><small v-if="row.userId && row.user && row.userId !== row.user" class="user-operations-user-id">{{ row.userId }}</small></td>
            <td>{{ row.department || '—' }}</td>
            <td>{{ row.description || '—' }}</td>
            <td class="mono">{{ row.ip || '—' }}</td>
            <td class="mono">{{ row.mac || '—' }}</td>
            <td>{{ row.version || '—' }}</td>
            <td><span class="status-pill" :data-status="row.outcome">{{ row.outcome || '—' }}</span></td>
            <td class="mono">{{ row.requestId || '—' }}</td>
            <td class="mono">{{ row.errorCode || '—' }}</td>
          </tr>
        </tbody>
      </table>
      <div class="user-operations-pagination"><span>最多每次读取 {{ limit }} 条</span><button v-if="nextCursor" class="button button-ghost button-compact" type="button" :disabled="!canLoadMore" @click="loadPage({ append: true })">{{ loading ? '正在读取…' : '加载更多' }}</button><span v-else>没有更多记录</span></div>
    </div>
  </section>
</template>

<style scoped>
.user-operations-panel.access-data-panel.patient-panel.access-data-panel { --uol-shell-w: 1008px; gap: 0; padding: 0; border: 1px solid #999; border-radius: 0; background: #f0f0f0; color: #202020; font: 16px SimSun, '宋体', serif; box-sizing: border-box; }
.user-operations-panel.access-data-panel.patient-panel :deep(.master-panel-header) {
  --uol-hdr-left-return: 15px;
  --uol-hdr-left-title: 110px;
  --uol-hdr-left-search: 813px;
  --uol-hdr-left-export: 909px;
  --uol-hdr-right-search: 105px;
  --uol-hdr-right-export: 9px;
  --uol-hdr-top-return: 9px;
  --uol-hdr-top-title: 2px;
  --uol-hdr-top-action: 7px;
  position: relative;
  min-height: 45px;
  height: 45px;
  padding: 4px 12px;
  margin: 0;
  border-bottom: 1px solid #999;
  border-radius: 0;
  background: #e8e8e8;
  box-sizing: border-box;
}
.user-operations-panel.access-data-panel.patient-panel :deep(.master-panel-copy p) { display: none; }
.user-operations-panel.access-data-panel.patient-panel :deep(.master-panel-symbol) { width: 28px; height: 28px; }
.user-operations-panel.access-data-panel.patient-panel :deep(h2) { font: 16px SimSun, '宋体', serif; }
.user-operations-panel.access-data-panel.patient-panel .user-operations-filters {
  --uol-left-label-module: 108px;
  --uol-left-ctrl-module: 183px;
  --uol-left-label-user: 385px;
  --uol-left-ctrl-user: 431px;
  --uol-left-label-begin: 108px;
  --uol-left-ctrl-begin: 183px;
  --uol-left-label-sep: 385px;
  --uol-left-ctrl-end: 431px;
  --uol-top-r1: 6px;
  --uol-top-r2: 37px;
  position: relative;
  --uol-filters-h: 81px;
  --uol-ctrl-w: 196px;
  --uol-label-module-w: 62px;
  --uol-label-user-w: 36px;
  --uol-field-gap: 8px;
  --uol-group-gap: 12px;
  --uol-row-gap: 3px;
  --uol-filters-padding-x: 12px;
  display: grid;
  grid-template-columns: calc(var(--uol-label-module-w) + var(--uol-field-gap) + var(--uol-ctrl-w)) calc(var(--uol-label-user-w) + var(--uol-field-gap) + var(--uol-ctrl-w));
  grid-template-rows: 17px 25px 25px;
  align-items: center;
  align-content: start;
  gap: var(--uol-row-gap) var(--uol-group-gap);
  min-height: var(--uol-filters-h);
  padding: 3px var(--uol-filters-padding-x) 4px;
  margin: 0;
  border: 0;
  border-bottom: 1px solid #999;
  border-radius: 0;
  background: #f0f0f0;
  box-sizing: border-box;
}
.user-operations-panel.access-data-panel.patient-panel .user-operations-filters > .field { flex-direction: row; align-items: center; gap: var(--uol-field-gap); margin: 0; }
.user-operations-panel.access-data-panel.patient-panel .user-operations-filters > .field > span { flex: 0 0 var(--uol-label-module-w); box-sizing: border-box; white-space: nowrap; }
.user-operations-panel.access-data-panel.patient-panel .user-operations-filters > .user-operations-user-field > span { flex-basis: var(--uol-label-user-w); }
.user-operations-panel.access-data-panel.patient-panel .user-operations-filters > .field > select, .user-operations-user-button-edit, .user-operations-date-range input { flex: 0 0 var(--uol-ctrl-w); width: var(--uol-ctrl-w); box-sizing: border-box; }
.user-operations-panel.access-data-panel.patient-panel .field { display: flex; flex-direction: column; gap: 5px; }
.user-operations-panel.access-data-panel.patient-panel .field > span { align-self: flex-start; padding: 3px 5px; border: 1px solid #bfbfbf; border-radius: 5px; background: #dedede; font: 12px/17px 'Segoe UI', sans-serif; }
.user-operations-panel.access-data-panel.patient-panel .field input, .user-operations-panel.access-data-panel.patient-panel .field select { min-height: 25px; height: 25px; padding: 2px 5px; border: 1px solid #999; border-radius: 0; background: white; color: #202020; font: 16px SimSun, '宋体', serif; }
.user-operations-panel.access-data-panel.patient-panel .button { min-height: 30px; height: 30px; border: 1px solid #aaa; border-radius: 3px; background: #e6e6e6; color: #202020; font: 14px SimSun, '宋体', serif; }
.user-operations-panel.access-data-panel.patient-panel .user-operations-results { margin: 0; border: 0; border-radius: 0; background: white; }
.user-operations-panel.access-data-panel.patient-panel .register-table { border-collapse: collapse; font: 16px SimSun, '宋体', serif; }
.user-operations-panel.access-data-panel.patient-panel .register-table th { padding: 5px 8px; background: #e7e7e7; color: #202020; border: 1px solid #b7b7b7; font-weight: normal; }
.user-operations-panel.access-data-panel.patient-panel .register-table td { padding: 4px 8px; border: 1px solid #d4d4d4; color: #202020; }
.user-operations-panel.access-data-panel.patient-panel .register-table tbody tr:nth-child(even) { background: #f4f4f4; }
.user-operations-panel.access-data-panel.patient-panel .result-empty, .user-operations-panel.access-data-panel.patient-panel .shell-status { min-height: 160px; padding: 20px; background: white; }
.user-operations-filters-legend { grid-column: 1 / -1; margin: 0; padding: 0; color: #202020; font: 12px/17px 'Segoe UI', sans-serif; }
.user-operations-record-date { grid-column: 1 / -1; min-width: 0; }
.user-operations-date-range { display: grid; grid-template-columns: var(--uol-ctrl-w) calc(var(--uol-group-gap) + var(--uol-label-user-w) + var(--uol-field-gap)) var(--uol-ctrl-w); align-items: center; min-width: 0; }
.user-operations-date-range input { min-width: 0; }
.user-operations-date-sep { justify-self: center; color: #202020; font: 12px/17px 'Segoe UI', sans-serif; }
.user-operations-filters .field { min-width: 0; }
.user-operations-user-field { position: relative; }
.user-operations-user-button-edit { display: flex; min-width: 0; }
.user-operations-panel.access-data-panel.patient-panel .user-operations-user-button-edit input { flex: 1 1 0; min-width: 0; width: 0; }
.user-operations-user-button-edit > button { flex: 0 0 27px; height: 25px; padding: 0; border: 1px solid #999; border-left: 0; border-radius: 0; background: #e6e6e6; color: #202020; font: bold 16px SimSun, '宋体', serif; cursor: pointer; }
.user-operations-user-picker { position: absolute; top: 100%; left: 0; z-index: 20; width: min(360px, calc(100vw - 48px)); padding: 8px; border: 1px solid #999; background: #f0f0f0; box-shadow: 0 3px 8px #0003; color: #202020; box-sizing: border-box; font: 14px SimSun, '宋体', serif; }
.user-operations-user-picker-actions, .user-operations-user-picker-search { display: flex; align-items: center; gap: 6px; margin-bottom: 8px; }
.user-operations-user-picker-actions strong { flex: 1; }
.user-operations-user-picker-search input { flex: 1; min-width: 0; }
.user-operations-user-picker button { padding: 3px 8px; border: 1px solid #999; border-radius: 0; background: #e6e6e6; color: #202020; font: inherit; cursor: pointer; }
.user-operations-user-picker button:disabled { cursor: default; opacity: 0.6; }
.user-operations-user-picker-list { max-height: 220px; margin: 0; padding: 0; overflow-y: auto; list-style: none; }
.user-operations-user-picker-list button { width: 100%; margin-bottom: 3px; text-align: left; overflow-wrap: anywhere; }
.user-operations-user-picker-list button[aria-pressed='true'] { border-color: #666; background: #dedede; }
.user-operations-user-picker p { margin: 8px 0; overflow-wrap: anywhere; }
.user-operations-results { max-width: 100%; }
.user-operations-time { min-width: 155px; white-space: nowrap; }
.user-operations-user-id { display: block; color: #829286; font-size: 10px; }
.user-operations-panel.access-data-panel.patient-panel .mono { font-family: ui-monospace, SFMono-Regular, Consolas, monospace; font-size: 10px; overflow-wrap: anywhere; }
.user-operations-pagination { display: flex; align-items: center; justify-content: space-between; gap: 12px; padding: 12px 0 2px; color: #829286; font-size: 11px; }
@media (min-width: 861px) {
  .user-operations-panel.access-data-panel.patient-panel {
    width: var(--uol-shell-w);
    max-width: 100%;
  }
  .user-operations-panel.access-data-panel.patient-panel .user-operations-results,
  .user-operations-panel.access-data-panel.patient-panel .result-empty,
  .user-operations-panel.access-data-panel.patient-panel .shell-status {
    position: relative;
    left: 0;
    width: 100%;
    max-width: none;
    box-sizing: border-box;
    margin: 0;
  }
}
@media (min-width: 861px) {
  .user-operations-panel.access-data-panel.patient-panel :deep(.master-panel-header) { display: block; padding-left: 0; padding-right: 0; }
  .user-operations-panel.access-data-panel.patient-panel :deep(.master-panel-symbol),
  .user-operations-panel.access-data-panel.patient-panel :deep(.master-count) { display: none; }
  .user-operations-panel.access-data-panel.patient-panel :deep(.master-panel-copy) {
    position: absolute;
    left: var(--uol-hdr-left-title);
    top: var(--uol-hdr-top-title);
    flex: none;
    width: auto;
    max-width: none;
    min-width: 0;
  }
  .user-operations-panel.access-data-panel.patient-panel :deep(.master-panel-title) { margin: 0; }
  .user-operations-panel.access-data-panel.patient-panel :deep(.master-panel-copy h2) { margin: 0; line-height: 25px; white-space: nowrap; }
  .user-operations-panel.access-data-panel.patient-panel :deep(.master-header-actions) { display: contents; }
  .user-operations-panel.access-data-panel.patient-panel .uol-hdr-return,
  .user-operations-panel.access-data-panel.patient-panel .uol-hdr-search,
  .user-operations-panel.access-data-panel.patient-panel .uol-hdr-export {
    position: absolute;
    z-index: 1;
    box-sizing: border-box;
  }
  .user-operations-panel.access-data-panel.patient-panel .uol-hdr-return {
    left: var(--uol-hdr-left-return);
    top: var(--uol-hdr-top-return);
  }
  .user-operations-panel.access-data-panel.patient-panel .uol-hdr-search {
    right: var(--uol-hdr-right-search);
    top: var(--uol-hdr-top-action);
  }
  .user-operations-panel.access-data-panel.patient-panel .uol-hdr-export {
    right: var(--uol-hdr-right-export);
    top: var(--uol-hdr-top-action);
  }
}
@media (min-width: 861px) {
  .user-operations-panel.access-data-panel.patient-panel .user-operations-filters { display: block; height: var(--uol-filters-h); }
  .user-operations-panel.access-data-panel.patient-panel .user-operations-filters > .field, .user-operations-date-range { display: contents; }
  .user-operations-user-field { position: static; }
  .user-operations-filters .uol-abs-label-module, .user-operations-filters .uol-abs-ctrl-module,
  .user-operations-filters .uol-abs-label-user, .user-operations-filters .uol-abs-ctrl-user,
  .user-operations-filters .uol-abs-label-begin, .user-operations-filters .uol-abs-ctrl-begin,
  .user-operations-filters .uol-abs-label-sep, .user-operations-filters .uol-abs-ctrl-end,
  .user-operations-filters .uol-abs-legend { position: absolute; }
  .uol-abs-label-module { left: var(--uol-left-label-module); top: var(--uol-top-r1); }
  .uol-abs-ctrl-module { left: var(--uol-left-ctrl-module); top: var(--uol-top-r1); }
  .uol-abs-label-user { left: var(--uol-left-label-user); top: var(--uol-top-r1); }
  .uol-abs-ctrl-user { left: var(--uol-left-ctrl-user); top: var(--uol-top-r1); }
  .uol-abs-label-begin { left: var(--uol-left-label-begin); top: var(--uol-top-r2); }
  .uol-abs-ctrl-begin { left: var(--uol-left-ctrl-begin); top: var(--uol-top-r2); }
  .uol-abs-label-sep { left: var(--uol-left-label-sep); top: var(--uol-top-r2); }
  .uol-abs-ctrl-end { left: var(--uol-left-ctrl-end); top: var(--uol-top-r2); }
  .uol-abs-legend { left: 3px; top: 13px; }
  .user-operations-user-picker { left: var(--uol-left-ctrl-user); top: calc(var(--uol-top-r1) + 25px); }
}
@media (max-width: 860px) {
  .user-operations-panel.access-data-panel.patient-panel {
    width: auto;
    max-width: 100%;
  }
}
@media (max-width: 860px) {
  .user-operations-panel.access-data-panel.patient-panel :deep(.master-panel-header) { display: flex; padding: 4px 12px; }
  .user-operations-panel.access-data-panel.patient-panel .uol-hdr-return,
  .user-operations-panel.access-data-panel.patient-panel .uol-hdr-search,
  .user-operations-panel.access-data-panel.patient-panel .uol-hdr-export { position: static; }
}
@media (max-width: 860px) {
  .user-operations-panel.access-data-panel.patient-panel .user-operations-filters { grid-template-columns: repeat(2, minmax(0, 1fr)); grid-template-rows: auto; padding-top: 6px; padding-bottom: 10px; }
  .user-operations-panel.access-data-panel.patient-panel .user-operations-filters > .field { flex-direction: column; align-items: stretch; gap: 5px; }
  .user-operations-panel.access-data-panel.patient-panel .user-operations-filters > .field > span { flex: none; }
  .user-operations-panel.access-data-panel.patient-panel .user-operations-filters > .field > select, .user-operations-user-button-edit, .user-operations-date-range input { flex: 1 1 auto; width: 100%; }
  .user-operations-date-range { grid-template-columns: minmax(0, 1fr) auto minmax(0, 1fr); gap: var(--uol-field-gap); }
}
@media (max-width: 520px) {
  .user-operations-panel.access-data-panel.patient-panel :deep(.master-panel-header) { height: auto; min-height: 45px; }
  .user-operations-panel.access-data-panel.patient-panel .user-operations-filters { grid-template-columns: 1fr; }
  .user-operations-pagination { align-items: flex-start; flex-direction: column; }
}
</style>
