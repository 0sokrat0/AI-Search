<script setup lang="ts">
import * as z from 'zod'
import type { FormSubmitEvent } from '@nuxt/ui'

const schema = z.object({
  name: z.string().min(2, 'Слишком короткое имя'),
  email: z.string().email('Некорректный email')
})
const open = ref(false)

type Schema = z.output<typeof schema>

const state = reactive<Partial<Schema>>({
  name: '',
  email: ''
})

const toast = useToast()
async function onSubmit(event: FormSubmitEvent<Schema>) {
  toast.add({ title: 'Успешно', description: `Клиент ${event.data.name} добавлен`, color: 'success' })
  open.value = false
}
</script>

<template>
  <UModal v-model:open="open" title="Новый клиент" description="Добавьте нового клиента в базу данных">
    <UButton label="Новый клиент" icon="i-lucide-plus" />

    <template #body>
      <UForm
        :schema="schema"
        :state="state"
        class="space-y-4"
        @submit="onSubmit"
      >
        <UFormField label="Имя" placeholder="Иван Иванов" name="name">
          <UInput v-model="state.name" class="w-full" />
        </UFormField>
        <UFormField label="Почта" placeholder="ivan@example.com" name="email">
          <UInput v-model="state.email" class="w-full" />
        </UFormField>
        <div class="flex justify-end gap-2">
          <UButton
            label="Отмена"
            color="neutral"
            variant="subtle"
            @click="open = false"
          />
          <UButton
            label="Создать"
            color="primary"
            variant="solid"
            type="submit"
          />
        </div>
      </UForm>
    </template>
  </UModal>
</template>
