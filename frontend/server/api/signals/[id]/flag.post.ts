import { proxyBackend, readJSONBody, requireRouteParam } from '~~/server/utils/api-proxy'

export default eventHandler(async (event) => {
  const id = requireRouteParam(event, 'id')
  const body = await readJSONBody(event)

  return await proxyBackend(event, `/api/v1/signals/${id}/flag`, {
    method: 'POST',
    body
  })
})
