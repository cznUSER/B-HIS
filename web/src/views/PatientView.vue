<script setup>
// 2026-10-16 患者建档页仅负责创建档案；查询由患者查询页承接。
import { computed, reactive, ref } from 'vue'
import { ApiError } from '../api/errors.js'
import { createPatient } from '../api/patientApi.js'
import {
  FARE_OPTIONS,
  ID_TYPE_OPTIONS,
  SEX_OPTIONS,
  fareLabel,
  sexLabel,
  todayInShanghai,
} from '../api/patientContract.js'
import { session } from '../auth/session.js'
import BusinessPageHeader from '../components/BusinessPageHeader.vue'
import { recordAction, setPatient } from '../workbench/outpatientContext.js'
import { hasEffectivePermission } from '../workbench/catalog.js'
import './clinical-design.css'

const today = todayInShanghai()

function blankForm() {
  return {
    name: '',
    sex: '',
    birthday: '',
    fareIdentity: '',
    idType: 'ID_CARD',
    idNumber: '',
    phone: '',
    address: '',
    cardNo: '',
    memberCardNo: '',
    memberLevel: '',
    contractUnit: '',
    patientCertificateNo: '',
    birthPlace: '',
    ethnicity: '',
    nationality: '',
    maritalStatus: '',
    occupationCategory: '',
    occupation: '',
    education: '',
    workUnit: '',
    workUnitPhone: '',
    workUnitPostcode: '',
    householdAddress: '',
    householdPhone: '',
    householdPostcode: '',
    contactName: '',
    contactAddress: '',
    contactPhone: '',
    contactRelation: '',
  }
}

const form = reactive(blankForm())
const current = ref(null)
const justCreated = ref(false)
const saving = ref(false)
const errorMessage = ref('')
const errorCode = ref('')
const errorStatus = ref(0)
const existingPatientId = ref('')
const detailMode = ref(false)

const locked = computed(() => Boolean(current.value))
const canCreatePatient = computed(() => hasEffectivePermission(session.user, ['patient:create'], 'outpatient'))
const requiredCompleted = computed(() => [form.name.trim(), form.sex, form.birthday, form.fareIdentity].filter(Boolean).length)
const summarySource = computed(() => current.value || form)
const summaryName = computed(() => summarySource.value.name?.trim() || '待创建患者')
const summaryPatientNo = computed(() => current.value?.patientNo || (current.value ? '患者号未返回' : '保存后自动生成患者号'))
const summarySex = computed(() => sexLabel(summarySource.value.sex) || '未填写')
const summaryBirthday = computed(() => summarySource.value.birthday || '未填写')
const summaryAge = computed(() => ageFromBirthday(summarySource.value.birthday))
const summaryFare = computed(() => fareLabel(summarySource.value.fareIdentity) || '未填写')
const summaryIdNumber = computed(() => {
  const value = summarySource.value.idNumber?.trim() || ''
  return value.length > 7 ? `${value.slice(0, 3)}${'*'.repeat(Math.min(6, value.length - 7))}${value.slice(-4)}` : value || '未填写'
})
const summaryPhone = computed(() => {
  const value = summarySource.value.phone?.trim() || ''
  return value.length > 7 ? `${value.slice(0, 3)}****${value.slice(-4)}` : value || '未填写'
})
const profileProgress = computed(() => `${requiredCompleted.value * 25}%`)
const summaryStatus = computed(() => {
  if (locked.value) return '档案已创建'
  return requiredCompleted.value === 4 ? '必填资料已完整' : '等待完善资料'
})
const registerTo = computed(() => {
  if (!current.value) return { name: 'outpatient-module', params: { module: 'register' } }
  return {
    name: 'outpatient-module',
    params: { module: 'register' },
    query: { patientId: current.value.patientId },
  }
})
const duplicateTo = computed(() => ({
  name: 'patient-search',
  query: existingPatientId.value
    ? { patientId: existingPatientId.value }
    : form.idNumber.trim()
      ? { idNumber: form.idNumber.trim() }
      : {},
}))
const isDuplicate = computed(() => ['DUPLICATE_PATIENT', 'DUPLICATE_ID_NUMBER', 'DUPLICATE_CARD_NO'].includes(errorCode.value))

function ageFromBirthday(birthday) {
  if (!birthday) return ''
  const [year, month, day] = birthday.split('-').map(Number)
  const [todayYear, todayMonth, todayDay] = today.split('-').map(Number)
  if (!year || !month || !day) return ''
  return todayYear - year - (todayMonth < month || (todayMonth === month && todayDay < day) ? 1 : 0)
}

function clearError() {
  errorMessage.value = ''
  errorCode.value = ''
  errorStatus.value = 0
  existingPatientId.value = ''
}

function applyPatient(patient) {
  current.value = patient
  setPatient(patient)
  recordAction('患者档案已打开', patient.name || patient.patientNo || patient.patientId || '')
}

function resetCreate() {
  Object.assign(form, blankForm())
  current.value = null
  justCreated.value = false
  detailMode.value = false
  clearError()
}

async function onCreate() {
  if (saving.value || current.value) return
  if (!canCreatePatient.value) {
    clearError()
    errorMessage.value = '当前账号没有患者建档权限'
    errorCode.value = 'FORBIDDEN'
    errorStatus.value = 403
    return
  }
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
    } else {
      errorMessage.value = '建档失败，请稍后重试'
    }
  } finally {
    saving.value = false
  }
}

</script>

<template>
  <section class="patient-page" data-testid="patient-page" aria-labelledby="patient-title">
    <div class="patient-top-chrome" data-testid="patient-top-chrome">
    <p class="crumb">
      <RouterLink :to="{ name: 'home' }">首页</RouterLink>
      <span aria-hidden="true"> / </span>
      <span>档案管理</span>
      <span aria-hidden="true"> / </span>
      <span>患者建档</span>
    </p>

    <BusinessPageHeader title-id="patient-title" title="患者建档" eyebrow="PATIENT ARCHIVE" icon="patient" description="完善患者资料，创建准确的患者档案。" />
    </div>

    <div class="patient-layout patient-create-layout">
      <aside class="patient-panel clinical-patient-summary" data-testid="patient-summary" aria-labelledby="patient-summary-title">
        <div class="patient-summary-head">
          <div class="patient-summary-identity">
            <span class="clinical-profile-avatar" aria-hidden="true">{{ summaryName.slice(0, 1) }}</span>
            <div>
              <h2 id="patient-summary-title">{{ summaryName }}</h2>
              <p>{{ summaryPatientNo }}</p>
            </div>
          </div>
          <span class="clinical-badge" :class="{ 'patient-summary-badge-created': locked }">{{ locked ? '已创建' : '待保存' }}</span>
        </div>

        <dl class="patient-summary-grid" data-testid="patient-summary-grid">
          <div><dt>性别</dt><dd>{{ summarySex }}</dd></div>
          <div><dt>出生日期</dt><dd>{{ summaryBirthday }}</dd></div>
          <div><dt>费别</dt><dd>{{ summaryFare }}</dd></div>
          <div><dt>证件号码</dt><dd>{{ summaryIdNumber }}</dd></div>
          <div><dt>手机号</dt><dd>{{ summaryPhone }}</dd></div>
        </dl>

        <div class="patient-summary-progress-head"><span>必填资料</span><strong>{{ requiredCompleted }} / 4</strong></div>
        <div class="patient-summary-progress" role="progressbar" :aria-valuenow="requiredCompleted" aria-valuemin="0" aria-valuemax="4" aria-label="建档必填资料完成度">
          <span :style="{ width: profileProgress }"></span>
        </div>

        <div class="patient-summary-state" :class="{ ready: locked || requiredCompleted === 4 }">
          <span class="patient-summary-state-dot" aria-hidden="true"></span>
          <div>
            <strong>{{ summaryStatus }}</strong>
            <p>{{ locked ? '患者号已生成，可继续挂号。' : '左侧摘要会随资料录入实时更新。' }}</p>
          </div>
        </div>

        <RouterLink v-if="locked" class="button button-compact patient-summary-register" :to="registerTo">去挂号</RouterLink>
      </aside>

      <form class="patient-panel clinical-create-panel" data-testid="patient-form" novalidate @submit.prevent="onCreate">
        <div class="panel-head">
          <div><span class="clinical-kicker">{{ locked ? '创建档案' : '新建档案' }}</span><h2>{{ locked ? '档案已创建' : '新建档案' }}</h2></div>
          <div class="patient-head-actions">
            <div class="patient-mode-switch" role="group" aria-label="建档模式" data-testid="patient-mode-switch">
              <button class="patient-mode-button" :class="{ active: !detailMode }" type="button" :aria-pressed="!detailMode" @click="detailMode = false">简易建档</button>
              <button class="patient-mode-button" :class="{ active: detailMode }" type="button" :aria-pressed="detailMode" @click="detailMode = true">详细建档</button>
            </div>
            <button v-if="locked" class="text-btn" type="button" @click="resetCreate">继续建档</button>
            <span v-else class="clinical-count">必填 {{ requiredCompleted }} / 4</span>
          </div>
        </div>

        <div v-if="!locked" class="clinical-progress" role="progressbar" :aria-valuenow="requiredCompleted" aria-valuemin="0" aria-valuemax="4" aria-label="必填资料完成度"><span :style="{ width: `${requiredCompleted * 25}%` }"></span></div>

        <fieldset class="clinical-section" data-testid="patient-simple-fields">
          <legend><span class="clinical-section-number">01</span> 身份资料 <small>用于识别患者与就诊费别</small></legend>
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
            <span>证件类型</span>
            <select v-model="form.idType" name="patient-id-type" :disabled="locked">
              <option v-for="item in ID_TYPE_OPTIONS" :key="item.value" :value="item.value">{{ item.label }}</option>
            </select>
          </label>
          <label class="field">
            <span>证件号码</span>
            <input v-model="form.idNumber" name="patient-id" autocomplete="off" :disabled="locked" />
          </label>
          <label class="field">
            <span>手机号</span>
            <input v-model="form.phone" name="patient-phone" autocomplete="off" inputmode="tel" :disabled="locked" />
          </label>
          </div>
        </fieldset>
        <fieldset v-if="detailMode" class="clinical-section" data-testid="patient-detail-fields">
          <legend><span class="clinical-section-number">02</span> 详细资料 <small>患者档案补充信息</small></legend>
          <div class="field-grid">
            <label class="field">
              <span>卡号</span>
              <input v-model="form.cardNo" name="patient-card-no" autocomplete="off" :disabled="locked" />
            </label>
            <label class="field">
              <span>会员卡号</span>
              <input v-model="form.memberCardNo" name="patient-member-card-no" autocomplete="off" :disabled="locked" />
            </label>
            <label class="field">
              <span>会员等级</span>
              <input v-model="form.memberLevel" name="patient-member-level" :disabled="locked" />
            </label>
            <label class="field">
              <span>合同单位</span>
              <input v-model="form.contractUnit" name="patient-contract-unit" :disabled="locked" />
            </label>
            <label class="field">
              <span>病人证号</span>
              <input v-model="form.patientCertificateNo" name="patient-certificate-no" :disabled="locked" />
            </label>
            <label class="field">
              <span>出生地</span>
              <input v-model="form.birthPlace" name="patient-birth-place" :disabled="locked" />
            </label>
            <label class="field">
              <span>民族</span>
              <input v-model="form.ethnicity" name="patient-ethnicity" :disabled="locked" />
            </label>
            <label class="field">
              <span>国籍</span>
              <input v-model="form.nationality" name="patient-nationality" :disabled="locked" />
            </label>
            <label class="field">
              <span>婚姻状况</span>
              <input v-model="form.maritalStatus" name="patient-marital-status" :disabled="locked" />
            </label>
            <label class="field">
              <span>年龄</span>
              <input :value="summaryAge" name="patient-age" disabled />
            </label>
            <label class="field">
              <span>职业分类</span>
              <input v-model="form.occupationCategory" name="patient-occupation-category" :disabled="locked" />
            </label>
            <label class="field">
              <span>职业</span>
              <input v-model="form.occupation" name="patient-occupation" :disabled="locked" />
            </label>
            <label class="field">
              <span>学历</span>
              <input v-model="form.education" name="patient-education" :disabled="locked" />
            </label>
          </div>
        </fieldset>
        <fieldset v-if="detailMode" class="clinical-section" data-testid="patient-contact-fields">
          <legend><span class="clinical-section-number">03</span> 单位与联系资料 <small>工作单位、户籍及联系人</small></legend>
          <div class="field-grid">
            <label class="field">
              <span>工作单位</span>
              <input v-model="form.workUnit" name="patient-work-unit" :disabled="locked" />
            </label>
            <label class="field">
              <span>单位联系电话</span>
              <input v-model="form.workUnitPhone" name="patient-work-unit-phone" inputmode="tel" :disabled="locked" />
            </label>
            <label class="field">
              <span>单位邮政编码</span>
              <input v-model="form.workUnitPostcode" name="patient-work-unit-postcode" inputmode="numeric" :disabled="locked" />
            </label>
            <label class="field field-wide">
              <span>户口地址</span>
              <input v-model="form.householdAddress" name="patient-household-address" :disabled="locked" />
            </label>
            <label class="field">
              <span>户籍联系电话</span>
              <input v-model="form.householdPhone" name="patient-household-phone" inputmode="tel" :disabled="locked" />
            </label>
            <label class="field">
              <span>户籍邮政编码</span>
              <input v-model="form.householdPostcode" name="patient-household-postcode" inputmode="numeric" :disabled="locked" />
            </label>
            <label class="field">
              <span>联系人</span>
              <input v-model="form.contactName" name="patient-contact-name" :disabled="locked" />
            </label>
            <label class="field">
              <span>关系</span>
              <input v-model="form.contactRelation" name="patient-contact-relation" :disabled="locked" />
            </label>
            <label class="field">
              <span>联系人电话</span>
              <input v-model="form.contactPhone" name="patient-contact-phone" inputmode="tel" :disabled="locked" />
            </label>
            <label class="field field-wide">
              <span>联系人地址</span>
              <input v-model="form.contactAddress" name="patient-contact-address" :disabled="locked" />
            </label>
            <label class="field field-wide">
              <span>地址</span>
              <textarea v-model="form.address" name="patient-address" rows="2" :disabled="locked"></textarea>
            </label>
          </div>
        </fieldset>

        <p v-if="errorMessage" class="banner" role="alert" data-testid="patient-error">
          <span class="banner-body">
            <span>{{ errorMessage }}</span>
            <span v-if="errorCode" class="banner-meta">
              <span class="meta-chip">{{ errorStatus === 409 ? '409 ' : '' }}{{ errorCode }}</span>
            </span>
            <RouterLink v-if="isDuplicate" class="text-btn text-btn-danger" :to="duplicateTo">在患者查询中查看已有档案</RouterLink>
          </span>
        </p>

        <p v-if="!canCreatePatient" class="banner" role="alert" data-testid="patient-create-forbidden">
          当前账号没有患者建档权限
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
        </div>

        <div v-if="!locked && canCreatePatient" class="clinical-form-footer" data-testid="patient-form-footer">
          <p><span class="clinical-foot-dot" aria-hidden="true"></span> 完善必填资料后保存，档案将生成患者号</p>
          <div class="panel-actions" data-testid="patient-panel-actions">
          <button class="button button-compact" type="submit" :disabled="saving">
            {{ saving ? '正在保存…' : '保存建档' }}
          </button>
          <button class="button button-ghost button-compact" type="button" :disabled="saving" @click="resetCreate">
            清空
          </button>
          </div>
        </div>
      </form>
    </div>
  </section>
</template>

<style scoped>
.patient-page .patient-layout.patient-create-layout {
  grid-template-columns: minmax(250px, .38fr) minmax(0, 1fr);
  max-width: var(--patient-shell-w, 868px);
  align-items: start;
  gap: 22px;
}

.clinical-patient-summary {
  align-self: start;
  gap: 20px;
  padding: 24px;
}

.patient-summary-head,
.patient-summary-identity,
.patient-head-actions,
.patient-summary-progress-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.patient-summary-identity { justify-content: flex-start; min-width: 0; }
.patient-summary-identity > div { min-width: 0; }
.patient-summary-identity h2 { overflow: hidden; max-width: 15em; color: #294e3e; font-size: 18px; font-weight: 650; text-overflow: ellipsis; white-space: nowrap; }
.patient-summary-identity p { overflow: hidden; margin-top: 4px; color: #8fa497; font-size: 11px; text-overflow: ellipsis; white-space: nowrap; }
.patient-summary-badge-created { color: #0d6b4a !important; background: #e8f6ef !important; }
.patient-summary-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 15px 12px; margin: 0; padding: 18px 0; border-block: 1px solid #e1ece3; }
.patient-summary-grid div { min-width: 0; }
.patient-summary-grid dt { color: #8da093; font-size: 10px; }
.patient-summary-grid dd { overflow: hidden; margin: 4px 0 0; color: #50705b; font-size: 12px; font-weight: 600; text-overflow: ellipsis; white-space: nowrap; }
.patient-summary-progress-head { color: #71877a; font-size: 11px; }
.patient-summary-progress-head strong { color: #496b58; font-size: 11px; font-variant-numeric: tabular-nums; }
.patient-summary-progress { height: 5px; margin-top: -14px; overflow: hidden; border-radius: 999px; background: #edf3ef; }
.patient-summary-progress span { display: block; height: 100%; border-radius: inherit; background: linear-gradient(90deg, #add6bf, #39a082); transition: width .25s ease; }
.patient-summary-state { display: flex; align-items: flex-start; gap: 9px; padding-top: 15px; border-top: 1px solid #edf3ee; }
.patient-summary-state-dot { width: 8px; height: 8px; flex: none; margin-top: 4px; border-radius: 50%; background: #c6a15d; }
.patient-summary-state.ready .patient-summary-state-dot { background: #2f8b69; }
.patient-summary-state strong { color: #496658; font-size: 12px; font-weight: 650; }
.patient-summary-state p { margin-top: 4px; color: #91a398; font-size: 10px; line-height: 1.6; }
.patient-summary-register { width: 100%; justify-content: center; }
.patient-head-actions { flex-wrap: wrap; justify-content: flex-end; }
.patient-mode-switch { display: inline-flex; flex: none; gap: 2px; padding: 3px; border: 1px solid #dce8e1; border-radius: 8px; background: #f5f9f6; }
.patient-mode-button { min-height: 32px; padding: 0 10px; border: 0; border-radius: 5px; background: transparent; color: #71877a; cursor: pointer; font: inherit; font-size: 11px; white-space: nowrap; }
.patient-mode-button.active { background: #fff; box-shadow: 0 1px 4px rgba(31, 66, 50, .12); color: #315f49; font-weight: 700; }
.patient-mode-button:focus-visible { outline: 2px solid #72ad91; outline-offset: 2px; }

@media (max-width: 1040px) {
  .patient-page .patient-layout.patient-create-layout { grid-template-columns: minmax(0, 1fr); }
  .clinical-patient-summary { position: static; }
}

@media (max-width: 640px) {
  .clinical-patient-summary { gap: 16px; padding: 18px; }
  .patient-summary-grid { gap: 13px 10px; }
  .patient-head-actions { justify-content: flex-end; }
  .patient-mode-switch { order: 3; width: 100%; }
  .patient-mode-button { flex: 1; }
}
</style>
