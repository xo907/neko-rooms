<template>
  <v-container class="py-8" style="max-width: 900px">
    <div class="d-flex align-center mb-6">
      <UserAvatar :user="user" :size="64" class="mr-4" />
      <div>
        <h1 class="nr-page-title nr-heading text-h4">{{ user.display_name || user.username }}</h1>
        <div class="text--secondary">@{{ user.username }} · {{ user.role }} · member since {{ user.created_at | datetime }}</div>
      </div>
    </div>

    <v-row>
      <v-col cols="12" md="6">
        <v-card>
          <v-card-title class="subtitle-1 font-weight-bold">Profile</v-card-title>
          <v-card-text>
            <v-text-field v-model="displayName" label="Display name" hint="Shown to others and used as your name inside rooms" persistent-hint outlined class="mb-3" />
            <v-text-field v-model="email" label="Email (optional)" outlined />
            <v-btn color="primary" depressed :loading="savingProfile" @click="saveProfile">Save profile</v-btn>
          </v-card-text>
        </v-card>
      </v-col>

      <v-col cols="12" md="6">
        <v-card>
          <v-card-title class="subtitle-1 font-weight-bold">Change password</v-card-title>
          <v-card-text>
            <v-text-field v-model="current" label="Current password" type="password" outlined autocomplete="current-password" />
            <v-text-field v-model="next" label="New password" type="password" outlined autocomplete="new-password" :hint="`At least ${minLength} characters`" />
            <v-text-field v-model="next2" label="Repeat new password" type="password" outlined autocomplete="new-password" />
            <v-btn color="primary" depressed :loading="savingPassword" @click="savePassword">Change password</v-btn>
          </v-card-text>
        </v-card>
      </v-col>

      <v-col cols="12">
        <v-card>
          <v-card-title class="subtitle-1 font-weight-bold">
            Active sessions
            <v-spacer />
            <v-btn small text color="error" @click="revokeOthers" :disabled="sessions.length < 2">Sign out other sessions</v-btn>
          </v-card-title>
          <v-simple-table>
            <thead>
              <tr><th>Device</th><th>IP</th><th>Last seen</th><th>Signed in</th></tr>
            </thead>
            <tbody>
              <tr v-for="s in sessions" :key="s.id">
                <td class="text-truncate" style="max-width: 320px" :title="s.user_agent">
                  {{ describeAgent(s.user_agent) }}
                  <v-chip v-if="s.current" x-small color="success" class="ml-2">this device</v-chip>
                </td>
                <td>{{ s.ip }}</td>
                <td>{{ s.last_seen_at | timeago }}</td>
                <td>{{ s.created_at | datetime }}</td>
              </tr>
            </tbody>
          </v-simple-table>
        </v-card>
      </v-col>
    </v-row>
  </v-container>
</template>

<script lang="ts">
import { Vue, Component } from 'vue-property-decorator'
import { api, errorMessage, Session, User } from '@/api-ext'
import UserAvatar from '@/components/community/UserAvatar.vue'

export function describeAgent(ua: string): string {
  if (!ua) return 'Unknown device'
  const browser = /Edg\//.test(ua) ? 'Edge' : /Firefox\//.test(ua) ? 'Firefox' : /Chrome\//.test(ua) ? 'Chrome' : /Safari\//.test(ua) ? 'Safari' : /curl/i.test(ua) ? 'curl' : 'Browser'
  const os = /Windows/.test(ua) ? 'Windows' : /Android/.test(ua) ? 'Android' : /iPhone|iPad/.test(ua) ? 'iOS' : /Mac OS/.test(ua) ? 'macOS' : /Linux/.test(ua) ? 'Linux' : ''
  return os ? `${browser} on ${os}` : browser
}

@Component({
  components: {
    UserAvatar,
  }
})
export default class AccountPage extends Vue {
  displayName = ''
  email = ''
  current = ''
  next = ''
  next2 = ''
  savingProfile = false
  savingPassword = false
  sessions: Session[] = []

  describeAgent = describeAgent

  get user(): User {
    return this.$store.getters.user
  }

  get minLength() {
    const p = this.$store.getters.policy
    return p ? p.password_min_length : 8
  }

  async loadSessions() {
    this.sessions = await api.sessions()
  }

  async saveProfile() {
    this.savingProfile = true
    try {
      // eslint-disable-next-line
      await api.updateAccount({ display_name: this.displayName, email: this.email })
      await this.$store.dispatch('APP_STATUS')
      this.$swal({ toast: true, position: 'bottom-end', timer: 2000, showConfirmButton: false, icon: 'success', title: 'Profile saved' })
    } catch (e) {
      this.$swal({ icon: 'error', title: 'Unable to save', text: errorMessage(e) })
    } finally {
      this.savingProfile = false
    }
  }

  async savePassword() {
    if (this.next !== this.next2) {
      this.$swal({ icon: 'error', title: 'Passwords do not match' })
      return
    }
    this.savingPassword = true
    try {
      await api.changePassword(this.current, this.next)
      this.current = this.next = this.next2 = ''
      await this.loadSessions()
      this.$swal({ icon: 'success', title: 'Password changed', text: 'Other sessions have been signed out.' })
    } catch (e) {
      this.$swal({ icon: 'error', title: 'Unable to change password', text: errorMessage(e) })
    } finally {
      this.savingPassword = false
    }
  }

  async revokeOthers() {
    await api.revokeOtherSessions()
    await this.loadSessions()
  }

  mounted() {
    this.displayName = this.user.display_name
    this.email = this.user.email
    this.loadSessions()
  }
}
</script>
