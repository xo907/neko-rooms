<template>
  <v-app>
    <div class="nr-ambient" aria-hidden="true">
      <div class="nr-ambient-aurora" />
      <div class="nr-ambient-orb nr-ambient-orb--a" />
      <div class="nr-ambient-orb nr-ambient-orb--b" />
      <svg class="nr-ambient-grain" xmlns="http://www.w3.org/2000/svg">
        <filter id="nr-grain"><feTurbulence type="fractalNoise" baseFrequency="0.8" numOctaves="3" stitchTiles="stitch" /></filter>
        <rect width="100%" height="100%" filter="url(#nr-grain)" />
      </svg>
    </div>

    <template v-if="!bare && branding">
      <v-app-bar
        v-if="branding.header.show"
        app
        flat
        class="nr-appbar"
        :elevation="branding.header.elevation"
        :height="branding.theme.dense ? 52 : 60"
      >
        <div class="nr-bar">
          <router-link to="/" class="nr-brand">
            <img
              v-if="branding.header.show_logo && logo"
              :src="logo"
              :alt="branding.app_name"
              :style="{ height: branding.header.logo_height + 'px' }"
            >
            <span v-if="branding.header.show_app_name" class="nr-appbar-text nr-brand-name">{{ branding.app_name }}</span>
          </router-link>

          <nav class="nr-nav d-none d-md-flex" aria-label="Primary">
            <router-link :to="{ name: 'home' }" exact class="nr-nav-link">Browse</router-link>
            <router-link v-if="user" :to="{ name: 'my-rooms' }" class="nr-nav-link">My rooms</router-link>
            <router-link v-if="user && policy && policy.friends_enabled" :to="{ name: 'friends' }" class="nr-nav-link">
              Friends<span v-if="pendingFriends > 0" class="nr-nav-badge">{{ pendingFriends }}</span>
            </router-link>
            <router-link v-if="isAdmin" :to="{ name: 'admin' }" class="nr-nav-link">Admin</router-link>
          </nav>

          <div class="nr-actions">
            <v-btn
              v-for="(link, i) in branding.header.links"
              :key="'hl' + i"
              text
              small
              :href="link.url"
              :target="link.new_tab ? '_blank' : undefined"
              rel="noopener"
              class="d-none d-sm-flex nr-quiet"
            >
              <v-icon v-if="link.icon" left small>{{ link.icon }}</v-icon>{{ link.label }}
            </v-btn>

            <v-tooltip bottom v-if="branding.theme.allow_user_toggle">
              <template v-slot:activator="{ on, attrs }">
                <v-btn icon small class="nr-icon-btn" v-bind="attrs" v-on="on" @click="$store.dispatch('APP_TOGGLE_DARK')" :aria-label="dark ? 'Light mode' : 'Dark mode'">
                  <v-icon small>{{ dark ? 'mdi-white-balance-sunny' : 'mdi-moon-waning-crescent' }}</v-icon>
                </v-btn>
              </template>
              <span>{{ dark ? 'Light mode' : 'Dark mode' }}</span>
            </v-tooltip>

            <template v-if="user">
              <v-menu offset-y left min-width="230" nudge-bottom="8" transition="slide-y-transition">
                <template v-slot:activator="{ on, attrs }">
                  <button v-bind="attrs" v-on="on" class="nr-avatar-btn" aria-label="Account menu">
                    <UserAvatar :user="user" :size="30" />
                  </button>
                </template>
                <v-list dense nav>
                  <div class="px-3 pt-2 pb-3">
                    <div class="font-weight-bold">{{ user.display_name || user.username }}</div>
                    <div class="caption text--secondary">@{{ user.username }} · {{ user.role }}</div>
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
              <v-btn text small :to="{ name: 'login', query: loginQuery }" class="nr-quiet">Sign in</v-btn>
              <v-btn v-if="policy && policy.registration_enabled" color="primary" depressed small :to="{ name: 'register' }" class="d-none d-sm-flex">
                Sign up
              </v-btn>
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
      <div class="nr-footer-inner">
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
  width: 100%;
  max-width: 1240px;
  margin: 0 auto;
  display: grid;
  grid-template-columns: 1fr auto 1fr;
  align-items: center;
  gap: 16px;
}
.nr-brand {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
  justify-self: start;
  text-decoration: none;
  img { display: block; border-radius: 8px; }
}
.nr-brand-name {
  font-size: 1.05rem;
  font-weight: 600;
  letter-spacing: -0.02em;
  white-space: nowrap;
}
.nr-nav {
  align-items: center;
  gap: 2px;
  padding: 4px;
  border-radius: 999px;
  border: 1px solid var(--nr-line);
  background: var(--nr-hover);
}
.nr-nav-link {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  height: 30px;
  padding: 0 14px;
  border-radius: 999px;
  color: var(--nr-muted) !important;
  font-size: 13.5px;
  font-weight: 500;
  text-decoration: none;
  transition: color .18s, background-color .18s;
  &:hover { color: var(--nr-text) !important; }
  &.router-link-active { color: var(--nr-text) !important; background: var(--nr-line-strong); }
}
.nr-nav-badge {
  min-width: 18px;
  height: 18px;
  padding: 0 5px;
  border-radius: 99px;
  background: var(--nr-primary);
  color: var(--nr-on-primary);
  font-size: 11px;
  font-weight: 700;
  line-height: 18px;
  text-align: center;
}
.nr-actions {
  justify-self: end;
  display: flex;
  align-items: center;
  gap: 6px;
}
.nr-avatar-btn {
  display: inline-flex;
  margin-left: 4px;
  border-radius: 50%;
  box-shadow: 0 0 0 1px var(--nr-line-strong);
  transition: box-shadow .18s;
  &:hover { box-shadow: 0 0 0 3px var(--nr-line-strong); }
}
.nr-announcement {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 10px;
  padding: 10px 16px;
  font-size: 13px;
  color: var(--nr-text);
  border-bottom: 1px solid var(--nr-line);
  background: color-mix(in srgb, var(--nr-primary) 7%, transparent);
  &.nr-announcement--warning { background: color-mix(in srgb, #fbbf24 9%, transparent); .nr-dot { background: #fbbf24; box-shadow: 0 0 10px #fbbf24; } }
  &.nr-announcement--error { background: color-mix(in srgb, #f87171 9%, transparent); .nr-dot { background: #f87171; box-shadow: 0 0 10px #f87171; } }
}
.nr-footer-inner {
  width: 100%;
  max-width: 1240px;
  margin: 0 auto;
  padding: 28px 16px 32px;
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 8px 24px;
  font-size: 12.5px;
  border-top: 1px solid var(--nr-line);
}
.nr-footer-links {
  display: flex;
  flex-wrap: wrap;
  gap: 4px 20px;
  a { text-decoration: none; opacity: .85; transition: opacity .18s; }
  a:hover { opacity: 1; }
}
</style>

<style lang="scss">
.nr-appbar .v-toolbar__content { padding: 0 16px !important; }
.v-application .nr-appbar .nr-quiet { color: var(--nr-muted) !important; }
.v-application .nr-appbar .nr-quiet .v-icon { color: inherit !important; }
.v-application .nr-appbar .nr-quiet:hover { color: var(--nr-text) !important; }
.v-application .nr-appbar .nr-icon-btn .v-icon { color: var(--nr-muted) !important; }
.v-application .nr-appbar .v-btn.primary { color: var(--nr-on-primary) !important; }
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
