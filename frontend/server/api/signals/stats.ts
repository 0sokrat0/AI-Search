import type { IngestStats } from '~/types'
import { proxyBackendData } from '~~/server/utils/api-proxy'

export default eventHandler(async (event) => {
  const query = getQuery(event)
  return await proxyBackendData<IngestStats>(event, '/api/v1/signals/stats', {
    query
  })
})
