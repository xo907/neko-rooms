<template>
  <div class="d-flex nr-admin">
    <v-navigation-drawer
      v-model="drawer"
      :permanent="$vuetify.breakpoint.mdAndUp"
      :temporary="!$vuetify.breakpoint.mdAndUp"
      :absolute="!$vuetify.breakpoint.mdAndUp"
      width="240"
      height="auto"
      class="nr-drawer"
      :style="{ minHeight: 'calc(100vh - 64px)' }"
    >
      <v-list nav dense class="pt-4">
        <v-subheader class="font-weight-bold">ADMIN PANEL</v-subheader>
        <v-list-item v-for="item in items" :key="item.to" :to="{ name: item.to }" :exact="item.exact" color="primary">
          <v-list-item-icon><v-icon>{{ item.icon }}</v-icon></v-list-item-icon>
          <v-list-item-title>{{ item.label }}</v-list-item-title>
        </v-list-item>
      </v-list>
    </v-navigation-drawer>

    <div class="flex-grow-1" style="min-width: 0">
      <v-container fluid class="pa-4 pa-md-8" style="max-width: 1400px">
        <div class="d-flex align-center mb-6">
          <v-btn v-if="!$vuetify.breakpoint.mdAndUp" icon class="mr-2" @click="drawer = !drawer"><v-icon>mdi-menu</v-icon></v-btn>
          <h1 class="nr-page-title nr-heading text-h4">{{ title }}</h1>
        </div>
        <router-view />
      </v-container>
    </div>
  </div>
</template>

<style scoped>
.nr-admin { min-height: calc(100vh - 64px); position: relative; }
</style>

<script lang="ts">
import { Vue, Component } from 'vue-property-decorator'

@Component
export default class AdminLayout extends Vue {
  drawer = true

  items = [
    { to: 'admin', label: 'Overview', icon: 'mdi-view-dashboard-outline', exact: true },
    { to: 'admin-rooms', label: 'Rooms', icon: 'mdi-television-play' },
    { to: 'admin-users', label: 'Users', icon: 'mdi-account-group-outline' },
    { to: 'admin-branding', label: 'Branding & theme', icon: 'mdi-palette-outline' },
    { to: 'admin-settings', label: 'Settings', icon: 'mdi-tune-variant' },
    { to: 'admin-images', label: 'Images', icon: 'mdi-docker' },
    { to: 'admin-audit', label: 'Audit log', icon: 'mdi-history' },
  ]

  get title() {
    const t = this.$route.matched.map(r => r.meta.title).filter(Boolean).pop()
    return t === 'Admin' ? 'Overview' : t
  }
}
</script>
