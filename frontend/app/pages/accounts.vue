<script setup lang="ts">
import type { TelegramAccount } from '~/types'

definePageMeta({
  middleware: 'auth'
})

const toast = useToast()
const { data: accounts, refresh, pending } = await useFetch<TelegramAccount[]>('/api/v1/accounts', {
  default: () => []
})

const isAddModalOpen = ref(false)
const newAccount = ref({
  method: 'qr' as 'qr' | 'code',
  phone: '',
  password: '',
  proxy: '',
  proxy_fallback: ''
})

const isAuthModalOpen = ref(false)
const currentAuthAccount = ref<TelegramAccount | null>(null)
const authMethod = ref<'qr' | 'code'>('qr')
const otpCode = ref('')
const password = ref('')
const authStep = ref<'method' | 'qr' | 'code' | 'password'>('method')
const loading = ref(false)
const autoSubmittingPassword = ref(false)

let pollTimer: any = null

const qrUrl = computed(() => {
  if (!currentAuthAccount.value) return ''
  const acc = accounts.value?.find(a => a.id === currentAuthAccount.value?.id)
  return acc?.qr_url || ''
})

watch(isAuthModalOpen, (val) => {
  if (val) {
    startPolling()
  } else {
    stopPolling()
  }
})

function startPolling() {
  stopPolling()
  pollTimer = setInterval(async () => {
    await refresh()
    const acc = accounts.value?.find(a => a.id === currentAuthAccount.value?.id)
    if (acc?.status === 'active' || acc?.status === 'authorized') {
      toast.add({
        title: acc.status === 'active' ? 'Аккаунт успешно подключен' : 'Сессия сохранена',
        description: acc.status === 'authorized' ? 'Теперь можно запустить парсер вручную.' : undefined,
        color: 'success'
      })
      isAuthModalOpen.value = false
      password.value = ''
    } else if (acc?.waiting_for_password) {
      if (password.value && !autoSubmittingPassword.value) {
        autoSubmittingPassword.value = true
        try {
          await submitPassword(true)
        } finally {
          autoSubmittingPassword.value = false
        }
      } else if (!password.value && authStep.value !== 'password') {
        authStep.value = 'password'
      }
    }
  }, 2000)
}

function stopPolling() {
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = null
  }
}

async function addAccount() {
  if (!newAccount.value.password) return
  if (newAccount.value.method === 'code' && !newAccount.value.phone) return
  loading.value = true
  try {
    const useQR = newAccount.value.method === 'qr'
    const acc = await $fetch<TelegramAccount>('/api/v1/accounts', {
      method: 'POST',
      body: {
        phone: useQR ? '' : newAccount.value.phone,
        proxy: newAccount.value.proxy,
        proxy_fallback: newAccount.value.proxy_fallback
      }
    })
    isAddModalOpen.value = false
    password.value = newAccount.value.password
    newAccount.value = { method: 'qr', phone: '', password: '', proxy: '', proxy_fallback: '' }
    toast.add({ title: 'Аккаунт добавлен', color: 'success' })
    await refresh()
    currentAuthAccount.value = acc
    authMethod.value = useQR ? 'qr' : 'code'
    authStep.value = useQR ? 'qr' : 'code'
    isAuthModalOpen.value = true

    await $fetch(`/api/v1/accounts/${acc.id}/auth?qr=${useQR}`, {
      method: 'POST'
    })
  } catch (e: any) {
    toast.add({ title: 'Ошибка', description: e.data?.error || e.message, color: 'error' })
  } finally {
    loading.value = false
  }
}

async function startAuth(account: TelegramAccount) {
  currentAuthAccount.value = account
  authStep.value = 'method'
  isAuthModalOpen.value = true
  otpCode.value = ''
  password.value = ''
  autoSubmittingPassword.value = false
}

async function selectAuthMethod(method: 'qr' | 'code') {
  if (!currentAuthAccount.value) return
  authMethod.value = method
  loading.value = true
  try {
    if (method === 'qr') {
      authStep.value = 'qr'
      await $fetch(`/api/v1/accounts/${currentAuthAccount.value.id}/auth?qr=true`, {
        method: 'POST'
      })
    } else {
      await $fetch(`/api/v1/accounts/${currentAuthAccount.value.id}/auth?qr=false`, {
        method: 'POST'
      })
      authStep.value = 'code'
    }
  } catch (e: any) {
    toast.add({ title: 'Ошибка', description: e.data?.error || e.message, color: 'error' })
    isAuthModalOpen.value = false
  } finally {
    loading.value = false
  }
}

async function submitCode() {
  if (!otpCode.value || !currentAuthAccount.value) return
  loading.value = true
  try {
    await $fetch(`/api/v1/accounts/${currentAuthAccount.value.id}/code`, {
      method: 'POST',
      body: { code: otpCode.value }
    })
    toast.add({ title: 'Код отправлен', color: 'success' })
    authStep.value = 'password'
  } catch (e: any) {
    toast.add({ title: 'Ошибка', description: e.data?.error || e.message, color: 'error' })
  } finally {
    loading.value = false
  }
}

async function submitPassword(silent = false) {
  if (!password.value || !currentAuthAccount.value) return
  loading.value = true
  try {
    await $fetch(`/api/v1/accounts/${currentAuthAccount.value.id}/password`, {
      method: 'POST',
      body: { password: password.value }
    })
    if (!silent) {
      toast.add({ title: 'Пароль принят', color: 'success' })
    }
    await refresh()
  } catch (e: any) {
    toast.add({ title: 'Ошибка', description: e.data?.error || e.message, color: 'error' })
  } finally {
    loading.value = false
  }
}

async function deleteAccount(id: string) {
  if (!confirm('Вы уверены, что хотите удалить этот аккаунт?')) return
  try {
    await $fetch(`/api/v1/accounts/${id}`, {
      method: 'DELETE'
    })
    toast.add({ title: 'Аккаунт удален', color: 'success' })
    await refresh()
  } catch (e: any) {
    toast.add({ title: 'Ошибка', description: e.data?.error || e.message, color: 'error' })
  }
}

async function startAccount(id: string) {
  loading.value = true
  try {
    await $fetch(`/api/v1/accounts/${id}/start`, {
      method: 'POST'
    })
    toast.add({ title: 'Запуск аккаунта начат', color: 'success' })
    await refresh()
  } catch (e: any) {
    toast.add({ title: 'Ошибка', description: e.data?.error || e.message, color: 'error' })
  } finally {
    loading.value = false
  }
}

async function stopAccount(id: string) {
  loading.value = true
  try {
    await $fetch(`/api/v1/accounts/${id}/stop`, {
      method: 'POST'
    })
    toast.add({ title: 'Парсинг остановлен', color: 'success' })
    await refresh()
  } catch (e: any) {
    toast.add({ title: 'Ошибка', description: e.data?.error || e.message, color: 'error' })
  } finally {
    loading.value = false
  }
}

async function restartAccount(id: string) {
  loading.value = true
  try {
    await $fetch(`/api/v1/accounts/${id}/restart`, {
      method: 'POST'
    })
    toast.add({ title: 'Рестарт аккаунта запущен', color: 'success' })
    await refresh()
  } catch (e: any) {
    toast.add({ title: 'Ошибка', description: e.data?.error || e.message, color: 'error' })
  } finally {
    loading.value = false
  }
}

function getStatusColor(status: string) {
  switch (status) {
    case 'active': return 'success'
    case 'authorized': return 'primary'
    case 'starting': return 'primary'
    case 'auth_pending': return 'warning'
    case 'unauthorized': return 'error'
    case 'disabled': return 'neutral'
    default: return 'neutral'
  }
}

function getStatusLabel(status: string) {
  switch (status) {
    case 'active': return 'Активен'
    case 'authorized': return 'Авторизован'
    case 'starting': return 'Запускается'
    case 'auth_pending': return 'Ожидает авторизации'
    case 'unauthorized': return 'Не авторизован'
    case 'disabled': return 'Отключен'
    default: return status
  }
}

const columns = [
  { id: 'phone', accessorKey: 'phone', header: 'Телефон' },
  { id: 'name', accessorKey: 'name', header: 'Имя / Сессия' },
  { id: 'status', accessorKey: 'status', header: 'Статус' },
  { id: 'actions', accessorKey: 'actions', header: '' }
]

onUnmounted(() => {
  stopPolling()
})
</script>

<template>
  <UDashboardPanel id="accounts" :ui="{ body: 'lg:py-12' }">
    <template #header>
      <UDashboardNavbar title="Аккаунты Telegram">
        <template #leading>
          <UDashboardSidebarCollapse />
        </template>
      </UDashboardNavbar>
    </template>

    <template #body>
      <div class="flex flex-col gap-4 sm:gap-6 lg:gap-12 w-full lg:max-w-5xl mx-auto">
        <div class="flex items-start justify-between gap-4">
          <div>
            <h1 class="text-2xl font-bold">Аккаунты Telegram</h1>
            <p class="text-sm text-muted">Управление сессиями для сбора сигналов.</p>
          </div>

          <UButton
            label="Добавить аккаунт"
            icon="i-lucide-plus"
            color="primary"
            class="shrink-0"
            @click="isAddModalOpen = true"
          />
        </div>

        <UPageCard variant="subtle" class="p-0 overflow-hidden">
          <UTable
            :data="accounts || []"
            :columns="columns"
            :loading="pending"
          >
            <template #phone-cell="{ row }">
              <div class="flex items-center gap-3">
                <UAvatar :src="row.original.avatar_url" :alt="row.original.phone" size="sm" />
                <span class="font-medium font-mono text-sm">{{ row.original.phone }}</span>
              </div>
            </template>

            <template #name-cell="{ row }">
              <div class="flex flex-col">
                <span class="text-sm">{{ row.original.name || '—' }}</span>
                <span v-if="row.original.username" class="text-xs text-muted">@{{ row.original.username }}</span>
              </div>
            </template>

            <template #status-cell="{ row }">
              <UBadge :color="getStatusColor(row.original.status)" variant="soft" size="sm">
                {{ getStatusLabel(row.original.status) }}
              </UBadge>
            </template>

            <template #actions-cell="{ row }">
              <div class="flex justify-end gap-2 px-4">
                <UButton
                  v-if="row.original.status === 'authorized'"
                  label="Запустить"
                  icon="i-lucide-play"
                  size="xs"
                  color="primary"
                  variant="subtle"
                  :loading="loading"
                  @click="startAccount(row.original.id)"
                />
                <UButton
                  v-if="row.original.status === 'active' || row.original.status === 'authorized' || row.original.status === 'unauthorized' || row.original.status === 'disabled'"
                  :label="row.original.status === 'unauthorized' ? 'Войти' : 'Переавторизовать'"
                  icon="i-lucide-log-in"
                  size="xs"
                  color="neutral"
                  variant="subtle"
                  @click="startAuth(row.original)"
                />
                <UButton
                  v-if="row.original.status === 'active' || row.original.status === 'starting'"
                  label="Стоп"
                  icon="i-lucide-square"
                  size="xs"
                  color="warning"
                  variant="subtle"
                  :loading="loading"
                  @click="stopAccount(row.original.id)"
                />
                <UButton
                  v-if="row.original.status === 'active' || row.original.status === 'authorized'"
                  label="Рестарт"
                  icon="i-lucide-rotate-cw"
                  size="xs"
                  color="neutral"
                  variant="subtle"
                  :loading="loading"
                  @click="restartAccount(row.original.id)"
                />
                <UButton
                  icon="i-lucide-trash-2"
                  color="error"
                  variant="ghost"
                  size="xs"
                  @click="deleteAccount(row.original.id)"
                />
              </div>
            </template>

            <template #empty-state>
              <div class="flex flex-col items-center justify-center py-12 gap-3 text-muted">
                <UIcon name="i-lucide-users" class="size-8" />
                <p>Аккаунтов пока нет. Добавьте первый, чтобы начать работу.</p>
                <UButton label="Добавить" variant="link" @click="isAddModalOpen = true" />
              </div>
            </template>
          </UTable>
        </UPageCard>
      </div>
    </template>
  </UDashboardPanel>

  <UModal v-model:open="isAddModalOpen" title="Добавить аккаунт">
    <template #body>
      <div class="space-y-6">
        <div class="space-y-2">
          <p class="text-sm font-medium">Способ входа</p>
          <div class="grid grid-cols-2 gap-2">
            <UButton
              label="QR-код"
              icon="i-lucide-qr-code"
              :color="newAccount.method === 'qr' ? 'primary' : 'neutral'"
              :variant="newAccount.method === 'qr' ? 'solid' : 'outline'"
              class="justify-center"
              @click="newAccount.method = 'qr'"
            />
            <UButton
              label="СМС-код"
              icon="i-lucide-message-square"
              :color="newAccount.method === 'code' ? 'primary' : 'neutral'"
              :variant="newAccount.method === 'code' ? 'solid' : 'outline'"
              class="justify-center"
              @click="newAccount.method = 'code'"
            />
          </div>
        </div>

        <UFormField v-if="newAccount.method === 'code'" label="Номер телефона">
          <UInput v-model="newAccount.phone" placeholder="+7..." icon="i-lucide-phone" class="w-full" />
          <template #help>В международном формате, например +79991234567</template>
        </UFormField>

        <UFormField label="2FA пароль" required>
          <UInput v-model="newAccount.password" type="password" placeholder="Облачный пароль Telegram" icon="i-lucide-lock" class="w-full" />
          <template #help>Обязательное поле. Если Telegram запросит 2FA, пароль отправится автоматически.</template>
        </UFormField>

        <UFormField label="Прокси">
          <UInput v-model="newAccount.proxy" placeholder="socks5://user:pass@host:port" icon="i-lucide-shield" class="w-full font-mono text-sm" />
          <template #help>SOCKS5 или MTProxy: <code>mtproto://host:port?secret=dd...</code></template>
        </UFormField>

        <UFormField label="Резервный прокси">
          <UInput v-model="newAccount.proxy_fallback" placeholder="mtproto://host:port?secret=dd..." icon="i-lucide-shield-check" class="w-full font-mono text-sm" />
          <template #help>Используется автоматически при недоступности основного прокси.</template>
        </UFormField>

        <div class="flex justify-end gap-3 mt-6">
          <UButton label="Отмена" color="neutral" variant="ghost" @click="isAddModalOpen = false" />
          <UButton
            :label="newAccount.method === 'qr' ? 'Создать и получить QR' : 'Создать и запросить код'"
            color="primary"
            :loading="loading"
            :disabled="!newAccount.password || (newAccount.method === 'code' && !newAccount.phone)"
            @click="addAccount"
          />
        </div>
      </div>
    </template>
  </UModal>

  <UModal v-model:open="isAuthModalOpen" :title="`Авторизация ${currentAuthAccount?.phone.startsWith('pending_qr') ? 'по QR-коду' : currentAuthAccount?.phone}`" prevent-close>
    <template #body>
      <div v-if="authStep === 'method'" class="py-4 space-y-4">
        <p class="text-sm">Выберите удобный способ входа в Telegram:</p>
        <div class="grid grid-cols-2 gap-4">
          <UButton
            label="QR-код"
            icon="i-lucide-qr-code"
            class="h-24 flex-col justify-center text-center"
            color="neutral"
            variant="outline"
            :loading="loading && authMethod === 'qr'"
            @click="selectAuthMethod('qr')"
          />
          <UButton
            label="Код из СМС"
            icon="i-lucide-message-square"
            class="h-24 flex-col justify-center text-center"
            color="neutral"
            variant="outline"
            :loading="loading && authMethod === 'code'"
            @click="selectAuthMethod('code')"
          />
        </div>
      </div>

      <div v-else-if="authStep === 'qr'" class="flex flex-col items-center gap-6 py-6 text-center">
        <p class="text-sm">
          Откройте Telegram -> Настройки -> Устройства -> Подключить устройство
        </p>
        <div class="bg-white p-4 rounded-xl border border-gray-200">
          <img v-if="qrUrl" :src="`https://api.qrserver.com/v1/create-qr-code/?size=256x256&data=${encodeURIComponent(qrUrl)}`" class="size-64" alt="QR" />
          <div v-else class="size-64 flex items-center justify-center">
            <UIcon name="i-lucide-loader-2" class="size-8 animate-spin text-primary" />
          </div>
        </div>
        <p class="text-xs text-muted">Код обновится автоматически, как только Telegram его сгенерирует.</p>
        <UButton label="Я отсканировал" color="primary" class="w-full" @click="isAuthModalOpen = false; refresh()" />
      </div>

      <div v-else-if="authStep === 'code'" class="space-y-4 py-4">
        <p class="text-sm">Введите 5-значный код подтверждения:</p>
        <UInput v-model="otpCode" placeholder="12345" size="lg" class="text-center font-mono tracking-widest" />
        <UButton label="Продолжить" color="primary" block :loading="loading" @click="submitCode" />
      </div>

      <div v-else-if="authStep === 'password'" class="space-y-4 py-4">
        <p class="text-sm text-warning">На аккаунте включен 2FA. Введите облачный пароль:</p>
        <UInput v-model="password" type="password" placeholder="Ваш пароль" size="lg" icon="i-lucide-lock" />
        <UButton label="Войти" color="primary" block :loading="loading" @click="() => submitPassword()" />
      </div>
    </template>
  </UModal>
</template>
