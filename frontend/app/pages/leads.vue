<script setup lang="ts">
import type { TableColumn } from '@nuxt/ui'
import { upperFirst } from 'scule'
import { formatDistanceToNow, isValid } from 'date-fns'
import { useIntersectionObserver } from '@vueuse/core'
import type { CursorPage, Lead, LeadStatus } from '~/types'

interface GroupedLead extends Lead {
  chatTitles: string[]
  allIds: string[]
}

definePageMeta({
  middleware: 'auth'
})

const UAvatar = resolveComponent('UAvatar')
const UButton = resolveComponent('UButton')
const UBadge = resolveComponent('UBadge')
const UDropdownMenu = resolveComponent('UDropdownMenu')
const UCheckbox = resolveComponent('UCheckbox')

const toast = useToast()
const table = useTemplateRef<any>('table')
const loadMoreTrigger = useTemplateRef<HTMLElement | null>('loadMoreTrigger')
const router = useRouter()
const route = useRoute()

const isDetailRoute = computed(() => Boolean(route.params.id))

const columnFilters = ref([{ id: 'contact', value: '' }])
const columnVisibility = ref<Record<string, boolean>>({
  nextAction: false,
  status: false,
  signalsCount: false,
  geo: false
})
const rowSelection = ref({})
const statusFilter = ref('all')
const categoryFilter = ref<'all' | 'traders' | 'merchants' | 'ps_offers'>('all')
const leadScope = ref<'in_work' | 'archive'>('in_work')
const bulkLoading = ref(false)
const bulkStatus = ref<LeadStatus>('qualified')
const bulkCompanyId = ref<string>('')

const queryClient = useQueryClient()
const { companies, loading: companiesLoading } = useCompanies()
const companySelectItems = computed(() => companies.value.map(c => ({ label: c.name, value: c.id })))
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

const scopedLeads = computed(() => {
  return leadScope.value === 'archive'
    ? data.value.filter((l: Lead) => l.status === 'converted' || l.status === 'rejected' || l.status === 'false_positive')
    : data.value.filter((l: Lead) => l.status !== 'converted' && l.status !== 'rejected' && l.status !== 'false_positive')
})

const groupedScopedLeads = computed<GroupedLead[]>(() => {
  return scopedLeads.value.map((l: Lead) => ({
    ...l,
    chatTitles: l.chatTitle ? [l.chatTitle] : [],
    allIds: [l.id]
  }))
})

function formatLastSeen(value?: string) {
  if (!value) return '—'
  const d = new Date(value)
  if (!isValid(d)) return '—'
  return formatDistanceToNow(d, { addSuffix: true })
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
  rejected: 'error'
}

const statusLabel: Record<LeadStatus, string> = {
  new: 'Новый',
  detected: 'Обнаружен',
  confirmed: 'Подтвержден',
  controversial: 'Спорный',
  false_positive: 'Ложный (FP)',
  contacted: 'Первичный контакт',
  qualified: 'В работе',
  converted: 'Сделка / подключен',
  rejected: 'Мусор / закрыт'
}

const categoryLabel: Record<string, string> = {
  traders: 'Трейдеры',
  merchants: 'Мерчанты',
  ps_offers: 'Предложения от ПС'
}

const categoryColor: Record<string, 'primary' | 'success' | 'warning' | 'error' | 'neutral' | 'info'> = {
  traders: 'success',
  merchants: 'info',
  ps_offers: 'info'
}

const qualificationSourceLabel: Record<string, string> = {
  ai_qualified: 'Квалифицировано ИИ',
  manual_approved: 'Ручной апрув'
}

const qualificationSourceColor: Record<string, 'primary' | 'success' | 'info' | 'neutral'> = {
  ai_qualified: 'info',
  manual_approved: 'success'
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

function nextActionLabel(status: LeadStatus): string {
  switch (status) {
    case 'new':
    case 'detected':
      return 'Квалифицировать'
    case 'confirmed':
      return 'В работу'
    case 'controversial':
      return 'Перепроверить'
    case 'false_positive':
      return 'В корзину'
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

async function deleteLead(id: string, name?: string) {
  const ok = window.confirm(`Удалить лид "${name || id}" из CRM?`)
  if (!ok) return

  try {
    await $fetch(`/api/leads/${id}`, { method: 'DELETE' })
    await queryClient.invalidateQueries({ queryKey: ['leads'] })
    toast.add({
      title: 'Лид удален',
      description: 'Карточка удалена из таблицы лидов',
      color: 'success'
    })
  } catch (e: any) {
    toast.add({
      title: 'Ошибка удаления',
      description: e?.message || 'Не удалось удалить лид',
      color: 'error'
    })
  }
}


async function updateLeadStatus(id: string, status: LeadStatus) {
  try {
    await $fetch(`/api/leads/${id}/status`, { method: 'PATCH', body: { status } })
    await queryClient.invalidateQueries({ queryKey: ['leads'] })
    toast.add({
      title: 'Статус обновлен',
      description: `Лид переведен в статус "${statusLabel[status]}"`,
      color: 'success'
    })
  } catch (e: any) {
    toast.add({
      title: 'Ошибка',
      description: e?.message || 'Не удалось обновить статус',
      color: 'error'
    })
  }
}

const columns: TableColumn<GroupedLead>[] = [
  {
    id: 'select',
    header: ({ table }) =>
      h(UCheckbox, {
        'modelValue': table.getIsSomePageRowsSelected()
          ? 'indeterminate'
          : table.getIsAllPageRowsSelected(),
        'onUpdate:modelValue': (value: boolean | 'indeterminate') =>
          table.toggleAllPageRowsSelected(!!value),
        'ariaLabel': 'Выбрать все лиды'
      }),
    cell: ({ row }) =>
      h(UCheckbox, {
        'modelValue': row.getIsSelected(),
        'onUpdate:modelValue': (value: boolean | 'indeterminate') => row.toggleSelected(!!value),
        'ariaLabel': 'Выбрать лид'
      })
  },
  {
    accessorKey: 'name',
    header: 'Лид',
    cell: ({ row }) => {
      const grouped = row.original as GroupedLead
      const count = grouped.allIds?.length ?? 1
      return h('button', {
        class: 'flex items-center gap-3 text-left w-full',
        onClick: () => openLead(row.original.id)
      }, [
        h(UAvatar, {
          ...row.original.avatar,
          alt: row.original.name,
          size: 'lg'
        }),
        h('div', { class: 'flex items-center gap-2' }, [
          h('p', { class: 'font-medium text-highlighted' }, row.original.name),
          count > 1
            ? h(UBadge, { color: 'neutral', variant: 'soft', size: 'xs' }, () => `×${count}`)
            : null
        ])
      ])
    }
  },
  {
    accessorKey: 'text',
    header: 'Сигнал',
    cell: ({ row }) => h('p', { class: 'text-xs text-muted line-clamp-2 max-w-xs break-words' }, row.original.text)
  },
  {
    accessorKey: 'contact',
    header: 'Контакт',
    cell: ({ row }) => {
      const contact = row.original.contact || '—'
      const href = buildContactHref(row.original.contact)
      return h('div', { class: 'flex items-center gap-1.5' }, [
        h('span', { class: 'truncate max-w-24 text-xs' }, contact),
        href
          ? h(UButton, {
              icon: 'i-lucide-send',
              color: 'neutral',
              variant: 'ghost',
              size: 'xs',
              href,
              target: '_blank'
            })
          : null
      ])
    }
  },
  {
    accessorKey: 'chatTitle',
    header: 'Чат',
    cell: ({ row }) => {
      const grouped = row.original as GroupedLead
      const titles = grouped.chatTitles?.length ? grouped.chatTitles : [row.original.chatTitle].filter(Boolean)
      if (!titles.length) return '—'
      if (titles.length === 1) return titles[0]
      return h('div', { class: 'flex flex-wrap items-center gap-1' }, [
        h('span', { class: 'truncate max-w-28' }, titles[0]),
        h(UBadge, { color: 'neutral', variant: 'soft', size: 'xs' }, () => `+${titles.length - 1}`)
      ])
    }
  },
  {
    accessorKey: 'qualificationSource',
    header: 'Источник',
    cell: ({ row }) => {
      const source = String(row.original.qualificationSource || '')
      if (!source) return '—'
      return h(UBadge, {
        color: qualificationSourceColor[source] || 'neutral',
        variant: 'soft'
      }, () => qualificationSourceLabel[source] || source)
    }
  },
  {
    accessorKey: 'semanticCategory',
    header: 'Категория',
    cell: ({ row }) => {
      const c = String(row.original.semanticCategory || '')
      if (!c) return '—'
      return h(UBadge, {
        color: categoryColor[c] || 'neutral',
        variant: 'subtle'
      }, () => categoryLabel[c] || c)
    }
  },
  {
    accessorKey: 'merchantId',
    header: 'Компания',
    cell: ({ row }) => {
      const mid = row.original.merchantId || row.original.companyId
      if (mid && mid !== 'default') {
        const name = companies.value.find(c => c.id === mid)?.name
        if (name) return h('span', name)
      }
      return '—'
    }
  },
  {
    id: 'nextAction',
    header: 'Следующий шаг',
    cell: ({ row }) => h(UBadge, { color: 'neutral', variant: 'subtle' }, () => nextActionLabel(row.original.status))
  },
  {
    accessorKey: 'status',
    header: 'Статус',
    filterFn: 'equals',
    cell: ({ row }) => h(UBadge, {
      variant: 'subtle',
      color: statusColor[row.original.status]
    }, () => statusLabel[row.original.status])
  },
  {
    accessorKey: 'signalsCount',
    header: 'Сигналы',
    cell: ({ row }) => row.original.signalsCount ?? 1
  },
  {
    accessorKey: 'geo',
    header: 'Гео',
    cell: ({ row }) => (row.original.geo ?? []).join(', ')
  },
  {
    accessorKey: 'lastSeenAt',
    header: 'Последний',
    cell: ({ row }) => formatLastSeen(row.original.lastSeenAt)
  },
  {
    id: 'actions',
    cell: ({ row }) => h(
      'div',
      { class: 'flex items-center justify-end gap-1' },
      [
        h(UButton, {
          icon: 'i-lucide-eye',
          color: 'neutral',
          variant: 'ghost',
          size: 'sm',
          square: true,
          title: 'Открыть профиль лида',
          onClick: () => openLead(row.original.id)
        }),
        h(UButton, {
          icon: 'i-lucide-trash-2',
          color: 'error',
          variant: 'ghost',
          size: 'sm',
          square: true,
          title: 'Удалить из CRM',
          onClick: () => deleteLead(row.original.id, row.original.name)
        })
      ]
    )
  }
]
watch(() => statusFilter.value, (newVal) => {
  if (!table?.value?.tableApi) return

  const statusColumn = table.value.tableApi.getColumn('status')
  if (!statusColumn) return

  if (newVal === 'all') {
    statusColumn.setFilterValue(undefined)
  } else {
    statusColumn.setFilterValue(newVal)
  }
})

watch([categoryFilter, leadScope, statusFilter], () => {
})

useIntersectionObserver(loadMoreTrigger, async ([entry]) => {
  if (!entry?.isIntersecting || !hasNextPage.value || isFetchingNextPage.value) return
  await fetchNextPage()
})

const contact = computed({
  get: (): string => {
    return (table.value?.tableApi?.getColumn('contact')?.getFilterValue() as string) || ''
  },
  set: (value: string) => {
    table.value?.tableApi?.getColumn('contact')?.setFilterValue(value || undefined)
  }
})

const selectedLeadRows = computed<Array<{ original: Lead }>>(() => table.value?.tableApi?.getFilteredSelectedRowModel().rows ?? [])
const selectedLeadIds = computed<string[]>(() => selectedLeadRows.value.map((row: { original: Lead }) => row.original.id))

async function withSelectedLeads(action: (leadId: string) => Promise<unknown>, successTitle: string) {
  if (!selectedLeadIds.value.length) return
  bulkLoading.value = true
  try {
    const ids = [...selectedLeadIds.value]
    await Promise.all(ids.map((id: string) => action(id)))
    rowSelection.value = {}
    await queryClient.invalidateQueries({ queryKey: ['leads'] })
    toast.add({
      title: successTitle,
      description: `Обработано лидов: ${ids.length}`,
      color: 'success'
    })
  } catch (e: any) {
    toast.add({
      title: 'Ошибка массовой операции',
      description: e?.message || 'Не удалось применить изменения',
      color: 'error'
    })
  } finally {
    bulkLoading.value = false
  }
}

async function bulkArchive() {
  await withSelectedLeads(
    leadId => $fetch(`/api/leads/${leadId}/status`, { method: 'PATCH', body: { status: 'rejected' } }),
    'Лиды перемещены в архив'
  )
}

async function bulkUpdateStatus() {
  await withSelectedLeads(
    leadId => $fetch(`/api/leads/${leadId}/status`, { method: 'PATCH', body: { status: bulkStatus.value } }),
    `Обновлен статус: ${statusLabel[bulkStatus.value]}`
  )
}

async function bulkAssignCompany() {
  if (!bulkCompanyId.value) return
  const companyName = companies.value.find(c => c.id === bulkCompanyId.value)?.name ?? bulkCompanyId.value
  await withSelectedLeads(
    leadId => $fetch(`/api/leads/${leadId}/merchant`, { method: 'PUT', body: { merchant_id: bulkCompanyId.value } }),
    `Компания назначена: ${companyName}`
  )
}

function exportCSV() {
  const rows = (table.value?.tableApi?.getFilteredRowModel().rows ?? []) as Array<{ original: Lead }>
  const headers = ['ID', 'Имя', 'Контакт', 'Чат', 'Компания', 'Тип', 'Статус', 'Следующий шаг', 'Гео', 'Сигналы', 'Последний']
  const csvRows = rows.map((row: { original: Lead }) => {
    const l = row.original
    return [
      l.id,
      l.name,
      l.contact,
      l.chatTitle,
      l.company ?? l.merchantId ?? '',
      l.semanticCategory ?? 'leads',
      l.status,
      nextActionLabel(l.status),
      (l.geo ?? []).join('; '),
      l.signalsCount,
      l.lastSeenAt ?? ''
    ].map(v => `"${String(v).replace(/"/g, '""')}"`).join(',')
  })
  const csv = [headers.join(','), ...csvRows].join('\n')
  const blob = new Blob(['\uFEFF' + csv], { type: 'text/csv;charset=utf-8;' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `лиды_${new Date().toISOString().slice(0, 10)}.csv`
  a.click()
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
      <div class="flex flex-wrap items-center justify-between gap-1.5">
        <UInput
          v-model="contact"
          class="max-w-sm"
          icon="i-lucide-search"
          placeholder="Фильтр по контакту..."
        />

        <div class="flex flex-wrap items-center gap-1.5">
          <USelect
            v-model="leadScope"
            :items="[
              { label: 'В работе', value: 'in_work' },
              { label: 'Архив', value: 'archive' }
            ]"
            class="min-w-32"
          />

          <USelect
            v-model="categoryFilter"
            :items="[
              { label: 'Все типы', value: 'all' },
              { label: 'Трейдеры (P2P)', value: 'traders' },
              { label: 'Мерчанты (Интеграция)', value: 'merchants' },
              { label: 'Предложения от ПС', value: 'ps_offers' }
            ]"
            class="min-w-52"
          />

          <USelect
            v-model="statusFilter"
            :items="[
              { label: 'Все', value: 'all' },
              { label: 'Обнаружен (Detected)', value: 'detected' },
              { label: 'Подтвержден (Confirmed)', value: 'confirmed' },
              { label: 'Спорный (Controversial)', value: 'controversial' },
              { label: 'Ложный (False Positive)', value: 'false_positive' },
              { label: 'Новый', value: 'new' },
              { label: 'Первичный контакт', value: 'contacted' },
              { label: 'В работе', value: 'qualified' },
              { label: 'Сделка / подключен', value: 'converted' },
              { label: 'Мусор / закрыт', value: 'rejected' }
            ]"
            :ui="{ trailingIcon: 'group-data-[state=open]:rotate-180 transition-transform duration-200' }"
            placeholder="Фильтр статуса"
            class="min-w-36"
          />
          <UButton
            label="Экспорт"
            color="neutral"
            variant="outline"
            icon="i-lucide-download"
            @click="exportCSV"
          />

          <UDropdownMenu
            :items="
              table?.tableApi
                ?.getAllColumns()
                .filter((column: any) => column.getCanHide())
                .map((column: any) => ({
                  label: upperFirst(column.id),
                  type: 'checkbox' as const,
                  checked: column.getIsVisible(),
                  onUpdateChecked(checked: boolean) {
                    table?.tableApi?.getColumn(column.id)?.toggleVisibility(!!checked)
                  },
                  onSelect(e?: Event) {
                    e?.preventDefault()
                  }
                }))
            "
            :content="{ align: 'end' }"
          >
            <UButton
              label="Видимость"
              color="neutral"
              variant="outline"
              trailing-icon="i-lucide-settings-2"
            />
          </UDropdownMenu>
        </div>
      </div>

      <div v-if="selectedLeadIds.length" class="rounded-lg border border-default p-3 space-y-2">
        <p class="text-xs text-muted">
          Bulk actions: выбрано {{ selectedLeadIds.length }} лидов
        </p>
        <div class="flex flex-wrap items-center gap-2">
          <UButton
            size="xs"
            color="neutral"
            variant="soft"
            :loading="bulkLoading"
            @click="bulkArchive"
          >
            Архивировать
          </UButton>
          <USelect
            v-model="bulkStatus"
            :items="[
              { label: 'Новый', value: 'new' },
              { label: 'Первичный контакт', value: 'contacted' },
              { label: 'В работе', value: 'qualified' },
              { label: 'Сделка / подключен', value: 'converted' },
              { label: 'Мусор / закрыт', value: 'rejected' }
            ]"
            size="xs"
            class="min-w-44"
          />
          <UButton
            size="xs"
            color="primary"
            variant="soft"
            :loading="bulkLoading"
            @click="bulkUpdateStatus"
          >
            Сменить статус
          </UButton>
          <USelect
            v-model="bulkCompanyId"
            :items="companySelectItems"
            :loading="companiesLoading"
            placeholder="Назначить компанию..."
            size="xs"
            class="min-w-56"
          />
          <UButton
            size="xs"
            color="info"
            variant="soft"
            :loading="bulkLoading"
            :disabled="!bulkCompanyId"
            @click="bulkAssignCompany"
          >
            Назначить
          </UButton>
        </div>
      </div>

      <UTable
        ref="table"
        v-model:column-filters="columnFilters"
        v-model:column-visibility="columnVisibility"
        v-model:row-selection="rowSelection"
        class="shrink-0"
        :data="groupedScopedLeads"
        :columns="columns"
        :loading="isPending"
        :ui="{
          base: 'table-fixed border-separate border-spacing-0',
          thead: '[&>tr]:bg-elevated/50 [&>tr]:after:content-none',
          tbody: '[&>tr]:last:[&>td]:border-b-0',
          th: 'py-2 first:rounded-l-lg last:rounded-r-lg border-y border-default first:border-l last:border-r',
          td: 'border-b border-default',
          separator: 'h-0'
        }"
      />

      <div class="flex items-center justify-between gap-3 border-t border-default pt-4 mt-auto">
        <div class="text-sm text-muted">
          Выбрано {{ table?.tableApi?.getFilteredSelectedRowModel().rows.length || 0 }} из
          {{ table?.tableApi?.getFilteredRowModel().rows.length || 0 }} лидов.
        </div>
      </div>

      <div ref="loadMoreTrigger" class="py-4">
        <div v-if="isFetchingNextPage" class="space-y-2">
          <USkeleton class="h-12 w-full" />
          <USkeleton class="h-12 w-full" />
        </div>
        <p v-else-if="hasNextPage" class="text-xs text-center text-muted">
          Прокрутите ниже, чтобы загрузить ещё лиды
        </p>
      </div>
    </template>
  </UDashboardPanel>
</template>
