import type { ChartDayBucket } from '~/types'
import { proxyBackendData } from '~~/server/utils/api-proxy'

export default eventHandler(async (event) => {
  const query = getQuery(event)
  return await proxyBackendData<ChartDayBucket[]>(event, '/api/v1/signals/chart', {
    method: 'GET',
    query: {
      from: query.from,
      to: query.to
    }
  })
})
