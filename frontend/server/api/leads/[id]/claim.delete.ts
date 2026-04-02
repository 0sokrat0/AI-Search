import { proxyAndReturnOK, requireRouteParam } from '~~/server/utils/api-proxy'

export default eventHandler(async (event) => {
  const id = requireRouteParam(event, 'id')
  return await proxyAndReturnOK(event, `/api/v1/leads/${id}/claim`, {
    method: 'DELETE'
  })
})
