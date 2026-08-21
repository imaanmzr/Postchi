import { defineStore } from 'pinia'
import { apiUrl } from '~/utils/apiBase'

interface User {
  id: string
  email: string
  display_name: string
}

let ensureSessionPromise: Promise<boolean> | null = null

export const useAuthStore = defineStore('auth', {
  state: () => ({
    user: null as User | null,
    accessToken: '' as string,
    refreshToken: '' as string,
    sessionChecked: false,
    sessionValid: false,
  }),
  getters: {
    isAuthenticated: (s) => !!s.accessToken && s.sessionValid,
  },
  actions: {
    loadFromStorage() {
      if (import.meta.client) {
        this.accessToken = localStorage.getItem('access_token') || ''
        this.refreshToken = localStorage.getItem('refresh_token') || ''
        const user = localStorage.getItem('user')
        if (user) {
          try {
            this.user = JSON.parse(user)
          } catch {
            this.user = null
          }
        }
      }
    },
    persist() {
      if (import.meta.client) {
        localStorage.setItem('access_token', this.accessToken)
        localStorage.setItem('refresh_token', this.refreshToken)
        localStorage.setItem('user', JSON.stringify(this.user))
      }
    },
    setSession(user: User, accessToken: string, refreshToken: string) {
      this.user = user
      this.accessToken = accessToken
      this.refreshToken = refreshToken
      this.sessionChecked = true
      this.sessionValid = true
      this.persist()
    },
    async login(email: string, password: string) {
      const api = useApi()
      const res = await api.post<{ user: User; tokens: { access_token: string; refresh_token: string } }>(
        '/api/auth/login',
        { email, password },
      )
      this.setSession(res.user, res.tokens.access_token, res.tokens.refresh_token)
    },
    async register(email: string, password: string, displayName: string) {
      const api = useApi()
      const res = await api.post<{ user: User; tokens: { access_token: string; refresh_token: string } }>(
        '/api/auth/register',
        { email, password, display_name: displayName },
      )
      this.setSession(res.user, res.tokens.access_token, res.tokens.refresh_token)
    },
    async changePassword(currentPassword: string, newPassword: string) {
      const api = useApi()
      const res = await api.post<{ status: string; tokens: { access_token: string; refresh_token: string } }>(
        '/api/auth/change-password',
        { current_password: currentPassword, new_password: newPassword },
      )
      if (res.tokens?.access_token && res.tokens?.refresh_token) {
        this.accessToken = res.tokens.access_token
        this.refreshToken = res.tokens.refresh_token
        this.persist()
      }
    },
    async refresh() {
      try {
        const config = useRuntimeConfig()
        const res = await fetch(apiUrl(config.public.apiUrl as string, '/api/auth/refresh'), {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ refresh_token: this.refreshToken }),
        })
        if (!res.ok) return false
        const data = await res.json()
        this.accessToken = data.access_token
        this.refreshToken = data.refresh_token
        this.sessionValid = true
        this.sessionChecked = true
        this.persist()
        return true
      } catch {
        return false
      }
    },
    async ensureSession(): Promise<boolean> {
      if (import.meta.server) return false

      this.loadFromStorage()

      if (!this.accessToken) {
        this.sessionChecked = true
        this.sessionValid = false
        return false
      }

      if (this.sessionChecked && this.sessionValid) {
        return true
      }

      if (ensureSessionPromise) return ensureSessionPromise

      ensureSessionPromise = this.validateSession()
      try {
        return await ensureSessionPromise
      } finally {
        ensureSessionPromise = null
      }
    },
    async validateSession(): Promise<boolean> {
      const config = useRuntimeConfig()

      const fetchMe = async (token: string) => fetch(apiUrl(config.public.apiUrl as string, '/api/auth/me'), {
        headers: { Authorization: `Bearer ${token}` },
      })

      let res = await fetchMe(this.accessToken)

      if (res.status === 401 && this.refreshToken) {
        const refreshed = await this.refresh()
        if (refreshed) {
          res = await fetchMe(this.accessToken)
        }
      }

      if (res.ok) {
        this.user = await res.json()
        this.sessionValid = true
        this.sessionChecked = true
        this.persist()
        return true
      }

      this.clearSession()
      return false
    },
    clearSession() {
      this.user = null
      this.accessToken = ''
      this.refreshToken = ''
      this.sessionChecked = true
      this.sessionValid = false
      if (import.meta.client) {
        localStorage.removeItem('access_token')
        localStorage.removeItem('refresh_token')
        localStorage.removeItem('user')
      }
    },
    async logout() {
      const refreshToken = this.refreshToken
      this.clearSession()
      if (import.meta.client && refreshToken) {
        try {
          const config = useRuntimeConfig()
          await fetch(apiUrl(config.public.apiUrl as string, '/api/auth/logout'), {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ refresh_token: refreshToken }),
          })
        } catch {
          // Best-effort server-side token revocation.
        }
      }
    },
  },
})
