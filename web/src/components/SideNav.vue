<script setup>
// 2026-10-12 P1-H05-HOME-DASH 左侧导航。延后项不进入页面。
// 2026-10-13 P1-H05-MASTER-UI 系统账号显示「基础数据」。
import { computed, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { session } from '../auth/session.js'
import { decorateModules } from '../workbench/catalog.js'
import { NAV, resolveNavChild } from '../workbench/nav.js'
import { outpatientTarget } from '../workbench/context.js'
import ModuleIcon from './ModuleIcon.vue'

const route = useRoute()
const open = ref({
  archive: true,
  outpatient: true,
  settings: false,
})

const modules = computed(() => decorateModules(session.user))

const groups = computed(() =>
  NAV.filter((item) => item.key !== 'home').map((group) => ({
    ...group,
    children: (group.children || []).map((child) => ({
      ...child,
      ...resolveNavChild(child, modules.value, session.user),
    })).filter((child) => !child.hidden),
  })),
)

const groupIcons = {
  archive: 'M4 7h6l2 2h8v10H4V7z M4 7V5h6l2 2',
  outpatient: 'M9 4h6v3h4v13H5V7h4V4z M9 13h6 M12 10v6',
  settings: 'M9 4h6l1 3 3 1v6l-3 1-1 3H9l-1-3-3-1V8l3-1 1-3z M12 9a2.5 2.5 0 1 0 0 5 2.5 2.5 0 0 0 0-5',
}

function toggle(key) {
  open.value = { ...open.value, [key]: !open.value[key] }
}

watch(
  () => route.path,
  (path) => {
    if (path.startsWith('/app/settings/master') || path.startsWith('/app/settings/users')) {
      open.value = { ...open.value, settings: true }
    }
  },
  { immediate: true },
)

function childActive(child) {
  if (child.module && child.to) return route.name === child.to.name
  if (child.module) {
    return route.name === 'outpatient-module' && String(route.params.module || '') === child.module
  }
  if (child.to && child.to.name === 'master') return route.path.startsWith('/app/settings/master')
  if (child.to && child.to.name === 'access-users') return route.path.startsWith('/app/settings/users')
  return Boolean(child.to && route.name === child.to.name)
}

function childTarget(child) {
  return child.module && !child.to ? outpatientTarget(child.module, route.query) : child.to
}

function overviewActive(group) {
  return Boolean(group.overview) && route.name === group.overview.to.name
}
</script>

<template>
  <nav class="sidenav" aria-label="主导航">
    <p class="nav-caption">工作空间 <span>WORKSPACE</span></p>
    <div v-for="group in groups" :key="group.key" class="nav-group">
      <button
        class="nav-group-btn"
        type="button"
        :aria-controls="`nav-children-${group.key}`"
        :aria-expanded="open[group.key] ? 'true' : 'false'"
        @click="toggle(group.key)"
      >
        <svg class="nav-icon" viewBox="0 0 24 24" aria-hidden="true"><path :d="groupIcons[group.key]" /></svg>
        <span>{{ group.label }}</span>
        <span class="chev" :class="{ 'is-open': open[group.key] }" aria-hidden="true"></span>
      </button>

      <div :id="`nav-children-${group.key}`" class="nav-expand" :class="{ 'is-open': open[group.key] }" :aria-hidden="!open[group.key]" :inert="!open[group.key]">
        <div class="nav-children">
          <div v-for="child in group.children" :key="child.key" class="nav-child">
            <RouterLink
              v-if="child.enabled"
              class="nav-sublink"
              :to="childTarget(child)"
              :class="{ 'is-active': childActive(child) }"
              :aria-current="childActive(child) ? 'page' : undefined"
            >
              <span class="nav-child-icon" aria-hidden="true">
                <ModuleIcon v-if="child.module" :name="child.module" />
                <svg v-else viewBox="0 0 24 24"><path d="M5 5h5v5H5z M14 5h5v5h-5z M5 14h5v5H5z M14 14h5v5h-5z" /></svg>
              </span>
              <span>{{ child.label }}</span>
              <span v-if="childActive(child)" class="nav-active-dot" aria-hidden="true"></span>
            </RouterLink>
            <span v-else class="nav-sublink is-disabled" :title="child.reason">
              <span class="nav-child-icon" aria-hidden="true">
                <ModuleIcon v-if="child.module" :name="child.module" />
                <svg v-else viewBox="0 0 24 24"><path d="M6 10h12v10H6z M9 10V7a3 3 0 0 1 6 0v3" /></svg>
              </span>
              <span>{{ child.label }}</span>
              <span class="nav-tag">{{ child.reason }}</span>
            </span>
          </div>

          <RouterLink
            v-if="group.overview"
            class="nav-sublink is-quiet"
            :to="group.overview.to"
            :class="{ 'is-active': overviewActive(group) }"
            :aria-current="overviewActive(group) ? 'page' : undefined"
          >
            <svg class="nav-overview-icon" viewBox="0 0 24 24" aria-hidden="true"><path d="M4 7h16M4 12h16M4 17h10 M18 15l3 3-3 3" /></svg>
            <span>{{ group.overview.label }}</span>
          </RouterLink>
          <p v-if="group.key === 'outpatient'" class="nav-tip">
            收费前须有挂号/就诊 · 支付演示现金 · 已发药先退药再退费
          </p>
        </div>
      </div>
    </div>
  </nav>
</template>
