import { proxyBackend, readJSONBody } from '~~/server/utils/api-proxy'

export default eventHandler(async (event) => {
  const method = getMethod(event)
  const query = getQuery(event)

  switch (method) {
    case 'GET':
      return await proxyBackend(event, '/api/v1/accounts', {
        method: 'GET',
        query
      })
    case 'POST': {
      const body = await readJSONBody(event)
      return await proxyBackend(event, '/api/v1/accounts', {
        method: 'POST',
        body
      })
    }
    default:
      throw createError({ statusCode: 405, message: 'Method Not Allowed' })
  }
})
