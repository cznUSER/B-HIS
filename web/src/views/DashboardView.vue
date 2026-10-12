<script setup>
// 2026-10-12 P1-H05-HOME-DASH 首页看板。时间筛选和刷新不改变占位数字。
import { computed, ref } from 'vue'
import { session } from '../auth/session.js'
import DonutPlaceholder from '../components/DonutPlaceholder.vue'
import { decorateModules, modulePath } from '../workbench/catalog.js'
import {
  CHARGE_METRICS,
  FEE_SLICES,
  PAY_SLICES,
  PERIODS,
  PLACEHOLDER_VALUE,
  TODO_CARDS,
  VISIT_METRICS,
  equalArcs,
} from '../workbench/dashboard.js'

const period = ref(PERIODS[0])
const modules = computed(() => decorateModules(session.user && session.user.roles))
const feeArcs = equalArcs(FEE_SLICES.length)
const payArcs = equalArcs(PAY_SLICES.length)

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

function onRefresh() {
  period.value = PERIODS[0]
}
</script>

<template>
  <section class="dashboard" aria-labelledby="dash-title">
    <header class="dash-head">
      <h1 id="dash-title">工作台</h1>
      <p class="dash-flag">演示 / 占位。数字固定为 0，切换时间或刷新都不会加载经营数据。</p>
    </header>

    <h2 id="todo-title" class="sr-only">待办</h2>
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
        <span class="todo-label">
          {{ card.label }}
          <span v-if="card.phase" class="phase-tag">{{ card.phase }}</span>
        </span>
        <strong v-if="card.value !== null" class="todo-value">{{ card.value }}</strong>
        <span v-else class="todo-dash">—</span>
      </component>
    </div>

    <section class="ops" aria-labelledby="ops-title">
      <div class="ops-head">
        <h2 id="ops-title">经营概况</h2>
        <div class="period" role="group" aria-label="时间">
          <button
            v-for="item in PERIODS"
            :key="item"
            class="period-btn"
            :class="{ 'is-on': item === period }"
            type="button"
            :aria-pressed="item === period"
            @click="period = item"
          >
            {{ item }}
          </button>
          <button class="period-btn period-icon" type="button" disabled title="二期" aria-label="选择日期，二期">
            <svg viewBox="0 0 24 24" aria-hidden="true" focusable="false">
              <rect x="4" y="5" width="16" height="15" rx="2" fill="none" stroke="currentColor" stroke-width="1.7" />
              <path d="M4 9.5h16M8 3.5v3M16 3.5v3" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" />
            </svg>
          </button>
          <button class="period-btn period-refresh" type="button" @click="onRefresh">刷新</button>
        </div>
      </div>

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
        <div v-for="label in CHARGE_METRICS" :key="`fee-${label}`" class="metric">
          <strong>{{ PLACEHOLDER_VALUE }}</strong>
          <span>{{ label }}</span>
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
        <div v-for="label in VISIT_METRICS" :key="`visit-${label}`" class="metric">
          <strong>{{ PLACEHOLDER_VALUE }}</strong>
          <span>{{ label }}</span>
        </div>
      </div>
    </section>

    <div class="pie-grid">
      <DonutPlaceholder title="收入来源分布" :slices="FEE_SLICES" :arcs="feeArcs" />
      <DonutPlaceholder title="支付方式分布" :slices="PAY_SLICES" :arcs="payArcs" />
    </div>
  </section>
</template>
