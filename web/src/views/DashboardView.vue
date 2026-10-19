<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { session } from '../auth/session.js'
import ModuleIcon from '../components/ModuleIcon.vue'
import { createDashboardRequestGate, fetchDashboardSummary } from '../api/dashboardApi.js'
import { formatYuan } from '../api/chargeContract.js'
import { decorateModules, modulePath } from '../workbench/catalog.js'
import {
  CHARGE_METRICS,
  PERIOD_OPTIONS,
  TODO_CARDS,
  VISIT_METRICS,
} from '../workbench/dashboard.js'

const period = ref(PERIOD_OPTIONS[0].value)
const refreshKey = ref(0)
const summary = ref(null)
const loading = ref(false)
const errorMessage = ref('')
const demoMode = ref(false)
const requestGate = createDashboardRequestGate()
const modules = computed(() => decorateModules(session.user))
const quickModules = computed(() => modules.value.filter((item) => item.allowed && item.status === 'available'))
const displayName = computed(() => session.user?.displayName || session.user?.username || '您')

async function loadSummary() {
  const requestId = requestGate.begin()
  const requestedPeriod = period.value
  loading.value = true
  errorMessage.value = ''
  summary.value = null
  demoMode.value = false
  try {
    const result = await fetchDashboardSummary(session.accessToken, requestedPeriod)
    if (!requestGate.isCurrent(requestId) || requestedPeriod !== period.value) return
    demoMode.value = result.source === 'mock'
    if (!demoMode.value) summary.value = result
  } catch (error) {
    if (!requestGate.isCurrent(requestId) || requestedPeriod !== period.value) return
    errorMessage.value = error && error.message ? error.message : '工作台汇总暂不可用'
  } finally {
    if (requestGate.isCurrent(requestId)) loading.value = false
  }
}

function canOpen(card) {
  if (!card.module || card.phase) return false
  const mod = modules.value.find((item) => item.key === card.module)
  return Boolean(mod && mod.allowed)
}

function cardTitle(card) {
  if (card.phase) return card.phase
  if (card.module && !canOpen(card)) return '无权限'
  return ''
}

function cardValue(card) {
  if (!summary.value || !card.valueKey || !canOpen(card)) return null
  if (card.module === 'charge' && !summary.value.available.charges) return null
  if (card.module === 'dispense' && !summary.value.available.dispenses) return null
  return summary.value[card.valueKey]
}

function metricValue(metric) {
  if (!summary.value) return '—'
  const value = summary.value[metric.key]
  return metric.format === 'money' ? formatYuan(value) : new Intl.NumberFormat('zh-CN').format(value)
}

function onRefresh() {
  refreshKey.value += 1
  loadSummary()
}

watch(period, loadSummary)
onMounted(loadSummary)
</script>

<template>
  <section class="dashboard" aria-labelledby="dash-title">
    <header class="dash-head dash-hero">
      <div class="dash-hero-copy">
        <p class="dash-eyebrow"><span></span> OUTPATIENT WORKSPACE</p>
        <h1 id="dash-title">工作台 <span>{{ displayName }}，欢迎回来</span></h1>
        <p class="dash-hero-message">把每一次诊疗，连接得更顺畅。</p>
        <p class="dash-hero-detail">从建档到发药，有序衔接每一步门诊业务。</p>
        <span class="hero-pill">专注诊疗 · 有序协作</span>
      </div>
      <div class="dash-hero-art" aria-hidden="true">
        <svg viewBox="0 0 380 180" focusable="false">
          <defs>
            <linearGradient id="dash-orbit" x1="0" y1="0" x2="1" y2="1">
              <stop offset="0%" stop-color="#8bd9c2" stop-opacity=".65" />
              <stop offset="100%" stop-color="#8bd9c2" stop-opacity="0" />
            </linearGradient>
          </defs>
          <circle class="hero-ring" cx="226" cy="91" r="71" fill="none" stroke="url(#dash-orbit)" />
          <circle cx="226" cy="91" r="51" fill="none" stroke="#8bd9c2" stroke-opacity=".13" stroke-dasharray="2 7" />
          <circle cx="226" cy="91" r="31" fill="#8bd9c2" fill-opacity=".08" stroke="#8bd9c2" stroke-opacity=".3" />
          <path d="M226 78v26M213 91h26" stroke="#a0e4d0" stroke-width="6" stroke-linecap="round" />
          <path class="hero-pulse" d="M0 105h78l13-20 16 40 16-68 19 47h39" fill="none" stroke="#8bd9c2" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" />
          <circle cx="155" cy="91" r="4" fill="#8bd9c2" />
          <circle cx="270" cy="35" r="4" fill="#8bd9c2" fill-opacity=".7" />
          <circle cx="278" cy="139" r="3" fill="#8bd9c2" fill-opacity=".5" />
          <path d="M18 161h78M304 20h43M323 24h17" stroke="#8bd9c2" stroke-opacity=".2" stroke-linecap="round" />
        </svg>
      </div>
    </header>

    <section class="quickflow" aria-labelledby="quickflow-title">
      <div class="dash-section-head">
        <div><h2 id="quickflow-title">门诊业务</h2><p>常用入口，触手可及</p></div>
        <RouterLink class="dash-text-link" :to="{ name: 'workbench' }">门诊总览 <span aria-hidden="true">↗</span></RouterLink>
      </div>
      <div v-if="quickModules.length" class="quickflow-track">
        <RouterLink v-for="(mod, index) in quickModules" :key="mod.key" class="quickflow-link" :to="mod.path" :style="{ '--flow-order': index }">
          <span class="quickflow-glyph"><ModuleIcon :name="mod.key" /></span>
          <span class="quickflow-copy"><strong>{{ mod.name }}</strong><span>{{ mod.blurb }}</span></span>
          <span class="quickflow-arrow" aria-hidden="true">↗</span>
        </RouterLink>
      </div>
      <p v-else class="quickflow-empty">当前角色未分配可用的门诊业务入口。</p>
    </section>

    <div class="dash-section-head todo-head">
      <div><h2 id="todo-title">待办</h2><p>当前待处理工作</p></div>
      <span class="dash-section-note">按角色进入对应业务</span>
    </div>
    <div class="todo-row" aria-labelledby="todo-title">
      <component
        :is="canOpen(card) ? 'RouterLink' : 'div'"
        v-for="card in TODO_CARDS"
        :key="card.key"
        class="todo-card"
        :class="{ 'is-muted': !canOpen(card) }"
        :to="canOpen(card) ? modulePath(card.module) : undefined"
        :title="cardTitle(card) || undefined"
        :aria-disabled="canOpen(card) ? undefined : 'true'"
      >
        <span class="todo-marker" :class="`todo-marker-${card.key}`" aria-hidden="true"></span>
        <span class="todo-label">
          {{ card.label }}
          <span v-if="card.phase" class="phase-tag">{{ card.phase }}</span>
        </span>
        <strong v-if="typeof cardValue(card) === 'number' && (card.module === 'charge' ? summary?.available.charges : card.module === 'dispense' ? summary?.available.dispenses : true)" class="todo-value">{{ cardValue(card) }}</strong>
        <span v-else class="todo-dash">—</span>
        <span class="todo-caption">{{ card.phase === '二期' ? '计划开放' : card.phase ? '阈值未配置' : !summary ? '暂无统计数据' : canOpen(card) ? '当前待办 · 进入业务' : '未分配业务权限' }}</span>
      </component>
    </div>

    <section class="ops" aria-labelledby="ops-title">
      <div class="ops-head">
        <div class="ops-title">
          <h2 id="ops-title">经营概况</h2>
          <p class="dash-flag">{{ demoMode ? '演示模式暂未接入汇总数据' : '待办为当前快照；其余数据按所选周期和当前权限范围统计' }}</p>
          <p v-if="errorMessage" class="dash-error" role="alert">{{ errorMessage }}</p>
        </div>
        <div class="period" role="group" aria-label="时间">
          <button
            v-for="item in PERIOD_OPTIONS"
            :key="item.value"
            class="period-btn"
            :class="{ 'is-on': item.value === period }"
            type="button"
            :aria-pressed="item.value === period"
            @click="period = item.value"
          >
            {{ item.label }}
          </button>
          <button class="period-btn period-icon" type="button" disabled title="二期" aria-label="选择日期，二期">
            <svg viewBox="0 0 24 24" aria-hidden="true" focusable="false">
              <rect x="4" y="5" width="16" height="15" rx="2" fill="none" stroke="currentColor" stroke-width="1.7" />
              <path d="M4 9.5h16M8 3.5v3M16 3.5v3" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" />
            </svg>
          </button>
          <button class="period-btn period-refresh" type="button" title="刷新汇总" :disabled="loading" @click="onRefresh"><svg :key="refreshKey" :class="{ 'is-refreshed': refreshKey > 0 }" viewBox="0 0 24 24" aria-hidden="true" focusable="false"><path d="M19 8a7.5 7.5 0 1 0 1 8M19 4v5h-5" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round" /></svg>{{ loading ? '加载中' : '刷新' }}</button>
        </div>
      </div>

      <div class="ops-metrics">
        <div class="metric-row">
          <div class="metric-name">
            <span class="metric-icon metric-icon-fee" aria-hidden="true">
              <svg viewBox="0 0 24 24" focusable="false">
                <circle cx="12" cy="12" r="8" fill="none" stroke="currentColor" stroke-width="1.7" />
                <path d="M12 7.2v9.6M9.4 9.4c.6-1 1.6-1.4 2.6-1.4 1.6 0 2.6.8 2.6 2s-1 1.9-2.6 2.2-2.6.7-2.6 2.1 1.1 2.1 2.8 2.1c1 0 2-.4 2.6-1.2" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" />
              </svg>
            </span>
            收费
          </div>
          <div v-for="(metric, index) in CHARGE_METRICS" :key="`fee-${metric.key}`" class="metric" :class="{ 'metric-primary': index === 0 }">
            <strong>{{ summary?.available.charges ? metricValue(metric) : '—' }}</strong>
            <span>{{ metric.label }}</span>
          </div>
        </div>

        <div class="metric-row">
          <div class="metric-name">
            <span class="metric-icon metric-icon-visit" aria-hidden="true">
              <svg viewBox="0 0 24 24" focusable="false">
                <path d="M4 16.5 9 11l3.2 3.2L20 7" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round" />
                <path d="M14.5 7H20v5.5" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round" />
              </svg>
            </span>
            客流
          </div>
          <div v-for="(metric, index) in VISIT_METRICS" :key="`visit-${metric.key}`" class="metric" :class="{ 'metric-primary': index === 0 }">
            <strong>{{ summary?.available.encounters ? metricValue(metric) : '—' }}</strong>
            <span>{{ metric.label }}</span>
          </div>
        </div>
      </div>
    </section>

  </section>
</template>

<style scoped>
.dashboard {
  gap: 0;
  flex: 1 1 auto;
  min-height: 0;
  overflow: hidden;
  max-width: 1540px;
  margin: 0 auto;
  color: var(--ink, #20394a);
}

.dash-hero {
  position: relative;
  display: flex;
  flex-wrap: nowrap;
  align-items: center;
  justify-content: space-between;
  gap: 32px;
  min-height: 216px;
  overflow: hidden;
  padding: 28px 34px;
  border: 1px solid #263f50;
  border-radius: 22px;
  background: radial-gradient(ellipse at 87% 70%, #275954 0%, transparent 50%), linear-gradient(114deg, #172f40, #203f4d);
  color: #fff;
  box-shadow: 0 12px 30px #183d4910;
  animation: dash-arrive .6s both;
}

.dash-hero-copy { position: relative; z-index: 1; }
.dash-eyebrow { display: flex; align-items: center; gap: 9px; color: #a4c8c3; font-size: 10px; font-weight: 600; letter-spacing: 2.2px; }
.dash-eyebrow span { width: 6px; height: 6px; border-radius: 50%; background: #8cdbc0; box-shadow: 0 0 0 4px #8cdbc012; }
.dash-head h1 { display: flex; align-items: baseline; flex-wrap: wrap; gap: 14px; margin-top: 14px; color: #fff; font-size: 30px; font-weight: 650; letter-spacing: -1px; }
.dash-head h1 span { color: #c6dadb; font-size: 14px; font-weight: 400; letter-spacing: 0; }
.dash-hero-message { margin-top: 12px; font-size: 16px; color: #e5f1ee; }
.dash-hero-detail { margin-top: 5px; color: #a9c0c3; font-size: 12px; }
.hero-pill { display: inline-flex; margin-top: 20px; padding: 5px 12px; border: 1px solid #8cdbc028; border-radius: 20px; background: #8cdbc009; color: #bddbd4; font-size: 10px; letter-spacing: 1px; }
.dash-hero-art { width: 380px; max-width: 39%; flex: none; opacity: .9; }
.dash-hero-art svg { display: block; width: 100%; }
.hero-ring { stroke-dasharray: 460; animation: dash-orbit-in 1.5s ease both; }
.hero-pulse { stroke-dasharray: 330; animation: dash-pulse-in 1.4s .15s ease both; }

.quickflow { margin-top: 20px; animation: dash-arrive .6s .07s both; }
.dash-section-head { display: flex; align-items: center; justify-content: space-between; flex-wrap: wrap; gap: 12px; margin-bottom: 14px; }
.dash-section-head > div { display: flex; align-items: baseline; gap: 12px; flex-wrap: wrap; }
.dash-section-head h2, .ops-head h2 { color: var(--ink, #20394a); font-size: 17px; font-weight: 650; letter-spacing: -.3px; }
.dash-section-head p, .dash-section-note { color: var(--muted, #71818b); font-size: 11px; }
.dash-text-link { display: inline-flex; align-items: center; gap: 10px; text-decoration: none; color: #297565; font-size: 12px; font-weight: 600; }
.dash-text-link span { transition: transform .2s; }
.dash-text-link:hover span { transform: translate(2px, -2px); }
.quickflow-track { display: flex; padding: 7px; border: 1px solid #dde6e5; border-radius: 16px; background: #fff; box-shadow: 0 3px 12px #1b394704; }
.quickflow-link { display: flex; position: relative; align-items: center; flex: 1 1 0; gap: 11px; min-width: 0; padding: 15px 13px; border-radius: 11px; color: inherit; text-decoration: none; transition: background .2s, transform .2s; animation: dash-arrive .5s calc(.12s + var(--flow-order, 0) * .045s) both; }
.quickflow-link + .quickflow-link::before { position: absolute; left: -1px; top: 22%; height: 56%; width: 1px; content: ''; background: #e6eceb; }
.quickflow-link:hover { background: #f0f7f4; transform: translateY(-2px); }
.quickflow-glyph { display: grid; place-items: center; flex: none; width: 40px; height: 40px; border: 1px solid #e6eeea; border-radius: 12px; background: #f2f7f4; color: #2c796b; }
.quickflow-glyph svg { width: 22px; height: 22px; }
.quickflow-copy { display: flex; min-width: 0; flex-direction: column; gap: 4px; }
.quickflow-copy strong { font-size: 13px; font-weight: 650; white-space: nowrap; }
.quickflow-copy > span { color: var(--muted, #71818b); font-size: 10px; line-height: 1.5; }
.quickflow-arrow { margin-left: auto; color: #91a69e; font-size: 14px; transition: transform .2s; }
.quickflow-link:hover .quickflow-arrow { transform: translate(2px, -2px); color: #2c796b; }
.quickflow-empty { padding: 20px; border: 1px solid #dde6e5; border-radius: 14px; color: var(--muted); font-size: 12px; }

.todo-head { margin-top: 20px; margin-bottom: 10px; animation: dash-arrive .6s .15s both; }
.todo-row { grid-template-columns: repeat(5, minmax(0, 1fr)); gap: 12px; animation: dash-arrive .6s .2s both; }
.todo-card { position: relative; display: grid; grid-template-columns: 1fr auto; align-items: center; gap: 8px; min-height: 101px; padding: 18px 20px; border: 1px solid #dde6e5; border-radius: 15px; background: #fff; color: var(--ink, #20394a); box-shadow: 0 3px 12px #1b394704; transition: transform .2s, box-shadow .2s, border-color .2s; }
a.todo-card:hover { border-color: #99c6b9; transform: translateY(-3px); box-shadow: 0 9px 20px #244b4e0b; }
.todo-card.is-muted { background: #ffffffa8; color: #677d85; }
.todo-marker { position: absolute; left: -1px; top: 24px; width: 3px; height: 21px; border-radius: 0 4px 4px 0; background: #6ba995; }
.todo-marker-dispense { background: #789da8; }
.todo-marker-upload, .todo-marker-standard { background: #ccd6d9; }
.todo-marker-stock { background: #c1b295; }
.todo-label { font-size: 12px; font-weight: 600; gap: 6px; }
.todo-value { color: #254b49; font-size: 28px; font-weight: 500; }
.todo-dash { font-size: 25px; color: #a8b6bb; }
.todo-caption { grid-column: 1 / -1; color: #7b8e95; font-size: 10px; }
.todo-card .phase-tag { padding: 2px 6px; border-radius: 5px; background: #edf2f1; color: #788b8f; font-size: 9px; font-weight: 400; }

.ops { display: grid; grid-template-rows: auto minmax(0, 1fr); flex: 1 1 auto; min-height: 0; margin-top: 12px; padding: 23px 24px 6px; border: 1px solid #dde6e5; border-radius: 18px; background: #fff; box-shadow: 0 3px 12px #1b394704; overflow: hidden; animation: dash-arrive .6s .25s both; }
.ops-head { justify-content: space-between; gap: 16px; padding-bottom: 20px; }
.ops-title { display: flex; flex-direction: column; gap: 6px; }
.ops-metrics { min-height: 0; overflow-y: auto; overflow-x: hidden; scrollbar-gutter: stable; }
.dash-flag { font-size: 10px; color: #849397; line-height: 1.6; }
.dash-error { color: #9a453e; font-size: 11px; line-height: 1.5; }
.period { flex-wrap: nowrap; gap: 3px; padding: 4px; border-radius: 10px; background: #f3f6f4; }
.period-btn { display: inline-flex; align-items: center; justify-content: center; gap: 6px; height: 31px; padding: 0 11px; border: 1px solid transparent; border-radius: 7px; background: transparent; color: #788b89; font-size: 11px; transition: background .2s, color .2s, box-shadow .2s; }
.period-btn:hover:enabled { color: #225b4e; background: #e5eeea; }
.period-btn.is-on { border-color: #e1e9e4; background: #fff; color: #285f50; box-shadow: 0 2px 4px #234e3910; }
.period-refresh { border-color: #d8e5dd; background: #e8f0eb; color: #3c6c57; }
.period-refresh svg { width: 14px; height: 14px; }
.period-refresh svg.is-refreshed { animation: dash-refresh .5s ease; }
.period-icon { width: 31px; padding: 0; }
.period-btn:disabled { opacity: .45; }
.metric-row { display: grid; grid-template-columns: 90px repeat(4, minmax(0, 1fr)); align-items: center; gap: 10px; padding: 21px 0; border-top: 1px solid #ecf0ed; }
.metric-row:last-child { grid-template-columns: 90px repeat(2, minmax(0, 1fr)); }
.metric-name { width: auto; gap: 9px; font-size: 12px; color: #526b72; font-weight: 600; }
.metric-icon { width: 32px; height: 32px; border-radius: 10px; }
.metric-icon svg { width: 17px; height: 17px; }
.metric-icon-fee { background: #f7f2e8; color: #a88d53; }
.metric-icon-visit { background: #ebf2f4; color: #6c909d; }
.metric { min-width: 0; gap: 4px; padding-left: 17px; border-left: 1px solid #edf1ee; }
.metric strong { color: #476068; font-size: 25px; font-weight: 450; line-height: 1.2; }
.metric span { font-size: 10px; color: #809093; }
.metric-primary strong { color: #2a7763; font-weight: 550; }
.quickflow-link:focus-visible, .dash-text-link:focus-visible, .todo-card:focus-visible, .period-btn:focus-visible { outline: 2px solid #428977; outline-offset: 4px; }
@keyframes dash-arrive { from { opacity: 0; transform: translateY(12px); } to { opacity: 1; transform: translateY(0); } }
@keyframes dash-orbit-in { from { stroke-dashoffset: 460; opacity: 0; } to { stroke-dashoffset: 0; opacity: 1; } }
@keyframes dash-pulse-in { from { stroke-dashoffset: 330; opacity: 0; } to { stroke-dashoffset: 0; opacity: 1; } }
@keyframes dash-refresh { to { transform: rotate(360deg); } }

@media (max-height: 800px) and (min-width: 981px) {
  .dash-hero { min-height: 154px; padding-block: 20px; }
  .dash-head h1 { margin-top: 8px; }
  .dash-hero-message { margin-top: 7px; }
  .hero-pill { margin-top: 11px; }
  .dash-hero-art { width: 330px; }
  .quickflow { margin-top: 12px; }
  .dash-section-head { margin-bottom: 8px; }
  .quickflow-link { gap: 9px; padding: 9px 8px; }
  .quickflow-glyph { width: 34px; height: 34px; }
  .todo-head { margin-top: 12px; margin-bottom: 8px; }
  .todo-card { min-height: 75px; padding-block: 10px; }
  .ops { margin-top: 10px; padding: 14px 20px 5px; }
  .ops-head { padding-bottom: 10px; }
  .metric-row { padding-block: 12px; }
}

@media (max-width: 1280px) {
  .quickflow-track { flex-wrap: wrap; gap: 4px; }
  .quickflow-link { flex: 1 1 calc(50% - 4px); }
  .quickflow-link + .quickflow-link::before { display: none; }
  .quickflow-copy > span { display: none; }
  .quickflow-link { gap: 9px; padding: 14px 10px; }
  .quickflow-arrow { display: none; }
  .todo-card { padding: 17px 15px; }
  .todo-label { font-size: 11px; }
  .metric-row { grid-template-columns: 70px repeat(4, minmax(0, 1fr)); }
  .metric-row:last-child { grid-template-columns: 70px repeat(2, minmax(0, 1fr)); }
  .metric { padding-left: 10px; }
  .metric-name { gap: 6px; }
}

@media (max-height: 800px) and (min-width: 981px) {
  .quickflow-track { display: grid; grid-template-columns: repeat(7, minmax(0, 1fr)); gap: 0; }
  .quickflow-link { gap: 5px; padding: 8px 5px; }
  .quickflow-glyph { width: 30px; height: 30px; border-radius: 9px; }
  .quickflow-glyph svg { width: 18px; height: 18px; }
  .quickflow-copy strong { font-size: 10px; }
  .quickflow-copy > span, .quickflow-arrow { display: none; }
}

@media (max-width: 1000px) {
  .todo-row { grid-template-columns: repeat(3, minmax(0, 1fr)); }
  .dash-hero { padding: 26px; }
  .metric-row { grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 18px 8px; }
  .metric-row:last-child { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .metric-name { grid-column: 1 / -1; }
  .ops-head { align-items: flex-start; }
}
@media (max-width: 720px) {
  .dash-hero { min-height: 210px; padding: 24px; border-radius: 17px; }
  .dash-head h1 { font-size: 26px; gap: 10px; }
  .dash-head h1 span { width: 100%; font-size: 12px; }
  .dash-hero-message { font-size: 14px; }
  .dash-hero-detail { font-size: 11px; }
  .dash-hero-art { position: absolute; right: -45px; bottom: -15px; width: 220px; max-width: none; opacity: .32; }
  .dash-eyebrow { font-size: 9px; }
  .hero-pill { margin-top: 17px; }
  .quickflow-track { flex-wrap: wrap; gap: 4px; }
  .quickflow-link { flex: 1 1 calc(50% - 4px); }
  .quickflow-link + .quickflow-link::before { display: none; }
  .quickflow-glyph { width: 35px; height: 35px; border-radius: 10px; }
  .dash-section-head > div { gap: 7px; }
  .dash-section-head p, .dash-section-note { font-size: 10px; }
  .dash-section-note { display: none; }
  .todo-row { grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 10px; }
  .todo-card { min-height: 93px; padding: 16px; }
  .ops { padding: 20px 17px 3px; }
  .period { flex-wrap: wrap; }
  .period-btn { padding: 0 10px; }
  .metric-row { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .metric-row:last-child { grid-template-columns: repeat(2, minmax(0, 1fr)); }
}
@media (prefers-reduced-motion: reduce) {
  *, *::before, *::after { animation: none !important; transition: none !important; }
}
</style>
