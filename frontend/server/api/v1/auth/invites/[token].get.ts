export default defineEventHandler(async (event): Promise<unknown> => {
  const config = useRuntimeConfig()
  const token = getRouterParam(event, 'token')

  try {
    return await $fetch(`${config.apiBaseInternal}/api/v1/auth/invites/${token}`, {
      method: 'GET'
    })
  } catch (err: any) {
    const status = err?.response?.status ?? err?.statusCode ?? 500
    const message = err?.data?.error ?? err?.message ?? 'Invite not found'
    throw createError({ statusCode: status, message })
  }
})
