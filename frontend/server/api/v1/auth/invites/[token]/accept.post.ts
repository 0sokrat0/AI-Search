import type { Role } from '~/types/auth'

const cookieOptions = {
  path: '/',
  sameSite: 'lax' as const,
  httpOnly: false,
  secure: process.env.NUXT_COOKIE_SECURE !== 'false' && process.env.NODE_ENV === 'production',
  maxAge: 60 * 60 * 24 * 7
}

export default defineEventHandler(async (event) => {
  const config = useRuntimeConfig()
  const token = getRouterParam(event, 'token')
  const body = await readBody(event)

  let response: any
  try {
    response = await $fetch<any>(`${config.apiBaseInternal}/api/v1/auth/invites/${token}/accept`, {
      method: 'POST',
      body,
      headers: { 'Content-Type': 'application/json' }
    })
  } catch (err: any) {
    const status = err?.response?.status ?? err?.statusCode ?? 500
    const message = err?.data?.error ?? err?.message ?? 'Invite accept failed'
    throw createError({ statusCode: status, message })
  }

  const payload = response?.data ?? response
  const backendUser = payload?.user ?? {}
  const accessToken = payload?.accessToken ?? payload?.token
  const refreshToken = payload?.refreshToken ?? accessToken

  if (!accessToken) {
    throw createError({
      statusCode: 502,
      message: 'Invalid invite accept response from backend'
    })
  }

  setCookie(event, 'auth_token', accessToken, cookieOptions)
  setCookie(event, 'refresh_token', refreshToken, cookieOptions)

  return {
    accessToken,
    refreshToken,
    user: {
      id: backendUser.id ?? '',
      email: backendUser.email ?? body.email ?? '',
      name: backendUser.name ?? '',
      roles: Array.isArray(backendUser.roles) ? backendUser.roles.filter((r: unknown): r is Role => ['super_admin', 'employee'].includes(String(r))) : [],
      tenantID: backendUser.tenantID ?? backendUser.tenant_id ?? '',
      createdAt: backendUser.createdAt ?? backendUser.created_at ?? new Date().toISOString()
    }
  }
})
