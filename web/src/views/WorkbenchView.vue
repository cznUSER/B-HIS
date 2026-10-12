<script setup>
// 2026-10-12 P1-H05-HOME 门诊一期工作台
import { computed, ref } from 'vue'
import { session } from '../auth/session.js'
import ModuleIcon from '../components/ModuleIcon.vue'
import { decorateModules } from '../workbench/catalog.js'

const TIP_KEY = 'bhis.workbench.tip.v1'

const cards = computed(() => decorateModules(session.user && session.user.roles))
const tipVisible = ref(readTip())

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
    <div v-if="tipVisible" class="tip" role="note">
      <p>收费前须有挂号/就诊 · 支付演示现金 · 已发药先退药再退费</p>
      <button class="tip-close" type="button" aria-label="关闭提示" @click="dismissTip">×</button>
    </div>

    <header class="work-head">
      <h1 id="outpatient-title">门诊业务</h1>
      <p class="flow">建档 → 挂号 → 开单 → 收费 → 发药 → 退药/退费</p>
    </header>

    <div class="module-grid">
      <article
        v-for="mod in cards"
        :key="mod.key"
        class="module-card"
        :class="{ 'is-locked': !mod.allowed, 'is-recommended': mod.recommended }"
      >
        <div class="card-kicker">
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
        <div class="card-foot">
          <p v-if="mod.reason" class="reason">{{ mod.reason }}</p>
          <RouterLink v-if="mod.allowed" class="button button-enter" :to="mod.path">进入</RouterLink>
          <button v-else class="button button-enter" type="button" disabled :title="mod.reason">进入</button>
        </div>
      </article>
    </div>
  </section>
</template>
