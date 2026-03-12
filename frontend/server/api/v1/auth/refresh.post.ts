const cookieOptions = {
  path: '/',
  sameSite: 'lax' as const,
  httpOnly: false,
  secure: false,
  maxAge: 60 * 60 * 24 * 7
}

export default defineEventHandler(async (event) => {
  const config = useRuntimeConfig()
  const body = await readBody(event)
  const refreshTokenFromCookie = getCookie(event, 'refresh_token')

  try {
    const response = await $fetch<any>(`${config.public.apiBase}/api/v1/auth/refresh`, {
      method: 'POST',
      body: {
        refreshToken: body?.refreshToken ?? refreshTokenFromCookie
      }
    })

    const payload = response?.data ?? response ?? {}
    const accessToken = payload?.accessToken ?? payload?.token
    const refreshToken = payload?.refreshToken ?? body?.refreshToken ?? refreshTokenFromCookie

    if (accessToken) {
      setCookie(event, 'auth_token', accessToken, cookieOptions)
    }
    if (refreshToken) {
      setCookie(event, 'refresh_token', refreshToken, cookieOptions)
    }

    return payload
  } catch (error: any) {
    throw createError({
      statusCode: error.statusCode || 500,
      message: error.data?.message || error.message || 'Token refresh failed'
    })
  }
})
