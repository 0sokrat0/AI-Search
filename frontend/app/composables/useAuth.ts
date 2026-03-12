export const useAuth = () => {
  const authStore = useAuthStore()
  const router = useRouter()

  const logout = async () => {
    try {
      await $fetch('/api/v1/auth/logout', {
        method: 'POST'
      })
    } catch (error) {
      console.error('Logout error:', error)
    } finally {
      authStore.clearAuth()
      router.push('/login')
    }
  }

  const requireAuth = () => {
    if (!authStore.isAuthenticated) {
      router.push('/login')
      return false
    }
    return true
  }

  const requireRole = (role: string) => {
    if (!authStore.hasRole(role)) {
      router.push('/')
      return false
    }
    return true
  }

  return {
    user: computed(() => authStore.currentUser),
    isAuthenticated: computed(() => authStore.isAuthenticated),
    isSuperAdmin: computed(() => authStore.isSuperAdmin),
    logout,
    requireAuth,
    requireRole
  }
}
