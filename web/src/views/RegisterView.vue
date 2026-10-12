<script setup>
// 2026-10-12 P1-H05-REGISTER-UI 挂号页。患者从建档 ?patientId= 带入，只选科室和医生。
// 2026-10-12 可联调：POST/GET /registers，字段 departmentId。
// 2026-10-12 P1-H05b-REGISTER-CHECKIN-UI 正常号、已用号显示报到。Mock 可模拟 409。
// 2026-10-13 P1-H05-MASTER-UI 科室和医生下拉只列启用项。
import { computed, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { ApiError } from '../api/errors.js'
import { useMock } from '../api/config.js'
import { cancelRegister, checkInRegister, createRegister, listRegisters } from '../api/registerApi.js'
import { fetchPatient } from '../api/patientApi.js'
import { sexLabel, todayInShanghai } from '../api/patientContract.js'
import { session } from '../auth/session.js'
import {
  REGISTER_SORT,
  canCancel,
  canCheckIn,
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
const patientRoute = { name: 'outpatient-module', params: { module: 'patient' } }

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
const simulateCheckInConflict = ref(false)
const mockMode = useMock()
const notice = ref('')
const noticeError = ref(false)

const patientId = computed(() => readQuery(route.query.patientId))
const departments = computed(() => selectableDepartments())
const doctors = computed(() => doctorsForDept(departmentId.value))
const patientEditTo = computed(() => ({
  name: 'outpatient-module',
  params: { module: 'patient' },
  query: patientId.value ? { patientId: patientId.value } : {},
}))

function readQuery(raw) {
  const value = Array.isArray(raw) ? raw[0] : raw
  return typeof value === 'string' ? value.trim() : ''
}

watch(departmentId, () => {
  if (!doctors.value.some((item) => item.doctorId === doctorId.value)) doctorId.value = ''
})

watch(patientId, () => {
  load()
}, { immediate: true })

async function load() {
  patient.value = null
  tickets.value = []
  notFound.value = false
  loadError.value = ''
  listNote.value = ''
  notice.value = ''
  departmentId.value = ''
  doctorId.value = ''
  if (!patientId.value) return
  loading.value = true
  try {
    patient.value = await fetchPatient(session.accessToken, patientId.value)
    await refreshTickets()
  } catch (error) {
    if (error instanceof ApiError && error.status === 404) {
      notFound.value = true
      return
    }
    loadError.value = error instanceof ApiError ? error.message : '无法读取患者'
  } finally {
    loading.value = false
  }
}

async function refreshTickets() {
  try {
    const listed = await listRegisters(session.accessToken, {
      patientId: patientId.value,
      visitDate,
    })
    tickets.value = listed.items
    listNote.value = ''
  } catch (error) {
    tickets.value = []
    listNote.value = error instanceof ApiError ? error.message : '暂时无法读取挂号列表'
  }
}

function showNotice(message, isError) {
  notice.value = message
  noticeError.value = isError
}

async function onSave() {
  if (saving.value || !patient.value) return
  showNotice('', false)
  if (!departmentId.value) {
    showNotice('请选择科室', true)
    return
  }
  if (!doctorId.value || !doctors.value.some((item) => item.doctorId === doctorId.value)) {
    showNotice('请选择医生', true)
    return
  }
  saving.value = true
  try {
    const created = await createRegister(session.accessToken, {
      patientId: patient.value.patientId,
      departmentId: departmentId.value,
      doctorId: doctorId.value,
      registerSort: REGISTER_SORT.registerSort,
      visitDate,
      registerFee: 0,
    })
    tickets.value = [created, ...tickets.value.filter((item) => item.registerId !== created.registerId)]
    showNotice('已保存，状态为正常号', false)
  } catch (error) {
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
  showNotice('', false)
  try {
    const updated = await cancelRegister(session.accessToken, row.registerId)
    tickets.value = tickets.value.map((item) => (item.registerId === updated.registerId ? updated : item))
    showNotice('已退号', false)
  } catch (error) {
    showNotice(error instanceof ApiError ? error.message : '退号失败，请稍后重试', true)
  } finally {
    cancellingId.value = ''
  }
}

async function onCheckIn(row) {
  if (!row || checkingInId.value || cancellingId.value) return
  if (!canCheckIn(row.status)) {
    showNotice('当前状态不允许该操作', true)
    return
  }
  checkingInId.value = row.registerId
  showNotice('', false)
  try {
    const updated = await checkInRegister(session.accessToken, row.registerId, {
      simulateConflict: mockMode && simulateCheckInConflict.value,
    })
    tickets.value = tickets.value.map((item) => (item.registerId === updated.registerId ? updated : item))
    showNotice('已报到', false)
  } catch (error) {
    showNotice(error instanceof ApiError ? error.message : '报到失败，请稍后重试', true)
  } finally {
    checkingInId.value = ''
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

    <header class="dash-head">
      <h1 id="register-title">挂号</h1>
      <p class="dash-flag">选择科室和医生。保存后为正常号。正常号和已用号可以报到，仅正常号可以退号。</p>
    </header>

    <div v-if="!patientId" class="empty-panel" data-testid="register-empty">
      <h2 class="empty-lead">请先选择或建档患者</h2>
      <p class="muted">挂号页不能新建主索引。请先到建档页选择或保存患者。</p>
      <RouterLink class="button button-ghost empty-back" :to="patientRoute">去建档</RouterLink>
    </div>

    <p v-else-if="loading" class="shell-status">正在读取患者…</p>

    <div v-else-if="notFound" class="empty-panel" data-testid="register-missing">
      <h2 class="empty-lead">未找到患者</h2>
      <p class="muted">这个患者标识没有对应档案。</p>
      <RouterLink class="button button-ghost empty-back" :to="patientRoute">去建档</RouterLink>
    </div>

    <p v-else-if="loadError" class="banner" role="alert">{{ loadError }}</p>

    <template v-else-if="patient">
      <section class="patient-panel" aria-labelledby="register-patient">
        <div class="panel-head">
          <h2 id="register-patient">患者</h2>
          <RouterLink class="text-btn" :to="patientEditTo">回建档页</RouterLink>
        </div>
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
        <p class="field-hint">主索引不能在挂号页修改。</p>
      </section>

      <form class="patient-panel" data-testid="register-form" novalidate @submit.prevent="onSave">
        <div class="panel-head">
          <h2>挂号信息</h2>
        </div>
        <div class="field-grid">
          <label class="field">
            <span>科室 <abbr class="req" title="必填">*</abbr></span>
            <select v-model="departmentId" data-testid="dept-select">
              <option value="">请选择科室</option>
              <option v-for="item in departments" :key="item.departmentId" :value="item.departmentId">{{ item.name }}</option>
            </select>
          </label>
          <label class="field">
            <span>医生 <abbr class="req" title="必填">*</abbr></span>
            <select v-model="doctorId" data-testid="doctor-select" :disabled="!departmentId">
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

        <div class="panel-actions">
          <button class="button button-compact" type="submit" data-testid="save-register" :disabled="saving">
            {{ saving ? '正在保存…' : '保存挂号' }}
          </button>
        </div>
      </form>

      <section class="patient-panel" aria-labelledby="register-list-title">
        <div class="panel-head">
          <h2 id="register-list-title">当日挂号</h2>
        </div>
        <label v-if="mockMode" class="checkin-demo">
          <input v-model="simulateCheckInConflict" type="checkbox" data-testid="mock-checkin-conflict" />
          <span>模拟报到冲突（下次报到返回 409，号状态不变）</span>
        </label>
        <p v-if="listNote" class="field-hint">{{ listNote }}</p>
        <p v-if="tickets.length === 0" class="result-empty">当日暂无挂号。</p>
        <div v-else class="result-wrap">
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
                <th>状态</th>
                <th>操作</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="row in tickets" :key="row.registerId">
                <td class="mono">{{ row.registerNo || row.registerId }}</td>
                <td>{{ deptName(row.departmentId) }}</td>
                <td>{{ doctorName(row.doctorId) }}</td>
                <td>{{ registerSortName(row.registerSort) }}</td>
                <td>
                  <span class="ticket-status" :data-status="row.status">{{ statusLabel(row.status) }}</span>
                </td>
                <td class="ticket-actions">
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
                    class="button button-ghost button-compact"
                    type="button"
                    :data-testid="`cancel-${row.status}`"
                    :disabled="!canCancel(row.status) || Boolean(cancellingId) || Boolean(checkingInId)"
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
    </template>
  </section>
</template>
