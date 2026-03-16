import { proxyBackendData } from '~~/server/utils/api-proxy'

export default eventHandler(async (event) => {
  const form = await readMultipartFormData(event)
  const filePart = form?.find(part => part.name === 'file')

  if (!filePart?.data || !filePart.filename) {
    throw createError({
      statusCode: 400,
      message: 'CSV file is required'
    })
  }

  const content = filePart.data.toString('utf-8')
  if (!content.trim()) {
    throw createError({
      statusCode: 400,
      message: 'CSV file is empty'
    })
  }

  return await proxyBackendData(event, '/api/v1/settings/knowledge/import', {
    method: 'POST',
    body: {
      fileName: filePart.filename,
      content
    }
  })
})
