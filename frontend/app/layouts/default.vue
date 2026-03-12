<script setup lang="ts">
import type { NavigationMenuItem } from '@nuxt/ui'
import type { SignalItem } from '~/types'

const open = ref(false)
const auth = useAuthStore()

const { data: sidebarSignals } = useFetch<SignalItem[]>('/api/signals', {
  default: () => [],
  query: { limit: 200 }
})

const queueCount = computed(() => {
  return (sidebarSignals.value || []).filter(s => !s.isIgnored).length
})

const links = computed<NavigationMenuItem[][]>(() => [[{
  label: 'Радар',
  icon: 'i-lucide-house',
  to: '/',
  onSelect: () => {
    open.value = false
  }
}, {
  label: 'Сигналы',
  icon: 'i-lucide-inbox',
  to: '/inbox',
  badge: queueCount.value > 0 ? String(queueCount.value) : undefined,
  onSelect: () => {
    open.value = false
  }
}, {
  label: 'Аккаунты Telegram',
  to: '/accounts',
  icon: 'i-lucide-send',
  onSelect: () => {
    open.value = false
  }
},
...(auth.isSuperAdmin
  ? [{
      label: 'Настройки',
      to: '/settings',
      icon: 'i-lucide-settings',
      defaultOpen: true,
      type: 'trigger' as const,
      children: [{
        label: 'Настройки ИИ',
        to: '/settings',
        exact: true,
        onSelect: () => {
          open.value = false
        }
      }, {
        label: 'Пользователи',
        to: '/settings/members',
        onSelect: () => {
          open.value = false
        }
      }]
    }]
  : [])
]])

const groups = computed(() => [{
  id: 'links',
  label: 'Навигация',
  items: links.value.flat()
}])

</script>

<template>
  <UDashboardGroup unit="rem">
    <UDashboardSidebar id="default" v-model:open="open" collapsible resizable class="bg-elevated/25"
      :ui="{ footer: 'lg:border-t lg:border-default' }">

      <template #default="{ collapsed }">

        <UNavigationMenu :collapsed="collapsed" :items="links[0]" orientation="vertical" tooltip popover />

        <UNavigationMenu :collapsed="collapsed" :items="links[1]" orientation="vertical" tooltip class="mt-auto" />
      </template>

      <template #footer="{ collapsed }">
        <ClientOnly>
          <UserMenu :collapsed="collapsed" />
        </ClientOnly>
      </template>
    </UDashboardSidebar>

    <UDashboardSearch :groups="groups" />

    <slot />

    <NotificationsSlideover />
  </UDashboardGroup>
</template>
