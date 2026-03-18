<script setup lang="ts">
import { eachDayOfInterval, eachWeekOfInterval, eachMonthOfInterval, format, parseISO, startOfDay, startOfMonth, startOfWeek } from 'date-fns'
import { VisXYContainer, VisLine, VisAxis, VisArea, VisCrosshair, VisTooltip } from '@unovis/vue'
import type { Period, Range, ChartDayBucket } from '~/types'

const cardRef = useTemplateRef<HTMLElement | null>('cardRef')

const props = defineProps<{
  period: Period
  range: Range
}>()

type DataRecord = {
  date: Date
  total: number
  target: number
}

const { width } = useElementSize(cardRef)

const data = ref<DataRecord[]>([])

const { data: chartBuckets } = await useFetch<ChartDayBucket[]>('/api/signals/chart', {
  query: computed(() => ({
    from: props.range.start.toISOString(),
    to: props.range.end.toISOString()
  })),
  default: () => [],
  watch: [() => props.range]
})

function toBucketKey(date: Date): string {
  if (props.period === 'daily') return format(startOfDay(date), 'yyyy-MM-dd')
  if (props.period === 'weekly') return format(startOfWeek(date, { weekStartsOn: 1 }), 'yyyy-MM-dd')
  return format(startOfMonth(date), 'yyyy-MM-dd')
}

watch([() => props.period, () => props.range, chartBuckets], () => {
  const dates = ({
    daily: eachDayOfInterval,
    weekly: eachWeekOfInterval,
    monthly: eachMonthOfInterval
  } as Record<Period, typeof eachDayOfInterval>)[props.period](props.range)

  const buckets = new Map<string, { total: number, target: number }>()
  for (const b of chartBuckets.value || []) {
    const dt = parseISO(b.day)
    const key = toBucketKey(dt)
    const existing = buckets.get(key) ?? { total: 0, target: 0 }
    buckets.set(key, { total: existing.total + b.total, target: existing.target + b.target })
  }

  data.value = dates.map((date) => {
    const key = toBucketKey(date)
    const b = buckets.get(key)
    return { date, total: b?.total ?? 0, target: b?.target ?? 0 }
  })
}, { immediate: true })

const x = (_: DataRecord, i: number) => i
const yTotal = (d: DataRecord) => d.total
const yTarget = (d: DataRecord) => d.target

const total = computed(() => data.value.reduce((acc: number, d) => acc + d.total, 0))
const targetTotal = computed(() => data.value.reduce((acc: number, d) => acc + d.target, 0))

const formatDate = (date: Date): string => {
  return ({
    daily: format(date, 'd MMM'),
    weekly: format(date, 'd MMM'),
    monthly: format(date, 'MMM yyyy')
  })[props.period]
}

const xTicks = (i: number) => {
  if (i === 0 || i === data.value.length - 1 || !data.value[i]) {
    return ''
  }
  return formatDate(data.value[i].date)
}

const tooltipTemplate = (d: DataRecord) => `${formatDate(d.date)}: всего ${d.total}, целевые ${d.target}`
</script>

<template>
  <UCard ref="cardRef" :ui="{ root: 'overflow-visible', body: '!px-0 !pt-0 !pb-3' }">
    <template #header>
      <div>
        <p class="text-xs text-muted uppercase mb-1.5 font-semibold tracking-wider">
          Интенсивность сигналов
        </p>
        <p class="text-3xl text-highlighted font-semibold">
          {{ total }}
        </p>
        <p class="text-sm text-muted mt-1">
          Целевые: <span class="text-success font-medium">{{ targetTotal }}</span>
        </p>
      </div>
    </template>

    <div v-if="data.length > 0">
      <VisXYContainer
        :data="data"
        :padding="{ top: 40, right: 20, left: 20 }"
        class="h-96"
        :width="width"
      >
        <VisLine
          :x="x"
          :y="yTotal"
          color="var(--ui-primary)"
        />
        <VisArea
          :x="x"
          :y="yTotal"
          color="var(--ui-primary)"
          :opacity="0.1"
        />
        <VisLine
          :x="x"
          :y="yTarget"
          color="var(--ui-success)"
        />

        <VisAxis
          type="x"
          :x="x"
          :tick-format="xTicks"
        />

        <VisCrosshair
          color="var(--ui-primary)"
          :template="tooltipTemplate"
        />

        <VisTooltip />
      </VisXYContainer>
      <div class="px-4 pt-2 flex items-center gap-4 text-xs text-muted border-t border-default/50 mt-2">
        <span class="inline-flex items-center gap-1"><span class="size-2 rounded-full bg-primary" /> Общий поток</span>
        <span class="inline-flex items-center gap-1"><span class="size-2 rounded-full bg-success" /> Целевые сигналы</span>
      </div>
    </div>
    <div v-else class="h-96 flex items-center justify-center text-muted text-sm">
      Нет данных за выбранный период
    </div>
  </UCard>
</template>

<style scoped>
.unovis-xy-container {
  --vis-crosshair-line-stroke-color: var(--ui-primary);
  --vis-crosshair-circle-stroke-color: var(--ui-bg);

  --vis-axis-grid-color: var(--ui-border);
  --vis-axis-tick-color: var(--ui-border);
  --vis-axis-tick-label-color: var(--ui-text-dimmed);

  --vis-tooltip-background-color: var(--ui-bg);
  --vis-tooltip-border-color: var(--ui-border);
  --vis-tooltip-text-color: var(--ui-text-highlighted);
}
</style>
