<script setup lang="ts">
import type { AppSettings } from '~/types'

definePageMeta({
  middleware: 'auth'
})

const toast = useToast()
const auth = useAuthStore()
const canManageAISettings = computed(() => auth.isSuperAdmin)

const { data: settings, refresh } = await useFetch<AppSettings>('/api/settings', {
  default: () => ({
    lead_threshold: '0.70',
    sender_window_seconds: '60',
    trader_threshold: '0.60',
    merchant_threshold: '0.60',
    ps_offer_threshold: '0.60',
    noise_cleanup_enabled: 'true',
    show_multi_account_badges: 'true'
  })
})

function sliderToNumber(value: number | number[] | string | undefined, fallback: number): number {
  if (typeof value === 'string') {
    const parsed = Number(value)
    if (Number.isFinite(parsed)) return parsed
  }
  if (typeof value === 'number' && Number.isFinite(value)) return value
  if (Array.isArray(value) && typeof value[0] === 'number' && Number.isFinite(value[0])) return value[0]
  return fallback
}

const leadThreshold = ref(0.70)
const senderWindow = ref(60)
const traderThreshold = ref(0.60)
const merchantThreshold = ref(0.60)
const psOfferThreshold = ref(0.60)
const ignoreKeywords = ref<string[]>([])
const noiseCleanupEnabled = ref(true)
const showMultiAccountBadges = ref(true)
const showCleanupNoiseModal = ref(false)
const cleanupNoiseHours = ref('72')
const cleanupNoiseLoading = ref(false)
watch(settings, (next) => {
  if (!next) return
  leadThreshold.value = sliderToNumber(next.lead_threshold, 0.70)
  senderWindow.value = Math.round(sliderToNumber(next.sender_window_seconds, 60))
  traderThreshold.value = sliderToNumber(next.trader_threshold, 0.60)
  merchantThreshold.value = sliderToNumber(next.merchant_threshold, 0.60)
  psOfferThreshold.value = sliderToNumber(next.ps_offer_threshold, 0.60)
  ignoreKeywords.value = (next.ignore_keywords || '')
    .split(',')
    .map(s => s.trim())
    .filter(Boolean)
  noiseCleanupEnabled.value = next.noise_cleanup_enabled !== 'false'
  showMultiAccountBadges.value = next.show_multi_account_badges !== 'false'
}, { immediate: true })

const saving = ref(false)

async function save() {
  if (!settings.value) return
  const clamp = (val: number, min: number, max: number) =>
    Math.min(max, Math.max(min, val)).toFixed(2)

  const payload: AppSettings = {
    ...settings.value,
    lead_threshold: clamp(sliderToNumber(leadThreshold.value, 0.70), 0.3, 0.99),
    sender_window_seconds: String(Math.min(3600, Math.max(5, Math.round(sliderToNumber(senderWindow.value, 60))))),
    trader_threshold: clamp(sliderToNumber(traderThreshold.value, 0.60), 0.3, 0.99),
    merchant_threshold: clamp(sliderToNumber(merchantThreshold.value, 0.60), 0.3, 0.99),
    ps_offer_threshold: clamp(sliderToNumber(psOfferThreshold.value, 0.60), 0.3, 0.99),
    ignore_keywords: ignoreKeywords.value.join(', '),
    noise_cleanup_enabled: String(noiseCleanupEnabled.value),
    show_multi_account_badges: String(showMultiAccountBadges.value)
  }

  saving.value = true
  try {
    await $fetch('/api/settings', {
      method: 'PUT',
      body: payload
    })
    settings.value = payload
    toast.add({ title: 'Настройки сохранены', color: 'success' })
    await refresh()
  } catch (e: any) {
    toast.add({ title: 'Ошибка сохранения', description: e?.message, color: 'error' })
  } finally {
    saving.value = false
  }
}

async function runNoiseCleanup() {
  cleanupNoiseLoading.value = true
  try {
    const result = await $fetch<{ deleted: number, hours: number }>('/api/settings/cleanup-noise', {
      method: 'POST',
      body: { older_than_hours: cleanupNoiseHours.value }
    })
    toast.add({
      title: 'Очистка завершена',
      description: `Удалено шумовых сообщений: ${result.deleted}`,
      color: 'success'
    })
    showCleanupNoiseModal.value = false
  } catch (e: any) {
    toast.add({
      title: 'Ошибка очистки',
      description: e?.message || 'Не удалось запустить очистку шума',
      color: 'error'
    })
  } finally {
    cleanupNoiseLoading.value = false
  }
}
</script>

<template>
  <div class="space-y-6">
    <UAlert
      v-if="!canManageAISettings"
      color="warning"
      variant="soft"
      title="Доступ ограничен"
      description="Параметры ИИ доступны только роли Super Admin."
    />

    <template v-else>
      <UPageCard
        title="Настройки ИИ"
        description="Служебные параметры авто-классификации. Рекомендуется менять только при необходимости."
        variant="naked"
        orientation="horizontal"
        class="mb-4"
      >
        <UButton
          label="Сохранить"
          color="neutral"
          :loading="saving"
          class="w-fit lg:ms-auto"
          @click="save"
        />
        <UButton
          label="Очистить шум"
          color="warning"
          variant="soft"
          class="w-fit"
          @click="showCleanupNoiseModal = true"
        />
      </UPageCard>

      <UModal v-model:open="showCleanupNoiseModal" title="Очистка шумовых сообщений">
        <template #body>
          <div class="space-y-4">
            <p class="text-sm text-muted">
              Удалит шумовые сообщения из Mongo старше указанного периода.
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
            <UButton color="warning" :loading="cleanupNoiseLoading" @click="runNoiseCleanup">
              Запустить очистку
            </UButton>
          </div>
        </template>
      </UModal>

      <UPageCard variant="subtle">
        <div class="flex max-sm:flex-col justify-between items-start gap-4 py-4">
          <div class="flex-1">
            <p class="font-medium text-sm">
              Минимальная уверенность для авто-пометки как лид
            </p>
            <p class="text-xs text-muted mt-1">
              Если уверенность ниже, сигнал не будет автоматически продвигаться в лиды.
            </p>
          </div>
          <div class="flex items-center gap-3 min-w-48">
            <USlider
              v-model="leadThreshold"
              :min="0.3"
              :max="0.99"
              :step="0.01"
              tooltip
              class="flex-1"
            />
            <span class="font-mono text-sm w-10 text-right">{{ leadThreshold.toFixed(2) }}</span>
          </div>
        </div>

        <USeparator />

        <div class="flex max-sm:flex-col justify-between items-start gap-4 py-4">
          <div class="flex-1">
            <p class="font-medium text-sm">
              Автоочистка шума
            </p>
            <p class="text-xs text-muted mt-1">
              Включает периодическую очистку старых шумовых сигналов на backend.
            </p>
          </div>
          <USwitch v-model="noiseCleanupEnabled" />
        </div>

        <USeparator />

        <div class="flex max-sm:flex-col justify-between items-start gap-4 py-4">
          <div class="flex-1">
            <p class="font-medium text-sm">
              Показывать сигналы из 2+ чатов
            </p>
            <p class="text-xs text-muted mt-1">
              Управляет бейджами и служебной информацией о контактах, встречающихся в нескольких чатах.
            </p>
          </div>
          <USwitch v-model="showMultiAccountBadges" />
        </div>

        <USeparator />

        <div class="flex max-sm:flex-col justify-between items-start gap-4 py-4">
          <div class="flex-1">
            <p class="font-medium text-sm">
              Окно объединения сообщений (сек)
            </p>
            <p class="text-xs text-muted mt-1">
              Сообщения одного отправителя в этом окне обрабатываются как единый контекст.
            </p>
          </div>
          <div class="flex items-center gap-3 min-w-48">
            <USlider
              v-model="senderWindow"
              :min="5"
              :max="600"
              :step="5"
              tooltip
              class="flex-1"
            />
            <span class="font-mono text-sm w-12 text-right">{{ senderWindow }}с</span>
          </div>
        </div>

        <USeparator />

        <div class="flex max-sm:flex-col justify-between items-start gap-4 py-4">
          <div class="flex-1">
            <p class="font-medium text-sm">
              Минимальная уверенность для категории «Трейдер»
            </p>
            <p class="text-xs text-muted mt-1">
              Ниже порога категория не присваивается автоматически.
            </p>
          </div>
          <div class="flex items-center gap-3 min-w-48">
            <USlider
              v-model="traderThreshold"
              :min="0.3"
              :max="0.99"
              :step="0.01"
              tooltip
              class="flex-1"
            />
            <span class="font-mono text-sm w-10 text-right">{{ traderThreshold.toFixed(2) }}</span>
          </div>
        </div>

        <USeparator />

        <div class="flex max-sm:flex-col justify-between items-start gap-4 py-4">
          <div class="flex-1">
            <p class="font-medium text-sm">
              Минимальная уверенность для категории «Мерчант»
            </p>
            <p class="text-xs text-muted mt-1">
              Ниже порога категория не присваивается автоматически.
            </p>
          </div>
          <div class="flex items-center gap-3 min-w-48">
            <USlider
              v-model="merchantThreshold"
              :min="0.3"
              :max="0.99"
              :step="0.01"
              tooltip
              class="flex-1"
            />
            <span class="font-mono text-sm w-10 text-right">{{ merchantThreshold.toFixed(2) }}</span>
          </div>
        </div>

        <USeparator />

        <div class="flex max-sm:flex-col justify-between items-start gap-4 py-4">
          <div class="flex-1">
            <p class="font-medium text-sm">
              Минимальная уверенность для категории «Предложение ПС»
            </p>
            <p class="text-xs text-muted mt-1">
              Ниже порога категория не присваивается автоматически.
            </p>
          </div>
          <div class="flex items-center gap-3 min-w-48">
            <USlider
              v-model="psOfferThreshold"
              :min="0.3"
              :max="0.99"
              :step="0.01"
              tooltip
              class="flex-1"
            />
            <span class="font-mono text-sm w-10 text-right">{{ psOfferThreshold.toFixed(2) }}</span>
          </div>
        </div>

        <USeparator />

        <div class="flex max-sm:flex-col justify-between items-start gap-4 py-4">
          <div class="flex-1">
            <p class="font-medium text-sm">
              Ключевые слова для игнорирования
            </p>
            <p class="text-xs text-muted mt-1">
              Сигналы, содержащие эти слова (через запятую), будут игнорироваться на этапе парсинга.
            </p>
          </div>
          <div class="flex-1 w-full max-w-lg">
            <UInputTags
              v-model="ignoreKeywords"
              placeholder="Добавить слово..."
              :add-on-paste="true"
              :delimiter="/[,;]/g"
              class="w-full"
            />
          </div>
        </div>
      </UPageCard>
    </template>
  </div>
</template>
