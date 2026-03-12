import type { Notification } from '~/types'
import { proxyBackendData } from '~~/server/utils/api-proxy'

export default eventHandler(async (event): Promise<Notification[]> => {
  const rows = await proxyBackendData<any[]>(event, '/api/v1/signals/inbox', {
    method: 'GET',
    query: { limit: 20, offset: 0 }
  })

  return rows.map((signal, index) => ({
    id: index + 1,
    unread: !signal.isIgnored,
    sender: {
      id: index + 1,
      name: signal.fromName || 'Unknown sender',
      email: signal.contact || '',
      status: 'subscribed',
      location: signal.chatTitle || ''
    },
    body: signal.text || '',
    date: signal.date || new Date().toISOString()
  }))
})
