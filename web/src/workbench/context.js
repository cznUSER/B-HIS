// 2026-10-14 Preserve only known business identifiers between outpatient pages.
export function outpatientTarget(module, source = {}, extra = {}) {
  const query = {}
  for (const key of ['patientId', 'registerId', 'orderId', 'charge_id', 'dispenseId']) {
    const raw = source[key] ?? (key === 'charge_id' ? source.chargeId : undefined)
    const value = Array.isArray(raw) ? raw[0] : raw
    if (typeof value === 'string' && value.trim()) query[key] = value.trim()
  }
  return { name: 'outpatient-module', params: { module }, query: { ...query, ...extra } }
}
