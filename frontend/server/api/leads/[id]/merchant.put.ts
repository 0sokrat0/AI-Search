import { proxyAndReturnOK, readJSONBody, requireRouteParam } from '~~/server/utils/api-proxy'

export default eventHandler(async (event) => {
  const id = requireRouteParam(event, 'id')
  const body = await readJSONBody(event)
  return await proxyAndReturnOK(event, `/api/v1/leads/${id}/merchant`, {
    method: 'PUT',
    body
  })
})
