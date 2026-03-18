<script setup lang="ts">
import { differenceInCalendarDays } from 'date-fns'
import type { Period, Range, LeadStats } from '~/types'

const props = defineProps<{
  period: Period
  range: Range
}>()

const statsDays = computed(() => {
  const start = props.range.start
  const end = props.range.end
  if (!(start instanceof Date) || !(end instanceof Date)) return 30
  const days = Math.max(1, differenceInCalendarDays(end, start) + 1)
  return Math.min(days, 365)
})

const { data: stats } = await useFetch<LeadStats>('/api/leads/stats', {
  query: computed(() => ({ days: statsDays.value })),
  watch: [statsDays],
  default: (): LeadStats => ({
    period: '30d',
    totalDetected: 0,
    approved: 0,
    rejected: 0,
    pending: 0,
    aiQualified: 0,
    manualApproved: 0,
    avgScore: 0,
    avgScoreApproved: 0,
    avgScoreRejected: 0,
    buckets: [],
    approvedByCategory: { traders: 0, merchants: 0, psOffers: 0 },
    rejectedByCategory: { traders: 0, merchants: 0, psOffers: 0 },
    series: []
  })
})

function formatDistribution(distribution: LeadStats['approvedByCategory']) {
  return `Тр ${distribution.traders} · М ${distribution.merchants} · ПС ${distribution.psOffers}`
}

const cards = computed(() => {
  const s = stats.value ?? {
    totalDetected: 0,
    approved: 0,
    rejected: 0,
    aiQualified: 0,
    manualApproved: 0,
    avgScoreRejected: 0,
    approvedByCategory: { traders: 0, merchants: 0, psOffers: 0 },
    rejectedByCategory: { traders: 0, merchants: 0, psOffers: 0 }
  }
  const totalDecisions = s.approved + s.rejected
  const approvalRate = totalDecisions > 0
    ? Math.round((s.approved / totalDecisions) * 100)
    : 0
  return [
    {
      title: 'Сигналов обнаружено',
      icon: 'i-lucide-radar',
      value: s.totalDetected,
      sub: `за ${statsDays.value} дн.`
    },
    {
      title: 'Лидов подтверждено',
      icon: 'i-lucide-badge-check',
      value: s.approved,
      sub: totalDecisions > 0 ? `${approvalRate}% · ${formatDistribution(s.approvedByCategory)}` : formatDistribution(s.approvedByCategory)
    },
    {
      title: 'Ложных срабатываний',
      icon: 'i-lucide-x-circle',
      value: s.rejected,
      sub: s.rejected > 0 ? formatDistribution(s.rejectedByCategory) : 'нет'
    },
    {
      title: 'Квалифицировано ИИ',
      icon: 'i-lucide-brain-circuit',
      value: s.aiQualified,
      sub: `Ручной апрув: ${s.manualApproved}`
    }
  ]
})
</script>

<template>
  <UPageGrid class="lg:grid-cols-4 gap-4 sm:gap-6 lg:gap-px">
    <UPageCard
      v-for="(card, index) in cards"
      :key="index"
      :icon="card.icon"
      :title="card.title"
      to="/leads"
      variant="subtle"
      :ui="{
        container: 'gap-y-1.5',
        wrapper: 'items-start',
        leading: 'p-2.5 rounded-full bg-primary/10 ring ring-inset ring-primary/25 flex-col',
        title: 'font-normal text-muted text-xs uppercase'
      }"
      class="lg:rounded-none first:rounded-l-lg last:rounded-r-lg hover:z-1"
    >
      <div class="flex items-center gap-2">
        <span class="text-2xl font-semibold text-highlighted">{{ card.value }}</span>
      </div>
      <p class="text-xs text-dimmed mt-0.5">
        {{ card.sub }}
      </p>
    </UPageCard>
  </UPageGrid>
</template>
