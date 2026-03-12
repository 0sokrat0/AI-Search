<script setup lang="ts">
import { h, resolveComponent, computed } from 'vue'
import type { TableColumn } from '@nuxt/ui'
import type { Period, Range, SignalItem } from '~/types'

const props = defineProps<{
  period: Period
  range: Range
}>()

interface SignalDigest {
  id: string
  date: string
  intent: 'lead_request' | 'service_offer' | 'market_discussion'
  source: string
  score: number
}

const UBadge = resolveComponent('UBadge')

const { data: signals } = await useFetch<SignalItem[]>('/api/signals', {
  watch: [() => props.period, () => props.range],
  default: () => []
})

const data = computed<SignalDigest[]>(() => {
  return (signals.value || [])
    .slice(0, 6)
    .map(signal => ({
      id: signal.id,
      date: signal.date,
      intent: detectIntent(signal),
      source: signal.chatTitle,
      score: Number(signal.similarityScore ?? signal.leadScore ?? 0)
    }))
})

function detectIntent(signal: SignalItem): SignalDigest['intent'] {
  if (signal.leadId) return 'lead_request'

  const d = (signal.semanticDirection || '').toLowerCase()
  if (d.includes('offer') || d.includes('предлож') || d.includes('ps')) return 'service_offer'
  if (d.includes('request') || d.includes('запрос') || d.includes('merchant') || d.includes('мерч')) return 'lead_request'

  return 'market_discussion'
}

const columns: TableColumn<SignalDigest>[] = [
  {
    accessorKey: 'id',
    header: 'Сигнал'
  },
  {
    accessorKey: 'date',
    header: 'Дата',
    cell: ({ row }) => {
      return new Date(row.getValue('date')).toLocaleString('ru-RU', {
        day: 'numeric',
        month: 'short',
        hour: '2-digit',
        minute: '2-digit',
        hour12: false
      })
    }
  },
  {
    accessorKey: 'intent',
    header: 'Намерение',
    cell: ({ row }) => {
      const value = row.getValue('intent') as SignalDigest['intent']
      const color = {
        lead_request: 'success' as const,
        service_offer: 'warning' as const,
        market_discussion: 'neutral' as const
      }[value]
      const label = {
        lead_request: 'Запрос лида',
        service_offer: 'Предложение услуги',
        market_discussion: 'Обсуждение рынка'
      }[value]
      return h(UBadge, { variant: 'subtle', color }, () => label)
    }
  },
  {
    accessorKey: 'source',
    header: 'Источник'
  },
  {
    accessorKey: 'score',
    header: () => h('div', { class: 'text-right' }, 'Оценка'),
    cell: ({ row }) => h('div', { class: 'text-right font-medium' }, Number(row.getValue('score')).toFixed(2))
  }
]
</script>

<template>
  <UTable
    :data="data"
    :columns="columns"
    class="shrink-0"
    :ui="{
      base: 'table-fixed border-separate border-spacing-0',
      thead: '[&>tr]:bg-elevated/50 [&>tr]:after:content-none',
      tbody: '[&>tr]:last:[&>td]:border-b-0',
      th: 'first:rounded-l-lg last:rounded-r-lg border-y border-default first:border-l last:border-r',
      td: 'border-b border-default'
    }"
  />
</template>
