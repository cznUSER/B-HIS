<script setup>
// 2026-10-13 P1-H05-MASTER-UI 科室：代码唯一，可启停。种子 dept-1 / dept-2 可改。
import { onMounted, reactive, ref } from 'vue'
import { ApiError } from '../../api/errors.js'
import { session } from '../../auth/session.js'
import { fetchDepartments, saveDepartment } from '../../master/api.js'
import { statusLabel } from '../../master/labels.js'

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
const form = reactive({ deptCode: '', deptName: '', status: 'ACTIVE', sortNo: 0 })

function show(message, isError = false, code = '') {
  notice.value = message
  noticeError.value = isError
  errorCode.value = code
}

async function load() {
  loading.value = true
  try {
    const listed = await fetchDepartments(session.accessToken, { q: q.value, status: status.value })
    rows.value = listed.items
  } catch (error) {
    show(error instanceof ApiError ? error.message : '无法读取科室', true, error instanceof ApiError ? error.code : '')
  } finally {
    loading.value = false
  }
}

function startCreate() {
  editing.value = true
  editingId.value = ''
  form.deptCode = ''
  form.deptName = ''
  form.status = 'ACTIVE'
  form.sortNo = 0
  show('', false)
}

function startEdit(row) {
  editing.value = true
  editingId.value = row.departmentId
  form.deptCode = row.deptCode
  form.deptName = row.deptName
  form.status = row.status
  form.sortNo = row.sortNo
  show('', false)
}

async function onSave() {
  if (saving.value) return
  saving.value = true
  const id = editingId.value
  try {
    await saveDepartment(session.accessToken, id, {
      deptCode: form.deptCode,
      deptName: form.deptName,
      status: form.status,
      sortNo: form.sortNo,
    })
    editing.value = false
    show(id ? '已保存科室' : '已新建科室', false)
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
    await saveDepartment(session.accessToken, row.departmentId, { status: next })
    show(next === 'ACTIVE' ? '已启用科室' : '已停用。新挂号和新开单不能再选该科室。', false)
    await load()
  } catch (error) {
    show(error instanceof ApiError ? error.message : '操作失败', true, error instanceof ApiError ? error.code : '')
  }
}

onMounted(load)
</script>

<template>
  <section class="patient-panel" aria-labelledby="dept-heading">
    <div class="panel-head">
      <h2 id="dept-heading">科室</h2>
      <button class="button button-compact" type="button" data-testid="dept-create" @click="startCreate">新建</button>
    </div>

    <form class="master-filters" @submit.prevent="load">
      <label class="field">
        <span>代码 / 名称</span>
        <input v-model="q" type="search" data-testid="dept-q" />
      </label>
      <label class="field">
        <span>状态</span>
        <select v-model="status" data-testid="dept-status-filter">
          <option value="">全部</option>
          <option value="ACTIVE">启用</option>
          <option value="INACTIVE">停用</option>
        </select>
      </label>
      <button class="button button-ghost button-compact" type="submit">查询</button>
    </form>

    <form v-if="editing" class="master-form" data-testid="dept-form" @submit.prevent="onSave">
      <div class="field-grid">
        <label class="field">
          <span>科室代码 <abbr class="req" title="必填">*</abbr></span>
          <input v-model="form.deptCode" required data-testid="dept-code" autocomplete="off" />
        </label>
        <label class="field">
          <span>科室名称 <abbr class="req" title="必填">*</abbr></span>
          <input v-model="form.deptName" required data-testid="dept-name" autocomplete="off" />
        </label>
        <label class="field">
          <span>状态</span>
          <select v-model="form.status">
            <option value="ACTIVE">启用</option>
            <option value="INACTIVE">停用</option>
          </select>
        </label>
        <label class="field">
          <span>排序</span>
          <input v-model.number="form.sortNo" type="number" step="1" />
        </label>
      </div>
      <p class="field-hint">代码在未删除范围内唯一。停用后再挂号、再开单时不会出现。</p>
      <div class="panel-actions">
        <button class="button button-compact" type="submit" data-testid="dept-save" :disabled="saving">
          {{ saving ? '正在保存…' : '保存' }}
        </button>
        <button class="button button-ghost button-compact" type="button" @click="editing = false">取消</button>
      </div>
    </form>

    <p v-if="notice" class="banner" :class="{ 'banner-ok': !noticeError }" :role="noticeError ? 'alert' : 'status'" data-testid="dept-notice">
      {{ notice }}<span v-if="errorCode">（{{ errorCode }}）</span>
    </p>
    <p v-if="loading" class="shell-status">正在读取…</p>
    <p v-else-if="rows.length === 0" class="result-empty" data-testid="dept-empty">
      {{ q || status ? '没有符合条件的科室' : '暂无科室，请新建' }}
    </p>
    <div v-else class="result-wrap">
      <table class="register-table" data-testid="dept-list">
        <caption class="sr-only">科室</caption>
        <thead>
          <tr>
            <th>代码</th>
            <th>名称</th>
            <th>状态</th>
            <th>排序</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="row in rows" :key="row.departmentId">
            <td class="mono">{{ row.deptCode }}</td>
            <td>{{ row.deptName }}</td>
            <td><span class="status-pill" :data-status="row.status">{{ statusLabel(row.status) }}</span></td>
            <td>{{ row.sortNo }}</td>
            <td class="ticket-actions">
              <button class="text-btn" type="button" @click="startEdit(row)">编辑</button>
              <button class="text-btn" type="button" :data-testid="`dept-toggle-${row.departmentId}`" @click="toggle(row)">
                {{ row.status === 'ACTIVE' ? '停用' : '启用' }}
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </section>
</template>
