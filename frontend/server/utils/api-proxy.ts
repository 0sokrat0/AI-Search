import type { H3Event } from 'h3'
import { apiFetchWithAuth, getTenantIDFromEvent } from '~~/server/utils/backend-auth'

type ProxyOptions = {
  method?: string
  query?: Record<string, unknown>
  body?: unknown
  headers?: Record<string, string>
}

export function unwrapBackendData<T>(response: unknown): T {
  const payload = response as { data?: T } | null
  return (payload?.data ?? response) as T
}

export function requireRouteParam(event: H3Event, name: string): string {
  const value = getRouterParam(event, name)
  if (!value) {
    throw createError({ statusCode: 400, message: `${name} is required` })
  }
  return value
}

export async function readJSONBody<T>(event: H3Event): Promise<T> {
  return await readBody<T>(event)
}

export function requireTenantID(event: H3Event, fallback?: unknown): string {
  const tenantID = String(fallback ?? getTenantIDFromEvent(event) ?? '').trim()
  if (!tenantID) {
    throw createError({ statusCode: 401, message: 'tenant_id is required' })
  }
  return tenantID
}

export async function proxyBackend<T>(event: H3Event, path: string, options: ProxyOptions = {}): Promise<T> {
  try {
    return await apiFetchWithAuth<T>(event, path, options)
  } catch (err: any) {
    const status = err?.statusCode ?? err?.response?.status ?? err?.status ?? 500
    const message = err?.data?.error ?? err?.data?.message ?? err?.message ?? 'Backend error'
    throw createError({ statusCode: status, message })
  }
}

export async function proxyBackendData<T>(event: H3Event, path: string, options: ProxyOptions = {}): Promise<T> {
  const response = await proxyBackend<unknown>(event, path, options)
  return unwrapBackendData<T>(response)
}

export async function proxyAndReturnOK(event: H3Event, path: string, options: ProxyOptions = {}): Promise<{ ok: true }> {
  await proxyBackend(event, path, options)
  return { ok: true }
}
