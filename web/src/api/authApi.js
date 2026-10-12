// 2026-10-12 P1-H05 登录界面
// POST /api/v1/auth/login | POST /api/v1/auth/logout | GET /api/v1/auth/me
// 口令只进入请求体，不写入日志。

import { apiUrl, useMock } from './config.js'
import { apiError, readErrorResponse } from './errors.js'
import { assertLoginRequest, assertLoginResponse, assertUserDto } from './contract.js'
import { mockLogin, mockLogout, mockMe } from './mockAuth.js'

const JSON_ACCEPT = 'application/json'

function delay(ms) {
  return new Promise((resolve) => {
    setTimeout(resolve, ms)
  })
}

function bearer(accessToken) {
  return { Authorization: `Bearer ${accessToken}` }
}

async function parseJson(response) {
  const text = await response.text()
  if (!text) return null
  try {
    return JSON.parse(text)
  } catch {
    throw apiError(response.status, 'HTTP_ERROR', '响应无法解析')
  }
}

export async function login(username, password) {
  const request = assertLoginRequest({ username, password })
  if (useMock()) {
    await delay(400)
    return mockLogin(request)
  }

  let response
  try {
    response = await fetch(apiUrl('/auth/login'), {
      method: 'POST',
      headers: {
        Accept: JSON_ACCEPT,
        'Content-Type': 'application/json; charset=utf-8',
      },
      body: JSON.stringify(request),
    })
  } catch {
    throw apiError(0, 'HTTP_ERROR', '无法连接登录服务')
  }

  if (!response.ok) throw await readErrorResponse(response)
  return assertLoginResponse(await parseJson(response))
}

export async function logout(accessToken) {
  if (!accessToken) return
  if (useMock()) {
    await delay(200)
    mockLogout(accessToken)
    return
  }

  let response
  try {
    response = await fetch(apiUrl('/auth/logout'), {
      method: 'POST',
      headers: {
        Accept: JSON_ACCEPT,
        ...bearer(accessToken),
      },
    })
  } catch {
    throw apiError(0, 'HTTP_ERROR', '无法连接登录服务')
  }

  if (response.status === 204) return
  if (response.status === 200) {
    const body = await parseJson(response)
    if (body && body.success === false) {
      throw apiError(200, 'HTTP_ERROR', '退出登录失败')
    }
    return
  }
  throw await readErrorResponse(response)
}

export async function fetchMe(accessToken) {
  if (useMock()) {
    return mockMe(accessToken)
  }

  let response
  try {
    response = await fetch(apiUrl('/auth/me'), {
      method: 'GET',
      headers: {
        Accept: JSON_ACCEPT,
        ...bearer(accessToken),
      },
    })
  } catch {
    throw apiError(0, 'HTTP_ERROR', '无法连接登录服务')
  }

  if (!response.ok) throw await readErrorResponse(response)
  return assertUserDto(await parseJson(response))
}
