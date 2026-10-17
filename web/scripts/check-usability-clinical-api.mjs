// 2026-10-14 P1-H05-WEB-USABILITY: isolated HTTP fixture; no business service or DB.
import assert from 'node:assert/strict'
import http from 'node:http'
import { saveOrder } from '../src/api/orderApi.js'

const liveCatalog = [
  { itemNo: 'DRUG-A', itemType: 'DRUG', itemName: '真实同编码药品', spec: '5mg', units: '瓶', price: 17.75, status: 'ACTIVE' },
  { itemNo: 'LIVE-NEW', itemType: 'TREAT', itemName: '真实新增治疗', units: '项', price: 6.25, status: 'ACTIVE' },
]
const writes = []
let catalogFails = false
const server = http.createServer(async (req, res) => {
  const url = new URL(req.url, 'http://localhost')
  const json = (status, data) => { res.writeHead(status, { 'Content-Type': 'application/json' }); res.end(JSON.stringify(data)) }
  if (req.method === 'GET' && url.pathname === '/api/v1/catalog-items') {
    if (catalogFails) json(503, { error: { code: 'HTTP_ERROR', message: '目录不可用' } })
    else json(200, { items: liveCatalog })
    return
  }
  if (['POST', 'PATCH'].includes(req.method) && url.pathname.startsWith('/api/v1/orders')) {
    const chunks = []; for await (const chunk of req) chunks.push(chunk)
    const body = JSON.parse(Buffer.concat(chunks).toString())
    writes.push(body)
    json(req.method === 'POST' ? 201 : 200, {
      orderId: 'order-live', orderNo: 'R-live', registerId: 'reg-live', patientId: 'patient-live', departmentId: 'dept-live', doctorId: 'doc-live',
      status: 'SAVE', totalFee: 41.75, charged: false, items: body.items,
    })
    return
  }
  json(404, { error: { code: 'NOT_FOUND', message: 'fixture route missing' } })
})
await new Promise((resolve) => server.listen(0, '127.0.0.1', resolve))
process.env.VITE_USE_MOCK = 'false'
process.env.VITE_API_BASE_URL = `http://127.0.0.1:${server.address().port}/api/v1`
const body = { registerId: 'reg-live', doctorId: 'doc-live', items: [{ itemNo: 'DRUG-A', amount: '2' }, { itemNo: 'LIVE-NEW', amount: 1 }] }
try {
  await saveOrder('token-live', body, liveCatalog)
  assert.deepEqual(writes[0].items.map(({ itemNo, itemType, itemName, units, price, totalFee }) => ({ itemNo, itemType, itemName, units, price, totalFee })), [
    { itemNo: 'DRUG-A', itemType: 'DRUG', itemName: '真实同编码药品', units: '瓶', price: 17.75, totalFee: 35.5 },
    { itemNo: 'LIVE-NEW', itemType: 'TREAT', itemName: '真实新增治疗', units: '项', price: 6.25, totalFee: 6.25 },
  ])
  console.log('PASS explicit live catalog supplies request name, price, units and new codes')
  await saveOrder('token-live', { ...body, orderId: 'order-live' })
  assert.equal(writes[1].items[0].price, 17.75)
  assert.equal(writes[1].items[1].itemNo, 'LIVE-NEW')
  assert.equal(writes[1].registerId, undefined)
  console.log('PASS real patch reads live catalog when caller omits snapshot')
  catalogFails = true
  const previousWrites = writes.length
  await assert.rejects(() => saveOrder('token-live', body), (error) => error.status === 503 && error.message === '目录不可用')
  assert.equal(writes.length, previousWrites)
  console.log('PASS failed catalog read blocks real write without mock fallback')
  await assert.rejects(() => saveOrder('token-live', body, []), (error) => error.code === 'CATALOG_ITEM_NOT_FOUND')
  assert.equal(writes.length, previousWrites)
  console.log('PASS empty live catalog cannot fall back to mock codes')
} finally { await new Promise((resolve) => server.close(resolve)) }
