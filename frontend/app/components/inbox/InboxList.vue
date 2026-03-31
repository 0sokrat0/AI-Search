<script setup lang="ts">
import { format, isToday } from 'date-fns'
import type { Mail } from '~/types'

const props = defineProps<{
  mails: Mail[]
}>()

const emit = defineEmits<{
  bottomChange: [value: boolean]
}>()

const mailsRefs = ref<Record<string, Element | null>>({})
const scrollContainer = ref<HTMLElement | null>(null)
const selectedIds = defineModel<string[]>('selectedIds', { default: () => [] })

const selectedMail = defineModel<Mail | null>()

function parseDateSafe(value?: string | null): Date | null {
  if (!value) return null
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? null : date
}

function formatListDate(value?: string | null): string {
  const date = parseDateSafe(value)
  if (!date) return '—'
  return isToday(date) ? format(date, 'HH:mm') : format(date, 'dd MMM')
}

function categoryLabel(category?: string | null): string {
  switch (String(category || '').toLowerCase()) {
    case 'trader_search':
      return 'Поиск трейдеров'
    case 'traders':
      return 'Трейдеры'
    case 'merchants':
    case 'processing_requests':
      return 'Мерчанты'
    case 'ps_offers':
      return 'Предложения от ПС'
    default:
      return 'Шум'
  }
}

function categoryColor(category?: string | null): 'success' | 'info' | 'warning' | 'primary' | 'error' {
  switch (String(category || '').toLowerCase()) {
    case 'trader_search':
      return 'warning'
    case 'traders':
      return 'info'
    case 'merchants':
    case 'processing_requests':
      return 'info'
    case 'ps_offers':
      return 'primary'
    default:
      return 'error'
  }
}

function bestBusinessMatch(mail: Mail): { label: string, percent: number } | null {
  const candidates = [
    { label: 'Трейдеры', score: Number(mail.traderScore ?? 0) },
    { label: 'Мерчанты', score: Number(mail.merchantScore ?? 0) },
    { label: 'ПС', score: Number(mail.psOfferScore ?? 0) }
  ]

  const best = candidates.sort((a, b) => b.score - a.score)[0]
  if (!best || best.score <= 0) return null

  return {
    label: best.label,
    percent: Math.round(best.score * 100)
  }
}

function currentCategoryShortLabel(mail: Mail): string | null {
  switch (String(mail.category || '').toLowerCase()) {
    case 'trader_search':
      return 'Поиск трейдеров'
    case 'traders':
      return 'Трейдеры'
    case 'merchants':
      return 'Мерчанты'
    case 'ps_offers':
      return 'ПС'
    default:
      return null
  }
}

function sourceLabel(mail: Mail): string | null {
  switch (String(mail.chatPeerType || '').toLowerCase()) {
    case 'channel':
      return 'Канал'
    case 'supergroup':
      return 'Супергруппа'
    case 'group':
      return 'Группа'
    default:
      return null
  }
}

function sourceColor(mail: Mail): 'info' | 'warning' | 'neutral' {
  switch (String(mail.chatPeerType || '').toLowerCase()) {
    case 'channel':
      return 'info'
    case 'supergroup':
      return 'warning'
    default:
      return 'neutral'
  }
}

function isSelected(signalId: string): boolean {
  return selectedIds.value.includes(signalId)
}

function toggleSelected(signalId: string, checked: boolean | 'indeterminate') {
  if (checked) {
    if (!selectedIds.value.includes(signalId)) {
      selectedIds.value = [...selectedIds.value, signalId]
    }
    return
  }
  selectedIds.value = selectedIds.value.filter(id => id !== signalId)
}

function reportBottomState() {
  const element = scrollContainer.value
  if (!element) {
    emit('bottomChange', false)
    return
  }

  const remaining = element.scrollHeight - element.scrollTop - element.clientHeight
  emit('bottomChange', remaining <= 24)
}

watch(selectedMail, () => {
  if (!selectedMail.value) {
    return
  }
  const ref = mailsRefs.value[selectedMail.value.signalId]
  if (ref) {
    ref.scrollIntoView({ block: 'nearest' })
  }
})

watch(() => props.mails.length, async () => {
  await nextTick()
  reportBottomState()
})

defineShortcuts({
  arrowdown: () => {
    const index = props.mails.findIndex((mail: Mail) => mail.signalId === selectedMail.value?.signalId)

    if (index === -1) {
      selectedMail.value = props.mails[0]
    } else if (index < props.mails.length - 1) {
      selectedMail.value = props.mails[index + 1]
    }
  },
  arrowup: () => {
    const index = props.mails.findIndex((mail: Mail) => mail.signalId === selectedMail.value?.signalId)

    if (index === -1) {
      selectedMail.value = props.mails[props.mails.length - 1]
    } else if (index > 0) {
      selectedMail.value = props.mails[index - 1]
    }
  }
})
</script>

<template>
  <div
    ref="scrollContainer"
    class="overflow-y-auto divide-y divide-default"
    aria-label="Список сигналов"
    @scroll.passive="reportBottomState"
  >
    <div
      v-for="(mail, index) in mails"
      :key="mail.signalId"
      :ref="(el) => { mailsRefs[mail.signalId] = el as Element | null }"
    >
      <button
        type="button"
        class="w-full text-left p-4 sm:px-6 text-sm cursor-pointer border-l-2 transition-colors"
        :class="[
          mail.isIgnored ? 'opacity-50' : '',
          mail.unread ? 'text-highlighted' : 'text-toned',
          selectedMail && selectedMail.signalId === mail.signalId
            ? 'border-primary bg-primary/10'
            : 'border-bg hover:border-primary hover:bg-primary/5'
        ]"
        :aria-label="`Открыть сигнал ${mail.subject}`"
        @click="selectedMail = mail"
      >
        <div class="mb-1.5" @click.stop>
          <UCheckbox
            :model-value="isSelected(mail.signalId)"
            aria-label="Выбрать сигнал"
            @update:model-value="(value) => toggleSelected(mail.signalId, value)"
          />
        </div>
        <div class="flex items-center justify-between" :class="[mail.unread && 'font-semibold']">
          <div class="flex items-center gap-1.5">
            {{ mail.from.name }}

            <UTooltip v-if="mail.isDm" text="Личное сообщение">
              <UIcon name="i-heroicons-envelope" class="size-3.5 text-muted shrink-0" />
            </UTooltip>

            <UBadge
              v-if="mail.unread"
              color="warning"
              variant="subtle"
              label="Новый"
            />
          </div>

          <span>{{ formatListDate(mail.date) }}</span>
        </div>
        <p class="truncate" :class="[mail.unread && 'font-semibold']">
          {{ mail.subject }}
        </p>
        <p class="text-dimmed line-clamp-1">
          {{ mail.body }}
        </p>

        <div class="flex items-center gap-1.5 mt-1.5 flex-wrap">
          <UBadge
            :label="categoryLabel(mail.category)"
            :color="categoryColor(mail.category)"
            variant="subtle"
            size="xs"
          />
          <UBadge
            v-if="sourceLabel(mail)"
            :label="sourceLabel(mail) || ''"
            :color="sourceColor(mail)"
            variant="soft"
            size="xs"
          />
          <UBadge
            v-if="bestBusinessMatch(mail)"
            :label="bestBusinessMatch(mail)?.label === currentCategoryShortLabel(mail)
              ? `${bestBusinessMatch(mail)?.percent}%`
              : `${bestBusinessMatch(mail)?.label} ${bestBusinessMatch(mail)?.percent}%`"
            color="neutral"
            variant="soft"
            size="xs"
          />
          <UBadge
            v-if="mail.category !== 'noise'"
            :label="mail.leadId ? 'В лид-воронке' : 'Только сигнал, не лид'"
            :color="mail.leadId ? 'success' : 'warning'"
            variant="soft"
            size="xs"
          />
          <UBadge
            v-if="mail.isBroadcast"
            icon="i-lucide-layers"
            :label="`${mail.broadcastCount} источников`"
            color="warning"
            variant="subtle"
            size="xs"
          />
          <UBadge
            v-if="mail.categoryAssignedAt"
            icon="i-lucide-clock"
            :label="formatListDate(mail.categoryAssignedAt)"
            color="neutral"
            variant="soft"
            size="xs"
          />
          <UBadge
            v-if="mail.semanticFlags?.includes('has_traffic')"
            icon="i-lucide-check-circle-2"
            label="Трафик"
            color="success"
            variant="subtle"
            size="xs"
          />
          <UBadge
            v-if="mail.merchantName"
            icon="i-lucide-building-2"
            :label="mail.merchantName"
            color="info"
            variant="subtle"
            size="xs"
          />
        </div>

        <div v-if="mail.isTeamMember || mail.isIgnored || (mail.showMultiAccountBadges !== false && mail.otherChatsCount > 1)" class="flex items-center gap-1.5 mt-1.5 flex-wrap">
          <UBadge
            v-if="mail.isTeamMember"
            icon="i-lucide-users"
            label="Команда"
            color="info"
            variant="subtle"
            size="xs"
          />
          <UBadge
            v-if="mail.isIgnored"
            icon="i-lucide-archive"
            label="Архив"
            color="warning"
            variant="subtle"
            size="xs"
          />
          <UBadge
            v-if="mail.showMultiAccountBadges !== false && mail.otherChatsCount > 1"
            icon="i-lucide-messages-square"
            :label="`${mail.otherChatsCount} чатов`"
            color="warning"
            variant="subtle"
            size="xs"
          />
        </div>
      </button>
    </div>
  </div>
</template>
