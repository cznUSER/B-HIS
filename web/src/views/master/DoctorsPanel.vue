<script setup>
// 2026-10-14 P1-H05-WEB-USABILITY 读取状态独立于写入提示；只有成功读取才可维护。
// 2026-10-13 P1-H05-MASTER-UI 医生：归属启用科室，代码唯一。种子 doc-1 / doc-2 可改。
import { computed, onMounted, reactive, ref } from 'vue'
import { ApiError } from '../../api/errors.js'
import { session } from '../../auth/session.js'
import { fetchDepartments, fetchDoctors, saveDoctor } from '../../master/api.js'
import { statusLabel } from '../../master/labels.js'
import MasterPanelHeader from './MasterPanelHeader.vue'

const rows = ref([])
const departments = ref([])
const loading = ref(false)
const listReady = ref(false)
const loadError = ref('')
const loadErrorCode = ref('')
const saving = ref(false)
const editing = ref(false)
const editingId = ref('')
const currentDeptId = ref('')
const q = ref('')
const status = ref('')
const departmentId = ref('')
const notice = ref('')
const noticeError = ref(false)
const errorCode = ref('')
const form = reactive({ doctorCode: '', doctorName: '', departmentId: '', status: 'ACTIVE' })

const deptOptions = computed(() => {
  const active = departments.value.filter((item) => item.status === 'ACTIVE')
  const current = departments.value.find((item) => item.departmentId === currentDeptId.value)
  if (current && current.status !== 'ACTIVE' && !active.some((item) => item.departmentId === current.departmentId)) {
    return [...active, current]
  }
  return active
})

function deptName(id) {
  const found = departments.value.find((item) => item.departmentId === id)
  return found ? found.deptName : id || '—'
}

const canMaintain = computed(() => listReady.value && !loading.value && !saving.value)

function show(message, isError = false, code = '') {
  notice.value = message
  noticeError.value = isError
  errorCode.value = code
}

async function load() {
  if (loading.value) return
  loading.value = true
  listReady.value = false
  loadError.value = ''
  loadErrorCode.value = ''
  try {
    const [deptList, listed] = await Promise.all([
      fetchDepartments(session.accessToken, {}),
      fetchDoctors(session.accessToken, {
        q: q.value,
        status: status.value,
        departmentId: departmentId.value,
      }),
    ])
    departments.value = deptList.items
    rows.value = listed.items
    listReady.value = true
  } catch (error) {
    loadError.value = error instanceof ApiError ? error.message : '无法读取医生或科室'
    loadErrorCode.value = error instanceof ApiError ? error.code : ''
  } finally {
    loading.value = false
  }
}

function startCreate() {
  if (!canMaintain.value) return
  editing.value = true
  editingId.value = ''
  currentDeptId.value = ''
  form.doctorCode = ''
  form.doctorName = ''
  form.departmentId = deptOptions.value[0] ? deptOptions.value[0].departmentId : ''
  form.status = 'ACTIVE'
  show('', false)
}

function startEdit(row) {
  if (!canMaintain.value) return
  editing.value = true
  editingId.value = row.doctorId
  currentDeptId.value = row.departmentId
  form.doctorCode = row.doctorCode
  form.doctorName = row.doctorName
  form.departmentId = row.departmentId
  form.status = row.status
  show('', false)
}

async function onSave() {
  if (!canMaintain.value) return
  saving.value = true
  const id = editingId.value
  try {
    await saveDoctor(session.accessToken, id, {
      doctorCode: form.doctorCode,
      doctorName: form.doctorName,
      departmentId: form.departmentId,
      status: form.status,
    })
    editing.value = false
    show(id ? '已保存医生' : '已新建医生', false)
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
  const next = row.status === 'ACTIVE' ? 'INACTIVE' : 'ACTIVE'
  try {
    await saveDoctor(session.accessToken, row.doctorId, { status: next })
    show(next === 'ACTIVE' ? '已启用医生' : '已停用。新挂号和新开单不能再选该医生。', false)
    await load()
  } catch (error) {
    show(error instanceof ApiError ? error.message : '操作失败', true, error instanceof ApiError ? error.code : '')
  } finally {
    saving.value = false
  }
}

onMounted(load)
</script>

<template>
  <section class="patient-panel master-data-panel" aria-labelledby="doctor-heading">
    <MasterPanelHeader title="医生" title-id="doctor-heading" icon="doctors" description="管理医生资料与科室归属，为门诊业务提供人员信息。" :count="listReady ? rows.length : null" :loading="loading">
      <template #actions>
      <button class="button button-compact" type="button" data-testid="doctor-create" :disabled="!canMaintain" @click="startCreate">新建</button>
      </template>
    </MasterPanelHeader>

    <form class="master-filters" @submit.prevent="!saving && load()">
      <label class="field">
        <span>代码 / 姓名</span>
        <input v-model="q" type="search" placeholder="搜索工号或医生姓名" />
      </label>
      <label class="field">
        <span>科室</span>
        <select v-model="departmentId" data-testid="doctor-filter-dept" :disabled="!listReady">
          <option value="">全部</option>
          <option v-for="item in departments" :key="item.departmentId" :value="item.departmentId">
            {{ item.deptName }}{{ item.status === 'ACTIVE' ? '' : '（已停用）' }}
          </option>
        </select>
      </label>
      <label class="field">
        <span>状态</span>
        <select v-model="status">
          <option value="">全部</option>
          <option value="ACTIVE">启用</option>
          <option value="INACTIVE">停用</option>
        </select>
      </label>
      <button class="button button-ghost button-compact" type="submit" :disabled="loading || saving">查询</button>
    </form>

    <form v-if="editing" class="master-form" data-testid="doctor-form" @submit.prevent="onSave">
      <div class="master-editor-head"><div><p class="master-eyebrow">医生维护</p><h3>{{ editingId ? '编辑医生' : '新建医生' }}</h3></div><span class="master-soft-badge">{{ editingId ? form.doctorCode : '新记录' }}</span></div>
      <fieldset class="master-form-group"><legend>医生信息与归属</legend><div class="field-grid">
        <label class="field">
          <span>工号 / 代码 <abbr class="req" title="必填">*</abbr></span>
          <input v-model="form.doctorCode" required data-testid="doctor-code" autocomplete="off" />
        </label>
        <label class="field">
          <span>姓名 <abbr class="req" title="必填">*</abbr></span>
          <input v-model="form.doctorName" required data-testid="doctor-name" autocomplete="off" />
        </label>
        <label class="field">
          <span>归属科室 <abbr class="req" title="必填">*</abbr></span>
          <select v-model="form.departmentId" required data-testid="doctor-dept">
            <option value="">请选择启用科室</option>
            <option v-for="item in deptOptions" :key="item.departmentId" :value="item.departmentId">
              {{ item.deptName }}{{ item.status === 'ACTIVE' ? '' : '（已停用）' }}
            </option>
          </select>
        </label>
        <label class="field">
          <span>状态</span>
          <select v-model="form.status">
            <option value="ACTIVE">启用</option>
            <option value="INACTIVE">停用</option>
          </select>
        </label>
      </div></fieldset>
      <p class="field-hint">新医生只能挂到启用科室。停用后不能再被新挂号、新开单选中。</p>
      <div class="panel-actions">
        <button class="button button-compact" type="submit" data-testid="doctor-save" :disabled="!canMaintain">
          {{ saving ? '正在保存…' : '保存' }}
        </button>
        <button class="button button-ghost button-compact" type="button" @click="editing = false">取消</button>
      </div>
    </form>

    <p v-if="notice" class="banner" :class="{ 'banner-ok': !noticeError }" :role="noticeError ? 'alert' : 'status'" data-testid="doctor-notice">
      {{ notice }}<span v-if="errorCode">（{{ errorCode }}）</span>
    </p>
    <p v-if="loading" class="shell-status" role="status">正在读取…</p>
    <div v-else-if="loadError" class="master-load-error" role="alert" data-testid="doctor-load-error">
      <p>{{ loadError }}<span v-if="loadErrorCode">（{{ loadErrorCode }}）</span></p>
      <button class="button button-ghost button-compact" type="button" data-testid="doctor-retry" @click="load">重新读取</button>
    </div>
    <p v-else-if="listReady && rows.length === 0" class="result-empty">{{ q || status || departmentId ? '没有符合条件的医生' : '暂无医生；请先维护科室' }}</p>
    <div v-else-if="listReady" class="result-wrap" tabindex="0" role="region" aria-label="医生列表，可横向滚动">
      <div class="master-table-head"><h3>医生列表</h3><span>{{ rows.length }} 条查询结果</span></div>
      <table class="register-table" data-testid="doctor-list">
        <caption class="sr-only">医生</caption>
        <thead>
          <tr>
            <th scope="col">代码</th>
            <th scope="col">姓名</th>
            <th scope="col">科室</th>
            <th scope="col">状态</th>
            <th scope="col">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="row in rows" :key="row.doctorId">
            <td class="mono">{{ row.doctorCode }}</td>
            <td class="master-primary-cell"><span class="master-person-dot" aria-hidden="true">{{ row.doctorName?.slice(0, 1) }}</span>{{ row.doctorName }}</td>
            <td>{{ deptName(row.departmentId) }}</td>
            <td><span class="status-pill" :data-status="row.status">{{ statusLabel(row.status) }}</span></td>
            <td class="ticket-actions">
              <button class="text-btn" type="button" :disabled="!canMaintain" @click="startEdit(row)">编辑</button>
              <button class="text-btn" type="button" :data-testid="`doctor-toggle-${row.doctorId}`" :disabled="!canMaintain" @click="toggle(row)">
                {{ row.status === 'ACTIVE' ? '停用' : '启用' }}
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
    <p class="master-list-foot">新医生需归属启用科室；停用人员不再出现在新业务的选择列表中。</p>
  </section>
</template>
