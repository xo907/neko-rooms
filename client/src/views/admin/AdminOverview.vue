<template>
  <div>
    <v-row>
      <v-col v-for="card in cards" :key="card.key" cols="6" md="4" lg="2">
        <v-card class="pa-4" :to="card.to">
          <div class="d-flex align-center">
            <v-avatar :color="card.color" size="40" class="mr-3"><v-icon dark>{{ card.icon }}</v-icon></v-avatar>
            <div>
              <div class="text-h5 font-weight-bold">{{ card.key in stats ? stats[card.key] : '–' }}</div>
              <div class="caption text--secondary">{{ card.label }}</div>
            </div>
          </div>
        </v-card>
      </v-col>
    </v-row>

    <v-row class="mt-2">
      <v-col cols="12" md="5">
        <v-card>
          <v-card-title class="subtitle-1 font-weight-bold">Quick actions</v-card-title>
          <v-list>
            <v-list-item :to="{ name: 'admin-users' }">
              <v-list-item-icon><v-icon>mdi-account-plus-outline</v-icon></v-list-item-icon>
              <v-list-item-title>Add or manage users</v-list-item-title>
            </v-list-item>
            <v-list-item :to="{ name: 'admin-branding' }">
              <v-list-item-icon><v-icon>mdi-palette-outline</v-icon></v-list-item-icon>
              <v-list-item-title>Customize branding & theme</v-list-item-title>
            </v-list-item>
            <v-list-item :to="{ name: 'admin-rooms' }">
              <v-list-item-icon><v-icon>mdi-star-outline</v-icon></v-list-item-icon>
              <v-list-item-title>Feature or moderate rooms</v-list-item-title>
            </v-list-item>
            <v-list-item :to="{ name: 'admin-settings' }">
              <v-list-item-icon><v-icon>mdi-account-lock-outline</v-icon></v-list-item-icon>
              <v-list-item-title>Registration & permissions</v-list-item-title>
            </v-list-item>
            <v-list-item :to="{ name: 'my-rooms' }">
              <v-list-item-icon><v-icon>mdi-plus-box-outline</v-icon></v-list-item-icon>
              <v-list-item-title>Create a room</v-list-item-title>
            </v-list-item>
          </v-list>
        </v-card>
      </v-col>
      <v-col cols="12" md="7">
        <v-card>
          <v-card-title class="subtitle-1 font-weight-bold">
            Recent activity
            <v-spacer />
            <v-btn small text :to="{ name: 'admin-audit' }">View all</v-btn>
          </v-card-title>
          <v-simple-table dense>
            <tbody>
              <tr v-for="e in audit" :key="e.id">
                <td class="text-no-wrap caption">{{ e.created_at | timeago }}</td>
                <td><strong>{{ e.username || 'anonymous' }}</strong></td>
                <td><code>{{ e.action }}</code></td>
                <td class="text-truncate" style="max-width: 200px">{{ e.target }}</td>
              </tr>
              <tr v-if="!audit.length"><td class="text--secondary">No activity yet.</td></tr>
            </tbody>
          </v-simple-table>
        </v-card>
      </v-col>
    </v-row>
  </div>
</template>

<script lang="ts">
import { Vue, Component } from 'vue-property-decorator'
import { api, AuditEntry } from '@/api-ext'

@Component
export default class AdminOverview extends Vue {
  stats: Record<string, number> = {}
  audit: AuditEntry[] = []

  cards = [
    { key: 'users', label: 'Users', icon: 'mdi-account-group', color: 'primary', to: { name: 'admin-users' } },
    { key: 'admins', label: 'Admins', icon: 'mdi-shield-crown', color: 'accent', to: { name: 'admin-users' } },
    { key: 'rooms', label: 'Rooms', icon: 'mdi-television-play', color: 'info', to: { name: 'admin-rooms' } },
    { key: 'rooms_running', label: 'Running', icon: 'mdi-play-circle', color: 'success', to: { name: 'admin-rooms' } },
    { key: 'rooms_public', label: 'Public', icon: 'mdi-earth', color: 'warning', to: { name: 'admin-rooms' } },
    { key: 'viewers', label: 'Viewers now', icon: 'mdi-eye', color: 'error', to: { name: 'admin-rooms' } },
  ]

  async mounted() {
    api.audit(8, 0).then(r => { this.audit = r.entries }).catch(() => { /* ignore */ })
    try {
      this.stats = await api.overview()
    } catch (e) {
      console.error(e)
    }
  }
}
</script>
