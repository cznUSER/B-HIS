<script setup>
// 2026-10-14 P1-H05-WEB-USABILITY 读取状态独立于写入提示；只有成功读取才可维护。
// 2026-10-13 P1-H05-MASTER-UI 目录：药品与治疗。新建药品自动建库存 0；治疗不入库存。
// 毒麻标记固定关闭，页面不提供开关。
import { computed, onMounted, reactive, ref } from 'vue'
import { ApiError } from '../../api/errors.js'
import { session } from '../../auth/session.js'
import { fetchCatalog, saveCatalogItem } from '../../master/api.js'
import { formatMoney, statusLabel, typeLabel } from '../../master/labels.js'
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
const itemType = ref('')
const notice = ref('')
const noticeError = ref(false)
const errorCode = ref('')
const form = reactive({
  itemNo: '',
  itemName: '',
  itemType: 'DRUG',
  spec: '',
  units: '',
  price: '',
  status: 'ACTIVE',
})

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
    const listed = await fetchCatalog(session.accessToken, {
      q: q.value,
      status: status.value,
      itemType: itemType.value,
    })
    rows.value = listed.items
    listReady.value = true
  } catch (error) {
    loadError.value = error instanceof ApiError ? error.message : '无法读取目录'
    loadErrorCode.value = error instanceof ApiError ? error.code : ''
  } finally {
    loading.value = false
  }
}

function startCreate() {
  if (!canMaintain.value) return
  editing.value = true
  editingId.value = ''
  form.itemNo = ''
  form.itemName = ''
  form.itemType = 'DRUG'
  form.spec = ''
  form.units = ''
  form.price = ''
  form.status = 'ACTIVE'
  show('', false)
}

function startEdit(row) {
  if (!canMaintain.value) return
  editing.value = true
  editingId.value = row.catalogItemId
  form.itemNo = row.itemNo
  form.itemName = row.itemName
  form.itemType = row.itemType
  form.spec = row.spec || ''
  form.units = row.units
  form.price = row.price
  form.status = row.status
  show('', false)
}

async function onSave() {
  if (!canMaintain.value) return
  saving.value = true
  const id = editingId.value
  const type = form.itemType
  try {
    const saved = await saveCatalogItem(session.accessToken, id, {
      itemNo: form.itemNo,
      itemName: form.itemName,
      itemType: type,
      spec: form.spec,
      units: form.units,
      price: form.price,
      status: form.status,
    })
    editing.value = false
    if (!id && type === 'DRUG' && saved.inventoryCreated) {
      show('已新建药品，并建立库存，当前数量 0。可到药品库存做期初调整。', false)
    } else if (!id && type === 'TREAT') {
      show('已新建治疗项目。治疗项目不入库存。', false)
    } else {
      show(id ? '已保存项目。改价只影响之后开单。' : '已新建项目', false)
    }
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
    await saveCatalogItem(session.accessToken, row.catalogItemId, { status: next })
    show(next === 'ACTIVE' ? '已启用项目' : '已停用。新开单不能再选该项目。', false)
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
  <section class="patient-panel master-data-panel" aria-labelledby="catalog-heading">
    <MasterPanelHeader title="收费 / 药品目录" title-id="catalog-heading" icon="catalog" description="统一维护治疗与药品项目、规格单位及收费价格。" :count="listReady ? rows.length : null" :loading="loading">
      <template #actions>
      <button class="button button-compact" type="button" data-testid="catalog-create" :disabled="!canMaintain" @click="startCreate">新建</button>
      </template>
    </MasterPanelHeader>

    <form class="master-filters" @submit.prevent="!saving && load()">
      <label class="field">
        <span>编码 / 名称</span>
        <input v-model="q" type="search" placeholder="搜索项目编码或名称" />
      </label>
      <label class="field">
        <span>类型</span>
        <select v-model="itemType" data-testid="catalog-type-filter">
          <option value="">全部</option>
          <option value="DRUG">药品</option>
          <option value="TREAT">治疗</option>
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

    <form v-if="editing" class="master-form" data-testid="catalog-form" @submit.prevent="onSave">
      <div class="master-editor-head"><div><p class="master-eyebrow">目录维护</p><h3>{{ editingId ? '编辑项目' : '新建项目' }}</h3></div><span class="master-soft-badge">{{ typeLabel(form.itemType) }}</span></div>
      <fieldset class="master-form-group"><legend>项目信息</legend><div class="field-grid">
        <label class="field">
          <span>编码 <abbr class="req" title="必填">*</abbr></span>
          <input v-model="form.itemNo" required data-testid="catalog-no" autocomplete="off" />
        </label>
        <label class="field">
          <span>名称 <abbr class="req" title="必填">*</abbr></span>
          <input v-model="form.itemName" required data-testid="catalog-name" autocomplete="off" />
        </label>
        <label class="field">
          <span>类型 <abbr class="req" title="必填">*</abbr></span>
          <select v-model="form.itemType" data-testid="catalog-type" :disabled="Boolean(editingId)">
            <option value="DRUG">药品</option>
            <option value="TREAT">治疗</option>
          </select>
        </label>
        <label class="field">
          <span>规格</span>
          <input v-model="form.spec" autocomplete="off" />
        </label>
      </div></fieldset>
      <fieldset class="master-form-group"><legend>价格与状态</legend><div class="field-grid">
        <label class="field">
          <span>单位 <abbr class="req" title="必填">*</abbr></span>
          <input v-model="form.units" required data-testid="catalog-units" autocomplete="off" />
        </label>
        <label class="field">
          <span>单价 <abbr class="req" title="必填">*</abbr></span>
          <input v-model="form.price" type="number" min="0" step="0.01" required data-testid="catalog-price" />
        </label>
        <label class="field">
          <span>状态</span>
          <select v-model="form.status">
            <option value="ACTIVE">启用</option>
            <option value="INACTIVE">停用</option>
          </select>
        </label>
      </div></fieldset>
      <p class="field-hint">毒麻精放标记本期关闭。新建药品会自动建立数量为 0 的库存；治疗项目不入库存。改价只影响之后开单。</p>
      <div class="panel-actions">
        <button class="button button-compact" type="submit" data-testid="catalog-save" :disabled="!canMaintain">
          {{ saving ? '正在保存…' : '保存' }}
        </button>
        <button class="button button-ghost button-compact" type="button" @click="editing = false">取消</button>
      </div>
    </form>

    <p v-if="notice" class="banner" :class="{ 'banner-ok': !noticeError }" :role="noticeError ? 'alert' : 'status'" data-testid="catalog-notice">
      {{ notice }}<span v-if="errorCode">（{{ errorCode }}）</span>
    </p>
    <p v-if="loading" class="shell-status" role="status">正在读取…</p>
    <div v-else-if="loadError" class="master-load-error" role="alert" data-testid="catalog-load-error">
      <p>{{ loadError }}<span v-if="loadErrorCode">（{{ loadErrorCode }}）</span></p>
      <button class="button button-ghost button-compact" type="button" data-testid="catalog-retry" @click="load">重新读取</button>
    </div>
    <p v-else-if="listReady && rows.length === 0" class="result-empty">
      {{ q || status || itemType ? '没有符合条件的项目' : '暂无收费/药品项目' }}
    </p>
    <div v-else-if="listReady" class="result-wrap" tabindex="0" role="region" aria-label="收费和药品目录，可横向滚动">
      <div class="master-table-head"><h3>项目列表</h3><span>{{ rows.length }} 条查询结果</span></div>
      <table class="register-table" data-testid="catalog-list">
        <caption class="sr-only">收费和药品目录</caption>
        <thead>
          <tr>
            <th scope="col">编码</th>
            <th scope="col">名称</th>
            <th scope="col">类型</th>
            <th scope="col">规格</th>
            <th scope="col">单位</th>
            <th scope="col">单价</th>
            <th scope="col">状态</th>
            <th scope="col">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="row in rows" :key="row.catalogItemId">
            <td class="mono">{{ row.itemNo }}</td>
            <td class="master-primary-cell">{{ row.itemName }}</td>
            <td><span class="master-type-tag" :data-type="row.itemType">{{ typeLabel(row.itemType) }}</span></td>
            <td>{{ row.spec || '—' }}</td>
            <td>{{ row.units }}</td>
            <td class="master-money-cell">{{ formatMoney(row.price) }}</td>
            <td><span class="status-pill" :data-status="row.status">{{ statusLabel(row.status) }}</span></td>
            <td class="ticket-actions">
              <button class="text-btn" type="button" :disabled="!canMaintain" @click="startEdit(row)">编辑</button>
              <button class="text-btn" type="button" :data-testid="`catalog-toggle-${row.itemNo}`" :disabled="!canMaintain" @click="toggle(row)">
                {{ row.status === 'ACTIVE' ? '停用' : '启用' }}
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
    <p class="master-list-foot">价格变更只影响之后的开单；新建药品将自动建立数量为 0 的库存。</p>
  </section>
</template>
