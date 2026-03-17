<script setup lang="ts">
import { format } from 'date-fns'
import type { Mail, SignalItem, Lead, LeadBrief, LeadStatus } from '~/types'

const props = defineProps<{
  mail: Mail
}>()

const emits = defineEmits<{
  'close': []
  'feedback': [payload: { id: number, signalId: string, isLead: boolean }]
  'flagged': [field: string, value: boolean]
  'company-assigned': [payload: { signalId: string, merchantName: string }]
}>()

const toast = useToast()
const auth = useAuthStore()
const canSeeTechnicalSignals = computed(() => auth.isSuperAdmin)

const feedbackLoading = ref<'lead' | 'noise' | null>(null)
const feedbackDone = ref<boolean | null>(null)
const categoryFeedbackLoading = ref(false)
const historyCollapsed = ref(true)

const isIgnored = ref(props.mail.isIgnored)
const isTeamMember = ref(props.mail.isTeamMember)
const isSpamSender = ref(props.mail.isSpamSender ?? false)
const flagLoading = ref<string | null>(null)

const leadBriefData = shallowRef<LeadBrief | null>(null)

async function fetchLeadBrief(leadId: string | null | undefined) {
  if (!leadId) {
    leadBriefData.value = null
    selectedCompanyId.value = ''
    return
  }
  try {
    const brief = await $fetch<LeadBrief>(`/api/leads/${leadId}/brief`)
    const lead = brief?.lead
    leadBriefData.value = brief
    const merchantId = lead?.merchantId || lead?.companyId || ''
    selectedCompanyId.value = merchantId
  } catch {
    leadBriefData.value = null
  }
}

const leadData = computed<Lead | null>(() => {
  const brief = leadBriefData.value
  return brief?.lead ?? null
})

const leadStatusColor: Record<LeadStatus, 'primary' | 'success' | 'warning' | 'error' | 'neutral'> = {
  new: 'primary',
  contacted: 'warning',
  qualified: 'success',
  converted: 'success',
  rejected: 'error'
}
const leadStatusLabel: Record<LeadStatus, string> = {
  new: 'Новый',
  contacted: 'Первичный контакт',
  qualified: 'В работе',
  converted: 'Подключен',
  rejected: 'Закрыт'
}
const leadCategoryColor: Record<string, 'primary' | 'success' | 'warning' | 'info' | 'neutral'> = {
  leads: 'primary',
  traders: 'success',
  merchants: 'info',
  processing_requests: 'info',
  ps_offers: 'info',
  noise: 'neutral'
}
const leadCategoryLabel: Record<string, string> = {
  leads: 'Лид',
  traders: 'Трейдер/Поиск трейдеров',
  merchants: 'Мерчант',
  processing_requests: 'Мерчант',
  ps_offers: 'Предложение ПС',
  noise: 'Шум'
}

const { companies, loading: companiesLoading, addCompany, refresh: refreshCompanies } = useCompanies()

const selectedCompanyId = ref('')
const assigningCompany = ref(false)
const showAddCompanyModal = ref(false)
const newCompanyName = ref('')
const addingCompany = ref(false)

const companySelectItems = computed(() =>
  companies.value.map(c => ({ label: c.name, value: c.id }))
)

const assignedCompanyName = computed(() =>
  companies.value.find(c => c.id === selectedCompanyId.value)?.name ?? null
)

watch([leadData, companies], ([lead]) => {
  if (!lead?.merchantId || !props.mail.leadId) return
  const name = companies.value.find(c => c.id === lead.merchantId)?.name
  if (name) {
    emits('company-assigned', { signalId: props.mail.signalId, merchantName: name })
  }
})

async function assignCompany(companyId: string) {
  if (!companyId || !props.mail.leadId) return
  assigningCompany.value = true
  try {
    await $fetch(`/api/leads/${props.mail.leadId}/merchant`, {
      method: 'PUT',
      body: { merchant_id: companyId }
    })
    const name = companies.value.find(c => c.id === companyId)?.name ?? ''
    toast.add({ title: 'Компания назначена', description: name, color: 'success' })
    emits('company-assigned', { signalId: props.mail.signalId, merchantName: name })
  } catch (e: any) {
    toast.add({ title: 'Ошибка', description: e?.message, color: 'error' })
  } finally {
    assigningCompany.value = false
  }
}

async function createCompany() {
  const name = newCompanyName.value.trim()
  if (!name) return
  addingCompany.value = true
  try {
    await addCompany(name)
    await refreshCompanies()
    const created = [...companies.value].reverse().find(c => c.name === name)
    if (created) {
      selectedCompanyId.value = created.id
      await assignCompany(created.id)
    }
    newCompanyName.value = ''
    showAddCompanyModal.value = false
  } catch (e: any) {
    toast.add({ title: 'Ошибка', description: e?.message, color: 'error' })
  } finally {
    addingCompany.value = false
  }
}

const {
  data: senderHistory,
  status: historyStatus,
  refresh: refreshHistory
} = await useFetch<SignalItem[]>(
  () => `/api/signals/sender/${props.mail.senderTelegramId}`,
  { default: () => [] as SignalItem[] }
)

watch(
  () => props.mail.senderTelegramId,
  () => { refreshHistory() }
)

const otherSignals = computed(() =>
  senderHistory.value.filter(s => s.id !== props.mail.signalId)
)

watch(
  () => props.mail,
  (mail) => {
    isIgnored.value = mail.isIgnored
    isTeamMember.value = mail.isTeamMember
    isSpamSender.value = mail.isSpamSender ?? false
    feedbackDone.value = null
    historyCollapsed.value = true
    leadBriefData.value = null
    fetchLeadBrief(mail.leadId)
  }
)

fetchLeadBrief(props.mail.leadId)

const categoryLabel = computed(() => {
  switch (props.mail.category) {
    case 'traders': return 'Трейдер/Поиск трейдеров'
    case 'merchants': return 'Мерчант'
    case 'ps_offers': return 'Предложение ПС'
    default: return 'Шум'
  }
})

const categoryColor = computed<'info' | 'primary' | 'neutral'>(() => {
  switch (props.mail.category) {
    case 'traders':
    case 'merchants':
      return 'info'
    case 'ps_offers': return 'primary'
    default: return 'neutral'
  }
})

function normalizeDirection(value: string | null | undefined): string {
  const normalized = String(value || '').trim().toLowerCase()
  switch (normalized) {
    case 'trader':
    case 'traders':
      return 'traders'
    case 'merchant':
    case 'merchants':
    case 'merch':
      return 'merchants'
    case 'processing_request':
    case 'processing_requests':
    case 'request':
      return 'merchants'
    case 'ps_offer':
    case 'ps_offers':
    case 'offer':
    case 'offers':
      return 'ps_offers'
    case 'noise':
    case 'spam':
      return 'noise'
    default:
      return normalized
  }
}

const technicalDirectionLabel = computed(() => {
  const direction = normalizeDirection(props.mail.semanticDirection)
  switch (direction) {
    case 'traders': return 'Трейдер/Поиск трейдеров'
    case 'merchants': return 'Мерчант'
    case 'ps_offers': return 'Предложение ПС'
    case 'noise': return 'Шум'
    default: return ''
  }
})

const showTechnicalDirection = computed(() => {
  if (!canSeeTechnicalSignals.value) return false
  if (props.mail.category === 'noise') return false
  if (!props.mail.semanticDirection) return false
  const direction = normalizeDirection(props.mail.semanticDirection)
  if (!direction || direction === 'noise') return false
  return direction !== props.mail.category
})

const telegramHref = computed(() => {
  const username = String(props.mail.telegramUsername || '').trim()
  if (username.startsWith('@')) {
    return `https://t.me/${username.slice(1)}`
  }
  if (username) {
    return `https://t.me/${username}`
  }
  return ''
})

type TimelineMessage = {
  id: string
  role: 'user' | 'assistant'
  parts: Array<{ type: 'text', text: string }>
  metaDate: string
  metaChat: string
  metaLead: boolean
}

const historyTimeline = computed<TimelineMessage[]>(() => {
  const items: TimelineMessage[] = [...otherSignals.value]
    .sort((a, b) => {
      const ta = new Date(a.date).getTime()
      const tb = new Date(b.date).getTime()
      if (Number.isNaN(ta) || Number.isNaN(tb)) return 0
      return ta - tb
    })
    .map(signal => ({
      id: signal.id,
      role: 'assistant' as const,
      parts: [{ type: 'text' as const, text: signal.text || '—' }],
      metaDate: formatDateSafe(signal.date),
      metaChat: signal.chatTitle || '—',
      metaLead: Boolean(signal.leadId)
    }))

  items.push({
    id: props.mail.signalId,
    role: 'user',
    parts: [{ type: 'text', text: props.mail.body || '—' }],
    metaDate: formatDateSafe(props.mail.date),
    metaChat: props.mail.subject || '—',
    metaLead: Boolean(props.mail.leadId)
  })

  return items
})

async function sendFeedback(isLead: boolean) {
  feedbackLoading.value = isLead ? 'lead' : 'noise'
  try {
    await $fetch(`/api/signals/${props.mail.signalId}/feedback`, {
      method: 'POST',
      body: { is_lead: isLead }
    })
    feedbackDone.value = isLead
    toast.add({
      title: isLead ? 'Добавлено как лид' : 'Добавлено как шум',
      description: 'RAG обновлён — следующие похожие сообщения будут классифицированы точнее',
      color: isLead ? 'success' : 'neutral'
    })
    emits('feedback', { id: props.mail.id, signalId: props.mail.signalId, isLead })
    if (!isLead) {
      await setFlag('is_ignored', true)
    }
  } catch (e: any) {
    toast.add({ title: 'Ошибка', description: e?.message, color: 'error' })
  } finally {
    feedbackLoading.value = null
  }
}

async function sendCategoryFeedback(category: 'traders' | 'merchants' | 'ps_offers' | 'noise') {
  categoryFeedbackLoading.value = true
  try {
    await $fetch(`/api/signals/${props.mail.signalId}/feedback`, {
      method: 'POST',
      body: { is_lead: category !== 'noise', category }
    })
    toast.add({
      title: 'Категория сохранена',
      description: `Сигнал добавлен в обучение: ${category}`,
      color: 'success'
    })
    emits('feedback', { id: props.mail.id, signalId: props.mail.signalId, isLead: category !== 'noise' })
  } catch (e: any) {
    toast.add({ title: 'Ошибка', description: e?.message, color: 'error' })
  } finally {
    categoryFeedbackLoading.value = false
  }
}

async function setFlag(field: 'is_ignored' | 'is_team_member' | 'is_spam_sender', value: boolean) {
  flagLoading.value = field
  try {
    await $fetch(`/api/signals/${props.mail.signalId}/flag`, {
      method: 'POST',
      body: { field, value }
    })
    if (field === 'is_ignored') {
      isIgnored.value = value
    } else if (field === 'is_team_member') {
      isTeamMember.value = value
    } else {
      isSpamSender.value = value
    }
    emits('flagged', field, value)
    const messages: Record<string, string> = {
      is_ignored: value ? 'Сигнал архивирован' : 'Сигнал восстановлен из архива',
      is_team_member: value ? 'Отмечен как сотрудник' : 'Отметка сотрудника снята',
      is_spam_sender: value ? 'Отправитель помечен как спам — новые сообщения будут игнорироваться' : 'Отметка спама снята'
    }
    toast.add({
      title: 'Обновлено',
      description: messages[field],
      color: 'success'
    })
  } catch (e: any) {
    toast.add({ title: 'Ошибка', description: e?.message, color: 'error' })
  } finally {
    flagLoading.value = null
  }
}

function parseDateSafe(value?: string | null): Date | null {
  if (!value) return null
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? null : date
}

function formatDateSafe(value?: string | null): string {
  const date = parseDateSafe(value)
  if (!date) return '—'
  return format(date, 'dd MMM HH:mm')
}
</script>

<template>
  <UDashboardPanel id="inbox-2">
    <UDashboardNavbar :title="mail.subject" :toggle="false">
      <template #leading>
        <UButton
          icon="i-lucide-x"
          color="neutral"
          variant="ghost"
          class="-ms-1.5"
          @click="emits('close')"
        />
      </template>

      <template #right>
        <UTooltip :text="isIgnored ? 'Восстановить из архива' : 'Архивировать'">
          <UButton
            :icon="isIgnored ? 'i-lucide-archive-x' : 'i-lucide-archive'"
            :color="isIgnored ? 'warning' : 'neutral'"
            variant="ghost"
            square
            :loading="flagLoading === 'is_ignored'"
            :disabled="flagLoading !== null"
            @click="setFlag('is_ignored', !isIgnored)"
          />
        </UTooltip>

        <UTooltip :text="isTeamMember ? 'Снять метку сотрудника' : 'Отметить как сотрудника'">
          <UButton
            :icon="isTeamMember ? 'i-lucide-user-x' : 'i-lucide-user-check'"
            :color="isTeamMember ? 'neutral' : 'info'"
            variant="ghost"
            square
            :loading="flagLoading === 'is_team_member'"
            :disabled="flagLoading !== null"
            @click="setFlag('is_team_member', !isTeamMember)"
          />
        </UTooltip>

        <UTooltip :text="isSpamSender ? 'Снять метку спама' : 'Спам-отправитель — игнорировать навсегда'">
          <UButton
            :icon="isSpamSender ? 'i-lucide-shield-x' : 'i-lucide-shield-off'"
            :color="isSpamSender ? 'error' : 'neutral'"
            variant="ghost"
            square
            :loading="flagLoading === 'is_spam_sender'"
            :disabled="flagLoading !== null"
            @click="setFlag('is_spam_sender', !isSpamSender)"
          />
        </UTooltip>

        <UTooltip v-if="telegramHref" text="Открыть в Telegram">
          <UButton
            icon="i-lucide-send"
            color="neutral"
            variant="ghost"
            :href="telegramHref"
            target="_blank"
          />
        </UTooltip>
      </template>
    </UDashboardNavbar>

    <div class="p-4 sm:px-6 border-b border-default space-y-3">
      <div class="flex items-start gap-3">
        <UAvatar :alt="mail.from.name" size="xl" />
        <div class="min-w-0 flex-1">
          <p class="font-semibold text-highlighted truncate">
            {{ mail.from.name }}
          </p>
          <p class="text-muted text-sm truncate">
            {{ mail.from.email }}
          </p>
          <p class="text-xs text-dimmed mt-0.5 truncate">
            {{ mail.from.location }}
          </p>
        </div>
        <p class="text-xs text-muted shrink-0">
          {{ formatDateSafe(mail.date) }}
        </p>
      </div>

      <div class="flex flex-wrap items-center gap-1.5">
        <UBadge :color="categoryColor" variant="soft" size="xs">
          {{ categoryLabel }}
        </UBadge>
        <UBadge
          v-if="mail.semanticFlags?.includes('has_traffic')"
          icon="i-lucide-check-circle-2"
          label="Предлагают трафик"
          color="success"
          variant="subtle"
          size="xs"
        />
        <UBadge
          v-if="mail.isDm"
          icon="i-heroicons-envelope"
          label="Личное сообщение"
          color="neutral"
          variant="subtle"
          size="xs"
        />
        <UBadge
          :icon="isTeamMember ? 'i-lucide-users' : 'i-lucide-user'"
          :label="isTeamMember ? 'Команда' : 'Внешний контакт'"
          :color="isTeamMember ? 'info' : 'neutral'"
          variant="subtle"
          size="xs"
        />
        <UBadge
          v-if="isIgnored"
          icon="i-lucide-archive"
          label="Архив"
          color="warning"
          variant="subtle"
          size="xs"
        />
        <UBadge
          v-if="isSpamSender"
          icon="i-lucide-shield-off"
          label="Спам-отправитель"
          color="error"
          variant="subtle"
          size="xs"
        />
        <UTooltip v-if="mail.otherChatsCount > 1" text="Встречался в нескольких чатах">
          <UBadge
            icon="i-lucide-messages-square"
            :label="`${mail.otherChatsCount} чатов`"
            color="warning"
            variant="subtle"
            size="xs"
          />
        </UTooltip>
        <UBadge
          v-if="showTechnicalDirection"
          icon="i-lucide-compass"
          :label="technicalDirectionLabel"
          color="info"
          variant="subtle"
          size="xs"
        />
      </div>
    </div>

    <div v-if="mail.leadId" class="px-4 sm:px-6 py-3 border-b border-default">
      <USkeleton v-if="!leadData" class="h-14 w-full rounded-lg" />

      <div v-else class="space-y-2.5">
        <div class="flex items-center gap-1.5 flex-wrap">
          <UIcon name="i-lucide-user-search" class="size-3.5 text-muted shrink-0" />
          <span class="text-xs font-medium text-muted">Лид</span>
          <UBadge :color="leadStatusColor[leadData.status]" variant="subtle" size="xs">
            {{ leadStatusLabel[leadData.status] }}
          </UBadge>
          <UBadge
            v-if="leadData.semanticCategory"
            :color="leadCategoryColor[leadData.semanticCategory] ?? 'neutral'"
            variant="soft"
            size="xs"
          >
            {{ leadCategoryLabel[leadData.semanticCategory] ?? leadData.semanticCategory }}
          </UBadge>
          <div v-if="leadData.geo.length || leadData.products.length" class="flex flex-wrap gap-1 ml-1">
            <UBadge
              v-for="g in leadData.geo"
              :key="g"
              icon="i-lucide-map-pin"
              color="neutral"
              variant="outline"
              size="xs"
            >
              {{ g }}
            </UBadge>
            <UBadge
              v-for="p in leadData.products"
              :key="p"
              color="primary"
              variant="outline"
              size="xs"
            >
              {{ p }}
            </UBadge>
          </div>
        </div>

        <div class="flex items-center gap-1.5">
          <USelect
            v-model="selectedCompanyId"
            :items="companySelectItems"
            :loading="companiesLoading || assigningCompany"
            icon="i-lucide-building-2"
            placeholder="Привязать компанию..."
            size="xs"
            class="flex-1 min-w-0"
            @update:model-value="assignCompany"
          />

          <UModal
            v-model:open="showAddCompanyModal"
            title="Новая компания"
            description="Название станет тегом — будет видно в списке сигналов"
          >
            <UTooltip text="Создать компанию">
              <UButton
                icon="i-lucide-plus"
                color="neutral"
                variant="outline"
                size="xs"
                square
              />
            </UTooltip>

            <template #body>
              <UInput
                v-model="newCompanyName"
                placeholder="Например: MegaPay"
                autofocus
                @keydown.enter.prevent="createCompany"
              />
              <div class="flex justify-end gap-2 mt-3">
                <UButton
                  label="Отмена"
                  color="neutral"
                  variant="ghost"
                  size="sm"
                  @click="showAddCompanyModal = false"
                />
                <UButton
                  label="Создать"
                  color="primary"
                  size="sm"
                  :loading="addingCompany"
                  :disabled="!newCompanyName.trim()"
                  @click="createCompany"
                />
              </div>
            </template>
          </UModal>

          <UBadge
            v-if="assignedCompanyName"
            icon="i-lucide-building-2"
            :label="assignedCompanyName"
            color="info"
            variant="subtle"
            size="xs"
          />
        </div>
      </div>
    </div>

    <div class="flex-1 p-4 sm:p-6 overflow-y-auto">
      <p class="whitespace-pre-wrap text-sm leading-relaxed">
        {{ mail.body }}
      </p>

      <template v-if="otherSignals.length > 0">
        <UCard
          variant="subtle"
          class="mt-4"
          :ui="{ header: 'flex items-center gap-1.5 text-dimmed' }"
        >
          <template #header>
            <UIcon name="i-lucide-history" class="size-4" />
            <span class="text-sm">История сигналов ({{ otherSignals.length }})</span>
            <UButton
              :icon="historyCollapsed ? 'i-lucide-chevron-down' : 'i-lucide-chevron-up'"
              color="neutral"
              variant="ghost"
              size="xs"
              class="ml-auto"
              @click="historyCollapsed = !historyCollapsed"
            />
            <UBadge
              :label="String(otherSignals.length)"
              color="neutral"
              variant="soft"
              size="xs"
              class="ml-1"
            />
          </template>

          <USkeleton
            v-if="!historyCollapsed && historyStatus === 'pending'"
            class="h-24 w-full"
          />

          <div v-else-if="!historyCollapsed" class="max-h-64 overflow-y-auto pr-1">
            <UChatMessages
              :messages="historyTimeline"
              :should-scroll-to-bottom="false"
              :should-auto-scroll="false"
              :user="{ side: 'right', variant: 'soft' }"
              :assistant="{ side: 'left', variant: 'naked' }"
            >
              <template #content="{ message }">
                <div class="space-y-1">
                  <div class="flex items-center gap-1.5 text-[11px] text-muted">
                    <span>{{ message.metaDate }}</span>
                    <span class="truncate">{{ message.metaChat }}</span>
                    <UBadge
                      v-if="message.metaLead"
                      label="лид"
                      color="success"
                      variant="soft"
                      size="xs"
                    />
                  </div>
                  <template
                    v-for="(part, index) in message.parts"
                    :key="`${message.id}-${part.type}-${index}`"
                  >
                    <p
                      v-if="part.type === 'text'"
                      class="whitespace-pre-wrap text-sm"
                    >
                      {{ part.text }}
                    </p>
                  </template>
                </div>
              </template>
            </UChatMessages>
          </div>

          <p v-else class="text-xs text-muted">
            История свернута. Нажмите на стрелку, чтобы раскрыть переписку отправителя.
          </p>
        </UCard>
      </template>
    </div>

    <div class="pb-4 px-4 sm:px-6 shrink-0">
      <UCard variant="subtle" :ui="{ header: 'flex items-center gap-1.5 text-dimmed' }">
        <template #header>
          <UIcon name="i-lucide-brain-circuit" class="size-4" />
          <span class="text-sm">Обучить классификатор</span>
          <UBadge
            v-if="feedbackDone !== null"
            :color="feedbackDone ? 'success' : 'neutral'"
            variant="soft"
            size="xs"
            class="ml-auto"
          >
            {{ feedbackDone ? 'Сохранено как лид' : 'Сохранено как шум' }}
          </UBadge>
        </template>

        <p class="text-xs text-muted mb-3">
          Ручной выбор приоритетен: отмечайте сигнал как лид или шум, чтобы система точнее ранжировала похожие кейсы.
        </p>

        <div class="flex gap-2">
          <UButton
            icon="i-lucide-thumbs-up"
            label="Продвинуть в лиды"
            color="success"
            variant="soft"
            :loading="feedbackLoading === 'lead'"
            :disabled="feedbackLoading !== null || feedbackDone !== null"
            class="flex-1"
            @click="sendFeedback(true)"
          />
          <UButton
            icon="i-lucide-thumbs-down"
            label="Оставить в шуме"
            color="neutral"
            variant="soft"
            :loading="feedbackLoading === 'noise'"
            :disabled="feedbackLoading !== null || feedbackDone !== null"
            class="flex-1"
            @click="sendFeedback(false)"
          />
        </div>

        <div class="mt-3 grid grid-cols-2 sm:grid-cols-4 gap-2">
          <UButton
            label="Трейдер/Поиск трейдеров"
            color="info"
            variant="soft"
            :loading="categoryFeedbackLoading"
            :disabled="categoryFeedbackLoading"
            @click="sendCategoryFeedback('traders')"
          />
          <UButton
            label="Мерчант"
            color="info"
            variant="soft"
            :loading="categoryFeedbackLoading"
            :disabled="categoryFeedbackLoading"
            @click="sendCategoryFeedback('merchants')"
          />
          <UButton
            label="Предложение ПС"
            color="primary"
            variant="soft"
            :loading="categoryFeedbackLoading"
            :disabled="categoryFeedbackLoading"
            @click="sendCategoryFeedback('ps_offers')"
          />
          <UButton
            label="Шум"
            color="neutral"
            variant="soft"
            :loading="categoryFeedbackLoading"
            :disabled="categoryFeedbackLoading"
            @click="sendCategoryFeedback('noise')"
          />
        </div>
      </UCard>
    </div>
  </UDashboardPanel>
</template>
