<script setup>
// 2026-10-12 P1-H05 登录界面
// 2026-10-12 P1-H05-LOGIN-POLISH 登录成功页视觉
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { fetchMe, logout } from '../api/authApi.js'
import { ApiError } from '../api/errors.js'
import { applyUser, clearSession, session } from '../auth/session.js'

const router = useRouter()
const checking = ref(true)
const loggingOut = ref(false)
const errorMessage = ref('')

onMounted(async () => {
  try {
    const user = await fetchMe(session.accessToken)
    applyUser(user)
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

async function onLogout() {
  if (loggingOut.value) return
  errorMessage.value = ''
  loggingOut.value = true
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
  <main class="page">
    <section class="card home-card">
      <header class="home-top">
        <div class="brand-row">
          <span class="mark" aria-hidden="true">
            <svg viewBox="0 0 24 24" focusable="false">
              <path
                fill="currentColor"
                d="M10.2 2.8h3.6v7.4h7.4v3.6h-7.4v7.4h-3.6v-7.4H2.8v-3.6h7.4V2.8z"
              />
            </svg>
          </span>
          <div>
            <p class="eyebrow">B-HIS</p>
            <h1>医院信息系统</h1>
          </div>
        </div>
        <span class="pill">已登录</span>
      </header>

      <p v-if="checking && !session.user" class="muted">正在确认登录状态…</p>
      <template v-else-if="session.user">
        <p class="hello">{{ session.user.displayName }}</p>
        <p class="muted">当前会话有效</p>
        <dl class="profile">
          <div>
            <dt>姓名</dt>
            <dd>{{ session.user.displayName }}</dd>
          </div>
          <div>
            <dt>用户名</dt>
            <dd>{{ session.user.username }}</dd>
          </div>
          <div>
            <dt>角色</dt>
            <dd class="roles">
              <span v-for="role in session.user.roles" :key="role" class="role">{{ role }}</span>
              <span v-if="!session.user.roles.length">—</span>
            </dd>
          </div>
        </dl>
      </template>

      <p v-if="errorMessage" class="banner" role="alert">{{ errorMessage }}</p>

      <button
        class="button button-ghost"
        type="button"
        :disabled="loggingOut || checking"
        @click="onLogout"
      >
        <span v-if="loggingOut" class="spinner" aria-hidden="true"></span>
        {{ loggingOut ? '正在退出…' : '退出登录' }}
      </button>
    </section>
  </main>
</template>
