import { proxyBackend, readJSONBody, requireRouteParam } from '~~/server/utils/api-proxy'

export default eventHandler(async (event) => {
  const id = requireRouteParam(event, 'id')
  const action = requireRouteParam(event, 'action')
  const method = getMethod(event)
  const query = getQuery(event)
  const body = method !== 'GET' ? await readJSONBody(event) : undefined

  return await proxyBackend(event, `/api/v1/accounts/${id}/${action}`, {
    method,
    query,
    body
  })
})
