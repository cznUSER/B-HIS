<script setup>
// 2026-10-13 P1-H05-MASTER-UI 医生：归属启用科室，代码唯一。种子 doc-1 / doc-2 可改。
import { computed, onMounted, reactive, ref } from 'vue'
import { ApiError } from '../../api/errors.js'
import { session } from '../../auth/session.js'
import { fetchDepartments, fetchDoctors, saveDoctor } from '../../master/api.js'
import { statusLabel } from '../../master/labels.js'

const rows = ref([])
const departments = ref([])
const loading = ref(false)
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

function show(message, isError = false, code = '') {
  notice.value = message
  noticeError.value = isError
  errorCode.value = code
}

async function loadDepartments() {
  const listed = await fetchDepartments(session.accessToken, {})
  departments.value = listed.items
}

async function load() {
  loading.value = true
  try {
    const listed = await fetchDoctors(session.accessToken, {
      q: q.value,
      status: status.value,
      departmentId: departmentId.value,
    })
    rows.value = listed.items
  } catch (error) {
    show(error instanceof ApiError ? error.message : '无法读取医生', true, error instanceof ApiError ? error.code : '')
  } finally {
    loading.value = false
  }
}

function startCreate() {
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
  if (saving.value) return
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
  const next = row.status === 'ACTIVE' ? 'INACTIVE' : 'ACTIVE'
  try {
    await saveDoctor(session.accessToken, row.doctorId, { status: next })
    show(next === 'ACTIVE' ? '已启用医生' : '已停用。新挂号和新开单不能再选该医生。', false)
    await load()
  } catch (error) {
    show(error instanceof ApiError ? error.message : '操作失败', true, error instanceof ApiError ? error.code : '')
  }
}

onMounted(async () => {
  try {
    await loadDepartments()
  } catch (error) {
    show(error instanceof ApiError ? error.message : '无法读取科室', true, error instanceof ApiError ? error.code : '')
  }
  await load()
})
</script>

<template>
  <section class="patient-panel" aria-labelledby="doctor-heading">
    <div class="panel-head">
      <h2 id="doctor-heading">医生</h2>
      <button class="button button-compact" type="button" data-testid="doctor-create" @click="startCreate">新建</button>
    </div>

    <form class="master-filters" @submit.prevent="load">
      <label class="field">
        <span>代码 / 姓名</span>
        <input v-model="q" type="search" />
      </label>
      <label class="field">
        <span>科室</span>
        <select v-model="departmentId" data-testid="doctor-filter-dept">
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
      <button class="button button-ghost button-compact" type="submit">查询</button>
    </form>

    <form v-if="editing" class="master-form" data-testid="doctor-form" @submit.prevent="onSave">
      <div class="field-grid">
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
      </div>
      <p class="field-hint">新医生只能挂到启用科室。停用后不能再被新挂号、新开单选中。</p>
      <div class="panel-actions">
        <button class="button button-compact" type="submit" data-testid="doctor-save" :disabled="saving">
          {{ saving ? '正在保存…' : '保存' }}
        </button>
        <button class="button button-ghost button-compact" type="button" @click="editing = false">取消</button>
      </div>
    </form>

    <p v-if="notice" class="banner" :class="{ 'banner-ok': !noticeError }" :role="noticeError ? 'alert' : 'status'" data-testid="doctor-notice">
      {{ notice }}<span v-if="errorCode">（{{ errorCode }}）</span>
    </p>
    <p v-if="departments.length === 0" class="result-empty">暂无医生；请先维护科室</p>
    <p v-else-if="loading" class="shell-status">正在读取…</p>
    <p v-else-if="rows.length === 0" class="result-empty">{{ q || status || departmentId ? '没有符合条件的医生' : '暂无医生；请先维护科室' }}</p>
    <div v-else class="result-wrap">
      <table class="register-table" data-testid="doctor-list">
        <caption class="sr-only">医生</caption>
        <thead>
          <tr>
            <th>代码</th>
            <th>姓名</th>
            <th>科室</th>
            <th>状态</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="row in rows" :key="row.doctorId">
            <td class="mono">{{ row.doctorCode }}</td>
            <td>{{ row.doctorName }}</td>
            <td>{{ deptName(row.departmentId) }}</td>
            <td><span class="status-pill" :data-status="row.status">{{ statusLabel(row.status) }}</span></td>
            <td class="ticket-actions">
              <button class="text-btn" type="button" @click="startEdit(row)">编辑</button>
              <button class="text-btn" type="button" :data-testid="`doctor-toggle-${row.doctorId}`" @click="toggle(row)">
                {{ row.status === 'ACTIVE' ? '停用' : '启用' }}
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </section>
</template>
