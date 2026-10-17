// 2026-10-14 P1-H05-WEB-USABILITY: mount real master SFCs and exercise their rendered controls.
// Only the network boundary is replaced. No service, database, browser package, or source-string assertions.
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { resolve, dirname } from 'node:path'
import { fileURLToPath, pathToFileURL } from 'node:url'
import { parse, compileScript, compileTemplate } from '@vue/compiler-sfc'
import { createRenderer, markRaw, nextTick } from 'vue'
import { session } from '../src/auth/session.js'

const webRoot = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const vueUrl = pathToFileURL(resolve(webRoot, 'node_modules/vue/dist/vue.runtime.esm-bundler.js')).href
const componentUrls = new Map()

function compileComponent(path) {
  if (componentUrls.has(path)) return componentUrls.get(path)
  const { descriptor } = parse(readFileSync(path, 'utf8'), { filename: path })
  const script = compileScript(descriptor, { id: path })
  const template = compileTemplate({
    id: path, filename: path, source: descriptor.template.content,
    // Keep static nodes in the same host path as interactive controls. The
    // memory renderer intentionally models element operations only, so this
    // prevents compiler hoisting from requiring a browser-only static host.
    compilerOptions: { bindingMetadata: script.bindings, hoistStatic: false },
  })
  assert.deepEqual(template.errors, [], `${path}: template compiles`)
  let code = `${script.content.replace('export default', 'const component =')}\n${template.code}\ncomponent.render = render; export default component`
  code = code.replace(/from (['"])([^'"]+)\1/g, (_, quote, specifier) => {
    const url = specifier === 'vue'
      ? vueUrl
      : specifier.endsWith('.vue')
        ? compileComponent(resolve(dirname(path), specifier))
        : pathToFileURL(resolve(dirname(path), specifier)).href
    return `from ${quote}${url}${quote}`
  })
  const url = `data:text/javascript;base64,${Buffer.from(code).toString('base64')}`
  componentUrls.set(path, url)
  return url
}

let activeElement = null
class Element {
  constructor(tag = '#text', text = '') {
    this.tag = tag
    this.tagName = tag.toUpperCase()
    this.text = text
    this.children = []
    this.props = {}
    this.parent = null
    this.listeners = {}
    this.value = ''
    // DOM elements are not reactive; give the memory host the same identity semantics.
    markRaw(this)
  }
  addEventListener(name, listener) { this.listeners[name] = listener }
  removeEventListener(name) { delete this.listeners[name] }
  setAttribute(name, value) { this.props[name] = value }
  getAttribute(name) { return this.props[name] }
  hasAttribute(name) { return name in this.props }
  focus() { activeElement = this }
  getRootNode() { return globalThis.document }
  get isConnected() { return Boolean(this.parent) }
  get options() { return this.children.filter((child) => child.tag === 'option') }
  querySelectorAll(selector) {
    const selectors = selector.split(',').map((item) => item.trim())
    return descendants(this).filter((child) => !child.props.disabled && selectors.some((item) => item === child.tag || item.startsWith(`${child.tag}:`) || (item === '[tabindex="0"]' && child.props.tabindex === '0')))
  }
  querySelector(selector) { return this.querySelectorAll(selector)[0] || null }
}

const browserListeners = new Map()
globalThis.window = {
  addEventListener(name, listener) { browserListeners.set(name, listener) },
  removeEventListener(name) { browserListeners.delete(name) },
}
globalThis.Document = class Document {}
globalThis.document = new class extends Document { get activeElement() { return activeElement } }()

const renderer = createRenderer({
  createElement: (tag) => new Element(tag),
  createText: (value) => new Element('#text', value),
  createComment: (value) => new Element('#comment', value),
  setText: (node, value) => { node.text = value },
  setElementText: (node, value) => { node.text = value; node.children = [] },
  patchProp: (node, key, previous, next) => { node.props[key] = next; if (key === 'value') node.value = next },
  insert(node, parent, anchor = null) {
    if (node.parent) node.parent.children.splice(node.parent.children.indexOf(node), 1)
    const index = anchor ? parent.children.indexOf(anchor) : -1
    parent.children.splice(index < 0 ? parent.children.length : index, 0, node)
    node.parent = parent
  },
  remove(node) { if (node.parent) node.parent.children.splice(node.parent.children.indexOf(node), 1); node.parent = null },
  parentNode: (node) => node.parent,
  nextSibling: (node) => node.parent?.children[node.parent.children.indexOf(node) + 1] || null,
})

function descendants(node) { return [node, ...node.children.flatMap(descendants)] }
function textOf(node) { return node.tag === '#comment' ? '' : node.text + node.children.map(textOf).join('') }
function find(root, predicate) { return descendants(root).find(predicate) }
function byId(root, id) { return find(root, (node) => node.props['data-testid'] === id) }
function classNodes(root, name) { return descendants(root).filter((node) => String(node.props.class || '').split(' ').includes(name)) }
async function settle() { for (let i = 0; i < 16; i += 1) { await new Promise((resolve) => setImmediate(resolve)); await nextTick() } }
async function click(node) {
  assert.ok(node, 'rendered control exists')
  assert.ok(!node.props.disabled, `${textOf(node)} is enabled`)
  node.focus()
  await node.props.onClick({ target: node, currentTarget: node, preventDefault() {}, stopPropagation() {} })
  await settle()
}
async function submit(node) { await node.props.onSubmit({ target: node, preventDefault() {}, stopPropagation() {} }); await settle() }
function setModel(node, value) { assert.ok(node); node.props['onUpdate:modelValue'](value) }
async function fillCreate(root, prefix) {
  const fields = {
    dept: { 'dept-code': 'DEPT9', 'dept-name': '测试科室' },
    doctor: { 'doctor-code': 'DOC9', 'doctor-name': '测试医生', 'doctor-dept': 'dept-1' },
    catalog: { 'catalog-no': 'DRUG-X', 'catalog-name': '测试药品', 'catalog-units': '盒', 'catalog-price': '2' },
    dict: { 'dict-code': 'expert', 'dict-name': '专家号' },
    user: { 'user-name': 'testuser', 'user-display': '测试人员' },
  }
  for (const [id, value] of Object.entries(fields[prefix])) setModel(byId(root, id), value)
  await nextTick()
}

const fixtures = {
  '/departments': { items: [{ departmentId: 'dept-1', deptCode: 'DEPT1', deptName: '内科', status: 'ACTIVE', sortNo: 1 }] },
  '/doctors': { items: [{ doctorId: 'doc-1', doctorCode: 'DOC1', doctorName: '周医生', departmentId: 'dept-1', status: 'ACTIVE' }] },
  '/catalog-items': { items: [{ catalogItemId: 'cat-1', itemNo: 'DRUG-A', itemName: '药品 A', itemType: 'DRUG', units: '盒', price: 2, status: 'ACTIVE', controlledFlag: false, isControlled: false }] },
  '/inventory/items': { items: [{ skuId: 'sku-a', itemNo: 'DRUG-A', itemName: '药品 A', qtyOnHand: 100, units: '盒', status: 'ACTIVE', rowVersion: '7' }] },
  '/inventory/batches': {
    items: [
      { batchId: 'P2-6A-EXPIRED', skuId: 'sku-a', itemNo: 'DRUG-A', itemName: '药品 A', storageUnitId: 'fixture-pharmacy', batchNo: 'B-EXPIRED', productDate: '2025-01-01', expireDate: '2026-10-14', qtyOnHand: '3.000', units: '盒', expiryStatus: 'EXPIRED', rowVersion: 'fixture-1' },
      { batchId: 'P2-6A-EXPIRING', skuId: 'sku-a', itemNo: 'DRUG-A', itemName: '药品 A', storageUnitId: 'fixture-pharmacy', batchNo: 'B-EXPIRING', productDate: '2026-01-01', expireDate: '2026-10-15', qtyOnHand: '1.000', units: '盒', expiryStatus: 'EXPIRING', rowVersion: 'fixture-2' },
      { batchId: 'P2-6A-EXPIRING-END', skuId: 'sku-a', itemNo: 'DRUG-A', itemName: '药品 A', storageUnitId: 'fixture-pharmacy', batchNo: 'B-EXPIRING-END', productDate: '2026-01-01', expireDate: '2026-11-14', qtyOnHand: '4.000', units: '盒', expiryStatus: 'EXPIRING', rowVersion: 'fixture-3' },
      { batchId: 'P2-6A-VALID', skuId: 'sku-a', itemNo: 'DRUG-A', itemName: '药品 A', storageUnitId: 'fixture-pharmacy', batchNo: 'B-VALID', productDate: '2026-01-01', expireDate: '2026-11-15', qtyOnHand: '10.000', units: '盒', expiryStatus: 'VALID', rowVersion: 'fixture-4' },
      { batchId: 'P2-6A-UNKNOWN', skuId: 'sku-a', itemNo: 'DRUG-A', itemName: '药品 A', storageUnitId: 'fixture-pharmacy', batchNo: 'B-UNKNOWN', productDate: null, expireDate: null, qtyOnHand: '7.000', units: '盒', expiryStatus: 'UNKNOWN', rowVersion: 'fixture-5' },
    ],
    summary: { batchCount: 5, qtyTotal: '25.000', qtyUnknown: '7.000', duplicateKeyCount: 0 },
    asOfDate: '2026-10-15', expiryWindowDays: 30, nextCursor: '',
  },
  '/dicts': { readonlyEnums: [{ dictType: 'sex', values: ['M', 'F', 'U'] }], maintainableTypes: [{ dictType: 'register_sort', name: '号别' }] },
  '/dicts/register_sort/items': { items: [{ dictItemId: 'dict-1', dictType: 'register_sort', code: 'normal', name: '普通号', status: 'ACTIVE', sortNo: 1 }] },
  '/users': { items: [{ userId: 'u-1', username: 'reg1', displayName: '挂号甲', roles: ['REGISTRAR'], status: 'ACTIVE' }] },
}
const panels = [
  ['DepartmentsPanel', 'dept', '/departments'],
  ['DoctorsPanel', 'doctor', '/doctors'],
  ['CatalogPanel', 'catalog', '/catalog-items'],
  ['InventoryPanel', 'stock', '/inventory/items'],
  ['DictsPanel', 'dict', '/dicts/register_sort/items'],
  ['UsersPanel', 'user', '/users'],
]
let failPath = ''
let returnEmpty = false
let forbidBatch = false
let batchPagination = false
let deferRead = null
let deferWrite = null
const calls = []
const originalFetch = globalThis.fetch
globalThis.fetch = async (url, options) => {
  const requestUrl = new URL(url, 'http://master.test')
  const path = requestUrl.pathname.replace('/api/v1', '')
  calls.push({ path, method: options.method, body: options.body, search: requestUrl.search })
  if (options.method === 'GET') {
    if (deferRead) await deferRead
    if (path === failPath) return new Response(JSON.stringify({ error: { code: 'SERVICE_UNAVAILABLE', message: '测试读取失败' } }), { status: 503 })
    if (path === '/inventory/batches' && forbidBatch) return new Response(JSON.stringify({ error: { code: 'FORBIDDEN', message: '没有批次查询权限' } }), { status: 403 })
    if (path === '/inventory/ledger') {
      const second = requestUrl.searchParams.get('cursor') === 'cursor-2'
      const rowNo = second ? '2' : '1'
      return new Response(JSON.stringify({
        items: [{ ledgerId: `ledger-${rowNo}`, skuId: 'sku-a', skuCode: 'DRUG-A', bizType: second ? 'DISPENSE' : 'ADJUST', bizId: `biz-${rowNo}`, qtyDelta: second ? -1 : 2, qtyAfter: second ? 101 : 102, remark: `流水${rowNo}`, operatorName: '测试人员', createdAt: `2026-10-1${rowNo}T00:00:00Z` }],
        nextCursor: second ? '' : 'cursor-2',
      }), { status: 200 })
    }
    if (path === '/inventory/batches') {
      const status = requestUrl.searchParams.get('expiryStatus') || ''
      const skuCode = requestUrl.searchParams.get('skuCode') || ''
      const items = fixtures[path].items.filter((item) => (!status || item.expiryStatus === status) && (!skuCode || item.itemNo === skuCode || item.skuId === skuCode))
      if (batchPagination) {
        const second = requestUrl.searchParams.get('cursor') === 'batch-cursor-2'
        const pageItems = second ? items.slice(2) : items.slice(0, 2)
        const pageSummary = pageItems.reduce((result, item) => {
          result.batchCount += 1
          result.qtyTotal = (Number(result.qtyTotal) + Number(item.qtyOnHand)).toFixed(3)
          if (item.expiryStatus === 'UNKNOWN') result.qtyUnknown = (Number(result.qtyUnknown) + Number(item.qtyOnHand)).toFixed(3)
          return result
        }, { batchCount: 0, qtyTotal: '0.000', qtyUnknown: '0.000', duplicateKeyCount: 0 })
        return new Response(JSON.stringify({ items: pageItems, summary: pageSummary, asOfDate: requestUrl.searchParams.get('asOfDate'), expiryWindowDays: Number(requestUrl.searchParams.get('expiryWindowDays')), nextCursor: second ? '' : 'batch-cursor-2' }), { status: 200 })
      }
      const summary = items.reduce((result, item) => {
        result.batchCount += 1
        result.qtyTotal = (Number(result.qtyTotal) + Number(item.qtyOnHand)).toFixed(3)
        if (item.expiryStatus === 'UNKNOWN') result.qtyUnknown = (Number(result.qtyUnknown) + Number(item.qtyOnHand)).toFixed(3)
        return result
      }, { batchCount: 0, qtyTotal: '0.000', qtyUnknown: '0.000', duplicateKeyCount: 0 })
      return new Response(JSON.stringify({ items, summary, asOfDate: requestUrl.searchParams.get('asOfDate'), expiryWindowDays: Number(requestUrl.searchParams.get('expiryWindowDays')), nextCursor: '' }), { status: 200 })
    }
    const body = returnEmpty && fixtures[path]?.items ? { items: [] } : fixtures[path]
    return new Response(JSON.stringify(body), { status: 200 })
  }
  if (deferWrite) await deferWrite
  if (path.endsWith('reset-password')) return new Response(null, { status: 204 })
  const base = Object.keys(fixtures).find((key) => path === key || path.startsWith(`${key}/`))
  const saved = { ...(fixtures[base]?.items?.[0] || {}), ...JSON.parse(options.body || '{}') }
  return new Response(JSON.stringify(saved), { status: options.method === 'POST' && !path.endsWith('/adjust') ? 201 : 200 })
}
process.env.VITE_USE_MOCK = 'false'
session.accessToken = 'test-token'
session.user = { displayName: '测试人员' }

async function mount(name) {
  const component = (await import(compileComponent(resolve(webRoot, `src/views/master/${name}.vue`)))).default
  const root = new Element('root')
  const app = renderer.createApp(component)
  app.mount(root)
  return { root, app }
}

const failures = []
let passed = 0
async function check(name, fn) {
  await settle()
  failPath = ''; returnEmpty = false; forbidBatch = false; deferRead = null; deferWrite = null; calls.length = 0
  try { await fn(); passed += 1; console.log(`PASS ${name}`) }
  catch (error) { failures.push({ name, error }); console.error(`FAIL ${name}: ${error.message}`) }
}

try {
  for (const [name, prefix, path] of panels) {
    await check(`${prefix}: initial loading exposes progress and blocks maintenance`, async () => {
      let release
      deferRead = new Promise((resolve) => { release = resolve })
      const { root, app } = await mount(name)
      try {
        await settle()
        assert.ok(find(root, (node) => node.props.role === 'status' && textOf(node).includes('正在读取')))
        assert.equal(byId(root, `${prefix}-list`), undefined)
        if (prefix !== 'stock') assert.equal(byId(root, `${prefix}-create`).props.disabled, true)
        release(); await settle()
        assert.ok(byId(root, `${prefix}-list`))
      } finally { release(); app.unmount() }
    })
    await check(`${prefix}: initial read failure is an error, never empty, and retry restores actions`, async () => {
      failPath = path
      const { root, app } = await mount(name)
      try {
        await settle()
        assert.equal(classNodes(root, 'result-empty').filter((node) => /暂无|无可维护|没有符合/.test(textOf(node))).length, 0, 'failed read must not render empty guidance')
        const error = byId(root, `${prefix}-load-error`)
        assert.ok(error && textOf(error).includes('测试读取失败'), 'explicit list error rendered')
        assert.ok(byId(root, `${prefix}-retry`), 'retry is available')
        if (prefix !== 'stock') assert.equal(byId(root, `${prefix}-create`).props.disabled, true, 'create disabled after failed read')
        failPath = ''
        await click(byId(root, `${prefix}-retry`))
        assert.ok(byId(root, `${prefix}-list`), 'successful retry renders rows')
        assert.equal(byId(root, `${prefix}-load-error`), undefined, 'successful retry clears error')
        if (prefix !== 'stock') assert.equal(byId(root, `${prefix}-create`).props.disabled, false, 'create enabled only after success')
      } finally { app.unmount() }
    })
    await check(`${prefix}: successful zero-row response renders genuine empty state`, async () => {
      returnEmpty = true
      const { root, app } = await mount(name)
      try {
        await settle()
        assert.ok(classNodes(root, 'result-empty').some((node) => /暂无|无可维护|没有符合/.test(textOf(node))), 'successful empty response has empty guidance')
        assert.equal(byId(root, `${prefix}-load-error`), undefined)
      } finally { app.unmount() }
    })
  }
  for (const [name, prefix, path] of panels.filter(([, prefix]) => prefix !== 'stock')) {
    await check(`${prefix}: refresh invalidates an open editor until the list succeeds`, async () => {
      const { root, app } = await mount(name)
      try {
        await settle()
        await click(byId(root, `${prefix}-create`))
        await fillCreate(root, prefix)
        failPath = path
        const query = find(root, (node) => node.tag === 'form' && String(node.props.class).includes('master-filters'))
        if (query) await submit(query)
        else {
          // Dicts has no search form; successful save initiates the same reload.
          await submit(byId(root, `${prefix}-form`))
        }
        if (prefix !== 'dict') {
          assert.equal(byId(root, `${prefix}-save`).props.disabled, true, 'open editor cannot submit after list failure')
          const writesBefore = calls.filter((call) => call.method !== 'GET').length
          await submit(byId(root, `${prefix}-form`))
          assert.equal(calls.filter((call) => call.method !== 'GET').length, writesBefore, 'Enter cannot bypass disabled maintenance')
        }
        assert.equal(byId(root, `${prefix}-create`).props.disabled, true)
        assert.equal(byId(root, `${prefix}-list`), undefined, 'failed refresh does not expose stale row actions')
      } finally { app.unmount() }
    })
    await check(`${prefix}: successful save stays successful when refresh fails`, async () => {
      const { root, app } = await mount(name)
      try {
        await settle()
        await click(byId(root, `${prefix}-create`))
        await fillCreate(root, prefix)
        failPath = path
        await submit(byId(root, `${prefix}-form`))
        const notice = byId(root, `${prefix}-notice`)
        assert.ok(notice && /已保存|已新建/.test(textOf(notice)), 'save success notice survives refresh failure')
        assert.equal(notice.props.role, 'status', 'successful write is not reported as failure')
        assert.ok(byId(root, `${prefix}-load-error`), 'refresh failure is independently visible')
        assert.equal(byId(root, `${prefix}-create`).props.disabled, true)
      } finally { app.unmount() }
    })
  }
  for (const [name, prefix] of panels.filter(([, prefix]) => !['stock', 'dict'].includes(prefix))) {
    await check(`${prefix}: Enter in search cannot race an in-flight save`, async () => {
      let release
      const { root, app } = await mount(name)
      try {
        await settle()
        await click(byId(root, `${prefix}-create`))
        await fillCreate(root, prefix)
        deferWrite = new Promise((resolve) => { release = resolve })
        const saving = submit(byId(root, `${prefix}-form`))
        await settle()
        const readsBefore = calls.filter((call) => call.method === 'GET').length
        await submit(find(root, (node) => node.tag === 'form' && String(node.props.class).includes('master-filters')))
        assert.equal(calls.filter((call) => call.method === 'GET').length, readsBefore, 'query waits for the save to complete')
        release(); await saving
      } finally { release?.(); app.unmount() }
    })
  }
  await check('doctor: department read failure blocks maintenance and retry reloads dependency', async () => {
    failPath = '/departments'
    const { root, app } = await mount('DoctorsPanel')
    try {
      await settle()
      assert.equal(byId(root, 'doctor-create').props.disabled, true)
      assert.ok(byId(root, 'doctor-load-error'))
      failPath = ''
      await click(byId(root, 'doctor-retry'))
      assert.equal(byId(root, 'doctor-create').props.disabled, false)
    } finally { app.unmount() }
  })
  await check('dict: overview failure is not an empty maintainable dictionary', async () => {
    failPath = '/dicts'
    const { root, app } = await mount('DictsPanel')
    try {
      await settle()
      assert.equal(classNodes(root, 'result-empty').length, 0)
      assert.equal(byId(root, 'dict-create').props.disabled, true)
      failPath = ''
      await click(byId(root, 'dict-retry'))
      assert.ok(byId(root, 'enum-list') && byId(root, 'dict-list'))
    } finally { app.unmount() }
  })
  await check('stock: adjustment success survives failed list refresh', async () => {
    const { root, app } = await mount('InventoryPanel')
    try {
      await settle()
      await click(byId(root, 'stock-adjust-DRUG-A'))
      setModel(byId(root, 'adjust-delta'), '1')
      setModel(byId(root, 'adjust-remark'), '测试盘点')
      await nextTick()
      failPath = '/inventory/items'
      await submit(byId(root, 'adjust-dialog'))
      assert.ok(textOf(byId(root, 'stock-notice')).includes('已调整库存'))
      assert.equal(byId(root, 'stock-notice').props.role, 'status')
      assert.ok(byId(root, 'stock-load-error'))
      assert.equal(byId(root, 'adjust-dialog'), undefined)
    } finally { app.unmount() }
  })
  await check('stock: ledger filters use SKU and calendar dates, and more appends the next page', async () => {
    const { root, app } = await mount('InventoryPanel')
    try {
      await settle()
      const ledgerButton = find(root, (node) => node.tag === 'button' && textOf(node).trim() === '流水')
      await click(ledgerButton)
      assert.equal(byId(root, 'ledger-list').children.filter((node) => node.tag === 'tbody')[0].children.filter((node) => node.tag === 'tr').length, 1)
      setModel(byId(root, 'ledger-biz-type'), 'ADJUST')
      setModel(byId(root, 'ledger-from'), '2026-10-01')
      setModel(byId(root, 'ledger-to'), '2026-10-15')
      await submit(byId(root, 'ledger-filters'))
      const filteredRequest = calls.find((call) => call.path === '/inventory/ledger' && call.search.includes('bizType=ADJUST'))
      assert.ok(filteredRequest.search.includes('skuCode=DRUG-A'), 'selected SKU code is sent')
      assert.ok(filteredRequest.search.includes('from=2026-10-01') && filteredRequest.search.includes('to=2026-10-15'), 'inclusive calendar dates are sent')
      await click(byId(root, 'ledger-more'))
      assert.ok(textOf(byId(root, 'ledger-list')).includes('流水2'), 'next page appends a row')
      assert.ok(calls.some((call) => call.path === '/inventory/ledger' && call.search.includes('cursor=cursor-2')), 'opaque cursor is sent on the next page')
      assert.equal(byId(root, 'ledger-more'), undefined, 'final page hides the more button')
    } finally { app.unmount() }
  })
  await check('stock: batch query sends read-only filters and renders four expiry states', async () => {
    const { root, app } = await mount('InventoryPanel')
    try {
      await settle()
      setModel(byId(root, 'batch-sku'), 'DRUG-A')
      setModel(byId(root, 'batch-status'), '')
      await submit(byId(root, 'batch-filters'))
      assert.ok(byId(root, 'batch-list'), 'batch table renders after explicit query')
      assert.equal(byId(root, 'batch-list').children.find((node) => node.tag === 'tbody').children.filter((node) => node.tag === 'tr').length, 5)
      assert.ok(textOf(byId(root, 'batch-list')).includes('已过期'))
      assert.ok(textOf(byId(root, 'batch-list')).includes('近效期'))
      assert.ok(textOf(byId(root, 'batch-list')).includes('有效'))
      assert.ok(textOf(byId(root, 'batch-list')).includes('未知效期'))
      assert.ok(textOf(byId(root, 'batch-summary')).includes('25'))
      const request = calls.find((call) => call.path === '/inventory/batches')
      assert.ok(request && request.search.includes('skuCode=DRUG-A'), 'SKU filter is sent')
      assert.ok(request.search.includes('asOfDate=2026-10-15') && request.search.includes('expiryWindowDays=30'), 'fixed reference inputs are sent')
      assert.equal(calls.filter((call) => call.path.includes('/inventory') && call.method !== 'GET').length, 0, 'batch query has no inventory write')
      setModel(byId(root, 'batch-status'), 'EXPIRED')
      await submit(byId(root, 'batch-filters'))
      assert.equal(byId(root, 'batch-list').children.find((node) => node.tag === 'tbody').children.filter((node) => node.tag === 'tr').length, 1, 'status filter narrows rows')
    } finally { app.unmount() }
  })
  await check('stock: batch pagination accumulates loaded rows and summary', async () => {
    const { root, app } = await mount('InventoryPanel')
    try {
      batchPagination = true
      setModel(byId(root, 'batch-sku'), 'DRUG-A')
      await submit(byId(root, 'batch-filters'))
      assert.equal(byId(root, 'batch-list').children.find((node) => node.tag === 'tbody').children.filter((node) => node.tag === 'tr').length, 2, 'first page renders two rows')
      assert.ok(textOf(byId(root, 'batch-summary')).includes('批次 2 条'))
      assert.ok(textOf(byId(root, 'batch-summary')).includes('4'))
      await click(byId(root, 'batch-more'))
      assert.equal(byId(root, 'batch-list').children.find((node) => node.tag === 'tbody').children.filter((node) => node.tag === 'tr').length, 5, 'second page appends three rows')
      assert.ok(textOf(byId(root, 'batch-summary')).includes('批次 5 条'))
      assert.ok(textOf(byId(root, 'batch-summary')).includes('25'))
      assert.ok(textOf(byId(root, 'batch-summary')).includes('未知效期 7'))
      assert.equal(byId(root, 'batch-more'), undefined, 'final page hides the more button')
      assert.equal(calls.filter((call) => call.path === '/inventory/batches' && call.method !== 'GET').length, 0, 'batch pagination has no inventory write')
    } finally { batchPagination = false; app.unmount() }
  })
  await check('stock: batch query without permission stays an explicit error', async () => {
    const { root, app } = await mount('InventoryPanel')
    try {
      await settle()
      forbidBatch = true
      await submit(byId(root, 'batch-filters'))
      assert.ok(byId(root, 'batch-load-error'))
      assert.ok(textOf(byId(root, 'batch-load-error')).includes('FORBIDDEN'))
      assert.equal(byId(root, 'batch-list'), undefined)
    } finally { app.unmount() }
  })
  for (const [name, prefix] of panels) {
    await check(`${prefix}: data tables expose named keyboard scroll regions`, async () => {
      const { root, app } = await mount(name)
      try {
        await settle()
        const regions = classNodes(root, 'result-wrap')
        assert.ok(regions.length > 0)
        for (const region of regions) {
          assert.equal(region.props.tabindex, '0', 'table can receive keyboard focus')
          assert.equal(region.props.role, 'region')
          assert.ok(region.props['aria-label'], 'scroll region has an accessible name')
        }
        for (const th of descendants(root).filter((node) => node.tag === 'th')) assert.equal(th.props.scope, 'col')
      } finally { app.unmount() }
    })
  }
  await check('stock: adjust dialog receives focus, traps Tab, and Escape restores trigger', async () => {
    const { root, app } = await mount('InventoryPanel')
    try {
      await settle()
      const trigger = byId(root, 'stock-adjust-DRUG-A')
      await click(trigger)
      const dialog = byId(root, 'adjust-dialog')
      assert.ok(descendants(dialog).includes(activeElement), 'opening dialog moves focus inside it')
      const controls = descendants(dialog).filter((node) => ['button', 'input'].includes(node.tag) && !node.props.disabled)
      controls.at(-1).focus()
      let prevented = false
      browserListeners.get('keydown')({ key: 'Tab', shiftKey: false, preventDefault() { prevented = true } })
      assert.ok(prevented, 'Tab at final control is contained')
      assert.ok(activeElement === controls[0], 'Tab wraps to first control')
      browserListeners.get('keydown')({ key: 'Escape', preventDefault() {} })
      await settle()
      assert.equal(byId(root, 'adjust-dialog'), undefined)
      assert.ok(activeElement === trigger, 'closing restores the adjustment trigger')
    } finally { app.unmount() }
  })
  await check('stock: focus stays in the adjustment dialog during an in-flight write', async () => {
    let release
    const { root, app } = await mount('InventoryPanel')
    try {
      await settle()
      await click(byId(root, 'stock-adjust-DRUG-A'))
      setModel(byId(root, 'adjust-delta'), '1')
      setModel(byId(root, 'adjust-remark'), '测试盘点')
      await nextTick()
      deferWrite = new Promise((resolve) => { release = resolve })
      const saving = submit(byId(root, 'adjust-dialog'))
      await settle()
      let prevented = false
      browserListeners.get('keydown')({ key: 'Tab', shiftKey: false, preventDefault() { prevented = true } })
      assert.ok(prevented, 'Tab cannot leave the busy modal')
      assert.ok(activeElement === byId(root, 'adjust-dialog'), 'busy modal remains focusable')
      release(); await saving
    } finally { release?.(); app.unmount() }
  })
} finally { globalThis.fetch = originalFetch }

console.log(`Master usability: ${passed} passed, ${failures.length} failed`)
if (failures.length) process.exitCode = 1
