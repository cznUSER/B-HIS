<script setup>
// 2026-10-16 患者查询独立承接检索、档案详情与去挂号。
import { computed, reactive, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { ApiError } from '../api/errors.js'
import { fetchPatient, searchPatients } from '../api/patientApi.js'
import { fareLabel, sexLabel } from '../api/patientContract.js'
import { session } from '../auth/session.js'
import BusinessPageHeader from '../components/BusinessPageHeader.vue'
import BusinessEmptyState from '../components/BusinessEmptyState.vue'
import { recordAction, setPatient } from '../workbench/outpatientContext.js'
import './clinical-design.css'

const route = useRoute()
const searchForm = reactive({ name: '', idNumber: '', patientNo: '', hospPkNo: '' })
const results = ref([])
const querying = ref(false)
const opening = ref(false)
const searchError = ref('')
const detailError = ref('')
const current = ref(null)
const activeId = ref('')

const registerTo = computed(() => current.value
  ? { name: 'outpatient-module', params: { module: 'register' }, query: { patientId: current.value.patientId } }
  : { name: 'outpatient-module', params: { module: 'register' } })

function readQuery(key) {
  const raw = route.query[key]
  const value = Array.isArray(raw) ? raw[0] : raw
  return typeof value === 'string' ? value.trim() : ''
}

async function onSearch() {
  if (querying.value) return
  querying.value = true
  results.value = []
  searchError.value = ''
  try {
    let offset = 0
    while (true) {
      const found = await searchPatients(session.accessToken, { ...searchForm }, { offset })
      results.value.push(...found.items)
      if (found.nextOffset === null) break
      if (found.nextOffset <= offset) throw new Error('患者列表分页位置未前进')
      offset = found.nextOffset
    }
  } catch (error) {
    searchError.value = error instanceof ApiError ? error.message : '查询失败，请稍后重试'
  } finally {
    querying.value = false
  }
}

async function openPatient(patientId) {
  if (!patientId || opening.value) return
  opening.value = true
  detailError.value = ''
  try {
    const patient = await fetchPatient(session.accessToken, patientId)
    current.value = patient
    activeId.value = patient.patientId
    setPatient(patient)
    recordAction('患者档案已打开', patient.name || patient.patientNo || patient.patientId || '')
  } catch (error) {
    current.value = null
    activeId.value = ''
    detailError.value = error instanceof ApiError ? error.message : '未找到患者'
  } finally {
    opening.value = false
  }
}

watch(() => route.fullPath, () => {
  for (const key of Object.keys(searchForm)) searchForm[key] = readQuery(key)
  searchError.value = ''
  const patientId = readQuery('patientId')
  void onSearch()
  if (patientId) void openPatient(patientId)
}, { immediate: true })
</script>

<template>
  <section class="patient-page" aria-labelledby="patient-search-title">
    <p class="crumb">
      <RouterLink :to="{ name: 'home' }">首页</RouterLink>
      <span aria-hidden="true"> / </span>
      <span>档案管理</span>
      <span aria-hidden="true"> / </span>
      <span>患者查询</span>
    </p>

    <BusinessPageHeader class="patient-search-header" title-id="patient-search-title" title="患者查询" eyebrow="PATIENT SEARCH" icon="patient" description="按姓名、证件号、患者号或院内档案号查找患者。" />

    <section class="patient-panel clinical-search-panel patient-query-panel" data-testid="patient-search">
      <div class="patient-query-head">
        <div>
          <span class="clinical-kicker">查找已有档案</span>
          <h2>患者查询</h2>
        </div>
        <div class="patient-query-meta">
          <span>任填一项即可检索</span>
          <span class="clinical-badge">档案管理</span>
        </div>
      </div>
      <form class="clinical-search-form" @submit.prevent="onSearch">
        <label class="field">
          <span>姓名</span>
          <input v-model="searchForm.name" name="search-name" autocomplete="off" placeholder="支持姓名中的连续文字" />
        </label>
        <label class="field">
          <span>身份证号</span>
          <input v-model="searchForm.idNumber" name="search-id" autocomplete="off" placeholder="完整证件号" />
        </label>
        <label class="field">
          <span>患者号</span>
          <input v-model="searchForm.patientNo" name="search-no" autocomplete="off" placeholder="完整患者号" />
        </label>
        <label class="field">
          <span>院内档案号</span>
          <input v-model="searchForm.hospPkNo" name="search-hosp-no" autocomplete="off" placeholder="完整院内档案号" />
        </label>
        <button class="button button-compact" type="submit" :disabled="querying">
          {{ querying ? '正在查询…' : '查询患者' }}
        </button>
      </form>
    </section>

    <div class="patient-layout clinical-search-workbench">
      <section class="patient-panel clinical-results-panel" aria-labelledby="patient-results-title">
        <div class="panel-head">
          <div><span class="clinical-kicker">患者档案</span><h2 id="patient-results-title">全部患者档案</h2></div>
          <span class="clinical-result-count" aria-live="polite">{{ querying ? '正在加载…' : `已显示 ${results.length} 条` }}</span>
        </div>
        <div class="result-wrap clinical-search-results">
          <p v-if="searchError" class="banner" role="alert">{{ searchError }}</p>
          <p v-else-if="querying" class="shell-status" role="status">正在加载患者档案…</p>
          <BusinessEmptyState v-else-if="results.length === 0" title="暂无患者档案。" description="当前没有符合条件的患者记录。" icon="patient" />
          <div v-else class="table-scroll" tabindex="0" role="region" aria-label="患者查询结果，可左右滚动">
            <table class="patient-table">
              <caption class="sr-only">患者查询结果</caption>
              <thead>
                <tr><th>患者号</th><th>姓名</th><th>性别</th><th>出生日期</th><th>身份证号</th><th>费别</th><th>手机号</th></tr>
              </thead>
              <tbody>
                <tr
                  v-for="item in results"
                  :key="item.patientId"
                  :class="{ 'is-active': item.patientId === activeId }"
                  tabindex="0"
                  @click="openPatient(item.patientId)"
                  @keydown.enter.prevent="openPatient(item.patientId)"
                >
                  <td>{{ item.patientNo || '—' }}</td>
                  <td><strong class="clinical-table-name">{{ item.name }}</strong><span v-if="item.patientId === activeId" class="clinical-selected">已选</span></td>
                  <td>{{ sexLabel(item.sex) }}</td>
                  <td>{{ item.birthday || '—' }}</td>
                  <td>{{ item.idNumber || '—' }}</td>
                  <td><span class="clinical-fare-tag">{{ fareLabel(item.fareIdentity) }}</span></td>
                  <td>{{ item.phone || '—' }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </section>

      <section class="patient-panel clinical-detail-panel" aria-labelledby="patient-detail-title" data-testid="patient-detail">
        <div class="panel-head">
          <div><span class="clinical-kicker">患者档案</span><h2 id="patient-detail-title">档案详情</h2></div>
          <span v-if="current" class="clinical-badge">已读取</span>
        </div>
        <p v-if="opening" class="shell-status" role="status">正在读取档案…</p>
        <p v-else-if="detailError" class="banner" role="alert">{{ detailError }}</p>
        <BusinessEmptyState v-else-if="!current" title="选择一位患者" description="查询后选择结果，在这里核对完整档案。" icon="patient" />
        <template v-else>
          <div class="clinical-profile-identity">
            <span class="clinical-profile-avatar" aria-hidden="true">{{ current.name?.slice(0, 1) || '患' }}</span>
            <div><strong>{{ current.name }}</strong><span>{{ current.patientNo || '患者号未提供' }}</span></div>
            <span class="clinical-detail-state">已读取</span>
          </div>
          <table class="clinical-detail-table">
            <tbody>
              <tr><th>姓名</th><td>{{ current.name }}</td></tr>
              <tr><th>性别</th><td>{{ sexLabel(current.sex) }}</td></tr>
              <tr><th>出生日期</th><td>{{ current.birthday || '—' }}</td></tr>
              <tr><th>身份证号</th><td>{{ current.idNumber || '—' }}</td></tr>
              <tr><th>费别</th><td>{{ fareLabel(current.fareIdentity) }}</td></tr>
              <tr><th>手机号</th><td>{{ current.phone || '—' }}</td></tr>
              <tr><th>院内档案号</th><td>{{ current.hospPkNo || '—' }}</td></tr>
              <tr><th>联系地址</th><td>{{ current.address || '—' }}</td></tr>
            </tbody>
          </table>
          <div class="clinical-detail-footer"><span>身份信息已读取，可继续门诊流程。</span><RouterLink class="clinical-link-button" :to="registerTo">去挂号　→</RouterLink></div>
        </template>
      </section>
    </div>
  </section>
</template>

<style scoped>
.patient-query-panel { display: block; }
.patient-query-head { display: flex; align-items: center; justify-content: space-between; gap: 16px; margin-bottom: 14px; }
.patient-query-head h2 { margin: 4px 0 0; color: #294a3d; font-size: 17px; font-weight: 650; }
.patient-query-meta { display: flex; align-items: center; gap: 12px; color: #899c92; font-size: 11px; }
.clinical-search-form { display: grid; grid-template-columns: minmax(130px, 1.05fr) minmax(170px, 1.55fr) minmax(130px, 1.05fr) minmax(160px, 1.15fr) auto; gap: 12px; align-items: end; }
.clinical-search-form .field { min-width: 0; }
.clinical-search-form .button { min-width: 104px; white-space: nowrap; }
.clinical-search-workbench { grid-template-columns: minmax(0, 1.35fr) minmax(330px, .78fr); align-items: start; gap: 18px; }
.clinical-results-panel, .clinical-detail-panel { min-width: 0; }
.clinical-result-count { color: #899c92; font-size: 12px; }
.clinical-profile-identity { display: flex; align-items: center; gap: 12px; min-width: 0; padding-block: 6px 16px; border-bottom: 1px solid #d9e8e1; }
.clinical-profile-identity > div { display: flex; min-width: 0; flex-direction: column; gap: 3px; }
.clinical-profile-identity strong { overflow-wrap: anywhere; color: #294e3e; font-size: 18px; font-weight: 650; }
.clinical-profile-identity span:not(.clinical-profile-avatar):not(.clinical-detail-state) { color: #8fa497; font-size: 12px; }
.clinical-profile-avatar { display: grid; place-items: center; width: 45px; height: 45px; flex: none; border: 1px solid #d2e6d9; border-radius: 13px; color: #4c947a; background: #e5f2e9; font-size: 20px; font-weight: 600; }
.clinical-detail-state { margin-left: auto; padding: 5px 9px; border-radius: 20px; color: #168b80; background: #e6f4ee; font-size: 11px; font-weight: 700; white-space: nowrap; }
.clinical-detail-table { width: 100%; margin-top: 14px; border: 1px solid #d9e8e1; border-radius: 9px; border-collapse: separate; border-spacing: 0; overflow: hidden; }
.clinical-detail-table th, .clinical-detail-table td { padding: 10px 12px; border-bottom: 1px solid #e8f0ec; text-align: left; }
.clinical-detail-table tr:last-child th, .clinical-detail-table tr:last-child td { border-bottom: 0; }
.clinical-detail-table th { width: 33%; color: #78948a; background: #f7fbf9; font-size: 11px; font-weight: 600; }
.clinical-detail-table td { color: #31554c; font-size: 13px; font-weight: 600; overflow-wrap: anywhere; }
.clinical-fare-tag { display: inline-flex; align-items: center; min-height: 23px; padding: 0 8px; border-radius: 12px; color: #327566; background: #e6f5ee; font-size: 11px; }
.clinical-detail-footer { display: flex; align-items: center; justify-content: space-between; gap: 12px; margin-top: 15px; color: #91a398; font-size: 11px; }
.clinical-link-button { flex: none; padding: 0; border: 0; background: transparent; color: #09675e; font: inherit; font-weight: 700; text-decoration: none; }
.clinical-link-button:hover { color: #0e8478; }
.table-scroll { max-width: 100%; overflow-x: auto; }
@media (max-width: 1100px) {
  .clinical-search-form { grid-template-columns: repeat(4, minmax(0, 1fr)); }
  .clinical-search-form .button { grid-column: 4; }
}
@media (max-width: 820px) {
  .patient-query-head { align-items: flex-start; flex-direction: column; }
  .clinical-search-workbench { grid-template-columns: 1fr; }
}
@media (max-width: 600px) {
  .clinical-search-form { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .clinical-search-form .button { grid-column: auto; }
  .clinical-detail-footer { align-items: flex-start; flex-direction: column; }
  .clinical-detail-state { margin-left: 0; }
}
</style>
