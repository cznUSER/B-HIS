export function buildUserAccessPayload(form, {
  creating = false,
  canEditDeveloperFlags = false,
  canEditSuperFlag = false,
  roleScopes,
} = {}) {
  const body = {
    displayName: form.displayName,
    description: form.description,
    inputCode: form.inputCode,
    ...(form.password ? { password: form.password } : {}),
    serviceStar: form.serviceStar === '' ? null : Number(form.serviceStar),
    expandInterface1: Number(form.expandInterface1),
    expandInterface2: Number(form.expandInterface2),
    inputCodeCompareMode: Number(form.inputCodeCompareMode),
    status: form.status,
    roles: form.roles,
    mustChangePassword: form.mustChangePassword,
    cannotModifyPassword: form.cannotModifyPassword,
    accountStopped: form.accountStopped,
  }

  if (creating) body.username = form.username
  else body.accountLocked = form.accountLocked

  if (Array.isArray(roleScopes)) {
    body.roleScopes = roleScopes.map(({ roleCode, departmentId, status }) => ({ roleCode, departmentId, status }))
  }

  if (canEditDeveloperFlags) {
    body.hidden = form.hidden
    body.system = form.system
    body.developer = form.developer
  }

  if (canEditSuperFlag) {
    body.super = form.super
    body.cardNo = form.cardNo
  }

  return body
}
