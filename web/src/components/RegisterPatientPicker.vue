<script setup>
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { ApiError } from '../api/errors.js'
import { searchRegistrationPatients } from '../api/patientApi.js'
import { sexLabel } from '../api/patientContract.js'

const props = defineProps({
  open: { type: Boolean, default: false },
  accessToken: { type: String, default: '' },
})
const emit = defineEmits(['close', 'select'])

const keyword = ref('')
const activeKeyword = ref('')
const items = ref([])
const total = ref(0)
const offset = ref(0)
const loading = ref(false)
const error = ref('')
const requestSeq = ref(0)
const page = computed(() => Math.floor(offset.value / 50) + 1)
const pageCount = computed(() => Math.max(1, Math.ceil(total.value / 50)))

watch(() => props.open, (open) => {
  if (!open) return
  keyword.value = ''
  activeKeyword.value = ''
  offset.value = 0
  void loadPage(0, '')
})

async function loadPage(nextOffset = offset.value, query = activeKeyword.value) {
  const seq = ++requestSeq.value
  loading.value = true
  error.value = ''
  try {
    const result = await searchRegistrationPatients(props.accessToken, query, { offset: nextOffset, limit: 50 })
    if (seq !== requestSeq.value) return
    items.value = result.items
    total.value = result.total
    offset.value = nextOffset
  } catch (cause) {
    if (seq !== requestSeq.value) return
    items.value = []
    total.value = 0
    error.value = cause instanceof ApiError ? cause.message : '暂时无法读取患者列表，请重试。'
  } finally {
    if (seq === requestSeq.value) loading.value = false
  }
}

function submitSearch() {
  activeKeyword.value = keyword.value.trim()
  void loadPage(0, activeKeyword.value)
}

function selectPatient(item) {
  if (item.status !== 'ACTIVE' || !item.selectable) return
  emit('select', item)
}

function close() {
  ++requestSeq.value
  emit('close')
}

function handleKeydown(event) {
  if (props.open && event.key === 'Escape') close()
}

onMounted(() => window.addEventListener('keydown', handleKeydown))
onBeforeUnmount(() => window.removeEventListener('keydown', handleKeydown))

</script>

<template>
  <div v-if="open" class="register-picker-backdrop" role="presentation" @click.self="close">
    <section class="register-picker" role="dialog" aria-modal="true" aria-labelledby="register-picker-title" data-testid="register-patient-picker">
      <header class="register-picker-head">
        <div><p class="register-picker-kicker">PATIENT ARCHIVE</p><h2 id="register-picker-title">选择患者</h2></div>
        <button class="register-picker-close" type="button" aria-label="关闭患者选择" title="关闭" @click="close">×</button>
      </header>

      <form class="register-picker-search" role="search" @submit.prevent="submitSearch">
        <label for="register-patient-keyword">患者关键词</label>
        <input id="register-patient-keyword" v-model="keyword" type="search" maxlength="64" autocomplete="off" placeholder="姓名 / 助记码 / 电话 / 身份证号" data-testid="register-patient-keyword" />
        <button type="submit" class="button button-compact" :disabled="loading">查询</button>
      </form>

      <div class="register-picker-meta" aria-live="polite">
        <span>共 {{ total }} 位患者</span>
        <span v-if="total">第 {{ page }} / {{ pageCount }} 页</span>
      </div>

      <div v-if="loading" class="register-picker-state" role="status">正在读取患者列表…</div>
      <div v-else-if="error" class="register-picker-state register-picker-error" role="alert" data-testid="register-picker-error">
        <p>{{ error }}</p>
        <button class="button button-ghost button-compact" type="button" @click="loadPage(offset, activeKeyword)">重试</button>
      </div>
      <div v-else-if="items.length === 0" class="register-picker-state" data-testid="register-picker-empty">没有匹配的患者</div>
      <div v-else class="register-picker-table-wrap">
        <table class="register-picker-table">
          <thead><tr><th>姓名</th><th>病历号</th><th>性别</th><th>联系电话</th><th>证件号码</th><th>状态</th><th><span class="sr-only">操作</span></th></tr></thead>
          <tbody>
            <tr v-for="item in items" :key="item.patientId" :class="{ 'is-inactive': item.status === 'INACTIVE' }" :data-testid="`register-patient-row-${item.patientId}`">
              <td class="register-picker-name">{{ item.name }}</td>
              <td class="mono">{{ item.patientNo || '—' }}</td>
              <td>{{ sexLabel(item.sex) }}</td>
              <td class="mono">{{ item.phoneMasked || '—' }}</td>
              <td class="mono">{{ item.idNumberMasked || '—' }}</td>
              <td><span class="register-picker-status" :data-status="item.status">{{ item.status === 'ACTIVE' ? '在用' : '停用' }}</span></td>
              <td><button type="button" class="button button-compact" :disabled="item.status !== 'ACTIVE' || !item.selectable" :title="item.status === 'ACTIVE' ? '选择此患者' : '停用患者不可选择'" @click="selectPatient(item)">{{ item.status === 'ACTIVE' ? '选择' : '不可选' }}</button></td>
            </tr>
          </tbody>
        </table>
      </div>

      <footer class="register-picker-foot">
        <span>每页 50 条</span>
        <div>
          <button type="button" class="button button-ghost button-compact" :disabled="loading || offset === 0" @click="loadPage(Math.max(0, offset - 50))">上一页</button>
          <button type="button" class="button button-ghost button-compact" :disabled="loading || offset + 50 >= total" @click="loadPage(offset + 50)">下一页</button>
          <button type="button" class="button button-ghost button-compact" @click="close">取消</button>
        </div>
      </footer>
    </section>
  </div>
</template>

<style scoped>
.register-picker-backdrop { position: fixed; inset: 0; z-index: 1200; display: grid; place-items: center; padding: 24px; background: rgb(16 33 42 / 42%); }
.register-picker { width: min(980px, 100%); max-height: min(760px, calc(100vh - 48px)); display: flex; flex-direction: column; overflow: hidden; border: 1px solid #cbdde3; border-radius: 8px; background: #fff; box-shadow: 0 20px 60px rgb(16 33 42 / 24%); color: #193b4b; }
.register-picker-head { display: flex; justify-content: space-between; align-items: center; padding: 20px 24px 16px; border-bottom: 1px solid #e1eaed; }
.register-picker-kicker { margin: 0 0 4px; color: #527381; font-size: 10px; font-weight: 700; }
.register-picker-head h2 { margin: 0; font-size: 20px; }
.register-picker-close { width: 36px; height: 36px; border: 1px solid #d8e4e8; border-radius: 4px; background: #fff; color: #315968; font-size: 22px; cursor: pointer; }
.register-picker-search { display: grid; grid-template-columns: auto minmax(200px, 1fr) auto; gap: 12px; align-items: center; padding: 16px 24px; }
.register-picker-search label { font-size: 13px; font-weight: 600; }
.register-picker-search input { min-width: 0; height: 40px; padding: 0 12px; border: 1px solid #cbdde3; border-radius: 4px; color: inherit; }
.register-picker-meta { display: flex; justify-content: space-between; padding: 0 24px 12px; color: #627f8a; font-size: 12px; }
.register-picker-table-wrap { min-height: 160px; overflow: auto; border-top: 1px solid #e1eaed; }
.register-picker-table { width: 100%; border-collapse: collapse; white-space: nowrap; font-size: 13px; }
.register-picker-table th, .register-picker-table td { padding: 11px 14px; border-bottom: 1px solid #edf2f4; text-align: left; }
.register-picker-table th { position: sticky; top: 0; background: #f5f9fa; color: #587783; font-size: 11px; font-weight: 600; }
.register-picker-name { font-weight: 700; }
.register-picker-status { padding: 3px 7px; border-radius: 3px; background: #e4f4ee; color: #26704e; font-size: 11px; }
.register-picker-status[data-status="INACTIVE"] { background: #f1f3f4; color: #71818a; }
.register-picker-table tr.is-inactive { color: #87979d; background: #fafbfb; }
.register-picker-table button:disabled { opacity: .6; cursor: not-allowed; }
.register-picker-state { display: grid; justify-items: center; align-content: center; gap: 12px; min-height: 200px; padding: 20px; color: #627f8a; text-align: center; }
.register-picker-state p { margin: 0; }
.register-picker-error { color: #8c3f35; }
.register-picker-foot { display: flex; justify-content: space-between; align-items: center; padding: 14px 24px; border-top: 1px solid #e1eaed; color: #627f8a; font-size: 12px; }
.register-picker-foot > div { display: flex; gap: 8px; }
@media (max-width: 680px) {
  .register-picker-backdrop { padding: 10px; }
  .register-picker { max-height: calc(100vh - 20px); }
  .register-picker-search { grid-template-columns: 1fr auto; padding: 12px 14px; }
  .register-picker-search label { grid-column: 1 / -1; }
  .register-picker-meta { padding-inline: 14px; }
  .register-picker-head, .register-picker-foot { padding-inline: 14px; }
  .register-picker-foot { align-items: flex-start; gap: 8px; flex-direction: column; }
  .register-picker-foot > div { width: 100%; justify-content: flex-end; }
}
</style>
