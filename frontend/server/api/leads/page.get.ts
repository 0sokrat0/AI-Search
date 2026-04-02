import type { CursorPage, Lead } from '~/types'
import { proxyBackendData } from '~~/server/utils/api-proxy'

export default eventHandler(async (event) => {
  const query = getQuery(event)
  const response = await proxyBackendData<CursorPage<any>>(event, '/api/v1/leads', {
    method: 'GET',
    query: {
      category: query.category,
      qualified_only: query.qualified_only ?? true,
      limit: query.limit ?? 50,
      cursor: query.cursor
    }
  })

  return {
    items: (response.items ?? []).map((row) => {
      const createdAt = row.createdAt ?? row.updatedAt ?? ''
      const semanticDirection = String(row.semanticDirection ?? '').trim()
      const semanticCategory = String(row.semanticCategory ?? (semanticDirection || 'leads'))
      return {
        ...row,
        semanticDirection,
        semanticCategory,
        ownerId: String(row.ownerId ?? ''),
        ownerName: String(row.ownerName ?? ''),
        ownerAssignedAt: String(row.ownerAssignedAt ?? ''),
        signalsCount: row.signalsCount ?? 1,
        lastSeenAt: row.lastSeenAt ?? createdAt
      } as Lead
    }),
    nextCursor: response.nextCursor ?? ''
  } satisfies CursorPage<Lead>
})
