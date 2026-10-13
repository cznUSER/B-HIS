<script setup>
// 2026-10-13 P1-H05-MASTER-UI 轻量用户：新建初始密码为 1，可启停和重置。没有权限矩阵。
import { onMounted, reactive, ref } from 'vue'
import { ApiError } from '../../api/errors.js'
import { session } from '../../auth/session.js'
import { fetchUsers, resetUserPassword, saveUser } from '../../master/api.js'
import { statusLabel } from '../../master/labels.js'
import { MASTER_ROLES } from '../../master/store.js'

const rows = ref([])
const loading = ref(false)
const saving = ref(false)
const editing = ref(false)
const editingId = ref('')
const q = ref('')
const status = ref('')
const notice = ref('')
const noticeError = ref(false)
const errorCode = ref('')
const form = reactive({ username: '', displayName: '', status: 'ACTIVE', roles: [] })

function show(message, isError = false, code = '') {
  notice.value = message
  noticeError.value = isError
  errorCode.value = code
}

function roleText(roles) {
  const list = Array.isArray(roles) ? roles : []
  if (list.length === 0) return '—'
  return list
    .map((code) => {
      const found = MASTER_ROLES.find((item) => item.code.toLowerCase() === String(code).toLowerCase())
      return found ? found.label : code
    })
    .join('、')
}

async function load() {
  loading.value = true
  try {
    const listed = await fetchUsers(session.accessToken, { q: q.value, status: status.value })
    rows.value = listed.items
  } catch (error) {
    show(error instanceof ApiError ? error.message : '无法读取用户', true, error instanceof ApiError ? error.code : '')
  } finally {
    loading.value = false
  }
}

function startCreate() {
  editing.value = true
  editingId.value = ''
  form.username = ''
  form.displayName = ''
  form.status = 'ACTIVE'
  form.roles = ['REGISTRAR']
  show('', false)
}

function startEdit(row) {
  editing.value = true
  editingId.value = row.userId
  form.username = row.username
  form.displayName = row.displayName
  form.status = row.status === 'ACTIVE' ? 'ACTIVE' : 'DISABLED'
  form.roles = [...row.roles]
  show('', false)
}

async function onSave() {
  if (saving.value) return
  saving.value = true
  const id = editingId.value
  try {
    if (id) {
      await saveUser(session.accessToken, id, {
        displayName: form.displayName,
        status: form.status,
        roles: form.roles,
      })
      show('已保存用户', false)
    } else {
      await saveUser(session.accessToken, '', {
        username: form.username,
        displayName: form.displayName,
        status: form.status,
        roles: form.roles,
      })
      show('已新建用户。初始密码为 1，页面不再回显。', false)
    }
    editing.value = false
    await load()
  } catch (error) {
    show(error instanceof ApiError ? error.message : '保存失败', true, error instanceof ApiError ? error.code : '')
  } finally {
    saving.value = false
  }
}

async function toggle(row) {
  const next = row.status === 'ACTIVE' ? 'DISABLED' : 'ACTIVE'
  try {
    await saveUser(session.accessToken, row.userId, { status: next })
    show(next === 'ACTIVE' ? '已启用账号' : '已停用。该账号再次登录会被拒绝。', false)
    await load()
  } catch (error) {
    show(error instanceof ApiError ? error.message : '操作失败', true, error instanceof ApiError ? error.code : '')
  }
}

async function onReset(row) {
  try {
    await resetUserPassword(session.accessToken, row.userId)
    show(`已将 ${row.username} 重置为演示初始密码，页面不回显。`, false)
  } catch (error) {
    show(error instanceof ApiError ? error.message : '重置失败', true, error instanceof ApiError ? error.code : '')
  }
}

onMounted(load)
</script>

<template>
  <section class="patient-panel" aria-labelledby="user-heading">
    <div class="panel-head">
      <h2 id="user-heading">用户账号</h2>
      <button class="button button-compact" type="button" data-testid="user-create" @click="startCreate">新建</button>
    </div>
    <p class="field-hint">只勾选预置角色。没有权限矩阵，也没有多院区。新建与重置都使用演示初始密码 1。</p>

    <form class="master-filters" @submit.prevent="load">
      <label class="field">
        <span>登录名 / 显示名</span>
        <input v-model="q" type="search" />
      </label>
      <label class="field">
        <span>状态</span>
        <select v-model="status">
          <option value="">全部</option>
          <option value="ACTIVE">启用</option>
          <option value="DISABLED">停用</option>
        </select>
      </label>
      <button class="button button-ghost button-compact" type="submit">查询</button>
    </form>

    <form v-if="editing" class="master-form" data-testid="user-form" @submit.prevent="onSave">
      <div class="field-grid">
        <label class="field">
          <span>登录名 <abbr class="req" title="必填">*</abbr></span>
          <input v-model="form.username" :disabled="Boolean(editingId)" required data-testid="user-name" autocomplete="off" />
        </label>
        <label class="field">
          <span>显示名 <abbr class="req" title="必填">*</abbr></span>
          <input v-model="form.displayName" required data-testid="user-display" autocomplete="off" />
        </label>
        <label class="field">
          <span>状态</span>
          <select v-model="form.status">
            <option value="ACTIVE">启用</option>
            <option value="DISABLED">停用</option>
          </select>
        </label>
      </div>
      <fieldset class="role-picks">
        <legend>角色</legend>
        <label v-for="role in MASTER_ROLES" :key="role.code" class="role-pick">
          <input v-model="form.roles" type="checkbox" :value="role.code" />
          <span>{{ role.label }}</span>
        </label>
      </fieldset>
      <p v-if="!editingId" class="field-hint">初始密码为 1。保存后不在列表或接口结果里回显。</p>
      <div class="panel-actions">
        <button class="button button-compact" type="submit" data-testid="user-save" :disabled="saving">
          {{ saving ? '正在保存…' : '保存' }}
        </button>
        <button class="button button-ghost button-compact" type="button" @click="editing = false">取消</button>
      </div>
    </form>

    <p v-if="notice" class="banner" :class="{ 'banner-ok': !noticeError }" :role="noticeError ? 'alert' : 'status'" data-testid="user-notice">
      {{ notice }}<span v-if="errorCode">（{{ errorCode }}）</span>
    </p>
    <p v-if="loading" class="shell-status">正在读取…</p>
    <p v-else-if="rows.length === 0" class="result-empty">{{ q || status ? '没有符合条件的用户' : '暂无用户' }}</p>
    <div v-else class="result-wrap">
      <table class="register-table" data-testid="user-list">
        <caption class="sr-only">用户账号</caption>
        <thead>
          <tr>
            <th>登录名</th>
            <th>显示名</th>
            <th>角色</th>
            <th>状态</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="row in rows" :key="row.userId">
            <td class="mono">{{ row.username }}</td>
            <td>{{ row.displayName }}</td>
            <td>{{ roleText(row.roles) }}</td>
            <td><span class="status-pill" :data-status="row.status">{{ statusLabel(row.status) }}</span></td>
            <td class="ticket-actions">
              <button class="text-btn" type="button" @click="startEdit(row)">编辑</button>
              <button class="text-btn" type="button" :data-testid="`user-toggle-${row.username}`" @click="toggle(row)">
                {{ row.status === 'ACTIVE' ? '停用' : '启用' }}
              </button>
              <button class="text-btn" type="button" :data-testid="`user-reset-${row.username}`" @click="onReset(row)">重置密码</button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </section>
</template>
