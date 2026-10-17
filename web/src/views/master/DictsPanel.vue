<script setup>
// 2026-10-14 P1-H05-WEB-USABILITY 读取状态独立于写入提示；只有成功读取才可维护。
// 2026-10-13 P1-H05-MASTER-UI 轻量字典：系统枚举只读，号别可维护。
import { computed, onMounted, reactive, ref } from 'vue'
import { ApiError } from '../../api/errors.js'
import { session } from '../../auth/session.js'
import { fetchDictItems, fetchDictOverview, saveDictItem } from '../../master/api.js'
import { enumLabel, statusLabel } from '../../master/labels.js'
import MasterPanelHeader from './MasterPanelHeader.vue'

const overview = ref(null)
const rows = ref([])
const loading = ref(false)
const listReady = ref(false)
const loadError = ref('')
const loadErrorCode = ref('')
const saving = ref(false)
const editing = ref(false)
const editingId = ref('')
const notice = ref('')
const noticeError = ref(false)
const errorCode = ref('')
const form = reactive({ code: '', name: '', status: 'ACTIVE', sortNo: 0 })

const dictType = 'register_sort'
const DICT_TITLES = {
  sex: '性别',
  cardStatus: '卡状态',
  payMethod: '支付方式',
  registerStatus: '挂号状态',
  orderStatus: '处方状态',
  id_type: '证件类型',
  pay_type: '支付方式',
  fare_identity: '费别',
  register_sort: '号别',
}

function dictTitle(group) {
  return group.name || DICT_TITLES[group.dictType] || group.dictType
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
    const groups = await fetchDictOverview(session.accessToken)
    const listed = await fetchDictItems(session.accessToken, dictType, {})
    overview.value = groups
    rows.value = listed.items
    listReady.value = true
  } catch (error) {
    loadError.value = error instanceof ApiError ? error.message : '无法读取字典'
    loadErrorCode.value = error instanceof ApiError ? error.code : ''
  } finally {
    loading.value = false
  }
}

function startCreate() {
  if (!canMaintain.value) return
  editing.value = true
  editingId.value = ''
  form.code = ''
  form.name = ''
  form.status = 'ACTIVE'
  form.sortNo = 0
  show('', false)
}

function startEdit(row) {
  if (!canMaintain.value) return
  editing.value = true
  editingId.value = row.dictItemId
  form.code = row.code
  form.name = row.name
  form.status = row.status
  form.sortNo = row.sortNo
  show('', false)
}

async function onSave() {
  if (!canMaintain.value) return
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
  if (!canMaintain.value) return
  saving.value = true
  const next = row.status === 'ACTIVE' ? 'INACTIVE' : 'ACTIVE'
  try {
    await saveDictItem(session.accessToken, dictType, row.dictItemId, { status: next })
    show(next === 'ACTIVE' ? '已启用' : '已停用', false)
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
  <div class="master-stack master-dictionary-layout">
    <section class="patient-panel master-data-panel master-enums-panel" aria-labelledby="enum-heading">
      <MasterPanelHeader title="系统枚举" title-id="enum-heading" icon="dicts" description="查看业务契约约定的基础枚举。" badge="只读" />
      <p class="field-hint master-inline-note">这些枚举与门诊契约一致，只读。修改走发版，不在这里配表。</p>
      <p v-if="loading" class="shell-status" role="status">正在读取系统枚举…</p>
      <p v-else-if="loadError" class="field-hint">字典读取未完成，请使用下方“重新读取”重试。</p>
      <p v-else-if="listReady && overview.readonlyEnums.length === 0" class="result-empty">暂无系统枚举</p>
      <div v-else-if="listReady && overview" class="result-wrap" tabindex="0" role="region" aria-label="只读系统枚举，可横向滚动">
        <table class="register-table" data-testid="enum-list">
          <caption class="sr-only">只读系统枚举</caption>
          <thead>
            <tr>
              <th scope="col">类别</th>
              <th scope="col">代码</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="group in overview.readonlyEnums" :key="group.dictType">
              <td class="master-primary-cell">{{ dictTitle(group) }}</td>
              <td>
                <span v-for="code in group.values" :key="code" class="enum-chip">{{ code }} {{ enumLabel(code) }}</span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>

    <section class="patient-panel master-data-panel" aria-labelledby="sort-heading">
      <MasterPanelHeader title="号别" title-id="sort-heading" icon="catalog" description="维护挂号号别的名称、顺序及启停状态。" :count="listReady ? rows.length : null" :loading="loading">
        <template #actions>
        <button class="button button-compact" type="button" data-testid="dict-create" :disabled="!canMaintain" @click="startCreate">新建</button>
        </template>
      </MasterPanelHeader>
      <p class="field-hint master-inline-note">可维护码表只有号别。挂号页仍默认普通号，这里改名不影响已挂出的号。</p>

      <form v-if="editing" class="master-form" data-testid="dict-form" @submit.prevent="onSave">
        <div class="master-editor-head"><div><p class="master-eyebrow">字典维护</p><h3>{{ editingId ? '编辑号别' : '新建号别' }}</h3></div><span class="master-soft-badge">{{ editingId ? form.code : '新记录' }}</span></div>
        <fieldset class="master-form-group"><legend>号别信息与状态</legend><div class="field-grid">
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
        </div></fieldset>
        <div class="panel-actions">
          <button class="button button-compact" type="submit" data-testid="dict-save" :disabled="!canMaintain">
            {{ saving ? '正在保存…' : '保存' }}
          </button>
          <button class="button button-ghost button-compact" type="button" @click="editing = false">取消</button>
        </div>
      </form>

      <p v-if="notice" class="banner" :class="{ 'banner-ok': !noticeError }" :role="noticeError ? 'alert' : 'status'" data-testid="dict-notice">
        {{ notice }}<span v-if="errorCode">（{{ errorCode }}）</span>
      </p>
      <p v-if="loading" class="shell-status" role="status">正在读取…</p>
    <div v-else-if="loadError" class="master-load-error" role="alert" data-testid="dict-load-error">
      <p>{{ loadError }}<span v-if="loadErrorCode">（{{ loadErrorCode }}）</span></p>
      <button class="button button-ghost button-compact" type="button" data-testid="dict-retry" @click="load">重新读取</button>
    </div>
      <p v-else-if="listReady && rows.length === 0" class="result-empty" data-testid="dict-empty">暂无号别，请新建</p>
      <div v-else-if="listReady" class="result-wrap" tabindex="0" role="region" aria-label="号别列表，可横向滚动">
        <div class="master-table-head"><h3>号别列表</h3><span>{{ rows.length }} 条记录</span></div>
        <table class="register-table" data-testid="dict-list">
          <caption class="sr-only">号别</caption>
          <thead>
            <tr>
              <th scope="col">代码</th>
              <th scope="col">名称</th>
              <th scope="col">状态</th>
              <th scope="col">排序</th>
              <th scope="col">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="row in rows" :key="row.dictItemId">
              <td class="mono">{{ row.code }}</td>
              <td class="master-primary-cell">{{ row.name }}</td>
              <td><span class="status-pill" :data-status="row.status">{{ statusLabel(row.status) }}</span></td>
              <td>{{ row.sortNo }}</td>
              <td class="ticket-actions">
                <button class="text-btn" type="button" :disabled="!canMaintain" @click="startEdit(row)">编辑</button>
                <button class="text-btn" type="button" :disabled="!canMaintain" @click="toggle(row)">{{ row.status === 'ACTIVE' ? '停用' : '启用' }}</button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      <p class="master-list-foot">名称修改不追溯已产生的挂号记录。</p>
    </section>
  </div>
</template>
