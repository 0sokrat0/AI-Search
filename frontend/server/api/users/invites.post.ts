import { proxyBackend, readJSONBody, requireTenantID } from '~~/server/utils/api-proxy'

export default eventHandler(async (event) => {
  const body = await readJSONBody<Record<string, unknown>>(event)
  const tenantID = requireTenantID(event)

  return await proxyBackend(event, '/api/v1/users/invites', {
    method: 'POST',
    body: {
      ...body,
      tenant_id: (body?.tenant_id as string | undefined) || tenantID
    }
  })
})
