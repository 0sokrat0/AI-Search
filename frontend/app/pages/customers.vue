<script setup lang="ts">
definePageMeta({
  middleware: 'auth'
})

await navigateTo('/leads', { redirectCode: 301 })
</script>

<template>
  <div />
</template>
