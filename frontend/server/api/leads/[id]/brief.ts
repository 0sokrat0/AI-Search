import type { LeadBrief } from '~/types'
import { apiFetchWithAuth } from '~~/server/utils/backend-auth'

export default eventHandler(async (event) => {
  const id = getRouterParam(event, 'id')

  const response = await apiFetchWithAuth<any>(event, `/api/v1/leads/${id}/brief`, {
    method: 'GET'
  })
  const raw = response?.data ?? response
  const signals = Array.isArray(raw?.signals) ? raw.signals : []
  const signalsCount = Number(raw?.signalsCount ?? signals.length ?? 0)
  const lastSeenAt = raw?.lastSeenAt ?? raw?.lead?.updatedAt ?? new Date().toISOString()
  const rawLead = raw?.lead ?? {}
  const lead = {
    ...rawLead,
    id: String(rawLead?.id ?? id ?? ''),
    name: String(rawLead?.name ?? 'Неизвестный контакт'),
    contact: String(rawLead?.contact ?? ''),
    chatTitle: String(rawLead?.chatTitle ?? ''),
    semanticDirection: String(rawLead?.semanticDirection ?? ''),
    semanticCategory: String(rawLead?.semanticCategory ?? ''),
    merchantId: String(rawLead?.merchantId ?? ''),
    companyId: String(rawLead?.companyId ?? rawLead?.merchantId ?? ''),
    company: String(rawLead?.company ?? rawLead?.merchantId ?? ''),
    ownerId: String(rawLead?.ownerId ?? ''),
    ownerName: String(rawLead?.ownerName ?? ''),
    ownerAssignedAt: String(rawLead?.ownerAssignedAt ?? ''),
    contactOwnerId: String(rawLead?.contactOwnerId ?? ''),
    contactOwnerName: String(rawLead?.contactOwnerName ?? ''),
    companyOwnerId: String(rawLead?.companyOwnerId ?? ''),
    companyOwnerName: String(rawLead?.companyOwnerName ?? ''),
    status: String(rawLead?.status ?? 'new'),
    priority: String(rawLead?.priority ?? 'medium'),
    score: Number(rawLead?.score ?? 0),
    geo: Array.isArray(rawLead?.geo) ? rawLead.geo : [],
    products: Array.isArray(rawLead?.products) ? rawLead.products : [],
    signalsCount,
    lastSeenAt
  }

  const mappedSignals = signals.map((signal: any) => ({
    id: String(signal?.id ?? ''),
    chatTitle: String(signal?.chatTitle ?? ''),
    fromName: String(signal?.fromName ?? ''),
    contact: String(signal?.contact ?? ''),
    text: String(signal?.text ?? ''),
    date: String(signal?.date ?? ''),
    score: Number(signal?.score ?? 0),
    isLead: Boolean(signal?.isLead),
    semanticDirection: String(signal?.semanticDirection ?? ''),
    semanticCategory: String(signal?.semanticCategory ?? '')
  }))

  return {
    lead,
    signals: mappedSignals,
    signalsCount,
    lastSeenAt
  } as LeadBrief
})
