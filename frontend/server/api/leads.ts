import type { Lead } from '~/types'
import { proxyBackendData } from '~~/server/utils/api-proxy'

export default eventHandler(async (event) => {
  const query = getQuery(event)
  const rows = await proxyBackendData<any[]>(event, '/api/v1/leads', {
    method: 'GET',
    query: {
      category: query.category
    }
  })
  return rows.map((row) => {
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
