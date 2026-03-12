<script setup lang="ts">
definePageMeta({
  middleware: 'auth'
})

import type { NavigationMenuItem } from '@nuxt/ui'

const auth = useAuthStore()

if (!auth.isSuperAdmin) {
  await navigateTo('/')
}

const links = computed<NavigationMenuItem[][]>(() => [[
  ...(auth.isSuperAdmin
    ? [{
        label: 'Настройки ИИ',
        icon: 'i-lucide-sliders-horizontal',
        to: '/settings',
        exact: true
      }, {
        label: 'RAG Данные',
        icon: 'i-lucide-database-zap',
        to: '/settings/knowledge'
      }]
    : []),
  {
    label: 'Пользователи',
    icon: 'i-lucide-users',
    to: '/settings/members'
  }
]])
</script>

<template>
  <UDashboardPanel id="settings" :ui="{ body: 'lg:py-12' }">
    <template #header>
      <UDashboardNavbar title="Настройки">
        <template #leading>
          <UDashboardSidebarCollapse />
        </template>
      </UDashboardNavbar>

      <UDashboardToolbar>
        <UNavigationMenu :items="links" highlight class="-mx-1 flex-1" />
      </UDashboardToolbar>
    </template>

    <template #body>
      <div class="flex flex-col gap-4 sm:gap-6 lg:gap-12 w-full lg:max-w-2xl mx-auto">
        <NuxtPage />
      </div>
    </template>
  </UDashboardPanel>
</template>
