<script setup lang="ts">
import { eachDayOfInterval, eachWeekOfInterval, eachMonthOfInterval, format, parseISO, startOfDay, startOfMonth, startOfWeek } from 'date-fns'
import { VisXYContainer, VisLine, VisAxis, VisCrosshair, VisTooltip } from '@unovis/vue'
import type { Period, Range, ChartDayBucket } from '~/types'

const cardRef = useTemplateRef<HTMLElement | null>('cardRef')

const props = defineProps<{
  period: Period
  range: Range
}>()

type DataRecord = {
  date: Date
  traders: number
  merchants: number
  psOffers: number
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

  const buckets = new Map<string, { traders: number, merchants: number, psOffers: number }>()
  for (const b of chartBuckets.value || []) {
    const dt = parseISO(b.day)
    const key = toBucketKey(dt)
    const existing = buckets.get(key) ?? { traders: 0, merchants: 0, psOffers: 0 }
    buckets.set(key, {
      traders: existing.traders + b.traders,
      merchants: existing.merchants + b.merchants,
      psOffers: existing.psOffers + b.psOffers
    })
  }

  data.value = dates.map((date) => {
    const key = toBucketKey(date)
    const b = buckets.get(key)
    return {
      date,
      traders: b?.traders ?? 0,
      merchants: b?.merchants ?? 0,
      psOffers: b?.psOffers ?? 0
    }
  })
}, { immediate: true })

const x = (_: DataRecord, i: number) => i
const yTraders = (d: DataRecord) => d.traders
const yMerchants = (d: DataRecord) => d.merchants
const yPSOffers = (d: DataRecord) => d.psOffers

const tradersTotal = computed(() => data.value.reduce((acc: number, d) => acc + d.traders, 0))
const merchantsTotal = computed(() => data.value.reduce((acc: number, d) => acc + d.merchants, 0))
const psOffersTotal = computed(() => data.value.reduce((acc: number, d) => acc + d.psOffers, 0))

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

const tooltipTemplate = (d: DataRecord) => `${formatDate(d.date)}: трейдеры ${d.traders}, мерчанты ${d.merchants}, ПС ${d.psOffers}`
</script>

<template>
  <UCard ref="cardRef" :ui="{ root: 'overflow-visible', body: '!px-0 !pt-0 !pb-3' }">
    <template #header>
      <div>
        <p class="text-xs text-muted uppercase mb-1.5 font-semibold tracking-wider">
          Классификация по типам
        </p>
        <p class="text-sm text-muted mt-1">
          Тр <span class="font-medium text-success">{{ tradersTotal }}</span> ·
          М <span class="font-medium text-info">{{ merchantsTotal }}</span> ·
          ПС <span class="font-medium text-primary">{{ psOffersTotal }}</span>
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
          :y="yTraders"
          color="var(--ui-success)"
        />
        <VisLine
          :x="x"
          :y="yMerchants"
          color="var(--ui-info)"
        />
        <VisLine
          :x="x"
          :y="yPSOffers"
          color="var(--ui-primary)"
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
        <span class="inline-flex items-center gap-1"><span class="size-2 rounded-full bg-success" /> Трейдеры</span>
        <span class="inline-flex items-center gap-1"><span class="size-2 rounded-full bg-info" /> Мерчанты</span>
        <span class="inline-flex items-center gap-1"><span class="size-2 rounded-full bg-primary" /> Предложения ПС</span>
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
