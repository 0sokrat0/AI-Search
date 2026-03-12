import type { H3Event } from 'h3'

const cookieOptions = {
  path: '/',
  sameSite: 'lax' as const,
  httpOnly: false,
  secure: false,
  maxAge: 60 * 60 * 24 * 7
}

function setAuthHeader(headers: Record<string, string>, value: string) {
  delete headers.authorization
  delete headers.Authorization
  headers.Authorization = value
}

function resolveAuthHeader(event: H3Event): string | undefined {
  const authHeader = getHeader(event, 'authorization')
  if (authHeader) {
    return authHeader
  }

  const token = getCookie(event, 'auth_token')
  if (!token) {
    return undefined
  }

  return `Bearer ${token}`
}

function decodeJwtPayload(token: string): Record<string, unknown> | null {
  const parts = token.split('.')
  const payloadPart = parts[1]
  if (!payloadPart) return null

  try {
    const base64 = payloadPart.replace(/-/g, '+').replace(/_/g, '/')
    const padded = base64.padEnd(Math.ceil(base64.length / 4) * 4, '=')
    const json = Buffer.from(padded, 'base64').toString('utf-8')
    return JSON.parse(json) as Record<string, unknown>
  } catch {
    return null
  }
}

export function getTenantIDFromEvent(event: H3Event): string | undefined {
  const authHeader = resolveAuthHeader(event)
  if (!authHeader) return undefined

  const token = authHeader.replace(/^Bearer\s+/i, '').trim()
  if (!token) return undefined

  const payload = decodeJwtPayload(token)
  const tenantID = payload?.tenant_id
  return typeof tenantID === 'string' && tenantID ? tenantID : undefined
}

async function refreshAccessToken(event: H3Event): Promise<string | undefined> {
  const config = useRuntimeConfig()
  const refreshToken = getCookie(event, 'refresh_token')
  if (!refreshToken) {
    return undefined
  }

  try {
    const response = await $fetch<any>(`${config.apiBaseInternal}/api/v1/auth/refresh`, {
      method: 'POST',
      body: { refreshToken }
    })

    const payload = response?.data ?? response ?? {}
    const accessToken = payload?.accessToken ?? payload?.token
    const nextRefreshToken = payload?.refreshToken ?? refreshToken
    if (!accessToken) {
      return undefined
    }

    setCookie(event, 'auth_token', accessToken, cookieOptions)
    setCookie(event, 'refresh_token', nextRefreshToken, cookieOptions)

    return `Bearer ${accessToken}`
  } catch {
    return undefined
  }
}

export async function apiFetchWithAuth<T>(event: H3Event, path: string, options: Record<string, any> = {}): Promise<T> {
  const config = useRuntimeConfig()
  const headers: Record<string, string> = { ...(options.headers ?? {}) }
  const authHeader = resolveAuthHeader(event)
  if (authHeader) {
    setAuthHeader(headers, authHeader)
  }

  try {
    const response = await $fetch(`${config.apiBaseInternal}${path}`, {
      ...options,
      headers
    })
    return response as T
  } catch (error: any) {
    const status = error?.statusCode ?? error?.status
    if (status !== 401) {
      throw error
    }

    const refreshedHeader = await refreshAccessToken(event)
    if (!refreshedHeader) {
      deleteCookie(event, 'auth_token')
      deleteCookie(event, 'refresh_token')
      throw createError({ statusCode: 401, message: 'Session expired' })
    }

    const retryHeaders: Record<string, string> = { ...headers }
    setAuthHeader(retryHeaders, refreshedHeader)

    const retryResponse = await $fetch(`${config.apiBaseInternal}${path}`, {
      ...options,
      headers: retryHeaders
    })

    return retryResponse as T
  }
}
