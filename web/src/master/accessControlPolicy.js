const ACCESS_TABS = [
  { name: 'access-users', label: '用户账号', icon: 'users', capability: 'users' },
  { name: 'access-roles', label: '角色与范围', icon: 'overview', capability: 'roles' },
  { name: 'access-permissions', label: '权限目录', icon: 'dicts', capability: 'permissions' },
  { name: 'access-user-logs', label: '操作日志', icon: 'overview', capability: 'logs' },
]

const ROUTE_CAPABILITY = Object.freeze({
  'access-users': 'users',
  'access-roles': 'roles',
  'access-permissions': 'permissions',
  'access-user-logs': 'logs',
})

export function accessControlCapabilities(user) {
  const roles = Array.isArray(user?.roles) ? user.roles : []
  const permissions = new Set(Array.isArray(user?.permissions) ? user.permissions : [])
  const denied = new Set(Array.isArray(user?.deniedPermissions) ? user.deniedPermissions : [])
  const hasPermissionData = user && typeof user === 'object' && (
    user.accessControlLoaded === true
    || Object.prototype.hasOwnProperty.call(user, 'permissions')
    || Object.prototype.hasOwnProperty.call(user, 'scopes')
    || Object.prototype.hasOwnProperty.call(user, 'deniedPermissions')
  )
  const legacySystem = !hasPermissionData && roles.some((role) => String(role).trim().toUpperCase() === 'SYSTEM')
  const elevated = user?.super === true || user?.developer === true || permissions.has('admin:*') || legacySystem
  const scopes = Array.isArray(user?.scopes) ? user.scopes.filter((scope) => typeof scope === 'string') : []
  const settingsScope = !hasPermissionData || scopes.length === 0 || scopes.some((scope) => {
    const module = scope.split(':', 1)[0]
    return module === '' || module === 'master'
  })
  const allows = (code) => {
    if (!settingsScope || denied.has('admin:*') || denied.has(code)) return false
    if (elevated) return true
    return permissions.has(code) || permissions.has('admin:*')
  }

  return {
    users: allows('admin:user'),
    roles: allows('admin:role'),
    permissions: allows('admin:permission'),
    logs: allows('audit:read'),
  }
}

export function visibleAccessTabs(user) {
  const capabilities = accessControlCapabilities(user)
  return ACCESS_TABS.filter((tab) => capabilities[tab.capability])
}

export function canOpenAccessRoute(user, routeName) {
  const capability = ROUTE_CAPABILITY[routeName]
  return Boolean(capability && accessControlCapabilities(user)[capability])
}

export function accessDeniedTitle(routeName) {
  return {
    'access-users': '需要用户管理权限',
    'access-roles': '需要角色管理权限',
    'access-permissions': '需要权限目录管理权限',
    'access-user-logs': '需要审计读取权限',
  }[routeName] || '当前账号无权查看此页'
}
