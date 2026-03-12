<script setup lang="ts">
definePageMeta({
  middleware: 'auth'
})

import * as z from 'zod'
import type { Member } from '~/types'

const toast = useToast()
const auth = useAuthStore()
const q = ref('')
const showInvite = ref(false)
const deleteTarget = ref<{ id: string; name: string } | null>(null)
const inviteLoading = ref(false)
const deleteLoading = ref(false)
const generatedInviteLink = ref('')

const { data: users, refresh } = await useFetch<Member[]>('/api/members', {
  default: () => []
})

const filtered = computed(() =>
  (users.value || []).filter(u =>
    u.name.toLowerCase().includes(q.value.toLowerCase()) ||
    u.email.toLowerCase().includes(q.value.toLowerCase())
  )
)

const inviteSchema = z.object({
  role: z.enum(['super_admin', 'employee'])
})

type InviteSchema = z.output<typeof inviteSchema>

const inviteForm = reactive<InviteSchema>({
  role: 'employee'
})

const roleOptions = [
  { label: 'Супер админ', value: 'super_admin' },
  { label: 'Сотрудник', value: 'employee' }
]

const roleColor: Record<string, 'success' | 'warning' | 'error' | 'neutral' | 'info' | 'primary'> = {
  super_admin: 'error',
  employee: 'success'
}

const roleLabel: Record<string, string> = {
  super_admin: 'Супер админ',
  employee: 'Сотрудник'
}

async function onGenerateInvite() {
  const parsed = inviteSchema.safeParse(inviteForm)
  if (!parsed.success) {
    const first = parsed.error.issues[0]
    toast.add({ title: 'Проверьте форму', description: first?.message || 'Некорректные данные', color: 'warning' })
    return
  }

  inviteLoading.value = true
  try {
    const response = await $fetch<any>('/api/users/invites', {
      method: 'POST',
      body: {
        role: parsed.data.role
      }
    })

    const payload = response?.data ?? response ?? {}
    const token = String(payload.token || '')
    if (!token) {
      throw new Error('Токен инвайта не вернулся')
    }

    const origin = window.location.origin
    generatedInviteLink.value = `${origin}/invite/${token}`
    await navigator.clipboard.writeText(generatedInviteLink.value)
    toast.add({ title: 'Инвайт создан', description: 'Ссылка скопирована в буфер обмена', color: 'success' })
  } catch (e: any) {
    toast.add({ title: 'Ошибка', description: e?.data?.message || e?.message, color: 'error' })
  } finally {
    inviteLoading.value = false
  }
}

async function onCopyInvite() {
  if (!generatedInviteLink.value) return
  await navigator.clipboard.writeText(generatedInviteLink.value)
  toast.add({ title: 'Ссылка скопирована', color: 'success' })
}

async function onDelete() {
  if (!deleteTarget.value) return
  deleteLoading.value = true
  try {
    await $fetch(`/api/users/${deleteTarget.value.id}`, { method: 'DELETE' })
    toast.add({ title: 'Пользователь удалён', color: 'success' })
    deleteTarget.value = null
    await refresh()
  } catch (e: any) {
    toast.add({ title: 'Ошибка удаления', description: e?.message, color: 'error' })
  } finally {
    deleteLoading.value = false
  }
}

watch(showInvite, (open) => {
  if (!open) {
    generatedInviteLink.value = ''
    inviteForm.role = 'employee'
  }
})
</script>

<template>
  <div>
    <UPageCard
      title="Пользователи системы"
      description="Управление доступом к панели администратора. Новые пользователи подключаются по инвайт-ссылке."
      variant="naked"
      orientation="horizontal"
      class="mb-4"
    >
      <UButton
        v-if="auth.isSuperAdmin"
        label="Сгенерировать инвайт"
        icon="i-lucide-user-plus"
        color="primary"
        class="w-fit lg:ms-auto"
        @click="showInvite = true"
      />
    </UPageCard>

    <UPageCard variant="subtle" :ui="{ container: 'p-0 sm:p-0 gap-y-0', wrapper: 'items-stretch', header: 'p-4 mb-0 border-b border-default' }">
      <template #header>
        <UInput
          v-model="q"
          icon="i-lucide-search"
          placeholder="Поиск по имени или почте..."
          class="w-full"
        />
      </template>

      <div v-if="!filtered.length" class="p-6 text-center text-muted text-sm">
        Пользователи не найдены
      </div>
      <ul v-else role="list" class="divide-y divide-default">
        <li
          v-for="user in filtered"
          :key="user.id"
          class="flex items-center justify-between gap-3 py-3 px-4 sm:px-6"
        >
          <div class="flex items-center gap-3 min-w-0">
            <UAvatar :alt="user.name" size="md" />
            <div class="text-sm min-w-0">
              <p class="text-highlighted font-medium truncate">{{ user.name }}</p>
              <p class="text-muted truncate">{{ user.email }}</p>
            </div>
          </div>

          <div class="flex items-center gap-2 shrink-0">
            <div class="flex gap-1 flex-wrap justify-end">
              <UBadge
                v-for="role in user.roles"
                :key="role"
                :color="roleColor[role] ?? 'neutral'"
                variant="subtle"
                size="xs"
              >
                {{ roleLabel[role] ?? role }}
              </UBadge>
            </div>

            <UBadge :color="user.is_active ? 'success' : 'neutral'" variant="subtle" size="sm">
              {{ user.is_active ? 'Активен' : 'Неактивен' }}
            </UBadge>

            <UDropdownMenu
              :items="[{
                label: 'Удалить',
                icon: 'i-lucide-trash-2',
                color: 'error',
                onSelect: () => deleteTarget = { id: user.id, name: user.name }
              }]"
              :content="{ align: 'end' }"
            >
              <UButton icon="i-lucide-ellipsis-vertical" color="neutral" variant="ghost" />
            </UDropdownMenu>
          </div>
        </li>
      </ul>
    </UPageCard>

    <UModal v-model:open="showInvite" title="Новый инвайт" :ui="{ footer: 'justify-end' }">
      <template #body>
        <UForm :schema="inviteSchema" :state="inviteForm" class="space-y-4" @submit="onGenerateInvite">
          <UFormField name="role" label="Роль" required>
            <USelect v-model="inviteForm.role" :items="roleOptions" class="w-full" />
          </UFormField>

          <UAlert
            color="neutral"
            variant="soft"
            title="Как это работает"
            description="Пользователь откроет ссылку, задаст почту и пароль, после чего аккаунт будет создан автоматически."
          />

          <UFormField v-if="generatedInviteLink" name="invite_link" label="Ссылка">
            <UInput :model-value="generatedInviteLink" readonly class="w-full" />
          </UFormField>
        </UForm>
      </template>
      <template #footer>
        <UButton label="Отмена" color="neutral" variant="ghost" @click="showInvite = false" />
        <UButton v-if="generatedInviteLink" label="Копировать" color="neutral" variant="soft" @click="onCopyInvite" />
        <UButton label="Сгенерировать" color="primary" :loading="inviteLoading" @click="onGenerateInvite" />
      </template>
    </UModal>

    <UModal
      v-if="deleteTarget"
      :open="!!deleteTarget"
      title="Удалить пользователя?"
      :ui="{ footer: 'justify-end' }"
      @update:open="(v) => { if (!v) deleteTarget = null }"
    >
      <template #body>
        <p class="text-sm text-muted">
          Пользователь <span class="font-semibold text-highlighted">{{ deleteTarget.name }}</span> будет удалён без возможности восстановления.
        </p>
      </template>
      <template #footer>
        <UButton label="Отмена" color="neutral" variant="ghost" @click="deleteTarget = null" />
        <UButton label="Удалить" color="error" :loading="deleteLoading" @click="onDelete" />
      </template>
    </UModal>
  </div>
</template>
