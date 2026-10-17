<script setup>
// 2026-10-12 P1-H05-HOME 工作台顶栏：身份来自登录会话，不写死账号。
// 2026-10-12 P1-H05-HOME-DASH 顶栏加机构、人员、通知占位、退出和本地时钟。
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { fetchMe, logout } from '../api/authApi.js'
import { useMock } from '../api/config.js'
import { ApiError } from '../api/errors.js'
import { applyUser, clearSession, session } from '../auth/session.js'
import { rememberReturnTo } from '../router/returnTo.js'
import { findModule, roleText } from '../workbench/catalog.js'
import SideNav from './SideNav.vue'
import WorkflowSteps from './WorkflowSteps.vue'
import './shell-design.css'

const route = useRoute()
const router = useRouter()
const mockMode = useMock()

const checking = ref(true)
const loggingOut = ref(false)
const errorMessage = ref('')
const menuOpen = ref(false)
const navOpen = ref(false)
const mobileNav = ref(false)
const userbox = ref(null)
const now = ref(new Date())
let clockTimer = 0
let mobileQuery

const displayName = computed(() => {
  const name = session.user && session.user.displayName
  return typeof name === 'string' && name.trim() ? name.trim() : '当前用户'
})

const rolesLabel = computed(() => roleText(session.user && session.user.roles))
const envLabel = mockMode ? '演示' : '开发'
const avatarChar = computed(() => displayName.value.slice(0, 1) || '用')
const pageTitle = computed(() => {
  if (route.name === 'outpatient-module') return findModule(String(route.params.module || ''))?.name || '门诊业务'
  return route.meta.title || '工作台'
})
const clockDate = computed(() => now.value.toLocaleDateString('zh-CN', { month: 'long', day: 'numeric', weekday: 'long' }))
const clockTime = computed(() => now.value.toLocaleTimeString('zh-CN', { hour12: false }))

const clockText = computed(() => {
  const date = now.value
  const pad = (value) => String(value).padStart(2, '0')
  const week = '日一二三四五六'[date.getDay()]
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())} 星期${week}`
})

const clockIso = computed(() => now.value.toISOString())

onMounted(async () => {
  mobileQuery = window.matchMedia('(max-width: 720px)')
  mobileNav.value = mobileQuery.matches
  mobileQuery.addEventListener('change', onViewportChange)
  clockTimer = window.setInterval(() => {
    now.value = new Date()
  }, 1000)
  document.addEventListener('pointerdown', onPointerDown)
  document.addEventListener('keydown', onKeydown)
  if (!session.accessToken) {
    clearSession()
    await router.replace({ name: 'login', query: { redirect: rememberReturnTo(route.fullPath) } })
    return
  }
  const checkingToken = session.accessToken
  try {
    const user = await fetchMe(checkingToken)
    if (session.accessToken === checkingToken) applyUser(user)
  } catch (error) {
    if (session.accessToken !== checkingToken) return
    if (error instanceof ApiError && error.status === 401) {
      clearSession()
      if (route.name !== 'login') {
        await router.replace({ name: 'login', query: { redirect: rememberReturnTo(route.fullPath), expired: '1' } })
      }
      return
    }
    if (error instanceof ApiError && error.code === 'PASSWORD_CHANGE_REQUIRED' && session.user?.mustChangePassword) {
      errorMessage.value = ''
    } else {
      errorMessage.value = error instanceof ApiError ? error.message : '无法确认登录状态'
    }
  } finally {
    checking.value = false
  }
})

onBeforeUnmount(() => {
  window.clearInterval(clockTimer)
  mobileQuery?.removeEventListener('change', onViewportChange)
  document.removeEventListener('pointerdown', onPointerDown)
  document.removeEventListener('keydown', onKeydown)
})

watch(
  () => route.fullPath,
  () => {
    navOpen.value = false
    menuOpen.value = false
  },
)

function onPointerDown(event) {
  if (!menuOpen.value) return
  const root = userbox.value
  if (root && event.target instanceof Node && root.contains(event.target)) return
  menuOpen.value = false
}

function onViewportChange(event) {
  mobileNav.value = event.matches
  navOpen.value = false
}

function onKeydown(event) {
  if (event.key === 'Escape') {
    menuOpen.value = false
    navOpen.value = false
  }
}

function toggleMenu() {
  menuOpen.value = !menuOpen.value
}

async function onLogout() {
  if (loggingOut.value) return
  errorMessage.value = ''
  loggingOut.value = true
  menuOpen.value = false
  const logoutToken = session.accessToken
  try {
    await logout(logoutToken)
    if (session.accessToken !== logoutToken) return
    clearSession()
    await router.replace({ name: 'login' })
  } catch (error) {
    if (session.accessToken && session.accessToken !== logoutToken) return
    if (error instanceof ApiError && error.status === 401) {
      clearSession()
      await router.replace({ name: 'login' })
      return
    }
    errorMessage.value = error instanceof ApiError ? error.message : '退出登录失败'
  } finally {
    loggingOut.value = false
  }
}
</script>

<template>
  <div class="shell">
    <header class="topbar">
      <button
        class="nav-toggle"
        type="button"
        :aria-expanded="navOpen"
        aria-controls="app-sidebar"
        @click="navOpen = !navOpen"
      >
        <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M4 6h16M4 12h16M4 18h16" /></svg>
        <span class="sr-only">菜单</span>
      </button>

      <div class="shell-context">
        <span class="shell-context-label">医疗工作空间</span>
        <strong>{{ pageTitle }}</strong>
      </div>

      <div class="identity">
        <p>
          <svg class="institution-icon" viewBox="0 0 24 24" aria-hidden="true"><path d="M4 20h16M6 20V7l6-3 6 3v13M10 20v-5h4v5M9 9h.01M15 9h.01M9 12h.01M15 12h.01" /></svg>
          <span class="id-k">机构</span>
          <strong>演示机构</strong>
          <span class="phase-tag">占位</span>
        </p>
      </div>

      <div class="top-actions">
        <time class="clock" :datetime="clockIso" :aria-label="clockText">
          <strong>{{ clockTime }}</strong>
          <span>{{ clockDate }}</span>
        </time>

        <div ref="userbox" class="userbox">
          <button
            class="user-trigger"
            type="button"
            :aria-expanded="menuOpen"
            aria-haspopup="menu"
            :aria-label="displayName"
            @click="toggleMenu"
          >
            <span class="avatar" aria-hidden="true">{{ avatarChar }}</span>
            <span class="user-copy">
              <strong class="id-name"><span class="sr-only">人员 </span>{{ displayName }}</strong>
              <span class="id-role">{{ rolesLabel }}</span>
            </span>
            <svg class="user-chevron" viewBox="0 0 16 16" aria-hidden="true"><path d="m4 6 4 4 4-4" /></svg>
          </button>
          <Transition name="account-pop">
            <div v-if="menuOpen" class="user-menu" role="menu">
              <p class="account-menu-label">当前账号 · {{ rolesLabel }}</p>
              <button class="menu-item" type="button" role="menuitem" disabled title="二期">
                <span>个人设置</span>
                <span class="phase-tag">二期</span>
              </button>
              <button class="menu-item" type="button" role="menuitem" :disabled="loggingOut" @click="onLogout">
                {{ loggingOut ? '正在退出…' : '退出登录' }}
              </button>
            </div>
          </Transition>
        </div>

        <button class="top-logout" type="button" :disabled="loggingOut" @click="onLogout">
          <svg viewBox="0 0 24 24" aria-hidden="true" focusable="false">
            <path d="M12 3.2v6.2" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" />
            <path
              d="M7.2 6.4a7.2 7.2 0 1 0 9.6 0"
              fill="none"
              stroke="currentColor"
              stroke-width="1.8"
              stroke-linecap="round"
            />
          </svg>
          <span>{{ loggingOut ? '正在退出…' : '退出登录' }}</span>
        </button>
      </div>
    </header>

    <div class="workspace">
      <Transition name="drawer-shade">
        <button v-if="navOpen" class="nav-backdrop" type="button" aria-label="关闭菜单" @click="navOpen = false"></button>
      </Transition>
      <aside id="app-sidebar" class="sidebar" :class="{ 'is-open': navOpen }" :inert="mobileNav && !navOpen">
        <div class="brand-cluster">
          <RouterLink class="brand-link" :to="{ name: 'home' }" :aria-current="route.name === 'home' ? 'page' : undefined">
            <span class="logo-mark" aria-hidden="true">
              <svg viewBox="0 0 24 24" focusable="false"><path fill="currentColor" d="M10.2 2.8h3.6v7.4h7.4v3.6h-7.4v7.4h-3.6v-7.4H2.8v-3.6h7.4V2.8z" /></svg>
            </span>
            <span class="brand-wordmark"><strong>B-HIS</strong><small>医疗协同工作台</small></span>
          </RouterLink>
        </div>
        <SideNav v-if="!session.user?.mustChangePassword" />
        <div class="sidebar-foot">
          <span class="sidebar-foot-mark" aria-hidden="true">+</span>
          <div><strong>专注每一程诊疗</strong><span>清晰流程 · 高效协同</span></div>
          <span class="env-badge" :class="mockMode ? 'env-demo' : 'env-dev'">{{ envLabel }}</span>
        </div>
      </aside>
      <main class="shell-main">
        <p v-if="errorMessage" class="banner shell-banner" role="alert">{{ errorMessage }}</p>
        <WorkflowSteps v-if="!checking && route.name === 'outpatient-module'" />
        <p v-if="checking && !session.user" class="shell-status">正在确认登录状态…</p>
        <RouterView v-else v-slot="{ Component, route: viewRoute }">
          <Transition name="workspace-page" mode="out-in">
            <component :is="Component" :key="viewRoute.path" />
          </Transition>
        </RouterView>
      </main>
    </div>
  </div>
</template>
