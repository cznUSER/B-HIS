<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { ApiError } from '../../api/errors.js'
import { session } from '../../auth/session.js'
import { fetchAccessModules, fetchAccessPermissions, fetchAccessRoles, fetchDepartments, fetchUserPermissionOverrides, fetchUsers, resetUserPassword, saveUser } from '../../master/api.js'
import { statusLabel } from '../../master/labels.js'
import { buildUserAccessPayload } from '../../master/userAccessPayload.js'
import { buildAssignedRoleScopePayload, canEnableRoleScope, roleScopeDepartmentState } from '../../master/userRoleScopes.js'
import MasterPanelHeader from '../master/MasterPanelHeader.vue'

const rows = ref([])
const roles = ref([])
const departments = ref([])
const q = ref('')
const status = ref('')
const loading = ref(false)
const saving = ref(false)
const ready = ref(false)
const error = ref('')
const notice = ref('')
const editingId = ref('')
const editing = ref(false)
const editorLoading = ref(false)
const resetTarget = ref(null)
const form = reactive({
  username: '', displayName: '', description: '', cardNo: '', inputCode: '', password: '', status: 'ACTIVE', roles: [],
  serviceStar: '', expandInterface1: 0, expandInterface2: 0, inputCodeCompareMode: 2,
  mustChangePassword: true, cannotModifyPassword: false, accountLocked: false, accountStopped: false,
  hidden: false, system: false, super: false, developer: false,
  lastLoginAt: '', lastLoginIp: '', lockUntil: '', passwordChangedAt: '', failedLoginCount: 0,
})
const overridePermissions = ref([])
const overrideModules = ref([])
const overrideRows = ref([])
const scopeRows = ref([])
const scopeDepartment = reactive({})
const overrideDraft = reactive({ permissionCode: '', moduleCode: '', departmentId: '', effect: 'ALLOW' })
const canMaintain = computed(() => ready.value && !loading.value && !saving.value && !editorLoading.value)
const canManageProtected = computed(() => session.user?.super === true || session.user?.developer === true)
const canEditDeveloperFlags = computed(() => session.user?.developer === true)
const canEditSuperFlag = computed(() => session.user?.super === true || session.user?.developer === true)

function failMessage(value) { return value instanceof ApiError ? value.message : '请求失败' }
function roleName(code) { return roles.value.find((item) => item.code === code)?.name || code }
function departmentName(id) { return departments.value.find((item) => item.departmentId === id)?.deptName || `科室 ${id}（目录中不可用）` }
function activeDepartments() { return departments.value.filter((department) => department.status === 'ACTIVE') }
function editableScopeRoles() {
  return roles.value.filter((role) => form.roles.includes(role.code) && role.status === 'ACTIVE')
}
function roleText(value) { return (Array.isArray(value) ? value : []).map(roleName).join('、') || '—' }
function formatTime(value) {
  if (!value) return '—'
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString('zh-CN', { hour12: false })
}
function protectedUser(row) { return row.hidden || row.system || row.super || row.developer }
function assignForm(row = {}) {
  Object.assign(form, {
    username: row.username || '', displayName: row.displayName || '', description: row.description || '',
    cardNo: row.cardNo || '', inputCode: row.inputCode || '', password: '', status: row.status === 'ACTIVE' ? 'ACTIVE' : 'DISABLED',
    serviceStar: row.serviceStar == null ? '' : row.serviceStar,
    expandInterface1: row.expandInterface1 ?? 0, expandInterface2: row.expandInterface2 ?? 0,
    inputCodeCompareMode: row.inputCodeCompareMode ?? 2,
    roles: [...(row.roles || [])], mustChangePassword: row.mustChangePassword === true,
    cannotModifyPassword: row.cannotModifyPassword === true, accountLocked: row.accountLocked === true,
    accountStopped: row.accountStopped === true, hidden: row.hidden === true, system: row.system === true,
    super: row.super === true, developer: row.developer === true, lastLoginAt: row.lastLoginAt || '',
    lastLoginIp: row.lastLoginIp || '', lockUntil: row.lockUntil || '', passwordChangedAt: row.passwordChangedAt || '',
    failedLoginCount: Number(row.failedLoginCount) || 0,
  })
}

async function load() {
  loading.value = true; ready.value = false; error.value = ''
  try {
    const [listed, roleList, permissionList, moduleList, departmentList] = await Promise.all([fetchUsers(session.accessToken, { q: q.value, status: status.value }), fetchAccessRoles(session.accessToken), fetchAccessPermissions(session.accessToken), fetchAccessModules(session.accessToken), fetchDepartments(session.accessToken, {})])
    rows.value = listed.items || []; roles.value = roleList.items || []; overridePermissions.value = permissionList.items || []; overrideModules.value = moduleList.items || []; departments.value = departmentList.items || []; ready.value = true
  } catch (e) { error.value = failMessage(e) } finally { loading.value = false }
}
function create() {
  editing.value = true; editingId.value = ''; overrideRows.value = []; scopeRows.value = []; resetTarget.value = null
  Object.assign(overrideDraft, { permissionCode: '', moduleCode: '', departmentId: '', effect: 'ALLOW' })
  assignForm({ status: 'ACTIVE', mustChangePassword: true, roles: roles.value.filter((r) => !r.system).slice(0, 1).map((r) => r.code) })
  notice.value = ''
}
async function edit(row) {
  if (protectedUser(row) && !canManageProtected.value) { notice.value = '该账号包含受保护属性，需要系统管理员、超级或开发者账号维护'; return }
  editing.value = true; editorLoading.value = true; editingId.value = row.userId; overrideRows.value = []; scopeRows.value = (row.roleScopes || []).map((scope) => ({ roleCode: scope.roleCode, departmentId: scope.departmentId, status: scope.status }))
  assignForm(row); notice.value = ''
  try { const listed = await fetchUserPermissionOverrides(session.accessToken, row.userId); overrideRows.value = listed.items || [] }
  catch (e) { notice.value = `读取权限例外失败：${failMessage(e)}`; editing.value = false }
  finally { editorLoading.value = false }
}
async function save() {
  if (!canMaintain.value) return
  saving.value = true
  const body = buildUserAccessPayload(form, {
    creating: !editingId.value,
    canEditDeveloperFlags: canEditDeveloperFlags.value,
    canEditSuperFlag: canEditSuperFlag.value,
    roleScopes: buildAssignedRoleScopePayload(scopeRows.value, form.roles),
  })
  body.permissionOverrides = overrideRows.value.map((item) => ({
    permissionCode: item.permissionCode,
    moduleCode: item.moduleCode || '',
    departmentId: item.departmentId || '',
    effect: item.effect,
  }))
  try {
    if (!editingId.value) {
      const saved = await saveUser(session.accessToken, '', body)
      editingId.value = saved?.userId || ''
      if (!editingId.value) throw new Error('用户已保存，但接口未返回用户编号')
    } else {
      await saveUser(session.accessToken, editingId.value, body)
    }
    editing.value = false; editingId.value = ''; notice.value = '用户及权限例外已保存'; await load()
  } catch (e) { notice.value = failMessage(e) } finally { saving.value = false }
}
async function toggle(row) {
  if (!canMaintain.value) return
  saving.value = true
  const enabling = row.status !== 'ACTIVE'
  const body = { status: enabling ? 'ACTIVE' : 'DISABLED' }
  if (enabling && row.accountStopped) body.accountStopped = false
  try { await saveUser(session.accessToken, row.userId, body); notice.value = '账号状态已更新'; await load() }
  catch (e) { notice.value = failMessage(e) } finally { saving.value = false }
}
function requestReset(row) { resetTarget.value = row }
async function confirmReset() {
  if (!canMaintain.value) return
  const target = resetTarget.value
  if (!target) return
  saving.value = true
  try {
    await resetUserPassword(session.accessToken, target.userId)
    resetTarget.value = null
    notice.value = '密码已重置；账号下次登录需先修改密码'
    await load()
  } catch (e) { notice.value = failMessage(e) } finally { saving.value = false }
}
function setCannotModifyPassword() { if (form.cannotModifyPassword) form.mustChangePassword = false }
function setMustChangePassword() { if (form.mustChangePassword) form.cannotModifyPassword = false }
function permissionName(code) { return overridePermissions.value.find((item) => item.code === code)?.name || code }
function moduleName(code) { return overrideModules.value.find((item) => item.code === code)?.name || code || '全模块' }
function addOverride() {
  const draft = { ...overrideDraft, permissionCode: overrideDraft.permissionCode.trim(), moduleCode: overrideDraft.moduleCode.trim(), departmentId: overrideDraft.departmentId.trim() }
  if (!draft.permissionCode || !draft.effect) return
  if (overrideRows.value.some((item) => item.permissionCode === draft.permissionCode && (item.moduleCode || '') === draft.moduleCode && (item.departmentId || '') === draft.departmentId)) {
    notice.value = '相同权限和范围的例外已存在'; return
  }
  overrideRows.value.push(draft)
  Object.assign(overrideDraft, { permissionCode: '', moduleCode: '', departmentId: '', effect: 'ALLOW' })
  notice.value = ''
}
function removeOverride(index) { overrideRows.value.splice(index, 1) }
function addScope(roleCode) {
  const departmentId = scopeDepartment[roleCode] || ''
  if (!departmentId || scopeRows.value.some((scope) => scope.roleCode === roleCode && scope.departmentId === departmentId)) return
  scopeRows.value.push({ roleCode, departmentId, status: 'ACTIVE' })
  scopeDepartment[roleCode] = ''
  notice.value = ''
}
function toggleScope(scope) {
  const target = scopeRows.value[scope.index]
  if (!target) return
  target.status = target.status === 'ACTIVE' ? 'DISABLED' : 'ACTIVE'
}
function removeScope(index) { scopeRows.value.splice(index, 1) }
function clearRoleScopes(roleCode) { scopeRows.value = scopeRows.value.filter((scope) => scope.roleCode !== roleCode) }
function scopesForRole(roleCode) { return scopeRows.value.map((scope, index) => ({ ...scope, index })).filter((scope) => scope.roleCode === roleCode) }
onMounted(load)
</script>

<template>
  <section class="patient-panel master-data-panel access-data-panel" aria-labelledby="access-users-heading">
    <MasterPanelHeader title="用户账号" title-id="access-users-heading" icon="users" description="维护账号资料、登录控制、角色权限与个人例外。" :count="ready ? rows.length : null" :loading="loading">
      <template #actions><button class="button button-compact" type="button" :disabled="!canMaintain" @click="create">新建用户</button></template>
    </MasterPanelHeader>
    <form class="master-filters" @submit.prevent="load">
      <label class="field"><span>登录名 / 显示名</span><input v-model="q" type="search" placeholder="搜索账号或人员" /></label>
      <label class="field"><span>状态</span><select v-model="status"><option value="">全部</option><option value="ACTIVE">启用</option><option value="DISABLED">停用</option></select></label>
      <button class="button button-ghost button-compact" type="submit" :disabled="loading || saving">查询</button>
    </form>
    <form v-if="editing" class="master-form" data-testid="access-user-form" @submit.prevent="save">
      <div class="master-editor-head"><div><p class="master-eyebrow">ACCOUNT PROFILE</p><h3>{{ editingId ? '编辑用户' : '新建用户' }}</h3></div><span class="master-soft-badge">{{ editingId ? form.username : '新记录' }}</span></div>
      <p v-if="editorLoading" class="shell-status">正在读取用户权限例外…</p>
      <div class="field-grid">
        <label class="field"><span>登录名</span><input v-model="form.username" :disabled="Boolean(editingId)" required autocomplete="off" /></label>
        <label class="field"><span>显示名</span><input v-model="form.displayName" required maxlength="128" /></label>
        <label class="field"><span>状态</span><select v-model="form.status"><option value="ACTIVE">启用</option><option value="DISABLED">停用</option></select></label>
        <label class="field"><span>助记码</span><input v-model="form.inputCode" maxlength="64" /></label>
        <label class="field"><span>{{ editingId ? '新密码（留空不变）' : '初始密码（可选）' }}</span><input v-model="form.password" type="password" minlength="8" maxlength="72" autocomplete="new-password" :placeholder="editingId ? '留空保持原密码' : '省略则使用默认初始口令'" /></label>
        <label class="field"><span>卡号</span><input v-model="form.cardNo" maxlength="64" :disabled="!canManageProtected" /></label>
        <label class="field"><span>用户说明</span><input v-model="form.description" maxlength="512" /></label>
      </div>
      <details class="access-advanced-fields">
        <summary>扩展用户字段</summary>
        <div class="field-grid">
          <label class="field"><span>服务星级（可空，1–5）</span><input v-model="form.serviceStar" type="number" min="1" max="5" step="1" /></label>
          <label class="field"><span>扩展参数 1（数值）</span><input v-model.number="form.expandInterface1" type="number" step="1" /></label>
          <label class="field"><span>扩展参数 2（数值）</span><input v-model.number="form.expandInterface2" type="number" step="1" /></label>
          <label class="field"><span>助记码比较模式代码</span><input v-model.number="form.inputCodeCompareMode" type="number" step="1" /></label>
        </div>
      </details>
      <fieldset class="role-picks"><legend>角色与权限并集</legend><label v-for="role in roles" :key="role.code" class="role-pick" :class="{ 'is-selected': form.roles.includes(role.code), 'is-disabled': role.status !== 'ACTIVE' }"><input v-model="form.roles" type="checkbox" :value="role.code" :disabled="(role.status !== 'ACTIVE' && !form.roles.includes(role.code)) || saving || editorLoading" /><span>{{ role.name }}（{{ role.code }}）<small v-if="role.status !== 'ACTIVE'">已停用，仅可移除</small></span></label></fieldset>
      <fieldset class="role-picks access-scope-fieldset"><legend>角色科室范围</legend>
        <p v-if="!editableScopeRoles().length" class="access-empty-note">请先选择有效角色。</p>
        <section v-for="role in editableScopeRoles()" :key="role.code" class="access-scope-role" :data-role-code="role.code">
          <div class="access-scope-head"><strong>{{ role.name }}（{{ role.code }}）</strong><button class="text-btn" type="button" :disabled="saving || editorLoading" @click="clearRoleScopes(role.code)">清空范围</button></div>
          <div class="field-grid access-scope-draft"><label class="field"><span>添加科室</span><select v-model="scopeDepartment[role.code]"><option value="">选择启用科室</option><option v-for="department in activeDepartments()" :key="department.departmentId" :value="department.departmentId">{{ department.deptName }}（{{ department.deptCode }}）</option></select></label><button class="button button-ghost button-compact" type="button" :disabled="!scopeDepartment[role.code] || saving || editorLoading" @click="addScope(role.code)">添加科室</button></div>
          <div v-if="scopesForRole(role.code).length" class="access-override-list">
            <div v-for="scope in scopesForRole(role.code)" :key="`${scope.roleCode}-${scope.departmentId}`" class="access-override-row" :data-scope-status="scope.status">
              <div><strong>{{ departmentName(scope.departmentId) }}</strong><code>{{ roleScopeDepartmentState(scope, departments).label }} · {{ scope.departmentId }}</code></div>
              <div class="access-scope-actions"><button class="text-btn" type="button" :disabled="saving || editorLoading || scope.status !== 'ACTIVE' && !canEnableRoleScope(scope, departments)" @click="toggleScope(scope)">{{ scope.status === 'ACTIVE' ? '停用' : '重新启用' }}</button><button class="text-btn" type="button" :disabled="saving || editorLoading" @click="removeScope(scope.index)">移除范围</button></div>
            </div>
          </div><p v-else class="access-empty-note">未配置显式范围，将沿用角色默认科室范围。</p>
        </section>
      </fieldset>
      <fieldset class="role-picks access-user-controls"><legend>账号控制</legend>
        <div class="access-checks">
          <label><input v-model="form.mustChangePassword" type="checkbox" :disabled="form.cannotModifyPassword" @change="setMustChangePassword" />下次登录强制修改密码</label>
          <label><input v-model="form.cannotModifyPassword" type="checkbox" @change="setCannotModifyPassword" />禁止自行修改密码</label>
          <label v-if="editingId"><input v-model="form.accountLocked" type="checkbox" />锁定账号</label>
          <label><input v-model="form.accountStopped" type="checkbox" />停用账号</label>
          <label><input v-model="form.hidden" type="checkbox" :disabled="!canEditDeveloperFlags" />隐藏账号</label>
          <label><input v-model="form.system" type="checkbox" :disabled="!canEditDeveloperFlags" />系统账号</label>
          <label><input v-model="form.super" type="checkbox" :disabled="!canEditSuperFlag" />超级账号</label>
          <label><input v-model="form.developer" type="checkbox" :disabled="!canEditDeveloperFlags" />开发者账号</label>
        </div>
        <p v-if="!canManageProtected" class="access-inline-note">卡号和超级账号由超级或开发者维护；隐藏、系统及开发者标记仅由开发者维护。</p>
      </fieldset>
      <div v-if="editingId" class="access-user-audit" aria-label="账号活动记录">
        <p><span>最后登录</span><strong>{{ formatTime(form.lastLoginAt) }}</strong></p>
        <p><span>登录 IP</span><strong>{{ form.lastLoginIp || '—' }}</strong></p>
        <p><span>锁定截止</span><strong>{{ formatTime(form.lockUntil) }}</strong></p>
        <p><span>密码更新</span><strong>{{ formatTime(form.passwordChangedAt) }}</strong></p>
        <p><span>连续失败次数</span><strong>{{ form.failedLoginCount }}</strong></p>
      </div>
      <fieldset class="role-picks access-override-fieldset"><legend>用户权限例外</legend><div class="field-grid access-override-draft"><label class="field"><span>权限</span><select v-model="overrideDraft.permissionCode"><option value="">选择权限</option><option v-for="item in overridePermissions" :key="item.code" :value="item.code">{{ item.name }}（{{ item.code }}）</option></select></label><label class="field"><span>模块范围</span><select v-model="overrideDraft.moduleCode"><option value="">全模块</option><option v-for="item in overrideModules" :key="item.code" :value="item.code">{{ item.name }}</option></select></label><label class="field"><span>科室范围</span><input v-model="overrideDraft.departmentId" placeholder="留空表示全科室" /></label><label class="field"><span>授权效果</span><select v-model="overrideDraft.effect"><option value="ALLOW">明确允许</option><option value="DENY">明确拒绝</option></select></label><button class="button button-ghost button-compact" type="button" :disabled="!overrideDraft.permissionCode || editorLoading || saving" @click="addOverride">添加例外</button></div><div v-if="overrideRows.length" class="access-override-list"><div v-for="(item, index) in overrideRows" :key="`${item.permissionCode}-${item.moduleCode}-${item.departmentId}`" class="access-override-row"><div><strong>{{ permissionName(item.permissionCode) }} · {{ item.effect === 'DENY' ? '明确拒绝' : '明确允许' }}</strong><code>{{ item.permissionCode }} · {{ moduleName(item.moduleCode) }}<span v-if="item.departmentId"> · 科室 {{ item.departmentId }}</span></code></div><button class="text-btn" type="button" :disabled="saving" :aria-label="`移除${permissionName(item.permissionCode)}权限例外`" @click="removeOverride(index)">移除</button></div></div><p v-else class="access-empty-note">未设置例外，用户仅继承角色权限。</p></fieldset>
      <div class="panel-actions"><button class="button button-compact" type="submit" :disabled="!canMaintain || form.cannotModifyPassword && form.mustChangePassword">{{ saving ? '正在保存…' : '保存' }}</button><button class="button button-ghost button-compact" type="button" :disabled="saving" @click="editing = false">取消</button></div>
    </form>
    <p v-if="notice" class="banner" role="status">{{ notice }}</p>
    <p v-if="loading" class="shell-status">正在读取用户…</p>
    <p v-else-if="error" class="banner" role="alert">{{ error }} <button class="button button-ghost button-compact" type="button" @click="load">重试</button></p>
    <p v-else-if="ready && !rows.length" class="result-empty">暂无符合条件的用户</p>
    <div v-else-if="ready" class="result-wrap" tabindex="0"><div class="master-table-head"><h3>账号列表</h3><span>{{ rows.length }} 条</span></div><table class="register-table"><thead><tr><th>登录名</th><th>用户资料</th><th>角色</th><th>账号状态</th><th>最近活动</th><th>操作</th></tr></thead><tbody><tr v-for="row in rows" :key="row.userId"><td class="mono">{{ row.username }}</td><td class="master-primary-cell">{{ row.displayName }}<small>{{ row.description || row.inputCode || '—' }}</small></td><td>{{ roleText(row.roles) }}</td><td><div class="access-user-badges"><span class="status-pill" :data-status="row.status">{{ statusLabel(row.status) }}</span><span v-if="row.mustChangePassword" class="access-user-badge is-warning">需改密</span><span v-if="row.accountLocked" class="access-user-badge is-danger">锁定</span><span v-if="row.accountStopped" class="access-user-badge is-muted">账号停用</span><span v-if="row.hidden || row.system || row.super || row.developer" class="access-user-badge">受保护</span></div></td><td><span>{{ formatTime(row.lastLoginAt) }}</span><small class="access-user-activity">{{ row.lastLoginIp || '无登录记录' }}<span v-if="row.failedLoginCount"> · 失败 {{ row.failedLoginCount }} 次</span></small><small v-if="row.lockUntil" class="access-user-activity">锁定至 {{ formatTime(row.lockUntil) }}</small></td><td class="ticket-actions"><button class="text-btn" type="button" :disabled="!canMaintain || (protectedUser(row) && !canManageProtected)" :title="protectedUser(row) && !canManageProtected ? '需要系统管理员、超级或开发者账号' : ''" @click="edit(row)">用户与权限</button><button class="text-btn" type="button" :disabled="!canMaintain" @click="toggle(row)">{{ row.status === 'ACTIVE' ? '停用' : '启用' }}</button><button class="text-btn" type="button" :disabled="!canMaintain" @click="requestReset(row)">重置密码</button></td></tr></tbody></table></div>
    <p class="master-list-foot">重置密码后账号下次登录需先修改密码；锁定、停用和角色权限分别生效。</p>
    <div v-if="resetTarget" class="access-reset-overlay" role="presentation" @click.self="resetTarget = null"><section class="access-reset-dialog" role="dialog" aria-modal="true" aria-labelledby="access-reset-heading"><div class="master-editor-head"><div><p class="master-eyebrow">RESET PASSWORD</p><h3 id="access-reset-heading">确认重置密码</h3></div><span class="master-soft-badge">{{ resetTarget.username }}</span></div><p>将重置 {{ resetTarget.displayName || resetTarget.username }} 的密码。该账号下次登录时必须设置新密码后才能继续使用。</p><div class="panel-actions"><button class="button button-compact" type="button" :disabled="saving" @click="confirmReset">{{ saving ? '正在重置…' : '确认重置' }}</button><button class="button button-ghost button-compact" type="button" :disabled="saving" @click="resetTarget = null">取消</button></div></section></div>
  </section>
</template>
