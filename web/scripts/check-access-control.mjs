import { readFileSync } from 'node:fs'
import { accessControlCapabilities, canOpenAccessRoute, visibleAccessTabs } from '../src/master/accessControlPolicy.js'
import { buildUserAccessPayload } from '../src/master/userAccessPayload.js'
import { buildAssignedRoleScopePayload, canEnableRoleScope, roleScopeDepartmentState } from '../src/master/userRoleScopes.js'
import { safeReturnTo } from '../src/router/returnTo.js'
import { mockLogin } from '../src/api/mockAuth.js'
import { saveUser } from '../src/master/api.js'

function assert(condition, message) {
  if (!condition) throw new Error(message)
}

const router = readFileSync(new URL('../src/router/index.js', import.meta.url), 'utf8')
const nav = readFileSync(new URL('../src/workbench/nav.js', import.meta.url), 'utf8')
const api = readFileSync(new URL('../src/master/api.js', import.meta.url), 'utf8')
const view = readFileSync(new URL('../src/views/AccessControlView.vue', import.meta.url), 'utf8')
const returnTo = readFileSync(new URL('../src/router/returnTo.js', import.meta.url), 'utf8')
const loginView = readFileSync(new URL('../src/views/LoginView.vue', import.meta.url), 'utf8')
const contract = readFileSync(new URL('../src/api/contract.js', import.meta.url), 'utf8')
const usersPanel = readFileSync(new URL('../src/views/access/UsersAccessPanel.vue', import.meta.url), 'utf8')
const passwordView = readFileSync(new URL('../src/views/PasswordChangeView.vue', import.meta.url), 'utf8')

assert(router.includes("path: 'settings/users'") && router.includes("name: 'access-users'"), 'access route')
assert(router.includes("name: 'access-roles'") && router.includes("name: 'access-permissions'") && router.includes("name: 'access-user-logs'"), 'access child routes')
assert(router.includes("name: 'master-users'") && router.includes("redirect: { name: 'access-users' }"), 'legacy users route redirects to full access workspace')
assert(router.includes("name: 'access-user-logs'") && returnTo.includes('roles|permissions|logs'), 'user operation log route and deep link')
assert(loginView.includes('safeReturnTo(route.query.redirect || saved, result.user)'), 'login deep link uses effective user access')
assert(view.includes('v-if="currentAllowed"') && view.includes('data-testid="access-denied"') && contract.includes('super: user.super === true') && contract.includes('developer: user.developer === true'), 'unauthorized view is not mounted and elevated claims are preserved')
assert(nav.includes("to: { name: 'access-users' }") && nav.includes("access: 'users'"), 'access nav')
for (const name of ['fetchAccessModules', 'fetchAccessCategories', 'fetchAccessModes', 'saveAccessModule', 'savePermissionCategory', 'savePermissionMode', 'fetchAccessPermissions', 'saveAccessPermission', 'fetchAccessRoles', 'saveAccessRole', 'fetchUserPermissionOverrides', 'saveUserPermissionOverrides']) {
  assert(api.includes(`export async function ${name}`), `${name} client function`)
}
assert(view.includes('角色权限采用并集') && view.includes('临床二次认证仍保持关闭'), 'access scope note')
assert(contract.includes('accessControlLoaded: user.accessControlLoaded === true'), 'effective access load marker is preserved')
const capabilityCases = [
  [{ permissions: ['admin:user'] }, ['access-users']],
  [{ permissions: ['admin:role'] }, ['access-roles']],
  [{ permissions: ['admin:permission'] }, ['access-permissions']],
  [{ permissions: ['audit:read'] }, ['access-user-logs']],
]
for (const [user, expected] of capabilityCases) {
  assert(JSON.stringify(visibleAccessTabs(user).map((tab) => tab.name)) === JSON.stringify(expected), `fine-grained tab visibility for ${expected[0]}`)
  assert(canOpenAccessRoute(user, expected[0]), `authorized direct route ${expected[0]}`)
  assert(!canOpenAccessRoute(user, 'access-users') || expected[0] === 'access-users', `other direct routes remain denied for ${expected[0]}`)
}
for (const user of [
  { super: true },
  { developer: true },
  { roles: ['SYSTEM'] },
  { permissions: ['admin:*'] },
]) {
  const all = accessControlCapabilities(user)
  assert(Object.values(all).every(Boolean) && visibleAccessTabs(user).length === 4, 'elevated access opens every section')
}
assert(visibleAccessTabs({ roles: ['SYSTEM'], permissions: [] }).length === 0, 'empty effective permissions do not fall back to SYSTEM role')
assert(visibleAccessTabs({ permissions: ['admin:*'], deniedPermissions: ['admin:role'] }).map((tab) => tab.name).join(',') === 'access-users,access-permissions,access-user-logs', 'explicit admin deny wins for settings tabs')
assert(visibleAccessTabs({ permissions: ['admin:user'], scopes: ['outpatient:dept-1'] }).length === 0, 'outpatient scope cannot open settings')
assert(!canOpenAccessRoute({ roles: ['ADMIN'] }, 'access-roles'), 'generic ADMIN role alone does not bypass capability permissions')
assert(safeReturnTo('/app/settings/users', { roles: ['REGISTRAR'], permissions: [], scopes: [], accessControlLoaded: true }) === '', 'unauthorized settings deep link is rejected')
assert(safeReturnTo('/app/settings/users/roles', { permissions: ['admin:role'], scopes: [], accessControlLoaded: true }) === '/app/settings/users/roles', 'authorized role settings deep link is preserved')
assert(safeReturnTo('/app/settings/users', { permissions: ['admin:role'], scopes: [], accessControlLoaded: true }) === '', 'settings deep link requires the matching capability')
const mockUser = await saveUser('', '', {
  username: 'access-contract-user', displayName: '权限契约用户', password: 'password1', roles: ['REGISTRAR'],
  roleScopes: [{ roleCode: 'REGISTRAR', departmentId: 'dept-1', status: 'ACTIVE' }],
  permissionOverrides: [{ permissionCode: 'patient:create', effect: 'DENY' }],
})
const mockSession = mockLogin({ username: mockUser.username, password: 'password1' }).user
assert(!mockSession.permissions.includes('patient:create') && mockSession.permissions.includes('encounter:create'), 'Mock login applies permission overrides')
assert(JSON.stringify(mockSession.scopes) === JSON.stringify(['outpatient:dept-1']), 'Mock login applies active role scopes')
const userForm = {
  username: 'ordinary', displayName: '普通用户', description: '', cardNo: 'CARD-1', inputCode: '',
  serviceStar: '', expandInterface1: 0, expandInterface2: 0, inputCodeCompareMode: 2,
  status: 'ACTIVE', roles: ['REGISTRAR'], mustChangePassword: true, cannotModifyPassword: false,
  accountLocked: false, accountStopped: false, hidden: false, system: false, super: false, developer: false,
}
const systemPayload = buildUserAccessPayload(userForm, { creating: false })
assert(!['hidden', 'system', 'developer', 'super', 'cardNo'].some((key) => Object.hasOwn(systemPayload, key)), 'SYSTEM-level editor omits protected fields')
const superPayload = buildUserAccessPayload(userForm, { canEditSuperFlag: true })
assert(superPayload.super === false && superPayload.cardNo === 'CARD-1' && !['hidden', 'system', 'developer'].some((key) => Object.hasOwn(superPayload, key)), 'super editor only sends super-level fields')
const developerPayload = buildUserAccessPayload(userForm, { canEditDeveloperFlags: true, canEditSuperFlag: true })
assert(['hidden', 'system', 'developer', 'super', 'cardNo'].every((key) => Object.hasOwn(developerPayload, key)), 'developer editor sends its authorized protected fields')
const multiRoleScopes = [
  { roleCode: 'REGISTRAR', departmentId: 'dept-1', status: 'ACTIVE' },
  { roleCode: 'REGISTRAR', departmentId: 'dept-2', status: 'DISABLED' },
  { roleCode: 'DOCTOR', departmentId: 'dept-3', status: 'ACTIVE' },
]
const scopeDepartments = [
  { departmentId: 'dept-1', deptName: '内科', status: 'ACTIVE' },
  { departmentId: 'dept-2', deptName: '外科', status: 'DISABLED' },
]
assert(roleScopeDepartmentState(multiRoleScopes[0], scopeDepartments).valid, 'active scope in active department remains valid')
assert(roleScopeDepartmentState(multiRoleScopes[1], scopeDepartments).label === '科室已停用 · 当前无效', 'active scope in disabled department is explicitly invalid')
assert(roleScopeDepartmentState({ ...multiRoleScopes[0], departmentId: 'missing' }, scopeDepartments).label === '当前无效 · 科室不在目录中', 'active scope missing from department directory is explicitly invalid')
assert(!canEnableRoleScope({ ...multiRoleScopes[1], status: 'DISABLED' }, scopeDepartments), 'disabled department scope cannot be re-enabled')
assert(JSON.stringify(buildAssignedRoleScopePayload(multiRoleScopes, ['REGISTRAR'])) === JSON.stringify(multiRoleScopes.slice(0, 2)), 'deselecting a role excludes its scopes from the full user payload')
const scopedPayload = buildUserAccessPayload({ ...userForm, roles: ['REGISTRAR', 'DOCTOR'] }, { roleScopes: multiRoleScopes })
assert(JSON.stringify(scopedPayload.roleScopes) === JSON.stringify(multiRoleScopes), 'payload preserves multiple departments per assigned role and each row status')
assert(!Object.hasOwn(systemPayload, 'roleScopes'), 'omitted roleScopes stays omitted for backward-compatible partial saves')
assert(JSON.stringify(buildUserAccessPayload(userForm, { roleScopes: [] }).roleScopes) === '[]', 'explicit empty roleScopes clears configured scopes')
assert(usersPanel.includes('buildUserAccessPayload(form') && usersPanel.includes('canEditDeveloperFlags.value') && usersPanel.includes('canEditSuperFlag.value'), 'user form uses permission-aware payload builder')
assert(usersPanel.includes('permissionOverrides') && !usersPanel.includes('saveUserPermissionOverrides') && usersPanel.includes('明确拒绝') && usersPanel.includes('科室范围'), 'user permission bundle overrides')
assert(usersPanel.includes('fetchDepartments(session.accessToken, {})') && usersPanel.includes('activeDepartments()') && usersPanel.includes('roleScopeDepartmentState(scope, departments)') && usersPanel.includes('DISABLED'), 'scope editor loads department state and only offers active departments')
assert(usersPanel.includes('重新启用') && usersPanel.includes('移除范围') && usersPanel.includes('清空范围'), 'scope editor supports re-enable, remove and explicit clear')
assert(usersPanel.includes('buildUserAccessPayload(form') && usersPanel.includes('roleScopes: buildAssignedRoleScopePayload(scopeRows.value, form.roles)'), 'full user save carries assigned roles, filtered scopes and overrides together')
const scopeToggle = usersPanel.match(/function toggleScope\(scope\) \{([^}]*)\}/)?.[1] || ''
assert(scopeToggle.includes('scopeRows.value[scope.index]') && scopeToggle.includes('target.status'), 'scope toggle mutates the reactive source row')
assert(usersPanel.includes('session.user?.super === true') && usersPanel.includes('session.user?.developer === true'), 'super and developer maintain protected users')
assert(!usersPanel.includes('首次登录需修改'), 'password reset notice does not overstate force-change')
for (const field of ['description', 'cardNo', 'inputCode', 'serviceStar', 'expandInterface1', 'expandInterface2', 'inputCodeCompareMode', 'mustChangePassword', 'cannotModifyPassword', 'accountLocked', 'accountStopped', 'hidden', 'system', 'super', 'developer', 'lastLoginAt', 'lastLoginIp', 'lockUntil', 'passwordChangedAt', 'failedLoginCount']) {
  assert(usersPanel.includes(field), `user lifecycle UI ${field}`)
}
assert(usersPanel.includes('requestReset(row)') && usersPanel.includes('确认重置密码') && usersPanel.includes('mustChangePassword'), 'password reset confirmation and state')
assert(router.includes("name: 'password-change'") && router.includes('session.user?.mustChangePassword'), 'forced password change route guard')
assert(passwordView.includes('currentPassword') && passwordView.includes('newPassword') && passwordView.includes('cannotModifyPassword'), 'forced password change form')
const permissionsPanel = readFileSync(new URL('../src/views/access/PermissionsPanel.vue', import.meta.url), 'utf8')
const rolesPanel = readFileSync(new URL('../src/views/access/RolesPanel.vue', import.meta.url), 'utf8')
for (const tab of ['permissions', 'modules', 'categories', 'modes']) {
  assert(permissionsPanel.includes(`catalogTab === '${tab}'`) || permissionsPanel.includes(`catalogTab = '${tab}'`), `permission catalogue tab ${tab}`)
}
for (const field of ['moduleCode', 'categoryId', 'modeCode', 'description', 'hidden', 'status']) {
  assert(permissionsPanel.includes(field), `catalog metadata editor ${field}`)
}
for (const field of ['departmentFunctions', 'needSpecifyDepartment', 'secondPermissionId', 'homePageUrl', 'dbRoleName', 'needSecondPermission']) {
  assert(permissionsPanel.includes(field), `permission catalogue metadata ${field}`)
}
assert(!permissionsPanel.includes('permissionId') && !api.includes("'permissionId'"), 'unsupported permission record number is omitted')
for (const field of ['orderSort', 'standName']) assert(rolesPanel.includes(field), `role metadata ${field}`)
for (const field of ['secondPermissionCode', 'actionsText', 'moduleCode', 'categoryId', 'modeCode', 'hidden', 'status']) {
  assert(permissionsPanel.includes(field), `permission editor ${field}`)
}
assert(usersPanel.includes("role.status !== 'ACTIVE' && !form.roles.includes(role.code)") && usersPanel.includes('已停用，仅可移除'), 'disabled assigned roles remain removable but cannot be newly assigned')
assert(rolesPanel.includes("permission.status !== 'ACTIVE'") && rolesPanel.includes('已停用'), 'disabled permissions cannot be newly assigned to roles')

process.env.VITE_USE_MOCK = 'true'
const accessApi = await import('../src/master/api.js')
const users = await accessApi.fetchUsers('')
const userId = users.items[0]?.userId
assert(userId, 'mock user seed')
const bundleOverride = { permissionCode: 'charge:refund', moduleCode: 'outpatient', departmentId: 'dept-1', effect: 'DENY' }
const lifecycleUser = await accessApi.saveUser('', '', { username: 'lifecycle-check', displayName: '生命周期检查', description: '用户管理测试', cardNo: 'C-10', inputCode: 'smcs', serviceStar: null, expandInterface1: 14, expandInterface2: 27, inputCodeCompareMode: 2, roles: ['REGISTRAR'], mustChangePassword: false, cannotModifyPassword: true, accountStopped: true, hidden: true, system: false, super: false, developer: false, permissionOverrides: [bundleOverride] })
assert(lifecycleUser.description === '用户管理测试' && lifecycleUser.cardNo === 'C-10' && lifecycleUser.inputCode === 'smcs', 'mock user metadata round-trip')
assert(lifecycleUser.mustChangePassword === false && lifecycleUser.cannotModifyPassword && lifecycleUser.accountStopped && lifecycleUser.hidden, 'mock user lifecycle round-trip')
assert(lifecycleUser.serviceStar === null && lifecycleUser.expandInterface1 === 14 && lifecycleUser.expandInterface2 === 27 && lifecycleUser.inputCodeCompareMode === 2, 'mock user extended fields')
const mockScopes = [
  { roleCode: 'REGISTRAR', departmentId: 'dept-1', status: 'ACTIVE' },
  { roleCode: 'REGISTRAR', departmentId: 'dept-2', status: 'ACTIVE' },
]
await accessApi.saveUser('', lifecycleUser.userId, { roles: ['REGISTRAR', 'DOCTOR'], roleScopes: [...mockScopes, { roleCode: 'DOCTOR', departmentId: 'dept-3', status: 'DISABLED' }], permissionOverrides: [bundleOverride] })
const scopedUser = (await accessApi.fetchUsers('')).items.find((item) => item.userId === lifecycleUser.userId)
assert(JSON.stringify(scopedUser.roleScopes) === JSON.stringify([...mockScopes, { roleCode: 'DOCTOR', departmentId: 'dept-3', status: 'DISABLED' }]), 'mock scope readback preserves multiple departments and role isolation')
await accessApi.saveUser('', lifecycleUser.userId, { accountLocked: false })
const partialUser = (await accessApi.fetchUsers('')).items.find((item) => item.userId === lifecycleUser.userId)
assert(JSON.stringify(partialUser.roleScopes) === JSON.stringify(scopedUser.roleScopes) && JSON.stringify(partialUser.roles) === JSON.stringify(scopedUser.roles), 'scope-only changes remain intact across unrelated partial update')
assert(JSON.stringify((await accessApi.fetchUserPermissionOverrides('', lifecycleUser.userId)).items) === JSON.stringify([bundleOverride]), 'scope-only changes preserve personal overrides')
await accessApi.saveUser('', lifecycleUser.userId, { roles: ['REGISTRAR', 'DOCTOR'], roleScopes: [] })
assert(JSON.stringify((await accessApi.fetchUsers('')).items.find((item) => item.userId === lifecycleUser.userId).roleScopes) === '[]', 'mock explicit clear is distinct from omission')
await accessApi.saveUser('', lifecycleUser.userId, { mustChangePassword: false, cannotModifyPassword: false, accountLocked: true, accountStopped: false, hidden: false, system: false, super: true, developer: true })
const lifecycleUpdated = (await accessApi.fetchUsers('')).items.find((item) => item.userId === lifecycleUser.userId)
assert(lifecycleUpdated.accountLocked && lifecycleUpdated.super && lifecycleUpdated.developer && !lifecycleUpdated.hidden && !lifecycleUpdated.accountStopped, 'mock user lifecycle patch')
const override = bundleOverride
await accessApi.saveUserPermissionOverrides('', userId, [override])
const roundTrip = await accessApi.fetchUserPermissionOverrides('', userId)
assert(roundTrip.items.length === 1 && roundTrip.items[0].effect === 'DENY' && roundTrip.items[0].departmentId === 'dept-1', 'mock user override round-trip')
await accessApi.saveAccessRole('', '', { code: 'CHECK_ROLE', name: '检查角色', orderSort: 17, standName: '挂号员', permissionCodes: ['patient:create'] })
const roleRoundTrip = await accessApi.fetchAccessRoles('')
assert(roleRoundTrip.items.some((role) => role.code === 'CHECK_ROLE' && role.orderSort === 17 && role.standName === '挂号员'), 'mock role create')
await accessApi.saveAccessPermission('', '', { code: 'test:review', name: '检查权限', moduleCode: 'outpatient', categoryId: 'patient', modeCode: 'card', secondPermissionCode: 'admin:user', dbRoleName: 'HIS_ADMIN', needSecondPermission: true, actions: ['N', 'M'], status: 'ACTIVE', hidden: true })
await accessApi.saveAccessPermission('', 'test:review', { name: '更新检查权限', moduleCode: 'master', categoryId: 'admin', modeCode: 'password', secondPermissionCode: '', dbRoleName: 'HIS_ADMIN', needSecondPermission: true, actions: ['S'], status: 'INACTIVE', hidden: false })
const permissionRoundTrip = await accessApi.fetchAccessPermissions('')
const savedPermission = permissionRoundTrip.items.find((item) => item.code === 'test:review')
assert(savedPermission?.name === '更新检查权限' && savedPermission.modeCode === 'password' && savedPermission.actions.join(',') === 'S' && savedPermission.hidden === false && savedPermission.dbRoleName === 'HIS_ADMIN' && savedPermission.needSecondPermission, 'mock permission create and update')
await accessApi.saveAccessModule('', '', { code: 'pharmacy', name: '药房', description: '药品业务', status: 'ACTIVE', hidden: false, departmentFunctions: '18446744073709551615', needSpecifyDepartment: true, secondPermissionId: 9001, homePageUrl: '/pharmacy' })
await accessApi.saveAccessModule('', 'pharmacy', { name: '药房管理', description: '药房模块', status: 'DISABLED', hidden: true })
const moduleRoundTrip = (await accessApi.fetchAccessModules('')).items.find((item) => item.code === 'pharmacy')
assert(moduleRoundTrip?.name === '药房管理' && moduleRoundTrip.status === 'DISABLED' && moduleRoundTrip.hidden && moduleRoundTrip.departmentFunctions === '18446744073709551615' && moduleRoundTrip.needSpecifyDepartment && moduleRoundTrip.secondPermissionId === 9001 && moduleRoundTrip.homePageUrl === '/pharmacy', 'mock module metadata create and update')
await accessApi.savePermissionCategory('', '', { id: 'report', name: '报表', description: '报表查询', status: 'ACTIVE' })
await accessApi.savePermissionCategory('', 'report', { name: '统计报表', description: '统计查询', status: 'INACTIVE' })
const categoryRoundTrip = (await accessApi.fetchAccessCategories('')).items.find((item) => item.id === 'report')
assert(categoryRoundTrip?.name === '统计报表' && categoryRoundTrip.status === 'INACTIVE', 'mock permission category metadata create and update')
await accessApi.savePermissionMode('', '', { code: 'pin', name: '工号口令', status: 'ACTIVE' })
await accessApi.savePermissionMode('', 'pin', { name: '工号密码', status: 'DISABLED' })
const modeRoundTrip = (await accessApi.fetchAccessModes('')).items.find((item) => item.code === 'pin')
assert(modeRoundTrip?.name === '工号密码' && modeRoundTrip.status === 'DISABLED', 'mock authentication mode create and update')
const previousFetch = globalThis.fetch
let capturedRequest
globalThis.fetch = async (url, init) => {
  capturedRequest = { url: String(url), init }
  return new Response(JSON.stringify({ code: 'live:permission', name: 'Live permission' }), { status: init.method === 'POST' ? 201 : 200, headers: { 'Content-Type': 'application/json' } })
}
process.env.VITE_USE_MOCK = 'false'
await accessApi.saveAccessPermission('token', '', { code: 'live:permission', name: 'Live permission', moduleCode: 'outpatient', categoryId: 'patient', modeCode: 'password', secondPermissionCode: '', dbRoleName: 'LIVE_ROLE', needSecondPermission: true, actions: ['N'], status: 'ACTIVE', hidden: false })
assert(capturedRequest.url === '/api/v1/access/permissions' && capturedRequest.init.method === 'POST', 'live permission create route')
const createBody = JSON.parse(capturedRequest.init.body)
assert(createBody.code === 'live:permission' && createBody.actions.join(',') === 'N' && createBody.hidden === false && createBody.dbRoleName === 'LIVE_ROLE' && createBody.needSecondPermission, 'live permission create body')
globalThis.fetch = async (url, init) => {
  capturedRequest = { url: String(url), init }
  return new Response(JSON.stringify({ code: 'live:permission', name: 'Edited permission' }), { status: init.method === 'POST' ? 201 : 200, headers: { 'Content-Type': 'application/json' } })
}
await accessApi.saveAccessPermission('token', 'live:permission', { name: 'Edited permission', moduleCode: 'master', categoryId: 'admin', modeCode: 'card', secondPermissionCode: 'admin:user', dbRoleName: 'LIVE_ROLE', needSecondPermission: false, actions: ['M', 'D'], status: 'INACTIVE', hidden: true })
assert(capturedRequest.url === '/api/v1/access/permissions/live%3Apermission' && capturedRequest.init.method === 'PATCH', 'live permission patch route')
assert(JSON.parse(capturedRequest.init.body).secondPermissionCode === 'admin:user', 'live permission patch body')
assert(JSON.parse(capturedRequest.init.body).needSecondPermission === false && JSON.parse(capturedRequest.init.body).dbRoleName === 'LIVE_ROLE', 'live permission metadata body')
for (const [save, key, base, patch, route] of [
  [accessApi.saveAccessModule, 'module', { code: 'live-module', name: 'Live module', status: 'ACTIVE', hidden: false, departmentFunctions: '18446744073709551615', needSpecifyDepartment: true, secondPermissionId: 77, homePageUrl: '/home' }, { name: 'Live module edited', status: 'DISABLED', hidden: true }, '/api/v1/access/modules'],
  [accessApi.savePermissionCategory, 'category', { id: 'live-category', name: 'Live category', status: 'ACTIVE' }, { name: 'Live category edited', status: 'INACTIVE' }, '/api/v1/access/categories'],
  [accessApi.savePermissionMode, 'mode', { code: 'live-mode', name: 'Live mode', status: 'ACTIVE' }, { name: 'Live mode edited', status: 'INACTIVE' }, '/api/v1/access/modes'],
]) {
  let identifier = key === 'category' ? base.id : base.code
  await save('token', '', base)
  assert(capturedRequest.url === route && capturedRequest.init.method === 'POST', `live ${key} create route`)
  assert(JSON.stringify(JSON.parse(capturedRequest.init.body)) === JSON.stringify(base), `live ${key} create body`)
  await save('token', identifier, patch)
  assert(capturedRequest.url === `${route}/${encodeURIComponent(identifier)}` && capturedRequest.init.method === 'PATCH', `live ${key} patch route`)
  assert(JSON.stringify(JSON.parse(capturedRequest.init.body)) === JSON.stringify(patch), `live ${key} patch body`)
}
await accessApi.saveAccessRole('token', '', { code: 'LIVE_ROLE', name: 'Live role', orderSort: 23, standName: '门诊医生', moduleCode: 'outpatient', departmentId: '', permissionCodes: [], status: 'ACTIVE', hidden: false, system: false })
assert(capturedRequest.url === '/api/v1/access/roles' && JSON.parse(capturedRequest.init.body).orderSort === 23 && JSON.parse(capturedRequest.init.body).standName === '门诊医生', 'live role metadata request body')
const liveRoleScopes = [
  { roleCode: 'REGISTRAR', departmentId: 'dept-1', status: 'ACTIVE' },
  { roleCode: 'REGISTRAR', departmentId: 'dept-2', status: 'DISABLED' },
  { roleCode: 'DOCTOR', departmentId: 'dept-3', status: 'ACTIVE' },
]
await accessApi.saveUser('token', 'user-1', { roles: ['REGISTRAR', 'DOCTOR'], roleScopes: liveRoleScopes, permissionOverrides: [bundleOverride] })
assert(capturedRequest.url === '/api/v1/users/user-1' && capturedRequest.init.method === 'PATCH', 'live full user bundle save route')
const liveUserBody = JSON.parse(capturedRequest.init.body)
assert(JSON.stringify(liveUserBody.roles) === JSON.stringify(['REGISTRAR', 'DOCTOR']) && JSON.stringify(liveUserBody.roleScopes) === JSON.stringify(liveRoleScopes) && JSON.stringify(liveUserBody.permissionOverrides) === JSON.stringify([bundleOverride]), 'live user save sends roles, scopes and personal overrides together')
await accessApi.saveUser('token', 'user-1', { roles: ['REGISTRAR', 'DOCTOR'] })
assert(!Object.hasOwn(JSON.parse(capturedRequest.init.body), 'roleScopes'), 'live user patch omits roleScopes when caller omits it')
await accessApi.saveUser('token', 'user-1', { roleScopes: [] })
assert(JSON.stringify(JSON.parse(capturedRequest.init.body).roleScopes) === '[]', 'live user patch preserves explicit empty roleScopes')
globalThis.fetch = previousFetch
process.env.VITE_USE_MOCK = 'false'
const authApi = await import('../src/api/authApi.js')
let passwordRequest
globalThis.fetch = async (url, init) => {
  passwordRequest = { url: String(url), init }
  return new Response(null, { status: 204 })
}
await authApi.changePassword('token', 'old-password', 'new-password')
assert(passwordRequest.url === '/api/v1/auth/change-password' && passwordRequest.init.method === 'POST', 'password change route')
assert(JSON.parse(passwordRequest.init.body).currentPassword === 'old-password' && JSON.parse(passwordRequest.init.body).newPassword === 'new-password', 'password change body')
globalThis.fetch = previousFetch
console.log('access-control checks passed')
