<script setup>
// 2026-10-12 P1-H05-HOME 工作台顶栏：身份来自登录会话，不写死账号。
// 2026-10-12 P1-H05-HOME-DASH 顶栏加机构、人员、通知占位、退出和本地时钟。
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { fetchMe, logout } from '../api/authApi.js'
import { useMock } from '../api/config.js'
import { ApiError } from '../api/errors.js'
import { applyUser, clearSession, session } from '../auth/session.js'
import { roleText } from '../workbench/catalog.js'
import SideNav from './SideNav.vue'

const route = useRoute()
const router = useRouter()
const mockMode = useMock()

const checking = ref(true)
const loggingOut = ref(false)
const errorMessage = ref('')
const menuOpen = ref(false)
const navOpen = ref(false)
const userbox = ref(null)
const now = ref(new Date())
let clockTimer = 0

const displayName = computed(() => {
  const name = session.user && session.user.displayName
  return typeof name === 'string' && name.trim() ? name.trim() : '当前用户'
})

const rolesLabel = computed(() => roleText(session.user && session.user.roles))
const envLabel = mockMode ? '演示' : '开发'
const avatarChar = computed(() => displayName.value.slice(0, 1) || '用')

const clockText = computed(() => {
  const date = now.value
  const pad = (value) => String(value).padStart(2, '0')
  const week = '日一二三四五六'[date.getDay()]
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())} 星期${week}`
})

const clockIso = computed(() => now.value.toISOString())

onMounted(async () => {
  clockTimer = window.setInterval(() => {
    now.value = new Date()
  }, 1000)
  document.addEventListener('pointerdown', onPointerDown)
  document.addEventListener('keydown', onKeydown)
  if (!session.accessToken) {
    clearSession()
    await router.replace({ name: 'login' })
    return
  }
  try {
    applyUser(await fetchMe(session.accessToken))
  } catch (error) {
    if (error instanceof ApiError && (error.status === 401 || error.status === 403)) {
      clearSession()
      await router.replace({ name: 'login' })
      return
    }
    errorMessage.value = error instanceof ApiError ? error.message : '无法确认登录状态'
  } finally {
    checking.value = false
  }
})

onBeforeUnmount(() => {
  window.clearInterval(clockTimer)
  document.removeEventListener('pointerdown', onPointerDown)
  document.removeEventListener('keydown', onKeydown)
})

watch(
  () => route.fullPath,
  () => {
    navOpen.value = false
  },
)

function onPointerDown(event) {
  if (!menuOpen.value) return
  const root = userbox.value
  if (root && event.target instanceof Node && root.contains(event.target)) return
  menuOpen.value = false
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
  try {
    await logout(session.accessToken)
    clearSession()
    await router.replace({ name: 'login' })
  } catch (error) {
    if (error instanceof ApiError && error.status === 401) {
      clearSession()
      await router.replace({ name: 'login' })
      return
    }
    errorMessage.value = error instanceof ApiError ? error.message : '退出登录失败'
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
        菜单
      </button>

      <div class="brand-cluster">
        <RouterLink class="brand-link" :to="{ name: 'home' }" :aria-current="route.name === 'home' ? 'page' : undefined">
          <span class="logo-mark" aria-hidden="true">
            <svg viewBox="0 0 24 24" focusable="false">
              <path
                fill="currentColor"
                d="M10.2 2.8h3.6v7.4h7.4v3.6h-7.4v7.4h-3.6v-7.4H2.8v-3.6h7.4V2.8z"
              />
            </svg>
          </span>
          <span>B-HIS</span>
        </RouterLink>
        <span class="env-badge" :class="mockMode ? 'env-demo' : 'env-dev'">{{ envLabel }}</span>
      </div>

      <div class="identity">
        <p>
          <span class="id-k">机构</span>
          <strong>演示机构</strong>
          <span class="id-code">—</span>
          <span class="phase-tag">占位</span>
        </p>
        <p>
          <span class="id-k">人员</span>
          <strong class="id-name">{{ displayName }}</strong>
          <span class="id-role">{{ rolesLabel }}</span>
        </p>
      </div>

      <div class="top-actions">
        <button class="icon-btn" type="button" disabled title="占位" aria-label="通知，占位，暂无">
          <svg viewBox="0 0 24 24" aria-hidden="true" focusable="false">
            <path
              fill="none"
              stroke="currentColor"
              stroke-width="1.7"
              stroke-linejoin="round"
              d="M6.2 16.2h11.6l-1.2-1.6V10a4.6 4.6 0 0 0-9.2 0v4.6l-1.2 1.6z"
            />
            <path
              fill="none"
              stroke="currentColor"
              stroke-width="1.7"
              stroke-linecap="round"
              d="M10 16.4a2 2 0 0 0 4 0"
            />
          </svg>
          <span class="count">0</span>
        </button>

        <div ref="userbox" class="userbox">
          <button
            class="user-trigger user-avatar-btn"
            type="button"
            :aria-expanded="menuOpen"
            aria-haspopup="menu"
            :aria-label="displayName"
            @click="toggleMenu"
          >
            <span class="avatar" aria-hidden="true">{{ avatarChar }}</span>
          </button>
          <div v-if="menuOpen" class="user-menu" role="menu">
            <button class="menu-item" type="button" role="menuitem" disabled title="二期">
              <span>个人设置</span>
              <span class="phase-tag">二期</span>
            </button>
            <button class="menu-item" type="button" role="menuitem" :disabled="loggingOut" @click="onLogout">
              {{ loggingOut ? '正在退出…' : '退出登录' }}
            </button>
          </div>
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

        <time class="clock" :datetime="clockIso">{{ clockText }}</time>
      </div>
    </header>

    <div class="workspace">
      <button v-if="navOpen" class="nav-backdrop" type="button" aria-label="关闭菜单" @click="navOpen = false"></button>
      <aside id="app-sidebar" class="sidebar" :class="{ 'is-open': navOpen }">
        <SideNav />
      </aside>
      <main class="shell-main">
        <p v-if="errorMessage" class="banner shell-banner" role="alert">{{ errorMessage }}</p>
        <p v-if="checking && !session.user" class="shell-status">正在确认登录状态…</p>
        <router-view v-else />
      </main>
    </div>
  </div>
</template>
