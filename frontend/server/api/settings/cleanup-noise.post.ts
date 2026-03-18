import { proxyBackendData, readJSONBody } from '~~/server/utils/api-proxy'

export default eventHandler(async (event) => {
  const body = await readJSONBody<{ older_than_hours: string, message_ids?: string[] }>(event)
  return await proxyBackendData<{ deleted: number, hours: number, message_ids?: number }>(event, '/api/v1/settings/cleanup-noise', {
    method: 'POST',
    body
  })
})
