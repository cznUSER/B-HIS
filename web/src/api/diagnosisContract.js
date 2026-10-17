// 2026-10-15 P2-DIAGNOSIS-UI 诊断最小实体契约：挂在挂号下，可选保存、提交、作废。
import { apiError } from './errors.js'

const STATUSES = new Set(['NEW', 'SAVE', 'COMMIT', 'ABATE'])

function text(value) {
  return typeof value === 'string' ? value.trim() : ''
}

export function assertCreateDiagnoseRequest(input) {
  const source = input && typeof input === 'object' ? input : {}
  const content = text(source.content)
  if (!content) throw apiError(400, 'VALIDATION_ERROR', '请输入诊断内容')
  if (content.length > 2000) throw apiError(400, 'VALIDATION_ERROR', '诊断内容不能超过 2000 个字符')
  return { content }
}

export function assertDiagnoseDto(body) {
  if (!body || typeof body !== 'object') throw apiError(502, 'HTTP_ERROR', '诊断记录不符合契约')
  const diagnoseId = text(body.diagnoseId)
  const encounterId = text(body.encounterId || body.registerId)
  const status = text(body.status)
  const content = text(body.content)
  if (!diagnoseId || !encounterId || !STATUSES.has(status)) {
    throw apiError(502, 'HTTP_ERROR', '诊断记录不符合契约')
  }
  return { diagnoseId, encounterId, status, content }
}

export function assertDiagnoseList(body) {
  if (!body || typeof body !== 'object' || !Array.isArray(body.items)) {
    throw apiError(502, 'HTTP_ERROR', '诊断列表不符合契约')
  }
  return { items: body.items.map(assertDiagnoseDto) }
}

export function diagnoseStatusLabel(status) {
  return {
    NEW: '新建',
    SAVE: '已暂存',
    COMMIT: '已提交',
    ABATE: '已作废',
  }[status] || status || '—'
}

export function diagnoseCanCommit(status) {
  return status === 'SAVE' || status === 'NEW'
}

export function diagnoseCanAbate(status) {
  return status === 'SAVE' || status === 'COMMIT'
}
