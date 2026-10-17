<script setup>
// 2026-10-14 P1-H05-WEB-USABILITY 读取状态独立于写入提示；只有成功读取才可维护。
// 2026-10-13 P1-H05-MASTER-UI 轻量用户：新建初始密码为 1，可启停和重置。没有权限矩阵。
import { computed, onMounted, reactive, ref } from 'vue'
import { ApiError } from '../../api/errors.js'
import { session } from '../../auth/session.js'
import { fetchUsers, resetUserPassword, saveUser } from '../../master/api.js'
import { statusLabel } from '../../master/labels.js'
import { MASTER_ROLES } from '../../master/store.js'
import MasterPanelHeader from './MasterPanelHeader.vue'

const rows = ref([])
const loading = ref(false)
const listReady = ref(false)
const loadError = ref('')
const loadErrorCode = ref('')
const saving = ref(false)
const editing = ref(false)
const editingId = ref('')
const q = ref('')
const status = ref('')
const notice = ref('')
const noticeError = ref(false)
const errorCode = ref('')
const form = reactive({ username: '', displayName: '', status: 'ACTIVE', roles: [] })

const canMaintain = computed(() => listReady.value && !loading.value && !saving.value)

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
  if (loading.value) return
  loading.value = true
  listReady.value = false
  loadError.value = ''
  loadErrorCode.value = ''
  try {
    const listed = await fetchUsers(session.accessToken, { q: q.value, status: status.value })
    rows.value = listed.items
    listReady.value = true
  } catch (error) {
    loadError.value = error instanceof ApiError ? error.message : '无法读取用户'
    loadErrorCode.value = error instanceof ApiError ? error.code : ''
  } finally {
    loading.value = false
  }
}

function startCreate() {
  if (!canMaintain.value) return
  editing.value = true
  editingId.value = ''
  form.username = ''
  form.displayName = ''
  form.status = 'ACTIVE'
  form.roles = ['REGISTRAR']
  show('', false)
}

function startEdit(row) {
  if (!canMaintain.value) return
  editing.value = true
  editingId.value = row.userId
  form.username = row.username
  form.displayName = row.displayName
  form.status = row.status === 'ACTIVE' ? 'ACTIVE' : 'DISABLED'
  form.roles = [...row.roles]
  show('', false)
}

async function onSave() {
  if (!canMaintain.value) return
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
  if (!canMaintain.value) return
  saving.value = true
  const next = row.status === 'ACTIVE' ? 'DISABLED' : 'ACTIVE'
  try {
    await saveUser(session.accessToken, row.userId, { status: next })
    show(next === 'ACTIVE' ? '已启用账号' : '已停用。该账号再次登录会被拒绝。', false)
    await load()
  } catch (error) {
    show(error instanceof ApiError ? error.message : '操作失败', true, error instanceof ApiError ? error.code : '')
  } finally {
    saving.value = false
  }
}

async function onReset(row) {
  if (!canMaintain.value) return
  saving.value = true
  try {
    await resetUserPassword(session.accessToken, row.userId)
    show(`已将 ${row.username} 重置为演示初始密码，页面不回显。`, false)
  } catch (error) {
    show(error instanceof ApiError ? error.message : '重置失败', true, error instanceof ApiError ? error.code : '')
  } finally {
    saving.value = false
  }
}

onMounted(load)
</script>

<template>
  <section class="patient-panel master-data-panel" aria-labelledby="user-heading">
    <MasterPanelHeader title="用户账号" title-id="user-heading" icon="users" description="维护工作账号、显示姓名与预置业务角色。" :count="listReady ? rows.length : null" :loading="loading">
      <template #actions>
      <button class="button button-compact" type="button" data-testid="user-create" :disabled="!canMaintain" @click="startCreate">新建</button>
      </template>
    </MasterPanelHeader>
    <p class="field-hint master-inline-note">只勾选预置角色。没有权限矩阵，也没有多院区。新建与重置都使用演示初始密码 1。</p>

    <form class="master-filters" @submit.prevent="!saving && load()">
      <label class="field">
        <span>登录名 / 显示名</span>
        <input v-model="q" type="search" placeholder="搜索登录名或显示名" />
      </label>
      <label class="field">
        <span>状态</span>
        <select v-model="status">
          <option value="">全部</option>
          <option value="ACTIVE">启用</option>
          <option value="DISABLED">停用</option>
        </select>
      </label>
      <button class="button button-ghost button-compact" type="submit" :disabled="loading || saving">查询</button>
    </form>

    <form v-if="editing" class="master-form" data-testid="user-form" @submit.prevent="onSave">
      <div class="master-editor-head"><div><p class="master-eyebrow">账号维护</p><h3>{{ editingId ? '编辑用户' : '新建用户' }}</h3></div><span class="master-soft-badge">{{ editingId ? form.username : '新记录' }}</span></div>
      <fieldset class="master-form-group"><legend>账号信息</legend><div class="field-grid">
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
      </div></fieldset>
      <fieldset class="role-picks">
        <legend>角色</legend>
        <label v-for="role in MASTER_ROLES" :key="role.code" class="role-pick" :class="{ 'is-selected': form.roles.includes(role.code) }">
          <input v-model="form.roles" type="checkbox" :value="role.code" />
          <span>{{ role.label }}</span>
        </label>
      </fieldset>
      <p v-if="!editingId" class="field-hint">初始密码为 1。保存后不在列表或接口结果里回显。</p>
      <div class="panel-actions">
        <button class="button button-compact" type="submit" data-testid="user-save" :disabled="!canMaintain">
          {{ saving ? '正在保存…' : '保存' }}
        </button>
        <button class="button button-ghost button-compact" type="button" @click="editing = false">取消</button>
      </div>
    </form>

    <p v-if="notice" class="banner" :class="{ 'banner-ok': !noticeError }" :role="noticeError ? 'alert' : 'status'" data-testid="user-notice">
      {{ notice }}<span v-if="errorCode">（{{ errorCode }}）</span>
    </p>
    <p v-if="loading" class="shell-status" role="status">正在读取…</p>
    <div v-else-if="loadError" class="master-load-error" role="alert" data-testid="user-load-error">
      <p>{{ loadError }}<span v-if="loadErrorCode">（{{ loadErrorCode }}）</span></p>
      <button class="button button-ghost button-compact" type="button" data-testid="user-retry" @click="load">重新读取</button>
    </div>
    <p v-else-if="listReady && rows.length === 0" class="result-empty">{{ q || status ? '没有符合条件的用户' : '暂无用户' }}</p>
    <div v-else-if="listReady" class="result-wrap" tabindex="0" role="region" aria-label="用户账号列表，可横向滚动">
      <div class="master-table-head"><h3>账号列表</h3><span>{{ rows.length }} 条查询结果</span></div>
      <table class="register-table" data-testid="user-list">
        <caption class="sr-only">用户账号</caption>
        <thead>
          <tr>
            <th scope="col">登录名</th>
            <th scope="col">显示名</th>
            <th scope="col">角色</th>
            <th scope="col">状态</th>
            <th scope="col">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="row in rows" :key="row.userId">
            <td class="mono">{{ row.username }}</td>
            <td class="master-primary-cell"><span class="master-person-dot" aria-hidden="true">{{ row.displayName?.slice(0, 1) }}</span>{{ row.displayName }}</td>
            <td>{{ roleText(row.roles) }}</td>
            <td><span class="status-pill" :data-status="row.status">{{ statusLabel(row.status) }}</span></td>
            <td class="ticket-actions">
              <button class="text-btn" type="button" :disabled="!canMaintain" @click="startEdit(row)">编辑</button>
              <button class="text-btn" type="button" :data-testid="`user-toggle-${row.username}`" :disabled="!canMaintain" @click="toggle(row)">
                {{ row.status === 'ACTIVE' ? '停用' : '启用' }}
              </button>
              <button class="text-btn" type="button" :data-testid="`user-reset-${row.username}`" :disabled="!canMaintain" @click="onReset(row)">重置密码</button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
    <p class="master-list-foot">停用账号将无法再次登录。密码重置结果不会在列表中回显。</p>
  </section>
</template>
