import type { LeadStats } from '~/types'
import { proxyBackendData } from '~~/server/utils/api-proxy'

export default eventHandler(async (event) => {
  const query = getQuery(event)
  return await proxyBackendData<LeadStats>(event, '/api/v1/leads/stats', {
    query
  })
})
