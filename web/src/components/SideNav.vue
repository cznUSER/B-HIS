<script setup>
// 2026-10-12 P1-H05-HOME-DASH 左侧导航。延后项不进入页面。
import { computed, ref } from 'vue'
import { useRoute } from 'vue-router'
import { session } from '../auth/session.js'
import { decorateModules } from '../workbench/catalog.js'
import { NAV, resolveNavChild } from '../workbench/nav.js'

const route = useRoute()
const open = ref({
  archive: true,
  outpatient: true,
  settings: false,
})

const modules = computed(() => decorateModules(session.user && session.user.roles))

const groups = computed(() =>
  NAV.filter((item) => item.key !== 'home').map((group) => ({
    ...group,
    children: (group.children || []).map((child) => ({
      ...child,
      ...resolveNavChild(child, modules.value),
    })),
  })),
)

const home = NAV.find((item) => item.key === 'home')

function toggle(key) {
  open.value = { ...open.value, [key]: !open.value[key] }
}

function childActive(child) {
  return route.name === 'outpatient-module' && String(route.params.module || '') === child.module
}

function overviewActive(group) {
  return Boolean(group.overview) && route.name === group.overview.to.name
}
</script>

<template>
  <nav class="sidenav" aria-label="主导航">
    <RouterLink
      class="nav-toplink"
      :to="home.to"
      :class="{ 'is-active': route.name === 'home' }"
      :aria-current="route.name === 'home' ? 'page' : undefined"
    >
      {{ home.label }}
    </RouterLink>

    <div v-for="group in groups" :key="group.key" class="nav-group">
      <button
        class="nav-group-btn"
        type="button"
        :aria-expanded="open[group.key] ? 'true' : 'false'"
        @click="toggle(group.key)"
      >
        <span>{{ group.label }}</span>
        <span class="chev" :class="{ 'is-open': open[group.key] }" aria-hidden="true"></span>
      </button>

      <div v-show="open[group.key]" class="nav-children">
        <template v-for="child in group.children" :key="child.key">
          <RouterLink
            v-if="child.enabled"
            class="nav-sublink"
            :to="child.to"
            :class="{ 'is-active': childActive(child) }"
            :aria-current="childActive(child) ? 'page' : undefined"
          >
            {{ child.label }}
          </RouterLink>
          <span v-else class="nav-sublink is-disabled" :title="child.reason">
            <span>{{ child.label }}</span>
            <span class="nav-tag">{{ child.reason }}</span>
          </span>
        </template>

        <p v-if="group.key === 'outpatient'" class="nav-tip">
          收费前须有挂号/就诊 · 支付演示现金 · 已发药先退药再退费
        </p>

        <RouterLink
          v-if="group.overview"
          class="nav-sublink is-quiet"
          :to="group.overview.to"
          :class="{ 'is-active': overviewActive(group) }"
          :aria-current="overviewActive(group) ? 'page' : undefined"
        >
          {{ group.overview.label }}
        </RouterLink>
      </div>
    </div>
  </nav>
</template>
