import { proxyBackendData, readJSONBody } from '~~/server/utils/api-proxy'

export default eventHandler(async (event) => {
  const body = await readJSONBody(event)
  return await proxyBackendData(event, '/api/v1/settings', {
    method: 'PUT',
    body
  })
})
