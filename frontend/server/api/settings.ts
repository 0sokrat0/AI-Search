import type { AppSettings } from '~/types'
import { proxyBackendData } from '~~/server/utils/api-proxy'

export default eventHandler(async (event) => {
  return await proxyBackendData<AppSettings>(event, '/api/v1/settings')
})
