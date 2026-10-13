<script setup>
// 2026-10-13 P1-H05-MASTER-UI 目录：药品与治疗。新建药品自动建库存 0；治疗不入库存。
// 毒麻标记固定关闭，页面不提供开关。
import { onMounted, reactive, ref } from 'vue'
import { ApiError } from '../../api/errors.js'
import { session } from '../../auth/session.js'
import { fetchCatalog, saveCatalogItem } from '../../master/api.js'
import { formatMoney, statusLabel, typeLabel } from '../../master/labels.js'

const rows = ref([])
const loading = ref(false)
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

function show(message, isError = false, code = '') {
  notice.value = message
  noticeError.value = isError
  errorCode.value = code
}

async function load() {
  loading.value = true
  try {
    const listed = await fetchCatalog(session.accessToken, {
      q: q.value,
      status: status.value,
      itemType: itemType.value,
    })
    rows.value = listed.items
  } catch (error) {
    show(error instanceof ApiError ? error.message : '无法读取目录', true, error instanceof ApiError ? error.code : '')
  } finally {
    loading.value = false
  }
}

function startCreate() {
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
  if (saving.value) return
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
  const next = row.status === 'ACTIVE' ? 'INACTIVE' : 'ACTIVE'
  try {
    await saveCatalogItem(session.accessToken, row.catalogItemId, { status: next })
    show(next === 'ACTIVE' ? '已启用项目' : '已停用。新开单不能再选该项目。', false)
    await load()
  } catch (error) {
    show(error instanceof ApiError ? error.message : '操作失败', true, error instanceof ApiError ? error.code : '')
  }
}

onMounted(load)
</script>

<template>
  <section class="patient-panel" aria-labelledby="catalog-heading">
    <div class="panel-head">
      <h2 id="catalog-heading">收费 / 药品目录</h2>
      <button class="button button-compact" type="button" data-testid="catalog-create" @click="startCreate">新建</button>
    </div>

    <form class="master-filters" @submit.prevent="load">
      <label class="field">
        <span>编码 / 名称</span>
        <input v-model="q" type="search" />
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
      <button class="button button-ghost button-compact" type="submit">查询</button>
    </form>

    <form v-if="editing" class="master-form" data-testid="catalog-form" @submit.prevent="onSave">
      <div class="field-grid">
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
      </div>
      <p class="field-hint">毒麻精放标记本期关闭。新建药品会自动建立数量为 0 的库存；治疗项目不入库存。改价只影响之后开单。</p>
      <div class="panel-actions">
        <button class="button button-compact" type="submit" data-testid="catalog-save" :disabled="saving">
          {{ saving ? '正在保存…' : '保存' }}
        </button>
        <button class="button button-ghost button-compact" type="button" @click="editing = false">取消</button>
      </div>
    </form>

    <p v-if="notice" class="banner" :class="{ 'banner-ok': !noticeError }" :role="noticeError ? 'alert' : 'status'" data-testid="catalog-notice">
      {{ notice }}<span v-if="errorCode">（{{ errorCode }}）</span>
    </p>
    <p v-if="loading" class="shell-status">正在读取…</p>
    <p v-else-if="rows.length === 0" class="result-empty">
      {{ q || status || itemType ? '没有符合条件的项目' : '暂无收费/药品项目' }}
    </p>
    <div v-else class="result-wrap">
      <table class="register-table" data-testid="catalog-list">
        <caption class="sr-only">收费和药品目录</caption>
        <thead>
          <tr>
            <th>编码</th>
            <th>名称</th>
            <th>类型</th>
            <th>规格</th>
            <th>单位</th>
            <th>单价</th>
            <th>状态</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="row in rows" :key="row.catalogItemId">
            <td class="mono">{{ row.itemNo }}</td>
            <td>{{ row.itemName }}</td>
            <td>{{ typeLabel(row.itemType) }}</td>
            <td>{{ row.spec || '—' }}</td>
            <td>{{ row.units }}</td>
            <td>{{ formatMoney(row.price) }}</td>
            <td><span class="status-pill" :data-status="row.status">{{ statusLabel(row.status) }}</span></td>
            <td class="ticket-actions">
              <button class="text-btn" type="button" @click="startEdit(row)">编辑</button>
              <button class="text-btn" type="button" :data-testid="`catalog-toggle-${row.itemNo}`" @click="toggle(row)">
                {{ row.status === 'ACTIVE' ? '停用' : '启用' }}
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </section>
</template>
