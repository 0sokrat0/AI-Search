import { proxyBackendData, requireRouteParam } from '~~/server/utils/api-proxy'

export default eventHandler(async (event) => {
  const senderID = requireRouteParam(event, 'senderID')
  const query = getQuery(event)

  return await proxyBackendData<any[]>(event, `/api/v1/signals/sender/${senderID}`, {
    query
  })
})
