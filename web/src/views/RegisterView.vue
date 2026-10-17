<script setup>
// 2026-10-12 P1-H05-REGISTER-UI 挂号页。患者从建档 ?patientId= 带入，只选科室和医生。
// 2026-10-12 可联调：POST/GET /registers，字段 departmentId。
// 2026-10-12 P1-H05b-REGISTER-CHECKIN-UI 正常号、已用号显示报到。Mock 可模拟 409。
// 2026-10-13 P1-H05-MASTER-UI 科室和医生下拉只列启用项。
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { ApiError } from '../api/errors.js'
import { useMock } from '../api/config.js'
import { cancelRegister, checkInRegister, completeRegister, createRegister, listRegisters } from '../api/registerApi.js'
import { createRegisterFeeCharge, listCharges, payExistingCharge, refundCharge } from '../api/chargeApi.js'
import { fetchPatient } from '../api/patientApi.js'
import { sexLabel, todayInShanghai } from '../api/patientContract.js'
import { session } from '../auth/session.js'
import { fetchDepartments, fetchDoctors } from '../master/api.js'
import BusinessPageHeader from '../components/BusinessPageHeader.vue'
import BusinessEmptyState from '../components/BusinessEmptyState.vue'
import { recordAction, setCharges, setPatient, setRegister } from '../workbench/outpatientContext.js'
import './clinical-design.css'
import {
  REGISTER_SORT,
  canCancel,
  canCheckIn,
  canComplete,
  completeRejectMessage,
  cancelRejectMessage,
  registerSortName,
  deptName,
  doctorName,
  doctorsForDept,
  selectableDepartments,
  maskIdNumber,
  statusLabel,
} from '../register/master.js'

const route = useRoute()
const visitDate = todayInShanghai()
const patientRoute = { name: 'patient-search' }

const patient = ref(null)
const tickets = ref([])
const loading = ref(false)
const notFound = ref(false)
const loadError = ref('')
const listNote = ref('')
const departmentId = ref('')
const doctorId = ref('')
const saving = ref(false)
const cancellingId = ref('')
const checkingInId = ref('')
const completingId = ref('')
const simulateCheckInConflict = ref(false)
const mockMode = useMock()
const notice = ref('')
const noticeError = ref(false)
const registerFee = ref(0)
const feeCharges = ref({})
const feeWorkingId = ref('')
// 2026-10-14 P1-H05-WEB-USABILITY: real reads never fall back to demo data.
const liveDepartments = ref([])
const liveDoctors = ref([])
const masterLoading = ref(!mockMode)
const masterReady = ref(mockMode)
const masterError = ref('')
const ticketsLoading = ref(false)
const ticketsReady = ref(false)
let patientLoadSeq = 0
let ticketsLoadSeq = 0

const patientId = computed(() => readQuery(route.query.patientId))
const departments = computed(() => {
  if (mockMode) return selectableDepartments()
  return liveDepartments.value.filter((item) => item.status === 'ACTIVE')
})
const doctors = computed(() => {
  if (mockMode) return doctorsForDept(departmentId.value)
  const id = departmentId.value
  if (!id) return []
  return liveDoctors.value.filter((item) => item.departmentId === id && item.status === 'ACTIVE')
})
const canSave = computed(() => Boolean(patient.value) && masterReady.value && !masterLoading.value && ticketsReady.value && !ticketsLoading.value && !saving.value)
function readQuery(raw) {
  const value = Array.isArray(raw) ? raw[0] : raw
  return typeof value === 'string' ? value.trim() : ''
}

function syncRegisterContext() {
  const selected = tickets.value.find((item) => item.registerId === readQuery(route.query.registerId)) || tickets.value[0]
  if (selected) setRegister(selected)
  setCharges(Object.values(feeCharges.value))
}

watch(departmentId, () => {
  if (!doctors.value.some((item) => item.doctorId === doctorId.value)) doctorId.value = ''
})

function departmentLabel(id) {
  if (mockMode) return deptName(id)
  return liveDepartments.value.find((item) => item.departmentId === id)?.name || id || '—'
}

function doctorLabel(id) {
  if (mockMode) return doctorName(id)
  return liveDoctors.value.find((item) => item.doctorId === id)?.name || id || '—'
}

watch(patientId, () => {
  load()
}, { immediate: true })

async function loadMaster() {
  if (mockMode) return
  masterLoading.value = true
  masterReady.value = false
  masterError.value = ''
  try {
    const [depts, docs] = await Promise.all([
      fetchDepartments(session.accessToken, {}),
      fetchDoctors(session.accessToken, {}),
    ])
    liveDepartments.value = depts.items.map((item) => ({
      departmentId: item.departmentId,
      name: item.deptName,
      status: item.status,
    }))
    liveDoctors.value = docs.items.map((item) => ({
      doctorId: item.doctorId,
      name: item.doctorName,
      departmentId: item.departmentId,
      status: item.status,
    }))
    masterReady.value = true
    if (!departments.value.some((item) => item.departmentId === departmentId.value)) departmentId.value = ''
    if (!doctors.value.some((item) => item.doctorId === doctorId.value)) doctorId.value = ''
  } catch (error) {
    liveDepartments.value = []
    liveDoctors.value = []
    masterError.value = error instanceof ApiError ? error.message : '无法读取科室和医生'
  } finally {
    masterLoading.value = false
  }
}

onMounted(loadMaster)

async function load() {
  const seq = ++patientLoadSeq
  ++ticketsLoadSeq
  const id = patientId.value
  patient.value = null
  tickets.value = []
  feeCharges.value = {}
  ticketsReady.value = false
  ticketsLoading.value = false
  notFound.value = false
  loadError.value = ''
  listNote.value = ''
  notice.value = ''
  departmentId.value = ''
  doctorId.value = ''
  if (!id) { loading.value = false; return }
  loading.value = true
  try {
    const found = await fetchPatient(session.accessToken, id)
    if (seq !== patientLoadSeq) return
    patient.value = found
    setPatient(found)
    await refreshTickets()
  } catch (error) {
    if (seq !== patientLoadSeq) return
    if (error instanceof ApiError && error.status === 404) {
      notFound.value = true
      return
    }
    loadError.value = error instanceof ApiError ? error.message : '无法读取患者'
  } finally {
    if (seq === patientLoadSeq) loading.value = false
  }
}

async function refreshTickets() {
  const id = patientId.value
  if (!id) return
  const seq = ++ticketsLoadSeq
  ticketsLoading.value = true
  listNote.value = ''
  try {
    const listed = await listRegisters(session.accessToken, {
      patientId: id,
      visitDate,
    })
    if (seq !== ticketsLoadSeq || id !== patientId.value) return
    tickets.value = listed.items
    syncRegisterContext()
    ticketsReady.value = true
    listNote.value = ''
  } catch (error) {
    if (seq !== ticketsLoadSeq || id !== patientId.value) return
    ticketsReady.value = false
    listNote.value = error instanceof ApiError ? error.message : '暂时无法读取挂号列表'
  } finally {
    if (seq === ticketsLoadSeq) ticketsLoading.value = false
  }
}

function feeChargeFor(row) {
  return feeCharges.value[row.registerId] || null
}

async function ensureFeeCharge(row) {
  const cached = feeChargeFor(row)
  if (cached) return cached
  const listed = await listCharges(session.accessToken, { registerId: row.registerId, todayOnly: false })
  const found = listed.items.find((item) => item.registerId === row.registerId && item.items.some((line) => line.itemNo === 'REGISTER-FEE'))
  if (found) feeCharges.value = { ...feeCharges.value, [row.registerId]: found }
  return found || null
}

async function onCreateFee(row) {
  if (!row || mockMode || feeWorkingId.value) return
  feeWorkingId.value = row.registerId
  showNotice('', false)
  try {
    const charge = await createRegisterFeeCharge(session.accessToken, row.registerId)
    feeCharges.value = { ...feeCharges.value, [row.registerId]: charge }
    syncRegisterContext()
    showNotice('挂号费收费单已生成，请确认现金收费。', false)
  } catch (error) {
    if (error instanceof ApiError && error.code === 'REGISTER_FEE_ALREADY_CHARGED') {
      try { await ensureFeeCharge(row) } catch { /* keep the server error below */ }
    }
    showNotice(error instanceof ApiError ? error.message : '挂号费收费单生成失败', true)
  } finally {
    feeWorkingId.value = ''
  }
}

async function onPayFee(row) {
  if (!row || mockMode || feeWorkingId.value) return
  feeWorkingId.value = row.registerId
  showNotice('', false)
  try {
    const charge = await ensureFeeCharge(row)
    if (!charge) throw new Error('请先生成挂号费收费单')
    const result = await payExistingCharge(session.accessToken, charge.chargeId, charge.squareFee, crypto.randomUUID())
    feeCharges.value = { ...feeCharges.value, [row.registerId]: { ...charge, ...result.charge, paid: true } }
    syncRegisterContext()
    recordAction('挂号费已收讫', result.charge?.squareNo || charge.chargeId || '')
    row.registerCharged = true
    showNotice('挂号费已现金收讫。', false)
  } catch (error) {
    showNotice(error instanceof ApiError ? error.message : '挂号费收费失败', true)
  } finally {
    feeWorkingId.value = ''
  }
}

async function onRefundFee(row) {
  if (!row || mockMode || feeWorkingId.value) return
  feeWorkingId.value = row.registerId
  showNotice('', false)
  try {
    const charge = await ensureFeeCharge(row)
    if (!charge || !charge.paid) throw new Error('当前没有可退的挂号费')
    const result = await refundCharge(session.accessToken, charge.chargeId, {
      payType: 'CASH',
      reason: '挂号费全额退费',
      idempotencyKey: crypto.randomUUID(),
      items: charge.items.map((item) => ({ chargeItemId: item.chargeItemId, refundQty: item.refundableQty || item.qty })),
    })
    if (result.fullRefund) {
      const next = { ...feeCharges.value }
      delete next[row.registerId]
      feeCharges.value = next
      syncRegisterContext()
      row.registerCharged = false
    }
    showNotice('挂号费已全额退费。', false)
  } catch (error) {
    showNotice(error instanceof ApiError ? error.message : '挂号费退费失败', true)
  } finally {
    feeWorkingId.value = ''
  }
}

function showNotice(message, isError) {
  notice.value = message
  noticeError.value = isError
}

async function onSave() {
  if (!canSave.value) return
  showNotice('', false)
  if (!departments.value.some((item) => item.departmentId === departmentId.value)) {
    showNotice('请选择科室', true)
    return
  }
  if (!doctorId.value || !doctors.value.some((item) => item.doctorId === doctorId.value)) {
    showNotice('请选择医生', true)
    return
  }
  saving.value = true
  const currentPatientId = patient.value.patientId
  const currentSeq = patientLoadSeq
  try {
    const created = await createRegister(session.accessToken, {
      patientId: currentPatientId,
      departmentId: departmentId.value,
      doctorId: doctorId.value,
      registerSort: REGISTER_SORT.registerSort,
      visitDate,
      registerFee: mockMode ? 0 : Math.max(0, Number(registerFee.value) || 0),
    })
    if (currentSeq !== patientLoadSeq || patient.value?.patientId !== currentPatientId) return
    tickets.value = [created, ...tickets.value.filter((item) => item.registerId !== created.registerId)]
    setRegister(created)
    recordAction('挂号已创建', created.registerNo || created.registerId || '')
    showNotice('已保存，状态为正常号', false)
  } catch (error) {
    if (currentSeq !== patientLoadSeq || patient.value?.patientId !== currentPatientId) return
    showNotice(error instanceof ApiError ? error.message : '挂号失败，请稍后重试', true)
  } finally {
    saving.value = false
  }
}

async function onCancel(row) {
  if (!row || cancellingId.value || checkingInId.value) return
  if (!canCancel(row.status)) {
    showNotice(cancelRejectMessage(row.status), true)
    return
  }
  cancellingId.value = row.registerId
  const currentSeq = patientLoadSeq
  showNotice('', false)
  try {
    const updated = await cancelRegister(session.accessToken, row.registerId)
    if (currentSeq !== patientLoadSeq) return
    tickets.value = tickets.value.map((item) => (item.registerId === updated.registerId ? updated : item))
    setRegister(updated)
    recordAction('挂号已退号', updated.registerNo || updated.registerId || '')
    showNotice('已退号', false)
  } catch (error) {
    if (currentSeq !== patientLoadSeq) return
    showNotice(error instanceof ApiError ? error.message : '退号失败，请稍后重试', true)
  } finally {
    cancellingId.value = ''
  }
}

async function onCheckIn(row) {
  if (!row || checkingInId.value || completingId.value || cancellingId.value) return
  if (!canCheckIn(row.status)) {
    showNotice('当前状态不允许该操作', true)
    return
  }
  checkingInId.value = row.registerId
  const currentSeq = patientLoadSeq
  showNotice('', false)
  try {
    const updated = await checkInRegister(session.accessToken, row.registerId, {
      simulateConflict: mockMode && simulateCheckInConflict.value,
    })
    if (currentSeq !== patientLoadSeq) return
    tickets.value = tickets.value.map((item) => (item.registerId === updated.registerId ? updated : item))
    setRegister(updated)
    recordAction('患者已报到', updated.registerNo || updated.registerId || '')
    showNotice('已报到', false)
  } catch (error) {
    if (currentSeq !== patientLoadSeq) return
    showNotice(error instanceof ApiError ? error.message : '报到失败，请稍后重试', true)
  } finally {
    checkingInId.value = ''
  }
}

async function onComplete(row) {
  if (!row || completingId.value || checkingInId.value || cancellingId.value) return
  if (!canComplete(row.status)) {
    showNotice(completeRejectMessage(row.status), true)
    return
  }
  completingId.value = row.registerId
  const currentSeq = patientLoadSeq
  showNotice('', false)
  try {
    const updated = await completeRegister(session.accessToken, row.registerId)
    if (currentSeq !== patientLoadSeq) return
    tickets.value = tickets.value.map((item) => (item.registerId === updated.registerId ? updated : item))
    setRegister(updated)
    recordAction('就诊已完成', updated.registerNo || updated.registerId || '')
    showNotice('已完成就诊', false)
  } catch (error) {
    if (currentSeq !== patientLoadSeq) return
    showNotice(error instanceof ApiError ? error.message : '完成就诊失败，请稍后重试', true)
  } finally {
    completingId.value = ''
  }
}
</script>

<template>
  <section class="register-page" aria-labelledby="register-title">
    <p class="crumb">
      <RouterLink :to="{ name: 'home' }">首页</RouterLink>
      <span aria-hidden="true"> / </span>
      <RouterLink :to="{ name: 'workbench' }">门诊业务</RouterLink>
      <span aria-hidden="true"> / </span>
      <span>挂号</span>
    </p>

    <BusinessPageHeader title-id="register-title" title="门诊挂号" eyebrow="OUTPATIENT REGISTRATION" icon="register" description="核对患者资料，选择科室和医生。挂号、报到、完成就诊与退号在同一工作区完成。" />

    <BusinessEmptyState v-if="!patientId" class="clinical-register-empty" data-testid="register-empty" title="请先选择患者" description="先在患者查询中选择一位患者，再为本次就诊安排科室和医生。" eyebrow="开始一次就诊" icon="register">
      <RouterLink class="button button-compact" :to="patientRoute">患者查询</RouterLink>
      <template #detail><div class="clinical-empty-steps"><span><b>01</b> 选择患者</span><i aria-hidden="true">→</i><span><b>02</b> 科室与医生</span><i aria-hidden="true">→</i><span><b>03</b> 保存挂号</span></div></template>
    </BusinessEmptyState>

    <p v-else-if="loading" class="shell-status">正在读取患者…</p>

    <BusinessEmptyState v-else-if="notFound" data-testid="register-missing" title="未找到患者" description="这个患者标识没有对应档案，请在患者查询中重新选择。" icon="patient"><RouterLink class="button button-ghost button-compact" :to="patientRoute">患者查询</RouterLink></BusinessEmptyState>

    <div v-else-if="loadError" class="banner clinical-read-error" role="alert"><p>{{ loadError }}</p><button class="button button-ghost button-compact" type="button" @click="load">重试患者资料</button></div>

    <div v-else-if="patient" class="clinical-register-workspace">
      <div class="clinical-register-layout">
      <section class="patient-panel clinical-patient-profile" aria-labelledby="register-patient">
        <div class="panel-head">
          <span class="clinical-kicker">就诊患者</span>
        </div>
        <div class="clinical-profile-identity"><span class="clinical-profile-avatar" aria-hidden="true">{{ patient.name?.slice(0, 1) || '患' }}</span><div><h2 id="register-patient">{{ patient.name }}</h2><p>{{ patient.patientNo || '患者号未提供' }}</p></div><span class="clinical-badge">档案已载入</span></div>
        <dl class="summary-grid">
          <div>
            <dt>姓名</dt>
            <dd>{{ patient.name }}</dd>
          </div>
          <div>
            <dt>性别</dt>
            <dd>{{ sexLabel(patient.sex) }}</dd>
          </div>
          <div>
            <dt>出生日期</dt>
            <dd>{{ patient.birthday || '—' }}</dd>
          </div>
          <div>
            <dt>患者号</dt>
            <dd>{{ patient.patientNo || '—' }}</dd>
          </div>
          <div>
            <dt>身份证</dt>
            <dd>{{ maskIdNumber(patient.idNumber) }}</dd>
          </div>
        </dl>
        <p class="clinical-profile-note"><span aria-hidden="true">◇</span> 主索引不能在挂号页修改。</p>
      </section>

      <form class="patient-panel clinical-register-form" data-testid="register-form" novalidate @submit.prevent="onSave">
        <div class="panel-head">
          <div><span class="clinical-kicker">本次就诊</span><h2>挂号信息</h2></div>
          <span class="clinical-date-tag">{{ visitDate }}</span>
        </div>
        <div class="clinical-visit-type"><span class="clinical-visit-icon" aria-hidden="true">+</span><div><strong>普通号</strong><p>当天门诊 · 选择科室后关联可用医生</p></div></div>
        <p v-if="masterLoading" class="shell-status" role="status">正在读取科室和医生…</p>
        <div v-else-if="masterError" class="banner clinical-read-error" role="alert" data-testid="register-master-error"><p>{{ masterError }}。读取成功后才能保存挂号。</p><button class="button button-ghost button-compact" type="button" @click="loadMaster">重新读取科室和医生</button></div>
        <p v-else-if="!departments.length" class="field-hint" role="status">暂无启用科室，请先维护基础数据。</p>
        <div class="field-grid">
          <label class="field">
            <span>科室 <abbr class="req" title="必填">*</abbr></span>
            <select v-model="departmentId" data-testid="dept-select" :disabled="!masterReady || masterLoading || saving">
              <option value="">请选择科室</option>
              <option v-for="item in departments" :key="item.departmentId" :value="item.departmentId">{{ item.name }}</option>
            </select>
          </label>
          <label class="field">
            <span>医生 <abbr class="req" title="必填">*</abbr></span>
            <select v-model="doctorId" data-testid="doctor-select" :disabled="!departmentId || !masterReady || masterLoading || saving">
              <option value="">请选择医生</option>
              <option v-for="item in doctors" :key="item.doctorId" :value="item.doctorId">{{ item.name }}</option>
            </select>
          </label>
          <label class="field">
            <span>号别</span>
            <input value="普通号" disabled />
          </label>
          <label class="field">
            <span>就诊日期</span>
            <input :value="visitDate" disabled />
          </label>
          <label v-if="!mockMode" class="field">
            <span>挂号费（元）</span>
            <input v-model.number="registerFee" type="number" min="0" max="9999999999999999.99" step="0.01" inputmode="decimal" />
          </label>
        </div>

        <p
          v-if="notice"
          class="banner"
          :class="{ 'banner-ok': !noticeError }"
          :role="noticeError ? 'alert' : 'status'"
          data-testid="register-notice"
        >
          {{ notice }}
        </p>

        <div class="clinical-form-footer">
          <p>保存后为正常号，可在下方报到</p>
          <div class="panel-actions">
          <button class="button button-compact" type="submit" data-testid="save-register" :disabled="!canSave">
            {{ saving ? '正在保存…' : '保存挂号' }}
          </button>
          </div>
        </div>
      </form>
      </div>

      <section class="patient-panel clinical-ticket-panel" aria-labelledby="register-list-title">
        <div class="panel-head">
          <div><span class="clinical-kicker">就诊记录</span><h2 id="register-list-title">当日挂号 <span class="clinical-count">{{ tickets.length }}</span></h2></div>
          <p class="clinical-panel-description">正常号和已用号可以报到，就诊中可以完成就诊，仅正常号可以退号。</p>
        </div>
        <label v-if="mockMode" class="checkin-demo">
          <input v-model="simulateCheckInConflict" type="checkbox" data-testid="mock-checkin-conflict" />
          <span>模拟报到冲突（下次报到返回 409，号状态不变）</span>
        </label>
        <p v-if="ticketsLoading" class="shell-status" role="status">正在读取当日挂号…</p>
        <div v-else-if="listNote" class="banner clinical-read-error" role="alert"><p>{{ listNote }}。请先重试以核对当日挂号。</p><button class="button button-ghost button-compact" type="button" @click="refreshTickets">重试挂号列表</button></div>
        <BusinessEmptyState v-else-if="ticketsReady && tickets.length === 0" title="当日暂无挂号。" description="为该患者保存挂号后，就诊记录与报到操作会显示在这里。" icon="register" />
        <div v-if="tickets.length" class="result-wrap clinical-table-scroll" tabindex="0" role="region" aria-label="当日挂号列表，可左右滚动">
          <table class="register-table" data-testid="register-list">
            <caption class="sr-only">
              当日挂号
            </caption>
            <thead>
              <tr>
                <th>单号</th>
                <th>科室</th>
                <th>医生</th>
                <th>号别</th>
                <th>挂号费</th>
                <th>状态</th>
                <th>操作</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="row in tickets" :key="row.registerId">
                <td class="mono">{{ row.registerNo || row.registerId }}</td>
                <td>{{ row.departmentName || departmentLabel(row.departmentId) }}</td>
                <td>{{ row.doctorName || doctorLabel(row.doctorId) }}</td>
                <td>{{ registerSortName(row.registerSort) }}</td>
                <td class="ticket-fee">
                  <span v-if="Number(row.registerFee) > 0">￥{{ Number(row.registerFee).toFixed(2) }}</span>
                  <span v-else>—</span>
                  <small v-if="row.registerCharged" class="ticket-fee-status">已收费</small>
                  <small v-else-if="feeChargeFor(row)?.paid" class="ticket-fee-status">已收费</small>
                  <small v-else-if="feeChargeFor(row)" class="ticket-fee-status">待收费</small>
                </td>
                <td>
                  <span class="ticket-status" :data-status="row.status">{{ statusLabel(row.status) }}</span>
                </td>
                <td class="ticket-actions">
                  <button v-if="!mockMode && Number(row.registerFee) > 0 && !row.registerCharged && !feeChargeFor(row)" class="button button-compact" type="button" :disabled="Boolean(feeWorkingId)" @click="onCreateFee(row)">生成收费单</button>
                  <button v-if="!mockMode && Number(row.registerFee) > 0 && !row.registerCharged && feeChargeFor(row)" class="button button-compact" type="button" :disabled="Boolean(feeWorkingId)" @click="onPayFee(row)">{{ feeWorkingId === row.registerId ? '收费中…' : '现金收费' }}</button>
                  <button v-if="!mockMode && Number(row.registerFee) > 0 && row.registerCharged" class="button button-ghost button-compact" type="button" :disabled="Boolean(feeWorkingId)" @click="onRefundFee(row)">{{ feeWorkingId === row.registerId ? '退费中…' : '挂号费退费' }}</button>
                  <button
                    v-if="canCheckIn(row.status)"
                    class="button button-compact"
                    type="button"
                    :data-testid="`checkin-${row.status}`"
                    :disabled="Boolean(checkingInId) || Boolean(cancellingId)"
                    @click="onCheckIn(row)"
                  >
                    {{ checkingInId === row.registerId ? '报到中…' : '报到' }}
                  </button>
                  <button
                    v-if="canComplete(row.status)"
                    class="button button-compact"
                    type="button"
                    :data-testid="`complete-${row.status}`"
                    :disabled="Boolean(completingId) || Boolean(cancellingId) || Boolean(checkingInId)"
                    @click="onComplete(row)"
                  >
                    {{ completingId === row.registerId ? '完成中…' : '完成就诊' }}
                  </button>
                  <button
                    class="button button-ghost button-compact"
                    type="button"
                    :data-testid="`cancel-${row.status}`"
                    :disabled="!canCancel(row.status) || Boolean(cancellingId) || Boolean(checkingInId) || Boolean(completingId)"
                    :title="canCancel(row.status) ? '退号' : cancelRejectMessage(row.status)"
                    @click="onCancel(row)"
                  >
                    {{ cancellingId === row.registerId ? '退号中…' : '退号' }}
                  </button>
                  <p v-if="!canCancel(row.status)" class="ticket-reject">{{ cancelRejectMessage(row.status) }}</p>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>
    </div>
  </section>
</template>
