import type { SignalItem } from '~/types'
import { proxyBackendData } from '~~/server/utils/api-proxy'

export default eventHandler(async (event) => {
  const query = getQuery(event)
  const limit = Number(query.limit || 200)
  const offset = Number(query.offset || 0)
  const tab = typeof query.tab === 'string' ? query.tab : undefined
  const category = typeof query.category === 'string' ? query.category : undefined

  return await proxyBackendData<SignalItem[]>(event, '/api/v1/signals/inbox', {
    method: 'GET',
    query: { limit, offset, tab, category }
  })
})
