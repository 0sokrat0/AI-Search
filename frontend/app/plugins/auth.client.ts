export default defineNuxtPlugin(() => {
  const authStore = useAuthStore()

  authStore.restoreAuth()
})
