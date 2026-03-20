<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { breakpointsTailwind } from '@vueuse/core'
import type { CursorPage, Mail, SignalItem } from '~/types'

definePageMeta({
  middleware: 'auth'
})

type SignalCategoryFilter = 'all' | 'trader_search' | 'traders' | 'merchants' | 'ps_offers' | 'noise'

const categoryItems = computed<Array<{ label: string, value: SignalCategoryFilter }>>(() => {
  const items: Array<{ label: string, value: SignalCategoryFilter }> = [{
    label: 'Все теги',
    value: 'all'
  }, {
    label: 'Мерчанты',
    value: 'merchants'
  }, {
    label: 'Поиск трейдеров',
    value: 'trader_search'
  }, {
    label: 'Трейдеры',
    value: 'traders'
  }, {
    label: 'Предложение ПС',
    value: 'ps_offers'
  }, {
    label: 'Шум',
    value: 'noise'
  }]

  return items
})

const selectedCategory = ref<SignalCategoryFilter>('all')
const selectedChat = ref<string>('all')
const showArchived = ref(false)
const selectedSignalIds = ref<string[]>([])
const bulkCategory = ref<'trader_search' | 'traders' | 'merchants' | 'ps_offers' | 'noise'>('traders')
const bulkLoading = ref(false)
const cleanupNoiseLoading = ref(false)
const cleanupNoiseHours = ref('72')
const showCleanupNoiseModal = ref(false)
const toast = useToast()
const queryClient = useQueryClient()
const route = useRoute()
const router = useRouter()
const hydrated = ref(false)
const showLoadMoreSignalsButton = ref(false)
const signalsQueryKey = computed(() => ['signals', selectedCategory.value, showArchived.value] as const)

onMounted(() => {
  hydrated.value = true
})

function normalizeCleanupHours(value: string | number, fallback = 72): string {
  const parsed = Number(value)
  if (!Number.isFinite(parsed)) return String(fallback)
  return String(Math.min(8760, Math.max(1, Math.round(parsed))))
}

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
const loadedSignalsCount = computed(() => mailboxSignals.value.length)
const { data: appSettings } = await useFetch('/api/settings', {
  default: () => ({ show_multi_account_badges: 'true' })
})

function isCategoryFilter(value: unknown): value is SignalCategoryFilter {
  return value === 'all'
    || value === 'traders'
    || value === 'trader_search'
    || value === 'merchants'
    || value === 'ps_offers'
    || value === 'noise'
}

function normalizeMailCategory(value?: string | null): Mail['category'] {
  switch (String(value || '').toLowerCase()) {
    case 'trader_search':
      return 'trader_search'
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
    primaryLabel: signal.primaryLabel ?? null,
    primaryPercent: signal.primaryPercent ?? null,
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
  if (!hasNextPage.value) {
    showLoadMoreSignalsButton.value = false
  }
})

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

watch([selectedCategory], ([category]) => {
  showLoadMoreSignalsButton.value = false
  const nextQuery = {
    ...route.query,
    category,
    tab: undefined
  }

  if (route.query.category === nextQuery.category && route.query.tab === undefined) {
    return
  }

  router.replace({ query: nextQuery })
})

watch(selectedChat, () => {
  showLoadMoreSignalsButton.value = false
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

async function runBulkAction(action: 'team' | 'category') {
  const targets = mailboxSignals.value.filter(mail => selectedSignalIds.value.includes(mail.signalId))
  if (!targets.length) return
  bulkLoading.value = true
  try {
    await Promise.all(targets.map((mail) => {
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

async function runSelectedNoiseCleanup() {
  if (!selectedSignalIds.value.length) return

  cleanupNoiseHours.value = normalizeCleanupHours(cleanupNoiseHours.value)
  cleanupNoiseLoading.value = true
  try {
    const result = await $fetch<{ deleted: number, hours: number, message_ids?: number }>('/api/settings/cleanup-noise', {
      method: 'POST',
      body: {
        older_than_hours: cleanupNoiseHours.value,
        message_ids: selectedSignalIds.value
      }
    })
    toast.add({
      title: 'Очистка завершена',
      description: `Удалено шумовых сообщений: ${result.deleted}`,
      color: 'success'
    })
    selectedSignalIds.value = []
    showCleanupNoiseModal.value = false
    await queryClient.invalidateQueries({ queryKey: ['signals'] })
  } catch (e: any) {
    toast.add({
      title: 'Ошибка очистки',
      description: e?.message || 'Не удалось удалить выбранные шумовые сообщения',
      color: 'error'
    })
  } finally {
    cleanupNoiseLoading.value = false
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
              { label: 'Поиск трейдеров', value: 'trader_search' },
              { label: 'Трейдеры', value: 'traders' },
              { label: 'Мерчанты', value: 'merchants' },
              { label: 'Предложения от ПС', value: 'ps_offers' },
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
          <UButton
            size="xs"
            color="warning"
            variant="soft"
            :loading="cleanupNoiseLoading"
            @click="showCleanupNoiseModal = true"
          >
            Очистить шум из Mongo
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
    <div v-else-if="hydrated" class="flex flex-col min-h-0">
      <InboxList
        v-model="selectedSignal"
        v-model:selected-ids="selectedSignalIds"
        :mails="mailboxSignals"
        class="pb-4"
        @bottom-change="showLoadMoreSignalsButton = $event && hasNextPage"
      />
      <div class="px-4 py-4">
        <div v-if="isFetchingNextPage" class="space-y-2">
          <USkeleton class="h-14 w-full" />
          <USkeleton class="h-14 w-full" />
        </div>
        <div v-else-if="hasNextPage && showLoadMoreSignalsButton" class="space-y-2">
          <p class="text-xs text-center text-muted">
            Загружено {{ loadedSignalsCount }} сигналов. Это не весь список.
          </p>
          <div class="flex justify-center">
            <UButton
              size="sm"
              color="neutral"
              variant="soft"
              @click="fetchNextPage()"
            >
              Загрузить ещё сигналы
            </UButton>
          </div>
        </div>
        <p v-else class="text-xs text-center text-muted">
          Показаны все загруженные сигналы: {{ loadedSignalsCount }}
        </p>
      </div>
    </div>
    <div v-else class="p-4 space-y-3">
      <USkeleton class="h-16 w-full" />
      <USkeleton class="h-16 w-full" />
      <USkeleton class="h-16 w-full" />
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

  <UModal v-model:open="showCleanupNoiseModal" title="Очистка выбранного шума">
    <template #body>
      <div class="space-y-4">
        <p class="text-sm text-muted">
          Будут удалены только выбранные шумовые сообщения старше указанного периода.
        </p>
        <div class="space-y-2">
          <p class="text-sm font-medium">
            Старше, чем (часов)
          </p>
          <UInput
            v-model="cleanupNoiseHours"
            type="number"
            min="1"
            max="8760"
            placeholder="72"
          />
        </div>
      </div>
    </template>
    <template #footer>
      <div class="flex justify-end gap-2 w-full">
        <UButton color="neutral" variant="ghost" @click="showCleanupNoiseModal = false">
          Отмена
        </UButton>
        <UButton color="warning" :loading="cleanupNoiseLoading" @click="runSelectedNoiseCleanup">
          Удалить
        </UButton>
      </div>
    </template>
  </UModal>
</template>
