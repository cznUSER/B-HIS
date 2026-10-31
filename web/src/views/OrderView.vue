<script setup>
import { useErrorDialog } from '../components/feedback/useErrorDialog.js'
import { createSubmitFeedback } from '../components/feedback/useSubmitFeedback.js'
import SubmitStatusBanner from '../components/feedback/SubmitStatusBanner.vue'
import OrderPrintPanel from '../components/outpatient/OrderPrintPanel.vue'
import OrderItemFinder from '../components/outpatient/OrderItemFinder.vue'
import {
  COMMON_ORDER_ITEMS,
  applyCommonItemToLines,
  resolveCommonItem,
} from '../workbench/commonOrderItems.js'
import {
  buildPreviewModel,
  canPreviewOrderStatus,
  createOrderPrintController,
  PLACEHOLDER_HONESTY,
  SESSION_PERSISTENCE_NOTE,
} from '../workbench/orderPrintStatus.js'
// 2026-10-12 P1-H05-ORDER-UI 门诊开单。主字段 registerId。
// 2026-10-12 VITE_USE_MOCK=false 时暂存/改草稿/提交/作废走 /api/v1/orders；无挂号仍是空态。
// 2026-10-13 P1-H05-MASTER-UI 开单医生和目录下拉跳过已停用项。
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
  openCatalog,
  orderStatusLabel,
  registerIsOrderable,
  round2,
  stockWarnings,
} from '../api/orderContract.js'
import { session } from '../auth/session.js'
import { fetchCatalog, fetchDepartments, fetchDoctors, fetchInventory } from '../master/api.js'
import { selectableDoctors } from '../register/master.js'
import BusinessPageHeader from '../components/BusinessPageHeader.vue'
import BusinessEmptyState from '../components/BusinessEmptyState.vue'
import { recordAction, setOrders, setRegister } from '../workbench/outpatientContext.js'
import './clinical-design.css'

const route = useRoute()
const router = useRouter()

const registers = ref([])
const pickedId = ref('')
const register = ref(null)
const invalidMessage = ref('')
const loadingRegister = ref(false)
const loadingRegisters = ref(false)
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
const submitFeedback = createSubmitFeedback()
const printController = createOrderPrintController()
const printStatus = computed(() => printController.status)
const printLocked = computed(() => printController.isLocked)
const printPreviewModel = computed(() => printController.previewModel)
const printCanPreview = computed(() => Boolean(orderId.value) && canPreviewOrderStatus(status.value))
const printErrorMessage = computed(() => printController.errorMessage)
const printLastPrintedAt = computed(() => printController.lastPrintedAt || '')
let printActionKind = 'print'
let saveAttempt = null
let commitSaveAttempt = null
let commitAttempt = null
let abateAttempt = null
let loadSeq = 0
const mockMode = useMock()
// 2026-10-14 P1-H05-WEB-USABILITY: distinguish failed live reads from empty data.
const remoteDepartments = ref([])
const remoteDoctors = ref([])
const remoteCatalog = ref([])
const remoteInventory = ref([])
const masterLoading = ref(!mockMode)
const masterReady = ref(mockMode)
const masterError = ref('')
const registersError = ref('')
const registerError = ref('')
const ordersLoading = ref(false)
const ordersReady = ref(false)
const ordersError = ref('')
let ordersLoadSeq = 0

const busy = computed(() => saving.value || committing.value || abating.value || submitFeedback.state.status === 'submitting' || printLocked.value)
const selectedLineCount = computed(() => lines.value.filter((line) => line.itemNo).length)
const orderableRegisters = computed(() => registers.value.filter((item) => registerIsOrderable(item)))
const orderDoctors = computed(() => {
  if (mockMode) return selectableDoctors()
  const activeDept = new Set(remoteDepartments.value.filter((item) => item.status === 'ACTIVE').map((item) => item.departmentId))
  return remoteDoctors.value.filter((item) => item.status === 'ACTIVE' && activeDept.has(item.departmentId)).map((item) => ({ ...item, name: item.doctorName }))
})
const drugOptions = computed(() => listedCatalog('DRUG'))
const treatOptions = computed(() => listedCatalog('TREAT'))
const findItems = computed(() => [...drugOptions.value, ...treatOptions.value])
const commonMissingTip = ref('')
const commonChips = computed(() =>
  COMMON_ORDER_ITEMS.map((item) => {
    const resolved = resolveCommonItem(item.itemNo, findItems.value)
    return {
      itemNo: item.itemNo,
      label: item.label,
      itemType: item.itemType,
      resolved: Boolean(resolved),
    }
  }),
)

function listedCatalog(itemType) {
  return (mockMode ? openCatalog() : remoteCatalog.value.filter((item) => item.status === 'ACTIVE')).filter((item) => item.itemType === itemType)
}
const locked = computed(() => ['COMMIT', 'ABATE', 'PROC'].includes(status.value) || charged.value)
const canAbate = computed(() => Boolean(orderId.value) && !charged.value && (status.value === 'SAVE' || status.value === 'COMMIT'))
const canWrite = computed(() => masterReady.value && !masterLoading.value)
const warnings = computed(() => {
  if (mockMode) return stockWarnings(lines.value.filter((line) => line.itemNo && Number(line.amount) > 0))
  return lines.value.filter(lineShort).map((line) => {
    const item = catalogOf(line.itemNo, line)
    return `${item.itemName} 可用 ${item.stockQty}，本次 ${Number(line.amount)}。库存不足，开单不拦截。`
  })
})
const totalFee = computed(() => round2(lines.value.reduce((sum, line) => sum + lineFee(line), 0)))
const chargeText = computed(() => {
  if (charged.value) return '已收费'
  if (status.value === 'COMMIT') return '可收费'
  if (status.value === 'ABATE') return '已作废'
  return '不可收费'
})
const patientContext = computed(() => register.value?.patientId || queryText(route.query.patientId))
const registerTo = computed(() => ({ name: 'outpatient-module', params: { module: 'register' }, query: patientContext.value ? { patientId: patientContext.value } : {} }))

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

function catalogOf(itemNo, line) {
  if (mockMode) return CATALOG.find((item) => item.itemNo === itemNo) || null
  const snapshot = line?.sourceItem?.itemNo === itemNo ? line.sourceItem : null
  const item = snapshot || remoteCatalog.value.find((item) => item.itemNo === itemNo)
  if (!item) return null
  const inventory = remoteInventory.value.find((row) => (row.itemNo || row.skuCode) === itemNo)
  return { ...item, stockQty: item.itemType === 'DRUG' && inventory ? Number(inventory.qtyOnHand) : null }
}

function lineFee(line) {
  const catalog = catalogOf(line.itemNo, line)
  const amount = Number(line.amount)
  if (!catalog || !Number.isFinite(amount) || amount <= 0) return 0
  if (!mockMode && line.sourceItem?.itemNo === line.itemNo && amount === line.sourceItem.amount) return round2(line.sourceItem.totalFee)
  return round2(amount * catalog.price)
}

function lineMoney(line) {
  if (!catalogOf(line.itemNo, line) || !Number.isFinite(Number(line.amount)) || Number(line.amount) <= 0) return '—'
  return money(lineFee(line))
}

function lineShort(line) {
  const catalog = catalogOf(line.itemNo, line)
  const amount = Number(line.amount)
  if (!catalog || catalog.stockQty == null || !Number.isFinite(amount)) return false
  return amount > catalog.stockQty
}

function queryRegisterId() {
  return queryText(route.query.registerId)
}

function queryText(value) {
  const raw = Array.isArray(value) ? value[0] : value
  return typeof raw === 'string' ? raw.trim() : ''
}

function clearBanner() {
  errorMessage.value = ''
  infoMessage.value = ''
  confirmAbate.value = false
  submitFeedback.reset()
}

function resetPrintSession() {
  printController.resetToIdle()
}

function currentPrintPreviewModel() {
  if (!register.value || !orderId.value) return null
  return buildPreviewModel({
    register: register.value,
    orderMeta: {
      patientId: register.value.patientId,
      patientName: register.value.patientName,
      patientNo: register.value.patientNo,
      registerId: register.value.registerId,
      registerNo: register.value.registerNo,
      orderId: orderId.value,
      orderNo: orderNo.value,
      status: status.value,
      diagnoseName: form.diagnoseName,
    },
    lines: lines.value,
    catalogLookup: (itemNo, line) => catalogOf(itemNo, line),
  })
}

function onPrintPreview() {
  if (!printCanPreview.value || printLocked.value) return
  try {
    const model = currentPrintPreviewModel()
    if (!model) return
    printController.openPreview(model)
  } catch (error) {
    errorMessage.value = error?.message || '无法打开预览'
  }
}

function runPrintPlaceholder(kind) {
  if (printLocked.value) return
  printActionKind = kind === 'reprint' ? 'reprint' : 'print'
  const started = kind === 'reprint' ? printController.reprint() : kind === 'retry' ? printController.retry() : printController.startOutput()
  if (!started) return
  try {
    if (typeof window !== 'undefined' && typeof window.print === 'function') {
      window.print()
    }
    // Keep PRINTING until user confirms; window.print is only a local placeholder.
  } catch (error) {
    printController.fail(error?.message || '本机打印占位失败')
  }
}

function onPrintOutput() {
  runPrintPlaceholder('print')
}

function onPrintConfirm() {
  if (!printController.confirmPrinted(new Date().toISOString())) return
  const label = printActionKind === 'reprint' ? '重打' : '打印'
  recordAction(label, orderNo.value || orderId.value || '')
  printActionKind = 'print'
}

function onPrintRetry() {
  runPrintPlaceholder('retry')
}

function onPrintReprint() {
  runPrintPlaceholder('reprint')
}

function onPrintClose() {
  if (printLocked.value) return
  resetPrintSession()
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
  resetPrintSession()
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
    sourceItem: { ...item },
  }))
  lines.value = next.length > 0 ? next : [blankLine()]
  // 2026-10-14: keep the confirmed response even if the following list read fails.
  orders.value = [order, ...orders.value.filter((item) => item.orderId !== order.orderId)]
  setOrders(orders.value)
  confirmAbate.value = false
  // Keep print session bound to current order id; rebuild on next preview.
  if (printController.previewModel && printController.previewModel.orderId && printController.previewModel.orderId !== order.orderId) resetPrintSession()
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
      // 2026-10-14: preserve the loaded response fields for an unchanged line.
      sourceItem: line.sourceItem && line.sourceItem.itemNo === line.itemNo ? { ...line.sourceItem } : undefined,
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
  const currentId = register.value?.registerId
  const seq = ++ordersLoadSeq
  if (!currentId) {
    orders.value = []
    ordersReady.value = false
    return
  }
  ordersLoading.value = true
  ordersError.value = ''
  try {
    const listed = await listOrders(session.accessToken, currentId)
    if (seq !== ordersLoadSeq || currentId !== register.value?.registerId) return
    orders.value = listed.items
    setOrders(orders.value)
    ordersReady.value = true
  } catch (error) {
    if (seq !== ordersLoadSeq || currentId !== register.value?.registerId) return
    ordersReady.value = false
    ordersError.value = error instanceof ApiError ? error.message : '暂时无法读取处方列表'
  } finally {
    if (seq === ordersLoadSeq) ordersLoading.value = false
  }
}

async function loadRegisters() {
  loadingRegisters.value = true
  registersError.value = ''
  try {
    const listed = await listOrderRegisters(session.accessToken)
    registers.value = listed.items
  } catch (error) {
    registersError.value = error instanceof ApiError ? error.message : '暂时无法读取当日挂号'
  } finally {
    loadingRegisters.value = false
  }
}

async function loadRegister(id) {
  const seq = ++loadSeq
  ++ordersLoadSeq
  register.value = null
  invalidMessage.value = ''
  registerError.value = ''
  orders.value = []
  ordersReady.value = false
  ordersError.value = ''
  ordersLoading.value = false
  if (!id) {
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
    setRegister(found)
    startDraft(found)
    await refreshOrders()
  } catch (error) {
    if (seq !== loadSeq) return
    if (error instanceof ApiError && (error.status === 404 || error.code === 'ENCOUNTER_REQUIRED')) {
      invalidMessage.value = error.code === 'ENCOUNTER_REQUIRED' ? '该挂号当前不能开单，请核对挂号记录。' : '未找到该挂号，请重新选择。'
    } else {
      registerError.value = error instanceof ApiError ? error.message : '暂时无法读取挂号'
    }
  } finally {
    if (seq === loadSeq) loadingRegister.value = false
  }
}

function requireDoctor() {
  if (orderDoctors.value.some((item) => item.doctorId === form.doctorId.trim())) return true
  errorMessage.value = '请选择开单医生'
  return false
}

async function onSave() {
  if (busy.value || locked.value || !register.value || !canWrite.value) return
  clearBanner()
  if (!requireDoctor()) return
  const body = draftBody()
  const fingerprint = JSON.stringify({ action: 'save', body })
  if (!saveAttempt || saveAttempt.fingerprint !== fingerprint) saveAttempt = { fingerprint, key: crypto.randomUUID() }
  saving.value = true
  submitFeedback.beginAttempt('save')
  const currentSeq = loadSeq
  try {
    const key = orderId.value ? undefined : saveAttempt.key
    const order = await saveOrder(session.accessToken, body, mockMode ? undefined : remoteCatalog.value, key)
    if (currentSeq !== loadSeq) return
    applyOrder(order)
    recordAction('\u5904\u65b9\u5df2\u6682\u5b58', order.orderNo || order.orderId || '')
    saveAttempt = null
    infoMessage.value = ''
    const tip = form.diagnoseName.trim()
      ? '\u5df2\u6682\u5b58\u3002\u6682\u5b58\u5904\u65b9\u4e0d\u80fd\u6536\u8d39\u3002'
      : '\u5df2\u6682\u5b58\u3002\u8bca\u65ad\u53ef\u4ee5\u540e\u8865\uff0c\u4e0d\u5f71\u54cd\u6682\u5b58\u3002\u6682\u5b58\u5904\u65b9\u4e0d\u80fd\u6536\u8d39\u3002'
    submitFeedback.succeed({ orderId: order.orderId || '', orderNo: order.orderNo || '' }, tip + '\u6d41\u6c34\u53f7 ' + (order.orderNo || order.orderId || ''))
    await refreshOrders()
  } catch (error) {
    if (currentSeq !== loadSeq) return
    const message = error instanceof ApiError ? error.message : '\u6682\u5b58\u5931\u8d25\uff0c\u8bf7\u7a0d\u540e\u91cd\u8bd5'
    errorMessage.value = message
    submitFeedback.fail(message, error?.code || '')
  } finally {
    saving.value = false
  }
}

async function onCommit() {
  if (busy.value || locked.value || !register.value || !canWrite.value) return
  clearBanner()
  if (!requireDoctor()) return
  const body = draftBody()
  const saveFingerprint = JSON.stringify({ action: 'commit-save', body })
  if (!commitSaveAttempt || commitSaveAttempt.fingerprint !== saveFingerprint) {
    commitSaveAttempt = { fingerprint: saveFingerprint, key: crypto.randomUUID() }
  }
  committing.value = true
  submitFeedback.beginAttempt('commit')
  const currentSeq = loadSeq
  try {
    const saveKey = orderId.value ? undefined : commitSaveAttempt.key
    const saved = await saveOrder(session.accessToken, body, mockMode ? undefined : remoteCatalog.value, saveKey)
    if (currentSeq === loadSeq) applyOrder(saved)
    const commitFingerprint = JSON.stringify({ action: 'commit', orderId: saved.orderId })
    if (!commitAttempt || commitAttempt.fingerprint !== commitFingerprint) {
      commitAttempt = { fingerprint: commitFingerprint, key: crypto.randomUUID() }
    }
    const committed = await commitOrder(session.accessToken, saved.orderId, commitAttempt.key)
    if (currentSeq !== loadSeq) return
    applyOrder(committed)
    recordAction('\u5904\u65b9\u5df2\u63d0\u4ea4', committed.orderNo || committed.orderId || '')
    commitSaveAttempt = null
    commitAttempt = null
    infoMessage.value = ''
    const tip = form.diagnoseName.trim()
      ? '\u5df2\u63d0\u4ea4\uff0c\u53ef\u4f9b\u6536\u8d39\u3002\u672a\u6536\u8d39\u524d\u53ef\u4ee5\u4f5c\u5e9f\u3002'
      : '\u5df2\u63d0\u4ea4\uff0c\u53ef\u4f9b\u6536\u8d39\u3002\u8bca\u65ad\u672a\u586b\uff0c\u4e0d\u5f71\u54cd\u63d0\u4ea4\u3002\u672a\u6536\u8d39\u524d\u53ef\u4ee5\u4f5c\u5e9f\u3002'
    submitFeedback.succeed({ orderId: committed.orderId || '', orderNo: committed.orderNo || '' }, tip + '\u6d41\u6c34\u53f7 ' + (committed.orderNo || committed.orderId || ''))
    await refreshOrders()
  } catch (error) {
    if (currentSeq !== loadSeq) return
    const message = error instanceof ApiError ? error.message : '\u63d0\u4ea4\u5931\u8d25\uff0c\u8bf7\u7a0d\u540e\u91cd\u8bd5'
    errorMessage.value = message
    submitFeedback.fail(message, error?.code || '')
    try {
      await refreshOrders()
    } catch {
      // keep form error when list refresh fails
    }
  } finally {
    committing.value = false
  }
}

async function onAbate() {
  if (busy.value || !canAbate.value) return
  errorMessage.value = ''
  infoMessage.value = ''
  submitFeedback.reset()
  const fingerprint = JSON.stringify({ action: 'abate', orderId: orderId.value })
  if (!abateAttempt || abateAttempt.fingerprint !== fingerprint) abateAttempt = { fingerprint, key: crypto.randomUUID() }
  abating.value = true
  submitFeedback.beginAttempt('abate')
  const currentSeq = loadSeq
  try {
    const order = await abateOrder(session.accessToken, orderId.value, abateAttempt.key)
    if (currentSeq !== loadSeq) return
    applyOrder(order)
    recordAction('\u5904\u65b9\u5df2\u4f5c\u5e9f', order.orderNo || order.orderId || '')
    abateAttempt = null
    infoMessage.value = ''
    submitFeedback.succeed({ orderId: order.orderId || '', orderNo: order.orderNo || '' }, '\u5df2\u4f5c\u5e9f\u3002\u5982\u9700\u7ee7\u7eed\uff0c\u8bf7\u65b0\u5f00\u4e00\u5f20\u3002\u6d41\u6c34\u53f7 ' + (order.orderNo || order.orderId || ''))
    await refreshOrders()
  } catch (error) {
    if (currentSeq !== loadSeq) return
    const message = error instanceof ApiError ? error.message : '\u4f5c\u5e9f\u5931\u8d25\uff0c\u8bf7\u7a0d\u540e\u91cd\u8bd5'
    errorMessage.value = message
    submitFeedback.fail(message, error?.code || '')
  } finally {
    abating.value = false
    confirmAbate.value = false
  }
}

function addLine() {
  if (locked.value || busy.value || !canWrite.value) return
  lines.value = [...lines.value, blankLine()]
}

function applyCommonOrderItem(itemNo) {
  if (locked.value || busy.value || !canWrite.value) return
  const resolved = resolveCommonItem(itemNo, findItems.value)
  if (!resolved) {
    commonMissingTip.value = '该常用项目当前目录不可用（未启用或不在目录中），未写入明细。'
    return
  }
  commonMissingTip.value = ''
  lines.value = applyCommonItemToLines(lines.value, resolved, { addBlankLine: () => blankLine() })
}

function removeLine(key) {
  if (locked.value || busy.value || !canWrite.value) return
  const next = lines.value.filter((line) => line.key !== key)
  lines.value = next.length > 0 ? next : [blankLine()]
}

async function usePicked() {
  if (!pickedId.value) return
  const selectedPatientId = registers.value.find((item) => item.registerId === pickedId.value)?.patientId || patientContext.value
  await router.replace({
    name: 'outpatient-module',
    params: { module: 'order' },
    query: { registerId: pickedId.value, ...(selectedPatientId ? { patientId: selectedPatientId } : {}) },
  })
}

function openListed(id) {
  resetPrintSession()
  if (busy.value) return
  const found = orders.value.find((item) => item.orderId === id)
  if (!found) return
  clearBanner()
  applyOrder(found)
}

function onNew() {
  resetPrintSession()
  if (!register.value || busy.value) return
  startDraft(register.value)
}

async function changeRegister() {
  if (busy.value) return
  await router.replace({ name: 'outpatient-module', params: { module: 'order' }, query: patientContext.value ? { patientId: patientContext.value } : {} })
}

onMounted(loadRegisters)
async function loadMaster() {
  if (mockMode) return
  masterLoading.value = true
  masterReady.value = false
  masterError.value = ''
  try {
    const [depts, docs, catalog] = await Promise.all([
      fetchDepartments(session.accessToken, {}),
      fetchDoctors(session.accessToken, {}),
      fetchCatalog(session.accessToken, {}),
    ])
    // Inventory is only used for a non-blocking stock hint. Doctors can open
    // prescriptions without inventory:read, so a permission response must
    // not prevent the catalog and doctor data from becoming usable.
    let inventory = { items: [] }
    try {
      inventory = await fetchInventory(session.accessToken, {})
    } catch (error) {
      if (!(error instanceof ApiError && (error.status === 403 || error.code === 'FORBIDDEN'))) throw error
    }
    if (catalog.items.some((item) => !item.itemNo || !item.itemName || !item.units || !['DRUG', 'TREAT'].includes(item.itemType) || !Number.isFinite(item.price) || item.price < 0)) throw new Error('invalid catalog')
    remoteDepartments.value = depts.items
    remoteDoctors.value = docs.items
    remoteCatalog.value = catalog.items
    remoteInventory.value = inventory.items
    masterReady.value = true
  } catch (error) {
    remoteDepartments.value = []
    remoteDoctors.value = []
    remoteCatalog.value = []
    remoteInventory.value = []
    masterError.value = error instanceof ApiError ? error.message : '无法读取开单基础数据'
  } finally {
    masterLoading.value = false
  }
}

function departmentLabel(current) {
  if (mockMode) return current.departmentName || current.departmentId
  return remoteDepartments.value.find((item) => item.departmentId === current.departmentId)?.deptName || current.departmentId || '—'
}

function doctorLabel(current) {
  if (mockMode) return current.doctorName || current.doctorId
  return remoteDoctors.value.find((item) => item.doctorId === current.doctorId)?.doctorName || current.doctorId || '—'
}

onMounted(loadMaster)

watch(() => queryRegisterId(), (id) => {
  loadRegister(id)
}, { immediate: true })

useErrorDialog([
  () => errorMessage.value,
  () => masterError.value,
  () => registersError.value,
  () => registerError.value,
  () => ordersError.value,
  () => invalidMessage.value,
])
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

    <BusinessPageHeader title-id="order-title" title="门诊开单" eyebrow="CLINICAL ORDERS" icon="order" description="聚焦本次就诊，在清晰的处方工作区完成诊断、药品和治疗项目的录入。" />

    <div v-if="loadingRegister" class="result-empty" role="status">正在读取挂号…</div>

    <div v-else-if="!register && registerError" class="banner clinical-read-error" role="alert"><p>{{ registerError }}。请重试当前挂号。</p><button class="button button-ghost button-compact" type="button" @click="loadRegister(queryRegisterId())">重试挂号资料</button></div>

    <p v-else-if="!register && loadingRegisters" class="result-empty" role="status">正在读取当日挂号…</p>

    <div v-else-if="!register && registersError" class="banner clinical-read-error" role="alert"><p>{{ registersError }}。暂时无法核对已有挂号。</p><button class="button button-ghost button-compact" type="button" @click="loadRegisters">重试当日挂号</button></div>

    <BusinessEmptyState
      v-else-if="!register"
      variant="revisit"
      module-label="开单"
      queue-label="待开单就诊"
      title="选择一条挂号开始开单"
      description="先选择有效挂号，患者资料、诊断和处方明细会在右侧连续展开。"
      empty-label="暂无可开单挂号"
      empty-description="有效挂号会进入队列；也可以直接去挂号创建一次就诊。"
      primary-text="去挂号"
      :primary-to="registerTo"
      secondary-text="患者查询"
      :secondary-to="{ name: 'patient-search' }"
      test-id="order-queue-workspace"
    >
      <template #detail>
      <p v-if="invalidMessage" class="banner order-empty-banner" role="alert">{{ invalidMessage }}</p>
      <p v-if="loadingRegisters" class="result-empty" role="status">正在读取当日挂号…</p>
      <form v-else-if="orderableRegisters.length" class="order-picker" @submit.prevent="usePicked">
        <label class="field">
          <span>选择已有挂号</span>
          <select v-model="pickedId" data-testid="register-picker">
            <option value="">请选择</option>
            <option v-for="item in orderableRegisters" :key="item.registerId" :value="item.registerId">
              {{ item.registerNo }} · {{ item.patientName || item.patientId }} · {{ departmentLabel(item) }} · {{ doctorLabel(item) }}
            </option>
          </select>
        </label>
        <button class="button button-compact" type="submit" :disabled="!pickedId">使用该挂号</button>
      </form>
      <div v-else class="clinical-empty-steps"><span><b>01</b> 有效挂号</span><i aria-hidden="true">→</i><span><b>02</b> 录入处方</span><i aria-hidden="true">→</i><span><b>03</b> 提交收费</span></div>
      </template>
    </BusinessEmptyState>

    <div v-else class="order-layout" data-testid="order-form">
      <form class="patient-panel clinical-order-editor" @submit.prevent>
        <div class="panel-head">
          <div><span class="clinical-kicker">处方工作区</span><h2>门诊处方</h2></div>
          <button class="text-btn" type="button" :disabled="busy" @click="changeRegister">更换挂号</button>
        </div>

        <section class="clinical-encounter-card" aria-label="本次就诊资料">
          <div class="clinical-encounter-name"><span class="clinical-profile-avatar" aria-hidden="true">{{ register.patientName?.slice(0, 1) || '患' }}</span><div><strong>{{ register.patientName || '—' }}</strong><span>{{ register.patientNo || '患者号未提供' }}</span></div><span class="clinical-badge">本次就诊</span></div>
          <dl class="clinical-encounter-meta"><div><dt>挂号单号</dt><dd>{{ register.registerNo || register.registerId }}</dd></div><div><dt>就诊日</dt><dd>{{ register.visitDate || '—' }}</dd></div><div><dt>挂号科室</dt><dd>{{ departmentLabel(register) }}</dd></div><div><dt>挂号医生</dt><dd>{{ doctorLabel(register) }}</dd></div></dl>
        </section>

        <p v-if="masterLoading" class="shell-status" role="status">正在读取开单数据…</p>
        <div v-else-if="masterError" class="banner clinical-read-error" role="alert" data-testid="order-master-error"><p>{{ masterError }}。读取成功后才能暂存或提交。</p><button class="button button-ghost button-compact" type="button" @click="loadMaster">重新读取开单数据</button></div>
        <p v-else-if="!orderDoctors.length" class="field-hint" role="status">暂无启用医生，请先维护基础数据。</p>

        <fieldset class="order-fields" :disabled="locked || busy || !canWrite">
          <div class="clinical-section-heading"><span class="clinical-section-number">01</span><div><h3>医生与诊断</h3><p>诊断建议填写，不强制</p></div><span class="clinical-recipe-tag">普通处方</span></div>
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
              <span>开单医生 <abbr class="req" title="必填">*</abbr></span>
              <select v-model="form.doctorId" data-testid="order-doctor">
                <option value="">请选择</option>
                <option v-for="item in orderDoctors" :key="item.doctorId" :value="item.doctorId">{{ item.name }}</option>
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

          <div class="clinical-section-heading clinical-line-heading"><span class="clinical-section-number">02</span><div><h3>处方明细</h3><p>选择药品或治疗项目，填写数量与用法</p></div><span class="clinical-count">{{ selectedLineCount }} 项</span></div>
          <div class="line-scroll clinical-table-scroll" tabindex="0" role="region" aria-label="处方明细，可左右滚动">
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
                  <th><span class="sr-only">操作</span></th>
                </tr>
              </thead>
              <tbody>
                <template v-for="(line, index) in lines" :key="line.key">
                  <tr>
                    <td>{{ index + 1 }}</td>
                    <td class="col-item">
                      <OrderItemFinder
                        v-model="line.itemNo"
                        :items="findItems"
                        :index="index"
                        :disabled="locked || busy || !canWrite"
                        :source-item="!mockMode ? line.sourceItem : null"
                      />
                    </td>
                    <td>{{ catalogOf(line.itemNo, line)?.spec || '—' }}</td>
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
                    <td>{{ catalogOf(line.itemNo, line)?.units || '—' }}</td>
                    <td>{{ catalogOf(line.itemNo, line) ? money(catalogOf(line.itemNo, line).price) : '—' }}</td>
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
                      <button class="text-btn text-btn-danger" type="button" :aria-label="`删除第 ${index + 1} 行`" @click="removeLine(line.key)">删除</button>
                    </td>
                  </tr>
                  <tr v-if="lineShort(line)" class="stock-row">
                    <td colspan="12">库存不足，仅提示，不拦截开单。</td>
                  </tr>
                </template>
              </tbody>
            </table>
          </div>
          <div class="order-common-bar" data-testid="order-common-bar">
            <span class="order-common-label">常用项目</span>
            <span class="order-common-honesty">演示短表（非个人收藏）</span>
            <button
              v-for="item in commonChips"
              :key="item.itemNo"
              type="button"
              class="order-common-chip"
              :data-testid="`order-common-item-${item.itemNo}`"
              :disabled="locked || busy || !canWrite || !item.resolved"
              :title="item.resolved ? ('填入 ' + item.label) : '当前目录不可用'"
              @click="applyCommonOrderItem(item.itemNo)"
            >
              {{ item.label }}
            </button>
          </div>
          <p v-if="commonMissingTip" class="field-hint" data-testid="order-common-missing" role="status">{{ commonMissingTip }}</p>
          <div class="panel-actions">
            <button class="button button-ghost button-compact" type="button" @click="addLine">添加明细</button>
          </div>
        </fieldset>

        <ul v-if="warnings.length" class="stock-list" data-testid="stock-tip">
          <li v-for="text in warnings" :key="text">{{ text }}</li>
        </ul>
        <p v-if="errorMessage && submitFeedback.state.status !== 'error'" class="banner" role="alert" data-testid="order-error">{{ errorMessage }}</p>
        <p v-if="infoMessage && submitFeedback.state.status !== 'success' && submitFeedback.state.status !== 'error'" class="ok-banner" role="status" data-testid="order-committed">{{ infoMessage }}</p>
        <SubmitStatusBanner :feedback="submitFeedback.state" />
        <OrderPrintPanel
          :preview-model="printPreviewModel"
          :print-status="printStatus"
          :locked="printLocked"
          :can-preview="printCanPreview"
          :honesty-note="PLACEHOLDER_HONESTY"
          :session-note="SESSION_PERSISTENCE_NOTE"
          :last-printed-at="printLastPrintedAt"
          :error-message="printErrorMessage"
          @preview="onPrintPreview"
          @print="onPrintOutput"
          @confirm="onPrintConfirm"
          @retry="onPrintRetry"
          @reprint="onPrintReprint"
          @close="onPrintClose"
        />

        <div class="clinical-order-actionbar">
          <p class="order-total" data-testid="order-total"><span>处方合计</span><span><strong>{{ money(totalFee) }}</strong> 元</span><span class="charge-chip" :data-charge="chargeText">{{ chargeText }}</span></p>
          <div class="panel-actions">
          <button v-if="!locked" class="button button-ghost button-compact" type="button" data-testid="order-save" :disabled="busy || !canWrite" @click="onSave">
            {{ saving ? '正在暂存…' : '暂存' }}
          </button>
          <button v-if="!locked" class="button button-compact" type="button" data-testid="order-commit" :disabled="busy || !canWrite" @click="onCommit">
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
        </div>
      </form>

      <aside class="patient-panel order-side clinical-order-history">
        <div class="panel-head">
          <div><span class="clinical-kicker">处方历史</span><h2>本挂号处方</h2></div><span class="clinical-count">{{ orders.length }}</span>
        </div>
        <p v-if="ordersLoading" class="shell-status" role="status">正在读取处方列表…</p>
        <div v-else-if="ordersError" class="banner clinical-read-error" role="alert"><p>{{ ordersError }}。当前处方状态已保留，请重试列表。</p><button class="button button-ghost button-compact" type="button" @click="refreshOrders">重试处方列表</button></div>
        <BusinessEmptyState v-else-if="ordersReady && orders.length === 0" title="尚无处方" description="这张挂号还没有处方。暂存后可在此切换查看。" icon="order" />
        <ul v-if="orders.length" class="order-list">
          <li v-for="item in orders" :key="item.orderId">
            <button class="order-list-item" :class="{ 'is-active': item.orderId === orderId }" type="button" :disabled="busy" :aria-pressed="item.orderId === orderId" @click="openListed(item.orderId)">
              <span class="clinical-history-no">{{ item.orderNo }}</span>
              <span class="clinical-history-bottom"><span class="ticket-status" :data-status="item.status">{{ orderStatusLabel(item.status) }}</span><strong><small>¥</small> {{ money(item.totalFee) }}</strong></span>
              <span class="clinical-history-charge">{{ orderChargeText(item) }}</span>
            </button>
          </li>
        </ul>
      </aside>
    </div>
  </section>
</template>

<style scoped>
.order-common-bar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  margin: 8px 0 4px;
}
.order-common-label {
  color: #84a18f;
  font-size: 10px;
  letter-spacing: 0.04em;
}
.order-common-honesty {
  color: #9aa8a2;
  font-size: 10px;
}
.order-common-chip {
  border: 0;
  padding: 2px 0;
  color: #1e6e5d;
  background: transparent;
  font: inherit;
  font-size: 11px;
  cursor: pointer;
}
.order-common-chip:disabled {
  color: #a0b0ab;
  cursor: default;
}
</style>
