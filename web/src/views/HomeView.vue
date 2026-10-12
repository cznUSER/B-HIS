<script setup>
// 2026-10-12 P1-H05 登录界面
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
    <section class="card home">
      <header class="brand">
        <span class="mark" aria-hidden="true">H</span>
        <div>
          <p class="eyebrow">B-HIS</p>
          <h1>医院信息系统</h1>
        </div>
      </header>

      <p class="status">已登录</p>
      <p v-if="checking" class="muted">正在确认登录状态…</p>
      <dl v-else-if="session.user" class="profile">
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
          <dd>{{ session.user.roles.join('、') || '—' }}</dd>
        </div>
      </dl>

      <p v-if="errorMessage" class="error" role="alert">{{ errorMessage }}</p>

      <button type="button" :disabled="loggingOut || checking" @click="onLogout">
        {{ loggingOut ? '正在退出…' : '退出登录' }}
      </button>
    </section>
  </main>
</template>
