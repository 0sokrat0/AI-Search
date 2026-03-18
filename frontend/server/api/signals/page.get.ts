import type { CursorPage, SignalItem } from '~/types'
import { proxyBackendData } from '~~/server/utils/api-proxy'

export default eventHandler(async (event) => {
  const query = getQuery(event)
  return await proxyBackendData<CursorPage<SignalItem>>(event, '/api/v1/signals/inbox', {
    method: 'GET',
    query: {
      limit: query.limit ?? 50,
      cursor: query.cursor,
      tab: query.tab,
      category: query.category,
      show_archived: query.show_archived
    }
  })
})
