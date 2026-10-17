<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { ApiError } from '../../api/errors.js'
import { session } from '../../auth/session.js'
import { fetchAccessCategories, fetchAccessModes, fetchAccessModules, fetchAccessPermissions, saveAccessModule, saveAccessPermission, savePermissionCategory, savePermissionMode } from '../../master/api.js'
import MasterPanelHeader from '../master/MasterPanelHeader.vue'

const rows = ref([])
const modules = ref([])
const categories = ref([])
const modes = ref([])
const moduleCode = ref('')
const categoryId = ref('')
const loading = ref(false)
const ready = ref(false)
const saving = ref(false)
const error = ref('')
const notice = ref('')
const editing = ref(false)
const editingCode = ref('')
const catalogTab = ref('permissions')
const catalogEditing = ref(false)
const catalogEditingKey = ref('')
const catalogForm = reactive({ code: '', name: '', description: '', status: 'ACTIVE', hidden: false, departmentFunctions: '0', needSpecifyDepartment: false, secondPermissionId: '', homePageUrl: '' })
const form = reactive({ code: '', name: '', description: '', moduleCode: '', categoryId: '', modeCode: 'password', secondPermissionCode: '', dbRoleName: '', needSecondPermission: false, actionsText: '', status: 'ACTIVE', hidden: false })

const filtered = computed(() => rows.value.filter((row) => (!moduleCode.value || row.moduleCode === moduleCode.value) && (!categoryId.value || row.categoryId === categoryId.value)))
const catalogRows = computed(() => ({ modules: modules.value, categories: categories.value, modes: modes.value })[catalogTab.value] || [])
const currentCount = computed(() => catalogTab.value === 'permissions' ? filtered.value.length : catalogRows.value.length)
const canMaintain = computed(() => ready.value && !loading.value && !saving.value)
const canEditProtected = computed(() => session.user?.developer === true)
function message(e) { return e instanceof ApiError ? e.message : e instanceof Error ? e.message : '请求失败' }
function categoryCode(item) { return item.code || String(item.id || '').replace(/^category-/, '') }
function moduleName(code) { return modules.value.find((item) => item.code === code)?.name || code || '全模块' }
function categoryName(id) { return categories.value.find((item) => categoryCode(item) === id)?.name || id || '—' }
function modeName(code) { return modes.value.find((item) => item.code === code)?.name || code || '—' }
function catalogIdentifier(row) { return catalogTab.value === 'categories' ? categoryCode(row) : row.code }

function switchCatalogTab(tab) {
  catalogTab.value = tab
  editing.value = false
  catalogEditing.value = false
  notice.value = ''
}

async function load() {
  loading.value = true
  ready.value = false
  error.value = ''
  try {
    const [list, moduleList, categoryList, modeList] = await Promise.all([
      fetchAccessPermissions(session.accessToken),
      fetchAccessModules(session.accessToken),
      fetchAccessCategories(session.accessToken),
      fetchAccessModes(session.accessToken),
    ])
    rows.value = list.items || []
    modules.value = moduleList.items || []
    categories.value = categoryList.items || []
    modes.value = modeList.items || []
    ready.value = true
  } catch (e) {
    error.value = message(e)
  } finally {
    loading.value = false
  }
}

function create() {
  catalogEditing.value = false
  editing.value = true
  editingCode.value = ''
  Object.assign(form, { code: '', name: '', description: '', moduleCode: modules.value[0]?.code || '', categoryId: categories.value.length ? categoryCode(categories.value[0]) : '', modeCode: modes.value[0]?.code || 'password', secondPermissionCode: '', dbRoleName: '', needSecondPermission: false, actionsText: '', status: 'ACTIVE', hidden: false })
  notice.value = ''
}

function edit(row) {
  catalogEditing.value = false
  editing.value = true
  editingCode.value = row.code
  Object.assign(form, { code: row.code, name: row.name, description: row.description || '', moduleCode: row.moduleCode || '', categoryId: row.categoryId || '', modeCode: row.modeCode || '', secondPermissionCode: row.secondPermissionCode || '', dbRoleName: row.dbRoleName || '', needSecondPermission: Boolean(row.needSecondPermission), actionsText: (row.actions || []).join(', '), status: row.status || 'ACTIVE', hidden: Boolean(row.hidden) })
  notice.value = ''
}

async function save() {
  if (!canMaintain.value) return
  saving.value = true
  try {
    await saveAccessPermission(session.accessToken, editingCode.value, {
      code: form.code,
      name: form.name,
      description: form.description,
      moduleCode: form.moduleCode,
      categoryId: form.categoryId,
      modeCode: form.modeCode,
      secondPermissionCode: form.secondPermissionCode,
      dbRoleName: form.dbRoleName,
      needSecondPermission: form.needSecondPermission,
      actions: form.actionsText.split(/[\s,，、;；]+/).map((item) => item.trim()).filter(Boolean),
      status: form.status,
      hidden: form.hidden,
    })
    editing.value = false
    notice.value = '权限配置已保存'
    await load()
  } catch (e) {
    notice.value = message(e)
  } finally {
    saving.value = false
  }
}

function createCatalogItem() {
  editing.value = false
  catalogEditing.value = true
  catalogEditingKey.value = ''
  Object.assign(catalogForm, { code: '', name: '', description: '', status: 'ACTIVE', hidden: false, departmentFunctions: '0', needSpecifyDepartment: false, secondPermissionId: '', homePageUrl: '' })
  notice.value = ''
}

function editCatalogItem(row) {
  editing.value = false
  catalogEditing.value = true
  catalogEditingKey.value = catalogIdentifier(row)
  Object.assign(catalogForm, { code: catalogEditingKey.value, name: row.name || '', description: row.description || '', status: row.status || 'ACTIVE', hidden: Boolean(row.hidden), departmentFunctions: row.departmentFunctions || '0', needSpecifyDepartment: Boolean(row.needSpecifyDepartment), secondPermissionId: row.secondPermissionId || '', homePageUrl: row.homePageUrl || '' })
  notice.value = ''
}

async function saveCatalogItem() {
  if (!canMaintain.value) return
  saving.value = true
  const payload = { name: catalogForm.name, description: catalogForm.description, status: catalogForm.status, hidden: catalogForm.hidden, departmentFunctions: catalogForm.departmentFunctions, needSpecifyDepartment: catalogForm.needSpecifyDepartment, secondPermissionId: catalogForm.secondPermissionId, homePageUrl: catalogForm.homePageUrl }
  if (!catalogEditingKey.value) {
    if (catalogTab.value === 'categories') payload.id = catalogForm.code
    else payload.code = catalogForm.code
  }
  try {
    if (catalogTab.value === 'modules') await saveAccessModule(session.accessToken, catalogEditingKey.value, payload)
    else if (catalogTab.value === 'categories') await savePermissionCategory(session.accessToken, catalogEditingKey.value, payload)
    else if (catalogTab.value === 'modes') await savePermissionMode(session.accessToken, catalogEditingKey.value, { code: payload.code, name: payload.name, status: payload.status })
    catalogEditing.value = false
    notice.value = '目录元数据已保存'
    await load()
  } catch (e) {
    notice.value = message(e)
  } finally {
    saving.value = false
  }
}

onMounted(load)
</script>

<template>
  <section class="patient-panel master-data-panel access-data-panel" aria-labelledby="access-permissions-heading">
    <MasterPanelHeader title="权限目录" title-id="access-permissions-heading" icon="dicts" description="维护权限项、模块、分类和认证模式目录。" :count="ready ? currentCount : null" :loading="loading">
      <template #actions>
        <button v-if="catalogTab === 'permissions'" class="button button-compact" type="button" :disabled="!canMaintain" @click="create">新增权限</button>
        <button v-else class="button button-compact" type="button" :disabled="!canMaintain" @click="createCatalogItem">新增{{ catalogTab === 'modules' ? '模块' : catalogTab === 'categories' ? '分类' : '认证模式' }}</button>
      </template>
    </MasterPanelHeader>
    <div class="access-catalog-tabs" role="tablist" aria-label="权限目录类型">
      <button v-for="tab in [{ code: 'permissions', name: '权限项' }, { code: 'modules', name: '模块' }, { code: 'categories', name: '分类' }, { code: 'modes', name: '认证模式' }]" :key="tab.code" class="access-catalog-tab" :class="{ 'is-active': catalogTab === tab.code }" type="button" role="tab" :aria-selected="catalogTab === tab.code" @click="switchCatalogTab(tab.code)">{{ tab.name }}</button>
    </div>
    <div v-if="catalogTab === 'permissions'" class="master-filters">
      <label class="field"><span>模块</span><select v-model="moduleCode"><option value="">全部模块</option><option v-for="item in modules" :key="item.code" :value="item.code">{{ item.name }}</option></select></label>
      <label class="field"><span>分类</span><select v-model="categoryId"><option value="">全部分类</option><option v-for="item in categories" :key="item.id || item.code" :value="categoryCode(item)">{{ item.name }}</option></select></label>
      <button class="button button-ghost button-compact" type="button" :disabled="loading" @click="load">刷新</button>
    </div>
    <form v-if="editing && catalogTab === 'permissions'" class="master-form access-permission-form" data-testid="access-permission-form" @submit.prevent="save">
      <div class="master-editor-head"><div><p class="master-eyebrow">PERMISSION DEFINITION</p><h3>{{ editingCode ? '编辑权限' : '新增权限' }}</h3></div><span class="master-soft-badge">{{ editingCode || '新记录' }}</span></div>
      <div class="field-grid">
        <label class="field"><span>权限编码</span><input v-model="form.code" :disabled="Boolean(editingCode)" required maxlength="128" placeholder="例如 patient:create" /></label>
        <label class="field"><span>权限名称</span><input v-model="form.name" required maxlength="128" /></label>
        <label class="field"><span>所属模块</span><select v-model="form.moduleCode" required><option value="" disabled>选择模块</option><option v-for="item in modules" :key="item.code" :value="item.code">{{ item.name }}（{{ item.code }}）</option></select></label>
        <label class="field"><span>权限分类</span><select v-model="form.categoryId" required><option value="" disabled>选择分类</option><option v-for="item in categories" :key="item.id || item.code" :value="categoryCode(item)">{{ item.name }}</option></select></label>
        <label class="field"><span>认证模式</span><select v-model="form.modeCode" required><option v-for="item in modes" :key="item.code" :value="item.code">{{ item.name }}（{{ item.code }}）</option></select></label>
        <label class="field"><span>状态</span><select v-model="form.status"><option value="ACTIVE">启用</option><option value="INACTIVE">停用</option><option value="DISABLED">禁用</option></select></label>
        <label class="field access-permission-wide"><span>二次权限编码（可选）</span><select v-model="form.secondPermissionCode"><option value="">不配置</option><option v-for="item in rows.filter((row) => row.code !== editingCode)" :key="item.code" :value="item.code">{{ item.name }}（{{ item.code }}）</option></select></label>
        <label class="field access-permission-wide"><span>说明</span><textarea v-model="form.description" maxlength="512" rows="2"></textarea></label>
        <label class="field access-permission-wide"><span>动作代码</span><input v-model="form.actionsText" placeholder="N, M, D, S" /><small>按逗号、空格或顿号分隔，例如 N、M、D、S。</small></label>
      </div>
      <label class="access-check"><input v-model="form.hidden" type="checkbox" :disabled="!canEditProtected" />隐藏权限</label>
      <label class="access-check"><input v-model="form.needSecondPermission" type="checkbox" />标记需要二次权限</label>
      <details class="access-advanced-fields">
        <summary>华易通权限映射</summary>
        <div class="field-grid"><label class="field"><span>数据库角色名</span><input v-model="form.dbRoleName" maxlength="128" /></label></div>
      </details>
      <p class="access-inline-note">二次权限只保存目录关系；临床规则运行开关保持关闭，配置不会自动启用临床校验。隐藏权限仅由开发者维护。</p>
      <div class="panel-actions"><button class="button button-compact" type="submit" :disabled="!canMaintain">{{ saving ? '正在保存…' : '保存权限' }}</button><button class="button button-ghost button-compact" type="button" :disabled="saving" @click="editing = false">取消</button></div>
    </form>
    <form v-if="catalogEditing && catalogTab !== 'permissions'" class="master-form access-catalog-form" data-testid="access-catalog-form" @submit.prevent="saveCatalogItem">
      <div class="master-editor-head"><div><p class="master-eyebrow">ACCESS CATALOGUE</p><h3>{{ catalogEditingKey ? '编辑' : '新增' }}{{ catalogTab === 'modules' ? '模块' : catalogTab === 'categories' ? '分类' : '认证模式' }}</h3></div><span class="master-soft-badge">{{ catalogEditingKey || '新记录' }}</span></div>
      <div class="field-grid">
        <label v-if="catalogTab !== 'modules'" class="field"><span>{{ catalogTab === 'categories' ? '分类编码' : '模式编码' }}</span><input v-model="catalogForm.code" :disabled="Boolean(catalogEditingKey)" required maxlength="64" /></label>
        <label v-else class="field"><span>模块编码</span><input v-model="catalogForm.code" :disabled="Boolean(catalogEditingKey)" required maxlength="64" /></label>
        <label class="field"><span>{{ catalogTab === 'modes' ? '认证模式名称' : '名称' }}</span><input v-model="catalogForm.name" required maxlength="128" /></label>
        <label class="field"><span>状态</span><select v-model="catalogForm.status"><option value="ACTIVE">启用</option><option value="INACTIVE">停用</option><option value="DISABLED">禁用</option></select></label>
        <label v-if="catalogTab !== 'modes'" class="field access-catalog-wide"><span>说明</span><textarea v-model="catalogForm.description" rows="2" maxlength="512"></textarea></label>
      </div>
      <label v-if="catalogTab === 'modules'" class="access-check"><input v-model="catalogForm.hidden" type="checkbox" />隐藏模块</label>
      <details v-if="catalogTab === 'modules'" class="access-advanced-fields">
        <summary>华易通模块扩展项</summary>
        <div class="field-grid">
          <label class="field"><span>科室功能值（十进制字符串）</span><input v-model="catalogForm.departmentFunctions" inputmode="numeric" pattern="-?[0-9]+" /></label>
          <label class="field"><span>二次权限记录号</span><input v-model="catalogForm.secondPermissionId" type="number" min="0" step="1" /></label>
          <label class="field"><span>首页地址</span><input v-model="catalogForm.homePageUrl" maxlength="512" /></label>
        </div>
        <label class="access-check"><input v-model="catalogForm.needSpecifyDepartment" type="checkbox" />要求指定科室</label>
        <p class="access-inline-note">科室功能值按十进制原样保存，不在此解释或换算。</p>
      </details>
      <p v-if="catalogTab === 'modes'" class="access-inline-note">认证模式只维护目录元数据；工卡登录和临床二次认证保持关闭。</p>
      <div class="panel-actions"><button class="button button-compact" type="submit" :disabled="!canMaintain">{{ saving ? '正在保存…' : '保存目录项' }}</button><button class="button button-ghost button-compact" type="button" :disabled="saving" @click="catalogEditing = false">取消</button></div>
    </form>
    <p v-if="notice" class="banner" role="status">{{ notice }}</p>
    <p v-if="loading" class="shell-status">正在读取权限目录…</p>
    <p v-else-if="error" class="banner" role="alert">{{ error }} <button class="button button-ghost button-compact" type="button" @click="load">重试</button></p>
    <p v-else-if="ready && currentCount === 0" class="result-empty">暂无{{ catalogTab === 'permissions' ? '权限目录' : catalogTab === 'modules' ? '模块' : catalogTab === 'categories' ? '权限分类' : '认证模式' }}</p>
    <div v-else-if="ready && catalogTab === 'permissions'" class="result-wrap" tabindex="0" role="region" aria-label="权限目录列表，可横向滚动">
      <div class="master-table-head"><h3>权限标识</h3><span>{{ filtered.length }} 条</span></div>
      <table class="register-table"><thead><tr><th>权限编码</th><th>名称</th><th>模块 / 分类</th><th>动作</th><th>认证模式</th><th>二次权限</th><th>状态</th><th>操作</th></tr></thead>
        <tbody><tr v-for="row in filtered" :key="row.code"><td class="mono">{{ row.code }}</td><td class="master-primary-cell">{{ row.name }}</td><td>{{ moduleName(row.moduleCode) }} / {{ categoryName(row.categoryId) }}</td><td>{{ (row.actions || []).join(', ') || '—' }}</td><td><span class="master-type-tag">{{ modeName(row.modeCode) }}</span></td><td>{{ row.secondPermissionCode || '—' }}</td><td><span class="status-pill" :data-status="row.status">{{ row.status }}</span><span v-if="row.hidden" class="master-type-tag">隐藏</span></td><td class="ticket-actions"><button class="text-btn" type="button" :disabled="!canMaintain" @click="edit(row)">编辑</button></td></tr></tbody>
      </table>
    </div>
    <div v-else-if="ready && catalogTab === 'modules'" class="result-wrap" tabindex="0" role="region" aria-label="模块目录列表，可横向滚动">
      <div class="master-table-head"><h3>模块</h3><span>{{ modules.length }} 条</span></div>
      <table class="register-table"><thead><tr><th>模块编码</th><th>名称 / 首页</th><th>科室功能值</th><th>科室要求</th><th>二次权限</th><th>说明</th><th>状态</th><th>操作</th></tr></thead><tbody><tr v-for="row in modules" :key="row.code"><td class="mono">{{ row.code }}</td><td class="master-primary-cell">{{ row.name }}<small>{{ row.homePageUrl || '—' }}</small></td><td class="mono">{{ row.departmentFunctions ?? '0' }}</td><td>{{ row.needSpecifyDepartment ? '要求指定' : '不限' }}</td><td>{{ row.secondPermissionId || '—' }}</td><td>{{ row.description || '—' }}</td><td><span class="status-pill" :data-status="row.status">{{ row.status }}</span><span v-if="row.hidden" class="master-type-tag">隐藏</span></td><td class="ticket-actions"><button class="text-btn" type="button" :disabled="!canMaintain" @click="editCatalogItem(row)">编辑</button></td></tr></tbody></table>
    </div>
    <div v-else-if="ready && catalogTab === 'categories'" class="result-wrap" tabindex="0" role="region" aria-label="权限分类列表，可横向滚动">
      <div class="master-table-head"><h3>权限分类</h3><span>{{ categories.length }} 条</span></div>
      <table class="register-table"><thead><tr><th>分类编码</th><th>名称</th><th>说明</th><th>状态</th><th>操作</th></tr></thead><tbody><tr v-for="row in categories" :key="row.id"><td class="mono">{{ categoryCode(row) }}</td><td class="master-primary-cell">{{ row.name }}</td><td>{{ row.description || '—' }}</td><td><span class="status-pill" :data-status="row.status">{{ row.status || 'ACTIVE' }}</span></td><td class="ticket-actions"><button class="text-btn" type="button" :disabled="!canMaintain" @click="editCatalogItem(row)">编辑</button></td></tr></tbody></table>
    </div>
    <div v-else-if="ready && catalogTab === 'modes'" class="result-wrap" tabindex="0" role="region" aria-label="认证模式列表，可横向滚动">
      <div class="master-table-head"><h3>认证模式</h3><span>{{ modes.length }} 条</span></div>
      <table class="register-table"><thead><tr><th>模式编码</th><th>名称</th><th>用途</th><th>状态</th><th>操作</th></tr></thead><tbody><tr v-for="row in modes" :key="row.code"><td class="mono">{{ row.code }}</td><td class="master-primary-cell">{{ row.name }}</td><td>{{ row.code === 'card' ? '预留，不开启工卡登录' : '目录配置' }}</td><td><span class="status-pill" :data-status="row.status">{{ row.status || 'ACTIVE' }}</span></td><td class="ticket-actions"><button class="text-btn" type="button" :disabled="!canMaintain" @click="editCatalogItem(row)">编辑</button></td></tr></tbody></table>
    </div>
    <p class="master-list-foot">工卡认证模式和二次权限作为权限元数据保存；支付范围仍为现金，临床规则运行开关保持关闭。</p>
  </section>
</template>
