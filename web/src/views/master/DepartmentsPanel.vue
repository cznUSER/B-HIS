<script setup>
// 2026-10-14 P1-H05-WEB-USABILITY 读取状态独立于写入提示；只有成功读取才可维护。
// 2026-10-13 P1-H05-MASTER-UI 科室：代码唯一，可启停。种子 dept-1 / dept-2 可改。
import { computed, onMounted, reactive, ref } from 'vue'
import { ApiError } from '../../api/errors.js'
import { session } from '../../auth/session.js'
import { fetchDepartments, saveDepartment } from '../../master/api.js'
import { statusLabel } from '../../master/labels.js'
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
const form = reactive({ deptCode: '', deptName: '', status: 'ACTIVE', sortNo: 0, phone: '', manager: '', inputCode: '' }) // UI90-04D-FIX-03: phone/manager/inputCode UI-only; never in saveDepartment
const activeTab = ref('normal') // 'normal' | 'nursing' · UI90-04D-FIX-01 tab shell only
// UI90-04D-FIX-02: nursing ListBox draft is UI-only / mock; never persists; never expands saveDepartment.
const nursingMembers = ref([])
const nursingSelectedIndex = ref(-1)

function resetNursingDraft() {
  nursingMembers.value = []
  nursingSelectedIndex.value = -1
}

function addNursingMember() {
  const n = nursingMembers.value.length + 1
  nursingMembers.value.push({
    id: `ui-only-${Date.now()}-${n}`,
    label: `UI 仅演示行 ${n}（未持久化 / mock）`,
  })
  nursingSelectedIndex.value = nursingMembers.value.length - 1
}

function delNursingMember() {
  const idx = nursingSelectedIndex.value
  if (idx < 0 || idx >= nursingMembers.value.length) return
  nursingMembers.value.splice(idx, 1)
  if (!nursingMembers.value.length) {
    nursingSelectedIndex.value = -1
  } else if (idx >= nursingMembers.value.length) {
    nursingSelectedIndex.value = nursingMembers.value.length - 1
  }
}

function cancelEdit() {
  editing.value = false
  resetNursingDraft()
  // UI90-04D-FIX-03: discard local UI-only drafts
  form.phone = ''
  form.manager = ''
  form.inputCode = ''
}

const canMaintain = computed(() => listReady.value && !loading.value && !saving.value)

function show(message, isError = false, code = '') {
  noticeError.value = false
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
    const listed = await fetchDepartments(session.accessToken, { q: q.value, status: status.value })
    rows.value = listed.items
    listReady.value = true
  } catch (error) {
    loadError.value = error instanceof ApiError ? error.message : '无法读取科室'
    loadErrorCode.value = error instanceof ApiError ? error.code : ''
  } finally {
    loading.value = false
  }
}

function startCreate() {
  if (!canMaintain.value) return
  editing.value = true
  activeTab.value = 'normal'
  resetNursingDraft()
  editingId.value = ''
  form.deptCode = ''
  form.deptName = ''
  form.status = 'ACTIVE'
  form.sortNo = 0
  form.phone = ''
  form.manager = ''
  form.inputCode = ''
  show('', false)
}

function startEdit(row) {
  if (!canMaintain.value) return
  editing.value = true
  activeTab.value = 'normal'
  resetNursingDraft()
  editingId.value = row.departmentId
  form.deptCode = row.deptCode
  form.deptName = row.deptName
  form.status = row.status
  form.sortNo = row.sortNo
  // UI90-04D-FIX-03: DTO has no phone/manager/inputCode — leave empty; do not claim server refill
  form.phone = ''
  form.manager = ''
  form.inputCode = ''
  show('', false)
}

async function onSave() {
  if (!canMaintain.value) return
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
    resetNursingDraft()
    show(id ? '已保存科室' : '已新建科室', false)
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
    await saveDepartment(session.accessToken, row.departmentId, { status: next })
    show(next === 'ACTIVE' ? '已启用科室' : '已停用。新挂号和新开单不能再选该科室。', false)
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
  <section class="patient-panel master-data-panel ui2-department-panel" aria-labelledby="dept-heading">
    <MasterPanelHeader title="科室" title-id="dept-heading" icon="departments" description="维护诊疗科室、显示顺序与启停状态。" :count="listReady ? rows.length : null" :loading="loading">
      <template #actions>
      <button class="button button-compact" type="button" data-testid="dept-create" :disabled="!canMaintain" @click="startCreate">新建</button>
      </template>
    </MasterPanelHeader>

    <form class="master-filters" @submit.prevent="!saving && load()">
      <label class="field">
        <span>代码 / 名称</span>
        <input v-model="q" type="search" placeholder="搜索科室代码或名称" data-testid="dept-q" />
      </label>
      <label class="field">
        <span>状态</span>
        <select v-model="status" data-testid="dept-status-filter">
          <option value="">全部</option>
          <option value="ACTIVE">启用</option>
          <option value="INACTIVE">停用</option>
        </select>
      </label>
      <button class="button button-ghost button-compact" type="submit" :disabled="loading || saving">查询</button>
    </form>

    <form v-if="editing" class="master-form" data-testid="dept-form" @submit.prevent="onSave">
      <div class="master-editor-head"><div><p class="master-eyebrow">科室维护</p><h3>{{ editingId ? '编辑科室' : '新建科室' }}</h3></div><span class="master-soft-badge">{{ editingId ? form.deptCode : '新记录' }}</span></div>
      <div class="dept-editor-tabs" role="tablist" aria-label="科室编辑页签" data-testid="dept-editor-tabs">
        <button
          class="dept-editor-tab"
          type="button"
          role="tab"
          data-testid="dept-tab-normal"
          :aria-selected="activeTab === 'normal'"
          :tabindex="activeTab === 'normal' ? 0 : -1"
          :class="{ 'is-active': activeTab === 'normal' }"
          @click="activeTab = 'normal'"
        >常规</button>
        <button
          class="dept-editor-tab"
          type="button"
          role="tab"
          data-testid="dept-tab-nursing"
          :aria-selected="activeTab === 'nursing'"
          :tabindex="activeTab === 'nursing' ? 0 : -1"
          :class="{ 'is-active': activeTab === 'nursing' }"
          @click="activeTab = 'nursing'"
        >护理单元包含科室</button>
      </div>
      <div
        v-show="activeTab === 'normal'"
        class="dept-editor-tabpanel"
        role="tabpanel"
        data-testid="dept-tabpanel-normal"
        aria-label="常规"
      >
        <fieldset class="master-form-group"><legend>科室信息与状态</legend><div class="field-grid">
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
          <label class="field">
            <span>电话号码</span>
            <input v-model="form.phone" type="text" data-testid="dept-phone" autocomplete="off" />
          </label>
          <label class="field">
            <span>负责人</span>
            <input v-model="form.manager" type="text" data-testid="dept-manager" autocomplete="off" />
          </label>
          <label class="field">
            <span>助记码</span>
            <input v-model="form.inputCode" type="text" data-testid="dept-input-code" autocomplete="off" />
          </label>
        </div></fieldset>
        <p class="field-hint">代码在未删除范围内唯一。停用后再挂号、再开单时不会出现。</p>
        <p class="field-hint" data-testid="dept-extra-fields-hint">本片新增字段（电话号码 / 负责人 / 助记码）为 UI 仅 / 未持久化；不写入 saveDepartment。</p>
      </div>
      <div
        v-show="activeTab === 'nursing'"
        class="dept-editor-tabpanel dept-editor-tabpanel-nursing"
        role="tabpanel"
        data-testid="dept-tabpanel-nursing"
        aria-label="护理单元包含科室"
      >
        <p class="dept-nursing-note" data-testid="dept-nursing-note">护理单元包含科室列表为 UI 仅演示（未持久化 / mock）；取消或关闭编辑丢弃本地草稿；不写入 saveDepartment。</p>
        <ul
          class="dept-nursing-list"
          role="listbox"
          data-testid="dept-nursing-list"
          aria-label="护理单元包含科室（UI 仅 / 未持久化）"
          tabindex="0"
        >
          <li
            v-if="!nursingMembers.length"
            class="dept-nursing-empty"
            data-testid="dept-nursing-empty"
            role="option"
            aria-disabled="true"
          >（空表 · 未持久化）</li>
          <li
            v-for="(item, idx) in nursingMembers"
            :key="item.id"
            class="dept-nursing-item"
            role="option"
            :aria-selected="nursingSelectedIndex === idx"
            :class="{ 'is-selected': nursingSelectedIndex === idx }"
            :data-testid="`dept-nursing-item-${idx}`"
            @click="nursingSelectedIndex = idx"
          >{{ item.label }}</li>
        </ul>
        <div class="dept-nursing-actions" data-testid="dept-nursing-actions">
          <button
            class="button button-compact dept-nursing-btn"
            type="button"
            data-testid="dept-nursing-add"
            @click="addNursingMember"
          >增加</button>
          <button
            class="button button-ghost button-compact dept-nursing-btn"
            type="button"
            data-testid="dept-nursing-del"
            :disabled="nursingSelectedIndex < 0"
            @click="delNursingMember"
          >删除</button>
        </div>
      </div>
      <div class="panel-actions">
        <button class="button button-compact" type="submit" data-testid="dept-save" :disabled="!canMaintain">
          {{ saving ? '正在保存…' : '保存' }}
        </button>
        <button class="button button-ghost button-compact" type="button" data-testid="dept-cancel" @click="cancelEdit">取消</button>
      </div>
    </form>

    <p v-if="notice" class="banner" :class="{ 'banner-ok': !noticeError }" :role="noticeError ? 'alert' : 'status'" data-testid="dept-notice">
      {{ notice }}<span v-if="errorCode">（{{ errorCode }}）</span>
    </p>
    <p v-if="loading" class="shell-status" role="status">正在读取…</p>
    <div v-else-if="loadError" class="master-load-error" role="alert" data-testid="dept-load-error">
      <p>{{ loadError }}<span v-if="loadErrorCode">（{{ loadErrorCode }}）</span></p>
      <button class="button button-ghost button-compact" type="button" data-testid="dept-retry" @click="load">重新读取</button>
    </div>
    <p v-else-if="listReady && rows.length === 0" class="result-empty" data-testid="dept-empty">
      {{ q || status ? '没有符合条件的科室' : '暂无科室，请新建' }}
    </p>
    <div v-else-if="listReady" class="result-wrap" tabindex="0" role="region" aria-label="科室列表，可横向滚动">
      <div class="master-table-head"><h3>科室列表</h3><span>{{ rows.length }} 条查询结果</span></div>
      <table class="register-table" data-testid="dept-list">
        <caption class="sr-only">科室</caption>
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
          <tr v-for="row in rows" :key="row.departmentId">
            <td class="mono">{{ row.deptCode }}</td>
            <td class="master-primary-cell">{{ row.deptName }}</td>
            <td><span class="status-pill" :data-status="row.status">{{ statusLabel(row.status) }}</span></td>
            <td>{{ row.sortNo }}</td>
            <td class="ticket-actions">
              <button class="text-btn" type="button" :disabled="!canMaintain" @click="startEdit(row)">编辑</button>
              <button class="text-btn" type="button" :data-testid="`dept-toggle-${row.departmentId}`" :disabled="!canMaintain" @click="toggle(row)">
                {{ row.status === 'ACTIVE' ? '停用' : '启用' }}
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
    <p class="master-list-foot">仅启用科室可在新的挂号与开单中使用。</p>
  </section>
</template>

<style scoped>
/* UI2.0/bsdata/DepartmentFrm: 408px editor, Songti 16px, 90x30 actions. */
.ui2-department-panel.patient-panel.master-data-panel { color: #202020; font-family: SimSun, '宋体', serif; }
.ui2-department-panel.patient-panel.master-data-panel .master-form { width: 408px; max-width: 100%; margin: 12px 0; padding: 12px; border: 1px solid #999; border-radius: 0; background: #f0f0f0; box-sizing: border-box; }
.ui2-department-panel.patient-panel.master-data-panel .master-editor-head { min-height: 30px; margin: -12px -12px 12px; padding: 6px 12px; background: #e4e4e4; border-bottom: 1px solid #999; }
.ui2-department-panel.patient-panel.master-data-panel .master-editor-head h3 { margin: 0; font: 16px SimSun, '宋体', serif; }
.ui2-department-panel.patient-panel.master-data-panel .master-eyebrow { display: none; }
.ui2-department-panel.patient-panel.master-data-panel .master-form-group { padding: 8px 0; border: 0; border-radius: 0; background: transparent; }
.ui2-department-panel.patient-panel.master-data-panel .master-form-group legend { font: 12px 'Segoe UI', sans-serif; }
.ui2-department-panel.patient-panel.master-data-panel .master-form .field-grid { display: grid; grid-template-columns: 1fr; gap: 7px; }
.ui2-department-panel.patient-panel.master-data-panel .master-form .field { display: grid; grid-template-columns: 76px minmax(0, 1fr); align-items: center; gap: 6px; }
.ui2-department-panel.patient-panel.master-data-panel .master-form .field > span { display: block; padding: 3px 5px; border: 1px solid #bbb; border-radius: 5px; background: #dedede; color: #202020; font: 12px/17px 'Segoe UI', sans-serif; }
.ui2-department-panel.patient-panel.master-data-panel .master-form .field input, .ui2-department-panel.patient-panel.master-data-panel .master-form .field select { width: 100%; min-width: 0; height: 25px; min-height: 25px; padding: 2px 5px; border: 1px solid #999; border-radius: 0; background: #fff; color: #202020; font: 16px SimSun, '宋体', serif; box-sizing: border-box; }
.ui2-department-panel.patient-panel.master-data-panel .master-form .panel-actions { display: flex; justify-content: flex-end; gap: 8px; margin-top: 20px; }
.ui2-department-panel.patient-panel.master-data-panel .master-form .panel-actions .button { width: 90px; height: 30px; min-height: 30px; padding: 0 5px; border: 1px solid #aaa; border-radius: 3px; background: #e6e6e6; color: #202020; font: 16px SimSun, '宋体', serif; }
.ui2-department-panel.patient-panel.master-data-panel .master-form .field-hint { margin-top: 10px; font: 12px/1.5 'Segoe UI', sans-serif; }
/* UI90-04D-FIX-01: DepartmentFrm pcMain tabs — 常规 / 护理单元包含科室; TabHeight≈30; editor w≈408; actions 90×30 conserved. */
.ui2-department-panel.patient-panel.master-data-panel .master-form {
  --dept-editor-w: 408px;
  --dept-tab-h: 30px;
  --dept-pc-h: 366px;
  width: var(--dept-editor-w);
}
.ui2-department-panel.patient-panel.master-data-panel .dept-editor-tabs {
  display: flex;
  align-items: stretch;
  gap: 0;
  height: var(--dept-tab-h);
  min-height: var(--dept-tab-h);
  margin: 0 0 8px;
  border-bottom: 1px solid #999;
  background: #e8e8e8;
}
.ui2-department-panel.patient-panel.master-data-panel .dept-editor-tab {
  height: var(--dept-tab-h);
  min-height: var(--dept-tab-h);
  padding: 0 12px;
  border: 1px solid transparent;
  border-bottom: 0;
  border-radius: 0;
  background: transparent;
  color: #202020;
  font: 12px/30px 'Segoe UI', SimSun, '宋体', sans-serif;
  cursor: pointer;
}
.ui2-department-panel.patient-panel.master-data-panel .dept-editor-tab.is-active {
  background: #f0f0f0;
  border-color: #999;
  border-bottom-color: #f0f0f0;
  margin-bottom: -1px;
  font-weight: 600;
}
.ui2-department-panel.patient-panel.master-data-panel .dept-editor-tabpanel {
  min-height: 120px;
  box-sizing: border-box;
}
.ui2-department-panel.patient-panel.master-data-panel .dept-editor-tabpanel-nursing {
  /* UI90-04D-FIX-02: DFM lbSubList 404×257 + btnAdd/btnDel 90×30; UI-only draft. */
  display: flex;
  flex-direction: column;
  align-items: stretch;
  justify-content: flex-start;
  gap: 8px;
  min-height: 160px;
  border: 0;
  background: transparent;
  --dept-nursing-list-h: 257px;
  --dept-nursing-btn-w: 90px;
  --dept-nursing-btn-h: 30px;
}
.ui2-department-panel.patient-panel.master-data-panel .dept-nursing-note {
  margin: 0;
  padding: 0 2px;
  color: #666;
  font: 12px/1.45 'Segoe UI', sans-serif;
}
.ui2-department-panel.patient-panel.master-data-panel .dept-nursing-list {
  list-style: none;
  margin: 0;
  padding: 0;
  height: var(--dept-nursing-list-h);
  min-height: 120px;
  max-height: var(--dept-nursing-list-h);
  overflow: auto;
  border: 1px solid #999;
  background: #fff;
  box-sizing: border-box;
}
.ui2-department-panel.patient-panel.master-data-panel .dept-nursing-empty,
.ui2-department-panel.patient-panel.master-data-panel .dept-nursing-item {
  margin: 0;
  padding: 2px 6px;
  min-height: 15px;
  color: #202020;
  font: 16px/15px SimSun, '宋体', serif;
  cursor: default;
  user-select: none;
}
.ui2-department-panel.patient-panel.master-data-panel .dept-nursing-empty {
  color: #888;
  font: 12px/1.5 'Segoe UI', sans-serif;
  padding: 8px 6px;
}
.ui2-department-panel.patient-panel.master-data-panel .dept-nursing-item.is-selected {
  background: #cce4f7;
}
.ui2-department-panel.patient-panel.master-data-panel .dept-nursing-actions {
  display: flex;
  gap: 7px;
  align-items: center;
}
.ui2-department-panel.patient-panel.master-data-panel .dept-nursing-btn {
  width: var(--dept-nursing-btn-w);
  height: var(--dept-nursing-btn-h);
  min-height: var(--dept-nursing-btn-h);
  padding: 0 5px;
  border: 1px solid #aaa;
  border-radius: 3px;
  background: #e6e6e6;
  color: #202020;
  font: 16px SimSun, '宋体', serif;
}
</style>
