<script setup lang="ts">
import { differenceInCalendarDays, sub } from 'date-fns'
import type { Period, IngestStats } from '~/types'

definePageMeta({
  middleware: 'auth'
})

const period = ref<Period>('weekly')
const range = ref({
  start: sub(new Date(), { days: 14 }),
  end: new Date()
})

const statsDays = computed(() => {
  const start = range.value.start
  const end = range.value.end
  if (!(start instanceof Date) || !(end instanceof Date)) return 30
  const days = Math.max(1, differenceInCalendarDays(end, start) + 1)
  return Math.min(days, 365)
})

const ingestDays = computed(() => Math.min(30, statsDays.value))

const { data: ingestStats } = await useFetch<IngestStats>('/api/signals/stats', {
  query: computed(() => ({ days: ingestDays.value })),
  watch: [ingestDays],
  default: (): IngestStats => ({
    period: '7d',
    totalSignals: 0,
    signalsToday: 0,
    signalsLastHour: 0,
    avgPerHour: 0,
    uniqueChats: 0,
    uniqueSenders: 0,
    leadCandidates: 0,
    teamMessages: 0,
    ignoredMessages: 0,
    lastSignalAt: null,
    hourly: [],
    topChats: []
  })
})

const candidateRate = computed(() => {
  const s = ingestStats.value
  if (!s || !s.totalSignals) return 0
  return Math.round((s.leadCandidates / s.totalSignals) * 100)
})
</script>

<template>
  <UDashboardPanel id="radar-home">
    <template #header>
      <UDashboardNavbar title="Панель радара лидов">
        <template #leading>
          <UDashboardSidebarCollapse />
        </template>

        <template #right>
          <HomePeriodSelect v-model="period" :range="range" />
          <HomeDateRangePicker v-model="range" />
        </template>
      </UDashboardNavbar>
    </template>

    <template #body>
      <HomeStats :period="period" :range="range" />

      <div class="grid grid-cols-1 xl:grid-cols-3 gap-4 mt-4">
        <div class="xl:col-span-2">
          <HomeChart :period="period" :range="range" />
        </div>

        <UCard>
          <template #header>
            <div class="flex items-center gap-2">
              <UIcon name="i-lucide-sliders-horizontal" class="size-4 text-muted" />
              <h3 class="font-semibold">Парсинг</h3>
            </div>
          </template>

          <div class="space-y-3 text-sm">
            <div class="flex justify-between items-center">
              <span class="text-muted">Сигналов за {{ ingestDays }}д</span>
              <span class="font-mono font-semibold">{{ ingestStats?.totalSignals ?? 0 }}</span>
            </div>
            <div class="flex justify-between items-center">
              <span class="text-muted">Кандидаты в лиды</span>
              <UBadge
                :color="(ingestStats?.totalSignals ?? 0) > 0 ? (candidateRate >= 20 ? 'success' : candidateRate >= 8 ? 'warning' : 'neutral') : 'neutral'"
                variant="subtle"
              >
                {{ (ingestStats?.totalSignals ?? 0) > 0 ? `${ingestStats?.leadCandidates ?? 0} (${candidateRate}%)` : 'нет данных' }}
              </UBadge>
            </div>
            <div class="flex justify-between items-center">
              <span class="text-muted">За последний час</span>
              <span class="font-semibold">{{ ingestStats?.signalsLastHour ?? 0 }}</span>
            </div>
            <div class="flex justify-between items-center">
              <span class="text-muted">Активные чаты / отправители</span>
              <span class="font-semibold">{{ ingestStats?.uniqueChats ?? 0 }} / {{ ingestStats?.uniqueSenders ?? 0 }}</span>
            </div>
            <div class="flex justify-between items-center">
              <span class="text-muted">Средний поток</span>
              <span class="font-semibold">{{ (ingestStats?.avgPerHour ?? 0).toFixed(1) }} / час</span>
            </div>
            <UButton to="/inbox" variant="ghost" color="neutral" size="xs" icon="i-lucide-radar" label="Открыть входящие сигналы" class="w-full mt-1" />
          </div>
        </UCard>
      </div>

      <UCard class="mt-4" :ui="{ body: 'p-0' }">
        <template #header>
          <div class="flex items-center justify-between">
            <h3 class="font-semibold">Последние сигналы</h3>
            <UButton to="/inbox" variant="ghost" color="neutral" icon="i-lucide-arrow-up-right" label="Открыть входящие" />
          </div>
        </template>

        <HomeSales :period="period" :range="range" />
      </UCard>
    </template>
  </UDashboardPanel>
</template>
