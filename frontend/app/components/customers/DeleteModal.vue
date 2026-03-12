<script setup lang="ts">
withDefaults(defineProps<{
  count?: number
}>(), {
  count: 0
})

const open = ref(false)

async function onSubmit() {
  await new Promise(resolve => setTimeout(resolve, 1000))
  open.value = false
}
</script>

<template>
  <UModal
    v-model:open="open"
    :title="`Удалить ${count} клиент${count > 1 ? 'ов' : 'а'}`"
    :description="'Подтвердите удаление. Это действие нельзя отменить.'"
  >
    <slot />

    <template #body>
      <div class="flex justify-end gap-2">
        <UButton
          label="Отмена"
          color="neutral"
          variant="subtle"
          @click="open = false"
        />
        <UButton
          label="Удалить"
          color="error"
          variant="solid"
          loading-auto
          @click="onSubmit"
        />
      </div>
    </template>
  </UModal>
</template>
