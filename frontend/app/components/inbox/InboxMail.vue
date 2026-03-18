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

const feedbackDone = ref<boolean | null>(null)
const categoryFeedbackLoading = ref(false)
const historyCollapsed = ref(true)

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

const leadStatusColor: Record<LeadStatus, 'primary' | 'success' | 'warning' | 'error' | 'neutral' | 'info'> = {
  new: 'primary',
  detected: 'info',
  confirmed: 'success',
  controversial: 'warning',
  false_positive: 'error',
  contacted: 'warning',
  qualified: 'success',
  converted: 'success',
  rejected: 'error'
}
const leadStatusLabel: Record<LeadStatus, string> = {
  new: 'Новый',
  detected: 'Обнаружен',
  confirmed: 'Подтвержден',
  controversial: 'Спорный',
  false_positive: 'Ложный',
  contacted: 'Первичный контакт',
  qualified: 'В работе',
  converted: 'Подключен',
  rejected: 'Закрыт'
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
    case 'traders': return 'Трейдеры / Поиск трейдеров'
    case 'merchants': return 'Мерчанты'
    case 'ps_offers': return 'Предложения от ПС'
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

const bestBusinessMatch = computed(() => {
  const candidates = [
    { label: 'Трейдеры / Поиск трейдеров', score: Number(props.mail.traderScore ?? 0) },
    { label: 'Мерчанты', score: Number(props.mail.merchantScore ?? 0) },
    { label: 'Предложения от ПС', score: Number(props.mail.psOfferScore ?? 0) }
  ]

  const best = candidates.sort((a, b) => b.score - a.score)[0]
  if (!best || best.score <= 0) return null

  return {
    label: best.label,
    percent: Math.round(best.score * 100)
  }
})

const leadPipelineLabel = computed(() => {
  if (props.mail.category === 'noise') return ''
  if (props.mail.leadId) return 'Прошёл в квалифицированные лиды'
  return 'Категоризирован как сигнал, но не прошёл в лиды'
})

const leadPipelineColor = computed<'success' | 'warning' | 'neutral'>(() => {
  if (props.mail.category === 'noise') return 'neutral'
  if (props.mail.leadId) return 'success'
  return 'warning'
})

const matchBarClass = computed(() => {
  switch (props.mail.category) {
    case 'traders':
      return 'bg-emerald-500'
    case 'merchants':
      return 'bg-sky-500'
    case 'ps_offers':
      return 'bg-indigo-500'
    default:
      return 'bg-zinc-400'
  }
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
    if (field === 'is_team_member') {
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
          v-if="leadPipelineLabel"
          :label="leadPipelineLabel"
          :color="leadPipelineColor"
          variant="soft"
          size="xs"
        />
        <UBadge
          v-if="mail.semanticFlags?.includes('has_traffic')"
          icon="i-lucide-check-circle-2"
          label="Предлагают трафик"
          color="success"
          variant="subtle"
          size="xs"
        />
        <UTooltip v-if="mail.showMultiAccountBadges !== false && mail.otherChatsCount > 1" text="Встречался в нескольких чатах">
          <UBadge
            icon="i-lucide-messages-square"
            :label="`${mail.otherChatsCount} чатов`"
            color="warning"
            variant="subtle"
            size="xs"
          />
        </UTooltip>
      </div>
    </div>

    <div v-if="mail.leadId" class="px-4 sm:px-6 py-3 border-b border-default">
      <USkeleton v-if="!leadData" class="h-14 w-full rounded-lg" />

      <div v-else class="space-y-2.5">
        <div class="flex items-center gap-1.5 flex-wrap">
          <UBadge :color="leadStatusColor[leadData.status]" variant="subtle" size="xs">
            {{ leadStatusLabel[leadData.status] }}
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
      <div v-if="bestBusinessMatch" class="mb-4 rounded-xl border border-default bg-elevated/30 p-4">
        <div class="flex items-center justify-between gap-3 text-sm">
          <span class="font-medium text-highlighted">Похожесть на {{ bestBusinessMatch.label }}</span>
          <span class="font-mono text-sm">{{ bestBusinessMatch.percent }}%</span>
        </div>
        <div class="mt-2 h-2 overflow-hidden rounded-full bg-muted">
          <div
            class="h-full rounded-full transition-all"
            :class="matchBarClass"
            :style="{ width: `${bestBusinessMatch.percent}%` }"
          />
        </div>
        <p class="mt-2 text-xs text-muted">
          {{ leadPipelineLabel || 'Остался в шуме' }}
        </p>
      </div>

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
          Ручная категоризация приоритетна: выберите бизнес-категорию или оставьте сигнал в шуме.
        </p>

        <div class="grid grid-cols-1 sm:grid-cols-3 gap-2">
          <UButton
            label="Трейдеры / Поиск трейдеров"
            color="info"
            variant="soft"
            :loading="categoryFeedbackLoading"
            :disabled="categoryFeedbackLoading"
            @click="sendCategoryFeedback('traders')"
          />
          <UButton
            label="Мерчанты"
            color="info"
            variant="soft"
            :loading="categoryFeedbackLoading"
            :disabled="categoryFeedbackLoading"
            @click="sendCategoryFeedback('merchants')"
          />
          <UButton
            label="Предложения от ПС"
            color="primary"
            variant="soft"
            :loading="categoryFeedbackLoading"
            :disabled="categoryFeedbackLoading"
            @click="sendCategoryFeedback('ps_offers')"
          />
        </div>

        <div class="mt-3">
          <UButton
            label="Шум"
            color="neutral"
            variant="soft"
            :loading="categoryFeedbackLoading"
            :disabled="categoryFeedbackLoading"
            block
            @click="sendCategoryFeedback('noise')"
          />
        </div>
      </UCard>
    </div>
  </UDashboardPanel>
</template>
