// 2026-10-13 P1-H05-MASTER-UI 基础数据入口：系统账号、SYSTEM、admin 权限可进。

export function canMaintainMaster(user) {
  if (!user || typeof user !== 'object') return false
  const hasPermissionData = user.accessControlLoaded === true
    || Object.prototype.hasOwnProperty.call(user, 'permissions')
    || Object.prototype.hasOwnProperty.call(user, 'scopes')
    || Object.prototype.hasOwnProperty.call(user, 'deniedPermissions')
  if (hasPermissionData) {
    const permissions = Array.isArray(user.permissions) ? user.permissions : []
    const denied = Array.isArray(user.deniedPermissions) ? user.deniedPermissions : []
    const scopes = Array.isArray(user.scopes) ? user.scopes.filter((scope) => typeof scope === 'string') : []
    const masterScoped = scopes.length === 0 || scopes.some((scope) => scope.split(':', 1)[0] === '' || scope.split(':', 1)[0] === 'master')
    return masterScoped && !denied.includes('admin:*') && !denied.some((code) => code.startsWith('admin:')) && (permissions.includes('admin:*') || permissions.some((code) => code.startsWith('admin:')))
  }
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
