<script setup lang="ts">
definePageMeta({
  middleware: 'auth'
})

import { format, formatDistanceToNow } from 'date-fns'
import type { LeadBrief, LeadStatus } from '~/types'

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

const { companies, loading: companiesLoading, addCompany, refresh: refreshCompanies } = useCompanies()

const companySelectItems = computed(() =>
  companies.value.map((c) => ({ label: c.name, value: c.id }))
)

const companyName = computed(() => {
  if (selectedCompanyId.value) {
    return companies.value.find((c) => c.id === selectedCompanyId.value)?.name
      ?? lead.value?.company
      ?? lead.value?.merchantId
      ?? 'Не назначена'
  }
  return lead.value?.company || lead.value?.merchantId || 'Не назначена'
})

const statusColor: Record<LeadStatus, 'primary' | 'success' | 'warning' | 'error' | 'neutral'> = {
  new: 'primary',
  contacted: 'warning',
  qualified: 'success',
  converted: 'success',
  rejected: 'error'
}

const statusLabel: Record<LeadStatus, string> = {
  new: 'Новый',
  contacted: 'Первичный контакт',
  qualified: 'В работе',
  converted: 'Сделка / подключен',
  rejected: 'Мусор / закрыт'
}

const categoryLabel: Record<string, string> = {
  leads: 'Лиды',
  traders: 'Трейдеры',
  merchants: 'Мерчанты',
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

function nextActionLabel(status: LeadStatus): string {
  switch (status) {
    case 'new':
      return 'Связаться'
    case 'contacted':
      return 'Уточнить запрос'
    case 'qualified':
      return 'Подготовить оффер'
    case 'converted':
      return 'Сопровождать'
    case 'rejected':
      return 'Архив'
    default:
      return 'Проверить'
  }
}

async function approve() {
  await $fetch(`/api/leads/${leadID}/approve`, { method: 'POST' })
  toast.add({ title: 'Лид одобрен', color: 'success' })
  await refresh()
}

async function reject() {
  await $fetch(`/api/leads/${leadID}/reject`, { method: 'POST' })
  toast.add({ title: 'Лид отклонён', color: 'warning' })
  await refresh()
}

async function deleteLead() {
  const ok = window.confirm(`Удалить лид "${lead.value?.name || leadID}" из CRM?`)
  if (!ok) return
  await $fetch(`/api/leads/${leadID}`, { method: 'DELETE' })
  toast.add({ title: 'Лид удален', color: 'success' })
  router.push('/leads')
}

function copyToClipboard(text: string) {
  window.navigator.clipboard.writeText(text)
}

async function setStatus(newStatus: LeadStatus) {
  await $fetch(`/api/leads/${leadID}/status`, {
    method: 'PATCH',
    body: { status: newStatus }
  })
  toast.add({ title: `Статус обновлён: ${statusLabel[newStatus]}`, color: 'success' })
  await refresh()
}

function goBackToLeads() {
  router.push('/leads')
}

function buildContactHref(raw?: string | null): string {
  const value = String(raw || '').trim()
  if (!value) return ''
  if (value.startsWith('@')) return `https://t.me/${value.slice(1)}`
  if (value.includes('@')) return `mailto:${value}`
  return ''
}

const contactHref = computed(() => {
  return buildContactHref(lead.value?.contact)
})

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

async function assignTeam() {
  const companyId = selectedCompanyId.value
  if (!companyId) return
  assigningTeam.value = true
  try {
    await $fetch(`/api/leads/${leadID}/merchant`, {
      method: 'PUT',
      body: { merchant_id: companyId }
    })
    const name = companies.value.find((c) => c.id === companyId)?.name ?? companyId
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
    const created = [...companies.value].reverse().find((c) => c.name === name)
    if (created) selectedCompanyId.value = created.id
    newCompanyName.value = ''
    toast.add({ title: 'Компания создана', description: 'Добавлена в общий список компаний', color: 'success' })
  } catch (e: any) {
    toast.add({ title: 'Ошибка', description: e?.message || 'Не удалось создать компанию', color: 'error' })
  } finally {
    addingCompany.value = false
  }
}

watch(lead, (value) => {
  if (!value) return
  const merchantId = value.merchantId || value.companyId || ''
  selectedCompanyId.value = merchantId
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
              icon="i-lucide-trash-2"
              label="Удалить из CRM"
              color="error"
              variant="soft"
              @click="deleteLead"
            />
            <UButton
              v-if="lead.status === 'new'"
              icon="i-lucide-check"
              label="Одобрить"
              color="success"
              variant="soft"
              @click="approve"
            />
            <UButton
              v-if="lead.status === 'new'"
              icon="i-lucide-x"
              label="Отклонить"
              color="error"
              variant="soft"
              @click="reject"
            />
          </div>
        </template>
      </UDashboardNavbar>
    </template>

    <template #body>
      <div v-if="status === 'pending'" class="space-y-4">
        <USkeleton class="h-24 w-full" />
        <USkeleton class="h-48 w-full" />
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
        <UCard class="sticky top-0 z-10 border border-primary/20 bg-primary/5 backdrop-blur">
          <template #header>
            <h3 class="font-semibold">Быстрые действия</h3>
          </template>
          <div class="flex flex-wrap items-center gap-2">
            <UButton
              v-if="contactHref"
              :href="contactHref"
              target="_blank"
              icon="i-lucide-message-circle"
              label="Связаться"
              color="primary"
              variant="soft"
            />
            <UButton
              icon="i-lucide-copy"
              label="Скопировать контакт"
              color="neutral"
              variant="soft"
              @click="copyToClipboard(lead.contact || '')"
            />
            <UButton
              icon="i-lucide-building-2"
              label="Назначить компанию"
              color="info"
              variant="soft"
              @click="assignTeam"
            />
            <UButton
              icon="i-lucide-circle-x"
              label="Закрыть лид"
              color="error"
              variant="soft"
              @click="setStatus('rejected')"
            />
          </div>
        </UCard>

        <div class="grid grid-cols-2 lg:grid-cols-4 gap-4">
          <UCard>
            <template #header><p class="text-xs text-muted">Статус</p></template>
            <UBadge :color="statusColor[lead.status]" variant="subtle" class="capitalize">
                {{ statusLabel[lead.status] }}
            </UBadge>
          </UCard>

          <UCard>
            <template #header><p class="text-xs text-muted">Следующий шаг</p></template>
            <UBadge color="neutral" variant="subtle">
              {{ nextActionLabel(lead.status) }}
            </UBadge>
          </UCard>

          <UCard>
            <template #header><p class="text-xs text-muted">Чат</p></template>
            <p class="text-sm font-medium truncate">{{ lead.chatTitle || '—' }}</p>
          </UCard>

          <UCard>
            <template #header><p class="text-xs text-muted">Сигналов</p></template>
            <p class="text-2xl font-semibold">{{ briefData?.signalsCount ?? 0 }}</p>
          </UCard>
        </div>

        <UCard>
          <template #header>
            <div class="flex items-center justify-between">
              <h3 class="font-semibold">Данные лида</h3>
              <p class="text-xs text-muted">
                {{ briefData?.lastSeenAt ? `Последний раз: ${formatDistanceSafe(briefData.lastSeenAt)}` : '' }}
              </p>
            </div>
          </template>

          <div class="grid grid-cols-1 sm:grid-cols-2 gap-3 text-sm">
            <div>
              <p class="text-muted text-xs">Контакт</p>
              <p>{{ lead.contact || '—' }}</p>
            </div>
            <div>
              <p class="text-muted text-xs">Компания</p>
              <p>{{ companyName }}</p>
            </div>
            <div>
              <p class="text-muted text-xs">Категория</p>
              <UBadge
                :color="categoryColor[String(lead.semanticCategory || 'leads')] || 'neutral'"
                variant="soft"
                size="sm"
              >
                {{ categoryLabel[String(lead.semanticCategory || 'leads')] || lead.semanticCategory || 'Лиды' }}
              </UBadge>
            </div>
            <div v-if="lead.geo.length">
              <p class="text-muted text-xs">Гео</p>
              <div class="flex flex-wrap gap-1 mt-1">
                <UBadge v-for="g in lead.geo" :key="g" color="neutral" variant="soft" size="sm">{{ g }}</UBadge>
              </div>
            </div>
            <div v-if="lead.products.length">
              <p class="text-muted text-xs">Продукты</p>
              <div class="flex flex-wrap gap-1 mt-1">
                <UBadge v-for="p in lead.products" :key="p" color="primary" variant="soft" size="sm">{{ p }}</UBadge>
              </div>
            </div>
          </div>
        </UCard>

        <UCard>
          <template #header>
            <div class="flex items-center justify-between">
              <h3 class="font-semibold">Компания лида</h3>
              <UBadge
                v-if="lead.company || lead.merchantId"
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
              placeholder="Создать компанию..."
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
          <template #header><h3 class="font-semibold">Воронка</h3></template>
          <div class="flex flex-wrap gap-2">
            <UButton
              v-for="s in (['new', 'contacted', 'qualified', 'converted', 'rejected'] as LeadStatus[])"
              :key="s"
              :color="lead.status === s ? statusColor[s] : 'neutral'"
              :variant="lead.status === s ? 'soft' : 'ghost'"
              size="sm"
              @click="setStatus(s)"
            >{{ statusLabel[s] }}</UButton>
          </div>
        </UCard>

        <UCard>
          <template #header><h3 class="font-semibold">История сообщений в чате</h3></template>

          <div v-if="!signals.length" class="text-sm text-muted">
            Нет сообщений.
          </div>

          <div v-else class="space-y-3">
            <div
              v-for="signal in signals"
              :key="signal.id"
              class="border border-default rounded-lg p-3"
            >
              <div class="flex items-center justify-between gap-2">
                <p class="font-medium text-sm">{{ signal.fromName || 'Неизвестный отправитель' }}</p>
                <p class="text-xs text-muted">{{ formatDateSafe(signal.date) }}</p>
              </div>
              <div class="mt-1 flex flex-wrap items-center gap-1.5">
                <p class="text-xs text-muted">{{ signal.chatTitle }}</p>
                <UButton
                  v-if="buildContactHref(signal.contact)"
                  :href="buildContactHref(signal.contact)"
                  target="_blank"
                  icon="i-lucide-send"
                  color="neutral"
                  variant="ghost"
                  size="xs"
                />
                <UBadge
                  v-if="signal.isLead"
                  label="лид"
                  color="success"
                  variant="soft"
                  size="xs"
                />
                <UBadge
                  v-if="signal.semanticCategory"
                  :label="categoryLabel[String(signal.semanticCategory)] || signal.semanticCategory"
                  :color="categoryColor[String(signal.semanticCategory)] || 'neutral'"
                  variant="soft"
                  size="xs"
                />
              </div>
              <p class="text-sm mt-2">{{ signal.text }}</p>
            </div>
          </div>
        </UCard>
      </div>
    </template>
  </UDashboardPanel>
</template>
