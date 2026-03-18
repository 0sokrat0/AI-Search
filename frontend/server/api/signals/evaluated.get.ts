import type { SignalItem } from '~/types'
import { proxyBackendData } from '~~/server/utils/api-proxy'

export default eventHandler(async (event) => {
  const query = getQuery(event)
  const limit = Number(query.limit || 50)
  const offset = Number(query.offset || 0)
  const approved = typeof query.approved === 'string' ? query.approved : undefined

  return await proxyBackendData<SignalItem[]>(event, '/api/v1/signals/evaluated', {
    method: 'GET',
    query: { limit, offset, approved }
  })
})
