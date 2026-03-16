import { defineStore } from 'pinia'
import type { User } from '~/types/auth'

interface AuthState {
  user: User | null
  accessToken: string | null
  refreshToken: string | null
}

const cookieMaxAge = 60 * 60 * 24 * 7

export const useAuthStore = defineStore('auth', {
  state: (): AuthState => ({
    user: null,
    accessToken: null,
    refreshToken: null
  }),

  getters: {
    isAuthenticated: state => !!state.accessToken && !!state.user,
    currentUser: state => state.user,
    hasRole: (state) => {
      return (role: string) => state.user?.roles.includes(role as any) ?? false
    },
    isSuperAdmin: (state) => {
      const roles = state.user?.roles ?? []
      return roles.includes('super_admin' as any)
    }
  },

  actions: {
    setAuth(accessToken: string, refreshToken: string, user: User) {
      this.accessToken = accessToken
      this.refreshToken = refreshToken
      this.user = user

      if (import.meta.client) {
        localStorage.setItem('accessToken', accessToken)
        localStorage.setItem('refreshToken', refreshToken)
        localStorage.setItem('user', JSON.stringify(user))

        document.cookie = `auth_token=${encodeURIComponent(accessToken)}; Path=/; Max-Age=${cookieMaxAge}; SameSite=Lax`
        document.cookie = `refresh_token=${encodeURIComponent(refreshToken)}; Path=/; Max-Age=${cookieMaxAge}; SameSite=Lax`
      }
    },

    clearAuth() {
      this.accessToken = null
      this.refreshToken = null
      this.user = null

      if (import.meta.client) {
        localStorage.removeItem('accessToken')
        localStorage.removeItem('refreshToken')
        localStorage.removeItem('user')

        document.cookie = 'auth_token=; Path=/; Max-Age=0; SameSite=Lax'
        document.cookie = 'refresh_token=; Path=/; Max-Age=0; SameSite=Lax'
      }
    },

    restoreAuth() {
      if (import.meta.client) {
        const accessToken = localStorage.getItem('accessToken')
        const refreshToken = localStorage.getItem('refreshToken')
        const userJson = localStorage.getItem('user')

        if (accessToken && refreshToken && userJson) {
          try {
            this.accessToken = accessToken
            this.refreshToken = refreshToken
            this.user = JSON.parse(userJson)

            document.cookie = `auth_token=${encodeURIComponent(accessToken)}; Path=/; Max-Age=${cookieMaxAge}; SameSite=Lax`
            document.cookie = `refresh_token=${encodeURIComponent(refreshToken)}; Path=/; Max-Age=${cookieMaxAge}; SameSite=Lax`
          } catch (error) {
            console.error('Failed to restore auth state:', error)
            this.clearAuth()
          }
        }
      }
    }
  }
})
