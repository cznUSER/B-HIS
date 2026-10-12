<script setup>
// 2026-10-12 P1-H05-HOME 模块空态：接口未接时不展示假数据。
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { session } from '../auth/session.js'
import ModuleIcon from '../components/ModuleIcon.vue'
import { decorateModules, findModule } from '../workbench/catalog.js'

const route = useRoute()

const mod = computed(() => {
  const found = findModule(String(route.params.module || ''))
  if (!found) return null
  return decorateModules(session.user && session.user.roles).find((item) => item.key === found.key) || null
})
</script>

<template>
  <section v-if="mod" class="module-page" :aria-labelledby="`module-${mod.key}`">
    <p class="crumb">
      <RouterLink :to="{ name: 'workbench' }">门诊业务</RouterLink>
      <span aria-hidden="true"> / </span>
      <span>{{ mod.name }}</span>
    </p>

    <div class="empty-panel">
      <span class="empty-mark" aria-hidden="true">
        <ModuleIcon :name="mod.key" />
      </span>
      <h1 :id="`module-${mod.key}`">{{ mod.name }}</h1>
      <template v-if="mod.allowed">
        <p class="empty-lead">接口就绪后开放</p>
        <p class="muted">{{ mod.blurb }}。门诊接口尚未接入，此页不展示业务数据。</p>
      </template>
      <template v-else>
        <p class="empty-lead">无权限</p>
        <p class="muted">当前账号不能进入此模块。</p>
      </template>
      <RouterLink class="button button-ghost empty-back" :to="{ name: 'workbench' }">返回工作台</RouterLink>
    </div>
  </section>
</template>
