<template>
  <v-container class="py-8" style="max-width: 900px">
    <div class="d-flex align-center flex-wrap mb-6" style="gap: 20px">
      <button type="button" class="nr-avatar-edit" :disabled="uploading" title="Change photo" @click="pickAvatar">
        <UserAvatar :user="user" :size="80" />
        <span class="nr-avatar-edit-overlay">
          <v-progress-circular v-if="uploading" indeterminate size="22" width="2" />
          <v-icon v-else>mdi-camera-outline</v-icon>
        </span>
      </button>
      <input ref="avatarInput" type="file" accept="image/png,image/jpeg,image/webp,image/gif" hidden @change="onAvatarFile">
      <div class="flex-grow-1" style="min-width: 0">
        <h1 class="nr-page-title nr-heading text-h4">{{ user.display_name || user.username }}</h1>
        <div class="text--secondary">@{{ user.username }} · {{ user.role }} · member since {{ user.created_at | datetime }}</div>
        <div class="mt-3 d-flex flex-wrap" style="gap: 8px">
          <v-btn small outlined :loading="uploading" @click="pickAvatar">
            <v-icon left small>mdi-image-outline</v-icon>{{ user.avatar ? 'Change photo' : 'Upload photo' }}
          </v-btn>
          <v-btn v-if="user.avatar" small text :disabled="uploading" @click="removeAvatar">Remove</v-btn>
        </div>
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

      <v-col cols="12">
        <v-card>
          <v-card-title class="subtitle-1 font-weight-bold">
            API tokens
            <v-spacer />
            <v-btn small depressed color="primary" @click="tokenDialog = true">
              <v-icon left small>mdi-key-plus</v-icon>New token
            </v-btn>
          </v-card-title>
          <v-card-text class="pb-0">
            Scripts can use the API as you, with <code>Authorization: Bearer &lt;token&gt;</code>. A token can do everything your account can.
          </v-card-text>
          <v-simple-table v-if="tokens.length">
            <thead>
              <tr><th>Name</th><th>Token</th><th>Last used</th><th>Expires</th><th /></tr>
            </thead>
            <tbody>
              <tr v-for="t in tokens" :key="t.id">
                <td>{{ t.name }}</td>
                <td><code>nrk_…{{ t.hint }}</code></td>
                <td><span v-if="t.last_used_at">{{ t.last_used_at | timeago }}</span><span v-else>never</span></td>
                <td><span v-if="t.expires_at">{{ t.expires_at | datetime }}</span><span v-else>never</span></td>
                <td class="text-right">
                  <v-btn icon small title="Revoke" @click="revokeToken(t)"><v-icon small>mdi-trash-can-outline</v-icon></v-btn>
                </td>
              </tr>
            </tbody>
          </v-simple-table>
          <v-card-text v-else class="text--secondary">No tokens yet.</v-card-text>
        </v-card>
      </v-col>
    </v-row>

    <v-dialog v-model="tokenDialog" max-width="480">
      <v-card>
        <v-card-title>{{ newToken ? 'Your new token' : 'New API token' }}</v-card-title>
        <v-card-text v-if="newToken">
          <p>Copy it now, it is not shown again.</p>
          <v-text-field :value="newToken" readonly outlined dense hide-details append-icon="mdi-content-copy" @click:append="copyToken" @focus="$event.target.select()" />
        </v-card-text>
        <v-card-text v-else>
          <v-text-field v-model="tokenName" label="Name" placeholder="Deploy script" outlined dense autofocus maxlength="60" />
          <v-select v-model="tokenDays" :items="tokenExpiry" label="Expires" outlined dense hide-details />
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <template v-if="newToken">
            <v-btn color="primary" depressed @click="closeToken">Done</v-btn>
          </template>
          <template v-else>
            <v-btn text @click="closeToken">Cancel</v-btn>
            <v-btn color="primary" depressed :loading="creatingToken" @click="createToken">Create</v-btn>
          </template>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </v-container>
</template>

<style lang="scss" scoped>
.nr-avatar-edit {
  position: relative;
  display: inline-flex;
  border-radius: 50%;
  cursor: pointer;
  box-shadow: 0 0 0 1px var(--nr-line-strong);
  transition: box-shadow .18s;
  &:hover, &:focus-visible { box-shadow: 0 0 0 3px color-mix(in srgb, var(--nr-primary) 45%, transparent); }
  &:hover .nr-avatar-edit-overlay, &:disabled .nr-avatar-edit-overlay { opacity: 1; }
}
.nr-avatar-edit-overlay {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  background: rgba(0, 0, 0, .5);
  opacity: 0;
  transition: opacity .18s;
  .v-icon { color: #fff !important; }
}
</style>

<script lang="ts">
import { Vue, Component } from 'vue-property-decorator'
import { api, ApiToken, errorMessage, Session, User } from '@/api-ext'
import UserAvatar from '@/components/community/UserAvatar.vue'

const AVATAR_SIZE = 256

// cropSquare center-crops an image file to a square avatar
async function cropSquare(file: File): Promise<Blob> {
  const url = URL.createObjectURL(file)
  try {
    const img = new Image()
    img.src = url
    await img.decode()

    const side = Math.min(img.naturalWidth, img.naturalHeight)
    const size = Math.min(AVATAR_SIZE, side)
    const canvas = document.createElement('canvas')
    canvas.width = canvas.height = size
    const ctx = canvas.getContext('2d')
    if (!ctx) throw new Error('unable to process the image')
    ctx.imageSmoothingQuality = 'high'
    ctx.drawImage(img, (img.naturalWidth - side) / 2, (img.naturalHeight - side) / 2, side, side, 0, 0, size, size)

    // browsers without webp encoding fall back to png
    const blob = await new Promise<Blob | null>(resolve => canvas.toBlob(resolve, 'image/webp', 0.9))
    if (!blob) throw new Error('unable to process the image')
    return blob
  } finally {
    URL.revokeObjectURL(url)
  }
}

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
  uploading = false
  sessions: Session[] = []
  tokens: ApiToken[] = []
  tokenDialog = false
  tokenName = ''
  tokenDays = 90
  newToken = ''
  creatingToken = false
  tokenExpiry = [
    { text: 'In 30 days', value: 30 },
    { text: 'In 90 days', value: 90 },
    { text: 'In a year', value: 365 },
    { text: 'Never', value: 0 },
  ]

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

  pickAvatar() {
    (this.$refs.avatarInput as HTMLInputElement).click()
  }

  async onAvatarFile(e: Event) {
    const input = e.target as HTMLInputElement
    const file = input.files && input.files[0]
    input.value = ''
    if (!file) return
    if (!file.type.startsWith('image/')) {
      this.$swal({ icon: 'error', title: 'Please pick an image' })
      return
    }

    this.uploading = true
    try {
      await api.uploadAvatar(await cropSquare(file))
      await this.$store.dispatch('APP_STATUS')
    } catch (err) {
      this.$swal({ icon: 'error', title: 'Unable to upload photo', text: errorMessage(err) })
    } finally {
      this.uploading = false
    }
  }

  async removeAvatar() {
    this.uploading = true
    try {
      await api.deleteAvatar()
      await this.$store.dispatch('APP_STATUS')
    } catch (err) {
      this.$swal({ icon: 'error', title: 'Unable to remove photo', text: errorMessage(err) })
    } finally {
      this.uploading = false
    }
  }

  async loadTokens() {
    this.tokens = await api.tokens()
  }

  async createToken() {
    this.creatingToken = true
    try {
      const res = await api.createToken(this.tokenName, this.tokenDays)
      this.newToken = res.token
      await this.loadTokens()
    } catch (e) {
      this.$swal({ icon: 'error', title: 'Unable to create a token', text: errorMessage(e) })
    } finally {
      this.creatingToken = false
    }
  }

  async copyToken() {
    try {
      await navigator.clipboard.writeText(this.newToken)
      this.$swal({ toast: true, position: 'bottom-end', timer: 1500, showConfirmButton: false, icon: 'success', title: 'Copied' })
    } catch { /* select it instead */ }
  }

  closeToken() {
    this.tokenDialog = false
    this.newToken = ''
    this.tokenName = ''
    this.tokenDays = 90
  }

  async revokeToken(t: ApiToken) {
    const { value } = await this.$swal({
      title: 'Revoke token',
      text: `Scripts using "${t.name}" stop working.`,
      icon: 'warning',
      showCancelButton: true,
      confirmButtonText: 'Revoke',
    })
    if (!value) return
    await api.deleteToken(t.id)
    await this.loadTokens()
  }

  async revokeOthers() {
    await api.revokeOtherSessions()
    await this.loadSessions()
  }

  mounted() {
    this.displayName = this.user.display_name
    this.email = this.user.email
    this.loadSessions()
    this.loadTokens()
  }
}
</script>
