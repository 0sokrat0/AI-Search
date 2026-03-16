import { proxyBackend, readJSONBody, requireRouteParam } from '~~/server/utils/api-proxy'

export default eventHandler(async (event) => {
  const id = requireRouteParam(event, 'id')
  const body = await readJSONBody<Record<string, unknown>>(event)
  return await proxyBackend(event, `/api/v1/users/${id}`, {
    method: 'PUT',
    body
  })
})