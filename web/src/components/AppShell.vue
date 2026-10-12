<script setup>
// 2026-10-12 P1-H05-HOME 工作台顶栏：身份来自登录会话，不写死账号。
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { fetchMe, logout } from '../api/authApi.js'
import { useMock } from '../api/config.js'
import { ApiError } from '../api/errors.js'
import { applyUser, clearSession, session } from '../auth/session.js'
import { roleText } from '../workbench/catalog.js'

const route = useRoute()
const router = useRouter()
const mockMode = useMock()

const checking = ref(true)
const loggingOut = ref(false)
const errorMessage = ref('')
const menuOpen = ref(false)
const userbox = ref(null)

const displayName = computed(() => {
  const name = session.user && session.user.displayName
  return typeof name === 'string' && name.trim() ? name.trim() : '当前用户'
})

const rolesLabel = computed(() => roleText(session.user && session.user.roles))
const envLabel = mockMode ? '演示' : '开发'

onMounted(async () => {
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
  document.removeEventListener('pointerdown', onPointerDown)
  document.removeEventListener('keydown', onKeydown)
})

function onPointerDown(event) {
  if (!menuOpen.value) return
  const root = userbox.value
  if (root && event.target instanceof Node && root.contains(event.target)) return
  menuOpen.value = false
}

function onKeydown(event) {
  if (event.key === 'Escape') menuOpen.value = false
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
      <div class="brand-cluster">
        <RouterLink
          class="brand-link"
          :to="{ name: 'workbench' }"
          :aria-current="route.name === 'workbench' ? 'page' : undefined"
        >
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

      <label class="search-slot">
        <span class="sr-only">患者搜索，二期</span>
        <input
          type="search"
          placeholder="搜索患者"
          title="二期"
          disabled
          autocomplete="off"
          aria-disabled="true"
        />
        <span class="phase-tag">二期</span>
      </label>

      <div class="top-actions">
        <button class="icon-btn" type="button" disabled title="二期" aria-label="通知（二期），暂无">
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
            class="user-trigger"
            type="button"
            :aria-expanded="menuOpen"
            aria-haspopup="menu"
            @click="toggleMenu"
          >
            <span class="user-name">{{ displayName }}</span>
            <span class="user-role">{{ rolesLabel }}</span>
            <span class="user-caret" aria-hidden="true"></span>
          </button>
          <div v-if="menuOpen" class="user-menu" role="menu">
            <button class="menu-item" type="button" role="menuitem" disabled title="二期">
              <span>个人设置</span>
              <span class="phase-tag">二期</span>
            </button>
            <button
              class="menu-item"
              type="button"
              role="menuitem"
              :disabled="loggingOut"
              @click="onLogout"
            >
              {{ loggingOut ? '正在退出…' : '退出登录' }}
            </button>
          </div>
        </div>
      </div>
    </header>

    <div class="shell-body">
      <p v-if="errorMessage" class="banner shell-banner" role="alert">{{ errorMessage }}</p>
      <p v-if="checking && !session.user" class="shell-status">正在确认登录状态…</p>
      <router-view v-else />
    </div>
  </div>
</template>
