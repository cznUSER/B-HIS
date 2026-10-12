<script setup>
// 2026-10-12 P1-H05 登录界面
// 2026-10-12 真实 API 页显示本机演示账号
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { login } from '../api/authApi.js'
import { useMock } from '../api/config.js'
import { ApiError } from '../api/errors.js'
import { applyLogin } from '../auth/session.js'

const router = useRouter()
const mockMode = useMock()

const username = ref('')
const password = ref('')
const loading = ref(false)
const errorMessage = ref('')
const errorCode = ref('')
const requestId = ref('')

async function onSubmit() {
  if (loading.value) return
  errorMessage.value = ''
  errorCode.value = ''
  requestId.value = ''
  loading.value = true
  try {
    const result = await login(username.value.trim(), password.value)
    applyLogin(result)
    password.value = ''
    await router.replace({ name: 'home' })
  } catch (error) {
    if (error instanceof ApiError) {
      errorMessage.value = error.message
      errorCode.value = error.code
      requestId.value = error.requestId
    } else {
      errorMessage.value = '登录失败，请稍后重试'
    }
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <main class="page">
    <section class="card">
      <header class="brand">
        <span class="mark" aria-hidden="true">H</span>
        <div>
          <p class="eyebrow">B-HIS</p>
          <h1>医院信息系统</h1>
        </div>
      </header>

      <form class="form" @submit.prevent="onSubmit">
        <label>
          <span>用户名</span>
          <input
            v-model="username"
            name="username"
            autocomplete="username"
            required
            :disabled="loading"
          />
        </label>
        <label>
          <span>密码</span>
          <input
            v-model="password"
            name="password"
            type="password"
            autocomplete="current-password"
            required
            :disabled="loading"
          />
        </label>

        <p v-if="errorMessage" class="error" role="alert">
          {{ errorMessage }}
          <span v-if="errorCode" class="error-meta">{{ errorCode }}</span>
          <span v-if="requestId" class="error-meta">{{ requestId }}</span>
        </p>

        <button type="submit" :disabled="loading">
          {{ loading ? '登录中…' : '登录' }}
        </button>
      </form>

      <p v-if="mockMode" class="hint">本地演示模式。账号 admin / admin123</p>
      <p v-else class="hint">本机 API 账号 admin / Admin@123</p>
    </section>
  </main>
</template>
