<script setup>
// 2026-10-12 P1-H05-HOME 门诊一期工作台
// 2026-10-12 P1-H05-HOME-DASH 六卡改为 /app/outpatient 总览，不再是登录默认页
import { computed, ref } from 'vue'
import { session } from '../auth/session.js'
import ModuleIcon from '../components/ModuleIcon.vue'
import OutpatientQueue from '../components/outpatient/OutpatientQueue.vue'
import BusinessPageHeader from '../components/BusinessPageHeader.vue'
import { decorateModules } from '../workbench/catalog.js'

const TIP_KEY = 'bhis.workbench.tip.v1'

const cards = computed(() => decorateModules(session.user).filter((item) => item.key !== 'patient'))
const availableCount = computed(() => cards.value.filter((card) => card.allowed && card.status === 'available').length)
const tipVisible = ref(readTip())
const moduleDescriptions = {
  register: '选择本次就诊科室和医生',
  diagnosis: '录入诊断并提交诊断记录',
  order: '录入项目，暂存或提交处方',
  charge: '核对明细，完成现金结算',
  dispense: '核对药品与库存，确认发药',
  refund: '按原业务单办理退药或退费',
}

function readTip() {
  try {
    return sessionStorage.getItem(TIP_KEY) !== '1'
  } catch {
    return true
  }
}

function dismissTip() {
  tipVisible.value = false
  try {
    sessionStorage.setItem(TIP_KEY, '1')
  } catch {
    // 关不掉存储时，本次仍先收起提示。
  }
}
</script>

<template>
  <section class="workbench" aria-labelledby="outpatient-title">
    <p class="crumb">
      <RouterLink :to="{ name: 'home' }">首页</RouterLink>
      <span aria-hidden="true"> / </span>
      <span>门诊业务</span>
    </p>
    <div v-if="tipVisible" class="tip" role="note">
      <p>收费前须有挂号/就诊 · 支付演示现金 · 已发药先退药再退费</p>
      <button class="tip-close" type="button" aria-label="关闭提示" @click="dismissTip">×</button>
    </div>

    <BusinessPageHeader title-id="outpatient-title" title="门诊业务" eyebrow="诊疗流程 · 全程衔接" icon="register" description="每个环节各司其职，让患者信息与诊疗业务有序流转。" />

    <OutpatientQueue />

    <div class="overview-flow">
      <div><span class="overview-section-tag">WORKFLOW</span><h2>门诊工作流程</h2><span class="overview-access"><strong>{{ availableCount }}</strong> 个可用工作区</span></div>
      <p class="flow">挂号 → 诊断 → 开单 → 收费 → 发药 → 退药/退费</p>
    </div>

    <div class="module-grid">
      <article
        v-for="(mod, index) in cards"
        :key="mod.key"
        class="module-card"
        :class="{ 'is-locked': !mod.allowed, 'is-recommended': mod.recommended }"
        :style="{ '--module-order': index }"
      >
        <div class="card-kicker">
          <span class="overview-step">{{ String(index + 1).padStart(2, '0') }}</span>
          <span class="glyph">
            <ModuleIcon :name="mod.key" />
          </span>
          <span class="status" :class="`status-${mod.status}`">{{ mod.statusLabel }}</span>
        </div>
        <div class="name-row">
          <h2>{{ mod.name }}</h2>
          <span v-if="mod.recommended" class="recommend">推荐</span>
        </div>
        <p class="blurb">{{ mod.blurb }}</p>
        <p class="overview-context">{{ moduleDescriptions[mod.key] }}</p>
        <div class="card-foot">
          <p v-if="mod.reason" class="reason">{{ mod.reason }}</p>
          <RouterLink v-if="mod.allowed" class="button button-enter" :to="mod.path">{{ mod.status === 'available' ? '进入工作区' : '查看办理入口' }}<span aria-hidden="true">↗</span></RouterLink>
          <button v-else class="button button-enter" type="button" disabled :title="mod.reason">进入</button>
        </div>
      </article>
    </div>
  </section>
</template>

<style scoped>
.shell-main .workbench { display: flex; flex-direction: column; gap: 20px; max-width: 1480px; margin: 0 auto; }
.shell-main .workbench .crumb { margin-bottom: 0; }
.shell-main .workbench .tip { order: 4; margin: 0; color: #73887d; border-color: #dde9e1; background: #f5f9f5; }
.overview-access { display: flex; align-items: baseline; gap: 8px; color: #749180; font-size: 12px; padding: 10px 16px; border: 1px solid #dce9df; border-radius: 12px; background: #ffffffad; }
.overview-access strong { color: #2e8067; font-size: 25px; font-weight: 550; }
.overview-flow { display: flex; align-items: center; justify-content: space-between; flex-wrap: wrap; gap: 14px; margin-top: 8px; }
.overview-flow > div { display: flex; align-items: baseline; gap: 12px; }
.overview-section-tag { color: #8fa697; font-size: 9px; letter-spacing: .1em; }
.overview-flow h2 { color: #395349; font-size: 16px; font-weight: 600; }
.shell-main .workbench .overview-flow .flow { margin: 0; font-size: 11px; color: #849a8c; }
.shell-main .workbench .module-grid { grid-template-columns: repeat(3, minmax(0, 1fr)); margin: 0; gap: 20px; }
.shell-main .workbench .module-card { position: relative; padding: 26px; gap: 12px; min-height: 284px; overflow: hidden; animation: module-arrive .45s calc(var(--module-order) * .04s) both; }
.shell-main .workbench .card-kicker { margin-bottom: 12px; }
.overview-step { position: absolute; right: 20px; bottom: 53px; color: #e9f0eb; font-size: 74px; font-weight: 650; line-height: 1; letter-spacing: -.06em; pointer-events: none; }
.shell-main .workbench .module-card .glyph { width: 48px; height: 48px; border-radius: 15px; background: #eaf4ec; color: #46896e; }
.shell-main .workbench .module-card:nth-child(2) .glyph, .shell-main .workbench .module-card:nth-child(5) .glyph { background: #eaf2f5; color: #5f8a99; }
.shell-main .workbench .module-card:nth-child(4) .glyph { background: #f5f1e7; color: #a1905f; }
.shell-main .workbench .module-card h2 { color: #354f46; font-size: 18px; font-weight: 600; }
.shell-main .workbench .blurb { min-height: 0; color: #718779; font-size: 12px; position: relative; }
.overview-context { max-width: 19em; color: #98a69e; font-size: 11px; line-height: 1.8; position: relative; }
.shell-main .workbench .card-foot { padding-top: 15px; position: relative; }
.shell-main .workbench .button-enter { height: 39px; justify-content: space-between; padding: 0 14px; background: #f1f7f2; color: #3a7e64; border: 1px solid #dceade; box-shadow: none; font-size: 12px; }
.shell-main .workbench .button-enter:hover { background: #e4f2e9; }
.shell-main .workbench .status { height: 23px; border-radius: 7px; font-size: 10px; font-weight: 500; }
.shell-main .workbench .module-card.is-locked .overview-context { color: #a2ada6; }
.shell-main .workbench .module-card.is-locked .button-enter { color: #9ba9a0; background: #edf1ee; border-color: #e2e8e3; }
@keyframes module-arrive { from { opacity: 0; transform: translateY(12px); } to { opacity: 1; transform: none; } }
@media (max-width: 1100px) { .shell-main .workbench .module-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); } }
@media (max-width: 760px) { .shell-main .workbench .module-grid { grid-template-columns: 1fr; gap: 14px; } .shell-main .workbench .module-card { min-height: 254px; padding: 22px; } .overview-flow .flow { line-height: 1.8; } }
@media (prefers-reduced-motion: reduce) { .shell-main .workbench .module-card { animation: none; } }
</style>

