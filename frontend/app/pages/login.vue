<script setup lang="ts">
definePageMeta({
  layout: 'auth',
  middleware: 'guest'
})

const authStore = useAuthStore()

onMounted(() => {
  if (authStore.isAuthenticated) {
    navigateTo('/')
  }
})
</script>

<template>
  <div class="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900 px-4 py-12">
    <AuthLoginForm />
  </div>
</template>
