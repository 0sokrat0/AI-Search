import { apiFetchWithAuth } from '~~/server/utils/backend-auth'

export default defineEventHandler(async (event) => {
  deleteCookie(event, 'auth_token', { path: '/' })
  deleteCookie(event, 'refresh_token', { path: '/' })

  try {
    const response = await apiFetchWithAuth<unknown>(event, '/api/v1/auth/logout', {
      method: 'POST'
    })

    return response
  } catch {
    return { success: true }
  }
})
