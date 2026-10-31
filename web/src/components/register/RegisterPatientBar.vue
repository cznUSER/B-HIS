<script setup>
import { fareLabel, sexLabel, todayInShanghai } from '../../api/patientContract.js'
defineProps({ patient: Object, loading: Boolean, disabled: Boolean })
defineEmits(['choose', 'read-id', 'read-medical'])
function patientAge(birthday) {
  if (!/^\d{4}-\d{2}-\d{2}$/.test(birthday || '')) return '年龄 —'
  const today = todayInShanghai()
  const years = Number(today.slice(0, 4)) - Number(birthday.slice(0, 4)) - (today.slice(5) < birthday.slice(5) ? 1 : 0)
  return years >= 0 ? `${years}岁` : '年龄 —'
}
</script>
<template>
  <section class="register-patient-bar" data-testid="register-patient-panel" aria-label="患者身份与风险信息">
    <div class="register-patient-identity" data-testid="register-patient-identity">
      <span class="register-patient-avatar" aria-hidden="true">{{ patient?.name?.slice(0, 1) || '患' }}</span>
      <div><strong>{{ patient?.name || (loading ? '正在读取患者' : '请选择患者') }}</strong><span>{{ patient ? sexLabel(patient.sex) : '性别 —' }} · {{ patientAge(patient?.birthday) }} · {{ patient?.ethnicity || '民族 —' }}</span><small>患者号 {{ patient?.patientNo || '—' }}</small></div>
    </div>
    <dl class="register-patient-fields" data-testid="register-patient-fields">
      <div><dt>身份证号</dt><dd>{{ patient?.idNumber || '—' }}</dd></div>
      <div><dt>手机号</dt><dd>{{ patient?.phone || '—' }}</dd></div>
      <div><dt>费别 / 医保</dt><dd class="register-insurance">{{ patient ? fareLabel(patient.fareIdentity) : '—' }}</dd></div>
      <div><dt>过敏史</dt><dd class="register-allergy" data-testid="register-allergy">{{ patient?.allergyHistory || '未记录，请核实' }}</dd></div>
      <div><dt>就诊卡号</dt><dd>{{ patient?.cardNo || '—' }}</dd></div>
      <div><dt>出生日期</dt><dd>{{ patient?.birthday || '—' }}</dd></div>
      <div><dt>院内档案号</dt><dd>{{ patient?.hospPkNo || '—' }}</dd></div>
      <div class="register-patient-address"><dt>联系地址</dt><dd>{{ patient?.address || '—' }}</dd></div>
    </dl>
    <div class="register-patient-actions" data-testid="register-info-strip">
      <button class="button button-compact" type="button" data-testid="register-choose-patient" :disabled="disabled" @click="$emit('choose')">选择患者</button>
      <button class="button button-ghost button-compact" type="button" data-testid="read-id-card" :disabled="disabled" @click="$emit('read-id')">读取身份证</button>
      <button class="button button-ghost button-compact" type="button" data-testid="read-medical-card" :disabled="disabled" @click="$emit('read-medical')">读取医保卡</button>
    </div>
  </section>
</template>
