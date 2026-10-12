<script setup>
// 2026-10-12 P1-H05 登录界面
// 2026-10-12 P1-H05-LOGIN-POLISH 视觉打磨，不改 Auth 字段
// 2026-10-12 P1-H05-LOGIN-BG 登录页全屏背景
// 2026-10-12 真实 API 页显示本机演示账号
// 2026-10-12 演示账号 system / 1；员工默认密码为 1
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
  <main class="page login-page">
    <section class="card login-card">
      <aside class="brand-pane">
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
        <p class="lede">工作人员登录入口</p>
      </aside>

      <div class="form-pane">
        <header class="form-head">
          <h2>登录</h2>
          <p>请输入用户名和密码</p>
        </header>

        <form class="form" :aria-busy="loading" @submit.prevent="onSubmit">
          <label class="field">
            <span>用户名</span>
            <input
              v-model="username"
              name="username"
              autocomplete="username"
              placeholder="请输入用户名"
              required
              :disabled="loading"
            />
          </label>
          <label class="field">
            <span>密码</span>
            <input
              v-model="password"
              name="password"
              type="password"
              autocomplete="current-password"
              placeholder="请输入密码"
              required
              :disabled="loading"
            />
          </label>

          <p v-if="errorMessage" class="banner" role="alert">
            <svg class="banner-icon" viewBox="0 0 20 20" aria-hidden="true">
              <circle cx="10" cy="10" r="8.25" fill="none" stroke="currentColor" stroke-width="1.6" />
              <path d="M10 5.6v5.2" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" />
              <circle cx="10" cy="14.1" r="0.9" fill="currentColor" />
            </svg>
            <span class="banner-body">
              <span>{{ errorMessage }}</span>
              <span v-if="errorCode || requestId" class="banner-meta">
                <span v-if="errorCode" class="meta-chip">{{ errorCode }}</span>
                <span v-if="requestId" class="meta-id">{{ requestId }}</span>
              </span>
            </span>
          </p>

          <button class="button" type="submit" :disabled="loading" :aria-busy="loading">
            <span v-if="loading" class="spinner" aria-hidden="true"></span>
            {{ loading ? '登录中…' : '登录' }}
          </button>
        </form>

        <p v-if="mockMode" class="hint">
          <span>本地演示模式</span>
          <code>system</code>
          <span>/</span>
          <code>1</code>
        </p>
        <p v-else class="hint">
          <span>本机 API 账号</span>
          <code>system</code>
          <span>/</span>
          <code>1</code>
        </p>
      </div>
    </section>
  </main>
</template>
