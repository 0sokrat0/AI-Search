<script setup lang="ts">
import { format, formatDistanceToNow } from 'date-fns'
import type { LeadBrief, LeadStatus } from '~/types'

definePageMeta({
  middleware: 'auth'
})

const route = useRoute()
const router = useRouter()
const leadID = route.params.id as string
const toast = useToast()

const { data: brief, status, refresh } = await useFetch<LeadBrief>(`/api/leads/${leadID}/brief`)

const lead = computed(() => (brief.value as LeadBrief | null)?.lead ?? null)
const signals = computed(() => (brief.value as LeadBrief | null)?.signals ?? [])
const briefData = computed(() => brief.value as LeadBrief | null)

const selectedCompanyId = ref<string>('')
const assigningCompany = ref(false)
const showAddCompanyModal = ref(false)
const newCompanyName = ref('')
const addingCompany = ref(false)
const updatingCategory = ref(false)
const selectedCategory = ref<string>('')

const { companies, loading: companiesLoading, addCompany, refresh: refreshCompanies } = useCompanies()

const companySelectItems = computed(() =>
  companies.value.map(c => ({ label: c.name, value: c.id }))
)

const companyName = computed(() => {
  const mid = selectedCompanyId.value || lead.value?.merchantId
  if (mid) return companies.value.find(c => c.id === mid)?.name ?? '—'
  return '—'
})

const categoryLabel: Record<string, string> = {
  leads: 'Лид',
  traders: 'Трейдеры / Поиск трейдеров',
  merchants: 'Мерчанты',
  ps_offers: 'Предложения от ПС',
  noise: 'Шум'
}

const categoryColor: Record<string, 'primary' | 'success' | 'warning' | 'info' | 'neutral'> = {
  leads: 'primary',
  traders: 'success',
  merchants: 'info',
  ps_offers: 'info',
  noise: 'neutral'
}

const categorySelectItems = [
  { label: 'Лид', value: 'leads' },
  { label: 'Трейдеры / Поиск трейдеров', value: 'traders' },
  { label: 'Мерчанты', value: 'merchants' },
  { label: 'Предложения от ПС', value: 'ps_offers' },
  { label: 'Шум / мусор', value: 'noise' }
]

const statusLabel: Record<LeadStatus, string> = {
  new: 'Новый',
  detected: 'Обнаружен',
  confirmed: 'Подтверждён',
  controversial: 'Спорный',
  false_positive: 'Ложный',
  contacted: 'Первичный контакт',
  qualified: 'В работе',
  converted: 'Подключён',
  rejected: 'Закрыт'
}

const statusColor: Record<LeadStatus, 'primary' | 'success' | 'warning' | 'error' | 'neutral' | 'info'> = {
  new: 'primary',
  detected: 'info',
  confirmed: 'success',
  controversial: 'warning',
  false_positive: 'error',
  contacted: 'warning',
  qualified: 'success',
  converted: 'success',
  rejected: 'neutral'
}

const qualificationSourceLabel: Record<string, string> = {
  ai_qualified: 'Квалифицировано ИИ',
  manual_approved: 'Ручная квалификация'
}

function shortCategoryLabel(category?: string | null): string {
  switch (String(category || '').toLowerCase()) {
    case 'traders':
      return 'Трейдеры'
    case 'merchants':
      return 'Мерчанты'
    case 'ps_offers':
      return 'ПС'
    case 'noise':
      return 'Шум'
    default:
      return 'Категория'
  }
}

const leadScorePercent = computed(() => Math.min(Math.round((lead.value?.score ?? 0) * 100), 100))
const leadScoreTitle = computed(() => `${shortCategoryLabel(lead.value?.semanticCategory)} ${leadScorePercent.value}%`)
const leadMatchBarClass = computed(() => {
  switch (String(lead.value?.semanticCategory || '').toLowerCase()) {
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

function buildContactHref(raw?: string | null): string {
  const value = String(raw || '').trim()
  if (!value) return ''
  if (value.startsWith('@')) return `https://t.me/${value.slice(1)}`
  if (value.includes('@')) return `mailto:${value}`
  return ''
}

const contactHref = computed(() => buildContactHref(lead.value?.contact))

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

function formatDistanceSafe(value?: string | null): string {
  const date = parseDateSafe(value)
  if (!date) return ''
  return formatDistanceToNow(date, { addSuffix: true })
}

function copyToClipboard(text: string) {
  window.navigator.clipboard.writeText(text)
  toast.add({ title: 'Скопировано', color: 'success' })
}

async function setStatus(newStatus: LeadStatus) {
  await $fetch(`/api/leads/${leadID}/status`, {
    method: 'PATCH',
    body: { status: newStatus }
  })
  toast.add({ title: `Статус: ${statusLabel[newStatus]}`, color: 'success' })
  await refresh()
}

async function updateCategory() {
  if (!selectedCategory.value || selectedCategory.value === lead.value?.semanticCategory) return
  updatingCategory.value = true
  try {
    await $fetch(`/api/leads/${leadID}/category`, {
      method: 'PATCH',
      body: { category: selectedCategory.value }
    })
    toast.add({ title: 'Категория обновлена', color: 'success' })
    await refresh()
  } catch (e: any) {
    toast.add({ title: 'Ошибка', description: e?.message, color: 'error' })
  } finally {
    updatingCategory.value = false
  }
}

async function assignCompany(companyId: string) {
  if (!companyId) return
  assigningCompany.value = true
  try {
    await $fetch(`/api/leads/${leadID}/merchant`, {
      method: 'PUT',
      body: { merchant_id: companyId }
    })
    const name = companies.value.find(c => c.id === companyId)?.name ?? companyId
    toast.add({ title: 'Компания назначена', description: name, color: 'success' })
    await refresh()
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
    toast.add({ title: 'Компания создана', color: 'success' })
  } catch (e: any) {
    toast.add({ title: 'Ошибка', description: e?.message, color: 'error' })
  } finally {
    addingCompany.value = false
  }
}

async function deleteLead() {
  const ok = window.confirm(`Удалить "${lead.value?.name || leadID}" из CRM?`)
  if (!ok) return
  await $fetch(`/api/leads/${leadID}`, { method: 'DELETE' })
  toast.add({ title: 'Удалено', color: 'success' })
  router.push('/leads')
}

function goBackToLeads() {
  router.push('/leads')
}

watch(lead, (value) => {
  if (!value) return
  selectedCompanyId.value = value.merchantId || value.companyId || ''
  selectedCategory.value = value.semanticCategory || 'leads'
}, { immediate: true })
</script>

<template>
  <UDashboardPanel id="lead-profile">
    <template #header>
      <UDashboardNavbar :title="lead ? lead.name : 'Карточка лида'">
        <template #leading>
          <UButton
            icon="i-lucide-arrow-left"
            color="neutral"
            variant="ghost"
            @click="goBackToLeads"
          />
        </template>
        <template #right>
          <div v-if="lead" class="flex items-center gap-1.5">
            <!-- Contact chip: shows @username prominently -->
            <div
              v-if="lead.contact && lead.contact !== '—'"
              class="hidden sm:flex items-center gap-1.5 px-2.5 py-1 rounded-lg border border-default bg-elevated text-xs font-mono text-muted select-all cursor-text"
              :title="lead.contact"
            >
              <UIcon name="i-lucide-at-sign" class="size-3 shrink-0" />
              <span class="max-w-32 truncate">{{ lead.contact }}</span>
            </div>

            <!-- Primary CTA: open Telegram direct -->
            <UButton
              v-if="contactHref"
              :href="contactHref"
              target="_blank"
              icon="i-lucide-send"
              label="Написать"
              color="primary"
              variant="solid"
              size="sm"
            />
            <UTooltip v-else text="Нет username — скопируйте имя для поиска в Telegram">
              <UButton
                icon="i-lucide-user-search"
                label="Нет ника"
                color="neutral"
                variant="soft"
                size="sm"
                @click="copyToClipboard(lead.name || '')"
              />
            </UTooltip>

            <!-- Copy username -->
            <UTooltip text="Скопировать @ник">
              <UButton
                icon="i-lucide-copy"
                color="neutral"
                variant="ghost"
                size="sm"
                square
                @click="copyToClipboard(lead.contact || lead.name || '')"
              />
            </UTooltip>

            <UButton
              icon="i-lucide-trash-2"
              color="error"
              variant="ghost"
              size="sm"
              square
              @click="deleteLead"
            />
          </div>
        </template>
      </UDashboardNavbar>
    </template>

    <template #body>
      <div v-if="status === 'pending'" class="space-y-4">
        <USkeleton class="h-28 w-full" />
        <USkeleton class="h-40 w-full" />
        <USkeleton class="h-40 w-full" />
      </div>

      <div v-else-if="!lead" class="flex h-full items-center justify-center">
        <UAlert
          color="warning"
          variant="soft"
          title="Лид не найден"
          description="Этот лид не существует или был удалён."
        />
      </div>

      <div v-else class="space-y-4">
        <UCard>
          <template #header>
            <div class="flex items-center justify-between">
              <h3 class="font-semibold">
                Данные
              </h3>
              <p class="text-xs text-muted">
                {{ briefData?.lastSeenAt ? `Последний: ${formatDistanceSafe(briefData.lastSeenAt)}` : '' }}
              </p>
            </div>
          </template>

          <div class="space-y-4">
            <div class="flex flex-wrap items-center gap-2">
              <UBadge
                :color="categoryColor[String(lead.semanticCategory || 'leads')] || 'neutral'"
                variant="soft"
              >
                {{ categoryLabel[String(lead.semanticCategory || 'leads')] || lead.semanticCategory }}
              </UBadge>
              <UBadge
                v-if="lead.qualificationSource"
                color="info"
                variant="soft"
              >
                {{ qualificationSourceLabel[lead.qualificationSource] || lead.qualificationSource }}
              </UBadge>
              <UBadge
                :color="statusColor[lead.status]"
                variant="subtle"
              >
                {{ statusLabel[lead.status] }}
              </UBadge>
            </div>

            <div class="rounded-xl border border-default bg-elevated/30 p-4">
              <div class="flex items-center justify-between gap-3 text-sm">
                <span class="font-medium text-highlighted">{{ leadScoreTitle }}</span>
                <span class="font-mono text-sm">{{ leadScorePercent }}%</span>
              </div>
              <div class="mt-2 h-2 overflow-hidden rounded-full bg-muted">
                <div
                  class="h-full rounded-full transition-all"
                  :class="leadMatchBarClass"
                  :style="{ width: `${leadScorePercent}%` }"
                />
              </div>
            </div>

            <div class="grid grid-cols-1 gap-3 text-sm sm:grid-cols-2">
              <div>
                <p class="text-xs text-muted">
                  Контакт
                </p>
                <div class="mt-0.5 flex items-center gap-1.5">
                  <span>{{ lead.contact || '—' }}</span>
                  <UButton
                    v-if="contactHref"
                    :href="contactHref"
                    target="_blank"
                    icon="i-lucide-send"
                    color="neutral"
                    variant="ghost"
                    size="xs"
                  />
                </div>
              </div>
              <div>
                <p class="text-xs text-muted">
                  Чат
                </p>
                <p class="mt-0.5">
                  {{ lead.chatTitle || '—' }}
                </p>
              </div>
              <div>
                <p class="text-xs text-muted">
                  Сигналов
                </p>
                <p class="mt-0.5 text-lg font-semibold">
                  {{ briefData?.signalsCount ?? 0 }}
                </p>
              </div>
              <div>
                <p class="text-xs text-muted">
                  Дата сигнала
                </p>
                <p class="mt-0.5">
                  {{ formatDateSafe(lead.categoryAssignedAt || lead.lastSeenAt) }}
                </p>
              </div>
              <div v-if="lead.geo.length">
                <p class="text-xs text-muted">
                  Гео
                </p>
                <div class="mt-0.5 flex flex-wrap gap-1">
                  <UBadge
                    v-for="g in lead.geo"
                    :key="g"
                    color="neutral"
                    variant="soft"
                    size="sm"
                  >
                    {{ g }}
                  </UBadge>
                </div>
              </div>
              <div v-if="lead.products.length">
                <p class="text-xs text-muted">
                  Продукты
                </p>
                <div class="mt-0.5 flex flex-wrap gap-1">
                  <UBadge
                    v-for="p in lead.products"
                    :key="p"
                    color="primary"
                    variant="soft"
                    size="sm"
                  >
                    {{ p }}
                  </UBadge>
                </div>
              </div>
            </div>

            <p class="flex items-center gap-1.5 border-t border-default pt-3 text-xs text-muted">
              Один лид здесь = один конкретный сигнал. Контакт используется только как источник и связь между сообщениями.
            </p>
          </div>
        </UCard>

        <UCard>
          <template #header>
            <div class="flex items-center justify-between">
              <h3 class="font-semibold">
                Компания
              </h3>
              <UBadge
                v-if="companyName !== '—'"
                icon="i-lucide-building-2"
                color="info"
                variant="subtle"
                size="sm"
              >
                {{ companyName }}
              </UBadge>
            </div>
          </template>

          <div class="space-y-2">
            <p class="text-sm text-highlighted">
              {{ companyName !== '—' ? companyName : 'Компания пока не привязана' }}
            </p>
            <p class="text-xs text-muted">
              Здесь показывается текущая компания, если сигнал уже к ней привязан.
            </p>
          </div>
        </UCard>

        <UCard>
          <template #header>
            <div class="flex items-center justify-between">
              <h3 class="font-semibold">
                Привязка к компании
              </h3>
              <p class="text-xs text-muted">
                Выбор или создание новой компании
              </p>
            </div>
          </template>

          <div class="flex items-center gap-2">
            <USelect
              v-model="selectedCompanyId"
              :items="companySelectItems"
              :loading="companiesLoading || assigningCompany"
              icon="i-lucide-building-2"
              placeholder="Выбрать компанию..."
              class="flex-1"
              @update:model-value="assignCompany"
            />

            <UModal
              v-model:open="showAddCompanyModal"
              title="Новая компания"
              description="Название станет тегом — будет видно в списке сигналов"
            >
              <UButton
                icon="i-lucide-plus"
                color="neutral"
                variant="outline"
                square
                title="Создать новую"
              />

              <template #body>
                <div class="space-y-4">
                  <UInput
                    v-model="newCompanyName"
                    placeholder="Например: MegaPay"
                    autofocus
                    @keydown.enter.prevent="createCompany"
                  />
                  <div class="flex justify-end gap-2">
                    <UButton
                      label="Отмена"
                      color="neutral"
                      variant="ghost"
                      @click="showAddCompanyModal = false"
                    />
                    <UButton
                      label="Создать и назначить"
                      color="primary"
                      :loading="addingCompany"
                      :disabled="!newCompanyName.trim()"
                      @click="createCompany"
                    />
                  </div>
                </div>
              </template>
            </UModal>
          </div>
        </UCard>

        <UCard>
          <template #header>
            <div class="flex items-center justify-between gap-2">
              <h3 class="font-semibold">
                Воронка и тип сигнала
              </h3>
              <UBadge
                :color="categoryColor[String(lead.semanticCategory || 'leads')] || 'neutral'"
                variant="soft"
              >
                {{ categoryLabel[String(lead.semanticCategory || 'leads')] || lead.semanticCategory }}
              </UBadge>
            </div>
          </template>

          <div class="space-y-4">
            <div class="flex flex-wrap gap-2">
              <UButton
                v-for="s in (['new', 'contacted', 'qualified', 'converted', 'rejected'] as LeadStatus[])"
                :key="s"
                :color="lead.status === s ? statusColor[s] : 'neutral'"
                :variant="lead.status === s ? 'soft' : 'ghost'"
                size="xs"
                @click="setStatus(s)"
              >
                {{ statusLabel[s] }}
              </UButton>
            </div>

            <div class="border-t border-default pt-4">
              <div class="flex flex-wrap items-center gap-2">
                <USelect
                  v-model="selectedCategory"
                  :items="categorySelectItems"
                  icon="i-lucide-tag"
                  class="min-w-44"
                />
                <UButton
                  label="Сменить тип"
                  color="neutral"
                  variant="ghost"
                  size="sm"
                  :loading="updatingCategory"
                  :disabled="selectedCategory === lead.semanticCategory"
                  @click="updateCategory"
                />
              </div>
            </div>
          </div>
        </UCard>

        <UCard>
          <template #header>
            <h3 class="font-semibold">
              История сигналов
            </h3>
          </template>

          <div v-if="!signals.length" class="text-sm text-muted py-2">
            Нет сигналов.
          </div>

          <div v-else class="space-y-3">
            <div
              v-for="signal in signals"
              :key="signal.id"
              class="border border-default rounded-lg p-3"
            >
              <div class="flex items-start justify-between gap-2">
                <div class="flex flex-wrap items-center gap-1.5 flex-1 min-w-0">
                  <template v-if="signal.semanticCategory && signal.semanticCategory !== 'stream'">
                    <UBadge
                      :color="categoryColor[String(signal.semanticCategory)] || 'neutral'"
                      variant="soft"
                      size="xs"
                    >
                      {{ categoryLabel[String(signal.semanticCategory)] || signal.semanticCategory }}
                    </UBadge>
                    <span
                      v-if="signal.categoryAssignedAt"
                      class="text-xs text-muted"
                    >
                      {{ formatDateSafe(signal.categoryAssignedAt) }}
                    </span>
                  </template>
                  <span class="text-xs text-muted truncate">{{ signal.chatTitle }}</span>
                </div>
                <p class="text-xs text-muted shrink-0">
                  {{ formatDateSafe(signal.date) }}
                </p>
              </div>
              <p class="text-sm mt-2 text-highlighted">
                {{ signal.text }}
              </p>
            </div>
          </div>
        </UCard>

      </div>
    </template>
  </UDashboardPanel>
</template>
