<script setup>
// 2026-10-12 P1-H05-REGISTER-UI 挂号页。患者从建档 ?patientId= 带入，只选科室和医生。
// 2026-10-12 可联调：POST/GET /registers，字段 departmentId。
// 2026-10-12 P1-H05b-REGISTER-CHECKIN-UI 正常号、已用号显示报到。Mock 可模拟 409。
// 2026-10-13 P1-H05-MASTER-UI 科室和医生下拉只列启用项。
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ApiError } from '../api/errors.js'
import { useMock } from '../api/config.js'
import { cancelRegister, checkInRegister, completeRegister, createRegister, listRegisters } from '../api/registerApi.js'
import { createRegisterFeeCharge, listCharges, payExistingCharge, refundCharge } from '../api/chargeApi.js'
import { fetchPatient } from '../api/patientApi.js'
import { sexLabel, todayInShanghai } from '../api/patientContract.js'
import { session } from '../auth/session.js'
import { fetchDepartments, fetchDoctors } from '../master/api.js'
import BusinessEmptyState from '../components/BusinessEmptyState.vue'
import RegisterPatientPicker from '../components/RegisterPatientPicker.vue'
import { outpatientContext, recordAction, setCharges, setPatient, setRegister } from '../workbench/outpatientContext.js'
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
const router = useRouter()
const today = todayInShanghai()
const visitDate = ref(todayInShanghai())
const patientRoute = { name: 'patient-search' }
const patientPickerOpen = ref(false)
const patientSwitching = ref(false)
const pendingDepartmentId = ref('')

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
const appointmentMode = ref(false)
const appointmentStart = ref('')
const appointmentEnd = ref('')
const appointmentNote = ref('')
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

const visitDateContext = computed(() => {
  if (visitDate.value > today) return { label: '未来日期 · 预约接待', tone: 'appointment' }
  if (visitDate.value < today) return { label: '历史日期 · 补录接待', tone: 'history' }
  return { label: '今天 · 现场接待', tone: 'today' }
})

const patientId = computed(() => readQuery(route.query.patientId) || String(outpatientContext.patient?.patientId || outpatientContext.patient?.patientNo || '').trim())
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
const selectedDateTickets = computed(() => tickets.value.filter((item) => item.visitDate === visitDate.value))
const registrationTickets = computed(() => selectedDateTickets.value.filter((item) => item.visitType !== 'APPOINTMENT'))
const appointmentTickets = computed(() => selectedDateTickets.value.filter((item) => item.visitType === 'APPOINTMENT'))
const todayActions = computed(() => (Array.isArray(outpatientContext.history) ? outpatientContext.history : []).slice(-5).reverse())
const ticketGroups = computed(() => {
  const groups = []
  if (registrationTickets.value.length) {
    groups.push({
      key: 'register',
      title: visitDate.value === today ? '当日挂号' : `${visitDate.value} 挂号`,
      testId: 'register-list',
      items: registrationTickets.value,
    })
  }
  if (appointmentTickets.value.length) {
    groups.push({ key: 'appointment', title: '预约记录', testId: 'appointment-list', items: appointmentTickets.value })
  }
  return groups
})
function readQuery(raw) {
  const value = Array.isArray(raw) ? raw[0] : raw
  return typeof value === 'string' ? value.trim() : ''
}

function appointmentDateTime(value) {
  const raw = typeof value === 'string' ? value.trim() : ''
  if (!raw) return ''
  return raw.length === 16 ? `${raw}:00+08:00` : `${raw}+08:00`
}

function preserveAppointmentDate(nextDate) {
  for (const key of ['appointmentStart', 'appointmentEnd']) {
    const current = key === 'appointmentStart' ? appointmentStart.value : appointmentEnd.value
    if (!current) continue
    const time = current.includes('T') ? current.slice(11, 16) : ''
    if (time) {
      if (key === 'appointmentStart') appointmentStart.value = `${nextDate}T${time}`
      else appointmentEnd.value = `${nextDate}T${time}`
    }
  }
}

watch(visitDate, (next, previous) => {
  if (next > today && !appointmentMode.value) appointmentMode.value = true
  if (next <= today && previous > today && appointmentMode.value) appointmentMode.value = false
  if (appointmentMode.value) preserveAppointmentDate(next)
  if (patient.value && !patientSwitching.value) refreshTickets()
})

watch(appointmentMode, (enabled) => {
  if (!enabled) {
    appointmentStart.value = ''
    appointmentEnd.value = ''
    appointmentNote.value = ''
    return
  }
  if (!appointmentStart.value) appointmentStart.value = `${visitDate.value}T09:00`
  if (!appointmentEnd.value) appointmentEnd.value = `${visitDate.value}T09:30`
})

function dateAfterMonths(months) {
  const [year, month, day] = today.split('-').map(Number)
  const targetMonth = new Date(Date.UTC(year, month - 1 + months, 1))
  const targetYear = targetMonth.getUTCFullYear()
  const targetMonthIndex = targetMonth.getUTCMonth()
  const lastDay = new Date(Date.UTC(targetYear, targetMonthIndex + 1, 0)).getUTCDate()
  const next = new Date(Date.UTC(targetYear, targetMonthIndex, Math.min(day, lastDay)))
  return [next.getUTCFullYear(), String(next.getUTCMonth() + 1).padStart(2, '0'), String(next.getUTCDate()).padStart(2, '0')].join('-')
}

function setVisitDate(value) {
  visitDate.value = value
}

function chooseDepartment(item) {
  pendingDepartmentId.value = item?.departmentId || ''
  patientPickerOpen.value = true
}

async function onSelectPatient(selected) {
  if (!selected || selected.status !== 'ACTIVE' || !selected.selectable) return
  patientSwitching.value = true
  departmentId.value = pendingDepartmentId.value || ''
  pendingDepartmentId.value = ''
  doctorId.value = ''
  registerFee.value = 0
  appointmentMode.value = false
  appointmentStart.value = ''
  appointmentEnd.value = ''
  appointmentNote.value = ''
  visitDate.value = today
  patientPickerOpen.value = false
  showNotice('', false)
  try {
    await router.replace({ name: 'outpatient-module', params: { module: 'register' }, query: { patientId: selected.patientId } })
  } finally {
    patientSwitching.value = false
  }
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
  const requestedDate = visitDate.value
  ticketsLoading.value = true
  listNote.value = ''
  try {
    const listed = await listRegisters(session.accessToken, {
      patientId: id,
      visitDate: requestedDate,
    })
    if (seq !== ticketsLoadSeq || id !== patientId.value) return
    tickets.value = listed.items.filter((item) => item.visitDate === requestedDate)
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
      visitDate: visitDate.value,
      visitType: appointmentMode.value ? 'APPOINTMENT' : 'WALK_IN',
      appointmentStart: appointmentMode.value ? appointmentDateTime(appointmentStart.value) : '',
      appointmentEnd: appointmentMode.value ? appointmentDateTime(appointmentEnd.value) : '',
      appointmentNote: appointmentMode.value ? appointmentNote.value : '',
      registerFee: mockMode ? 0 : Math.max(0, Number(registerFee.value) || 0),
    })
    if (currentSeq !== patientLoadSeq || patient.value?.patientId !== currentPatientId) return
    if (created.visitDate === visitDate.value) {
      tickets.value = [created, ...tickets.value.filter((item) => item.registerId !== created.registerId)]
    }
    setRegister(created)
    recordAction(appointmentMode.value ? '预约已创建' : '挂号已创建', created.registerNo || created.registerId || '')
    showNotice(appointmentMode.value ? '预约已保存，记录显示在下方预约列表。' : '已保存挂号，状态为正常号。', false)
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
    showNotice(row.visitType === 'APPOINTMENT' ? '预约已报到' : '已报到', false)
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

    <div v-if="!patientId" class="register-empty-workspace" data-testid="register-empty">
      <section class="register-info-strip" data-testid="register-info-strip" aria-label="读卡入口">
        <button class="button button-ghost button-compact" type="button" data-testid="read-id-card" @click="patientPickerOpen = true">读取身份证</button>
        <button class="button button-ghost button-compact" type="button" data-testid="read-medical-card" @click="patientPickerOpen = true">读取医保卡</button>
      </section>

      <div class="register-empty-grid">
        <section class="patient-panel register-empty-card register-empty-patient" data-testid="register-patient-panel" aria-labelledby="register-empty-patient-title">
          <div class="panel-head">
            <div><span class="clinical-kicker">患者档案</span><h2 id="register-empty-patient-title">患者基本信息</h2></div>
          </div>
          <div class="register-empty-patient-state">
            <div class="register-empty-visual" aria-hidden="true">
              <span class="register-empty-orbit"></span><span class="register-empty-orbit register-empty-orbit-inner"></span>
              <span class="register-empty-symbol"><span class="register-calendar-icon">▣</span></span>
              <span class="register-empty-dot register-empty-dot-one"></span><span class="register-empty-dot register-empty-dot-two"></span>
            </div>
            <div class="register-empty-copy">
              <span class="register-empty-eyebrow">下一步</span>
              <h3>请先选择患者</h3>
              <p>选择患者后，可在本页完成科室、医生和就诊日期安排。</p>
              <button class="button button-compact" type="button" data-testid="register-choose-patient" @click="patientPickerOpen = true">选择患者</button>
            </div>
          </div>
        </section>

        <section class="patient-panel register-empty-card register-empty-department" data-testid="register-department-panel" aria-labelledby="register-empty-department-title">
          <div class="panel-head">
            <div><span class="clinical-kicker">门诊安排</span><h2 id="register-empty-department-title">科室基本信息</h2></div>
            <span class="clinical-date-tag">待选择</span>
          </div>
          <div class="register-department-grid" data-testid="register-department-list">
            <p v-if="masterLoading" class="register-empty-note">正在读取科室…</p>
            <p v-else-if="masterError" class="register-empty-note" role="alert">{{ masterError }}</p>
            <p v-else-if="!departments.length" class="register-empty-note">暂无启用科室。</p>
            <button v-for="item in departments" :key="item.departmentId" class="register-department-tile" type="button" @click="chooseDepartment(item)">
              <strong>{{ item.name }}</strong><span>可挂号</span>
            </button>
          </div>
        </section>

        <section class="patient-panel register-empty-card register-empty-today" data-testid="register-today-panel" aria-labelledby="register-empty-today-title">
          <div class="panel-head">
            <div><span class="clinical-kicker">今日工作</span><h2 id="register-empty-today-title" class="sr-only">今日工作</h2></div>
            <span class="clinical-count">{{ todayActions.length }}</span>
          </div>
          <ul v-if="todayActions.length" class="register-today-list">
            <li v-for="item in todayActions" :key="`${item.at}-${item.label}`"><strong>{{ item.label }}</strong><span>{{ item.detail || '已更新' }}</span></li>
          </ul>
          <p v-else class="register-empty-note">今天还没有挂号操作记录。</p>
        </section>
      </div>

      <section class="patient-panel register-empty-fee" data-testid="register-fee-panel" aria-labelledby="register-empty-fee-title">
        <div class="panel-head"><div><span class="clinical-kicker">本次费用</span><h2 id="register-empty-fee-title">费用信息</h2></div></div>
        <p class="register-empty-note">选择患者并填写挂号信息后显示费用。</p>
        <div class="panel-actions"><button class="button button-compact" type="button" data-testid="register-empty-submit" @click="patientPickerOpen = true">确认挂号</button></div>
      </section>

      <section class="patient-panel register-empty-history" data-testid="register-history-panel" aria-labelledby="register-empty-history-title">
        <div class="panel-head"><div><span class="clinical-kicker">历史信息</span><h2 id="register-empty-history-title" class="sr-only">历史信息</h2></div></div>
        <p class="register-empty-note">选择患者后读取历史挂号记录；诊断信息在诊断环节维护。</p>
      </section>
    </div>

    <p v-else-if="loading" class="shell-status">正在读取患者…</p>

    <BusinessEmptyState v-else-if="notFound" data-testid="register-missing" title="未找到患者" description="这个患者标识没有对应档案，请在患者查询中重新选择。" icon="patient"><RouterLink class="button button-ghost button-compact" :to="patientRoute">患者查询</RouterLink></BusinessEmptyState>

    <div v-else-if="loadError" class="banner clinical-read-error" role="alert"><p>{{ loadError }}</p><button class="button button-ghost button-compact" type="button" @click="load">重试患者资料</button></div>

    <div v-else-if="patient" class="clinical-register-workspace" data-testid="register-layout">
      <div class="clinical-register-layout">
        <section class="patient-panel clinical-patient-profile" data-testid="register-patient-panel" aria-labelledby="register-patient">
          <div class="panel-head">
            <span class="clinical-kicker">就诊患者</span>
            <button class="button button-ghost button-compact" type="button" data-testid="register-change-patient" @click="patientPickerOpen = true">更换患者</button>
          </div>
          <div class="clinical-profile-identity"><span class="clinical-profile-avatar" aria-hidden="true">{{ patient.name?.slice(0, 1) || '患' }}</span><div><h2 id="register-patient">{{ patient.name }}</h2><p>{{ patient.patientNo || '患者号未提供' }}</p></div><span class="clinical-badge">档案已载入</span></div>
          <dl class="summary-grid register-patient-details">
            <div><dt>性别</dt><dd>{{ sexLabel(patient.sex) }}</dd></div>
            <div><dt>出生日期</dt><dd>{{ patient.birthday || '—' }}</dd></div>
            <div><dt>患者号</dt><dd>{{ patient.patientNo || '—' }}</dd></div>
            <div><dt>身份证</dt><dd>{{ maskIdNumber(patient.idNumber) }}</dd></div>
          </dl>
          <p class="clinical-profile-note"><span aria-hidden="true">◇</span> 主索引不能在挂号页修改。</p>
        </section>

        <form class="patient-panel clinical-register-form" data-testid="register-form" novalidate @submit.prevent="onSave">
          <div class="panel-head">
            <div><span class="clinical-kicker">本次就诊</span><h2>挂号信息</h2></div>
            <span class="clinical-date-tag">{{ visitDate }}</span>
          </div>
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
            <label class="field"><span>号别</span><input value="普通号" disabled /></label>
            <label class="field">
              <span>就诊日期</span>
              <input v-model="visitDate" type="date" :disabled="saving" />
              <small class="clinical-date-hint" :data-tone="visitDateContext.tone">{{ visitDateContext.label }}</small>
            </label>
            <label class="field">
              <span>接待方式</span>
              <select v-model="appointmentMode" :disabled="saving">
                <option :value="false">现场号</option>
                <option :value="true">预约号</option>
              </select>
            </label>
            <label v-if="!mockMode" class="field">
              <span>挂号费（元）</span>
              <input v-model.number="registerFee" type="number" min="0" max="9999999999999999.99" step="0.01" inputmode="decimal" />
            </label>
            <div class="clinical-date-presets field-wide" aria-label="就诊日期快捷选择">
              <span>快速选择</span>
              <button type="button" class="clinical-date-preset" :class="{ active: visitDate === today }" :disabled="saving" @click="setVisitDate(today)">今天</button>
              <button type="button" class="clinical-date-preset" :class="{ active: visitDate === dateAfterMonths(1) }" :disabled="saving" @click="setVisitDate(dateAfterMonths(1))">1 个月后</button>
              <button type="button" class="clinical-date-preset" :class="{ active: visitDate === dateAfterMonths(2) }" :disabled="saving" @click="setVisitDate(dateAfterMonths(2))">2 个月后</button>
            </div>
            <template v-if="appointmentMode">
              <label class="field"><span>预约开始</span><input v-model="appointmentStart" type="datetime-local" :disabled="saving" /></label>
              <label class="field"><span>预约结束</span><input v-model="appointmentEnd" type="datetime-local" :disabled="saving" /></label>
              <label class="field field-wide"><span>预约备注</span><input v-model="appointmentNote" maxlength="256" placeholder="可填写患者到访说明或特殊安排" :disabled="saving" /></label>
            </template>
          </div>
          <p v-if="mockMode" class="field-hint">演示模式不收取挂号费。</p>
          <p v-if="notice" class="banner" :class="{ 'banner-ok': !noticeError }" :role="noticeError ? 'alert' : 'status'" data-testid="register-notice">{{ notice }}</p>
          <div class="clinical-form-footer">
            <p>保存后为正常号，可在下方报到</p>
            <div class="panel-actions"><button class="button button-compact" type="submit" data-testid="save-register" :disabled="!canSave">{{ saving ? '正在保存…' : '确认挂号' }}</button></div>
          </div>
        </form>
      </div>

      <section class="patient-panel clinical-ticket-panel" data-testid="register-history-panel" aria-label="所选日期挂号记录">
        <div class="panel-head">
          <div><span class="clinical-kicker">历史信息</span><h2>挂号记录</h2></div>
          <span class="clinical-count">{{ selectedDateTickets.length }} 条</span>
        </div>
        <div class="clinical-ticket-content">
        <label v-if="mockMode" class="checkin-demo">
          <input v-model="simulateCheckInConflict" type="checkbox" data-testid="mock-checkin-conflict" />
          <span>模拟报到冲突（下次报到返回 409，号状态不变）</span>
        </label>
        <p v-if="ticketsLoading" class="shell-status" role="status">正在读取 {{ visitDate }} 挂号记录…</p>
        <div v-else-if="listNote" class="banner clinical-read-error" role="alert"><p>{{ listNote }}。请先重试以核对所选日期记录。</p><button class="button button-ghost button-compact" type="button" @click="refreshTickets">重试挂号列表</button></div>
        <BusinessEmptyState v-else-if="ticketsReady && selectedDateTickets.length === 0" :title="visitDate === today ? '当日暂无挂号。' : '所选日期暂无记录。'" description="为该患者保存挂号或预约后，记录与可用操作会显示在这里。" icon="register" />
        <section v-for="group in ticketGroups" :key="group.key" class="clinical-ticket-group" :aria-label="group.title">
          <h3>{{ group.title }} <span class="clinical-count">{{ group.items.length }}</span></h3>
          <div class="result-wrap clinical-table-scroll" tabindex="0" role="region" :aria-label="`${group.title}列表，可左右滚动`">
          <table class="register-table" :data-testid="group.testId">
            <caption class="sr-only">{{ group.title }}</caption>
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
              <tr v-for="row in group.items" :key="row.registerId">
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
                    {{ checkingInId === row.registerId ? '报到中…' : row.visitType === 'APPOINTMENT' ? '预约报到' : '报到' }}
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
      </div>
    <RegisterPatientPicker :open="patientPickerOpen" :access-token="session.accessToken" @close="patientPickerOpen = false" @select="onSelectPatient" />
  </section>
</template>

<style scoped>
.register-empty-workspace { display:grid; gap:14px; min-width:0; }
.register-info-strip { display:flex; align-items:center; gap:8px; min-height:52px; padding:10px 14px; border:1px solid #cfe3e8; border-radius:8px; background:#fff; }
.register-empty-grid { display:grid; grid-template-columns:minmax(0,1.25fr) minmax(0,1.25fr) minmax(220px,.8fr); align-items:start; gap:14px; min-width:0; }
.register-empty-card { min-width:0; }
.register-empty-patient, .register-empty-department { min-height:448px; }
.register-empty-patient .panel-head, .register-empty-department .panel-head, .register-empty-today .panel-head, .register-empty-fee .panel-head, .register-empty-history .panel-head { align-items:flex-start; }
.register-empty-patient .panel-head h2, .register-empty-department .panel-head h2, .register-empty-fee .panel-head h2 { color:#173747; font-size:15px; }
.register-empty-patient .panel-head h2::before, .register-empty-department .panel-head h2::before, .register-empty-fee .panel-head h2::before { display:none; }
.register-empty-patient-state { display:flex; align-items:center; justify-content:center; gap:28px; min-height:320px; margin-top:12px; padding:24px 20px; border-radius:14px; background:linear-gradient(135deg,#f9fcfd,#fff); }
.register-empty-visual { position:relative; display:grid; flex:none; width:150px; height:150px; place-items:center; }
.register-empty-orbit { position:absolute; inset:2px; border:1px dashed #c9ded1; border-radius:50%; }
.register-empty-orbit-inner { inset:18px; border-style:solid; border-color:#e1ece4; }
.register-empty-symbol { display:grid; width:74px; height:86px; place-items:center; border:1px solid #c6e2d1; border-radius:22px; background:linear-gradient(135deg,#e6f4eb,#f9fcfa); color:#4a9c82; box-shadow:0 12px 20px #24523a0a; transform:rotate(-7deg); }
.register-calendar-icon { font-size:30px; line-height:1; }
.register-empty-dot { position:absolute; width:8px; height:8px; border-radius:50%; background:#84bca0; box-shadow:0 0 0 5px #edf5ef; }
.register-empty-dot-one { top:21px; left:28px; }.register-empty-dot-two { right:20px; bottom:19px; width:5px; height:5px; background:#adc9b7; }
.register-empty-copy { max-width:150px; min-width:0; }
.register-empty-eyebrow { display:block; margin-bottom:8px; color:#88a392; font-size:10px; }
.register-empty-copy h3 { color:#38554a; font-size:20px; font-weight:650; line-height:1.45; }
.register-empty-copy p { margin-top:10px; color:#7c9085; font-size:13px; line-height:1.8; }
.register-empty-copy .button { margin-top:20px; }
.register-department-grid { display:grid; grid-template-columns:repeat(2,minmax(0,1fr)); align-content:start; gap:10px; min-height:0; height:335px; padding:18px 0 0; }
.register-department-tile { display:flex; align-items:center; justify-content:space-between; gap:8px; min-width:0; min-height:48px; padding:0 12px; border:1px solid #d5e8ed; border-radius:6px; color:#165a6b; background:#f8fcfd; text-align:left; }
.register-department-tile:hover { border-color:#8fc6d0; background:#f1fafb; }
.register-department-tile strong { overflow:hidden; font-size:13px; text-overflow:ellipsis; white-space:nowrap; }
.register-department-tile span { flex:none; padding:4px 7px; border-radius:4px; color:#247d70; background:#e7f5ee; font-size:10px; white-space:nowrap; }
.register-empty-note { margin:0; color:#7c929c; font-size:12px; line-height:1.8; }
.register-department-grid .register-empty-note { grid-column:1 / -1; padding:18px 12px; }
.register-empty-today { min-height:128px; }
.register-today-list { display:grid; gap:8px; margin:16px 0 0; padding:0; list-style:none; }
.register-today-list li { display:grid; gap:2px; min-width:0; }
.register-today-list strong, .register-today-list span { overflow:hidden; text-overflow:ellipsis; white-space:nowrap; }
.register-today-list strong { color:#365665; font-size:11px; }.register-today-list span { color:#8aa0a8; font-size:10px; }
.register-empty-fee { min-height:190px; padding-bottom:20px !important; }
.register-empty-fee .panel-actions { justify-content:flex-end; margin-top:26px; }
.register-empty-history { min-height:126px; height:126px; }
.register-empty-history .register-empty-note { margin-top:12px; }
.clinical-register-layout { display: grid; grid-template-columns: minmax(250px, .8fr) minmax(0, 1.2fr); align-items: start; gap: 14px; min-width: 0; }
.clinical-register-layout > *, .clinical-register-form { min-width: 0; }
.clinical-register-layout .clinical-patient-profile { gap: 9px; padding: 14px; }
.clinical-register-layout .panel-head { align-items: flex-start; }
.clinical-register-layout .panel-head h2 { color: #173747; font-size: 15px; }
.clinical-register-layout .panel-head h2::before { display: none; }
.clinical-register-layout .clinical-profile-identity { min-width: 0; padding-block: 4px 10px; }
.clinical-register-layout .clinical-profile-identity > div { min-width: 0; }
.clinical-register-layout .clinical-profile-identity h2 { overflow: hidden; color: #173747; font-size: 16px; font-weight: 650; text-overflow: ellipsis; white-space: nowrap; }
.clinical-register-layout .clinical-profile-identity p { overflow: hidden; color: #718995; font-size: 11px; text-overflow: ellipsis; white-space: nowrap; }
.clinical-register-layout .register-patient-details { grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 8px 12px; padding: 11px 12px; }
.clinical-register-layout .register-patient-details > div { min-width: 0; }
.clinical-register-layout .register-patient-details dd { overflow: hidden; color: #365f6b; font-size: 11px; text-overflow: ellipsis; white-space: nowrap; }
.clinical-register-workspace { gap: 10px; }
.clinical-register-form { display: grid; gap: 10px; padding: 14px !important; border: 1px solid #dcebe1 !important; background: #fff !important; box-shadow: 0 5px 18px rgba(29, 81, 96, .04) !important; }
.shell-main .register-page .clinical-register-form .field-grid { grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 9px 12px; }
.shell-main .register-page .clinical-register-form .field-grid .field-wide { grid-column: 1 / -1; }
.shell-main .register-page .clinical-register-form .field { gap: 5px; }
.shell-main .register-page .clinical-register-form :is(input:not([type='checkbox']):not([type='radio']), select) { height: 36px; }
.shell-main .register-page .clinical-register-form .clinical-form-footer { margin-top: 0; padding-top: 10px; }
.clinical-register-form .field-wide { grid-column: 1 / -1; }
.clinical-ticket-panel { display: grid; grid-template-rows: auto minmax(0, 1fr); gap: 8px; min-width: 0; max-height: 210px; margin-top: 0; padding: 10px !important; overflow: hidden; }
.clinical-ticket-content { min-height: 0; overflow: auto; overscroll-behavior: contain; }
.clinical-ticket-panel .clinical-table-scroll { max-height: none; overflow: visible; }
.clinical-ticket-panel .business-empty-state { min-height: 62px; padding: 8px; gap: 8px; border: 0; box-shadow: none; }
.clinical-ticket-panel .business-empty-visual { display: none; }
.clinical-ticket-panel .business-empty-copy h2 { font-size: 12px; }
.clinical-ticket-panel .business-empty-description { display: none; }
.clinical-ticket-panel .register-table thead th { position: sticky; top: 0; z-index: 1; background: #f4f8f5; }
@media (max-width: 1100px) {
  .clinical-register-layout { grid-template-columns: minmax(220px, .75fr) minmax(0, 1.25fr); }
}
@media (max-width: 820px) {
  .register-empty-grid { grid-template-columns:1fr; }
  .register-empty-patient, .register-empty-department { min-height:0; }
  .register-empty-today { min-height:0; }
  .register-empty-patient-state { min-height:260px; }
  .clinical-register-layout { grid-template-columns: 1fr; }
  .clinical-register-form .field-grid { grid-template-columns: 1fr; }
  .clinical-register-form .field-wide { grid-column: auto; }
}
@media (max-width: 560px) {
  .register-empty-patient-state { align-items:flex-start; flex-direction:column; gap:16px; }
  .register-empty-copy { max-width:none; }
  .register-department-grid { grid-template-columns:1fr; }
}
</style>
