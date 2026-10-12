<script setup>
// 2026-10-12 P1-H05-ORDER-UI 门诊开单 Mock。主字段 registerId；真实 /orders 尚未联调。
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useMock } from '../api/config.js'
import { ApiError } from '../api/errors.js'
import {
  abateOrder,
  commitOrder,
  fetchOrderRegister,
  listOrderRegisters,
  listOrders,
  saveOrder,
} from '../api/orderApi.js'
import {
  CATALOG,
  FREQUENCY_TEXTS,
  USAGE_TEXTS,
  money,
  orderStatusLabel,
  registerIsOrderable,
  round2,
  stockWarnings,
} from '../api/orderContract.js'
import { session } from '../auth/session.js'
import { DOCTORS } from '../register/master.js'

const route = useRoute()
const router = useRouter()
const mockMode = useMock()

const registers = ref([])
const pickedId = ref('')
const register = ref(null)
const invalidMessage = ref('')
const loadingRegister = ref(false)
const orders = ref([])
const orderId = ref('')
const orderNo = ref('')
const status = ref('')
const charged = ref(false)
const form = reactive({
  doctorId: '',
  diagnoseName: '',
})
const lines = ref([blankLine()])
const errorMessage = ref('')
const infoMessage = ref('')
const saving = ref(false)
const committing = ref(false)
const abating = ref(false)
const confirmAbate = ref(false)
let loadSeq = 0

const busy = computed(() => saving.value || committing.value || abating.value)
const orderableRegisters = computed(() => registers.value.filter((item) => registerIsOrderable(item)))
const drugOptions = computed(() => CATALOG.filter((item) => item.itemType === 'DRUG'))
const treatOptions = computed(() => CATALOG.filter((item) => item.itemType === 'TREAT'))
const locked = computed(() => ['COMMIT', 'ABATE', 'PROC'].includes(status.value) || charged.value)
const canAbate = computed(() => Boolean(orderId.value) && !charged.value && (status.value === 'SAVE' || status.value === 'COMMIT'))
const warnings = computed(() => stockWarnings(lines.value.filter((line) => line.itemNo && Number(line.amount) > 0)))
const totalFee = computed(() => round2(lines.value.reduce((sum, line) => {
  const catalog = catalogOf(line.itemNo)
  const amount = Number(line.amount)
  if (!catalog || !Number.isFinite(amount) || amount <= 0) return sum
  return sum + round2(amount * catalog.price)
}, 0)))
const chargeText = computed(() => {
  if (charged.value) return '已收费'
  if (status.value === 'COMMIT') return '可收费'
  if (status.value === 'ABATE') return '已作废'
  return '不可收费'
})
const registerTo = { name: 'outpatient-module', params: { module: 'register' } }

function blankLine() {
  return {
    key: crypto.randomUUID(),
    itemNo: '',
    amount: '',
    usageText: '',
    frequencyText: '',
    dosage: '',
    dosageUnit: '',
  }
}

function catalogOf(itemNo) {
  return CATALOG.find((item) => item.itemNo === itemNo) || null
}

function lineMoney(line) {
  const catalog = catalogOf(line.itemNo)
  const amount = Number(line.amount)
  if (!catalog || !Number.isFinite(amount) || amount <= 0) return '—'
  return money(amount * catalog.price)
}

function lineShort(line) {
  const catalog = catalogOf(line.itemNo)
  const amount = Number(line.amount)
  if (!catalog || catalog.stockQty == null || !Number.isFinite(amount)) return false
  return amount > catalog.stockQty
}

function queryRegisterId() {
  const value = route.query.registerId
  const raw = Array.isArray(value) ? value[0] : value
  return typeof raw === 'string' ? raw.trim() : ''
}

function clearBanner() {
  errorMessage.value = ''
  infoMessage.value = ''
  confirmAbate.value = false
}

function startDraft(current) {
  orderId.value = ''
  orderNo.value = ''
  status.value = ''
  charged.value = false
  form.doctorId = current.doctorId || ''
  form.diagnoseName = ''
  lines.value = [blankLine()]
  clearBanner()
}

function applyOrder(order) {
  orderId.value = order.orderId
  orderNo.value = order.orderNo
  status.value = order.status
  charged.value = order.charged === true
  form.doctorId = order.doctorId || (register.value ? register.value.doctorId : '')
  form.diagnoseName = order.diagnoseName || ''
  const next = order.items.map((item) => ({
    key: item.orderItemId || crypto.randomUUID(),
    itemNo: item.itemNo,
    amount: String(item.amount),
    usageText: item.usageText || '',
    frequencyText: item.frequencyText || '',
    dosage: item.dosage || '',
    dosageUnit: item.dosageUnit || '',
  }))
  lines.value = next.length > 0 ? next : [blankLine()]
  confirmAbate.value = false
}

function draftBody() {
  return {
    orderId: orderId.value,
    registerId: register.value.registerId,
    departmentId: register.value.departmentId,
    doctorId: form.doctorId,
    diagnoseName: form.diagnoseName,
    patientId: register.value.patientId,
    items: lines.value.map((line) => ({
      itemNo: line.itemNo,
      amount: line.amount,
      usageText: line.usageText,
      frequencyText: line.frequencyText,
      dosage: line.dosage,
      dosageUnit: line.dosageUnit,
    })),
  }
}

function orderChargeText(order) {
  if (order.charged) return '已收费'
  if (order.status === 'COMMIT') return '可收费'
  if (order.status === 'ABATE') return '已作废'
  return '不可收费'
}

async function refreshOrders() {
  if (!register.value) {
    orders.value = []
    return
  }
  const listed = await listOrders(session.accessToken, register.value.registerId)
  orders.value = listed.items
}

async function loadRegisters() {
  if (!mockMode) return
  try {
    const listed = await listOrderRegisters(session.accessToken)
    registers.value = listed.items
  } catch {
    registers.value = []
  }
}

async function loadRegister(id) {
  const seq = ++loadSeq
  register.value = null
  invalidMessage.value = ''
  if (!mockMode || !id) {
    loadingRegister.value = false
    return
  }
  loadingRegister.value = true
  clearBanner()
  try {
    const found = await fetchOrderRegister(session.accessToken, id)
    if (seq !== loadSeq) return
    if (!registerIsOrderable(found)) {
      invalidMessage.value = found.status === 'BACK' ? '该挂号已退号，不能开单。' : '该挂号当前不能开单。'
      return
    }
    register.value = found
    startDraft(found)
    await refreshOrders()
  } catch (error) {
    if (seq !== loadSeq) return
    invalidMessage.value = error instanceof ApiError && error.code === 'ENCOUNTER_REQUIRED'
      ? '请先挂号'
      : (error instanceof ApiError ? error.message : '未找到有效挂号')
  } finally {
    if (seq === loadSeq) loadingRegister.value = false
  }
}

function requireDoctor() {
  if (form.doctorId.trim()) return true
  errorMessage.value = '请选择开单医生'
  return false
}

async function onSave() {
  if (busy.value || locked.value || !register.value) return
  clearBanner()
  if (!requireDoctor()) return
  saving.value = true
  try {
    const order = await saveOrder(session.accessToken, draftBody())
    applyOrder(order)
    infoMessage.value = form.diagnoseName.trim()
      ? '已暂存。暂存处方不能收费。'
      : '已暂存。诊断可以后补，不影响暂存。暂存处方不能收费。'
    await refreshOrders()
  } catch (error) {
    errorMessage.value = error instanceof ApiError ? error.message : '暂存失败，请稍后重试'
  } finally {
    saving.value = false
  }
}

async function onCommit() {
  if (busy.value || locked.value || !register.value) return
  clearBanner()
  if (!requireDoctor()) return
  committing.value = true
  try {
    const saved = await saveOrder(session.accessToken, draftBody())
    applyOrder(saved)
    const committed = await commitOrder(session.accessToken, saved.orderId)
    applyOrder(committed)
    infoMessage.value = form.diagnoseName.trim()
      ? '已提交，可供收费。未收费前可以作废。'
      : '已提交，可供收费。诊断未填，不影响提交。未收费前可以作废。'
    await refreshOrders()
  } catch (error) {
    errorMessage.value = error instanceof ApiError ? error.message : '提交失败，请稍后重试'
    try {
      await refreshOrders()
    } catch {
      // 列表刷新失败时仍保留表单上的错误。
    }
  } finally {
    committing.value = false
  }
}

async function onAbate() {
  if (busy.value || !canAbate.value) return
  abating.value = true
  errorMessage.value = ''
  try {
    const order = await abateOrder(session.accessToken, orderId.value)
    applyOrder(order)
    infoMessage.value = '已作废。如需继续，请新开一张。'
    await refreshOrders()
  } catch (error) {
    errorMessage.value = error instanceof ApiError ? error.message : '作废失败，请稍后重试'
  } finally {
    abating.value = false
    confirmAbate.value = false
  }
}

function addLine() {
  if (locked.value) return
  lines.value = [...lines.value, blankLine()]
}

function removeLine(key) {
  if (locked.value) return
  const next = lines.value.filter((line) => line.key !== key)
  lines.value = next.length > 0 ? next : [blankLine()]
}

async function usePicked() {
  if (!pickedId.value) return
  await router.replace({
    name: 'outpatient-module',
    params: { module: 'order' },
    query: { registerId: pickedId.value },
  })
}

function openListed(id) {
  const found = orders.value.find((item) => item.orderId === id)
  if (!found) return
  clearBanner()
  applyOrder(found)
}

function onNew() {
  if (!register.value || busy.value) return
  startDraft(register.value)
}

async function changeRegister() {
  await router.replace({ name: 'outpatient-module', params: { module: 'order' } })
}

onMounted(loadRegisters)

watch(() => queryRegisterId(), (id) => {
  loadRegister(id)
}, { immediate: true })
</script>

<template>
  <section class="order-page" aria-labelledby="order-title">
    <p class="crumb">
      <RouterLink :to="{ name: 'home' }">首页</RouterLink>
      <span aria-hidden="true"> / </span>
      <RouterLink :to="{ name: 'workbench' }">门诊业务</RouterLink>
      <span aria-hidden="true"> / </span>
      <span>开单</span>
    </p>

    <header class="dash-head">
      <h1 id="order-title">开单</h1>
      <p class="dash-flag">门诊处方。须先有未退号的挂号；诊断建议填写，不强制。</p>
    </header>

    <div v-if="!mockMode" class="empty-panel order-empty" data-testid="order-empty">
      <h2>请先挂号</h2>
      <p class="muted">没有有效挂号时不能开单。</p>
      <p class="banner order-empty-banner" role="status">开单接口尚未联调，此页不请求处方服务。</p>
      <RouterLink class="button button-compact empty-back" :to="registerTo">去挂号</RouterLink>
    </div>

    <div v-else-if="loadingRegister" class="result-empty" role="status">正在读取挂号…</div>

    <div v-else-if="!register" class="empty-panel order-empty" data-testid="order-empty">
      <h2>请先挂号</h2>
      <p class="muted">没有有效挂号时不能开单。</p>
      <p v-if="invalidMessage" class="banner order-empty-banner" role="alert">{{ invalidMessage }}</p>
      <RouterLink class="button button-compact empty-back" :to="registerTo">去挂号</RouterLink>
      <form v-if="orderableRegisters.length" class="order-picker" @submit.prevent="usePicked">
        <label class="field">
          <span>选择已有挂号</span>
          <select v-model="pickedId" data-testid="register-picker">
            <option value="">请选择</option>
            <option v-for="item in orderableRegisters" :key="item.registerId" :value="item.registerId">
              {{ item.registerNo }} · {{ item.patientName || item.patientId }} · {{ item.departmentName }} · {{ item.doctorName }}
            </option>
          </select>
        </label>
        <button class="button button-compact" type="submit" :disabled="!pickedId">使用该挂号</button>
      </form>
    </div>

    <div v-else class="order-layout" data-testid="order-form">
      <form class="patient-panel" @submit.prevent>
        <div class="panel-head">
          <h2>门诊处方</h2>
          <button class="text-btn" type="button" @click="changeRegister">更换挂号</button>
        </div>

        <fieldset class="order-fields" :disabled="locked || busy">
          <div class="field-grid">
            <label class="field">
              <span>处方号</span>
              <input :value="orderNo || '保存后生成'" disabled data-testid="recipe-no" />
            </label>
            <label class="field">
              <span>状态</span>
              <input :value="status ? orderStatusLabel(status) : '未保存'" disabled data-testid="order-status" />
            </label>
            <label class="field">
              <span>患者</span>
              <input :value="register.patientName || '—'" disabled />
            </label>
            <label class="field">
              <span>患者号</span>
              <input :value="register.patientNo || '—'" disabled />
            </label>
            <label class="field">
              <span>挂号单号</span>
              <input :value="register.registerNo || register.registerId" disabled />
            </label>
            <label class="field">
              <span>就诊日</span>
              <input :value="register.visitDate || '—'" disabled />
            </label>
            <label class="field">
              <span>挂号科室</span>
              <input :value="register.departmentName || register.departmentId" disabled />
            </label>
            <label class="field">
              <span>挂号医生</span>
              <input :value="register.doctorName || register.doctorId" disabled />
            </label>
            <label class="field">
              <span>开单医生 <abbr class="req" title="必填">*</abbr></span>
              <select v-model="form.doctorId" data-testid="order-doctor">
                <option value="">请选择</option>
                <option v-for="item in DOCTORS" :key="item.doctorId" :value="item.doctorId">{{ item.name }}</option>
              </select>
              <span class="field-hint">默认是挂号医生，可以改。</span>
            </label>
            <label class="field">
              <span>处方类别</span>
              <input value="普通" disabled />
            </label>
            <label class="field field-wide">
              <span>诊断简述</span>
              <input
                v-model="form.diagnoseName"
                data-testid="order-diagnose"
                maxlength="80"
                placeholder="建议填写，例如上感"
                autocomplete="off"
              />
              <span class="field-hint">选填。建议填写，不填也可以暂存和提交。</span>
            </label>
          </div>

          <div class="line-scroll">
            <table class="line-table">
              <caption class="sr-only">处方明细</caption>
              <thead>
                <tr>
                  <th>行号</th>
                  <th>项目</th>
                  <th>规格</th>
                  <th>数量</th>
                  <th>单位</th>
                  <th>单价</th>
                  <th>金额</th>
                  <th>用法</th>
                  <th>频次</th>
                  <th>剂量</th>
                  <th>剂量单位</th>
                  <th></th>
                </tr>
              </thead>
              <tbody>
                <template v-for="(line, index) in lines" :key="line.key">
                  <tr>
                    <td>{{ index + 1 }}</td>
                    <td class="col-item">
                      <select v-model="line.itemNo" :data-testid="`line-item-${index}`" :aria-label="`第 ${index + 1} 行项目`">
                        <option value="">请选择</option>
                        <optgroup label="药品">
                          <option v-for="item in drugOptions" :key="item.itemNo" :value="item.itemNo">{{ item.itemName }}</option>
                        </optgroup>
                        <optgroup label="治疗">
                          <option v-for="item in treatOptions" :key="item.itemNo" :value="item.itemNo">{{ item.itemName }}</option>
                        </optgroup>
                      </select>
                    </td>
                    <td>{{ catalogOf(line.itemNo)?.spec || '—' }}</td>
                    <td class="col-narrow">
                      <input
                        v-model="line.amount"
                        :data-testid="`line-amount-${index}`"
                        type="number"
                        min="0"
                        step="0.01"
                        inputmode="decimal"
                        :aria-label="`第 ${index + 1} 行数量`"
                      />
                    </td>
                    <td>{{ catalogOf(line.itemNo)?.units || '—' }}</td>
                    <td>{{ catalogOf(line.itemNo) ? money(catalogOf(line.itemNo).price) : '—' }}</td>
                    <td>{{ lineMoney(line) }}</td>
                    <td>
                      <select v-model="line.usageText" :aria-label="`第 ${index + 1} 行用法`">
                        <option value="">可不填</option>
                        <option v-for="item in USAGE_TEXTS" :key="item" :value="item">{{ item }}</option>
                      </select>
                    </td>
                    <td>
                      <select v-model="line.frequencyText" :aria-label="`第 ${index + 1} 行频次`">
                        <option value="">可不填</option>
                        <option v-for="item in FREQUENCY_TEXTS" :key="item" :value="item">{{ item }}</option>
                      </select>
                    </td>
                    <td class="col-narrow">
                      <input v-model="line.dosage" :aria-label="`第 ${index + 1} 行剂量`" autocomplete="off" />
                    </td>
                    <td class="col-narrow">
                      <input v-model="line.dosageUnit" :aria-label="`第 ${index + 1} 行剂量单位`" autocomplete="off" />
                    </td>
                    <td>
                      <button class="text-btn text-btn-danger" type="button" @click="removeLine(line.key)">删除</button>
                    </td>
                  </tr>
                  <tr v-if="lineShort(line)" class="stock-row">
                    <td colspan="12">库存不足，仅提示，不拦截开单。</td>
                  </tr>
                </template>
              </tbody>
            </table>
          </div>
          <div class="panel-actions">
            <button class="button button-ghost button-compact" type="button" @click="addLine">添加明细</button>
          </div>
        </fieldset>

        <p class="order-total" data-testid="order-total">
          合计 <strong>{{ money(totalFee) }}</strong> 元
          <span class="charge-chip" :data-charge="chargeText">{{ chargeText }}</span>
        </p>
        <ul v-if="warnings.length" class="stock-list" data-testid="stock-tip">
          <li v-for="text in warnings" :key="text">{{ text }}</li>
        </ul>
        <p v-if="errorMessage" class="banner" role="alert" data-testid="order-error">{{ errorMessage }}</p>
        <p v-if="infoMessage" class="ok-banner" role="status" data-testid="order-committed">{{ infoMessage }}</p>

        <div class="panel-actions">
          <button v-if="!locked" class="button button-ghost button-compact" type="button" data-testid="order-save" :disabled="busy" @click="onSave">
            {{ saving ? '正在暂存…' : '暂存' }}
          </button>
          <button v-if="!locked" class="button button-compact" type="button" data-testid="order-commit" :disabled="busy" @click="onCommit">
            {{ committing ? '正在提交…' : '提交' }}
          </button>
          <button v-if="canAbate && !confirmAbate" class="button button-ghost button-compact" type="button" :disabled="busy" @click="confirmAbate = true">
            作废
          </button>
          <button v-if="canAbate && confirmAbate" class="button button-ghost button-compact" type="button" data-testid="order-abate" :disabled="busy" @click="onAbate">
            {{ abating ? '正在作废…' : '确认作废' }}
          </button>
          <button v-if="orderId" class="button button-ghost button-compact" type="button" :disabled="busy" @click="onNew">新开一张</button>
        </div>
      </form>

      <aside class="patient-panel order-side">
        <div class="panel-head">
          <h2>本挂号处方</h2>
        </div>
        <p v-if="orders.length === 0" class="result-empty">这张挂号还没有处方。</p>
        <ul v-else class="order-list">
          <li v-for="item in orders" :key="item.orderId">
            <button class="order-list-item" :class="{ 'is-active': item.orderId === orderId }" type="button" @click="openListed(item.orderId)">
              <span>{{ item.orderNo }}</span>
              <span>{{ orderStatusLabel(item.status) }} · {{ orderChargeText(item) }}</span>
              <strong>{{ money(item.totalFee) }}</strong>
            </button>
          </li>
        </ul>
      </aside>
    </div>
  </section>
</template>
