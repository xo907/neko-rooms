import Vue from 'vue'
import { ActionContext, Module } from 'vuex'
import { api, AuthStatus, Branding, Friend, Room } from '@/api-ext'
import { resolveDark, setUserThemePref } from '@/branding/theme'

declare global {
  interface Window {
    __BRANDING__?: Branding
  }
}

export interface AppState {
  status: AuthStatus | null
  branding: Branding | null
  dark: boolean
  directory: Room[]
  directoryLoaded: boolean
  friends: Friend[]
}

type Ctx = ActionContext<AppState, unknown>

const initialBranding = window.__BRANDING__ || null

const app: Module<AppState, unknown> = {
  state: () => ({
    status: null,
    branding: initialBranding,
    dark: initialBranding ? resolveDark(initialBranding) : true,
    directory: [],
    directoryLoaded: false,
    friends: [],
  }),
  getters: {
    user: (s: AppState) => s.status ? s.status.user : null,
    isAdmin: (s: AppState) => !!(s.status && s.status.user && s.status.user.role === 'admin'),
    policy: (s: AppState) => s.status ? s.status.policy : null,
    pendingFriendRequests: (s: AppState) => s.friends.filter(f => f.status === 'pending' && f.direction === 'incoming').length,
    canCreateRooms: (s: AppState) => {
      const st = s.status
      if (!st || !st.user) return false
      if (st.user.role === 'admin') return true
      return st.policy.users_can_create_rooms && (st.room_limit === 0 || st.room_count < st.room_limit)
    },
  },
  mutations: {
    STATUS_SET(s: AppState, status: AuthStatus) {
      Vue.set(s, 'status', status)
    },
    BRANDING_SET(s: AppState, b: Branding) {
      Vue.set(s, 'branding', b)
    },
    DARK_SET(s: AppState, dark: boolean) {
      s.dark = dark
    },
    DIRECTORY_SET(s: AppState, rooms: Room[]) {
      Vue.set(s, 'directory', rooms)
      s.directoryLoaded = true
    },
    FRIENDS_SET(s: AppState, friends: Friend[]) {
      Vue.set(s, 'friends', friends)
    },
  },
  actions: {
    async APP_STATUS({ commit }: Ctx) {
      const status = await api.status()
      commit('STATUS_SET', status)
      return status
    },
    async APP_LOGIN({ dispatch }: Ctx, { username, password }: { username: string; password: string }) {
      await api.login(username, password)
      await dispatch('APP_STATUS')
    },
    async APP_LOGOUT({ commit, dispatch }: Ctx) {
      await api.logout()
      commit('FRIENDS_SET', [])
      await dispatch('APP_STATUS')
    },
    async APP_BRANDING({ commit, state }: Ctx) {
      const b = await api.branding()
      commit('BRANDING_SET', b)
      if (!state.status) commit('DARK_SET', resolveDark(b))
      return b
    },
    APP_TOGGLE_DARK({ commit, state }: Ctx) {
      const dark = !state.dark
      setUserThemePref(dark ? 'dark' : 'light')
      commit('DARK_SET', dark)
    },
    async APP_DIRECTORY({ commit }: Ctx) {
      commit('DIRECTORY_SET', await api.directory())
    },
    async APP_FRIENDS({ commit, state }: Ctx) {
      if (!state.status || !state.status.user || !state.status.policy.friends_enabled) {
        commit('FRIENDS_SET', [])
        return
      }
      commit('FRIENDS_SET', await api.friends())
    },
  },
}

export default app
