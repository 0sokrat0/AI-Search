<script setup lang="ts">
definePageMeta({
  middleware: 'auth'
})
await navigateTo('/accounts', { redirectCode: 301 })
</script>

<template />
