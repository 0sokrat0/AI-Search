<script setup lang="ts">
import { differenceInCalendarDays } from 'date-fns'
import type { Period, Range, LeadStats, ScoreBucket } from '~/types'

const props = defineProps<{
  period: Period
  range: Range
}>()

const statsDays = computed(() => {
  const start = props.range.start
  const end = props.range.end
  if (!(start instanceof Date) || !(end instanceof Date)) return 30
  const d = differenceInCalendarDays(end, start) + 1
  return Math.min(Math.max(1, d), 365)
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
    avgScore: 0,
    avgScoreApproved: 0,
    avgScoreRejected: 0,
    buckets: []
  })
})

// 10 fixed score buckets: 0–0.1, 0.1–0.2, …, 0.9–1.0
const allBuckets = computed<ScoreBucket[]>(() => {
  const map = new Map<string, ScoreBucket>()
  for (const b of (stats.value?.buckets ?? [])) {
    map.set(b.from.toFixed(1), b)
  }
  return Array.from({ length: 10 }, (_, i) => {
    const from = i / 10
    const key = from.toFixed(1)
    return map.get(key) ?? { from, to: (i + 1) / 10, count: 0, approved: 0, rejected: 0 }
  })
})

const hasData = computed(() => allBuckets.value.some(b => b.count > 0))
const maxCount = computed(() => Math.max(...allBuckets.value.map(b => b.count), 1))

const BAR_MAX_PX = 140

function barHeightPx(b: ScoreBucket): number {
  if (b.count === 0) return 4
  return Math.max(6, Math.round((b.count / maxCount.value) * BAR_MAX_PX))
}

function pct(part: number, total: number): string {
  if (!total) return '0%'
  return `${Math.round((part / total) * 100)}%`
}

const approvalRate = computed(() => {
  const s = stats.value
  if (!s?.totalDetected) return 0
  return Math.round((s.approved / s.totalDetected) * 100)
})

const falsePositiveRate = computed(() => {
  const s = stats.value
  if (!s?.totalDetected) return 0
  return Math.round((s.rejected / s.totalDetected) * 100)
})

// Hover state
const hoveredIndex = ref<number | null>(null)
</script>

<template>
  <UCard>
    <template #header>
      <div class="flex flex-wrap items-start justify-between gap-4">
        <div>
          <p class="text-xs text-muted uppercase tracking-wide mb-1.5">
            Качество детекции лидов
          </p>
          <div class="flex items-baseline gap-2">
            <p class="text-3xl font-semibold text-highlighted">
              {{ stats?.totalDetected ?? 0 }}
            </p>
            <p class="text-sm text-muted">
              лидов за {{ statsDays }} дн.
            </p>
          </div>
        </div>

        <div class="flex items-center gap-5 text-sm">
          <div class="flex flex-col items-end">
            <span class="text-success font-bold text-xl leading-tight">{{ stats?.approved ?? 0 }}</span>
            <span class="text-xs text-muted">подтверждено</span>
            <span class="text-xs font-medium text-success">{{ approvalRate }}%</span>
          </div>
          <div class="flex flex-col items-end">
            <span class="text-error font-bold text-xl leading-tight">{{ stats?.rejected ?? 0 }}</span>
            <span class="text-xs text-muted">ложных</span>
            <span class="text-xs font-medium text-error">{{ falsePositiveRate }}%</span>
          </div>
          <div class="flex flex-col items-end">
            <span class="text-warning font-bold text-xl leading-tight">{{ stats?.pending ?? 0 }}</span>
            <span class="text-xs text-muted">не проверено</span>
          </div>
        </div>
      </div>
    </template>

    <div v-if="!hasData" class="flex flex-col items-center justify-center py-12 text-muted gap-2">
      <UIcon name="i-lucide-bar-chart-2" class="size-8 opacity-30" />
      <p class="text-sm">
        Нет данных о лидах за выбранный период
      </p>
    </div>

    <div v-else>
      <p class="text-xs text-muted mb-4">
        Распределение по score — насколько уверенно ИИ определял лиды:
      </p>

      <!-- Bar chart -->
      <div class="flex items-end gap-1.5 px-1" :style="`height: ${BAR_MAX_PX + 24}px`">
        <div
          v-for="(bucket, i) in allBuckets"
          :key="i"
          class="flex-1 flex flex-col justify-end relative group cursor-default"
          @mouseenter="hoveredIndex = i"
          @mouseleave="hoveredIndex = null"
        >
          <!-- Tooltip -->
          <Transition name="fade">
            <div
              v-if="hoveredIndex === i && bucket.count > 0"
              class="absolute bottom-full mb-2 left-1/2 -translate-x-1/2 z-20 pointer-events-none"
            >
              <div class="bg-inverted text-inverted text-xs rounded-lg px-2.5 py-1.5 shadow-lg whitespace-nowrap text-center">
                <span class="font-mono font-medium">{{ Math.round(bucket.from * 100) }}–{{ Math.round(bucket.to * 100) }}%</span>
                <div class="mt-0.5 space-y-px">
                  <div>Всего: <b>{{ bucket.count }}</b></div>
                  <div class="text-success-300">✓ {{ bucket.approved }} подтверждено</div>
                  <div class="text-error-300">✗ {{ bucket.rejected }} отклонено</div>
                  <div v-if="bucket.count - bucket.approved - bucket.rejected > 0" class="opacity-70">
                    ⏳ {{ bucket.count - bucket.approved - bucket.rejected }} ожидает
                  </div>
                </div>
              </div>
              <div class="absolute left-1/2 -translate-x-1/2 top-full size-2 rotate-45 bg-inverted -mt-1" />
            </div>
          </Transition>

          <!-- Count label above bar -->
          <p
            v-if="bucket.count > 0"
            class="text-center text-[10px] font-medium mb-0.5 transition-colors"
            :class="hoveredIndex === i ? 'text-highlighted' : 'text-muted'"
          >
            {{ bucket.count }}
          </p>

          <!-- Bar itself -->
          <div
            class="w-full rounded-t overflow-hidden transition-all duration-200"
            :class="bucket.count === 0 ? 'rounded opacity-30' : ''"
            :style="`height: ${barHeightPx(bucket)}px`"
          >
            <template v-if="bucket.count > 0">
              <!-- Stacked from top to bottom: rejected / pending / approved -->
              <div class="h-full flex flex-col">
                <div
                  class="bg-error/60 transition-all"
                  :style="`height: ${pct(bucket.rejected, bucket.count)}`"
                />
                <div
                  class="bg-neutral/30 transition-all"
                  :style="`height: ${pct(bucket.count - bucket.approved - bucket.rejected, bucket.count)}`"
                />
                <div
                  class="bg-success/70 flex-1 transition-all"
                />
              </div>
            </template>
            <template v-else>
              <div class="h-full bg-border rounded" />
            </template>
          </div>

          <!-- X-axis label -->
          <p class="text-center font-mono mt-1 text-[10px] text-muted">
            {{ Math.round(bucket.from * 100) }}
          </p>
        </div>
      </div>

      <!-- Legend + avg scores -->
      <div class="mt-3 flex flex-wrap items-center justify-between gap-3 px-1">
        <div class="flex items-center gap-4 text-xs text-muted">
          <span class="inline-flex items-center gap-1.5">
            <span class="size-2.5 rounded-sm bg-success/70" />
            Подтверждено
          </span>
          <span class="inline-flex items-center gap-1.5">
            <span class="size-2.5 rounded-sm bg-neutral/30" />
            Не проверено
          </span>
          <span class="inline-flex items-center gap-1.5">
            <span class="size-2.5 rounded-sm bg-error/60" />
            Ложные
          </span>
        </div>
        <div v-if="stats?.avgScoreApproved" class="flex items-center gap-3 text-xs text-muted">
          <span>
            Ср. score подтверждённых:
            <b class="text-success">{{ stats.avgScoreApproved.toFixed(2) }}</b>
          </span>
          <span v-if="stats.avgScoreRejected">
            Ложных:
            <b class="text-error">{{ stats.avgScoreRejected.toFixed(2) }}</b>
          </span>
        </div>
      </div>
    </div>
  </UCard>
</template>

<style scoped>
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.1s ease;
}
.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
