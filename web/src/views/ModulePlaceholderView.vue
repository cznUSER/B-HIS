<script setup>
// 2026-10-12 P1-H05-HOME 模块空态：接口未接时不展示假数据。
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { session } from '../auth/session.js'
import BusinessPageHeader from '../components/BusinessPageHeader.vue'
import BusinessEmptyState from '../components/BusinessEmptyState.vue'
import { decorateModules, findModule } from '../workbench/catalog.js'
import { outpatientTarget } from '../workbench/context.js'

const route = useRoute()

const mod = computed(() => {
  const found = findModule(String(route.params.module || ''))
  if (!found) return null
  return decorateModules(session.user).find((item) => item.key === found.key) || null
})
const canCharge = computed(() => decorateModules(session.user).find((item) => item.key === 'charge')?.allowed)
const canDispense = computed(() => decorateModules(session.user).find((item) => item.key === 'dispense')?.allowed)
</script>

<template>
  <section v-if="mod" class="module-page" :aria-labelledby="`module-${mod.key}`">
    <p class="crumb">
      <RouterLink :to="{ name: 'home' }">首页</RouterLink>
      <span aria-hidden="true"> / </span>
      <RouterLink :to="{ name: 'workbench' }">门诊业务</RouterLink>
      <span aria-hidden="true"> / </span>
      <span>{{ mod.name }}</span>
    </p>

    <BusinessPageHeader :title-id="`module-${mod.key}`" :title="mod.name" :icon="mod.key" eyebrow="门诊业务" :description="mod.blurb" />
    <BusinessEmptyState v-if="!mod.allowed" title="无权限" description="当前账号不能进入此模块。可返回工作台查看当前角色可用的业务。" :icon="mod.key" eyebrow="访问权限">
      <RouterLink class="button button-ghost button-compact" :to="{ name: 'home' }">返回首页</RouterLink>
    </BusinessEmptyState>
    <BusinessEmptyState v-else title="独立办理入口正在完善" description="接口就绪后开放。退药和退费目前可分别从发药记录、收费单详情办理。" :icon="mod.key" eyebrow="办理指引">
      <RouterLink v-if="canDispense" class="button button-compact" :to="outpatientTarget('dispense', route.query, { view: 'issued' })">前往发药记录</RouterLink>
      <RouterLink v-if="canCharge" class="button button-ghost button-compact" :to="outpatientTarget('charge', route.query, { view: 'records' })">前往收费查询</RouterLink>
    </BusinessEmptyState>
    <section v-if="mod.allowed" class="refund-guide" aria-label="办理顺序">
      <article><span>01</span><div><h2>找到原业务单据</h2><p>在发药记录或已收查询中，核对患者和原单信息。</p></div></article>
      <article><span>02</span><div><h2>已发药品先退药</h2><p>从发药记录发起退药，填写数量和原因。</p></div></article>
      <article><span>03</span><div><h2>核对可退明细</h2><p>从收费单详情办理退费，已发药品须先完成退药。</p></div></article>
    </section>
  </section>
</template>

<style scoped>
.module-page { display: flex; flex-direction: column; gap: 20px; }
.refund-guide { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 18px; }
.refund-guide article { display: flex; align-items: flex-start; gap: 15px; padding: 24px; border: 1px solid #e1ebe3; border-radius: 16px; background: #ffffffaa; }
.refund-guide article > span { display: grid; place-items: center; flex: none; width: 32px; height: 32px; border-radius: 10px; color: #709f84; background: #edf5ef; font-size: 11px; }
.refund-guide h2 { color: #4e695a; font-size: 13px; font-weight: 600; }
.refund-guide p { margin-top: 9px; color: #8b9e91; font-size: 12px; line-height: 1.8; }
@media (max-width: 1100px) { .refund-guide { grid-template-columns: 1fr; gap: 12px; } }
</style>
