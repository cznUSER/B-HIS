<script setup>
import { createSubmitFeedback } from '../components/feedback/useSubmitFeedback.js'
import SubmitStatusBanner from '../components/feedback/SubmitStatusBanner.vue'
// 2026-10-12 P1-H05-REGISTER-UI 挂号页。患者从建档 ?patientId= 带入，只选科室和医生。
// 2026-10-12 可联调：POST/GET /registers，字段 departmentId。
// 2026-10-12 P1-H05b-REGISTER-CHECKIN-UI 正常号、已用号显示报到。Mock 可模拟 409。
// 2026-10-13 P1-H05-MASTER-UI 科室和医生下拉只列启用项。
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ApiError } from '../api/errors.js'
import { useMock } from '../api/config.js'
import { cancelRegister, checkInRegister, completeRegister, createRegister, listRegisters, fetchRegisterSlotBoard } from '../api/registerApi.js'
import { createRegisterFeeCharge, listCharges, payExistingCharge, refundCharge } from '../api/chargeApi.js'
import { fetchPatient } from '../api/patientApi.js'
import { fareLabel, sexLabel, todayInShanghai } from '../api/patientContract.js'
import { session } from '../auth/session.js'
import { fetchDepartments, fetchDoctors } from '../master/api.js'
import RegisterPatientBar from '../components/register/RegisterPatientBar.vue'
import RegisterDeptGrid from '../components/register/RegisterDeptGrid.vue'
import RegisterDoctorCards from '../components/register/RegisterDoctorCards.vue'
import RegisterSlotGrid from '../components/register/RegisterSlotGrid.vue'
import RegisterConfirmRail from '../components/register/RegisterConfirmRail.vue'
import RegisterPatientHistory from '../components/register/RegisterPatientHistory.vue'
import { canConfirmSelection, historySince, patientHistory, slotTime } from '../register/slotCardModel.js'
import './register-slot-card.css'
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
  statusLabel,
} from '../register/master.js'

const route = useRoute()
const router = useRouter()
const today = todayInShanghai()
const visitDate = ref(todayInShanghai())
const patientPickerOpen = ref(false)
const patientSwitching = ref(false)
const pendingDepartmentId = ref('')
const pendingDoctorId = ref('')

const patient = ref(null)
const tickets = ref([])
const todayTickets = ref([])
const loading = ref(false)
const notFound = ref(false)
const loadError = ref('')
const listNote = ref('')
const departmentId = pendingDepartmentId
const doctorId = pendingDoctorId
const saving = ref(false)
const cancellingId = ref('')
const checkingInId = ref('')
const completingId = ref('')
const simulateCheckInConflict = ref(false)
const mockMode = useMock()
const notice = ref('')
const noticeError = ref(false)
const submitFeedback = createSubmitFeedback()
let registerAttempt = null
const registerFee = ref(0)
const appointmentMode = ref(false)
const appointmentStart = ref('')
const appointmentEnd = ref('')
const appointmentNote = ref('')
const paymentPreference = ref('CASH')
const selectedSlot = ref(null)
const slotBoard = ref({ departments: [], doctors: [] })
const boardLoading = ref(false)
const boardReady = ref(false)
const boardError = ref('')
let boardLoadSeq = 0
const selectedDepartment = computed(() => slotBoard.value.departments.find(d => d.departmentId === departmentId.value))
const selectedDoctor = computed(() => slotBoard.value.doctors.find(d => d.doctorId === doctorId.value))
const boardDoctors = computed(() => slotBoard.value.doctors.filter(d => d.departmentId === departmentId.value))
const selectedSlotLabel = computed(() => selectedSlot.value ? `${slotTime(selectedSlot.value.start)}–${slotTime(selectedSlot.value.end)}` : selectedDoctor.value?.scheduleConfigured === false && !appointmentMode.value ? '现场号 · 未配置时段' : '')
const historyRows = computed(() => patientHistory(tickets.value, patientId.value, today))
const confirmReason = computed(() => !patient.value ? '请先选择患者' : boardLoading.value ? '正在刷新号源…' : boardError.value ? '号源读取失败，请重试' : !departmentId.value ? '请选择科室' : !doctorId.value ? '请选择医生' : selectedDoctor.value?.scheduleConfigured === false && appointmentMode.value ? '无排班医生仅支持现场号' : !selectedSlotLabel.value ? '请选择可用号源时段' : '确认后生成挂号单；正费用需另行收费')
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
const todayLoading = ref(false)
const todayReady = ref(false)
const todayError = ref('')
let patientLoadSeq = 0
let ticketsLoadSeq = 0
let todayLoadSeq = 0
let todayLoadPromise = null

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
const pendingDoctors = computed(() => {
  if (!pendingDepartmentId.value) return []
  if (mockMode) return doctorsForDept(pendingDepartmentId.value)
  return liveDoctors.value.filter((item) => item.departmentId === pendingDepartmentId.value && item.status === 'ACTIVE')
})
const registerWriteBusy = computed(() => saving.value || Boolean(cancellingId.value) || Boolean(checkingInId.value) || Boolean(completingId.value) || Boolean(feeWorkingId.value))
const canSave = computed(() => Boolean(patient.value) && masterReady.value && !masterLoading.value && ticketsReady.value && !ticketsLoading.value && boardReady.value && !boardLoading.value && !registerWriteBusy.value && submitFeedback.state.status !== 'submitting' && canConfirmSelection({ patientId: patient.value?.patientId, departmentId: departmentId.value, doctor: selectedDoctor.value, slot: selectedSlot.value, appointmentMode: appointmentMode.value }))
const selectedDateTickets = computed(() => tickets.value.filter((item) => item.visitDate === visitDate.value))
const registrationTickets = computed(() => selectedDateTickets.value.filter((item) => item.visitType !== 'APPOINTMENT'))
const appointmentTickets = computed(() => selectedDateTickets.value.filter((item) => item.visitType === 'APPOINTMENT'))
const todayWorkRows = computed(() => [...todayTickets.value]
  .sort((left, right) => String(right.registeredAt || '').localeCompare(String(left.registeredAt || '')))
  .map((item) => ({
    id: item.registerId,
    patientName: item.patientName || '—',
    registerFee: `￥${Number(item.registerFee || 0).toFixed(2)}`,
    registeredTime: formatRegisteredTime(item.registeredAt),
  })))
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
  if (/([+-]\d{2}:\d{2}|Z)$/.test(value || '')) return value
  const raw = typeof value === 'string' ? value.trim() : ''
  if (!raw) return ''
  return raw.length === 16 ? `${raw}:00+08:00` : `${raw}+08:00`
}

function formatRegisteredTime(value) {
  const date = new Date(typeof value === 'string' ? value : '')
  if (!Number.isFinite(date.getTime())) return '—'
  return new Intl.DateTimeFormat('zh-CN', {
    timeZone: 'Asia/Shanghai',
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
  }).format(date)
}

function clearSlot() {
  selectedSlot.value = null
  appointmentStart.value = ''
  appointmentEnd.value = ''
}
function selectDepartment(id) {
  if (saving.value || !slotBoard.value.departments.some(d => d.departmentId === id)) return
  departmentId.value = id
  doctorId.value = ''
  clearSlot()
}
function selectDoctor(id) {
  const doctor = boardDoctors.value.find(d => d.doctorId === id)
  if (saving.value || !doctor || (doctor.scheduleConfigured && doctor.remain === 0)) return
  doctorId.value = id
  clearSlot()
}
function selectSlot(slot) {
  if (saving.value || !slot || slot.remain <= 0 || !selectedDoctor.value?.slots.some(s => s.start === slot.start && s.remain > 0)) return
  selectedSlot.value = slot
  appointmentStart.value = slot.start
  appointmentEnd.value = slot.end
}
function readCard(kind) {
  showNotice(`${kind}读卡设备尚未接入，请使用“选择患者”查询真实档案。`, false)
}
watch(visitDate, (next, previous) => {
  if (next > today) appointmentMode.value = true
  else if (previous > today) appointmentMode.value = false
  doctorId.value = ''
  clearSlot()
  loadSlotBoard()
  if (patient.value && !patientSwitching.value) refreshTickets()
})
watch(appointmentMode, () => clearSlot())

async function loadSlotBoard() {
  const seq = ++boardLoadSeq
  const date = visitDate.value
  boardLoading.value = true
  boardReady.value = false
  boardError.value = ''
  try {
    const board = await fetchRegisterSlotBoard(session.accessToken, { visitDate: date })
    if (seq !== boardLoadSeq || date !== visitDate.value) return
    if (board.visitDate !== date) throw new Error('号源日期与查询日期不一致')
    slotBoard.value = board
    boardReady.value = true
    const doctor = board.doctors.find(d => d.doctorId === doctorId.value)
    if (!board.departments.some(d => d.departmentId === departmentId.value)) departmentId.value = ''
    if (!doctor || doctor.departmentId !== departmentId.value || (doctor.scheduleConfigured && doctor.remain === 0)) { doctorId.value = ''; clearSlot() }
    if (selectedSlot.value) {
      const fresh = doctor?.slots.find(s => s.start === selectedSlot.value.start && s.end === selectedSlot.value.end && s.remain > 0)
      if (fresh) selectedSlot.value = fresh
      else clearSlot()
    }
  } catch (error) {
    if (seq !== boardLoadSeq) return
    slotBoard.value = { departments: [], doctors: [] }
    clearSlot()
    boardError.value = error instanceof ApiError ? error.message : '暂时无法读取号源'
  } finally { if (seq === boardLoadSeq) boardLoading.value = false }
}

async function onSelectPatient(selected) {
  if (!selected || selected.status !== 'ACTIVE' || !selected.selectable) return
  patientSwitching.value = true
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

watch(departmentId, () => { doctorId.value = ''; clearSlot() }, { flush: 'sync' })
watch(doctorId, () => clearSlot(), { flush: 'sync' })

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
      ...item,
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
onMounted(loadTodayTickets)
onMounted(loadSlotBoard)

async function loadTodayTickets(force = false) {
  if (todayReady.value && !force) return true
  if (todayLoadPromise && !force) return todayLoadPromise
  const seq = ++todayLoadSeq
  todayLoading.value = true
  todayError.value = ''
  const pending = (async () => {
    try {
      const listed = await listRegisters(session.accessToken, { visitDate: today })
      if (seq !== todayLoadSeq) return false
      todayTickets.value = listed.items.filter((item) => item.visitDate === today)
      todayReady.value = true
      return true
    } catch (error) {
      if (seq !== todayLoadSeq) return false
      todayTickets.value = []
      todayReady.value = false
      todayError.value = error instanceof ApiError ? error.message : '暂时无法读取今日挂号'
      return false
    } finally {
      if (seq === todayLoadSeq) todayLoading.value = false
    }
  })()
  todayLoadPromise = pending
  try {
    return await pending
  } finally {
    if (todayLoadPromise === pending) todayLoadPromise = null
  }
}

async function load() {
  const seq = ++patientLoadSeq
  ++ticketsLoadSeq
  const id = patientId.value
  departmentId.value = ''
  doctorId.value = ''
  clearSlot()
  registerFee.value = 0
  paymentPreference.value = 'CASH'
  appointmentNote.value = ''
  patient.value = null
  tickets.value = []
  feeCharges.value = {}
  ticketsReady.value = false
  ticketsLoading.value = false
  notFound.value = false
  loadError.value = ''
  listNote.value = ''
  notice.value = ''
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
    let offset = 0
    const collected = []
    // Include subsequent appointments while retaining every record in the previous 90 days.
    for (;;) {
      const listed = await listRegisters(session.accessToken, { patientId: id, visitDateFrom: historySince(today), offset, limit: 100 })
      if (seq !== ticketsLoadSeq || id !== patientId.value) return
      collected.push(...listed.items)
      if (!listed.hasMore) break
      if (!Number.isInteger(listed.nextOffset) || listed.nextOffset <= offset) throw new Error('历史分页响应无效，请重新读取')
      offset = listed.nextOffset
    }
    const nextTickets = patientHistory([...new Map(collected.map(row => [row.registerId, row])).values()], id, today)
    if (seq !== ticketsLoadSeq || id !== patientId.value) return
    tickets.value = nextTickets
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
  const seq = patientLoadSeq
  const listed = await listCharges(session.accessToken, { registerId: row.registerId, todayOnly: false })
  const found = listed.items.find((item) => item.registerId === row.registerId && item.items.some((line) => line.itemNo === 'REGISTER-FEE'))
  if (found && seq === patientLoadSeq) feeCharges.value = { ...feeCharges.value, [row.registerId]: found }
  return found || null
}

async function onCreateFee(row) {
  if (!row || registerWriteBusy.value) return
  const seq = patientLoadSeq
  feeWorkingId.value = row.registerId
  submitFeedback.beginAttempt('fee-create')
  showNotice('', false)
  try {
    const charge = await createRegisterFeeCharge(session.accessToken, row.registerId)
    if (seq !== patientLoadSeq) return
    feeCharges.value = { ...feeCharges.value, [row.registerId]: charge }
    syncRegisterContext()
    submitFeedback.succeed({ registerId: row.registerId, registerNo: row.registerNo || '' })
  } catch (error) {
    if (seq !== patientLoadSeq) return
    if (error instanceof ApiError && error.code === 'REGISTER_FEE_ALREADY_CHARGED') {
      try { await ensureFeeCharge(row) } catch { /* keep the server error below */ }
    }
    submitFeedback.fail(error instanceof ApiError ? error.message : '挂号费收费单生成失败', error?.code)
  } finally {
    feeWorkingId.value = ''
  }
}

async function onPayFee(row) {
  if (!row || registerWriteBusy.value) return
  const seq = patientLoadSeq
  feeWorkingId.value = row.registerId
  submitFeedback.beginAttempt('fee-pay')
  showNotice('', false)
  try {
    const charge = await ensureFeeCharge(row)
    if (seq !== patientLoadSeq) return
    if (!charge) throw new Error('请先生成挂号费收费单')
    const result = await payExistingCharge(session.accessToken, charge.chargeId, charge.squareFee, crypto.randomUUID())
    if (seq !== patientLoadSeq) return
    feeCharges.value = { ...feeCharges.value, [row.registerId]: { ...charge, ...result.charge, paid: true, payType: 'CASH' } }
    syncRegisterContext()
    recordAction('挂号费已收讫', result.charge?.squareNo || charge.chargeId || '')
    row.registerCharged = true
    submitFeedback.succeed({ registerId: row.registerId, registerNo: row.registerNo || '' })
  } catch (error) {
    if (seq !== patientLoadSeq) return
    submitFeedback.fail(error instanceof ApiError ? error.message : '挂号费收费失败', error?.code)
  } finally {
    feeWorkingId.value = ''
  }
}

async function onRefundFee(row) {
  if (!row || registerWriteBusy.value) return
  const seq = patientLoadSeq
  feeWorkingId.value = row.registerId
  submitFeedback.beginAttempt('fee-refund')
  showNotice('', false)
  try {
    const charge = await ensureFeeCharge(row)
    if (seq !== patientLoadSeq) return
    if (!charge || !charge.paid) throw new Error('当前没有可退的挂号费')
    const result = await refundCharge(session.accessToken, charge.chargeId, {
      payType: 'CASH',
      reason: '挂号费全额退费',
      idempotencyKey: crypto.randomUUID(),
      items: charge.items.map((item) => ({ chargeItemId: item.chargeItemId, refundQty: item.refundableQty || item.qty })),
    })
    if (seq !== patientLoadSeq) return
    if (result.fullRefund) {
      const next = { ...feeCharges.value }
      delete next[row.registerId]
      feeCharges.value = next
      syncRegisterContext()
      row.registerCharged = false
    }
    submitFeedback.succeed({ registerId: row.registerId, registerNo: row.registerNo || '' })
  } catch (error) {
    if (seq !== patientLoadSeq) return
    submitFeedback.fail(error instanceof ApiError ? error.message : '挂号费退费失败', error?.code)
  } finally {
    feeWorkingId.value = ''
  }
}

function showNotice(message, isError) {
  noticeError.value = false
  notice.value = message
  noticeError.value = isError
}

async function onSave() {
  if (!canSave.value) return
  showNotice('', false)
  if (!departments.value.some((item) => item.departmentId === departmentId.value)) {
    showNotice('\u8bf7\u9009\u62e9\u79d1\u5ba4', true)
    return
  }
  if (!doctorId.value || !doctors.value.some((item) => item.doctorId === doctorId.value)) {
    showNotice('\u8bf7\u9009\u62e9\u533b\u751f', true)
    return
  }
  saving.value = true
  const currentPatientId = patient.value.patientId
  const currentSeq = patientLoadSeq
  const request = {
    patientId: currentPatientId,
    departmentId: departmentId.value,
    doctorId: doctorId.value,
    registerSort: REGISTER_SORT.registerSort,
    visitDate: visitDate.value,
    visitType: appointmentMode.value ? 'APPOINTMENT' : 'WALK_IN',
    appointmentStart: selectedSlot.value ? selectedSlot.value.start : '',
    appointmentEnd: selectedSlot.value ? selectedSlot.value.end : '',
    appointmentNote: appointmentMode.value ? appointmentNote.value : '',
    registerFee: mockMode ? 0 : Math.max(0, Number(registerFee.value) || 0),
  }
  const fingerprint = JSON.stringify(request)
  if (!registerAttempt || registerAttempt.fingerprint !== fingerprint) registerAttempt = { fingerprint, key: crypto.randomUUID() }
  submitFeedback.beginAttempt('register')
  try {
    const created = await createRegister(session.accessToken, request, registerAttempt.key)
    if (currentSeq !== patientLoadSeq || patient.value?.patientId !== currentPatientId) return
    if (created.patientId === currentPatientId) {
      tickets.value = [created, ...tickets.value.filter((item) => item.registerId !== created.registerId)]
    }
    if (created.visitDate === today) {
      const todayRecord = { ...created, patientName: patient.value?.name || created.patientName || '' }
      todayTickets.value = [todayRecord, ...todayTickets.value.filter((item) => item.registerId !== created.registerId)]
      todayReady.value = true
    }
    setRegister(created)
    recordAction(appointmentMode.value ? '\u9884\u7ea6\u5df2\u521b\u5efa' : '\u6302\u53f7\u5df2\u521b\u5efa', created.registerNo || created.registerId || '')
    registerAttempt = null
    showNotice('', false)
    submitFeedback.succeed(created, '\u6302\u53f7\u6210\u529f\uff0c\u6d41\u6c34\u53f7 ' + (created.registerNo || created.registerId))
    clearSlot()
    await loadSlotBoard()
  } catch (error) {
    if (currentSeq !== patientLoadSeq || patient.value?.patientId !== currentPatientId) return
    const message = error instanceof ApiError ? error.message : '\u6302\u53f7\u5931\u8d25\uff0c\u8bf7\u7a0d\u540e\u91cd\u8bd5'
    submitFeedback.fail(message, error?.code)
    showNotice('', false)
    if (error.code === 'APPOINTMENT_CONFLICT' || error.code === 'SLOT_FULL') { clearSlot(); await loadSlotBoard() }
  } finally {
    saving.value = false
  }
}

async function onCancel(row) {
  if (!row || registerWriteBusy.value) return
  if (!canCancel(row.status)) {
    showNotice(cancelRejectMessage(row.status), true)
    return
  }
  cancellingId.value = row.registerId
  const currentSeq = patientLoadSeq
  submitFeedback.beginAttempt('cancel')
  showNotice('', false)
  try {
    const updated = await cancelRegister(session.accessToken, row.registerId)
    if (currentSeq !== patientLoadSeq) return
    tickets.value = tickets.value.map((item) => (item.registerId === updated.registerId ? updated : item))
    setRegister(updated)
    recordAction('挂号已退号', updated.registerNo || updated.registerId || '')
    showNotice('', false)
    submitFeedback.succeed(updated)
    await loadSlotBoard()
  } catch (error) {
    if (currentSeq !== patientLoadSeq) return
    submitFeedback.fail(error instanceof ApiError ? error.message : '退号失败，请稍后重试', error?.code)
  } finally {
    cancellingId.value = ''
  }
}

async function onCheckIn(row) {
  if (!row || registerWriteBusy.value) return
  if (!canCheckIn(row.status)) {
    showNotice('当前状态不允许该操作', true)
    return
  }
  checkingInId.value = row.registerId
  const currentSeq = patientLoadSeq
  submitFeedback.beginAttempt('check-in')
  showNotice('', false)
  try {
    const updated = await checkInRegister(session.accessToken, row.registerId, {
      simulateConflict: mockMode && simulateCheckInConflict.value,
    })
    if (currentSeq !== patientLoadSeq) return
    tickets.value = tickets.value.map((item) => (item.registerId === updated.registerId ? updated : item))
    setRegister(updated)
    recordAction('患者已报到', updated.registerNo || updated.registerId || '')
    showNotice('', false)
    submitFeedback.succeed(updated, (row.visitType === 'APPOINTMENT' ? '预约已报到' : '已报到') + '，流水号 ' + (updated.registerNo || updated.registerId || ''))
  } catch (error) {
    if (currentSeq !== patientLoadSeq) return
    submitFeedback.fail(error instanceof ApiError ? error.message : '报到失败，请稍后重试', error?.code)
  } finally {
    checkingInId.value = ''
  }
}

async function onComplete(row) {
  if (!row || registerWriteBusy.value) return
  if (!canComplete(row.status)) {
    showNotice(completeRejectMessage(row.status), true)
    return
  }
  completingId.value = row.registerId
  const currentSeq = patientLoadSeq
  submitFeedback.beginAttempt('complete')
  showNotice('', false)
  try {
    const updated = await completeRegister(session.accessToken, row.registerId)
    if (currentSeq !== patientLoadSeq) return
    tickets.value = tickets.value.map((item) => (item.registerId === updated.registerId ? updated : item))
    setRegister(updated)
    recordAction('就诊已完成', updated.registerNo || updated.registerId || '')
    showNotice('', false)
    submitFeedback.succeed(updated, '已完成就诊，流水号 ' + (updated.registerNo || updated.registerId || ''))
  } catch (error) {
    if (currentSeq !== patientLoadSeq) return
    submitFeedback.fail(error instanceof ApiError ? error.message : '完成就诊失败，请稍后重试', error?.code)
  } finally {
    completingId.value = ''
  }
}


</script>

<template>
  <section class="register-page register-slot-card-page" data-testid="register-page" aria-labelledby="register-title">
    <div class="register-top-chrome" data-testid="register-top-chrome"><div class="register-heading"><div><p class="crumb"><RouterLink :to="{ name: 'home' }">首页</RouterLink><span> / </span><RouterLink :to="{ name: 'workbench' }">门诊业务</RouterLink><span> / 挂号</span></p><h1 id="register-title">门诊挂号 <small>号源工作台</small></h1></div><div class="register-heading-note"><span>{{ today }} · 现场号</span><button class="text-btn" type="button" :disabled="boardLoading || saving" @click="loadSlotBoard">刷新号源</button></div></div></div>
    <form class="register-empty-workspace register-slot-workspace" data-testid="register-form" novalidate @submit.prevent="onSave">
      <RegisterPatientBar :patient="patient" :loading="loading" :disabled="registerWriteBusy" @choose="patientPickerOpen = true" @read-id="readCard('身份证')" @read-medical="readCard('医保卡')" />
      <div v-if="notFound || loadError" class="banner" role="alert">{{ notFound ? '未找到患者，请重新选择。' : loadError }}<button type="button" class="text-btn" @click="load">重试患者信息</button></div>
      <div class="register-flow-row" data-testid="register-layout">
        <div class="register-flow-main">
          <div v-if="masterError" class="banner" role="alert" data-testid="register-master-error">{{ masterError }}<button class="text-btn" type="button" @click="loadMaster">重新读取科室和医生</button></div>
          <div v-if="boardError" class="banner" role="alert" data-testid="register-slot-error">{{ boardError }}<button class="text-btn" type="button" @click="loadSlotBoard">重试号源</button></div>
          <p v-if="boardLoading" class="register-board-loading" role="status">正在读取当日号源…</p>
          <div class="register-card-panels" :aria-busy="boardLoading">
            <section class="register-flow-panel" data-testid="register-department-panel"><header><h2>开放科室</h2><span>当日放号余量</span></header><RegisterDeptGrid :departments="slotBoard.departments" :selected="departmentId" :today="visitDate === today" :disabled="saving || boardLoading || !masterReady" @select="selectDepartment" /></section>
            <section class="register-flow-panel"><header><h2>出诊医生</h2><span>{{ selectedDepartment?.departmentName || '先选左侧科室' }}</span></header><RegisterDoctorCards :doctors="boardDoctors" :selected="doctorId" :department-id="departmentId" :disabled="saving || boardLoading" :fee="registerFee" @select="selectDoctor" /></section>
            <section class="register-flow-panel"><header><h2>号源时段</h2><span>上午 / 下午</span></header><RegisterSlotGrid :doctor="selectedDoctor" :selected="selectedSlot?.start" :disabled="saving || boardLoading" :appointment-mode="appointmentMode" @select="selectSlot" /></section>
          </div>
          <label v-if="appointmentMode" class="register-appointment-note">预约备注<input v-model="appointmentNote" maxlength="256" placeholder="患者到访说明或特殊安排" :disabled="saving" /></label>
          <label v-if="mockMode" class="register-card-note"><input v-model="simulateCheckInConflict" type="checkbox" /> 模拟报到冲突（仅 Mock）</label>
          <details class="register-today-fold" data-testid="register-today-panel" open><summary>今日工作 · {{ todayWorkRows.length }} 笔 <span>挂号操作记录</span></summary><div class="register-history-scroll"><table class="register-today-table" data-testid="register-today-table"><thead><tr><th>患者姓名</th><th>挂号费用</th><th>挂号时间</th></tr></thead><tbody><tr v-for="item in todayWorkRows" :key="item.id"><td>{{ item.patientName }}</td><td>{{ item.registerFee }}</td><td>{{ item.registeredTime }}</td></tr><tr v-if="todayLoading"><td colspan="3">正在读取今日挂号…</td></tr><tr v-else-if="todayError"><td colspan="3">{{ todayError }} <button type="button" class="text-btn" @click="loadTodayTickets(true)">重试</button></td></tr><tr v-else-if="!todayWorkRows.length"><td colspan="3">今天还没有挂号操作记录。</td></tr></tbody></table></div></details>
        </div>
        <RegisterConfirmRail :patient="patient" :department="selectedDepartment" :doctor="selectedDoctor" :slot-label="selectedSlotLabel" v-model:fee="registerFee" v-model:payment="paymentPreference" :can-save="canSave" :saving="saving" :mock-mode="mockMode" :reason="confirmReason"><p v-if="notice && submitFeedback.state.status !== 'success' && submitFeedback.state.status !== 'error'" class="banner" :class="{ 'banner-ok': !noticeError }" :role="noticeError ? 'alert' : 'status'" data-testid="register-notice">{{ notice }}</p><SubmitStatusBanner :feedback="submitFeedback.state" /></RegisterConfirmRail>
      </div>
      <RegisterPatientHistory :patient="patient" :rows="historyRows" :loading="ticketsLoading" :error="listNote" :department-label="departmentLabel" :doctor-label="doctorLabel" :charges="feeCharges" @retry="refreshTickets"><template #actions="{ row }">
                      <button v-if="!mockMode && Number(row.registerFee) > 0 && !row.registerCharged && !feeChargeFor(row)" class="button button-compact" type="button" :disabled="registerWriteBusy" @click="onCreateFee(row)">生成收费单</button>
                      <button v-if="!mockMode && Number(row.registerFee) > 0 && !row.registerCharged && feeChargeFor(row)" class="button button-compact" type="button" :disabled="registerWriteBusy" @click="onPayFee(row)">{{ feeWorkingId === row.registerId ? '收费中…' : '现金收费' }}</button>
                      <button v-if="!mockMode && Number(row.registerFee) > 0 && row.registerCharged" class="button button-ghost button-compact" type="button" :disabled="registerWriteBusy" @click="onRefundFee(row)">{{ feeWorkingId === row.registerId ? '退费中…' : '挂号费退费' }}</button>
                      <button v-if="canCheckIn(row.status)" class="button button-compact" type="button" :data-testid="`checkin-${row.status}`" :disabled="registerWriteBusy" @click="onCheckIn(row)">{{ checkingInId === row.registerId ? '报到中…' : row.visitType === 'APPOINTMENT' ? '预约报到' : '报到' }}</button>
                      <button v-if="canComplete(row.status)" class="button button-compact" type="button" :data-testid="`complete-${row.status}`" :disabled="registerWriteBusy" @click="onComplete(row)">{{ completingId === row.registerId ? '完成中…' : '完成就诊' }}</button>
                      <button class="button button-ghost button-compact" type="button" :data-testid="`cancel-${row.status}`" :disabled="!canCancel(row.status) || registerWriteBusy" :title="canCancel(row.status) ? '退号' : cancelRejectMessage(row.status)" @click="onCancel(row)">{{ cancellingId === row.registerId ? '退号中…' : '退号' }}</button>
                      <p v-if="!canCancel(row.status)" class="ticket-reject">{{ cancelRejectMessage(row.status) }}</p>

      </template></RegisterPatientHistory>
    </form>
    <RegisterPatientPicker :open="patientPickerOpen" :access-token="session.accessToken" @close="patientPickerOpen = false" @select="onSelectPatient" />
  </section>
</template>
