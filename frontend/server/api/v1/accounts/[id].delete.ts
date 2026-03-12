import { proxyBackend, requireRouteParam } from '~~/server/utils/api-proxy'

export default eventHandler(async (event) => {
  const id = requireRouteParam(event, 'id')
  return await proxyBackend(event, `/api/v1/accounts/${id}`, {
    method: 'DELETE'
  })
})
