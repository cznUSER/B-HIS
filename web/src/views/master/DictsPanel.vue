<script setup>
// 2026-10-13 P1-H05-MASTER-UI 轻量字典：系统枚举只读，号别可维护。
import { onMounted, reactive, ref } from 'vue'
import { ApiError } from '../../api/errors.js'
import { session } from '../../auth/session.js'
import { fetchDictItems, fetchDictOverview, saveDictItem } from '../../master/api.js'
import { enumLabel, statusLabel } from '../../master/labels.js'

const overview = ref(null)
const rows = ref([])
const loading = ref(false)
const saving = ref(false)
const editing = ref(false)
const editingId = ref('')
const notice = ref('')
const noticeError = ref(false)
const errorCode = ref('')
const form = reactive({ code: '', name: '', status: 'ACTIVE', sortNo: 0 })

const dictType = 'register_sort'

function show(message, isError = false, code = '') {
  notice.value = message
  noticeError.value = isError
  errorCode.value = code
}

async function load() {
  loading.value = true
  try {
    overview.value = await fetchDictOverview(session.accessToken)
    const listed = await fetchDictItems(session.accessToken, dictType, {})
    rows.value = listed.items
  } catch (error) {
    show(error instanceof ApiError ? error.message : '无法读取字典', true, error instanceof ApiError ? error.code : '')
  } finally {
    loading.value = false
  }
}

function startCreate() {
  editing.value = true
  editingId.value = ''
  form.code = ''
  form.name = ''
  form.status = 'ACTIVE'
  form.sortNo = 0
  show('', false)
}

function startEdit(row) {
  editing.value = true
  editingId.value = row.dictItemId
  form.code = row.code
  form.name = row.name
  form.status = row.status
  form.sortNo = row.sortNo
  show('', false)
}

async function onSave() {
  if (saving.value) return
  saving.value = true
  const id = editingId.value
  try {
    const body = id
      ? { name: form.name, status: form.status, sortNo: form.sortNo }
      : { code: form.code, name: form.name, status: form.status, sortNo: form.sortNo }
    await saveDictItem(session.accessToken, dictType, id, body)
    editing.value = false
    show(id ? '已保存号别' : '已新建号别', false)
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
    await saveDictItem(session.accessToken, dictType, row.dictItemId, { status: next })
    show(next === 'ACTIVE' ? '已启用' : '已停用', false)
    await load()
  } catch (error) {
    show(error instanceof ApiError ? error.message : '操作失败', true, error instanceof ApiError ? error.code : '')
  }
}

onMounted(load)
</script>

<template>
  <div class="master-stack">
    <section class="patient-panel" aria-labelledby="enum-heading">
      <div class="panel-head">
        <h2 id="enum-heading">系统枚举</h2>
      </div>
      <p class="field-hint">这些枚举与门诊契约一致，只读。修改走发版，不在这里配表。</p>
      <p v-if="overview && overview.readonlyEnums.length === 0" class="result-empty">当前仅系统枚举（只读）</p>
      <div v-else-if="overview" class="result-wrap">
        <table class="register-table" data-testid="enum-list">
          <caption class="sr-only">只读系统枚举</caption>
          <thead>
            <tr>
              <th>类别</th>
              <th>代码</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="group in overview.readonlyEnums" :key="group.dictType">
              <td>{{ group.name }}</td>
              <td>
                <span v-for="code in group.values" :key="code" class="enum-chip">{{ code }} {{ enumLabel(code) }}</span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>

    <section class="patient-panel" aria-labelledby="sort-heading">
      <div class="panel-head">
        <h2 id="sort-heading">号别</h2>
        <button class="button button-compact" type="button" data-testid="dict-create" @click="startCreate">新建</button>
      </div>
      <p class="field-hint">可维护码表只有号别。挂号页仍默认普通号，这里改名不影响已挂出的号。</p>

      <form v-if="editing" class="master-form" data-testid="dict-form" @submit.prevent="onSave">
        <div class="field-grid">
          <label class="field">
            <span>代码 <abbr class="req" title="必填">*</abbr></span>
            <input v-model="form.code" :disabled="Boolean(editingId)" required data-testid="dict-code" autocomplete="off" />
          </label>
          <label class="field">
            <span>名称 <abbr class="req" title="必填">*</abbr></span>
            <input v-model="form.name" required data-testid="dict-name" autocomplete="off" />
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
        <div class="panel-actions">
          <button class="button button-compact" type="submit" data-testid="dict-save" :disabled="saving">
            {{ saving ? '正在保存…' : '保存' }}
          </button>
          <button class="button button-ghost button-compact" type="button" @click="editing = false">取消</button>
        </div>
      </form>

      <p v-if="notice" class="banner" :class="{ 'banner-ok': !noticeError }" :role="noticeError ? 'alert' : 'status'" data-testid="dict-notice">
        {{ notice }}<span v-if="errorCode">（{{ errorCode }}）</span>
      </p>
      <p v-if="loading" class="shell-status">正在读取…</p>
      <p v-else-if="rows.length === 0" class="result-empty">无可维护码表（仅系统枚举）</p>
      <div v-else class="result-wrap">
        <table class="register-table" data-testid="dict-list">
          <caption class="sr-only">号别</caption>
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
            <tr v-for="row in rows" :key="row.dictItemId">
              <td class="mono">{{ row.code }}</td>
              <td>{{ row.name }}</td>
              <td><span class="status-pill" :data-status="row.status">{{ statusLabel(row.status) }}</span></td>
              <td>{{ row.sortNo }}</td>
              <td class="ticket-actions">
                <button class="text-btn" type="button" @click="startEdit(row)">编辑</button>
                <button class="text-btn" type="button" @click="toggle(row)">{{ row.status === 'ACTIVE' ? '停用' : '启用' }}</button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>
  </div>
</template>
