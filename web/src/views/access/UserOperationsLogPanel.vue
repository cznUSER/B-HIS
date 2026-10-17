<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { ApiError } from '../../api/errors.js'
import { session } from '../../auth/session.js'
import { fetchAccessModules } from '../../master/api.js'
import { buildUserOperationsCsv, fetchUserOperationLogs } from '../../master/accessAuditApi.js'
import MasterPanelHeader from '../master/MasterPanelHeader.vue'

const rows = ref([])
const nextCursor = ref('')
const modules = ref([])
const filters = reactive({ moduleId: '', userId: '', begin: '', end: '' })
const loading = ref(false)
const exporting = ref(false)
const ready = ref(false)
const error = ref('')
const limit = 50

const canLoadMore = computed(() => ready.value && Boolean(nextCursor.value) && !loading.value)
const canExport = computed(() => ready.value && rows.value.length > 0 && !loading.value && !exporting.value)

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
    <MasterPanelHeader title="用户操作日志" title-id="user-operations-heading" icon="overview" description="查询账号与权限相关的操作记录。" :count="ready ? rows.length : null" count-label="已加载记录" :loading="loading">
      <template #actions><RouterLink class="button button-ghost button-compact" :to="{ name: 'access-users' }">返回</RouterLink><button class="button button-compact" type="submit" form="user-log-search" :disabled="loading">{{ loading ? '正在查询…' : '检索(S)' }}</button><button class="button button-ghost button-compact" type="button" :disabled="!canExport" title="导出已加载" @click="exportCsv">{{ exporting ? '正在导出…' : '导出(E)' }}</button></template>
    </MasterPanelHeader>

    <form id="user-log-search" class="master-filters user-operations-filters" @submit.prevent="search">
      <label class="field"><span>模块名称</span><select v-model="filters.moduleId"><option value="">全部模块</option><option v-for="item in modules" :key="item.id" :value="String(item.id)">{{ item.name }}（{{ item.id }}）</option></select></label>
      <label class="field"><span>用户</span><input v-model.trim="filters.userId" type="search" maxlength="64" placeholder="精确匹配用户编号" aria-label="用户编号" /></label>
      <label class="field"><span>开始日期</span><input v-model="filters.begin" type="date" /></label>
      <label class="field"><span>结束日期</span><input v-model="filters.end" type="date" /></label>
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
.user-operations-panel.access-data-panel.patient-panel.access-data-panel { gap: 0; padding: 0; border: 1px solid #999; border-radius: 0; background: #f0f0f0; color: #202020; font: 16px SimSun, '宋体', serif; }
.user-operations-panel.access-data-panel.patient-panel :deep(.master-panel-header) { min-height: 45px; height: 45px; padding: 4px 12px; margin: 0; border-bottom: 1px solid #999; border-radius: 0; background: #e8e8e8; box-sizing: border-box; }
.user-operations-panel.access-data-panel.patient-panel :deep(.master-panel-copy p) { display: none; }
.user-operations-panel.access-data-panel.patient-panel :deep(.master-panel-symbol) { width: 28px; height: 28px; }
.user-operations-panel.access-data-panel.patient-panel :deep(h2) { font: 16px SimSun, '宋体', serif; }
.user-operations-panel.access-data-panel.patient-panel .user-operations-filters { display: grid; grid-template-columns: repeat(4, minmax(130px, 1fr)) 90px; align-items: end; gap: 8px; min-height: 81px; padding: 6px 12px 10px; margin: 0; border: 0; border-bottom: 1px solid #999; border-radius: 0; background: #f0f0f0; box-sizing: border-box; }
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
.user-operations-filters .field { min-width: 0; }
.user-operations-results { max-width: 100%; }
.user-operations-time { min-width: 155px; white-space: nowrap; }
.user-operations-user-id { display: block; color: #829286; font-size: 10px; }
.user-operations-panel.access-data-panel.patient-panel .mono { font-family: ui-monospace, SFMono-Regular, Consolas, monospace; font-size: 10px; overflow-wrap: anywhere; }
.user-operations-pagination { display: flex; align-items: center; justify-content: space-between; gap: 12px; padding: 12px 0 2px; color: #829286; font-size: 11px; }
@media (max-width: 860px) {
  .user-operations-panel.access-data-panel.patient-panel .user-operations-filters { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .user-operations-filters > button { width: 100%; }
}
@media (max-width: 520px) {
  .user-operations-panel.access-data-panel.patient-panel :deep(.master-panel-header) { height: auto; min-height: 45px; }
  .user-operations-panel.access-data-panel.patient-panel .user-operations-filters { grid-template-columns: 1fr; }
  .user-operations-pagination { align-items: flex-start; flex-direction: column; }
}
</style>
