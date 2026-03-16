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
const assigningTeam = ref(false)
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
  traders: 'Трейдер',
  merchants: 'Мерчант',
  ps_offers: 'Предложение ПС',
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
  { label: 'Трейдер', value: 'traders' },
  { label: 'Мерчант', value: 'merchants' },
  { label: 'Предложение ПС', value: 'ps_offers' },
  { label: 'Шум / мусор', value: 'noise' }
]

const statusLabel: Record<LeadStatus, string> = {
  new: 'Новый',
  contacted: 'Первичный контакт',
  qualified: 'В работе',
  converted: 'Подключён',
  rejected: 'Закрыт'
}

const statusColor: Record<LeadStatus, 'primary' | 'success' | 'warning' | 'error' | 'neutral'> = {
  new: 'primary',
  contacted: 'warning',
  qualified: 'success',
  converted: 'success',
  rejected: 'neutral'
}

// qualification: userFeedback=null → unreviewed, true → lead, false → not-lead
const qualificationState = computed(() => {
  if (lead.value?.userFeedback === true) return 'lead'
  if (lead.value?.userFeedback === false) return 'not-lead'
  return 'unreviewed'
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

async function approve() {
  await $fetch(`/api/leads/${leadID}/approve`, { method: 'POST' })
  toast.add({ title: 'Отмечен как лид', description: 'Данные переданы ИИ-классификатору', color: 'success' })
  await refresh()
}

async function reject() {
  await $fetch(`/api/leads/${leadID}/reject`, { method: 'POST' })
  toast.add({ title: 'Отмечен как не-лид', description: 'Данные переданы ИИ-классификатору', color: 'warning' })
  await refresh()
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

async function assignTeam() {
  const companyId = selectedCompanyId.value
  if (!companyId) return
  assigningTeam.value = true
  try {
    await $fetch(`/api/leads/${leadID}/merchant`, {
      method: 'PUT',
      body: { merchant_id: companyId }
    })
    const name = companies.value.find(c => c.id === companyId)?.name ?? companyId
    toast.add({ title: 'Компания назначена', description: name, color: 'success' })
    await refresh()
  } finally {
    assigningTeam.value = false
  }
}

async function addCompanyFromLeadCard() {
  const name = newCompanyName.value.trim()
  if (!name) return
  addingCompany.value = true
  try {
    await addCompany(name)
    await refreshCompanies()
    const created = [...companies.value].reverse().find(c => c.name === name)
    if (created) selectedCompanyId.value = created.id
    newCompanyName.value = ''
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
          <div v-if="lead" class="flex items-center gap-2">
            <UButton
              v-if="contactHref"
              :href="contactHref"
              target="_blank"
              icon="i-lucide-send"
              label="Написать"
              color="primary"
              variant="soft"
              size="sm"
            />
            <UButton
              icon="i-lucide-copy"
              label="Контакт"
              color="neutral"
              variant="soft"
              size="sm"
              @click="copyToClipboard(lead.contact || '')"
            />
            <UButton
              icon="i-lucide-trash-2"
              color="error"
              variant="ghost"
              size="sm"
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
            <div class="flex items-center justify-between gap-2">
              <h3 class="font-semibold">
                Это целевой лид?
              </h3>
              <UBadge
                v-if="qualificationState === 'lead'"
                color="success"
                variant="soft"
                icon="i-lucide-check-circle"
              >
                Подтверждён как лид
              </UBadge>
              <UBadge
                v-else-if="qualificationState === 'not-lead'"
                color="neutral"
                variant="soft"
                icon="i-lucide-x-circle"
              >
                Не является лидом
              </UBadge>
              <UBadge
                v-else
                color="warning"
                variant="soft"
                icon="i-lucide-circle-dashed"
              >
                Не оценено
              </UBadge>
            </div>
          </template>

          <div class="flex flex-wrap gap-2">
            <UButton
              :color="qualificationState === 'lead' ? 'success' : 'neutral'"
              :variant="qualificationState === 'lead' ? 'soft' : 'outline'"
              icon="i-lucide-thumbs-up"
              label="Да, это лид"
              @click="approve"
            />
            <UButton
              :color="qualificationState === 'not-lead' ? 'error' : 'neutral'"
              :variant="qualificationState === 'not-lead' ? 'soft' : 'outline'"
              icon="i-lucide-thumbs-down"
              label="Не лид / мусор"
              @click="reject"
            />
          </div>
          <p class="text-xs text-muted mt-3 flex items-center gap-1.5">
            <UIcon name="i-lucide-brain-circuit" class="size-3.5 shrink-0" />
            Ваш ответ передаётся ИИ-классификатору и улучшает точность автоматического определения
          </p>
        </UCard>

        <UCard>
          <template #header>
            <div class="flex items-center justify-between gap-2">
              <h3 class="font-semibold">
                Классификация ИИ
              </h3>
              <UBadge
                :color="categoryColor[String(lead.semanticCategory || 'leads')] || 'neutral'"
                variant="soft"
              >
                {{ categoryLabel[String(lead.semanticCategory || 'leads')] || lead.semanticCategory }}
              </UBadge>
            </div>
          </template>

          <div class="space-y-3">
            <!-- Confidence score bar -->
            <div class="flex items-center gap-3">
              <span class="text-xs text-muted w-24 shrink-0">Уверенность</span>
              <div class="flex-1 bg-muted/30 rounded-full h-1.5">
                <div
                  class="h-1.5 rounded-full bg-primary transition-all"
                  :style="`width: ${Math.min(Math.round((lead.score ?? 0) * 100), 100)}%`"
                />
              </div>
              <span class="text-xs font-mono w-10 text-right">
                {{ Math.min(Math.round((lead.score ?? 0) * 100), 100) }}%
              </span>
            </div>

            <div>
              <p class="text-xs text-muted mb-1.5">
                Категория определена автоматически. Если ошибочна — исправьте:
              </p>
              <div class="flex items-center gap-2">
                <USelect
                  v-model="selectedCategory"
                  :items="categorySelectItems"
                  icon="i-lucide-tag"
                  class="min-w-44"
                />
                <UButton
                  label="Исправить"
                  color="neutral"
                  variant="soft"
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
            <div class="flex items-center justify-between gap-2">
              <h3 class="font-semibold">
                Воронка CRM
              </h3>
              <UBadge
                :color="statusColor[lead.status]"
                variant="subtle"
              >
                {{ statusLabel[lead.status] }}
              </UBadge>
            </div>
          </template>

          <div class="flex flex-wrap gap-2">
            <UButton
              v-for="s in (['new', 'contacted', 'qualified', 'converted', 'rejected'] as LeadStatus[])"
              :key="s"
              :color="lead.status === s ? statusColor[s] : 'neutral'"
              :variant="lead.status === s ? 'soft' : 'ghost'"
              size="sm"
              @click="setStatus(s)"
            >
              {{ statusLabel[s] }}
            </UButton>
          </div>
        </UCard>

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

          <div class="grid grid-cols-1 sm:grid-cols-2 gap-3 text-sm">
            <div>
              <p class="text-muted text-xs">
                Контакт
              </p>
              <div class="flex items-center gap-1.5 mt-0.5">
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
              <p class="text-muted text-xs">
                Чат
              </p>
              <p class="mt-0.5">
                {{ lead.chatTitle || '—' }}
              </p>
            </div>
            <div>
              <p class="text-muted text-xs">
                Сигналов
              </p>
              <p class="text-lg font-semibold mt-0.5">
                {{ briefData?.signalsCount ?? 0 }}
              </p>
            </div>
            <div v-if="lead.geo.length">
              <p class="text-muted text-xs">
                Гео
              </p>
              <div class="flex flex-wrap gap-1 mt-0.5">
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
              <p class="text-muted text-xs">
                Продукты
              </p>
              <div class="flex flex-wrap gap-1 mt-0.5">
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
        </UCard>

        <UCard>
          <template #header>
            <div class="flex items-center justify-between">
              <h3 class="font-semibold">
                Компания
              </h3>
              <UBadge
                v-if="companyName !== '—'"
                color="neutral"
                variant="subtle"
                size="sm"
              >
                {{ companyName }}
              </UBadge>
            </div>
          </template>
          <div class="flex flex-wrap items-center gap-2">
            <USelect
              v-model="selectedCompanyId"
              :items="companySelectItems"
              :loading="companiesLoading"
              icon="i-lucide-building-2"
              placeholder="Выбрать компанию..."
              class="min-w-56"
            />
            <UButton
              icon="i-lucide-save"
              label="Назначить"
              color="primary"
              :loading="assigningTeam"
              :disabled="!selectedCompanyId"
              @click="assignTeam"
            />
          </div>
          <div class="mt-3 flex flex-wrap items-center gap-2">
            <UInput
              v-model="newCompanyName"
              icon="i-lucide-plus"
              placeholder="Создать новую компанию..."
              class="min-w-56"
              @keydown.enter.prevent="addCompanyFromLeadCard"
            />
            <UButton
              icon="i-lucide-plus"
              label="Создать"
              color="neutral"
              variant="soft"
              :loading="addingCompany"
              :disabled="!newCompanyName.trim()"
              @click="addCompanyFromLeadCard"
            />
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
                  <UBadge
                    v-if="signal.isLead"
                    color="success"
                    variant="soft"
                    size="xs"
                    icon="i-lucide-target"
                  >
                    лид
                  </UBadge>
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
