<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { breakpointsTailwind, useIntersectionObserver } from '@vueuse/core'
import type { CursorPage, Mail, SignalItem } from '~/types'

definePageMeta({
  middleware: 'auth'
})

type InboxTab = 'all' | 'lead'
type SignalCategoryFilter = 'all' | 'traders' | 'merchants' | 'ps_offers' | 'noise'

const tabItems: Array<{ label: string, value: InboxTab }> = [{
  label: 'Все',
  value: 'all'
}, {
  label: 'Лиды',
  value: 'lead'
}]

const categoryItems = computed<Array<{ label: string, value: SignalCategoryFilter }>>(() => {
  const items: Array<{ label: string, value: SignalCategoryFilter }> = [{
    label: 'Все теги',
    value: 'all'
  }, {
    label: 'Мерч',
    value: 'merchants'
  }, {
    label: 'Трейдер/Поиск трейдеров',
    value: 'traders'
  }, {
    label: 'Предложение ПС',
    value: 'ps_offers'
  }]

  if (selectedTab.value !== 'lead') {
    items.push({
      label: 'Шум',
      value: 'noise'
    })
  }

  return items
})

const selectedTab = ref<InboxTab>('all')
const selectedCategory = ref<SignalCategoryFilter>('all')
const selectedChat = ref<string>('all')
const showArchived = ref(false)
const selectedSignalIds = ref<string[]>([])
const bulkCategory = ref<'traders' | 'merchants' | 'ps_offers' | 'noise'>('traders')
const bulkLoading = ref(false)
const toast = useToast()
const queryClient = useQueryClient()
const route = useRoute()
const router = useRouter()
const loadMoreTrigger = useTemplateRef<HTMLElement | null>('loadMoreTrigger')
const signalsQueryKey = computed(() => ['signals', selectedTab.value, selectedCategory.value, showArchived.value] as const)

const {
  data: signalsPages,
  isPending,
  hasNextPage,
  isFetchingNextPage,
  fetchNextPage
} = useAuthInfiniteQuery<CursorPage<SignalItem>>(
  signalsQueryKey,
  ({ pageParam }) => $fetch<CursorPage<SignalItem>>('/api/signals/page', {
    query: {
      limit: 50,
      cursor: pageParam || undefined,
      tab: selectedTab.value,
      category: selectedCategory.value,
      show_archived: showArchived.value
    }
  }),
  {
    initialPageParam: '',
    getNextPageParam: (lastPage: CursorPage<SignalItem>) => lastPage.nextCursor || undefined,
    refetchInterval: 30_000,
    staleTime: 10_000
  }
)

const signals = computed<SignalItem[]>(() => (signalsPages.value?.pages as CursorPage<SignalItem>[] | undefined)?.flatMap((page: CursorPage<SignalItem>) => page.items) ?? [])
const { data: appSettings } = await useFetch('/api/settings', {
  default: () => ({ show_multi_account_badges: 'true' })
})

function isInboxTab(value: unknown): value is InboxTab {
  return value === 'all' || value === 'lead'
}

function isCategoryFilter(value: unknown): value is SignalCategoryFilter {
  return value === 'all'
    || value === 'traders'
    || value === 'merchants'
    || value === 'ps_offers'
    || value === 'noise'
}

function normalizeMailCategory(value?: string | null): Mail['category'] {
  switch (String(value || '').toLowerCase()) {
    case 'traders':
      return 'traders'
    case 'merchants':
    case 'processing_request':
    case 'processing_requests':
      return 'merchants'
    case 'ps_offers':
      return 'ps_offers'
    default:
      return 'noise'
  }
}

const allMailboxSignals = computed<Mail[]>(() => {
  return signals.value.map((signal: SignalItem, index: number) => ({
    id: index + 1,
    signalId: signal.id,
    unread: false,
    merchantName: signal.leadId ? leadCompanyMap.value[signal.leadId] : undefined,
    from: {
      id: index + 1,
      name: signal.fromName,
      email: signal.contact || signal.fromName,
      status: 'subscribed' as const,
      location: signal.chatTitle
    },
    telegramUsername: signal.contact || '',
    subject: signal.chatTitle,
    body: signal.text,
    date: signal.date,
    leadId: signal.leadId ?? null,
    leadScore: signal.leadScore ?? null,
    similarityScore: signal.similarityScore ?? null,
    classifiedAsLead: signal.classifiedAsLead ?? null,
    semanticDirection: signal.semanticDirection ?? null,
    semanticCategory: signal.semanticCategory ?? undefined,
    classificationReason: signal.classificationReason ?? undefined,
    traderScore: signal.traderScore ?? null,
    merchantScore: signal.merchantScore ?? null,
    processingRequestScore: signal.processingRequestScore ?? null,
    psOfferScore: signal.psOfferScore ?? null,
    noiseScore: signal.noiseScore ?? null,
    senderTelegramId: signal.senderTelegramId,
    isIgnored: signal.isIgnored,
    isTeamMember: signal.isTeamMember,
    isSpamSender: signal.isSpamSender ?? false,
    isDm: signal.isDm,
    otherChatsCount: signal.otherChatsCount,
    showMultiAccountBadges: appSettings.value?.show_multi_account_badges !== 'false',
    category: normalizeMailCategory(signal.semanticCategory),
    categoryReason: signal.classificationReason ?? ''
  }))
})

const chatItems = computed(() => {
  const counts = new Map<string, number>()
  for (const m of allMailboxSignals.value) {
    const chat = m.from.location || ''
    if (chat) counts.set(chat, (counts.get(chat) ?? 0) + 1)
  }
  const options = Array.from(counts.entries())
    .sort((a, b) => b[1] - a[1])
    .map(([chat, count]) => ({ label: `${chat} (${count})`, value: chat }))
  return [{ label: 'Все чаты', value: 'all' }, ...options]
})

const archivedCount = computed(() =>
  allMailboxSignals.value.filter(m => m.isIgnored).length
)

const mailboxSignals = computed<Mail[]>(() => {
  let result = allMailboxSignals.value
  if (selectedChat.value !== 'all') {
    result = result.filter(m => m.from.location === selectedChat.value)
  }
  return result
})

const selectedSignal = ref<Mail | null>()

const leadCompanyMap = ref<Record<string, string>>({})

function onCompanyAssigned(payload: { signalId: string, merchantName: string }) {
  const mail = mailboxSignals.value.find(m => m.signalId === payload.signalId)
  if (mail?.leadId) {
    leadCompanyMap.value = { ...leadCompanyMap.value, [mail.leadId]: payload.merchantName }
  }
}

const isSignalPanelOpen = computed({
  get() {
    return !!selectedSignal.value
  },
  set(value: boolean) {
    if (!value) {
      selectedSignal.value = null
    }
  }
})

watch(mailboxSignals, () => {
  if (selectedSignal.value && !mailboxSignals.value.find(s => s.signalId === selectedSignal.value?.signalId)) {
    selectedSignal.value = null
  }
  selectedSignalIds.value = selectedSignalIds.value.filter(id => mailboxSignals.value.some(s => s.signalId === id))
})

watch(
  () => route.query.tab,
  (tab) => {
    if (isInboxTab(tab)) {
      selectedTab.value = tab
      return
    }
    selectedTab.value = 'all'
  },
  { immediate: true }
)

watch(
  () => route.query.category,
  (category) => {
    if (category === 'processing_requests') {
      selectedCategory.value = 'merchants'
      return
    }
    if (isCategoryFilter(category)) {
      selectedCategory.value = category
      return
    }
    selectedCategory.value = 'all'
  },
  { immediate: true }
)

watch([selectedTab, selectedCategory], ([tab, category]) => {
  if (tab === 'lead' && category === 'noise') {
    selectedCategory.value = 'all'
    return
  }
  const nextQuery = {
    ...route.query,
    tab,
    category
  }

  if (route.query.tab === nextQuery.tab && route.query.category === nextQuery.category) {
    return
  }

  router.replace({ query: nextQuery })
})

useIntersectionObserver(loadMoreTrigger, async ([entry]) => {
  if (!entry?.isIntersecting || !hasNextPage.value || isFetchingNextPage.value) return
  await fetchNextPage()
})

function onFlagged(field: string, value: boolean) {
  if (!selectedSignal.value || !signals.value) return

  const signalId = selectedSignal.value.signalId

  if (field === 'is_ignored' && value && !showArchived.value) {
    const currentIndex = mailboxSignals.value.findIndex(s => s.signalId === signalId)
    selectedSignal.value = mailboxSignals.value[currentIndex + 1] ?? mailboxSignals.value[currentIndex - 1] ?? null
  }

  const signal = signals.value.find((s: SignalItem) => s.id === signalId)
  if (!signal) return

  if (field === 'is_ignored') {
    signal.isIgnored = value
  } else if (field === 'is_team_member') {
    signal.isTeamMember = value
  }
}

async function onFeedback(payload: { signalId: string }) {
  const currentIndex = mailboxSignals.value.findIndex(s => s.signalId === payload.signalId)
  if (currentIndex !== -1) {
    selectedSignalIds.value = selectedSignalIds.value.filter(id => id !== payload.signalId)
    const nextSignal = mailboxSignals.value[currentIndex + 1] || mailboxSignals.value[currentIndex - 1] || null
    selectedSignal.value = nextSignal
  }
  await queryClient.invalidateQueries({ queryKey: ['signals'] })
}

const breakpoints = useBreakpoints(breakpointsTailwind)
const isMobile = breakpoints.smaller('lg')

async function runBulkAction(action: 'archive' | 'team' | 'category') {
  const targets = mailboxSignals.value.filter(mail => selectedSignalIds.value.includes(mail.signalId))
  if (!targets.length) return
  bulkLoading.value = true
  try {
    await Promise.all(targets.map((mail) => {
      if (action === 'archive') {
        return $fetch(`/api/signals/${mail.signalId}/flag`, { method: 'POST', body: { field: 'is_ignored', value: true } })
      }
      if (action === 'team') {
        return $fetch(`/api/signals/${mail.signalId}/flag`, { method: 'POST', body: { field: 'is_team_member', value: true } })
      }
      return $fetch(`/api/signals/${mail.signalId}/feedback`, {
        method: 'POST',
        body: { is_lead: bulkCategory.value !== 'noise', category: bulkCategory.value }
      })
    }))
    toast.add({ title: 'Массовая операция выполнена', description: `Обработано сигналов: ${targets.length}`, color: 'success' })
    selectedSignalIds.value = []
    await queryClient.invalidateQueries({ queryKey: ['signals'] })
  } catch (e: any) {
    toast.add({ title: 'Ошибка bulk-операции', description: e?.message || 'Не удалось обработать сигналы', color: 'error' })
  } finally {
    bulkLoading.value = false
  }
}
</script>

<template>
  <UDashboardPanel
    v-if="!isMobile || !selectedSignal"
    id="inbox-1"
    :default-size="25"
    :min-size="20"
    :max-size="30"
    resizable
  >
    <UDashboardNavbar title="Сигналы">
      <template #leading>
        <UDashboardSidebarCollapse />
      </template>
      <template #trailing>
        <UBadge :label="mailboxSignals.length" variant="subtle" />
      </template>
    </UDashboardNavbar>

    <div class="px-3 pt-3 pb-3 border-b border-default flex items-center gap-2">
      <div class="flex items-center gap-1 flex-1 overflow-x-auto">
        <UButton
          v-for="tab in tabItems"
          :key="tab.value"
          :label="tab.label"
          :variant="selectedTab === tab.value ? 'soft' : 'ghost'"
          :color="selectedTab === tab.value ? 'primary' : 'neutral'"
          size="xs"
          class="shrink-0"
          @click="selectedTab = tab.value"
        />
      </div>

      <UTooltip :text="showArchived ? 'Скрыть архив' : `Архив${archivedCount ? ` (${archivedCount})` : ''}`">
        <UButton
          :icon="showArchived ? 'i-lucide-archive-x' : 'i-lucide-archive'"
          :variant="showArchived ? 'soft' : 'ghost'"
          :color="showArchived ? 'warning' : 'neutral'"
          size="xs"
          square
          :class="{ 'opacity-50': !showArchived && !archivedCount }"
          @click="showArchived = !showArchived"
        />
      </UTooltip>
    </div>

    <div class="px-3 py-3 border-b border-default space-y-2">
      <USelect
        v-model="selectedCategory"
        :items="categoryItems"
        size="sm"
        class="w-full"
      />
      <USelect
        v-model="selectedChat"
        :items="chatItems"
        size="sm"
        class="w-full"
        icon="i-lucide-message-square"
        placeholder="Все чаты"
      />

      <div v-if="selectedSignalIds.length" class="rounded-lg border border-default p-2.5 space-y-2">
        <p class="text-xs text-muted">
          Выбрано сигналов: {{ selectedSignalIds.length }}
        </p>
        <div class="flex flex-wrap items-center gap-2">
          <UButton
            size="xs"
            color="neutral"
            variant="soft"
            :loading="bulkLoading"
            @click="runBulkAction('archive')"
          >
            Архивировать
          </UButton>
          <UButton
            size="xs"
            color="info"
            variant="soft"
            :loading="bulkLoading"
            @click="runBulkAction('team')"
          >
            Отметить как команда
          </UButton>
          <USelect
            v-model="bulkCategory"
            :items="[
              { label: 'Трейдер/Поиск трейдеров', value: 'traders' },
              { label: 'Мерчант', value: 'merchants' },
              { label: 'Предложение ПС', value: 'ps_offers' },
              { label: 'Шум', value: 'noise' }
            ]"
            size="xs"
            class="min-w-44"
          />
          <UButton
            size="xs"
            color="primary"
            variant="soft"
            :loading="bulkLoading"
            @click="runBulkAction('category')"
          >
            Проставить тег
          </UButton>
        </div>
      </div>
    </div>

    <div v-if="isPending" class="p-4 space-y-3">
      <USkeleton class="h-16 w-full" />
      <USkeleton class="h-16 w-full" />
      <USkeleton class="h-16 w-full" />
    </div>
    <div v-else-if="!mailboxSignals.length" class="p-6">
      <UAlert
        color="neutral"
        variant="soft"
        title="Нет сигналов"
        description="Сообщения из Telegram чатов появятся здесь."
      />
    </div>
    <div v-else class="flex flex-col min-h-0">
      <InboxList
        v-model="selectedSignal"
        v-model:selected-ids="selectedSignalIds"
        :mails="mailboxSignals"
        class="pb-4"
      />
      <div ref="loadMoreTrigger" class="px-4 py-4">
        <div v-if="isFetchingNextPage" class="space-y-2">
          <USkeleton class="h-14 w-full" />
          <USkeleton class="h-14 w-full" />
        </div>
        <p v-else-if="hasNextPage" class="text-xs text-center text-muted">
          Прокрутите ниже, чтобы загрузить ещё сигналы
        </p>
      </div>
    </div>
  </UDashboardPanel>

  <InboxMail
    v-if="selectedSignal && !isMobile"
    :mail="selectedSignal"
    @close="selectedSignal = null"
    @flagged="onFlagged"
    @feedback="onFeedback"
    @company-assigned="onCompanyAssigned"
  />
  <div v-if="!selectedSignal && !isMobile" class="hidden lg:flex flex-1 items-center justify-center">
    <div class="flex flex-col items-center gap-3 text-center">
      <UIcon name="i-lucide-radar" class="size-20 text-dimmed" />
      <p class="text-sm text-muted">
        Выберите сигнал для просмотра деталей.
      </p>
    </div>
  </div>

  <ClientOnly>
    <div v-if="isMobile && selectedSignal" class="flex-1 flex flex-col h-full overflow-hidden bg-bg">
      <InboxMail
        :mail="selectedSignal"
        @close="selectedSignal = null"
        @flagged="onFlagged"
        @feedback="onFeedback"
        @company-assigned="onCompanyAssigned"
      />
    </div>
  </ClientOnly>
</template>
