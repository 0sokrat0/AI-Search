import type { Member } from '~/types'
import { proxyBackendData, requireTenantID } from '~~/server/utils/api-proxy'

export default eventHandler(async (event) => {
  const query = getQuery(event) as Record<string, string | number | undefined>
  const tenantID = requireTenantID(event, query.tenant_id)

  return await proxyBackendData<Member[]>(event, '/api/v1/users', {
    query: {
      ...query,
      tenant_id: tenantID
    }
  })
})
