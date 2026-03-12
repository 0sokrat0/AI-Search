<script setup lang="ts">
definePageMeta({
  layout: 'auth',
  middleware: 'guest'
})

import { z } from 'zod'
import type { FormSubmitEvent } from '#ui/types'
import type { Role } from '~/types/auth'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()

const token = computed(() => String(route.params.token || ''))

const { data: inviteData, error: inviteError } = await useFetch<any>(() => `/api/v1/auth/invites/${token.value}`, {
  default: () => null,
  watch: [token]
})

const state = ref({
  name: '',
  email: '',
  password: '',
  passwordConfirm: ''
})

const schema = z.object({
  name: z.string().min(2, 'Минимум 2 символа'),
  email: z.string().email('Некорректный email'),
  password: z.string().min(8, 'Минимум 8 символов'),
  passwordConfirm: z.string().min(8, 'Минимум 8 символов')
}).refine((data) => data.password === data.passwordConfirm, {
  message: 'Пароли не совпадают',
  path: ['passwordConfirm']
})

type Schema = z.output<typeof schema>

const loading = ref(false)
const error = ref<string | null>(null)

type AcceptInviteResponse = {
  accessToken: string
  refreshToken: string
  user: {
    id: string
    email: string
    name: string
    roles: Role[]
    tenantID: string
    createdAt: string
  }
}

async function handleAccept(event: FormSubmitEvent<Schema>) {
  loading.value = true
  error.value = null

  try {
    const data = await $fetch<AcceptInviteResponse>(`/api/v1/auth/invites/${token.value}/accept`, {
      method: 'POST',
      body: {
        name: event.data.name,
        email: event.data.email,
        password: event.data.password,
        password_confirm: event.data.passwordConfirm
      }
    })

    authStore.setAuth(data.accessToken, data.refreshToken, data.user)
    await router.push('/')
  } catch (e: any) {
    error.value = e?.data?.message || e?.message || 'Не удалось активировать инвайт'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900 px-4 py-12">
    <div class="w-full max-w-md space-y-6">
      <div class="text-center">
        <h1 class="text-3xl font-bold text-gray-900 dark:text-white">
          Регистрация
        </h1>
        <p class="mt-2 text-sm text-gray-600 dark:text-gray-400">
          Создайте аккаунт по инвайт-ссылке
        </p>
      </div>

      <UCard>
        <template v-if="inviteError">
          <UAlert
            color="error"
            variant="soft"
            title="Инвайт недействителен"
            :description="inviteError.data?.message || inviteError.message || 'Ссылка истекла или уже была использована.'"
          />
        </template>

        <template v-else>
          <UForm :state="state" :schema="schema" @submit="handleAccept">
            <UFormField label="Имя" name="name" class="mb-4">
              <UInput v-model="state.name" class="w-full" placeholder="Иван Иванов" :disabled="loading" size="lg" />
            </UFormField>

            <UFormField label="Почта" name="email" class="mb-4">
              <UInput v-model="state.email" class="w-full" placeholder="user@example.com" type="email" :disabled="loading" size="lg" />
            </UFormField>

            <UFormField label="Пароль" name="password" class="mb-4">
              <UInput v-model="state.password" class="w-full" placeholder="••••••••" type="password" :disabled="loading" size="lg" />
            </UFormField>

            <UFormField label="Повтор пароля" name="passwordConfirm" class="mb-4">
              <UInput v-model="state.passwordConfirm" class="w-full" placeholder="••••••••" type="password" :disabled="loading" size="lg" />
            </UFormField>

            <UButton type="submit" color="primary" size="lg" block :loading="loading">
              Создать аккаунт
            </UButton>
          </UForm>
        </template>

        <template #footer>
          <UAlert v-if="!inviteError && error" color="error" variant="soft" :title="error" />
        </template>
      </UCard>
    </div>
  </div>
</template>
