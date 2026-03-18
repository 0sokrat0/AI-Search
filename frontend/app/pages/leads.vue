<script setup lang="ts">
import { formatDistanceToNow, isValid } from 'date-fns'
import { DynamicScroller, DynamicScrollerItem } from 'vue-virtual-scroller'
import type { CursorPage, Lead } from '~/types'

interface GroupedLead extends Lead {
  chatTitles: string[]
  allIds: string[]
}

definePageMeta({
  middleware: 'auth'
})

const toast = useToast()
const queryClient = useQueryClient()
const router = useRouter()
const route = useRoute()
const hydrated = ref(false)
const categoryFilter = ref<'all' | 'traders' | 'merchants' | 'ps_offers'>('all')
const selectedChat = ref('all')
const contact = ref('')

const isDetailRoute = computed(() => Boolean(route.params.id))

onMounted(() => {
  hydrated.value = true
})

const {
  data: leadPages,
  isPending,
  hasNextPage,
  isFetchingNextPage,
  fetchNextPage
} = useAuthInfiniteQuery<CursorPage<Lead>>(
  computed(() => ['leads', categoryFilter.value]),
  ({ pageParam }) => $fetch<CursorPage<Lead>>('/api/leads/page', {
    query: {
      category: categoryFilter.value === 'all' ? undefined : categoryFilter.value,
      qualified_only: true,
      limit: 50,
      cursor: pageParam || undefined
    }
  }),
  {
    initialPageParam: '',
    getNextPageParam: (lastPage: CursorPage<Lead>) => lastPage.nextCursor || undefined,
    refetchInterval: 60_000,
    staleTime: 15_000
  }
)

const data = computed<Lead[]>(() => (leadPages.value?.pages as CursorPage<Lead>[] | undefined)?.flatMap((page: CursorPage<Lead>) => page.items) ?? [])

const normalizedContactQuery = computed(() => String(contact.value || '').trim().toLowerCase())

const chatItems = computed(() => {
  const counts = new Map<string, number>()

  for (const lead of data.value) {
    const chat = String(lead.chatTitle || '').trim()
    if (!chat) continue
    counts.set(chat, (counts.get(chat) ?? 0) + 1)
  }

  const items = Array.from(counts.entries())
    .sort((a, b) => b[1] - a[1])
    .map(([chat, count]) => ({
      label: `${chat} (${count})`,
      value: chat
    }))

  return [{ label: 'Все чаты', value: 'all' }, ...items]
})

function normalizeCategory(raw?: string | null): 'traders' | 'merchants' | 'ps_offers' | 'noise' | '' {
  switch (String(raw || '').toLowerCase()) {
    case 'trader':
    case 'traders':
      return 'traders'
    case 'merchant':
    case 'merchants':
    case 'processing_requests':
      return 'merchants'
    case 'ps_offer':
    case 'ps_offers':
      return 'ps_offers'
    case 'noise':
      return 'noise'
    default:
      return ''
  }
}

const filteredLeads = computed(() => {
  return data.value.filter((lead: Lead) => {
    const matchesChat = selectedChat.value === 'all' || lead.chatTitle === selectedChat.value
    if (!matchesChat) return false

    const query = normalizedContactQuery.value
    if (!query) return true

    const haystack = [
      lead.name,
      lead.contact,
      lead.chatTitle,
      lead.text,
      lead.company
    ]
      .map(value => String(value || '').toLowerCase())
      .join(' ')

    return haystack.includes(query)
  })
})

const groupedScopedLeads = computed<GroupedLead[]>(() => {
  return filteredLeads.value.map((lead: Lead) => ({
    ...lead,
    chatTitles: lead.chatTitle ? [lead.chatTitle] : [],
    allIds: [lead.id]
  }))
})

const loadedLeadsCount = computed(() => groupedScopedLeads.value.length)

watch(categoryFilter, () => {
  selectedChat.value = 'all'
})

function formatLastSeen(value?: string) {
  if (!value) return '—'
  const date = new Date(value)
  if (!isValid(date)) return '—'
  return formatDistanceToNow(date, { addSuffix: true })
}

function openLead(id: string) {
  router.push(`/leads/${id}`)
}

function buildContactHref(raw?: string | null): string {
  const value = String(raw || '').trim()
  if (!value) return ''
  if (value.startsWith('@')) return `https://t.me/${value.slice(1)}`
  if (value.includes('@')) return `mailto:${value}`
  return ''
}

function categoryLabel(raw?: string | null): string {
  switch (normalizeCategory(raw)) {
    case 'traders':
      return 'Трейдеры / Поиск трейдеров'
    case 'merchants':
      return 'Мерчанты'
    case 'ps_offers':
      return 'Предложения от ПС'
    case 'noise':
      return 'Шум'
    default:
      return 'Без категории'
  }
}

function categoryColor(raw?: string | null): 'primary' | 'success' | 'warning' | 'error' | 'neutral' | 'info' {
  switch (normalizeCategory(raw)) {
    case 'traders':
      return 'success'
    case 'merchants':
      return 'info'
    case 'ps_offers':
      return 'primary'
    case 'noise':
      return 'warning'
    default:
      return 'neutral'
  }
}

function qualificationSourceLabel(source?: string | null): string {
  switch (String(source || '').toLowerCase()) {
    case 'ai_qualified':
      return 'Квалифицировано ИИ'
    case 'manual_approved':
      return 'Ручной апрув'
    default:
      return '—'
  }
}

function qualificationSourceColor(source?: string | null): 'primary' | 'success' | 'info' | 'neutral' {
  switch (String(source || '').toLowerCase()) {
    case 'ai_qualified':
      return 'info'
    case 'manual_approved':
      return 'success'
    default:
      return 'neutral'
  }
}

function scorePercent(value?: number | null): number | null {
  const numeric = Number(value ?? 0)
  if (!Number.isFinite(numeric) || numeric <= 0) return null
  return Math.round(numeric * 100)
}

function scrollerSizeDependencies(lead: GroupedLead) {
  return [
    lead.name,
    lead.contact,
    lead.chatTitle,
    lead.text,
    lead.company,
    lead.qualificationSource,
    lead.semanticCategory,
    lead.lastSeenAt,
    lead.signalsCount
  ]
}

async function handleLoadMore() {
  if (!hasNextPage.value || isFetchingNextPage.value) return
  await fetchNextPage()
}

async function deleteLead(id: string, name?: string) {
  const ok = window.confirm(`Удалить лид "${name || id}" из списка?`)
  if (!ok) return

  try {
    await $fetch(`/api/leads/${id}`, { method: 'DELETE' })
    await queryClient.invalidateQueries({ queryKey: ['leads'] })
    toast.add({
      title: 'Лид удален',
      description: 'Карточка удалена из списка',
      color: 'success'
    })
  } catch (error: any) {
    toast.add({
      title: 'Ошибка удаления',
      description: error?.message || 'Не удалось удалить лид',
      color: 'error'
    })
  }
}

function exportCSV() {
  const headers = ['ID', 'Имя', 'Контакт', 'Чат', 'Категория', 'Источник', 'Скор', 'Сигналы', 'Последний']
  const csvRows = groupedScopedLeads.value.map((lead: GroupedLead) => [
    lead.id,
    lead.name,
    lead.contact,
    lead.chatTitle,
    categoryLabel(lead.semanticCategory),
    qualificationSourceLabel(lead.qualificationSource),
    scorePercent(lead.score) ?? '',
    lead.signalsCount ?? 1,
    lead.lastSeenAt ?? ''
  ].map(value => `"${String(value).replace(/"/g, '""')}"`).join(','))

  const csv = [headers.join(','), ...csvRows].join('\n')
  const blob = new Blob(['\uFEFF' + csv], { type: 'text/csv;charset=utf-8;' })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = `квалифицированные_лиды_${new Date().toISOString().slice(0, 10)}.csv`
  link.click()
  URL.revokeObjectURL(url)
}
</script>

<template>
  <NuxtPage v-if="isDetailRoute" />

  <UDashboardPanel v-else id="leads">
    <template #header>
      <UDashboardNavbar title="Квалифицированные лиды">
        <template #leading>
          <UDashboardSidebarCollapse />
        </template>
      </UDashboardNavbar>
    </template>

    <template #body>
      <div class="flex flex-wrap items-center justify-between gap-2">
        <UInput
          v-model="contact"
          class="max-w-sm"
          icon="i-lucide-search"
          placeholder="Поиск по контакту, тексту или чату..."
        />

        <div class="flex flex-wrap items-center gap-2">
          <USelect
            v-model="categoryFilter"
            :items="[
              { label: 'Все категории', value: 'all' },
              { label: 'Трейдеры / Поиск трейдеров', value: 'traders' },
              { label: 'Мерчанты', value: 'merchants' },
              { label: 'Предложения от ПС', value: 'ps_offers' }
            ]"
            class="min-w-52"
          />

          <USelect
            v-model="selectedChat"
            :items="chatItems"
            :ui="{ trailingIcon: 'group-data-[state=open]:rotate-180 transition-transform duration-200' }"
            placeholder="Фильтр чата"
            class="min-w-52"
          />

          <UButton
            label="Экспорт"
            color="neutral"
            variant="outline"
            icon="i-lucide-download"
            @click="exportCSV"
          />
        </div>
      </div>

      <div v-if="isPending" class="space-y-3 py-4">
        <USkeleton class="h-24 w-full" />
        <USkeleton class="h-24 w-full" />
        <USkeleton class="h-24 w-full" />
      </div>

      <div v-else-if="!groupedScopedLeads.length" class="py-6">
        <UAlert
          color="neutral"
          variant="soft"
          title="Нет лидов"
          description="Подходящие сигналы появятся здесь после квалификации."
        />
      </div>

      <div v-else-if="hydrated" class="space-y-4">
        <DynamicScroller
          :items="groupedScopedLeads"
          key-field="id"
          :min-item-size="168"
          class="h-[calc(100vh-18rem)] min-h-[28rem] overflow-y-auto rounded-lg border border-default"
          @scroll-end="handleLoadMore"
        >
          <template #default="{ item, index, active }">
            <DynamicScrollerItem
              :item="item"
              :active="active"
              :size-dependencies="scrollerSizeDependencies(item as GroupedLead)"
              :data-index="index"
            >
              <template v-for="lead in [item as GroupedLead]" :key="lead.id">
                <div class="border-b border-default last:border-b-0">
                  <div class="flex flex-col gap-3 p-4 sm:p-5">
                    <div class="flex items-start justify-between gap-3">
                      <button
                        type="button"
                        class="flex min-w-0 items-center gap-3 text-left"
                        @click="openLead(lead.id)"
                      >
                        <UAvatar
                          v-bind="lead.avatar"
                          :alt="lead.name"
                          size="lg"
                        />
                        <div class="min-w-0 space-y-1">
                          <div class="flex flex-wrap items-center gap-2">
                            <p class="truncate font-medium text-highlighted">
                              {{ lead.name || 'Без имени' }}
                            </p>
                            <UBadge
                              :label="categoryLabel(lead.semanticCategory)"
                              :color="categoryColor(lead.semanticCategory)"
                              variant="subtle"
                              size="xs"
                            />
                            <UBadge
                              v-if="lead.qualificationSource"
                              :label="qualificationSourceLabel(lead.qualificationSource)"
                              :color="qualificationSourceColor(lead.qualificationSource)"
                              variant="soft"
                              size="xs"
                            />
                            <UBadge
                              v-if="scorePercent(lead.score) !== null"
                              :label="`${scorePercent(lead.score)}%`"
                              color="neutral"
                              variant="soft"
                              size="xs"
                            />
                          </div>
                          <p class="line-clamp-2 text-sm text-muted break-words">
                            {{ lead.text || 'Текст сигнала недоступен' }}
                          </p>
                        </div>
                      </button>

                      <div class="flex items-center gap-1">
                        <UButton
                          icon="i-lucide-eye"
                          color="neutral"
                          variant="ghost"
                          size="sm"
                          square
                          title="Открыть карточку"
                          @click="openLead(lead.id)"
                        />
                        <UButton
                          icon="i-lucide-trash-2"
                          color="error"
                          variant="ghost"
                          size="sm"
                          square
                          title="Удалить лид"
                          @click="deleteLead(lead.id, lead.name)"
                        />
                      </div>
                    </div>

                    <div class="flex flex-wrap items-center gap-2 text-xs text-muted">
                      <span v-if="lead.contact">Контакт: {{ lead.contact }}</span>
                      <span v-if="lead.chatTitle">Чат: {{ lead.chatTitle }}</span>
                      <span>Сигналов: {{ lead.signalsCount ?? 1 }}</span>
                      <span>Последний: {{ formatLastSeen(lead.lastSeenAt) }}</span>
                      <span v-if="lead.company">Компания: {{ lead.company }}</span>
                    </div>

                    <div class="flex flex-wrap items-center gap-2">
                      <UButton
                        v-if="buildContactHref(lead.contact)"
                        color="neutral"
                        variant="outline"
                        size="xs"
                        icon="i-lucide-send"
                        :href="buildContactHref(lead.contact)"
                        target="_blank"
                      >
                        Написать
                      </UButton>
                      <UButton
                        color="neutral"
                        variant="ghost"
                        size="xs"
                        trailing-icon="i-lucide-arrow-right"
                        @click="openLead(lead.id)"
                      >
                        Открыть детали
                      </UButton>
                    </div>
                  </div>
                </div>
              </template>
            </DynamicScrollerItem>
          </template>
        </DynamicScroller>

        <div class="space-y-2">
          <p class="text-xs text-center text-muted">
            <template v-if="hasNextPage">
              Загружено {{ loadedLeadsCount }} лидов. Список продолжится при прокрутке вниз.
            </template>
            <template v-else>
              Показано {{ loadedLeadsCount }} лидов.
            </template>
          </p>

          <div v-if="isFetchingNextPage" class="space-y-2">
            <USkeleton class="h-16 w-full" />
            <USkeleton class="h-16 w-full" />
          </div>

          <div v-else-if="hasNextPage" class="flex justify-center">
            <UButton
              size="sm"
              color="neutral"
              variant="soft"
              @click="handleLoadMore"
            >
              Загрузить ещё лиды
            </UButton>
          </div>
        </div>
      </div>

      <div v-else class="space-y-2 py-4">
        <USkeleton class="h-24 w-full" />
        <USkeleton class="h-24 w-full" />
      </div>
    </template>
  </UDashboardPanel>
</template>
