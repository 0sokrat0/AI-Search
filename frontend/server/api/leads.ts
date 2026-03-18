import type { Lead } from '~/types'
import { proxyBackendData } from '~~/server/utils/api-proxy'

export default eventHandler(async (event) => {
  const query = getQuery(event)
  const response = await proxyBackendData<{ items?: any[] }>(event, '/api/v1/leads', {
    method: 'GET',
    query: {
      category: query.category,
      qualified_only: query.qualified_only ?? true,
      limit: query.limit ?? 50,
      cursor: query.cursor
    }
  })

  return (response.items ?? []).map((row) => {
    const createdAt = row.createdAt ?? row.updatedAt ?? new Date().toISOString()
    const semanticDirection = String(row.semanticDirection ?? '').trim()
    const semanticCategory = String(row.semanticCategory ?? (semanticDirection || 'leads'))
    return {
      ...row,
      semanticDirection,
      semanticCategory,
      signalsCount: row.signalsCount ?? 1,
      lastSeenAt: row.lastSeenAt ?? createdAt
    } as Lead
  })
})
