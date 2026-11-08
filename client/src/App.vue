<template>
  <v-app>
    <template v-if="!bare && branding">
      <v-app-bar
        v-if="branding.header.show"
        app
        flat
        class="nr-appbar"
        :height="56"
      >
        <div class="nr-bar nr-container">
          <router-link to="/" class="nr-brand">
            <img
              v-if="branding.header.show_logo && logo"
              :src="logo"
              :alt="branding.app_name"
              :style="{ height: Math.min(branding.header.logo_height, 28) + 'px' }"
            >
            <span v-if="branding.header.show_app_name" class="nr-brand-name">{{ branding.app_name }}</span>
            <span class="nr-brand-host nr-mono d-none d-sm-inline">{{ host }}</span>
          </router-link>

          <nav class="nr-nav d-none d-md-flex" aria-label="Primary">
            <router-link :to="{ name: 'home' }" exact class="nr-nav-link">Browse</router-link>
            <router-link v-if="user" :to="{ name: 'my-rooms' }" class="nr-nav-link">My rooms</router-link>
            <router-link v-if="user && policy && policy.friends_enabled" :to="{ name: 'friends' }" class="nr-nav-link">
              Friends<span v-if="pendingFriends > 0" class="nr-nav-badge nr-mono">{{ pendingFriends }}</span>
            </router-link>
            <router-link v-if="isAdmin" :to="{ name: 'admin' }" class="nr-nav-link">Admin</router-link>
          </nav>

          <div class="nr-actions">
            <span v-if="telemetry" class="nr-telemetry nr-mono d-none d-lg-inline-flex">
              <span class="nr-dot" />{{ telemetry }}
            </span>

            <a
              v-for="(link, i) in branding.header.links"
              :key="'hl' + i"
              :href="link.url"
              :target="link.new_tab ? '_blank' : undefined"
              rel="noopener"
              class="nr-btn is-sm d-none d-sm-inline-flex"
            >
              <v-icon v-if="link.icon" small>{{ link.icon }}</v-icon>{{ link.label }}
            </a>

            <v-tooltip bottom v-if="branding.theme.allow_user_toggle">
              <template v-slot:activator="{ on, attrs }">
                <button type="button" class="nr-btn is-icon" v-bind="attrs" v-on="on" @click="$store.dispatch('APP_TOGGLE_DARK')" :aria-label="dark ? 'Light mode' : 'Dark mode'">
                  <v-icon small>{{ dark ? 'mdi-white-balance-sunny' : 'mdi-moon-waning-crescent' }}</v-icon>
                </button>
              </template>
              <span>{{ dark ? 'Light mode' : 'Dark mode' }}</span>
            </v-tooltip>

            <template v-if="user">
              <router-link v-if="canCreate" :to="{ name: 'my-rooms', query: { create: '1' } }" class="nr-btn is-primary is-sm" aria-label="New room">
                <v-icon small>mdi-plus</v-icon><span class="d-none d-sm-inline">New room</span>
              </router-link>
              <v-menu offset-y left min-width="232" nudge-bottom="8" transition="slide-y-transition">
                <template v-slot:activator="{ on, attrs }">
                  <button v-bind="attrs" v-on="on" type="button" class="nr-avatar-btn" aria-label="Account menu">
                    <UserAvatar :user="user" :size="28" />
                  </button>
                </template>
                <v-list dense nav>
                  <div class="px-3 pt-2 pb-3">
                    <div class="nr-menu-name">{{ user.display_name || user.username }}</div>
                    <div class="nr-mono nr-menu-meta">@{{ user.username }} · {{ user.role }}</div>
                  </div>
                  <v-divider class="mb-1" />
                  <v-list-item :to="{ name: 'my-rooms' }" class="d-md-none">
                    <v-list-item-icon><v-icon small>mdi-television-play</v-icon></v-list-item-icon>
                    <v-list-item-title>My rooms</v-list-item-title>
                  </v-list-item>
                  <v-list-item v-if="policy && policy.friends_enabled" :to="{ name: 'friends' }" class="d-md-none">
                    <v-list-item-icon><v-icon small>mdi-account-multiple-outline</v-icon></v-list-item-icon>
                    <v-list-item-title>Friends<span v-if="pendingFriends > 0"> ({{ pendingFriends }})</span></v-list-item-title>
                  </v-list-item>
                  <v-list-item v-if="isAdmin" :to="{ name: 'admin' }" class="d-md-none">
                    <v-list-item-icon><v-icon small>mdi-shield-crown-outline</v-icon></v-list-item-icon>
                    <v-list-item-title>Admin</v-list-item-title>
                  </v-list-item>
                  <v-list-item :to="{ name: 'account' }">
                    <v-list-item-icon><v-icon small>mdi-account-cog-outline</v-icon></v-list-item-icon>
                    <v-list-item-title>Account</v-list-item-title>
                  </v-list-item>
                  <v-list-item @click="logout">
                    <v-list-item-icon><v-icon small>mdi-logout</v-icon></v-list-item-icon>
                    <v-list-item-title>Sign out</v-list-item-title>
                  </v-list-item>
                </v-list>
              </v-menu>
            </template>
            <template v-else>
              <router-link :to="{ name: 'login', query: loginQuery }" class="nr-btn is-sm">Sign in</router-link>
              <router-link v-if="policy && policy.registration_enabled" :to="{ name: 'register' }" class="nr-btn is-primary is-sm d-none d-sm-inline-flex">
                Sign up
              </router-link>
            </template>
          </div>
        </div>
      </v-app-bar>
    </template>

    <v-main>
      <div
        v-if="!bare && branding && branding.header.announcement"
        class="nr-announcement"
        :class="'nr-announcement--' + (branding.header.announcement_type || 'info')"
      >
        <span class="nr-dot" />{{ branding.header.announcement }}
      </div>

      <router-view />
    </v-main>

    <RoomPlayer />

    <v-footer v-if="!bare && branding && branding.footer.show" app absolute class="nr-footer" padless>
      <div class="nr-footer-inner nr-container">
        <span>{{ expand(branding.footer.text) }}</span>
        <span class="nr-footer-links">
          <a
            v-for="(link, i) in branding.footer.links"
            :key="'fl' + i"
            :href="link.url"
            :target="link.new_tab ? '_blank' : undefined"
            rel="noopener"
          >
            <v-icon v-if="link.icon" x-small class="mr-1" style="color: inherit">{{ link.icon }}</v-icon>{{ link.label }}
          </a>
          <a v-if="branding.footer.show_powered_by" href="https://github.com/m1k1o/neko" target="_blank" rel="noopener">
            {{ branding.footer.powered_by_label || 'based on n.eko' }}
          </a>
        </span>
      </div>
    </v-footer>
  </v-app>
</template>

<style lang="scss" scoped>
.nr-bar {
  height: 100%;
  display: flex;
  align-items: center;
  gap: 24px;
}
.nr-brand {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  text-decoration: none;
  color: var(--text-primary) !important;
  img { display: block; border-radius: var(--radius-sm); }
}
.nr-brand-name {
  font-size: 15px;
  font-weight: 600;
  letter-spacing: -0.03em;
  white-space: nowrap;
}
.nr-brand-host {
  font-size: 11.5px;
  color: var(--text-tertiary);
  white-space: nowrap;
}
.nr-nav {
  align-items: center;
  gap: 2px;
}
.nr-nav-link {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  height: 32px;
  padding: 0 12px;
  border-radius: 7px;
  color: var(--text-secondary) !important;
  font-size: 13.5px;
  font-weight: 500;
  text-decoration: none;
  transition: color var(--dur) var(--ease), background-color var(--dur) var(--ease);
  &:hover { color: var(--text-primary) !important; }
  &.router-link-active { color: var(--text-primary) !important; background: var(--bg-surface-2); }
}
.nr-nav-badge {
  min-width: 18px;
  height: 18px;
  padding: 0 5px;
  border-radius: 9px;
  background: var(--accent-primary);
  color: var(--accent-ink);
  font-size: 10.5px;
  font-weight: 600;
  line-height: 18px;
  text-align: center;
}
.nr-actions {
  margin-left: auto;
  display: flex;
  align-items: center;
  gap: 8px;
}
.nr-telemetry {
  align-items: center;
  gap: 8px;
  margin-right: 8px;
  font-size: 11.5px;
  color: var(--text-tertiary);
  white-space: nowrap;
}
.nr-avatar-btn {
  display: inline-flex;
  border-radius: 50%;
  box-shadow: 0 0 0 1px var(--border-hover);
  transition: box-shadow var(--dur) var(--ease);
  &:hover { box-shadow: 0 0 0 3px var(--border-hover); }
}
.nr-menu-name { font-size: 14px; font-weight: 600; letter-spacing: -0.02em; color: var(--text-primary); }
.nr-menu-meta { font-size: 11.5px; color: var(--text-tertiary); margin-top: 2px; }
.nr-announcement {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 8px 16px;
  font-size: 13px;
  color: var(--text-primary);
  border-bottom: 1px solid var(--border-subtle);
  background: var(--bg-surface-1);
  &.nr-announcement--warning .nr-dot { background: var(--status-provisioning); box-shadow: 0 0 8px var(--status-provisioning); }
  &.nr-announcement--error .nr-dot { background: var(--status-danger); box-shadow: 0 0 8px var(--status-danger); }
}
.nr-footer-inner {
  padding-top: 24px;
  padding-bottom: 32px;
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 8px 24px;
  font: 400 12px var(--font-mono);
  color: var(--text-tertiary);
  border-top: 1px solid var(--border-subtle);
}
.nr-footer-links {
  display: flex;
  flex-wrap: wrap;
  gap: 4px 24px;
  a { text-decoration: none; color: var(--text-tertiary) !important; transition: color var(--dur) var(--ease); }
  a:hover { color: var(--text-primary) !important; }
}
</style>

<style lang="scss">
.v-application .nr-appbar.v-app-bar {
  background-color: color-mix(in srgb, var(--bg-canvas) 80%, transparent) !important;
  backdrop-filter: blur(12px);
  -webkit-backdrop-filter: blur(12px);
  border-bottom: 1px solid var(--border-subtle) !important;
  color: var(--text-primary) !important;
}
.nr-appbar .v-toolbar__content { padding: 0 !important; }
.v-application .nr-footer.v-footer { background-color: transparent !important; }
</style>

<script lang="ts">
import { Vue, Component, Watch } from 'vue-property-decorator'
import { Branding, User, PublicPolicy } from '@/api-ext'
import { applyBranding, expand, logoFor, setTitle } from '@/branding/theme'
import UserAvatar from '@/components/community/UserAvatar.vue'
import RoomPlayer from '@/components/RoomPlayer.vue'

@Component({
  components: {
    UserAvatar,
    RoomPlayer,
  }
})
export default class App extends Vue {
  get branding(): Branding | null {
    return this.$store.state.app.branding
  }

  get dark(): boolean {
    return this.$store.state.app.dark
  }

  get user(): User | null {
    return this.$store.getters.user
  }

  get isAdmin(): boolean {
    return this.$store.getters.isAdmin
  }

  get policy(): PublicPolicy | null {
    return this.$store.getters.policy
  }

  get pendingFriends(): number {
    return this.$store.getters.pendingFriendRequests
  }

  get canCreate(): boolean {
    return this.$store.getters.canCreateRooms
  }

  get host(): string {
    return window.location.host
  }

  // a line of telemetry: ports in use for admins once known, else how many
  // rooms are live
  get telemetry(): string {
    // eslint-disable-next-line
    const st = this.$store.state as any
    const ports = st.roomsConfig && st.roomsConfig.connections
    if (this.isAdmin && ports && st.rooms && st.rooms.length) {
      // eslint-disable-next-line
      const used = st.rooms.reduce((sum: number, r: any) => sum + (r.max_connections || 1), 0)
      return used + '/' + ports + ' ports'
    }
    const live = (st.app.directory || []).filter((r: { ready: boolean }) => r.ready).length
    return live ? live + ' live' : ''
  }

  get bare(): boolean {
    return this.$route.matched.some(r => r.meta.bare)
  }

  get logo() {
    return this.branding ? logoFor(this.branding, this.dark) : ''
  }

  get loginQuery() {
    return this.$route.name && this.$route.name !== 'home' ? { redirect: this.$route.fullPath } : {}
  }

  expand(s: string) {
    return this.branding ? expand(this.branding, s) : s
  }

  @Watch('branding', { deep: true, immediate: true })
  @Watch('dark')
  onTheme() {
    if (!this.branding) return
    applyBranding(this.$vuetify, this.branding, this.dark)
    this.updateTitle()
  }

  @Watch('$route')
  updateTitle() {
    if (!this.branding) return
    const title = this.$route.matched.map(r => r.meta.title).filter(Boolean).pop()
    setTitle(this.branding, title)
  }

  @Watch('user', { immediate: true })
  onUser() {
    this.$store.dispatch('APP_FRIENDS').catch(() => { /* ignore */ })
  }

  private friendsTimer = 0

  mounted() {
    // keep friend requests badge fresh
    this.friendsTimer = window.setInterval(() => {
      if (this.user && !document.hidden) {
        this.$store.dispatch('APP_FRIENDS').catch(() => { /* ignore */ })
      }
    }, 30000)
  }

  beforeDestroy() {
    window.clearInterval(this.friendsTimer)
  }

  async logout() {
    await this.$store.dispatch('APP_LOGOUT')
    if (this.$route.name !== 'home') {
      this.$router.push({ name: 'home' }).catch(() => { /* ignore */ })
    }
  }
}
</script>
