<script setup>
// 2026-10-12 P1-H05 登录界面
// 2026-10-12 P1-H05-LOGIN-POLISH 视觉打磨，不改 Auth 字段
// 2026-10-12 P1-H05-LOGIN-BG 登录页全屏背景
// 2026-10-12 真实 API 页显示本机演示账号
// 2026-10-12 演示账号 system / 1；员工默认密码为 1
// 2026-10-12 P1-H05-HOME 登录成功进入门诊工作台
// 2026-10-12 P1-H05-HOME-DASH 登录成功进入首页看板
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { login } from '../api/authApi.js'
import { useMock } from '../api/config.js'
import { ApiError } from '../api/errors.js'
import { applyLogin } from '../auth/session.js'
import { safeReturnTo, takeReturnTo } from '../router/returnTo.js'

const router = useRouter()
const route = useRoute()
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
    if (result.user.mustChangePassword) {
      await router.replace({ name: 'password-change' })
      return
    }
    const saved = takeReturnTo()
    // Validate the deep link against the server's effective permissions so a
    // role cannot bypass a user-level deny or department scope.
    const target = safeReturnTo(route.query.redirect || saved, result.user)
    await router.replace(target || { name: 'home' })
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
    <div class="login-ambient login-ambient-one" aria-hidden="true"></div>
    <div class="login-ambient login-ambient-two" aria-hidden="true"></div>
    <section class="card login-card">
      <aside class="brand-pane">
        <div class="login-brand">
          <span class="mark" aria-hidden="true">
            <svg viewBox="0 0 24 24" focusable="false">
              <path
                fill="currentColor"
                d="M10.2 2.8h3.6v7.4h7.4v3.6h-7.4v7.4h-3.6v-7.4H2.8v-3.6h7.4V2.8z"
              />
            </svg>
          </span>
          <p class="eyebrow">B-HIS <span>医院信息系统</span></p>
        </div>

        <div class="login-story">
          <p class="login-kicker">以专注，连接每一次诊疗</p>
          <h1>让日常诊疗<br />更加从容。</h1>
          <p class="lede">连接患者、诊疗与药房业务，<br />在清晰有序的工作台开启新一天。</p>
        </div>

        <div class="login-art" aria-hidden="true">
          <div class="login-orbit login-orbit-outer"></div>
          <div class="login-orbit login-orbit-inner"></div>
          <div class="login-art-grid"></div>
          <div class="login-cross">
            <svg viewBox="0 0 100 100" focusable="false">
              <path d="M39 12h22v27h27v22H61v27H39V61H12V39h27z" fill="currentColor" />
            </svg>
          </div>
          <span class="login-art-dot login-art-dot-one"></span>
          <span class="login-art-dot login-art-dot-two"></span>
          <span class="login-art-label login-art-label-one">
            <svg viewBox="0 0 20 20" focusable="false"><path d="M6 3.5h8a1.5 1.5 0 0 1 1.5 1.5v11a1 1 0 0 1-1 1h-9a1 1 0 0 1-1-1V5A1.5 1.5 0 0 1 6 3.5Z"/><path d="M7 7h6M7 10h6M7 13h3"/></svg>
            患者档案
          </span>
          <span class="login-art-label login-art-label-two">
            <svg viewBox="0 0 20 20" focusable="false"><path d="M7 3.5h6v4h4v6h-4v4H7v-4H3v-6h4z"/></svg>
            门诊诊疗
          </span>
        </div>

        <div class="login-brand-foot"><span class="login-foot-line"></span>专为日常医疗工作而设计</div>
      </aside>

      <div class="form-pane">
        <header class="form-head">
          <p class="login-form-kicker">工作人员登录入口</p>
          <h2>欢迎回来</h2>
          <p>登录账号，进入您的工作台</p>
        </header>

        <form class="form" :aria-busy="loading" @submit.prevent="onSubmit">
          <p v-if="route.query.expired === '1' && !errorMessage" class="hint" role="status">登录已失效，重新登录后返回原页面。</p>
          <p v-if="route.query.passwordChanged === '1' && !errorMessage" class="hint" role="status">密码已更新，请使用新密码登录。</p>
          <label class="field">
            <span>用户名</span>
            <div class="login-input">
              <svg viewBox="0 0 20 20" aria-hidden="true"><circle cx="10" cy="6.5" r="3"/><path d="M4.5 17v-1.5A4.5 4.5 0 0 1 9 11h2a4.5 4.5 0 0 1 4.5 4.5V17"/></svg>
              <input
                v-model="username"
                name="username"
                autocomplete="username"
                placeholder="system"
                required
                :disabled="loading"
              />
            </div>
          </label>
          <label class="field">
            <span>密码</span>
            <div class="login-input">
              <svg viewBox="0 0 20 20" aria-hidden="true"><rect x="4.5" y="8.5" width="11" height="8" rx="2"/><path d="M7 8.5V6a3 3 0 0 1 6 0v2.5M10 12v1.5"/></svg>
              <input
                v-model="password"
                name="password"
                type="password"
                autocomplete="current-password"
                placeholder="1"
                required
                :disabled="loading"
              />
            </div>
          </label>

          <Transition name="login-feedback">
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
          </Transition>

          <button class="button" type="submit" :disabled="loading" :aria-busy="loading">
            <span v-if="loading" class="spinner" aria-hidden="true"></span>
            {{ loading ? '登录中…' : '登录' }}
            <svg v-if="!loading" viewBox="0 0 20 20" aria-hidden="true"><path d="M4 10h12M11 5l5 5-5 5"/></svg>
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
        <p class="login-form-foot">B-HIS · 医院信息系统</p>
      </div>
    </section>
    <p class="login-page-foot">让信息有序流转，让医护专注诊疗</p>
  </main>
</template>

<style scoped src="./login-design.css"></style>
