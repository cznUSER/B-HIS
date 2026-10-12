// 2026-10-13 P1-H05-MASTER-UI 基础数据入口：系统账号、SYSTEM、admin 权限可进。

export function canMaintainMaster(user) {
  if (!user || typeof user !== 'object') return false
  const username = typeof user.username === 'string' ? user.username.trim().toLowerCase() : ''
  if (username === 'system') return true
  const roles = Array.isArray(user.roles) ? user.roles : []
  return roles.some((role) => {
    const text = String(role ?? '').trim()
    if (!text) return false
    const upper = text.toUpperCase()
    if (upper === 'SYSTEM' || upper === 'ADMIN') return true
    return text.toLowerCase().startsWith('admin:')
  })
}
