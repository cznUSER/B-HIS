<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { changePassword, logout } from '../api/authApi.js'
import { ApiError } from '../api/errors.js'
import { clearSession, session } from '../auth/session.js'

const router = useRouter()
const currentPassword = ref('')
const newPassword = ref('')
const confirmPassword = ref('')
const busy = ref(false)
const error = ref('')
const canChange = computed(() => !session.user?.cannotModifyPassword)
const passwordDialog = ref(null)
onMounted(() => passwordDialog.value?.showModal())

async function submit() {
  if (busy.value || !canChange.value) return
  error.value = ''
  if (newPassword.value.length < 8 || newPassword.value.length > 72) {
    error.value = '新密码须为 8 至 72 个字符'
    return
  }
  if (newPassword.value !== confirmPassword.value) {
    error.value = '两次输入的新密码不一致'
    return
  }
  if (newPassword.value === currentPassword.value) {
    error.value = '新密码不能与当前密码相同'
    return
  }
  busy.value = true
  try {
    await changePassword(session.accessToken, currentPassword.value, newPassword.value)
    clearSession()
    await router.replace({ name: 'login', query: { passwordChanged: '1' } })
  } catch (e) {
    error.value = e instanceof ApiError ? e.message : '修改密码失败'
  } finally {
    busy.value = false
  }
}

async function leave() {
  if (busy.value) return
  busy.value = true
  try { await logout(session.accessToken) } catch { /* The local session still needs to be cleared. */ }
  clearSession()
  await router.replace({ name: 'login' })
}
</script>

<template>
  <dialog ref="passwordDialog" class="ui2-password-dialog" aria-labelledby="password-change-title" aria-describedby="password-change-guidance" @cancel.prevent="leave">
  <section class="password-change-page" aria-labelledby="password-change-title">
    <p class="crumb"><span>账号安全</span><span aria-hidden="true"> / </span><span>修改密码</span></p>
    <header class="master-page-header">
      <div><p class="master-eyebrow">ACCOUNT SECURITY</p><h1 id="password-change-title">修改密码</h1><p id="password-change-guidance" class="master-page-description sr-only">管理员已重置此账号密码。新密码须为 8 至 72 个字符，修改后请重新登录。</p></div>
      <div class="master-access-tag"><span></span>需要完成</div>
    </header>
    <form v-if="canChange" class="master-form password-change-form" data-testid="password-change-form" @submit.prevent="submit">
      <label class="field"><span>请输入原密码</span><input v-model="currentPassword" type="password" autocomplete="current-password" required :disabled="busy" /></label>
      <label class="field"><span>请输入新密码</span><input v-model="newPassword" type="password" autocomplete="new-password" minlength="8" maxlength="72" required :disabled="busy" /></label>
      <label class="field"><span>请在输入一次新密码</span><input v-model="confirmPassword" type="password" autocomplete="new-password" minlength="8" maxlength="72" required :disabled="busy" /></label>
      <p class="access-inline-note">密码长度为 8 至 72 个字符，不能与当前密码相同。</p>
      <p v-if="error" class="banner" role="alert">{{ error }}</p>
      <div class="panel-actions"><button class="button button-compact" type="submit" autofocus :disabled="busy">{{ busy ? '正在提交…' : '确定(O)' }}</button><button class="button button-ghost button-compact" type="button" :disabled="busy" title="取消修改并退出登录" @click="leave">取消(C)</button></div>
    </form>
    <div v-else class="master-data-panel password-change-blocked" data-testid="password-change-blocked">
      <h2>此账号不允许自行修改密码</h2><p>请联系系统管理员处理后再登录。</p>
      <button class="button button-ghost button-compact" type="button" :disabled="busy" @click="leave">{{ busy ? '正在退出…' : '退出登录' }}</button>
    </div>
  </section>
  </dialog>
</template>

<style scoped>
.ui2-password-dialog { width: 326px; max-width: calc(100vw - 24px); padding: 0; border: 1px solid ButtonShadow; border-radius: 0; background: ButtonFace; color: ButtonText; box-shadow: 0 8px 24px #0003; }
.ui2-password-dialog::backdrop { background: #0002; }
.password-change-page { display: flex; flex-direction: column; gap: 0; align-items: center; padding: 0; background: ButtonFace; color: ButtonText; font-family: SimSun, '宋体', serif; font-size: 16px; }
.password-change-page .crumb, .master-eyebrow, .master-access-tag { display: none; }
.password-change-page .master-page-header { display: block; width: 326px; max-width: 100%; margin: 0; padding: 6px 10px; border: 0; border-bottom: 1px solid ButtonShadow; background: ButtonFace; box-sizing: border-box; }
.password-change-page.password-change-page h1 { font: 12px 'Microsoft YaHei UI', sans-serif; margin: 0; }
.password-change-page .master-page-description { font: 12px/1.5 'Segoe UI', sans-serif; margin: 7px 0 0; color: #515151; }
.password-change-page .password-change-form { display: block; width: 326px; max-width: 100%; min-height: 301px; padding: 27px 27px 30px 47px; box-sizing: border-box; border: 1px solid #a5a5a5; border-top: 0; border-radius: 0; background: #f0f0f0; box-shadow: 0 8px 20px #00000012; }
.password-change-page .password-change-form .field { display: block; margin: 0 0 7px; min-width: 0; }
.password-change-page .password-change-form .field > span { width: 88px; display: table; height: 25px; margin-bottom: 7px; padding: 3px 5px; border: 1px solid #cacaca; border-radius: 5px; background: #ddd; box-sizing: border-box; font: 12px/17px 'Segoe UI', sans-serif; color: #000; }
.password-change-page .password-change-form .field:nth-child(3) > span { width: 127px; }
.password-change-page.password-change-page .password-change-form input { width: 250px; max-width: 100%; height: 25px; min-height: 25px; padding: 2px 5px; border: 1px solid #999; border-radius: 0; background: white; color: #000; box-sizing: border-box; font: 16px SimSun, '宋体', serif; }
.password-change-page .password-change-form input:focus { outline: 2px solid #0078d4; outline-offset: 1px; }
.password-change-page .password-change-form .access-inline-note { display: none; font: 12px/1.5 'Segoe UI', sans-serif; margin: 0 0 10px; }
.password-change-page .password-change-form .panel-actions { display: flex; justify-content: flex-end; gap: 8px; margin-top: 29px; }
.password-change-page .password-change-form .button { flex: 0 0 90px; width: 90px; height: 30px; min-height: 30px; padding: 0 4px; border: 1px solid #aeaeae; border-radius: 3px; background: #e7e7e7; color: #000; font: 16px SimSun, '宋体', serif; box-shadow: none; }
.password-change-page .password-change-form .button:hover:not(:disabled) { border-color: #929292; background: #ddd; }
.password-change-page .password-change-form .button:focus-visible { outline: 1px dotted #000; outline-offset: -4px; border-color: #0078d7; background: #f0f0f0; }
.password-change-page .password-change-form .button:active:not(:disabled) { border-color: #636363; background: #d4d4d4; }
.password-change-page .password-change-form .button:disabled { background: #ebebeb; border-color: #d4d4d4; color: #a0a0a0; cursor: wait; }
.password-change-page .password-change-form .banner { font: 12px/1.5 'Segoe UI', sans-serif; margin: 8px 0; }
.password-change-blocked { width: 326px; max-width: 100%; box-sizing: border-box; }
@media (max-width: 350px) { .password-change-page .password-change-form { padding-left: 24px; padding-right: 24px; } }
</style>
