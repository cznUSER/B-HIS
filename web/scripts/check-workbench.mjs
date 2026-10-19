// 2026-10-12 P1-H05-HOME 工作台入口顺序与占位规则
import { existsSync, readFileSync } from 'node:fs'
import {
  OUTPATIENT_MODULES,
  blockReason,
  canEnter,
  decorateModules,
  hasEffectivePermission,
  modulePath,
  roleText,
} from '../src/workbench/catalog.js'
import {
  CHARGE_METRICS,
  PERIOD_OPTIONS,
  TODO_CARDS,
  VISIT_METRICS,
} from '../src/workbench/dashboard.js'
import { assertDashboardSummary, createDashboardRequestGate, DASHBOARD_PERIODS } from '../src/api/dashboardApi.js'
import { NAV, resolveNavChild } from '../src/workbench/nav.js'

function assert(condition, message) {
  if (!condition) throw new Error(message)
}

const names = ['患者建档', '挂号', '诊断', '开单', '收费', '发药', '退药/退费']
assert(OUTPATIENT_MODULES.map((item) => item.name).join(',') === names.join(','), 'module order')

const keys = ['patient', 'register', 'diagnosis', 'order', 'charge', 'dispense', 'refund']
assert(OUTPATIENT_MODULES.map((item) => item.key).join(',') === keys.join(','), 'module keys')
for (const key of keys) {
  assert(modulePath(key) === `/app/outpatient/${key}`, `path ${key}`)
}
const diagnosisModule = OUTPATIENT_MODULES.find((item) => item.key === 'diagnosis')
assert(diagnosisModule.status === 'available', 'diagnosis module available')
assert(diagnosisModule.roles.join(',') === 'DOCTOR,SYSTEM', 'diagnosis role scope')

const usable = new Set(['patient', 'register', 'diagnosis', 'order', 'charge', 'dispense', 'refund'])
for (const item of OUTPATIENT_MODULES) {
  if (usable.has(item.key)) assert(item.status === 'available', `${item.key} is usable`)
  else assert(item.status === 'developing', `${item.key} status`)
  assert(item.status !== 'disabled', `${item.key} should stay enterable`)
}

const system = decorateModules(['SYSTEM'])
assert(system.every((item) => item.allowed), 'system can open all')
assert(system.find((item) => item.key === 'patient').statusLabel === '可用', 'patient badge')
assert(system.find((item) => item.key === 'register').statusLabel === '可用', 'register badge')
assert(system.find((item) => item.key === 'diagnosis').statusLabel === '可用', 'diagnosis badge')
assert(system.find((item) => item.key === 'order').statusLabel === '可用', 'order badge')
assert(
  system.filter((item) => !usable.has(item.key)).every((item) => item.statusLabel === '开发中'),
  'other modules stay developing',
)
assert(system.every((item) => !item.recommended), 'system has no single recommendation')

const registrar = decorateModules(['REGISTRAR'])
assert(registrar.find((item) => item.key === 'patient').allowed, 'registrar patient')
assert(registrar.find((item) => item.key === 'register').recommended, 'registrar register')
assert(!registrar.find((item) => item.key === 'charge').allowed, 'registrar charge locked')
assert(registrar.find((item) => item.key === 'charge').reason === '无权限', 'registrar reason')

const doctor = decorateModules(['DOCTOR'])
assert(doctor.find((item) => item.key === 'order').recommended, 'doctor order')
assert(doctor.find((item) => item.key === 'diagnosis').recommended, 'doctor diagnosis')
assert(!doctor.find((item) => item.key === 'dispense').allowed, 'doctor dispense locked')
assert(!canEnter(diagnosisModule, { permissions: ['encounter:check-in'], scopes: ['outpatient:dept-1'] }), 'check-in does not grant diagnosis')
assert(canEnter(diagnosisModule, { permissions: ['diagnose:write'], scopes: ['outpatient:dept-1'] }), 'diagnosis permission grants diagnosis')

const cashier = decorateModules(['CASHIER'])
assert(cashier.find((item) => item.key === 'charge').allowed, 'cashier charge')
assert(cashier.find((item) => item.key === 'refund').recommended, 'cashier refund')
assert(!cashier.find((item) => item.key === 'order').allowed, 'cashier order locked')

const pharmacist = decorateModules(['PHARMACIST'])
assert(pharmacist.find((item) => item.key === 'dispense').recommended, 'pharmacist dispense')
assert(pharmacist.find((item) => item.key === 'refund').allowed, 'pharmacist refund')
assert(!pharmacist.find((item) => item.key === 'register').allowed, 'pharmacist register locked')

assert(decorateModules([]).every((item) => !item.allowed), 'empty roles locked')
assert(roleText(['SYSTEM']) === '系统', 'system role label')
assert(roleText(['CASHIER', 'PHARMACIST']) === '收费员、药师', 'joined roles')
assert(!roleText(['SYSTEM']).toLowerCase().includes('admin'), 'role label')

assert(canEnter({ status: 'disabled', roles: ['SYSTEM'] }, ['SYSTEM']) === false, 'disabled never navigates')
assert(blockReason({ status: 'disabled', roles: ['SYSTEM'] }, ['SYSTEM']) === '未启用', 'disabled reason')
const scopedOrderUser = { roles: ['CUSTOM'], permissions: ['order:submit'], scopes: ['outpatient:dept-1'] }
assert(canEnter(OUTPATIENT_MODULES.find((item) => item.key === 'order'), scopedOrderUser), 'effective permission opens scoped order')
assert(!canEnter(OUTPATIENT_MODULES.find((item) => item.key === 'order'), { ...scopedOrderUser, scopes: ['master:dept-1'] }), 'other module scope stays locked')
assert(!canEnter(diagnosisModule, { roles: ['REGISTRAR'], permissions: ['encounter:check-in'], scopes: ['outpatient:dept-1'], accessControlLoaded: true }), 'registrar check-in permission cannot open diagnosis')
assert(canEnter(diagnosisModule, { roles: ['DOCTOR'], permissions: ['diagnose:write'], scopes: ['outpatient:dept-1'], accessControlLoaded: true }), 'doctor diagnosis permission opens diagnosis')
assert(!hasEffectivePermission({ permissions: ['charge:refund'], deniedPermissions: ['charge:refund'] }, ['charge:refund']), 'explicit deny wins')
assert(hasEffectivePermission({ permissions: ['charge:refund'], scopes: ['outpatient:dept-1'] }, ['charge:refund']), 'effective charge scope opens refund')
assert(!canEnter(OUTPATIENT_MODULES.find((item) => item.key === 'charge'), { roles: ['CASHIER'], permissions: [], scopes: [] }), 'empty effective permissions do not fall back to role')
assert(canEnter(OUTPATIENT_MODULES.find((item) => item.key === 'patient'), { roles: ['SYSTEM'], permissions: [], scopes: [], accessControlLoaded: true, super: true }), 'elevated modern user bypasses empty effective permissions')

assert(DASHBOARD_PERIODS.join(',') === 'TODAY,YESTERDAY,WEEK,MONTH', 'summary periods')
assert(PERIOD_OPTIONS.map((item) => item.label).join(',') === '今天,昨天,本周,本月', 'period labels')
assert(PERIOD_OPTIONS.map((item) => item.value).join(',') === DASHBOARD_PERIODS.join(','), 'period values')
assert(CHARGE_METRICS.map((item) => item.label).join(',') === '收费金额,退费金额,净收入,收费笔数', 'charge metrics')
assert(VISIT_METRICS.map((item) => item.label).join(',') === '门诊人次,平均客单价', 'visit metrics')
assert(!CHARGE_METRICS.some((item) => ['微信', '支付宝', '银行卡', '医保'].includes(item.label)), 'unsupported payment methods are hidden')

const summaryContract = assertDashboardSummary({
  period: 'TODAY',
  rangeStart: '2026-10-14T00:00:00+08:00',
  rangeEnd: '2026-10-15T00:00:00+08:00',
  revenueCents: 2000,
  refundCents: 500,
  netRevenueCents: 1500,
  chargeCount: 1,
  refundCount: 1,
  encounterCount: 2,
  averageTicketCents: 750,
  pendingChargeCount: 3,
  pendingDispenseCount: 4,
  available: { charges: true, encounters: true, dispenses: false },
}, 'TODAY')
assert(summaryContract.netRevenueCents === 1500 && summaryContract.available.dispenses === false, 'summary response contract')
const requestGate = createDashboardRequestGate()
const firstRequest = requestGate.begin()
const latestRequest = requestGate.begin()
assert(requestGate.isCurrent(latestRequest), 'latest dashboard request remains current')
assert(!requestGate.isCurrent(firstRequest), 'stale dashboard response is ignored')
assert(TODO_CARDS.find((item) => item.key === 'charge').valueKey === 'pendingChargeCount', 'todo charge reads summary')
assert(TODO_CARDS.find((item) => item.key === 'dispense').valueKey === 'pendingDispenseCount', 'todo dispense reads summary')
for (const key of ['upload', 'standard']) {
  const card = TODO_CARDS.find((item) => item.key === key)
  assert(card.phase === '二期' && card.value === null && !card.valueKey, `${key} is deferred`)
}
assert(TODO_CARDS.find((item) => item.key === 'stock').value === null, 'stock threshold is not configured')
let invalidSummaryRejected = false
try {
  assertDashboardSummary({ ...summaryContract, netRevenueCents: 0 }, 'TODAY')
} catch {
  invalidSummaryRejected = true
}
assert(invalidSummaryRejected, 'inconsistent totals are rejected')

const todoLabels = ['待收费', '待发药', '进销存未上传', '未贯标提醒', '库存预警']
assert(TODO_CARDS.map((item) => item.label).join(',') === todoLabels.join(','), 'todo order')
assert(TODO_CARDS.find((item) => item.key === 'charge').module === 'charge', 'todo charge link')
assert(TODO_CARDS.find((item) => item.key === 'dispense').module === 'dispense', 'todo dispense link')
assert(TODO_CARDS.find((item) => item.key === 'charge').valueKey === 'pendingChargeCount', 'todo charge uses live count')
assert(TODO_CARDS.find((item) => item.key === 'dispense').valueKey === 'pendingDispenseCount', 'todo dispense uses live count')
for (const key of ['upload', 'standard']) {
  const card = TODO_CARDS.find((item) => item.key === key)
  assert(card.phase === '二期' && card.value === null && !card.module, `${key} is deferred`)
}
const stock = TODO_CARDS.find((item) => item.key === 'stock')
assert(stock.value === null && stock.phase === '未配置' && !stock.module, 'stock threshold is unconfigured')

const home = NAV.find((item) => item.key === 'home')
assert(home.to.name === 'home', 'home nav')
const archive = NAV.find((item) => item.key === 'archive')
const archiveCreate = archive.children.find((item) => item.key === 'archive-patient-create')
const archiveSearch = archive.children.find((item) => item.key === 'archive-patient-search')
assert(archiveCreate?.label === '患者建档' && archiveCreate.module === 'patient', 'archive create uses patient access')
assert(archiveSearch?.label === '患者查询' && archiveSearch.module === 'patient', 'archive search uses patient access')
assert(archiveCreate.to?.name === 'patient-create' && archiveSearch.to?.name === 'patient-search', 'archive routes are independent')
assert(archive.children.at(-1).phase === '二期' && !archive.children.at(-1).module, 'archive card deferred')
const outpatient = NAV.find((item) => item.key === 'outpatient')
assert(
  outpatient.children.map((item) => item.label).join(',') === '挂号,诊断,开单,收费,发药,退药/退费',
  'sidebar steps',
)
assert(outpatient.children.map((item) => item.module).join(',') === keys.filter((key) => key !== 'patient').join(','), 'sidebar modules')
assert(outpatient.overview.to.name === 'workbench', 'overview route name')
const settings = NAV.find((item) => item.key === 'settings')
const masterNav = settings.children.find((item) => item.key === 'settings-master')
const rbacNav = settings.children.find((item) => item.key === 'settings-users')
assert(masterNav && masterNav.label === '基础数据' && masterNav.to.name === 'master', 'master nav')
assert(!settings.children.some((item) => item.label === '基础字典'), 'dict placeholder replaced')
assert(rbacNav && rbacNav.to.name === 'access-users' && rbacNav.access === 'users' && rbacNav.label === '用户与权限', 'full rbac route')
assert(resolveNavChild(masterNav, system, { username: 'system', roles: ['SYSTEM'] }).enabled === true, 'system opens master')
assert(resolveNavChild(masterNav, system, { username: 'system', roles: ['SYSTEM'] }).to.name === 'master', 'master route')
assert(resolveNavChild(masterNav, registrar, { username: 'reg', roles: ['REGISTRAR'] }).hidden === true, 'registrar hides master')
assert(resolveNavChild(rbacNav, system, { username: 'system', roles: ['SYSTEM'] }).to.name === 'access-users', 'rbac item route')
assert(resolveNavChild(rbacNav, [], { permissions: ['admin:role'] }).to.name === 'access-roles', 'rbac nav opens first granted tab')
assert(resolveNavChild(rbacNav, [], { permissions: ['admin:permission'] }).to.name === 'access-permissions', 'permission-only nav route')
assert(resolveNavChild(rbacNav, [], { permissions: ['audit:read'] }).to.name === 'access-user-logs', 'audit-only nav route')
assert(resolveNavChild(rbacNav, [], { permissions: [] }).hidden === true, 'rbac nav hides with no access capabilities')

const labels = []
for (const item of NAV) {
  labels.push(item.label)
  for (const child of item.children || []) labels.push(child.label)
  if (item.overview) labels.push(item.overview.label)
}
for (const word of ['住院', '病历', '采购', '报表']) {
  assert(!labels.some((label) => label.includes(word)), `nav ${word}`)
}

const systemNav = decorateModules(['SYSTEM'])
const registrarNav = decorateModules(['REGISTRAR'])
const chargeChild = outpatient.children.find((item) => item.module === 'charge')
assert(resolveNavChild(archiveCreate, systemNav).to.name === 'patient-create', 'create route passes patient module access')
assert(resolveNavChild(archiveSearch, systemNav).to.name === 'patient-search', 'search route passes patient module access')
const deniedPatientNav = resolveNavChild(archiveSearch, [], { roles: ['REGISTRAR'], permissions: [], scopes: [], accessControlLoaded: true })
assert(!deniedPatientNav.enabled && deniedPatientNav.to === null, 'patient route is hidden without effective permission')
assert(resolveNavChild(chargeChild, registrarNav).enabled === false, 'registrar cannot open charge from nav')
assert(resolveNavChild(chargeChild, systemNav).enabled === true, 'system can open charge from nav')
assert(resolveNavChild(archive.children.at(-1), systemNav).to === null, 'phase item has no route')
assert(resolveNavChild(rbacNav, systemNav).to === null, 'settings rbac has no route')

const uiFiles = [
  '../src/components/AppShell.vue',
  '../src/components/SideNav.vue',
  '../src/views/DashboardView.vue',
  '../src/views/WorkbenchView.vue',
  '../src/views/ModulePlaceholderView.vue',
]
const ui = uiFiles
  .map((file) => readFileSync(new URL(file, import.meta.url), 'utf8'))
  .join('\n')
  .replace(/<template>/g, '\n<template>')
  .match(/<template>[\s\S]*?<\/template>/g)
  .join('\n')
const required = [
  'B-HIS',
  '首页',
  '工作台',
  '门诊业务',
  '挂号 → 诊断 → 开单 → 收费 → 发药 → 退药/退费',
  '收费前须有挂号/就诊 · 支付演示现金 · 已发药先退药再退费',
  '个人设置',
  '退出登录',
  '接口就绪后开放',
  '机构',
  '人员',
  '演示机构',
  '经营概况',
  '待办为当前快照',
  '待办',
  '刷新',
  '二期',
]
for (const text of required) assert(ui.includes(text), `missing copy: ${text}`)
const shellTemplate = readFileSync(new URL('../src/components/AppShell.vue', import.meta.url), 'utf8')
const sideNavTemplate = readFileSync(new URL('../src/components/SideNav.vue', import.meta.url), 'utf8')
assert(shellTemplate.includes('class="brand-link" :to="{ name: \'home\' }"'), 'B-HIS brand links to home')
assert(!sideNavTemplate.includes('class="nav-toplink"'), 'sidebar has no standalone home link')
const dashboardTemplate = readFileSync(new URL('../src/views/DashboardView.vue', import.meta.url), 'utf8')
const dashboardSource = readFileSync(new URL('../src/workbench/dashboard.js', import.meta.url), 'utf8')
for (const label of ['收费金额', '退费金额', '净收入', '收费笔数', '门诊人次', '平均客单价']) {
  assert(dashboardSource.includes(label), `missing dashboard metric: ${label}`)
}
assert(dashboardTemplate.includes('fetchDashboardSummary'), 'dashboard requests actual summary')
assert(!dashboardTemplate.includes('PLACEHOLDER_VALUE'), 'dashboard has no fixed metric value')
assert(!dashboardTemplate.includes('DonutPlaceholder'), 'dashboard has no equal-slice chart')
for (const text of ['住院', '病历', '报表', '采购', 'admin', 'H05', 'H02', 'H03', '搜索患者', '1959']) {
  assert(!ui.includes(text), `forbidden copy: ${text}`)
}
assert(!ui.includes('module-grid') || readFileSync(new URL('../src/views/DashboardView.vue', import.meta.url), 'utf8').includes('module-grid') === false, 'dashboard is not the card wall')
assert(readFileSync(new URL('../src/views/DashboardView.vue', import.meta.url), 'utf8').includes('module-grid') === false, 'dashboard template has no card wall')
assert(readFileSync(new URL('../src/views/WorkbenchView.vue', import.meta.url), 'utf8').includes('module-grid'), 'overview keeps six cards')

const routerSrc = readFileSync(new URL('../src/router/index.js', import.meta.url), 'utf8')
assert(/path:\s*''[\s\S]{0,180}name:\s*'home'/.test(routerSrc), 'default child is home')
assert(/path:\s*'outpatient'[\s\S]{0,180}name:\s*'workbench'/.test(routerSrc), 'overview is /app/outpatient')
assert(!routerSrc.includes("import ReceptionDeskView"), 'today reception view is no longer mounted')
assert(!routerSrc.includes("name: 'reception'"), 'today reception route is removed')
assert(/path:\s*'outpatient\/reception'[\s\S]{0,180}redirect:\s*\{\s*name:\s*'workbench'\s*\}/.test(routerSrc), 'legacy today reception URL redirects to overview')
assert(!routerSrc.includes("import ReceptionSettingsPanel"), 'today reception settings panel is removed')
assert(/path:\s*'reception'[\s\S]{0,140}redirect:\s*\{\s*name:\s*'master'\s*\}/.test(routerSrc), 'legacy reception settings URL redirects to master overview')
assert(routerSrc.includes("name: 'patient-create'") && routerSrc.includes("name: 'patient-search'"), 'archive pages have independent routes')
assert(routerSrc.includes("path: 'outpatient/patient'") && routerSrc.includes("name: 'patient-create'"), 'legacy outpatient create URL redirects to archive')
assert(routerSrc.includes('patientAccess') && routerSrc.includes('canEnter(findModule(\'patient\')'), 'both archive routes preserve patient access guard')
assert(/name:\s*'home'/.test(readFileSync(new URL('../src/views/LoginView.vue', import.meta.url), 'utf8')), 'login lands on home')
assert(!readFileSync(new URL('../src/views/LoginView.vue', import.meta.url), 'utf8').includes("name: 'workbench'"), 'login skips card wall')

const workbenchView = readFileSync(new URL('../src/views/WorkbenchView.vue', import.meta.url), 'utf8')
assert(workbenchView.includes("filter((item) => item.key !== 'patient')"), 'overview filters patient access module from outpatient cards')
assert(!workbenchView.split('<template>')[1].split('</template>')[0].includes('建档'), 'overview has no archive step copy')
const patientView = readFileSync(new URL('../src/views/PatientView.vue', import.meta.url), 'utf8')
assert(!patientView.includes('data-testid="patient-search"') && !patientView.includes('searchPatients') && !patientView.includes('searchForm'), 'create page has no query panel or query state')
assert(patientView.includes("name: 'patient-search'"), 'duplicate feedback links to independent search page')
assert(!patientView.includes('patient-success-reception') && !patientView.includes("name: 'reception'"), 'patient success does not expose today reception')
const patientSearchView = readFileSync(new URL('../src/views/PatientSearchView.vue', import.meta.url), 'utf8')
for (const text of ['patient-search', 'searchPatients', 'fetchPatient', '去挂号']) {
  assert(patientSearchView.includes(text), `search page keeps ${text}`)
}
const frameView = readFileSync(new URL('../src/components/OutpatientWorkbenchFrame.vue', import.meta.url), 'utf8')
assert(!frameView.includes("['patient', '建档']"), 'outpatient frame has no archive step')
assert(!frameView.includes('outpatient-left-rail'), 'outpatient frame removes the left rail')
assert(frameView.includes('data-testid="outpatient-context"'), 'outpatient frame exposes patient context')
assert(frameView.includes('data-testid="outpatient-status"'), 'outpatient frame exposes current workflow status')
assert(frameView.includes('data-testid="outpatient-status-chain"'), 'outpatient frame exposes workflow status chain')
assert(frameView.includes('WorkflowSteps'), 'outpatient frame renders shared workflow navigation')
const shellView = readFileSync(new URL('../src/components/AppShell.vue', import.meta.url), 'utf8')
assert(!shellView.includes("import WorkflowSteps"), 'app shell no longer imports outpatient workflow strip')
assert(!shellView.includes('<WorkflowSteps'), 'app shell no longer renders outpatient workflow strip')
const navSource = readFileSync(new URL('../src/workbench/nav.js', import.meta.url), 'utf8')
assert(!navSource.includes('今日接待') && !navSource.includes('quick:'), 'sidebar removes today reception shortcut')
const masterView = readFileSync(new URL('../src/views/MasterView.vue', import.meta.url), 'utf8')
assert(!masterView.includes('master-reception') && !masterView.includes('门诊运营'), 'master tabs remove reception-only settings')
assert(!existsSync(new URL('../src/views/ReceptionDeskView.vue', import.meta.url)), 'today reception page source is removed')
assert(!existsSync(new URL('../src/views/master/ReceptionSettingsPanel.vue', import.meta.url)), 'today reception settings source is removed')

const diagnosisApi = readFileSync(new URL('../src/api/diagnosisApi.js', import.meta.url), 'utf8')
assert(diagnosisApi.includes("/diagnoses/${encodeURIComponent(diagnoseId)}/commit"), 'diagnosis commit route')
assert(diagnosisApi.includes("/diagnoses/${encodeURIComponent(diagnoseId)}/abate"), 'diagnosis abate route')
const diagnosisView = readFileSync(new URL('../src/views/DiagnosisView.vue', import.meta.url), 'utf8')
assert(diagnosisView.includes("canEnter(findModule('diagnosis'), session.user)"), 'diagnosis uses effective permission gate')
for (const text of ['当前挂号', '保存诊断', '提交', '作废', '临床规则当前保持关闭']) {
  assert(diagnosisView.includes(text), `diagnosis copy: ${text}`)
}
for (const text of ['主诉', '现病史', '既往史 / 过敏史', '查体与检查', '初步诊断', '处置计划', '全科初诊', 'serializeStructured', 'templateCode', 'signedAt', 'diagnoseVersion']) {
  assert(diagnosisView.includes(text), `structured diagnosis copy: ${text}`)
}

console.log('workbench checks passed')
