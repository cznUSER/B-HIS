<script setup>
// 2026-10-12 P1-H05-PATIENT-UI 患者建档与查询。侧栏仍由 AppShell 提供。
import { computed, reactive, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { ApiError } from '../api/errors.js'
import { createPatient, fetchPatient, searchPatients } from '../api/patientApi.js'
import {
  FARE_OPTIONS,
  SEX_OPTIONS,
  fareLabel,
  sexLabel,
  todayInShanghai,
} from '../api/patientContract.js'
import { session } from '../auth/session.js'

const route = useRoute()
const today = todayInShanghai()

function blankForm() {
  return {
    name: '',
    sex: '',
    birthday: '',
    fareIdentity: '',
    idNumber: '',
    phone: '',
    address: '',
  }
}

const form = reactive(blankForm())
const searchForm = reactive({ name: '', idNumber: '', patientNo: '' })
const results = ref([])
const searched = ref(false)
const mode = ref('create')
const current = ref(null)
const justCreated = ref(false)
const saving = ref(false)
const querying = ref(false)
const opening = ref(false)
const errorMessage = ref('')
const errorCode = ref('')
const errorStatus = ref(0)
const existingPatientId = ref('')
const searchError = ref('')
const activeId = ref('')

const locked = computed(() => mode.value === 'view')
const registerTo = computed(() => {
  if (!current.value) return { name: 'outpatient-module', params: { module: 'register' } }
  return {
    name: 'outpatient-module',
    params: { module: 'register' },
    query: { patientId: current.value.patientId },
  }
})

function clearError() {
  errorMessage.value = ''
  errorCode.value = ''
  errorStatus.value = 0
  existingPatientId.value = ''
}

function applyPatient(patient) {
  current.value = patient
  activeId.value = patient.patientId
  form.name = patient.name || ''
  form.sex = patient.sex || ''
  form.birthday = patient.birthday || ''
  form.fareIdentity = patient.fareIdentity || ''
  form.idNumber = patient.idNumber || ''
  form.phone = patient.phone || ''
  form.address = patient.address || ''
  mode.value = 'view'
}

function resetCreate() {
  Object.assign(form, blankForm())
  current.value = null
  justCreated.value = false
  activeId.value = ''
  mode.value = 'create'
  clearError()
}

async function onCreate() {
  if (saving.value || locked.value) return
  clearError()
  justCreated.value = false
  saving.value = true
  try {
    const patient = await createPatient(session.accessToken, { ...form })
    applyPatient(patient)
    justCreated.value = true
  } catch (error) {
    if (error instanceof ApiError) {
      errorMessage.value = error.message
      errorCode.value = error.code
      errorStatus.value = error.status || 0
      const details = error.details
      if (details && typeof details.patientId === 'string') existingPatientId.value = details.patientId
      const duplicateId =
        error.code === 'DUPLICATE_PATIENT' || error.code === 'DUPLICATE_ID_NUMBER'
      if (!existingPatientId.value && duplicateId && form.idNumber.trim()) {
        try {
          const found = await searchPatients(session.accessToken, { idNumber: form.idNumber.trim() })
          if (found.items.length > 0) existingPatientId.value = found.items[0].patientId
        } catch {
          // 拒绝提示仍然保留，缺少主键时不提供打开链接。
        }
      }
    } else {
      errorMessage.value = '建档失败，请稍后重试'
    }
  } finally {
    saving.value = false
  }
}

async function onSearch() {
  if (querying.value) return
  querying.value = true
  searched.value = true
  searchError.value = ''
  try {
    const found = await searchPatients(session.accessToken, { ...searchForm })
    results.value = found.items
  } catch (error) {
    results.value = []
    searchError.value = error instanceof ApiError ? error.message : '查询失败，请稍后重试'
  } finally {
    querying.value = false
  }
}

function queryPatientId() {
  const raw = route.query.patientId
  const value = Array.isArray(raw) ? raw[0] : raw
  return typeof value === 'string' ? value.trim() : ''
}

async function openPatient(patientId) {
  if (!patientId || opening.value) return
  opening.value = true
  clearError()
  justCreated.value = false
  try {
    const patient = await fetchPatient(session.accessToken, patientId)
    applyPatient(patient)
  } catch (error) {
    errorMessage.value = error instanceof ApiError ? error.message : '未找到患者'
    errorCode.value = error instanceof ApiError ? error.code : ''
  } finally {
    opening.value = false
  }
}

watch(queryPatientId, (id) => {
  if (id) openPatient(id)
}, { immediate: true })
</script>

<template>
  <section class="patient-page" aria-labelledby="patient-title">
    <p class="crumb">
      <RouterLink :to="{ name: 'home' }">首页</RouterLink>
      <span aria-hidden="true"> / </span>
      <RouterLink :to="{ name: 'workbench' }">门诊业务</RouterLink>
      <span aria-hidden="true"> / </span>
      <span>患者建档</span>
    </p>

    <header class="dash-head">
      <h1 id="patient-title">患者建档</h1>
      <p class="dash-flag">填写档案并查询已有患者。保存后可去挂号。</p>
    </header>

    <div class="patient-layout">
      <form class="patient-panel" data-testid="patient-form" novalidate @submit.prevent="onCreate">
        <div class="panel-head">
          <h2>{{ locked ? '已有患者' : '新建档案' }}</h2>
          <button v-if="locked" class="text-btn" type="button" @click="resetCreate">继续建档</button>
        </div>

        <div class="field-grid">
          <label class="field">
            <span>姓名 <abbr class="req" title="必填">*</abbr></span>
            <input v-model="form.name" name="patient-name" autocomplete="off" :disabled="locked" required />
          </label>
          <label class="field">
            <span>性别 <abbr class="req" title="必填">*</abbr></span>
            <select v-model="form.sex" name="patient-sex" :disabled="locked" required>
              <option value="">请选择</option>
              <option v-for="item in SEX_OPTIONS" :key="item.value" :value="item.value">{{ item.label }}</option>
            </select>
          </label>
          <label class="field">
            <span>出生日期 <abbr class="req" title="必填">*</abbr></span>
            <input
              v-model="form.birthday"
              name="patient-birthday"
              type="date"
              min="1900-01-02"
              :max="today"
              :disabled="locked"
              required
            />
          </label>
          <label class="field">
            <span>费别 <abbr class="req" title="必填">*</abbr></span>
            <select v-model="form.fareIdentity" name="patient-fare" :disabled="locked" required>
              <option value="">请选择</option>
              <option v-for="item in FARE_OPTIONS" :key="item.value" :value="item.value">{{ item.label }}</option>
            </select>
          </label>
          <label class="field">
            <span>身份证号</span>
            <input v-model="form.idNumber" name="patient-id" autocomplete="off" :disabled="locked" />
          </label>
          <label class="field">
            <span>手机号</span>
            <input v-model="form.phone" name="patient-phone" autocomplete="off" inputmode="tel" :disabled="locked" />
          </label>
          <label class="field field-wide">
            <span>地址</span>
            <textarea v-model="form.address" name="patient-address" rows="2" :disabled="locked"></textarea>
            <span class="field-hint">选填，可留空。</span>
          </label>
        </div>

        <p v-if="errorMessage" class="banner" role="alert" data-testid="patient-error">
          <span class="banner-body">
            <span>{{ errorMessage }}</span>
            <span v-if="errorCode" class="banner-meta">
              <span class="meta-chip">{{ errorStatus === 409 ? '409 ' : '' }}{{ errorCode }}</span>
            </span>
            <button
              v-if="existingPatientId"
              class="text-btn text-btn-danger"
              type="button"
              :disabled="opening"
              @click="openPatient(existingPatientId)"
            >
              打开已有患者
            </button>
          </span>
        </p>

        <div v-if="current" class="ok-banner" role="status" data-testid="patient-success">
          <p v-if="justCreated" class="ok-title">建档成功</p>
          <p>
            患者号 <strong>{{ current.patientNo || '—' }}</strong>
          </p>
          <p>
            患者标识 <strong class="mono">{{ current.patientId }}</strong>
          </p>
          <p v-if="current.hospPkNo">
            院内主键 <strong>{{ current.hospPkNo }}</strong>
          </p>
          <RouterLink class="button button-compact" :to="registerTo">去挂号</RouterLink>
        </div>

        <div class="panel-actions">
          <button v-if="!locked" class="button button-compact" type="submit" :disabled="saving">
            {{ saving ? '正在保存…' : '保存建档' }}
          </button>
          <button v-if="!locked" class="button button-ghost button-compact" type="button" :disabled="saving" @click="resetCreate">
            清空
          </button>
        </div>
      </form>

      <form class="patient-panel" data-testid="patient-search" @submit.prevent="onSearch">
        <div class="panel-head">
          <h2>查询</h2>
        </div>
        <div class="field-grid">
          <label class="field">
            <span>姓名</span>
            <input v-model="searchForm.name" name="search-name" autocomplete="off" placeholder="支持姓名中的连续文字" />
          </label>
          <label class="field">
            <span>身份证号</span>
            <input v-model="searchForm.idNumber" name="search-id" autocomplete="off" placeholder="完整证件号" />
          </label>
          <label class="field field-wide">
            <span>患者号</span>
            <input v-model="searchForm.patientNo" name="search-no" autocomplete="off" placeholder="完整患者号" />
          </label>
        </div>
        <div class="panel-actions">
          <button class="button button-compact" type="submit" :disabled="querying">
            {{ querying ? '正在查询…' : '查询' }}
          </button>
        </div>

        <div class="result-wrap">
          <p v-if="searchError" class="banner" role="alert">{{ searchError }}</p>
          <p v-else-if="!searched" class="result-empty">输入姓名、身份证号或患者号后查询。</p>
          <p v-else-if="results.length === 0" class="result-empty">没有匹配的患者。</p>
          <table v-else class="patient-table">
            <caption class="sr-only">
              患者查询结果
            </caption>
            <thead>
              <tr>
                <th>患者号</th>
                <th>姓名</th>
                <th>性别</th>
                <th>出生日期</th>
                <th>身份证号</th>
                <th>费别</th>
                <th>手机号</th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="item in results"
                :key="item.patientId"
                :class="{ 'is-active': item.patientId === activeId }"
                tabindex="0"
                @click="openPatient(item.patientId)"
                @keydown.enter.prevent="openPatient(item.patientId)"
              >
                <td>{{ item.patientNo || '—' }}</td>
                <td>{{ item.name }}</td>
                <td>{{ sexLabel(item.sex) }}</td>
                <td>{{ item.birthday || '—' }}</td>
                <td>{{ item.idNumber || '—' }}</td>
                <td>{{ fareLabel(item.fareIdentity) }}</td>
                <td>{{ item.phone || '—' }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </form>
    </div>
  </section>
</template>
