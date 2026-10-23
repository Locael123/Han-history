import { defineStore } from 'pinia'
import { api } from '@/api'
import { clearToken, getToken, setToken } from '@/api/http'
import type { AuthResult, MeView, UserBrief } from '@/api/types'

export const useAuthStore = defineStore('auth', {
  state: () => ({
    token: getToken(),
    user: null as UserBrief | null,
    me: null as MeView | null,
    loading: false,
  }),
  getters: {
    isLoggedIn: (state) => !!state.token,
    displayName: (state) => state.user?.nickname || state.user?.username || '未署名',
  },
  actions: {
    setSession(result: AuthResult) {
      this.token = result.token
      this.user = result.user
      setToken(result.token)
    },
    async login(username: string, password: string) {
      this.loading = true
      try {
        const result = await api.login(username, password)
        this.setSession(result)
        await this.fetchMe()
        return result
      } finally {
        this.loading = false
      }
    },
    async register(username: string, password: string, nickname: string) {
      this.loading = true
      try {
        const result = await api.register(username, password, nickname)
        this.setSession(result)
        await this.fetchMe()
        return result
      } finally {
        this.loading = false
      }
    },
    async fetchMe() {
      if (!this.token) return
      try {
        const me = await api.me()
        this.me = me
        this.user = { id: me.id, username: me.username, nickname: me.nickname }
      } catch {
        this.logout()
      }
    },
    logout() {
      this.token = ''
      this.user = null
      this.me = null
      clearToken()
    },
  },
})
