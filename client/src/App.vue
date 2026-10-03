<template>
  <v-app>
    <template v-if="!bare && branding">
      <v-app-bar
        v-if="branding.header.show"
        app
        class="nr-appbar"
        :elevation="branding.header.elevation"
        :dense="branding.theme.dense"
      >
        <router-link to="/" class="d-flex align-center nr-brand">
          <img
            v-if="branding.header.show_logo && logo"
            :src="logo"
            :alt="branding.app_name"
            :style="{ height: branding.header.logo_height + 'px' }"
            class="mr-3"
          >
          <span v-if="branding.header.show_app_name" class="nr-appbar-text nr-brand-name">{{ branding.app_name }}</span>
        </router-link>

        <div class="ml-6 d-none d-md-flex">
          <v-btn text :to="{ name: 'home' }" exact>
            <v-icon left small>mdi-compass-outline</v-icon> Browse
          </v-btn>
          <v-btn v-if="user" text :to="{ name: 'my-rooms' }">
            <v-icon left small>mdi-television-play</v-icon> My rooms
          </v-btn>
          <v-btn v-if="user && policy && policy.friends_enabled" text :to="{ name: 'friends' }">
            <v-badge :value="pendingFriends > 0" :content="pendingFriends" color="accent" overlap offset-x="-2" offset-y="8">
              <v-icon left small>mdi-account-multiple-outline</v-icon> Friends
            </v-badge>
          </v-btn>
          <v-btn v-if="isAdmin" text :to="{ name: 'admin' }">
            <v-icon left small>mdi-shield-crown-outline</v-icon> Admin
          </v-btn>
        </div>

        <v-spacer />

        <v-btn
          v-for="(link, i) in branding.header.links"
          :key="'hl' + i"
          text
          :href="link.url"
          :target="link.new_tab ? '_blank' : undefined"
          rel="noopener"
          class="d-none d-sm-flex"
        >
          <v-icon v-if="link.icon" left small>{{ link.icon }}</v-icon>{{ link.label }}
        </v-btn>

        <v-tooltip bottom v-if="branding.theme.allow_user_toggle">
          <template v-slot:activator="{ on, attrs }">
            <v-btn icon v-bind="attrs" v-on="on" @click="$store.dispatch('APP_TOGGLE_DARK')">
              <v-icon>{{ dark ? 'mdi-weather-sunny' : 'mdi-weather-night' }}</v-icon>
            </v-btn>
          </template>
          <span>{{ dark ? 'Light mode' : 'Dark mode' }}</span>
        </v-tooltip>

        <template v-if="user">
          <v-menu offset-y left min-width="220">
            <template v-slot:activator="{ on, attrs }">
              <v-btn icon v-bind="attrs" v-on="on" class="ml-1">
                <UserAvatar :user="user" :size="34" />
              </v-btn>
            </template>
            <v-list dense nav>
              <v-list-item two-line>
                <v-list-item-content>
                  <v-list-item-title class="font-weight-bold">{{ user.display_name || user.username }}</v-list-item-title>
                  <v-list-item-subtitle>@{{ user.username }} · {{ user.role }}</v-list-item-subtitle>
                </v-list-item-content>
              </v-list-item>
              <v-divider class="mb-1" />
              <v-list-item :to="{ name: 'my-rooms' }" class="d-md-none">
                <v-list-item-icon><v-icon>mdi-television-play</v-icon></v-list-item-icon>
                <v-list-item-title>My rooms</v-list-item-title>
              </v-list-item>
              <v-list-item v-if="policy && policy.friends_enabled" :to="{ name: 'friends' }" class="d-md-none">
                <v-list-item-icon><v-icon>mdi-account-multiple-outline</v-icon></v-list-item-icon>
                <v-list-item-title>Friends</v-list-item-title>
              </v-list-item>
              <v-list-item v-if="isAdmin" :to="{ name: 'admin' }" class="d-md-none">
                <v-list-item-icon><v-icon>mdi-shield-crown-outline</v-icon></v-list-item-icon>
                <v-list-item-title>Admin panel</v-list-item-title>
              </v-list-item>
              <v-list-item :to="{ name: 'account' }">
                <v-list-item-icon><v-icon>mdi-account-cog-outline</v-icon></v-list-item-icon>
                <v-list-item-title>Account</v-list-item-title>
              </v-list-item>
              <v-list-item @click="logout">
                <v-list-item-icon><v-icon>mdi-logout</v-icon></v-list-item-icon>
                <v-list-item-title>Sign out</v-list-item-title>
              </v-list-item>
            </v-list>
          </v-menu>
        </template>
        <template v-else>
          <v-btn text :to="{ name: 'login', query: loginQuery }" class="ml-1">Sign in</v-btn>
          <v-btn v-if="policy && policy.registration_enabled" color="primary" depressed :to="{ name: 'register' }" class="ml-2 d-none d-sm-flex">
            Sign up
          </v-btn>
        </template>
      </v-app-bar>
    </template>

    <v-main>
      <v-alert
        v-if="!bare && branding && branding.header.announcement"
        :type="branding.header.announcement_type || 'info'"
        tile
        dense
        class="mb-0 text-center"
      >{{ branding.header.announcement }}</v-alert>

      <router-view />
    </v-main>

    <v-footer v-if="!bare && branding && branding.footer.show" app absolute class="nr-footer px-4" padless>
      <v-container class="d-flex flex-wrap align-center py-3">
        <span class="caption">{{ expand(branding.footer.text) }}</span>
        <v-spacer />
        <a
          v-for="(link, i) in branding.footer.links"
          :key="'fl' + i"
          :href="link.url"
          :target="link.new_tab ? '_blank' : undefined"
          rel="noopener"
          class="caption ml-4 text-decoration-none"
        >
          <v-icon v-if="link.icon" x-small class="mr-1" style="color: inherit">{{ link.icon }}</v-icon>{{ link.label }}
        </a>
        <a v-if="branding.footer.show_powered_by" href="https://github.com/m1k1o/neko" target="_blank" rel="noopener" class="caption ml-4 text-decoration-none">
          {{ branding.footer.powered_by_label || 'based on n.eko' }}
        </a>
      </v-container>
    </v-footer>
  </v-app>
</template>

<style lang="scss" scoped>
.nr-brand {
  text-decoration: none;
  min-width: 0;
}
.nr-brand-name {
  font-size: 1.35rem;
  font-weight: 800;
  letter-spacing: -0.01em;
  white-space: nowrap;
}
</style>

<script lang="ts">
import { Vue, Component, Watch } from 'vue-property-decorator'
import { Branding, User, PublicPolicy } from '@/api-ext'
import { applyBranding, expand, logoFor, setTitle } from '@/branding/theme'
import UserAvatar from '@/components/community/UserAvatar.vue'

@Component({
  components: {
    UserAvatar,
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
