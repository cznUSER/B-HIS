// 2026-10-12 P1-H05 登录界面
// OpenAPI 0.3.2：LoginRequest / LoginResponse / UserDto。

import { apiError } from './errors.js'

export function assertLoginRequest(body) {
  const username = body && typeof body.username === 'string' ? body.username : ''
  const password = body && typeof body.password === 'string' ? body.password : ''
  if (username.length < 1 || password.length < 1) {
    throw apiError(400, 'VALIDATION_ERROR', '请输入用户名和密码')
  }
  return { username, password }
}

export function assertUserDto(user) {
  if (!user || typeof user !== 'object') {
    throw apiError(502, 'HTTP_ERROR', '用户信息不符合契约')
  }
  const { id, username, displayName, roles } = user
  if (
    typeof id !== 'string' ||
    id.length < 1 ||
    typeof username !== 'string' ||
    username.length < 1 ||
    typeof displayName !== 'string' ||
    !Array.isArray(roles) ||
    roles.some((role) => typeof role !== 'string')
  ) {
    throw apiError(502, 'HTTP_ERROR', '用户信息不符合契约')
  }
  const permissions = Array.isArray(user.permissions) ? user.permissions.filter((value) => typeof value === 'string') : []
  const scopes = Array.isArray(user.scopes) ? user.scopes.filter((value) => typeof value === 'string') : []
  const deniedPermissions = Array.isArray(user.deniedPermissions) ? user.deniedPermissions.filter((value) => typeof value === 'string') : []
  return {
    id,
    username,
    displayName,
    roles,
    permissions,
    scopes,
    deniedPermissions,
    accessControlLoaded: user.accessControlLoaded === true,
    super: user.super === true,
    developer: user.developer === true,
    mustChangePassword: user.mustChangePassword === true,
    cannotModifyPassword: user.cannotModifyPassword === true,
  }
}

export function assertLoginResponse(body) {
  if (!body || typeof body !== 'object') {
    throw apiError(502, 'HTTP_ERROR', '登录响应不符合契约')
  }
  const { accessToken, tokenType, expiresIn, user } = body
  if (typeof accessToken !== 'string' || accessToken.length < 1) {
    throw apiError(502, 'HTTP_ERROR', '登录响应缺少 accessToken')
  }
  if (tokenType !== 'Bearer') {
    throw apiError(502, 'HTTP_ERROR', '登录响应 tokenType 须为 Bearer')
  }
  if (typeof expiresIn !== 'number' || !Number.isFinite(expiresIn) || expiresIn <= 0) {
    throw apiError(502, 'HTTP_ERROR', '登录响应 expiresIn 无效')
  }
  return {
    accessToken,
    tokenType,
    expiresIn,
    user: assertUserDto(user),
  }
}
