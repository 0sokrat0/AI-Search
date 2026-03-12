export default defineNuxtRouteMiddleware((to, from) => {
  const authStore = useAuthStore()
  const token = useCookie('auth_token')

  const isAuth = authStore.isAuthenticated || !!token.value

  if (!isAuth) {
    return navigateTo('/login')
  }
})
