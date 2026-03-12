<script setup lang="ts">
import { z } from 'zod'
import type { FormSubmitEvent } from '#ui/types'
import type { Role } from '~/types/auth'

const authStore = useAuthStore()
const router = useRouter()

const state = ref({
  email: '',
  password: ''
})

const schema = z.object({
  email: z.string().email('Некорректный email'),
  password: z.string().min(1, 'Введите пароль')
})

type Schema = z.output<typeof schema>

const loading = ref(false)
const error = ref<string | null>(null)

type LoginResponse = {
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

async function handleLogin(event: FormSubmitEvent<Schema>) {
  loading.value = true
  error.value = null

  try {
    const data = await $fetch<LoginResponse>('/api/v1/auth/login', {
      method: 'POST',
      body: {
        email: event.data.email,
        password: event.data.password
      }
    })

    authStore.setAuth(data.accessToken, data.refreshToken, data.user)
    await router.push('/')
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : 'Неизвестная ошибка'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="w-full max-w-md space-y-6">
    <div class="text-center">
      <h1 class="text-3xl font-bold text-gray-900 dark:text-white">
        Вход в систему
      </h1>
      <p class="mt-2 text-sm text-gray-600 dark:text-gray-400">
        Админ-панель MRG Assistant
      </p>
    </div>

    <UCard>
      <UForm :state="state" :schema="schema" @submit="handleLogin">
        <UFormField label="Почта" name="email" class="mb-4">
          <UInput class="w-full" v-model="state.email" placeholder="admin@example.com" icon="i-lucide-mail" autocomplete="email"
            :disabled="loading" size="lg" />
        </UFormField>

        <UFormField label="Пароль" name="password" class="mb-4">
          <UInput class="w-full" v-model="state.password" type="password" placeholder="••••••••" icon="i-lucide-lock"
            autocomplete="current-password" :disabled="loading" size="lg" />
        </UFormField>

        <UButton type="submit" color="primary" size="lg" block :loading="loading">
          Войти
        </UButton>
      </UForm>

      <template #footer>
        <UAlert v-if="error" color="error" variant="soft" :title="error" />
      </template>
    </UCard>

    <div class="text-center text-xs text-gray-500 dark:text-gray-400">
      © 2025 MRG Assistant. Все права защищены.
    </div>
  </div>
</template>
